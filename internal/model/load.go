package model

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
)

// prepareAndStartProcess contains the shared preparation logic for both Load and LoadAsync.
// It finds the llama.cpp binary, allocates a port, resolves the model path, builds the command,
// starts the process, and sets up the output handler.
// On error, it updates status to StateError and releases any allocated port.
func (m *Manager) prepareAndStartProcess(req *LoadRequest, model *Model, status *ModelStatus) (*process.Process, int, error) {
	// Find llama.cpp binary
	binPath := m.findLlamaCppBinary()
	if binPath == "" {
		err := fmt.Errorf("llama.cpp binary not found")
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		return nil, 0, err
	}

	// Allocate port
	allocatedPort, err := m.portAllocator.NextPort()
	if err != nil {
		wrappedErr := fmt.Errorf("no available ports: %w", err)
		m.mu.Lock()
		status.State = StateError
		status.Error = wrappedErr
		m.mu.Unlock()
		return nil, 0, wrappedErr
	}

	// Determine model path (use first shard for split models)
	modelPath := model.Path
	if len(model.ShardFiles) > 0 {
		modelPath = model.ShardFiles[0]
		logger.Info("using shard model primary file", "modelId", req.ModelID, "path", modelPath, "shardCount", len(model.ShardFiles))
	}

	// Build command
	procReq := toProcessLoadRequest(req, modelPath, allocatedPort)
	cmd, err := process.BuildCommandFromRequest(procReq, binPath)
	if err != nil {
		m.portAllocator.Release(allocatedPort)
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		return nil, 0, err
	}

	// Start process
	proc, err := m.processMgr.Start(req.ModelID, model.Name, cmd, binPath)
	if err != nil {
		m.portAllocator.Release(allocatedPort)
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		return nil, 0, err
	}

	// Set output handler to forward llama.cpp logs
	proc.SetOutputHandler(func(line string) {
		if !strings.Contains(line, "update_slots") && !strings.Contains(line, "log_server_r") {
			logger.Debug(fmt.Sprintf("[%s] %s", req.ModelID, line))
		}
	})

	return proc, allocatedPort, nil
}

// LoadAsync 异步加载模型（立即返回，后台加载）
func (m *Manager) LoadAsync(req *LoadRequest) (*LoadResult, error) {
	// Get model
	model, exists := m.GetModel(req.ModelID)
	if !exists {
		return nil, fmt.Errorf("model not found: %s", req.ModelID)
	}

	// Check if already loaded
	m.mu.RLock()
	if status, exists := m.statuses[req.ModelID]; exists {
		if status.State == StateLoaded {
			m.mu.RUnlock()
			return &LoadResult{
				Success:       true,
				ModelID:       req.ModelID,
				Port:          status.Port,
				Async:         true,
				AlreadyLoaded: true,
			}, nil
		}
		if status.State == StateLoading {
			m.mu.RUnlock()
			return &LoadResult{
				Success: true,
				ModelID: req.ModelID,
				Async:   true,
				Loading: true,
			}, nil
		}
	}
	m.mu.RUnlock()

	// 创建初始状态
	m.mu.Lock()
	status := &ModelStatus{
		ID:    req.ModelID,
		Name:  model.Name,
		State: StateLoading,
	}
	m.statuses[req.ModelID] = status
	m.mu.Unlock()

	// 启动异步加载（传入已获取的 model，避免在 goroutine 中再次获取锁）
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

	logger.Info("开始异步加载模型", "modelId", req.ModelID, "modelName", model.Name)

	// 根据模型大小计算超时时间
	timeout := m.calculateLoadTimeout(model.Size)
	logger.Info("模型加载超时设置", "modelId", req.ModelID, "sizeGB", float64(model.Size)/(1024*1024*1024), "timeout", timeout)

	// 准备并启动进程
	proc, port, err := m.prepareAndStartProcess(req, model, status)
	if err != nil {
		logger.Error("异步模型加载失败", "modelId", req.ModelID, "error", err)
		return
	}

	logger.Info("进程启动成功，准备获取 PID", "modelId", req.ModelID)

	// 获取 PID (可能在短时间内阻塞，但应该很快返回)
	pid := proc.GetPID()
	logger.Info("异步模型加载: 进程已启动", "modelId", req.ModelID, "pid", pid, "port", port)

	// 等待加载完成（监控进程输出）
	loadCompleted := make(chan bool, 1)
	loadError := make(chan error, 1)
	stopHealthCheck := make(chan bool, 1) // 用于停止健康检查

	// 启动进程健康检查 goroutine
	logger.Info("启动健康检查 goroutine", "modelId", req.ModelID, "healthURL", fmt.Sprintf("http://localhost:%d/health", port))
	go func() {
		// 立即执行一次健康检查（不等待 ticker）
		httpClient := &http.Client{Timeout: 2 * time.Second}
		healthURL := fmt.Sprintf("http://localhost:%d/health", port)

		// 启动 ticker 进行定期检查
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		// 健康检查失败计数器，超过10次则放弃
		failureCount := 0
		const maxFailures = 10

		// 首次立即检查
		func() {
			logger.Info("首次健康检查", "modelId", req.ModelID, "url", healthURL)
			if !proc.IsRunning() {
				logger.Error("首次健康检查发现进程已退出", "modelId", req.ModelID)
				select {
				case loadError <- fmt.Errorf("进程启动后立即退出"):
				default:
				}
				return
			}

			resp, err := httpClient.Get(healthURL)
			if err != nil {
				failureCount++
				logger.Warn("首次健康检查请求失败", "modelId", req.ModelID, "error", err, "failureCount", failureCount, "maxFailures", maxFailures)
			} else if resp.StatusCode != 200 {
				failureCount++
				logger.Warn("首次健康检查返回非200状态", "modelId", req.ModelID, "statusCode", resp.StatusCode, "failureCount", failureCount, "maxFailures", maxFailures)
				utils.CloseQuietly(resp.Body)
			} else {
				body, _ := io.ReadAll(resp.Body)
				utils.CloseQuietly(resp.Body)
				logger.Info("首次健康检查响应", "modelId", req.ModelID, "body", string(body))
				if strings.Contains(string(body), `"status":"ok"`) {
					logger.Info("首次健康检查成功，模型已就绪", "modelId", req.ModelID, "port", port)
					select {
					case loadCompleted <- true:
					default:
					}
					return
				}
			}
		}()

		// 定期检查
		checkCount := 0
		for {
			select {
			case <-stopHealthCheck:
				// 收到停止信号，退出健康检查
				logger.Info("健康检查收到停止信号", "modelId", req.ModelID, "checkCount", checkCount)
				return
			case <-ticker.C:
				checkCount++
				logger.Info("执行定期健康检查", "modelId", req.ModelID, "checkCount", checkCount, "url", healthURL)

				// 定期检查进程是否仍在运行
				if !proc.IsRunning() {
					// 进程意外退出
					logger.Error("定期健康检查发现进程已退出", "modelId", req.ModelID, "pid", proc.GetPID())
					select {
					case loadError <- fmt.Errorf("进程意外退出 (PID: %d)", proc.GetPID()):
					default:
					}
					return
				}

				// 检查 HTTP 健康端点（备用检测机制）
				resp, err := httpClient.Get(healthURL)
				if err != nil {
					failureCount++
					logger.Warn("HTTP 健康检查请求失败", "modelId", req.ModelID, "url", healthURL, "error", err, "failureCount", failureCount, "maxFailures", maxFailures)

					// 检查是否超过最大失败次数
					if failureCount >= maxFailures {
						logger.Error("健康检查失败次数超过限制，终止进程", "modelId", req.ModelID, "failureCount", failureCount, "maxFailures", maxFailures)
						// 杀死进程
						utils.StopQuietly(proc)
						// 返回错误
						select {
						case loadError <- fmt.Errorf("健康检查连续失败 %d 次，已终止进程 (PID: %d)", failureCount, proc.GetPID()):
						default:
						}
						return
					}
				} else if resp.StatusCode != 200 {
					failureCount++
					logger.Warn("HTTP 健康检查返回非200状态", "modelId", req.ModelID, "statusCode", resp.StatusCode, "failureCount", failureCount, "maxFailures", maxFailures)
					utils.CloseQuietly(resp.Body)

					// 检查是否超过最大失败次数
					if failureCount >= maxFailures {
						logger.Error("健康检查失败次数超过限制，终止进程", "modelId", req.ModelID, "failureCount", failureCount, "maxFailures", maxFailures)
						// 杀死进程
						utils.StopQuietly(proc)
						// 返回错误
						select {
						case loadError <- fmt.Errorf("健康检查连续失败 %d 次，已终止进程 (PID: %d)", failureCount, proc.GetPID()):
						default:
						}
						return
					}
				} else {
					body, _ := io.ReadAll(resp.Body)
					utils.CloseQuietly(resp.Body)
					logger.Info("HTTP 健康检查响应", "modelId", req.ModelID, "body", string(body))
					// 检查响应内容是否为 {"status":"ok"}
					if strings.Contains(string(body), `"status":"ok"`) {
						logger.Info("HTTP 健康检查成功，模型已就绪", "modelId", req.ModelID, "port", port, "checkCount", checkCount)
						select {
						case loadCompleted <- true:
						default:
						}
						return
					}
					// 响应成功但状态不是 ok，重置失败计数
					failureCount = 0
				}
			}
		}
	}()

	// 设置输出处理器检测加载完成并转发日志
	proc.SetOutputHandler(func(line string) {
		// 将 llama.cpp 输出转发到日志系统
		// 过滤掉过于频繁的日志
		if !strings.Contains(line, "update_slots") && !strings.Contains(line, "log_server_r") {
			// 使用 debug 级别记录 llama.cpp 输出，避免日志过多
			logger.Debug(fmt.Sprintf("[%s] %s", req.ModelID, line))
		}

		// 检测加载完成
		if strings.Contains(line, "all slots are idle") {
			select {
			case loadCompleted <- true:
			default:
			}
		}
	})

	// 等待加载完成或超时
	select {
	case <-loadCompleted:
		// 停止健康检查
		close(stopHealthCheck)
		m.mu.Lock()
		status.State = StateLoaded
		status.ProcessID = proc.ID
		status.Port = port
		status.LoadedAt = time.Now()
		m.mu.Unlock()
		duration := time.Since(startTime)
		logger.Info("异步模型加载成功", "modelId", req.ModelID, "port", port, "duration", duration.String())

	case err := <-loadError:
		// 停止健康检查
		close(stopHealthCheck)
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		logger.Error("异步模型加载失败", "modelId", req.ModelID, "error", err)
		// 清理进程和端口
		m.processMgr.Stop(req.ModelID) //errcheck:ignore
		m.portAllocator.Release(port)

	case <-time.After(timeout):
		// 停止健康检查
		close(stopHealthCheck)
		m.mu.Lock()
		status.State = StateError
		status.Error = fmt.Errorf("模型加载超时 (%v)", timeout)
		m.mu.Unlock()
		logger.Error("异步模型加载超时", "modelId", req.ModelID, "timeout", timeout)
		// 清理进程和端口
		m.processMgr.Stop(req.ModelID) //errcheck:ignore
		m.portAllocator.Release(port)
	}
}

// calculateLoadTimeout 根据模型大小计算动态超时时间
// 规则：每GB 1分钟，最少5分钟，最多30分钟
func (m *Manager) calculateLoadTimeout(modelSize int64) time.Duration {
	const (
		minTimeout   = 5 * time.Minute
		maxTimeout   = 30 * time.Minute
		minutesPerGB = 1 * time.Minute
		gigabyte     = int64(1024 * 1024 * 1024)
	)

	// 计算基于模型大小的超时时间
	sizeGB := float64(modelSize) / float64(gigabyte)
	dynamicTimeout := time.Duration(sizeGB) * minutesPerGB

	// 限制在最小和最大值之间
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
		logger.Warn("模型卸载失败: 模型未加载", "modelId", modelID)
		return fmt.Errorf("model not loaded: %s", modelID)
	}

	if status.State != StateLoaded {
		logger.Warn("模型卸载失败: 模型未处于已加载状态", "modelId", modelID, "state", status.State)
		return fmt.Errorf("model not in loaded state: %s", modelID)
	}

	logger.Info("开始卸载模型", "modelId", modelID, "modelName", status.Name, "port", status.Port)

	// Stop process
	if err := m.processMgr.Stop(modelID); err != nil {
		logger.Error("模型卸载失败: 停止进程失败", "modelId", modelID, "error", err)
		return err
	}

	// Release allocated port
	if status.Port > 0 {
		m.portAllocator.Release(status.Port)
		logger.Info("已释放端口", "modelId", modelID, "port", status.Port)
	}

	// Update status
	status.State = StateUnloaded
	status.ProcessID = ""
	status.Port = 0

	logger.Info("模型卸载成功", "modelId", modelID, "modelName", status.Name)

	return nil
}

// findLlamaCppBinary finds the llama.cpp binary
// 从配置管理器获取最新配置，而不是使用初始化时的静态快照
func (m *Manager) findLlamaCppBinary() string {
	// 从配置管理器获取最新配置
	cfg := m.configMgr.Get()

	// Check configured paths
	for _, llamacppPath := range cfg.Llamacpp.Paths {
		binaryPath := filepath.Join(llamacppPath.Path, "llama-server")
		if _, err := os.Stat(binaryPath); err == nil {
			return llamacppPath.Path
		}
	}

	// Check common locations
	commonPaths := []string{
		"/usr/local/bin",
		"/usr/bin",
		"./llama.cpp",
	}

	for _, path := range commonPaths {
		binaryPath := filepath.Join(path, "llama-server")
		if _, err := os.Stat(binaryPath); err == nil {
			return path
		}
	}

	return ""
}

// toProcessLoadRequest converts model.LoadRequest to process.LoadRequest for command building
// This bridges the canonical LoadRequest (with ModelID/NodeID) to the command-building LoadRequest (with ModelPath/Port)
func toProcessLoadRequest(req *LoadRequest, modelPath string, port int) *process.LoadRequest {
	return &process.LoadRequest{
		ModelPath:        modelPath,
		Port:             port,
		CtxSize:          req.CtxSize,
		BatchSize:        req.BatchSize,
		Threads:          req.Threads,
		GPULayers:        req.GPULayers,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		TopK:             req.TopK,
		RepeatPenalty:    req.RepeatPenalty,
		Seed:             req.Seed,
		NPredict:         req.NPredict,
		Devices:          req.Devices,
		MainGPU:          req.MainGPU,
		CustomCmd:        req.CustomCmd,
		ExtraParams:      req.ExtraParams,
		MmprojPath:       req.MmprojPath,
		EnableVision:     req.EnableVision,
		FlashAttention:   req.FlashAttention,
		NoMmap:           req.NoMmap,
		LockMemory:       req.LockMemory,
		NoWebUI:          req.NoWebUI,
		EnableMetrics:    req.EnableMetrics,
		SlotSavePath:     req.SlotSavePath,
		CacheRAM:         req.CacheRAM,
		ChatTemplateFile: req.ChatTemplateFile,
		Timeout:          req.Timeout,
		Alias:            req.Alias,
		UBatchSize:       req.UBatchSize,
		ParallelSlots:    req.ParallelSlots,
		KVCacheTypeK:     req.KVCacheTypeK,
		KVCacheTypeV:     req.KVCacheTypeV,
		KVCacheUnified:   req.KVCacheUnified,
		KVCacheSize:      req.KVCacheSize,
		// Additional sampling parameters
		LogitsAll:        req.LogitsAll,
		Reranking:        req.Reranking,
		MinP:             req.MinP,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
		// Template and processing
		DirectIo:     req.DirectIo,
		DisableJinja: req.DisableJinja,
		ChatTemplate: req.ChatTemplate,
		ContextShift: req.ContextShift,
		// Thread configuration
		ThreadsBatch: req.ThreadsBatch,
		// Extended sampling parameters
		RepeatLastN: req.RepeatLastN,
		TypicalP:    req.TypicalP,
		IgnoreEOS:   req.IgnoreEOS,
		// Multi-GPU configuration
		SplitMode:   req.SplitMode,
		TensorSplit: req.TensorSplit,
		// Server optimization
		ContBatching: req.ContBatching,
		CachePrompt:  req.CachePrompt,
		// Structured generation
		Grammar:     req.Grammar,
		GrammarFile: req.GrammarFile,
		// LoRA adapter support
		Lora:       req.Lora,
		LoraScaled: req.LoraScaled,
		// Chat template kwargs
		ChatTemplateKwargs: req.ChatTemplateKwargs,
		// RoPE scaling
		RopeScaling:   req.RopeScaling,
		RopeScale:     req.RopeScale,
		RopeFreqBase:  req.RopeFreqBase,
		RopeFreqScale: req.RopeFreqScale,
	}
}
