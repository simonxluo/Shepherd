// Package server provides system, config, process, and log HTTP handler methods.
package server

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/gpu"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	api "github.com/shepherd-project/shepherd/Shepherd/internal/handler"
)

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
		"name":      "Shepherd",
		"status":    "running",
		"role":      s.config.ServerCfg.Node.Role,
		"ports": gin.H{
			"web":       s.config.WebPort,
			"anthropic": s.config.AnthropicPort,
			"ollama":    s.config.OllamaPort,
			"lmstudio":  s.config.LMStudioPort,
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
	if s.config != nil && s.config.ServerCfg != nil && len(s.config.ServerCfg.Llamacpp.Paths) > 0 {
		for _, p := range s.config.ServerCfg.Llamacpp.Paths {
			if fileInfo, err := os.Stat(p.Path); err == nil && fileInfo.IsDir() {
				benchPaths = append(benchPaths,
					p.Path+"/llama-bench",
					p.Path+"/build/bin/llama-bench",
					p.Path+"/bin/llama-bench",
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

					// 使用正则表达式提取内存信息和分离名称
					memRe := regexp.MustCompile(`^(.+?)\s*\((\d+)\s+MiB(?:,\s*(\d+)\s+MiB\s+free)?\)`)
					if memMatches := memRe.FindStringSubmatch(rest); len(memMatches) > 0 {
						gpuName = strings.TrimSpace(memMatches[1])
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

// HandleGetLlamacppBackends returns the list of available inference backends.
// @Summary      Get available llama.cpp backends
// @Description  Returns available inference backends including llama.cpp paths and other backends (vLLM, vLLM-Omni)
// @Tags         System
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/system/llamacpp-backends [get]
func (s *Server) HandleGetLlamacppBackends(c *gin.Context) {
	backends := []gin.H{}
	inferenceBackends := []gin.H{}

	if s.config != nil && s.config.ServerCfg != nil {
		// Llama.cpp backends (backward compatible)
		paths := s.config.ServerCfg.Llamacpp.Paths
		for _, p := range paths {
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

		// vLLM backend (separate array to avoid breaking frontend)
		if s.config.ServerCfg.Backends.VLLM != nil {
			vcfg := s.config.ServerCfg.Backends.VLLM
			inferenceBackends = append(inferenceBackends, gin.H{
				"type":      "vllm",
				"name":      "vLLM",
				"condaEnv":  vcfg.CondaEnv,
				"enabled":   vcfg.Enabled,
				"available": vcfg.Enabled && vcfg.CondaEnv != "",
			})
		}

		// vLLM-omni backend
		if s.config.ServerCfg.Backends.VLLMOmni != nil {
			ocfg := s.config.ServerCfg.Backends.VLLMOmni
			inferenceBackends = append(inferenceBackends, gin.H{
				"type":      "vllm_omni",
				"name":      "vLLM-Omni",
				"condaEnv":  ocfg.CondaEnv,
				"enabled":   ocfg.Enabled,
				"available": ocfg.Enabled && ocfg.CondaEnv != "",
			})
		}
	}

	api.Success(c, gin.H{
		"backends":          backends,
		"inferenceBackends": inferenceBackends,
		"count":             len(backends),
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
			"host":           s.config.Host,
			"web_port":       s.config.WebPort,
			"anthropic_port": s.config.AnthropicPort,
			"ollama_port":    s.config.OllamaPort,
			"lm_studio_port": s.config.LMStudioPort,
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
		"llamacpp": gin.H{
			"paths": cfg.Llamacpp.Paths,
		},
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
		Mode      string   `json:"mode"`
		WebPort   int      `json:"web_port"`
		AutoScan  bool     `json:"auto_scan"`
		ScanPaths []string `json:"scan_paths"`
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

	// 更新扫描路径
	if len(req.ScanPaths) > 0 {
		s.config.ServerCfg.Model.Paths = req.ScanPaths

		// 触发重新扫描
		if req.AutoScan {
			go func() {
				if _, err := s.modelMgr.Scan(c.Request.Context()); err != nil {
					logger.Warnf("模型扫描失败: error=%v", err)
				}
			}()
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
					"index":       g.Index,
					"name":        g.Name,
					"vendor":      g.Vendor,
					"memoryUsed":  g.UsedMemory,
					"memoryTotal": g.TotalMemory,
				})
			}

			api.Success(c, gin.H{
				"cpu": gin.H{
					"used":    snapshot.CPUUsed,
					"total":   snapshot.CPUTotal,
					"percent": cpuPercent,
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
