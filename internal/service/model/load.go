package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/infra/process"
)

// prepareAndStartProcess resolves the backend plugin, discovers it, allocates
// a port, builds the command, starts the process, and sets up the output handler.
func (m *Manager) prepareAndStartProcess(req *LoadRequest, model *Model, status *ModelStatus) (*process.Process, int, backend.Plugin, error) {
	modelPath := model.Path
	if len(model.ShardFiles) > 0 {
		modelPath = model.ShardFiles[0]
		logger.Infof("using shard model primary file: modelId=%s, path=%s, shardCount=%d", req.ModelID, modelPath, len(model.ShardFiles))
	}

	// Build capability hint for smart backend routing.
	var capHint *backend.CapabilityHint
	if caps := m.GetModelCapabilities(req.ModelID); caps != nil {
		capHint = &backend.CapabilityHint{
			TTS:             caps.TTS,
			ASR:             caps.ASR,
			ImageGeneration: caps.ImageGeneration,
		}
	}

	pluginID := backend.ID(req.BackendType)
	plugin, cfg, err := m.backendRegistry.Resolve(modelPath, pluginID, capHint)
	if err != nil {
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}

	info, err := plugin.Discover(cfg)
	if err != nil {
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}
	if !info.Available {
		err := fmt.Errorf("backend %s is not available", plugin.ID())
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}

	allocatedPort, err := m.portAllocator.NextPort()
	if err != nil {
		wrappedErr := fmt.Errorf("no available ports: %w", err)
		status.transitionTo(StateError)
		status.Error = wrappedErr
		return nil, 0, nil, wrappedErr
	}

	// Convert flat LoadRequest → RawParams → plugin.DecodeParams
	rawParams := loadRequestToRawParams(req)
	params, err := plugin.DecodeParams(rawParams)
	if err != nil {
		m.portAllocator.Release(allocatedPort)
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}

	bindHost := "0.0.0.0"
	if cfg != nil && cfg.BindHost != "" {
		bindHost = cfg.BindHost
	}

	backendReq := &backend.LoadRequest{
		ModelPath: modelPath,
		Port:      allocatedPort,
		Devices:   req.Devices,
		BindHost:  bindHost,
		Params:    params,
		EnvVars:   req.EnvVars,
		ExtraArgs: req.ExtraParams,
	}

	startCfg, err := plugin.BuildStartConfig(info, backendReq)
	if err != nil {
		m.portAllocator.Release(allocatedPort)
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}

	// Merge env vars: StartConfig < global backend config < model-level
	var cfgEnvVars []string
	if cfg != nil {
		if envRaw, ok := cfg.Raw["env"]; ok {
			switch v := envRaw.(type) {
			case []string:
				cfgEnvVars = v
			case []any:
				for _, item := range v {
					if s, ok := item.(string); ok {
						cfgEnvVars = append(cfgEnvVars, s)
					}
				}
			}
		}
	}
	envVars := make([]string, 0, len(startCfg.EnvVars)+len(cfgEnvVars)+len(req.EnvVars))
	envVars = append(envVars, startCfg.EnvVars...)
	envVars = append(envVars, cfgEnvVars...)
	envVars = append(envVars, req.EnvVars...)
	if len(envVars) > 0 {
		startCfg.EnvVars = envVars
	}

	cmdLine := ""
	if startCfg.CommandSpec != nil {
		cmdLine = startCfg.CommandSpec.CommandLine()
	}
	proc, err := m.processMgr.Start(req.ModelID, model.Name, cmdLine, startCfg.BinPath, startCfg.SkipLDLibraryPath, startCfg.EnvVars)
	if err != nil {
		m.portAllocator.Release(allocatedPort)
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}

	return proc, allocatedPort, plugin, nil
}

// LoadAsync loads a model asynchronously (returns immediately, loads in background).
func (m *Manager) LoadAsync(req *LoadRequest) (*LoadResult, error) {
	if req.InstanceID == "" {
		req.InstanceID = generateRuntimeInstanceID(req.ModelID)
	}

	// Get model
	model, exists := m.GetModel(req.ModelID)
	if !exists {
		return nil, fmt.Errorf("model not found: %s", req.ModelID)
	}

	// Use write lock for both check-and-set to prevent TOCTOU race:
	// Two concurrent callers could both pass an RLock check and overwrite each other's status.
	m.mu.Lock()
	if existing, exists := m.statuses[req.ModelID]; exists {
		if existing.State == StateLoaded {
			m.mu.Unlock()
			return &LoadResult{
				Success:       true,
				ModelID:       req.ModelID,
				InstanceID:    existing.InstanceID,
				Port:          existing.Port,
				Async:         true,
				AlreadyLoaded: true,
			}, nil
		}
		if existing.State == StateLoading {
			m.mu.Unlock()
			return &LoadResult{
				Success:    true,
				ModelID:    req.ModelID,
				InstanceID: existing.InstanceID,
				Async:      true,
				Loading:    true,
			}, nil
		}
	}

	status := &ModelStatus{
		ID:         req.ModelID,
		InstanceID: req.InstanceID,
		Name:       model.Name,
	}
	applyRuntimeConfig(status, req.UnloadAfterMinutes, req.ConcurrencyLimit)
	status.LoadWait.Add(1)
	m.statuses[req.ModelID] = status
	m.instances[req.InstanceID] = &RuntimeInstance{
		InstanceID: req.InstanceID,
		ModelID:    req.ModelID,
		ModelName:  model.Name,
		ProfileID:  req.ProfileID,
		State:      StateLoading.String(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	m.modelInstances[req.ModelID] = append(m.modelInstances[req.ModelID], req.InstanceID)
	m.bumpVersion()
	m.mu.Unlock()

	m.swapBeforeLoad(req.ModelID)

	if err := status.transitionTo(StateLoading); err != nil {
		status.LoadWait.Done()
		m.mu.Lock()
		delete(m.statuses, req.ModelID)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to transition to loading: %w", err)
	}

	go m.loadModelAsync(req, status, model)

	return &LoadResult{
		Success:    true,
		ModelID:    req.ModelID,
		InstanceID: req.InstanceID,
		Async:      true,
		Loading:    true,
	}, nil
}

// loadModelAsync runs the actual model loading in a background goroutine.
// The model parameter is pre-fetched in LoadAsync to avoid re-acquiring locks in this goroutine.
func (m *Manager) loadModelAsync(req *LoadRequest, status *ModelStatus, model *Model) {
	startTime := time.Now()

	logger.Infof("开始异步加载模型: modelId=%s, modelName=%s", req.ModelID, model.Name)

	// 根据模型大小计算超时时间
	timeout := m.calculateLoadTimeout(model.Size)
	logger.Infof("模型加载超时设置: modelId=%s, sizeGB=%.2f, timeout=%s", req.ModelID, float64(model.Size)/(1024*1024*1024), timeout)

	// 准备并启动进程
	proc, port, b, err := m.prepareAndStartProcess(req, model, status)
	if err != nil {
		logger.Errorf("异步模型加载失败: modelId=%s, error=%v", req.ModelID, err)
		return
	}

	logger.Infof("进程启动成功，准备获取 PID: modelId=%s", req.ModelID)

	// 获取 PID
	pid := proc.GetPID()
	logger.Infof("异步模型加载: 进程已启动: modelId=%s, pid=%d, port=%d, backend=%s", req.ModelID, pid, port, b.ID())

	// 等待加载完成（监控进程输出）
	loadCompleted := make(chan bool, 1)
	loadError := make(chan error, 1)
	stopHealthCheck := make(chan bool, 1)

	// 启动进程健康检查 goroutine
	// 不设独立总超时，由外层 calculateLoadTimeout 统一控制
	// 503 视为正常加载中，不作为失败处理
	go func() {
		checkInterval := 5 * time.Second

		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		checkCount := 0
		for {
			select {
			case <-stopHealthCheck:
				logger.Infof("健康检查收到停止信号: modelId=%s, checkCount=%d", req.ModelID, checkCount)
				return
			case <-ticker.C:
				checkCount++

				// 检查进程是否仍在运行
				if !proc.IsRunning() {
					logger.Errorf("健康检查发现进程已退出: modelId=%s, pid=%d", req.ModelID, proc.GetPID())
					select {
					case loadError <- fmt.Errorf("进程意外退出 (PID: %d)", proc.GetPID()):
					default:
					}
					return
				}

				// 发送健康请求 - 503 (模型加载中) 不算失败
				result, err := b.CheckHealth(port)
				if err != nil {
					// 连接失败，可能是进程尚未监听端口
					logger.Debugf("健康检查连接失败（正常：后端可能仍在启动）: modelId=%s, error=%v, checkCount=%d", req.ModelID, err, checkCount)
				} else if !result.Healthy {
					// HTTP 503 等 - 模型仍在加载中，继续等待
					logger.Infof("后端尚未就绪（正常加载中）: modelId=%s, statusCode=%s, checkCount=%d", req.ModelID, "non-200", checkCount)
				} else {
					// 健康检查成功！
					logger.Infof("健康检查成功，模型已就绪: modelId=%s, port=%d, checkCount=%d", req.ModelID, port, checkCount)
					select {
					case loadCompleted <- true:
					default:
					}
					return
				}
			}
		}
	}()

	// 设置输出处理器检测加载完成并转发日志
	proc.SetOutputHandler(func(line string) {
		if !strings.Contains(line, "update_slots") && !strings.Contains(line, "log_server_r") {
			logger.Debug(fmt.Sprintf("[%s] %s", req.ModelID, line))
		}

		// Use backend-specific load completion detection
		if b.IsLoadComplete(line) {
			select {
			case loadCompleted <- true:
			default:
			}
		}
	})

	// 等待加载完成或超时
	select {
	case <-loadCompleted:
		close(stopHealthCheck)
		m.mu.Lock()
		status.transitionTo(StateLoaded)
		status.ProcessID = proc.ID
		status.Port = port
		status.LoadedAt = time.Now()
		status.BackendType = string(b.ID())
		if inst := m.instances[req.InstanceID]; inst != nil {
			inst.ProcessID = proc.ID
			inst.Port = port
			inst.State = StateLoaded.String()
			inst.BackendType = string(b.ID())
			inst.UpdatedAt = time.Now()
		}
		m.bumpVersion()
		m.mu.Unlock()
		status.LoadWait.Done()
		duration := time.Since(startTime)
		logger.Infof("异步模型加载成功: modelId=%s, port=%d, duration=%s, backend=%s", req.ModelID, port, duration.String(), b.ID())

	case err := <-loadError:
		close(stopHealthCheck)
		m.mu.Lock()
		status.transitionTo(StateError)
		status.Error = err
		if inst := m.instances[req.InstanceID]; inst != nil {
			inst.State = StateError.String()
			inst.LastError = err.Error()
			inst.UpdatedAt = time.Now()
		}
		m.mu.Unlock()
		status.LoadWait.Done()
		logger.Errorf("异步模型加载失败: modelId=%s, error=%v", req.ModelID, err)
		m.processMgr.Stop(req.ModelID) //errcheck:ignore
		m.portAllocator.Release(port)

	case <-time.After(timeout):
		close(stopHealthCheck)
		m.mu.Lock()
		status.transitionTo(StateError)
		status.Error = fmt.Errorf("模型加载超时 (%v)", timeout)
		if inst := m.instances[req.InstanceID]; inst != nil {
			inst.State = StateError.String()
			inst.LastError = status.Error.Error()
			inst.UpdatedAt = time.Now()
		}
		m.mu.Unlock()
		status.LoadWait.Done()
		logger.Errorf("异步模型加载超时: modelId=%s, timeout=%s", req.ModelID, timeout)
		m.processMgr.Stop(req.ModelID) //errcheck:ignore
		m.portAllocator.Release(port)
	}
}

// calculateLoadTimeout computes a dynamic timeout based on model file size.
func (m *Manager) calculateLoadTimeout(modelSize int64) time.Duration {
	const (
		minTimeout   = 5 * time.Minute
		maxTimeout   = 30 * time.Minute
		minutesPerGB = 1 * time.Minute
		gigabyte     = int64(1024 * 1024 * 1024)
	)

	sizeGB := float64(modelSize) / float64(gigabyte)
	dynamicTimeout := time.Duration(sizeGB) * minutesPerGB

	dynamicTimeout = max(minTimeout, min(maxTimeout, dynamicTimeout))

	return dynamicTimeout
}

// Unload unloads a model
func (m *Manager) Unload(modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, exists := m.statuses[modelID]
	if !exists {
		logger.Warnf("模型卸载失败: 模型未加载: modelId=%s", modelID)
		return fmt.Errorf("model not loaded: %s", modelID)
	}

	if status.State != StateLoaded {
		logger.Warnf("模型卸载失败: 模型未处于已加载状态: modelId=%s, state=%v", modelID, status.State)
		return fmt.Errorf("model not in loaded state: %s", modelID)
	}

	logger.Infof("开始卸载模型: modelId=%s, modelName=%s, port=%d", modelID, status.Name, status.Port)

	status.transitionTo(StateUnloading)
	status.InflightWait()

	if err := m.processMgr.Stop(modelID); err != nil {
		logger.Errorf("模型卸载失败: 停止进程失败: modelId=%s, error=%v", modelID, err)
		return err
	}

	if status.Port > 0 {
		m.portAllocator.Release(status.Port)
		logger.Infof("已释放端口: modelId=%s, port=%d", modelID, status.Port)
	}

	status.transitionTo(StateUnloaded)
	status.ProcessID = ""
	status.Port = 0
	if inst := m.instances[status.InstanceID]; inst != nil {
		inst.State = StateUnloaded.String()
		inst.Port = 0
		inst.ProcessID = ""
		inst.UpdatedAt = time.Now()
	}
	m.bumpVersion()

	logger.Infof("模型卸载成功: modelId=%s, modelName=%s", modelID, status.Name)

	return nil
}

// loadRequestToRawParams converts a flat model.LoadRequest into a backend.RawParams
// map suitable for plugin.DecodeParams. Uses JSON round-trip for simplicity.
func loadRequestToRawParams(req *LoadRequest) backend.RawParams {
	data, err := json.Marshal(req)
	if err != nil {
		return backend.RawParams{}
	}
	var raw backend.RawParams
	if err := json.Unmarshal(data, &raw); err != nil {
		return backend.RawParams{}
	}
	return raw
}

// ListRuntimeInstances returns all known runtime instances.
func (m *Manager) ListRuntimeInstances() []*RuntimeInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instances := make([]*RuntimeInstance, 0, len(m.instances))
	for _, inst := range m.instances {
		copy := *inst
		instances = append(instances, &copy)
	}
	return instances
}

// GetRuntimeInstance returns a runtime instance by ID.
func (m *Manager) GetRuntimeInstance(instanceID string) (*RuntimeInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, ok := m.instances[instanceID]
	if !ok {
		return nil, false
	}
	copy := *inst
	return &copy, true
}

func generateRuntimeInstanceID(modelID string) string {
	return fmt.Sprintf("inst_%s_%d", modelID, time.Now().UnixNano())
}

func applyRuntimeConfig(status *ModelStatus, unloadAfterMinutes, concurrencyLimit int) {
	if unloadAfterMinutes > 0 {
		status.SetUnloadAfter(time.Duration(unloadAfterMinutes) * time.Minute)
	}

	if concurrencyLimit > 0 {
		status.InitConcurrency(concurrencyLimit)
	}
}
