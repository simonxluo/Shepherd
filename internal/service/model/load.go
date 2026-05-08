package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model/backend"
)

// prepareAndStartProcess contains the shared preparation logic for both Load and LoadAsync.
// It resolves the appropriate backend, discovers it, allocates a port, builds the command,
// starts the process, and sets up the output handler.
// On error, it updates status to StateError and releases any allocated port.
func (m *Manager) prepareAndStartProcess(req *LoadRequest, model *Model, status *ModelStatus) (*process.Process, int, backend.Backend, error) {
	// Resolve which backend to use
	modelPath := model.Path
	if len(model.ShardFiles) > 0 {
		modelPath = model.ShardFiles[0]
		logger.Infof("using shard model primary file: modelId=%s, path=%s, shardCount=%d", req.ModelID, modelPath, len(model.ShardFiles))
	}

	bt, parseErr := backend.ParseBackendType(req.BackendType)
	if parseErr != nil {
		status.transitionTo(StateError)
		status.Error = parseErr
		return nil, 0, nil, parseErr
	}
	b, bcfg, err := m.backendRegistry.Resolve(modelPath, bt)
	if err != nil {
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}

	// Discover/validate backend
	info, err := b.Discover(bcfg)
	if err != nil {
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}
	if !info.Available {
		err := fmt.Errorf("backend %s is not available", b.Type())
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

	// Build command via backend interface
	backendReq := m.toBackendLoadRequest(req, modelPath, allocatedPort)
	startCfg, err := b.BuildStartConfig(info, backendReq)
	if err != nil {
		m.portAllocator.Release(allocatedPort)
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}

	// 从后端配置中附加环境变量
	if len(bcfg.EnvVars) > 0 {
		startCfg.EnvVars = bcfg.EnvVars
	}

	proc, err := m.processMgr.Start(req.ModelID, model.Name, startCfg.Command, startCfg.BinPath, startCfg.SkipLDLibraryPath, startCfg.EnvVars)
	if err != nil {
		m.portAllocator.Release(allocatedPort)
		status.transitionTo(StateError)
		status.Error = err
		return nil, 0, nil, err
	}

	return proc, allocatedPort, b, nil
}

// LoadAsync 异步加载模型（立即返回，后台加载）
func (m *Manager) LoadAsync(req *LoadRequest) (*LoadResult, error) {
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
				Port:          existing.Port,
				Async:         true,
				AlreadyLoaded: true,
			}, nil
		}
		if existing.State == StateLoading {
			m.mu.Unlock()
			return &LoadResult{
				Success: true,
				ModelID: req.ModelID,
				Async:   true,
				Loading: true,
			}, nil
		}
	}

	status := &ModelStatus{
		ID:   req.ModelID,
		Name: model.Name,
	}
	applyRuntimeConfig(status, req.UnloadAfterMinutes, req.ConcurrencyLimit)
	status.LoadWait.Add(1)
	m.statuses[req.ModelID] = status
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
		Success: true,
		ModelID: req.ModelID,
		Async:   true,
		Loading: true,
	}, nil
}

// loadModelAsync 后台异步加载模型
// model 参数已在 LoadAsync 中获取，避免在此 goroutine 中再次获取锁导致死锁
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
	logger.Infof("异步模型加载: 进程已启动: modelId=%s, pid=%d, port=%d, backend=%s", req.ModelID, pid, port, b.Type())

	// 等待加载完成（监控进程输出）
	loadCompleted := make(chan bool, 1)
	loadError := make(chan error, 1)
	stopHealthCheck := make(chan bool, 1)

	// 启动进程健康检查 goroutine (llama-swap 风格：单一总超时，不计失败次数)
	// 503 视为正常加载中，不作为失败处理
	go func() {
		checkStart := time.Now()
		healthCheckTimeout := 120 * time.Second
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

				// 检查总超时
				if time.Since(checkStart) > healthCheckTimeout {
					logger.Errorf("健康检查总超时: modelId=%s, timeout=%s, checkCount=%d", req.ModelID, healthCheckTimeout, checkCount)
					select {
					case loadError <- fmt.Errorf("健康检查超时 (%v)，模型可能加载失败", healthCheckTimeout):
					default:
					}
					return
				}

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
					logger.Warnf("健康检查连接失败（正常：后端可能仍在启动）: modelId=%s, error=%v, checkCount=%d", req.ModelID, err, checkCount)
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
		status.State = StateLoaded
		status.ProcessID = proc.ID
		status.Port = port
		status.LoadedAt = time.Now()
		status.BackendType = b.Type().String()
		m.mu.Unlock()
		status.LoadWait.Done()
		duration := time.Since(startTime)
		logger.Infof("异步模型加载成功: modelId=%s, port=%d, duration=%s, backend=%s", req.ModelID, port, duration.String(), b.Type())

	case err := <-loadError:
		close(stopHealthCheck)
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		status.LoadWait.Done()
		logger.Errorf("异步模型加载失败: modelId=%s, error=%v", req.ModelID, err)
		m.processMgr.Stop(req.ModelID) //errcheck:ignore
		m.portAllocator.Release(port)

	case <-time.After(timeout):
		close(stopHealthCheck)
		m.mu.Lock()
		status.State = StateError
		status.Error = fmt.Errorf("模型加载超时 (%v)", timeout)
		m.mu.Unlock()
		status.LoadWait.Done()
		logger.Errorf("异步模型加载超时: modelId=%s, timeout=%s", req.ModelID, timeout)
		m.processMgr.Stop(req.ModelID) //errcheck:ignore
		m.portAllocator.Release(port)
	}
}

// calculateLoadTimeout 根据模型大小计算动态超时时间
func (m *Manager) calculateLoadTimeout(modelSize int64) time.Duration {
	const (
		minTimeout   = 5 * time.Minute
		maxTimeout   = 30 * time.Minute
		minutesPerGB = 1 * time.Minute
		gigabyte     = int64(1024 * 1024 * 1024)
	)

	sizeGB := float64(modelSize) / float64(gigabyte)
	dynamicTimeout := time.Duration(sizeGB) * minutesPerGB

	if dynamicTimeout < minTimeout {
		dynamicTimeout = minTimeout
	}
	if dynamicTimeout > maxTimeout {
		dynamicTimeout = maxTimeout
	}

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

	status.State = StateUnloaded
	status.ProcessID = ""
	status.Port = 0

	logger.Infof("模型卸载成功: modelId=%s, modelName=%s", modelID, status.Name)

	return nil
}

// toBackendLoadRequest converts model.LoadRequest to backend.LoadRequest
func (m *Manager) toBackendLoadRequest(req *LoadRequest, modelPath string, port int) *backend.LoadRequest {
	br := &backend.LoadRequest{
		ModelPath:   modelPath,
		Port:        port,
		CtxSize:     req.CtxSize,
		GPULayers:   req.GPULayers,
		Threads:     req.Threads,
		Devices:     req.Devices,
		SpecDecoding: req.SpecDecoding.ToBackend(),
	}

	// Map all fields to the appropriate backend params based on the backend type
	bt, _ := backend.ParseBackendType(req.BackendType)
	switch bt {
	case backend.BackendVLLM:
		br.VLLMParams = &backend.VLLMLoadParams{
			DataType:             req.DataType,
			MaxModelLen:          req.MaxModelLen,
			GPUMemoryUtilization: req.GPUMemoryUtilization,
			TensorParallelSize:   req.TensorParallelSize,
			PipelineParallelSize: req.PipelineParallelSize,
			TrustRemoteCode:      req.TrustRemoteCode,
			ServedModelName:      req.ServedModelName,
			Quantization:         req.Quantization,
			MaxNumSeqs:           req.MaxNumSeqs,
			MaxNumBatchedTokens:  req.MaxNumBatchedTokens,
			EnablePrefixCaching:  req.EnablePrefixCaching,
			EnableChunkedPrefill: req.EnableChunkedPrefill,
			DisableLogRequests:   req.DisableLogRequests,
			EnforceEager:         req.EnforceEager,
			ExtraArgs:            req.ExtraParams,
		}
	case backend.BackendVLLMOmni:
		br.VLLOmniParams = &backend.VLLOmniLoadParams{
			VLLMLoadParams: backend.VLLMLoadParams{
				DataType:             req.DataType,
				MaxModelLen:          req.MaxModelLen,
				GPUMemoryUtilization: req.GPUMemoryUtilization,
				TensorParallelSize:   req.TensorParallelSize,
				PipelineParallelSize: req.PipelineParallelSize,
				TrustRemoteCode:      req.TrustRemoteCode,
				ServedModelName:      req.ServedModelName,
				Quantization:         req.Quantization,
				MaxNumSeqs:           req.MaxNumSeqs,
				MaxNumBatchedTokens:  req.MaxNumBatchedTokens,
				EnablePrefixCaching:  req.EnablePrefixCaching,
				EnableChunkedPrefill: req.EnableChunkedPrefill,
				DisableLogRequests:   req.DisableLogRequests,
				EnforceEager:         req.EnforceEager,
				ExtraArgs:            req.ExtraParams,
			},
			Omni:             req.Omni,
			VideoPruningRate: req.VideoPruningRate,
			MMTensorIPC:      req.MMTensorIPC,
		}
	default:
		// Default: map to llama.cpp params (backward compatible)
		br.LlamacppParams = &backend.LlamacppLoadParams{
			BatchSize:          req.BatchSize,
			Temperature:        req.Temperature,
			TopP:               req.TopP,
			TopK:               req.TopK,
			RepeatPenalty:      req.RepeatPenalty,
			Seed:               req.Seed,
			NPredict:           req.NPredict,
			MainGPU:            req.MainGPU,
			CustomCmd:          req.CustomCmd,
			ExtraParams:        req.ExtraParams,
			MmprojPath:         req.MmprojPath,
			EnableVision:       req.EnableVision,
			FlashAttention:     req.FlashAttention,
			NoMmap:             req.NoMmap,
			LockMemory:         req.LockMemory,
			NoWebUI:            req.NoWebUI,
			EnableMetrics:      req.EnableMetrics,
			SlotSavePath:       req.SlotSavePath,
			CacheRAM:           req.CacheRAM,
			ChatTemplateFile:   req.ChatTemplateFile,
			Timeout:            req.Timeout,
			Alias:              req.Alias,
			UBatchSize:         req.UBatchSize,
			ParallelSlots:      req.ParallelSlots,
			KVCacheTypeK:       req.KVCacheTypeK,
			KVCacheTypeV:       req.KVCacheTypeV,
			KVCacheUnified:     req.KVCacheUnified,
			KVCacheSize:        req.KVCacheSize,
			LogitsAll:          req.LogitsAll,
			Reranking:          req.Reranking,
			MinP:               req.MinP,
			PresencePenalty:    req.PresencePenalty,
			FrequencyPenalty:   req.FrequencyPenalty,
			DirectIo:           req.DirectIo,
			DisableJinja:       req.DisableJinja,
			ChatTemplate:       req.ChatTemplate,
			ContextShift:       req.ContextShift,
			ThreadsBatch:       req.ThreadsBatch,
			RepeatLastN:        req.RepeatLastN,
			TypicalP:           req.TypicalP,
			IgnoreEOS:          req.IgnoreEOS,
			SplitMode:          req.SplitMode,
			TensorSplit:        req.TensorSplit,
			ContBatching:       req.ContBatching,
			CachePrompt:        req.CachePrompt,
			Grammar:            req.Grammar,
			GrammarFile:        req.GrammarFile,
			Lora:               req.Lora,
			LoraScaled:         req.LoraScaled,
			ChatTemplateKwargs: req.ChatTemplateKwargs,
			RopeScaling:        req.RopeScaling,
			RopeScale:          req.RopeScale,
			RopeFreqBase:       req.RopeFreqBase,
			RopeFreqScale:      req.RopeFreqScale,
		}
	}

	return br
}

func applyRuntimeConfig(status *ModelStatus, unloadAfterMinutes, concurrencyLimit int) {
	if unloadAfterMinutes > 0 {
		status.SetUnloadAfter(time.Duration(unloadAfterMinutes) * time.Minute)
	}

	if concurrencyLimit > 0 {
		status.InitConcurrency(concurrencyLimit)
	}
}
