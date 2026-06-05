// Package server provides system, config, process, and log HTTP handler methods.
package server

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/comm/gpu"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/types"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	api "github.com/simonxluo/Shepherd/internal/handler"
)

// gpuMemRe is a pre-compiled regex for GPU memory info, avoiding recompilation per request
var gpuMemRe = regexp.MustCompile(`^(.+?)\s*\((\d+)\s+MiB(?:,\s*(\d+)\s+MiB\s+free)?\)`)

// HandleServerInfo returns server information including version, ports, and status.
// @Summary      Get server info
// @Description  Returns server version, build info, status, role and configured ports
// @Tags         System
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/info [get]
func (s *Server) HandleServerInfo(c *gin.Context) {
	api.Success(c, gin.H{
		"version":   s.config.Version,
		"buildTime": s.config.BuildTime,
		"gitCommit": s.config.GitCommit,
		"goVersion": runtime.Version(),
		"name":      "Shepherd",
		"status":    "running",
		"role":      s.config.ServerCfg.Node.Role,
		"ports": gin.H{
			"web":       s.config.WebPort,
			"anthropic": s.config.AnthropicPort,
			"ollama":    s.config.ServerCfg.Compatibility.Ollama.Port,
			"lmstudio":  s.config.ServerCfg.Compatibility.LMStudio.Port,
		},
	})
}

// HandleGetGPUs returns the list of available GPUs in the system.
// @Summary      Get available GPUs
// @Description  Returns system GPU list compatible with LlamacppServer device list format. Supports specifying llama.cpp path via query parameter.
// @Tags         System
// @Produce      json
// @Param        llamacppPath query string false "Path to llama.cpp installation"
// @Success      200 {object} map[string]interface{}
// @Router       /api/system/gpus [get]
func (s *Server) HandleGetGPUs(c *gin.Context) {
	// 获取查询参数中的 llama.cpp 路径
	requestedPath := c.Query("llamacppPath")

	// 收集可能的 llama-bench 路径
	benchPaths := []string{}

	// 1. 优先使用请求指定的路径
	if requestedPath != "" {
		// 尝试多个可能的子目录
		benchPaths = append(benchPaths,
			requestedPath+"/llama-bench",
			requestedPath+"/build/bin/llama-bench",
			requestedPath+"/bin/llama-bench",
		)
	}

	// 2. 从配置路径收集
	if s.config != nil && s.config.ServerCfg != nil {
		for _, bp := range s.config.ServerCfg.BackendPaths("llamacpp") {
			if fileInfo, err := os.Stat(bp.Path); err == nil && fileInfo.IsDir() {
				benchPaths = append(benchPaths,
					bp.Path+"/llama-bench",
					bp.Path+"/build/bin/llama-bench",
					bp.Path+"/bin/llama-bench",
				)
			}
		}
	}

	// 3. 添加系统路径
	benchPaths = append(benchPaths,
		"/usr/local/bin/llama-bench",
		"/usr/bin/llama-bench",
		"llama-bench", // 使用系统 PATH
	)

	deviceStrings := []string{} // 简单设备描述字符串（兼容 LlamacppServer）
	gpus := []gin.H{}           // 详细 GPU 信息（Shepherd 扩展）

	// 尝试每个可能的路径，使用统一的设备列表解析函数
	for _, benchPath := range benchPaths {
		devices, err := utils.GetLlamacppDeviceList(benchPath)
		if err == nil && len(devices) > 0 {
			// 解析每个设备行，提取详细信息
			for _, deviceLine := range devices {
				// 解析设备行，例如: "ROCm0: AMD Radeon Graphics (122880 MiB, 115050 MiB free)"
				parts := strings.SplitN(deviceLine, ":", 2)
				if len(parts) == 2 {
					deviceID := strings.TrimSpace(parts[0])
					rest := strings.TrimSpace(parts[1])

					deviceStrings = append(deviceStrings, deviceLine)

					// 提取 GPU 名称（去掉括号中的内存信息）
					gpuName := rest
					var totalMemory, freeMemory string

					// 使用预编译正则提取内存信息
					if memMatches := gpuMemRe.FindStringSubmatch(rest); len(memMatches) > 0 {
						if totalMiB, err := strconv.ParseInt(memMatches[2], 10, 64); err == nil {
							// 转换为 GB（保留两位小数）
							totalGB := float64(totalMiB) / 1024
							totalMemory = fmt.Sprintf("%.2f GB", totalGB)
						}
						if len(memMatches) > 3 && memMatches[3] != "" {
							if freeMiB, err := strconv.ParseInt(memMatches[3], 10, 64); err == nil {
								freeGB := float64(freeMiB) / 1024
								freeMemory = fmt.Sprintf("%.2f GB", freeGB)
							}
						}
					}

					gpus = append(gpus, gin.H{
						"id":          deviceID,
						"name":        gpuName,
						"totalMemory": totalMemory,
						"freeMemory":  freeMemory,
						"available":   true,
					})
				}
			}
			// 如果成功找到设备，停止尝试其他路径
			break
		}
	}

	// 如果 llama-bench 失败，尝试使用 rocminfo 获取 GPU 信息
	// rocminfo 对于 APU 能提供更准确的内存池大小（共享系统内存）
	if len(deviceStrings) == 0 {
		rocmGPUs, err := gpu.DetectROCmGPUs()
		if err == nil && len(rocmGPUs) > 0 {
			for i, gpuInfo := range rocmGPUs {
				totalGB := float64(gpuInfo.TotalKB) / 1024 / 1024
				totalMemory := fmt.Sprintf("%.2f GB", totalGB)

				deviceID := fmt.Sprintf("ROCm%d", i)
				gpuName := gpuInfo.MarketingName
				if gpuName == "" {
					gpuName = "AMD GPU"
				}

				deviceString := fmt.Sprintf("%s: %s (%s)", deviceID, gpuName, totalMemory)
				deviceStrings = append(deviceStrings, deviceString)

				gpus = append(gpus, gin.H{
					"id":          deviceID,
					"name":        gpuName,
					"totalMemory": totalMemory,
					"freeMemory":  totalMemory,
					"available":   true,
				})
			}
		}
	}

	api.Success(c, gin.H{
		"devices": deviceStrings, // 简单设备字符串列表（兼容 LlamacppServer）
		"gpus":    gpus,          // 详细 GPU 信息（Shepherd 扩展）
		"count":   len(gpus),
	})
}

// HandleGetInferenceBackends returns the list of available inference backends.
// @Summary      Get available inference backends
// @Description  Returns available inference backends including llama.cpp paths and other backends (vLLM, vLLM-Omni)
// @Tags         System
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/system/inference-backends [get]
func (s *Server) HandleGetInferenceBackends(c *gin.Context) {
	backends := []gin.H{}
	inferenceBackends := []gin.H{}

	if s.config != nil && s.config.ServerCfg != nil {
		cfg := s.config.ServerCfg

		// Llama.cpp backends (path-based discovery)
		for _, p := range cfg.BackendPaths("llamacpp") {
			available := false
			if fileInfo, err := os.Stat(p.Path); err == nil {
				available = fileInfo.IsDir()
			}
			backends = append(backends, gin.H{
				"type":        "llamacpp",
				"path":        p.Path,
				"name":        p.Name,
				"description": p.Description,
				"available":   available,
			})
		}

		// Other backends as inferenceBackends — iterate every registered
		// plugin except llamacpp (which is reported above as path-based).
		for _, plugin := range backend.Default().List() {
			id := string(plugin.ID())
			if plugin.ID() == backend.IDLlamaCpp {
				continue
			}
			raw := cfg.BackendRaw(id)
			if raw == nil {
				continue
			}
			enabled := cfg.BackendEnabled(id)
			condaEnv, _ := raw["conda_env"].(string)
			inferenceBackends = append(inferenceBackends, gin.H{
				"type":      id,
				"name":      plugin.DisplayName(),
				"condaEnv":  condaEnv,
				"enabled":   enabled,
				"available": enabled && condaEnv != "",
			})
		}
	}

	api.Success(c, gin.H{
		"backends":          backends,
		"inferenceBackends": inferenceBackends,
		"count":             len(backends),
	})
}

// HandleListBackends returns the list of all registered backend plugins.
// @Summary      List registered backend plugins
// @Description  Returns plugin id + display name for every backend registered in the plugin registry
// @Tags         System
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/backends [get]
func (s *Server) HandleListBackends(c *gin.Context) {
	plugins := backend.Default().List()
	items := make([]gin.H, 0, len(plugins))
	for _, p := range plugins {
		items = append(items, gin.H{
			"id":          string(p.ID()),
			"displayName": p.DisplayName(),
		})
	}
	api.Success(c, gin.H{
		"backends": items,
		"count":    len(items),
	})
}

// HandleGetBackendParamSchema returns the parameter schema for a backend plugin.
// Path: /api/backends/:id/param-schema.
func (s *Server) HandleGetBackendParamSchema(c *gin.Context) {
	id := c.Param("id")
	plugin, ok := backend.Default().Get(backend.ID(id))
	if !ok {
		api.NotFound(c, "Backend plugin")
		return
	}
	api.Success(c, plugin.ParamSchema())
}

// HandlePreviewBackendCommand returns the resolved launch command without starting a process.
// Path: /api/backends/:id/preview.
func (s *Server) HandlePreviewBackendCommand(c *gin.Context) {
	id := c.Param("id")
	plugin, ok := backend.Default().Get(backend.ID(id))
	if !ok {
		api.NotFound(c, "Backend plugin")
		return
	}

	var req struct {
		BinPath   string         `json:"binPath"`
		ModelPath string         `json:"modelPath"`
		Port      int            `json:"port"`
		BindHost  string         `json:"bindHost"`
		Devices   []string       `json:"devices"`
		Params    map[string]any `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "Invalid request body")
		return
	}

	params, err := plugin.DecodeParams(backend.RawParams(req.Params))
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "Failed to decode params", err.Error())
		return
	}

	bindHost := req.BindHost
	if bindHost == "" {
		bindHost = "0.0.0.0"
	}

	// Discover to get binary path if not provided.
	binPath := req.BinPath
	if binPath == "" {
		cfg, _ := backend.Default().GetConfig(backend.ID(id))
		if cfg != nil {
			info, err := plugin.Discover(cfg)
			if err == nil && info.Available {
				binPath = info.BinPath
			}
		}
	}

	info := &backend.Info{ID: backend.ID(id), DisplayName: plugin.DisplayName(), BinPath: binPath, Available: true}
	backendReq := &backend.LoadRequest{
		ModelPath: req.ModelPath,
		Port:      req.Port,
		Devices:   req.Devices,
		BindHost:  bindHost,
		Params:    params,
	}

	startCfg, err := plugin.BuildStartConfig(info, backendReq)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "Failed to build command", err.Error())
		return
	}

	cmdLine := ""
	if startCfg.CommandSpec != nil {
		cmdLine = startCfg.CommandSpec.CommandLine()
	}
	api.Success(c, gin.H{
		"command": cmdLine,
		"spec":    startCfg.CommandSpec,
	})
}

// HandleGetConfig returns the current server configuration (excluding sensitive information).
// @Summary      Get server configuration
// @Description  Returns the current server configuration excluding sensitive information
// @Tags         Config
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/config [get]
func (s *Server) HandleGetConfig(c *gin.Context) {
	if s.config == nil || s.config.ServerCfg == nil {
		api.Error(c, types.ErrInternalError, "配置未初始化")
		return
	}

	cfg := s.config.ServerCfg

	api.Success(c, gin.H{
		"role": s.config.ServerCfg.Node.Role,
		"server": gin.H{
			"host":            s.config.Host,
			"model_bind_host": s.config.ServerCfg.Server.ModelBindHost,
			"web_port":        s.config.WebPort,
			"anthropic_port":  s.config.AnthropicPort,
			"ollama_port":     s.config.ServerCfg.Compatibility.Ollama.Port,
			"lm_studio_port":  s.config.ServerCfg.Compatibility.LMStudio.Port,
		},
		"storage": gin.H{
			"type":   cfg.Storage.Type,
			"sqlite": cfg.Storage.SQLite,
		},
		"models": gin.H{
			"paths":     cfg.Model.Paths,
			"auto_scan": cfg.Model.AutoScan,
		},
		"node": gin.H{
			"role": cfg.Node.Role,
			"id":   cfg.Node.ID,
			"name": cfg.Node.Name,
		},
		"backends": cfg.Backends,
	})
}

// HandleUpdateConfig updates the server configuration.
// @Summary      Update server configuration
// @Description  Updates server configuration including ports, scan paths, and auto-scan settings
// @Tags         Config
// @Accept       json
// @Produce      json
// @Param        request body object true "Configuration update request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /api/config [put]
func (s *Server) HandleUpdateConfig(c *gin.Context) {
	var req struct {
		Mode          string   `json:"mode"`
		WebPort       int      `json:"web_port"`
		AutoScan      bool     `json:"auto_scan"`
		ScanPaths     []string `json:"scan_paths"`
		ModelBindHost string   `json:"model_bind_host"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求格式", err.Error())
		return
	}

	restartRequired := false

	// 更新端口（需要重启）
	if req.WebPort > 0 && req.WebPort != s.config.WebPort {
		s.config.WebPort = req.WebPort
		restartRequired = true
	}

	// 更新模型绑定地址（需要重启已加载的模型才生效）
	if req.ModelBindHost != "" {
		if req.ModelBindHost != "0.0.0.0" && req.ModelBindHost != "127.0.0.1" {
			api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的绑定地址", "model_bind_host must be 0.0.0.0 or 127.0.0.1")
			return
		}
		if req.ModelBindHost != s.config.ServerCfg.Server.ModelBindHost {
			s.config.ServerCfg.Server.ModelBindHost = req.ModelBindHost
			// Re-sync backend registry so new models use the updated bind host
			if s.modelMgr != nil {
				s.modelMgr.SyncBackendRegistry(s.config.ServerCfg)
			}
			restartRequired = true
		}
	}

	// 更新扫描路径
	if len(req.ScanPaths) > 0 {
		s.config.ServerCfg.Model.Paths = req.ScanPaths

		// 触发重新扫描
		if req.AutoScan {
			go func() {
				if _, err := s.modelMgr.Scan(s.ctx); err != nil {
					logger.Warnf("模型扫描失败: error=%v", err)
				}
			}()
		}
	}

	// 持久化配置到磁盘
	if s.config.ConfigMgr != nil {
		if err := s.config.ConfigMgr.Save(s.config.ServerCfg); err != nil {
			logger.Warnf("配置持久化失败: error=%v", err)
		}
	}

	api.Success(c, gin.H{
		"message":          "配置更新成功",
		"restart_required": restartRequired,
	})
}

// HandleListProcesses returns all running and loading model processes.
// @Summary      List processes
// @Description  Returns all running and loading model server processes with their stats
// @Tags         Processes
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/processes [get]
func (s *Server) HandleListProcesses(c *gin.Context) {
	processMgr := s.modelMgr.GetProcessManager()
	if processMgr == nil {
		api.Error(c, types.ErrInternalError, "进程管理器未初始化")
		return
	}

	running, loading := processMgr.ListAll()

	// 转换为切片格式
	type ProcessInfo struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		PID     int    `json:"pid"`
		Port    int    `json:"port"`
		CtxSize int    `json:"ctx_size"`
		Running bool   `json:"running"`
		Loading bool   `json:"loading"`
	}

	var processes []ProcessInfo
	for _, p := range running {
		processes = append(processes, ProcessInfo{
			ID:      p.ID,
			Name:    p.Name,
			PID:     p.GetPID(),
			Port:    p.GetPort(),
			CtxSize: p.GetCtxSize(),
			Running: p.IsRunning(),
			Loading: false,
		})
	}
	for _, p := range loading {
		processes = append(processes, ProcessInfo{
			ID:      p.ID,
			Name:    p.Name,
			PID:     p.GetPID(),
			Port:    p.GetPort(),
			CtxSize: p.GetCtxSize(),
			Running: p.IsRunning(),
			Loading: true,
		})
	}

	api.Success(c, gin.H{
		"processes": processes,
		"stats": gin.H{
			"running": len(running),
			"loading": len(loading),
			"total":   len(running) + len(loading),
		},
	})
}

// HandleGetProcess returns details of a specific process.
// @Summary      Get process details
// @Description  Returns detailed information about a specific model server process
// @Tags         Processes
// @Produce      json
// @Param        id path string true "Process ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /api/processes/{id} [get]
func (s *Server) HandleGetProcess(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "进程ID不能为空")
		return
	}

	processMgr := s.modelMgr.GetProcessManager()
	if processMgr == nil {
		api.Error(c, types.ErrInternalError, "进程管理器未初始化")
		return
	}

	proc, exists := processMgr.Get(id)
	if !exists {
		api.NotFound(c, "进程")
		return
	}

	api.Success(c, gin.H{
		"process": gin.H{
			"id":       proc.ID,
			"name":     proc.Name,
			"cmd":      proc.Cmd,
			"bin_path": proc.BinPath,
			"pid":      proc.GetPID(),
			"port":     proc.GetPort(),
			"ctx_size": proc.GetCtxSize(),
			"running":  proc.IsRunning(),
		},
	})
}

// HandleStopProcess stops a running process by its ID.
// @Summary      Stop a process
// @Description  Stops a running model server process by its ID
// @Tags         Processes
// @Produce      json
// @Param        id path string true "Process ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/processes/{id}/stop [post]
func (s *Server) HandleStopProcess(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "进程ID不能为空")
		return
	}

	processMgr := s.modelMgr.GetProcessManager()
	if processMgr == nil {
		api.Error(c, types.ErrInternalError, "进程管理器未初始化")
		return
	}

	if err := processMgr.Stop(id); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "停止进程失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"message": "进程已停止",
		"id":      id,
	})
}

// HandleGetResources returns real-time system resource usage.
// @Summary      Get system resources
// @Description  Returns current CPU, memory, disk, GPU usage, load average, and uptime
// @Tags         System
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/system/resources [get]
func (s *Server) HandleGetResources(c *gin.Context) {
	// Try to get resources from the node adapter
	if s.nodeAdapter != nil {
		snapshot := s.nodeAdapter.GetResourceSnapshot()
		if snapshot != nil {
			// Calculate percentages
			var cpuPercent, memPercent, diskPercent float64
			if snapshot.CPUTotal > 0 {
				cpuPercent = float64(snapshot.CPUUsed) / float64(snapshot.CPUTotal) * 100
			}
			if snapshot.MemoryTotal > 0 {
				memPercent = float64(snapshot.MemoryUsed) / float64(snapshot.MemoryTotal) * 100
			}
			if snapshot.DiskTotal > 0 {
				diskPercent = float64(snapshot.DiskUsed) / float64(snapshot.DiskTotal) * 100
			}

			// Build GPU array
			gpuList := []gin.H{}
			for _, g := range snapshot.GPUInfo {
				gpuList = append(gpuList, gin.H{
					"index":         g.Index,
					"name":          g.Name,
					"vendor":        g.Vendor,
					"memoryUsed":    g.UsedMemory,
					"memoryTotal":   g.TotalMemory,
					"temperature":   g.Temperature,
					"utilization":   g.Utilization,
					"powerUsage":    g.PowerUsage,
					"driverVersion": g.DriverVersion,
				})
			}

			api.Success(c, gin.H{
				"cpu": gin.H{
					"used":    snapshot.CPUUsed,
					"total":   snapshot.CPUTotal,
					"percent": cpuPercent,
					"model":   snapshot.CPUModel,
				},
				"memory": gin.H{
					"used":    snapshot.MemoryUsed,
					"total":   snapshot.MemoryTotal,
					"percent": memPercent,
				},
				"disk": gin.H{
					"used":    snapshot.DiskUsed,
					"total":   snapshot.DiskTotal,
					"percent": diskPercent,
				},
				"gpu":           gpuList,
				"loadAverage":   snapshot.LoadAverage,
				"uptime":        snapshot.Uptime,
				"kernelVersion": snapshot.KernelVersion,
				"rocmVersion":   snapshot.ROCmVersion,
				"platform":      snapshot.Platform,
				"arch":          snapshot.Arch,
				"hostname":      snapshot.Hostname,
				"hostIp":        snapshot.HostIP,
			})
			return
		}
	}

	// Fallback: return empty/minimal response if no resource monitor available
	api.Success(c, gin.H{
		"cpu":         gin.H{"used": 0, "total": 0, "percent": 0},
		"memory":      gin.H{"used": 0, "total": 0, "percent": 0},
		"disk":        gin.H{"used": 0, "total": 0, "percent": 0},
		"gpu":         []gin.H{},
		"loadAverage": []float64{0, 0, 0},
		"uptime":      0,
	})
}

// HandleGetModelStatistics returns usage statistics for all loaded models.
// @Summary      Get model statistics
// @Description  Returns request counts, token usage, latency and uptime for loaded models
// @Tags         System
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/system/model-stats [get]
func (s *Server) HandleGetModelStatistics(c *gin.Context) {
	stats := s.modelMgr.GetModelStatistics()
	api.Success(c, gin.H{
		"models": stats,
		"count":  len(stats),
	})
}

// HandleLogStreamText streams server logs as plain text in real-time.
// @Summary      Stream server logs
// @Description  Streams server logs in real-time as plain text using chunked transfer encoding
// @Tags         Logs
// @Produce      plain
// @Param        no-history query string false "Skip sending log history"
// @Success      200 {string} string "Log stream"
// @Failure      500 {object} map[string]interface{}
// @Router       /api/logs/stream/text [get]
func (s *Server) HandleLogStreamText(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Accel-Buffering", "no")

	monitor := logger.GetMonitor()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "streaming unsupported")
		return
	}

	_, skipHistory := c.GetQuery("no-history")
	if !skipHistory {
		history := monitor.GetHistory()
		if len(history) > 0 {
			c.Writer.Write(history)
			flusher.Flush()
		}
	}

	ch := monitor.Subscribe()
	defer monitor.Unsubscribe(ch)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-ch:
			c.Writer.Write(data)
			flusher.Flush()
		case <-ticker.C:
			c.Writer.Write([]byte("\n"))
			flusher.Flush()
		}
	}
}
