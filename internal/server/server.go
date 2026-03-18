// Package server provides the HTTP server for the Shepherd application.
// It handles HTTP requests, routing, middleware, and serves the web UI.
package server

import (
	"context"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	api "github.com/shepherd-project/shepherd/Shepherd/internal/api"
	"github.com/shepherd-project/shepherd/Shepherd/internal/api/anthropic"
	benchmarkapi "github.com/shepherd-project/shepherd/Shepherd/internal/api/benchmark"
	compatibilityapi "github.com/shepherd-project/shepherd/Shepherd/internal/api/compatibility"
	filesystemapi "github.com/shepherd-project/shepherd/Shepherd/internal/api/filesystem"
	"github.com/shepherd-project/shepherd/Shepherd/internal/api/ollama"
	"github.com/shepherd-project/shepherd/Shepherd/internal/api/openai"
	"github.com/shepherd-project/shepherd/Shepherd/internal/api/paths"
	storageapi "github.com/shepherd-project/shepherd/Shepherd/internal/api/storage"
	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/download"
	"github.com/shepherd-project/shepherd/Shepherd/internal/langchain"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
	modelrepoclient "github.com/shepherd-project/shepherd/Shepherd/internal/modelrepo"
	"github.com/shepherd-project/shepherd/Shepherd/internal/storage"
	"github.com/shepherd-project/shepherd/Shepherd/internal/types"
	"github.com/shepherd-project/shepherd/Shepherd/internal/websocket"
)

// ModelDTO represents a model for API responses
type ModelDTO struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Alias       string                 `json:"alias"`
	Path        string                 `json:"path"`
	PathPrefix  string                 `json:"pathPrefix"`
	Size        int64                  `json:"size"`
	TotalSize   int64                  `json:"totalSize,omitempty"`  // 包含所有分卷的总大小
	ShardCount  int                    `json:"shardCount,omitempty"` // 分卷数量
	ShardFiles  []string               `json:"shardFiles,omitempty"` // 所有分卷文件路径
	MmprojPath  string                 `json:"mmprojPath,omitempty"` // mmproj 文件路径
	Favourite   bool                   `json:"favourite"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      string                 `json:"status"`
	IsLoaded    bool                   `json:"isLoaded"`
	ScannedAt   string                 `json:"scannedAt,omitempty"` // 扫描时间（ISO 8601 格式）
}

// nonEmptyString 返回非空字符串，如果是空字符串则返回 nil
// 这样 JSON 序列化时会省略该字段而不是返回空字符串
func nonEmptyString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// Server represents the HTTP server
type Server struct {
	engine           *gin.Engine
	httpServer       *http.Server
	config           *Config
	handlers         *Handlers
	wsMgr            *websocket.Manager
	modelMgr         *model.Manager
	storageMgr       *storage.Manager
	downloadMgr      *download.Manager       // 下载管理器
	nodeAdapter      *api.NodeAdapter        // Node API 适配器
	repoClient       *modelrepoclient.Client // 模型仓库客户端
	langchainHandler *langchain.Handler      // LangChainGo API 处理器

	// 新增字段：WebSocket Hub
	wsHub             *WebSocketHub
	downloadTasksFile string // 下载任务持久化文件路径

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Config contains server configuration
type Config struct {
	WebPort       int
	AnthropicPort int
	OllamaPort    int
	LMStudioPort  int
	Host          string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	WebUIPath     string
	ServerCfg     *config.Config
	ConfigMgr     *config.Manager // 配置管理器
	// Version information
	Version   string // 版本号
	BuildTime string // 构建时间
	GitCommit string // Git commit hash
}

// Handlers contains handler instances
type Handlers struct {
	OpenAI        *openai.Handler
	Ollama        *ollama.Handler
	Anthropic     *anthropic.Handler
	Paths         *paths.Handler
	Storage       *storageapi.Handler
	Compatibility *compatibilityapi.Handler
	Filesystem    *filesystemapi.Handler
	Benchmark     *benchmarkapi.Handler
}

// NewServer creates a new HTTP server
func NewServer(config *Config, modelMgr *model.Manager) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
		handlers: &Handlers{},
		modelMgr: modelMgr,
	}

	// Initialize storage manager
	storageMgr, err := storage.NewManager(&config.ServerCfg.Storage)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize storage manager: %w", err)
	}
	s.storageMgr = storageMgr

	// Create download manager
	s.downloadMgr = download.NewManager(download.DownloadConfig{MaxConcurrent: 3}) // 最多3个并发下载

	// Create WebSocket manager
	s.wsMgr = websocket.NewManager(modelMgr)

	// Create WebSocket Hub (新增)
	s.wsHub = NewWebSocketHub()

	// Create model repository client with config
	cfg := config.ConfigMgr.Get()
	timeout := time.Duration(cfg.ModelRepo.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	s.repoClient = modelrepoclient.NewClientWithConfig(cfg.ModelRepo.Endpoint, cfg.ModelRepo.Token, timeout)

	// Create compatibility server manager
	compatServerManager := compatibilityapi.NewServerManager(modelMgr)

	// Create API handlers
	s.handlers.OpenAI = openai.NewHandler(modelMgr)
	s.handlers.Ollama = ollama.NewHandler(modelMgr)
	s.handlers.Anthropic = anthropic.NewHandler(modelMgr)
	s.handlers.Paths = paths.NewHandler(config.ConfigMgr)
	s.handlers.Storage = storageapi.NewHandler(config.ConfigMgr, storageMgr)
	s.handlers.Compatibility = compatibilityapi.NewHandler(config.ConfigMgr, compatServerManager)
	s.handlers.Filesystem = filesystemapi.NewHandler()
	// 压测 handler 使用存储层管理数据
	s.handlers.Benchmark = benchmarkapi.NewHandler(logger.GetLogger(), storageMgr.GetStore())

	// Setup Gin engine
	if config.WebUIPath == "" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	s.engine = gin.New()
	s.setupMiddleware()
	s.setupRoutes()

	return s, nil
}

// setupMiddleware configures server middleware
func (s *Server) setupMiddleware() {
	s.engine.Use(
		api.RequestID(), // 请求 ID 追踪
		api.RecoveryMiddleware(logger.GetLogger()), // 统一恢复中间件
		api.CORSMiddleware([]string{"*"}),          // 统一 CORS
		api.LoggerMiddleware(logger.GetLogger()),   // 统一日志
		api.ErrorHandler(logger.GetLogger()),       // 统一错误处理
	)
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// WebSocket endpoint (for SSE)
	s.engine.GET("/api/events", s.handleEvents)

	// WebSocket endpoint (新增)
	s.engine.GET("/ws", s.handleWebSocket)

	// API routes
	api := s.engine.Group("/api")
	{
		// Server info
		api.GET("/info", s.handleServerInfo)

		// System info
		api.GET("/system/gpus", s.handleGetGPUs)
		api.GET("/system/llamacpp-backends", s.handleGetLlamacppBackends)

		// Configuration routes
		config := api.Group("/config")
		{
			config.GET("", s.handleGetConfig)
			config.PUT("", s.handleUpdateConfig)

			// Llama.cpp paths
			llamacpp := config.Group("/llamacpp/paths")
			{
				llamacpp.GET("", s.handlers.Paths.GetLlamaCppPaths)
				llamacpp.POST("", s.handlers.Paths.AddLlamaCppPath)
				llamacpp.PUT("", s.handlers.Paths.UpdateLlamaCppPath)
				llamacpp.DELETE("", s.handlers.Paths.RemoveLlamaCppPath)
				llamacpp.POST("/test", s.handlers.Paths.TestLlamaCppPath)
			}

			// Model paths
			models := config.Group("/models/paths")
			{
				models.GET("", s.handlers.Paths.GetModelPaths)
				models.POST("", s.handlers.Paths.AddModelPath)
				models.PUT("", s.handlers.Paths.UpdateModelPath)
				models.DELETE("", s.handlers.Paths.RemoveModelPath)
			}

			// Storage configuration
			storage := config.Group("/storage")
			{
				storage.GET("", s.handlers.Storage.GetStorageConfig)
				storage.PUT("", s.handlers.Storage.UpdateStorageConfig)
				storage.GET("/stats", s.handlers.Storage.GetStats)
			}

			// Compatibility configuration
			compatibility := config.Group("/compatibility")
			{
				compatibility.GET("", s.handlers.Compatibility.GetCompatibility)
				compatibility.PUT("", s.handlers.Compatibility.UpdateCompatibility)
				compatibility.POST("/test", s.handlers.Compatibility.TestConnection)
			}
		}

		// Chat/Conversation routes
		conversations := api.Group("/conversations")
		{
			conversations.GET("", s.handlers.Storage.GetConversations)
			conversations.GET("/:id", s.handlers.Storage.GetConversation)
			conversations.DELETE("/:id", s.handlers.Storage.DeleteConversation)
		}

		// Model routes
		models := api.Group("/models")
		{
			models.GET("", s.handleListModels)
			models.GET("/loaded", s.handleListLoadedModels)

			// 模型能力管理（必须在 :id 路由之前）
			models.GET("/capabilities/get", s.handleGetModelCapabilities)
			models.POST("/capabilities/set", s.handleSetModelCapabilities)
			models.GET("/capabilities/auto-detect", s.handleAutoDetectCapabilities)

			// 显存估算（必须在 :id 路由之前）
			models.POST("/vram/estimate", s.handleEstimateVRAM)

			// Benchmark 压测路由（必须在 :id 路由之前）
			benchmark := models.Group("/benchmark")
			{
				benchmark.POST("", s.handlers.Benchmark.Create)
				benchmark.GET("/tasks", s.handlers.Benchmark.List)
				benchmark.GET("/tasks/:benchmarkId", s.handlers.Benchmark.Get)
				benchmark.POST("/tasks/:benchmarkId/cancel", s.handlers.Benchmark.Cancel)
				// 配置管理
				benchmark.GET("/configs", s.handlers.Benchmark.ListConfigs)
				benchmark.POST("/configs", s.handlers.Benchmark.SaveConfig)
				benchmark.GET("/configs/:name", s.handlers.Benchmark.GetConfig)
				benchmark.DELETE("/configs/:name", s.handlers.Benchmark.DeleteConfig)
			}

			// 模型具体操作（包含 :id 参数的路由必须在最后）
			models.GET("/:id", s.handleGetModel)
			models.POST("/:id/load", s.handleLoadModel)
			models.POST("/:id/unload", s.handleUnloadModel)
			models.PUT("/:id/alias", s.handleSetAlias)
			models.PUT("/:id/favourite", s.handleSetFavourite)

			// 模型加载配置管理
			models.GET("/:id/load-config", s.handleGetModelLoadConfig)
			models.PUT("/:id/load-config", s.handleSaveModelLoadConfig)
			models.DELETE("/:id/load-config", s.handleDeleteModelLoadConfig)
		}

		// Model scan routes
		modelScan := api.Group("/model/scan")
		{
			modelScan.POST("", s.handleScanModels)
			modelScan.GET("/status", s.handleGetScanStatus)
		}

		// Model device routes
		api.GET("/model/device/list", s.handlers.Benchmark.GetDevices)

		// Model parameter routes
		api.GET("/models/param/benchmark/list", s.handlers.Benchmark.GetParams)

		// Llama.cpp routes
		api.GET("/llamacpp/list", s.handlers.Paths.GetLlamaCppPaths)

		// Download routes
		downloads := api.Group("/downloads")
		{
			downloads.GET("", s.handleListDownloads)
			downloads.POST("", s.handleCreateDownload)
			downloads.GET("/:id", s.handleGetDownload)
			downloads.POST("/:id/pause", s.handlePauseDownload)
			downloads.POST("/:id/resume", s.handleResumeDownload)
			downloads.DELETE("/:id", s.handleDeleteDownload)
		}

		// Model repository routes (远程模型仓库文件浏览)
		// 路由格式: /api/repo/files?source=huggingface&repoId=Qwen/Qwen2-7B-Instruct
		// 使用查询参数以支持 repoId 中包含斜杠
		repo := api.Group("/repo")
		{
			repo.GET("/files", s.handleListModelFiles)
			repo.GET("/search", s.handleSearchModels)
			repo.GET("/config", s.handleGetModelRepoConfig)
			repo.PUT("/config", s.handleUpdateModelRepoConfig)
			repo.GET("/endpoints", s.handleGetAvailableEndpoints)
		}

		// Process routes
		processes := api.Group("/processes")
		{
			processes.GET("", s.handleListProcesses)
			processes.GET("/:id", s.handleGetProcess)
			processes.POST("/:id/stop", s.handleStopProcess)
		}

		// Log routes
		logs := api.Group("/logs")
		{
			logs.GET("/stream", s.handleLogStream)
			logs.GET("/entries", s.handleLogEntries)
			logs.GET("/files", s.handleLogFiles)
			logs.GET("/files/:filename", s.handleLogFileContent)
			logs.GET("/files/:filename/stats", s.handleLogFileStats)
			logs.DELETE("/files/:filename", s.handleDeleteLogFile)
		}

		// System routes
		system := api.Group("/system")
		{
			system.GET("/filesystem", s.handlers.Filesystem.ListDirectory)
			system.POST("/filesystem/validate", s.handlers.Filesystem.ValidatePath)
		}
	}

	// OpenAI compatible API
	openai := s.engine.Group("/v1")
	{
		openai.POST("/chat/completions", s.handleOpenAIChat)
		openai.POST("/completions", s.handleOpenAIComplete)
		openai.GET("/models", s.handleOpenAIModels)
	}

	// Anthropic compatible API
	anthropic := s.engine.Group("/v1")
	{
		anthropic.POST("/messages", s.handleAnthropicMessages)
	}

	// Ollama compatible API
	ollama := s.engine.Group("/api")
	{
		ollama.POST("/chat", s.handleOllamaChat)
		ollama.POST("/tags", s.handleOllamaTags)
	}

	// Static files for Web UI
	s.engine.Static("/assets", s.config.WebUIPath+"/assets")
	s.engine.Static("/favicon.svg", s.config.WebUIPath+"/favicon.svg")
	s.engine.StaticFile("/", s.config.WebUIPath+"/index.html")
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already started
	if s.httpServer != nil {
		return fmt.Errorf("server already started")
	}

	// Start WebSocket manager
	s.wsMgr.Start()

	// Start WebSocket Hub (新增)
	go s.wsHub.Run()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.WebPort)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Start server in background
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		logger.Infof("启动 HTTP 服务器，监听 %s", addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP 服务器错误: %v", err)
		}
		logger.Info("HTTP 服务器已停止")
	}()

	return nil
}

// Stop stops the HTTP server gracefully
func (s *Server) Stop() error {
	s.mu.Lock()
	if s.httpServer == nil {
		s.mu.Unlock()
		return fmt.Errorf("server not started")
	}
	s.mu.Unlock()

	logger.Info("开始停止 HTTP 服务器...")

	// Step 1: Cancel context to signal all goroutines
	s.cancel()

	// Step 2: Stop accepting new connections (but don't close existing ones)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 3: Shutdown HTTP server gracefully
	s.mu.Lock()
	if s.httpServer != nil {
		logger.Info("关闭 HTTP 服务器...")
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("HTTP 服务器关闭失败: %v", err)
			// Force close if graceful shutdown fails
			utils.CloseQuietly(s.httpServer)
		} else {
			logger.Info("HTTP 服务器已优雅关闭")
		}
		s.httpServer = nil
	}
	s.mu.Unlock()

	// Step 4: Stop WebSocket manager
	logger.Info("停止 WebSocket 管理器...")
	s.wsMgr.Stop()
	logger.Info("WebSocket 管理器已停止")

	// Step 4.5: Stop download manager
	logger.Info("停止下载管理器...")
	if s.downloadMgr != nil {
		if err := s.downloadMgr.Close(); err != nil {
			logger.Errorf("下载管理器关闭失败: %v", err)
		} else {
			logger.Info("下载管理器已停止")
		}
	}

	// Step 5: Close storage manager
	logger.Info("关闭存储管理器...")
	if s.storageMgr != nil {
		if err := s.storageMgr.Close(); err != nil {
			logger.Errorf("存储管理器关闭失败: %v", err)
		} else {
			logger.Info("存储管理器已关闭")
		}
	}

	// Step 6: Wait for all goroutines to finish
	logger.Info("等待所有协程完成...")
	s.wg.Wait()
	logger.Info("所有协程已完成")

	return nil
}

// Shutdown performs graceful shutdown with context
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("开始优雅关闭...")

	// Create a channel for shutdown completion
	done := make(chan error, 1)

	go func() {
		done <- s.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Errorf("优雅关闭失败: %v", err)
			return err
		}
		logger.Info("优雅关闭完成")
		return nil
	case <-ctx.Done():
		logger.Warn("优雅关闭超时，强制退出")
		// Force stop
		s.mu.Lock()
		if s.httpServer != nil {
			utils.CloseQuietly(s.httpServer)
			s.httpServer = nil
		}
		s.mu.Unlock()
		return ctx.Err()
	}
}

// GetEngine returns the Gin engine (for testing)
func (s *Server) GetEngine() *gin.Engine {
	return s.engine
}

// GetWebSocketManager returns the WebSocket manager

// RegisterMasterHandler 注册 Master Handler（已废弃）
// Deprecated: 请使用 RegisterNodeAdapter 代替
func (s *Server) RegisterMasterHandler(handler interface{}) {
	logger.Warn("RegisterMasterHandler 已废弃，请使用 RegisterNodeAdapter 代替")
}

// RegisterNodeAdapter 注册 Node API 适配器
func (s *Server) RegisterNodeAdapter(nodeAdapter *api.NodeAdapter) {
	s.nodeAdapter = nodeAdapter

	// 设置事件回调，将客户端资源更新广播到 WebSocket
	nodeAdapter.SetEventCallback(func(eventType string, data interface{}) {
		if s.wsHub != nil {
			s.wsHub.Emit(eventType, data)
		}
	})

	// 注册 Node API 路由
	api := s.engine.Group("/api")
	nodeAdapter.RegisterRoutes(api)
	logger.Info("Node API 适配器路由已注册")
}

// RegisterLangChainHandler 注册 LangChainGo API 处理器
func (s *Server) RegisterLangChainHandler(handler *langchain.Handler) {
	s.langchainHandler = handler

	// 注册 LangChainGo API 路由
	api := s.engine.Group("/api")
	handler.RegisterRoutes(api)
	logger.Info("LangChainGo API 路由已注册: /api/langchain/*")

}

// Middleware

// corsMiddleware handles CORS
// loggerMiddleware logs requests
func (s *Server) handleServerInfo(c *gin.Context) {
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

// handleGetGPUs 返回系统可用的 GPU 列表
// 返回格式兼容 LlamacppServer 的设备列表格式
// 支持通过查询参数 llamacppPath 指定 llama.cpp 路径
func (s *Server) handleGetGPUs(c *gin.Context) {
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
		cmd := exec.Command("rocminfo")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			type GPUMemoryInfo struct {
				deviceIndex   int
				name          string
				marketingName string
				totalKB       int64 // 总内存（KB）
			}
			gpuMemories := []GPUMemoryInfo{}

			currentAgentType := ""
			var currentGPU GPUMemoryInfo
			inPoolInfo := false
			poolIndex := 0

			for _, line := range lines {
				line = strings.TrimSpace(line)

				// 检测新的 Agent
				if regexp.MustCompile(`^Agent\s+\d+\s*$`).MatchString(line) {
					// 保存前一个 GPU 的信息（如果是 GPU）
					if currentAgentType == "GPU" && currentGPU.totalKB > 0 {
						gpuMemories = append(gpuMemories, currentGPU)
					}
					// 重置状态
					currentAgentType = ""
					currentGPU = GPUMemoryInfo{}
					inPoolInfo = false
					poolIndex = 0
				}

				// 检测 Device Type（在 Marketing Name 之后，但需要先收集信息）
				if regexp.MustCompile(`^\s*Device Type:\s+GPU`).MatchString(line) {
					currentAgentType = "GPU"
				}

				// 解析 Agent 基本信息
				if matches := regexp.MustCompile(`^\s*Name:\s+(\S+)`).FindStringSubmatch(line); len(matches) > 1 {
					currentGPU.name = matches[1]
				}
				// Marketing Name
				if strings.Contains(line, "Marketing Name:") {
					marketingName := strings.ReplaceAll(line, "Marketing Name:", "")
					currentGPU.marketingName = strings.TrimSpace(marketingName)
				}

				// 检测 Pool Info 开始
				if regexp.MustCompile(`^\s*Pool Info:\s*$`).MatchString(line) {
					inPoolInfo = true
					poolIndex = 0
				}

				// 检测 Pool 开始
				if inPoolInfo && regexp.MustCompile(`^\s*Pool\s+(\d+)\s*$`).MatchString(line) {
					if matches := regexp.MustCompile(`Pool\s+(\d+)`).FindStringSubmatch(line); len(matches) > 1 {
						if idx, err := strconv.Atoi(matches[1]); err == nil {
							poolIndex = idx
						}
					}
				}

				// 解析 Pool Size（只在 Pool 1 解析）
				if inPoolInfo && poolIndex == 1 {
					if matches := regexp.MustCompile(`^\s*Size:\s+(\d+)\s*\(0x[0-9a-fA-F]+\)\s*KB`).FindStringSubmatch(line); len(matches) > 1 {
						if kb, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
							currentGPU.totalKB = kb
						}
					}
				}
			}

			// 保存最后一个 GPU 的信息
			if currentAgentType == "GPU" && currentGPU.totalKB > 0 {
				gpuMemories = append(gpuMemories, currentGPU)
			}

			// 生成 GPU 列表
			for i, gpuInfo := range gpuMemories {
				// 转换为 GB（保留两位小数）
				totalGB := float64(gpuInfo.totalKB) / 1024 / 1024
				totalMemory := fmt.Sprintf("%.2f GB", totalGB)

				deviceID := fmt.Sprintf("ROCm%d", i)
				gpuName := gpuInfo.marketingName
				if gpuName == "" {
					gpuName = "AMD GPU"
				}

				deviceString := fmt.Sprintf("%s: %s", deviceID, gpuName)
				deviceString += fmt.Sprintf(" (%s)", totalMemory)

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

// handleGetLlamacppBackends 返回可用的 llama.cpp 后端列表
func (s *Server) handleGetLlamacppBackends(c *gin.Context) {
	backends := []gin.H{}

	// 从配置中获取 llama.cpp 路径
	if s.config != nil && s.config.ServerCfg != nil {
		paths := s.config.ServerCfg.Llamacpp.Paths
		for _, p := range paths {
			// 检查路径是否存在
			available := false
			if fileInfo, err := os.Stat(p.Path); err == nil {
				// 检查是否是目录
				available = fileInfo.IsDir()
			}

			backends = append(backends, gin.H{
				"path":        p.Path,
				"name":        p.Name,
				"description": p.Description,
				"available":   available,
			})
		}
	}

	api.Success(c, gin.H{
		"backends": backends,
		"count":    len(backends),
	})
}

// handleEstimateVRAM 估算模型显存需求
func (s *Server) handleEstimateVRAM(c *gin.Context) {
	var req struct {
		ModelID        string `json:"modelId"`
		LlamaBinPath   string `json:"llamaBinPath"`
		CtxSize        int    `json:"ctxSize"`
		BatchSize      int    `json:"batchSize"`
		UBatchSize     int    `json:"uBatchSize"`
		Parallel       int    `json:"parallel"`
		FlashAttention bool   `json:"flashAttention"`
		KVUnified      bool   `json:"kvUnified"`
		CacheTypeK     string `json:"cacheTypeK"`
		CacheTypeV     string `json:"cacheTypeV"`
		ExtraParams    string `json:"extraParams"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求: " + err.Error()})
		return
	}

	// 验证必需参数
	if req.ModelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 modelId 参数"})
		return
	}
	if req.LlamaBinPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 llamaBinPath 参数"})
		return
	}

	// 从模型管理器获取模型信息
	model, exists := s.modelMgr.GetModel(req.ModelID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "模型不存在: " + req.ModelID})
		return
	}

	// 构建模型文件路径
	var modelPath string
	if model.ShardCount > 0 && len(model.ShardFiles) > 0 {
		// 分卷模型，使用主模型文件
		modelPath = model.ShardFiles[0]
	} else {
		modelPath = model.Path
	}

	// 构建 llama-fit-params 命令
	args := []string{
		"--model", modelPath,
	}

	// 添加支持的参数
	if req.CtxSize > 0 {
		args = append(args, "--ctx-size", fmt.Sprintf("%d", req.CtxSize))
	}
	if req.BatchSize > 0 {
		args = append(args, "--batch-size", fmt.Sprintf("%d", req.BatchSize))
	}
	if req.UBatchSize > 0 {
		args = append(args, "--ubatch-size", fmt.Sprintf("%d", req.UBatchSize))
	}
	if req.Parallel > 0 {
		args = append(args, "--parallel", fmt.Sprintf("%d", req.Parallel))
	}
	if req.FlashAttention {
		args = append(args, "--flash-attn", "1")
	}
	if req.KVUnified {
		args = append(args, "--kv-unified", "1")
	}
	if req.CacheTypeK != "" {
		args = append(args, "--cache-type-k", req.CacheTypeK)
	}
	if req.CacheTypeV != "" {
		args = append(args, "--cache-type-v", req.CacheTypeV)
	}

	// 构建完整命令
	cmdPath := filepath.Join(req.LlamaBinPath, "llama-fit-params")
	cmd := exec.Command(cmdPath, args...)

	// 执行命令（设置30秒超时）
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// 检查是否有部分输出
		logger.Error("llama-fit-params 执行失败", "error", err.Error(), "output", outputStr)

		// 尝试从错误输出中提取错误信息
		errorMsg := "估算失败"
		if strings.Contains(outputStr, "llama_model_load") || strings.Contains(outputStr, "failed to load model") {
			errorMsg = "模型加载失败，请检查模型文件是否正确"
		} else if strings.Contains(outputStr, "llama_params_fit") {
			errorMsg = "参数拟合失败，请检查参数是否有效"
		}

		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   errorMsg,
			"details": outputStr,
		})
		return
	}

	// 解析输出，提取显存估算值
	vramMB := 0

	// 匹配格式: "llama_params_fit_impl: projected to use XXX MiB of device memory"
	vramRe := regexp.MustCompile(`llama_params_fit_impl: projected to use (\d+) MiB`)
	if matches := vramRe.FindStringSubmatch(outputStr); len(matches) > 1 {
		vramMB, _ = strconv.Atoi(matches[1])
	}

	// 构建响应
	result := gin.H{
		"success": vramMB > 0,
	}

	if vramMB > 0 {
		result["vram"] = fmt.Sprintf("%d", vramMB)
		result["vramMB"] = vramMB
		result["vramGB"] = fmt.Sprintf("%.2f", float64(vramMB)/1024)
	} else {
		// 如果没有找到显存值，检查是否有错误信息
		errorRe := regexp.MustCompile(`llama_init_from_model.*`)
		if errorMatch := errorRe.FindString(outputStr); errorMatch != "" {
			result["error"] = strings.TrimSpace(errorMatch)
		} else {
			result["error"] = "无法解析显存估算结果"
		}
		result["details"] = outputStr
	}

	if vramMB > 0 {
		api.Success(c, result)
	} else {
		errorMsg := "无法解析显存估算结果"
		if errStr, ok := result["error"].(string); ok {
			errorMsg = errStr
		}
		api.ErrorWithDetails(c, types.ErrInternalError, "无法解析显存估算结果", errorMsg)
	}
}

// handleGetConfig 返回当前配置（不包含敏感信息）
func (s *Server) handleGetConfig(c *gin.Context) {
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

// handleUpdateConfig 更新配置
func (s *Server) handleUpdateConfig(c *gin.Context) {
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
					logger.Warn("模型扫描失败", "error", err)
				}
			}()
		}
	}

	api.Success(c, gin.H{
		"message":          "配置更新成功",
		"restart_required": restartRequired,
	})
}

func (s *Server) handleListModels(c *gin.Context) {
	models := s.modelMgr.ListModels()
	statuses := s.modelMgr.ListStatus()

	fmt.Printf("[DEBUG] handleListModels: 总共 %d 个模型\n", len(models))

	var dtos []ModelDTO
	for _, m := range models {
		// 调试日志：检查分卷信息
		if m.ShardCount > 0 {
			fmt.Printf("[DEBUG] 分卷模型: %s, ShardCount=%d, TotalSize=%d, Files=%d\n",
				m.Name, m.ShardCount, m.TotalSize, len(m.ShardFiles))
		}

		dto := ModelDTO{
			ID:          m.ID,
			Name:        m.Name,
			DisplayName: m.DisplayName,
			Alias:       m.Alias,
			Path:        m.Path,
			PathPrefix:  m.PathPrefix,
			Size:        m.Size,
			Favourite:   m.Favourite,
			Status:      "stopped",
			IsLoaded:    false,
		}

		// 添加分卷信息
		if m.ShardCount > 0 {
			dto.ShardCount = m.ShardCount
			dto.TotalSize = m.TotalSize
			dto.ShardFiles = m.ShardFiles
		}

		// 添加 mmproj 路径
		if m.MmprojPath != "" {
			dto.MmprojPath = m.MmprojPath
		}

		// 添加扫描时间（处理零值情况）
		if !m.ScannedAt.IsZero() {
			dto.ScannedAt = m.ScannedAt.Format(time.RFC3339)
		}

		// Convert metadata - 包含所有 gguf-parser-go 提供的字段
		if m.Metadata != nil {
			metadata := map[string]interface{}{
				"name":            m.Metadata.Name,
				"architecture":    m.Metadata.Architecture,
				"quantization":    m.Metadata.Quantization,
				"contextLength":   m.Metadata.ContextLength,
				"embeddingLength": m.Metadata.EmbeddingLength,
				"layerCount":      m.Metadata.BlockSize,
				"headCount":       m.Metadata.HeadCount,
				// 新增的 gguf-parser-go 字段
				"type":                nonEmptyString(m.Metadata.Type),
				"author":              nonEmptyString(m.Metadata.Author),
				"url":                 nonEmptyString(m.Metadata.URL),
				"description":         nonEmptyString(m.Metadata.Description),
				"license":             nonEmptyString(m.Metadata.License),
				"fileType":            m.Metadata.FileType,
				"fileTypeDescriptor":  nonEmptyString(m.Metadata.FileTypeDescriptor),
				"quantizationVersion": m.Metadata.QuantizationVersion,
				"parameters":          m.Metadata.Parameters,
				"bitsPerWeight":       m.Metadata.BitsPerWeight,
				"alignment":           m.Metadata.Alignment,
				"fileSize":            m.Metadata.FileSize,
				"modelSize":           m.Metadata.ModelSize,
			}
			dto.Metadata = metadata
		}

		// Add status info
		if status, ok := statuses[m.ID]; ok {
			dto.Status = status.State.String()
			dto.IsLoaded = status.State == model.StateLoaded
		}

		dtos = append(dtos, dto)
	}

	api.Success(c, gin.H{"models": dtos, "total": len(dtos)})
}

// handleListLoadedModels 返回已加载的模型列表
func (s *Server) handleListLoadedModels(c *gin.Context) {
	models := s.modelMgr.ListModels()
	statuses := s.modelMgr.ListStatus()

	var loadedModels []ModelDTO
	for _, m := range models {
		// 只返回已加载的模型
		if status, ok := statuses[m.ID]; ok && status.State == model.StateLoaded {
			dto := ModelDTO{
				ID:          m.ID,
				Name:        m.Name,
				DisplayName: m.DisplayName,
				Alias:       m.Alias,
				Path:        m.Path,
				PathPrefix:  m.PathPrefix,
				Size:        m.Size,
				Favourite:   m.Favourite,
				Status:      "running",
				IsLoaded:    true,
			}

			// 添加分卷信息
			if m.ShardCount > 0 {
				dto.ShardCount = m.ShardCount
				dto.TotalSize = m.TotalSize
				dto.ShardFiles = m.ShardFiles
			}

			// 添加 mmproj 路径
			if m.MmprojPath != "" {
				dto.MmprojPath = m.MmprojPath
			}

			// 添加扫描时间
			if !m.ScannedAt.IsZero() {
				dto.ScannedAt = m.ScannedAt.Format(time.RFC3339)
			}

			// 添加元数据
			if m.Metadata != nil {
				metadata := map[string]interface{}{
					"name":            m.Metadata.Name,
					"architecture":    m.Metadata.Architecture,
					"quantization":    m.Metadata.Quantization,
					"contextLength":   m.Metadata.ContextLength,
					"embeddingLength": m.Metadata.EmbeddingLength,
				}
				dto.Metadata = metadata
			}

			loadedModels = append(loadedModels, dto)
		}
	}

	api.Success(c, gin.H{"success": true, "models": loadedModels, "total": len(loadedModels)})
}

func (s *Server) handleGetModel(c *gin.Context) {
	id := c.Param("id")
	m, exists := s.modelMgr.GetModel(id)
	if !exists {
		api.NotFound(c, "模型")
		return
	}

	// 转换为 ModelDTO 格式（与 handleListModels 保持一致）
	statuses := s.modelMgr.ListStatus()
	dto := ModelDTO{
		ID:          m.ID,
		Name:        m.Name,
		DisplayName: m.DisplayName,
		Alias:       m.Alias,
		Path:        m.Path,
		PathPrefix:  m.PathPrefix,
		Size:        m.Size,
		Favourite:   m.Favourite,
		Status:      "stopped",
		IsLoaded:    false,
	}

	// 添加分卷信息
	if m.ShardCount > 0 {
		dto.ShardCount = m.ShardCount
		dto.TotalSize = m.TotalSize
		dto.ShardFiles = m.ShardFiles
	}

	// 添加 mmproj 路径
	if m.MmprojPath != "" {
		dto.MmprojPath = m.MmprojPath
	}

	// 添加扫描时间
	if !m.ScannedAt.IsZero() {
		dto.ScannedAt = m.ScannedAt.Format(time.RFC3339)
	}

	// 转换元数据
	if m.Metadata != nil {
		metadata := map[string]interface{}{
			"name":                m.Metadata.Name,
			"architecture":        m.Metadata.Architecture,
			"quantization":        m.Metadata.Quantization,
			"contextLength":       m.Metadata.ContextLength,
			"embeddingLength":     m.Metadata.EmbeddingLength,
			"layerCount":          m.Metadata.BlockSize,
			"headCount":           m.Metadata.HeadCount,
			"type":                nonEmptyString(m.Metadata.Type),
			"author":              nonEmptyString(m.Metadata.Author),
			"url":                 nonEmptyString(m.Metadata.URL),
			"description":         nonEmptyString(m.Metadata.Description),
			"license":             nonEmptyString(m.Metadata.License),
			"fileType":            m.Metadata.FileType,
			"fileTypeDescriptor":  nonEmptyString(m.Metadata.FileTypeDescriptor),
			"quantizationVersion": m.Metadata.QuantizationVersion,
			"parameters":          m.Metadata.Parameters,
			"bitsPerWeight":       m.Metadata.BitsPerWeight,
			"alignment":           m.Metadata.Alignment,
			"fileSize":            m.Metadata.FileSize,
			"modelSize":           m.Metadata.ModelSize,
		}
		dto.Metadata = metadata
	}

	// 添加状态信息
	if status, ok := statuses[m.ID]; ok {
		dto.Status = status.State.String()
		dto.IsLoaded = status.State == model.StateLoaded
	}

	api.Success(c, gin.H{"model": dto})
}

func (s *Server) handleLoadModel(c *gin.Context) {
	id := c.Param("id")

	// 检查是否异步加载（查询参数 async=true）
	asyncMode := c.DefaultQuery("async", "true") == "true"

	// 定义包含 capabilities 的请求结构
	var req struct {
		model.LoadRequest
		Capabilities *storage.Capabilities `json:"capabilities"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// 使用默认值
		req.LoadRequest = model.LoadRequest{
			ModelID: id,
			CtxSize: 4096,
		}
	} else {
		req.LoadRequest.ModelID = id
	}

	// 如果请求中包含 capabilities，保存到数据库
	if req.Capabilities != nil {
		// 验证并应用约束规则
		if err := req.Capabilities.Validate(); err != nil {
			api.BadRequest(c, err.Error())
			return
		}
		req.Capabilities.ApplyConstraints()

		// 保存到数据库
		ctx := context.Background()
		existingMeta, err := s.storageMgr.GetStore().GetModelMetadata(ctx, req.LoadRequest.ModelID)
		if err == nil && existingMeta != nil {
			existingMeta.Capabilities = req.Capabilities
			if err := s.storageMgr.GetStore().SaveModelMetadata(ctx, existingMeta); err != nil {
				logger.Warn("保存模型能力失败", "modelId", req.LoadRequest.ModelID, "error", err)
			}
		} else {
			if err := s.storageMgr.GetStore().SaveModelMetadata(ctx, &storage.ModelMetadata{
				ModelID:      req.LoadRequest.ModelID,
				Capabilities: req.Capabilities,
			}); err != nil {
				logger.Warn("保存模型能力失败", "modelId", req.LoadRequest.ModelID, "error", err)
			}
		}

		logger.Info("模型能力已更新", "modelId", req.LoadRequest.ModelID,
			"thinking", req.Capabilities.Thinking,
			"tools", req.Capabilities.Tools,
			"rerank", req.Capabilities.Rerank,
			"embedding", req.Capabilities.Embedding)
	}

	var result *model.LoadResult
	var err error

	// 如果指定了节点 ID，使用 Scheduler 分发到指定节点
	if req.NodeID != "" && s.nodeAdapter != nil {
		scheduler := s.nodeAdapter.GetScheduler()
		if scheduler == nil {
			api.ErrorWithDetails(c, types.ErrInternalError, "调度器未初始化", "无法分发到指定节点")
			return
		}

		// 构建任务负载
		payload := map[string]interface{}{
			"modelId":   req.ModelID,
			"ctxSize":   req.CtxSize,
			"batchSize": req.BatchSize,
			"threads":   req.Threads,
			"gpuLayers": req.GPULayers,
		}

		// 提交任务到指定节点
		task, err := scheduler.SubmitTask("load_model", payload, req.NodeID)
		if err != nil {
			api.ErrorWithDetails(c, types.ErrInternalError, "提交模型加载任务失败", err.Error())
			return
		}

		logger.Info("模型加载任务已提交到指定节点", "modelId", req.ModelID, "nodeId", req.NodeID, "taskId", task.ID)

		// 返回任务信息，前端可以轮询任务状态
		api.Success(c, gin.H{
			"message":  "模型加载任务已提交",
			"model_id": req.ModelID,
			"node_id":  req.NodeID,
			"task_id":  task.ID,
			"async":    true,
			"status":   "pending",
		})
		return
	}

	if asyncMode {
		// 异步加载
		result, err = s.modelMgr.LoadAsync(&req.LoadRequest)
		if err != nil {
			api.ErrorWithDetails(c, types.ErrInternalError, "加载模型失败", err.Error())
			return
		}

		// 立即返回异步响应
		if result.Loading {
			api.Success(c, gin.H{
				"message":  "模型正在加载中",
				"model_id": result.ModelID,
				"async":    true,
				"status":   "loading",
			})
			return
		}

		if result.AlreadyLoaded {
			// 模型已加载（运行中）
			api.Success(c, gin.H{
				"message":  "模型已在运行中",
				"model_id": result.ModelID,
				"port":     result.Port,
				"status":   "running",
			})
			return
		}
	} else {
		// 同步加载
		result, err = s.modelMgr.Load(&req.LoadRequest)
		if err != nil {
			api.ErrorWithDetails(c, types.ErrInternalError, "加载模型失败", err.Error())
			return
		}
	}

	if !result.Success {
		api.ErrorWithDetails(c, types.ErrInternalError, "加载模型失败", result.Error.Error())
		return
	}

	api.Success(c, gin.H{
		"message":  "模型加载成功",
		"model_id": result.ModelID,
		"port":     result.Port,
		"ctx_size": result.CtxSize,
	})
}

func (s *Server) handleUnloadModel(c *gin.Context) {
	id := c.Param("id")
	if err := s.modelMgr.Unload(id); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "卸载模型失败", err.Error())
		return
	}
	api.SuccessWithMessage(c, "模型卸载成功")
}

func (s *Server) handleSetAlias(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Alias string `json:"alias"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "无效的请求")
		return
	}

	if err := s.modelMgr.SetAlias(id, req.Alias); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "设置别名失败", err.Error())
		return
	}

	api.SuccessWithMessage(c, "别名设置成功")
}

func (s *Server) handleSetFavourite(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Favourite bool `json:"favourite"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "无效的请求")
		return
	}

	if err := s.modelMgr.SetFavourite(id, req.Favourite); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "设置收藏失败", err.Error())
		return
	}

	api.SuccessWithMessage(c, "收藏设置成功")
}

// handleGetModelCapabilities 获取模型能力配置
func (s *Server) handleGetModelCapabilities(c *gin.Context) {
	modelID := c.Query("modelId")
	if modelID == "" {
		api.BadRequest(c, "缺少 modelId 参数")
		return
	}

	// 从数据库获取模型元数据
	ctx := context.Background()
	metadata, err := s.storageMgr.GetStore().GetModelMetadata(ctx, modelID)
	if err != nil {
		// 如果没有找到，返回默认值（全部为 false）
		api.Success(c, gin.H{
			"modelId":      modelID,
			"capabilities": &storage.Capabilities{},
		})
		return
	}

	caps := metadata.Capabilities
	if caps == nil {
		caps = &storage.Capabilities{}
	}

	api.Success(c, gin.H{
		"modelId":      modelID,
		"capabilities": caps,
	})
}

// handleSetModelCapabilities 设置模型能力配置
func (s *Server) handleSetModelCapabilities(c *gin.Context) {
	var req struct {
		ModelID      string                `json:"modelId"`
		Capabilities *storage.Capabilities `json:"capabilities"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "无效的请求: "+err.Error())
		return
	}

	if req.ModelID == "" {
		api.BadRequest(c, "缺少 modelId")
		return
	}

	if req.Capabilities == nil {
		api.BadRequest(c, "缺少 capabilities")
		return
	}

	// 验证并应用约束规则
	if err := req.Capabilities.Validate(); err != nil {
		api.BadRequest(c, err.Error())
		return
	}
	req.Capabilities.ApplyConstraints()

	// 保存到数据库
	ctx := context.Background()
	existingMeta, err := s.storageMgr.GetStore().GetModelMetadata(ctx, req.ModelID)
	if err == nil && existingMeta != nil {
		// 更新现有元数据
		existingMeta.Capabilities = req.Capabilities
		if err := s.storageMgr.GetStore().SaveModelMetadata(ctx, existingMeta); err != nil {
			logger.Warn("保存模型能力失败", "modelId", req.ModelID, "error", err)
		}
	} else {
		// 创建新的元数据条目
		if err := s.storageMgr.GetStore().SaveModelMetadata(ctx, &storage.ModelMetadata{
			ModelID:      req.ModelID,
			Capabilities: req.Capabilities,
		}); err != nil {
			logger.Warn("保存模型能力失败", "modelId", req.ModelID, "error", err)
		}
	}

	logger.Info("模型能力已更新", "modelId", req.ModelID,
		"thinking", req.Capabilities.Thinking,
		"tools", req.Capabilities.Tools,
		"rerank", req.Capabilities.Rerank,
		"embedding", req.Capabilities.Embedding)

	api.Success(c, gin.H{
		"modelId":      req.ModelID,
		"capabilities": req.Capabilities,
	})
}

// handleAutoDetectCapabilities 自动检测模型能力
func (s *Server) handleAutoDetectCapabilities(c *gin.Context) {
	modelID := c.Query("modelId")
	if modelID == "" {
		api.BadRequest(c, "缺少 modelId 参数")
		return
	}

	caps, err := s.modelMgr.AutoDetectCapabilities(modelID)
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}

	api.Success(c, gin.H{
		"modelId":      modelID,
		"capabilities": caps,
	})
}

func (s *Server) handleScanModels(c *gin.Context) {
	result, err := s.modelMgr.Scan(c.Request.Context())
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "扫描失败", err.Error())
		return
	}
	api.Success(c, gin.H{
		"message":      "扫描完成",
		"models_found": len(result.Models),
		"errors":       len(result.Errors),
		"duration_ms":  result.Duration.Milliseconds(),
		"models":       result.Models,
		"scan_errors":  result.Errors,
	})
}

func (s *Server) handleGetScanStatus(c *gin.Context) {
	status := s.modelMgr.GetScanStatus()
	api.Success(c, gin.H{
		"scanning":     status.Scanning,
		"progress":     status.Progress,
		"current_path": status.CurrentPath,
		"started_at":   status.StartedAt,
		"errors":       status.Errors,
	})
}

func (s *Server) handleListDownloads(c *gin.Context) {
	tasks := s.downloadMgr.ListTasks()

	downloads := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		downloads = append(downloads, mapDownloadTask(t))
	}

	api.Success(c, gin.H{
		"downloads": downloads,
		"total":     len(downloads),
	})
}

// mapDownloadTask transforms download.Task to the frontend DownloadTask format
func mapDownloadTask(t *download.Task) gin.H {
	progress := float64(0)
	if t.TotalBytes > 0 {
		progress = float64(t.DownloadedBytes) / float64(t.TotalBytes)
	}

	errStr := ""
	if t.Error != nil {
		errStr = t.Error.Error()
	}

	var completedAt string
	if !t.FinishedAt.IsZero() {
		completedAt = t.FinishedAt.Format(time.RFC3339)
	}

	result := gin.H{
		"id":              t.ID,
		"source":          t.SourceType,
		"repoId":          t.RepoID,
		"fileName":        t.FileName,
		"path":            t.Path,
		"state":           t.State.String(),
		"downloadedBytes": t.DownloadedBytes,
		"totalBytes":      t.TotalBytes,
		"partsCompleted":  t.PartsCompleted,
		"partsTotal":      t.PartsTotal,
		"progress":        progress,
		"speed":           t.Speed,
		"eta":             t.ETA,
		"error":           errStr,
		"createdAt":       t.CreatedAt.Format(time.RFC3339),
	}

	if completedAt != "" {
		result["completedAt"] = completedAt
	}

	return result
}

func (s *Server) handleCreateDownload(c *gin.Context) {
	// 支持两种请求格式:
	// 1. 新格式: { source, repoId, fileName, path } - 用于从模型仓库下载
	// 2. 旧格式: { url, target_path } - 直接URL下载(向后兼容)

	var req struct {
		Source   modelrepoclient.Source `json:"source"`
		RepoID   string                 `json:"repoId"`
		FileName string                 `json:"fileName"`
		Path     string                 `json:"path"`

		// 旧格式参数(向后兼容)
		URL        string `json:"url"`
		TargetPath string `json:"target_path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "无效的请求格式: "+err.Error())
		return
	}

	var downloadURL string
	var downloadDir string
	var fileName string
	var source string
	var repoId string

	// 使用新格式(source + repoId)
	if req.Source != "" && req.RepoID != "" {
		// 生成下载 URL
		url, err := s.repoClient.GenerateDownloadURL(req.Source, req.RepoID, req.FileName)
		if err != nil {
			api.ErrorWithDetails(c, types.ErrInvalidRequest, "生成下载URL失败", err.Error())
			return
		}
		downloadURL = url
		source = string(req.Source)
		repoId = req.RepoID
		fileName = req.FileName

		// Determine directory
		if req.Path != "" {
			downloadDir = req.Path
		} else {
			// Default path logic
			cfg := s.config.ConfigMgr.Get()
			if len(cfg.Model.Paths) > 0 {
				downloadDir = cfg.Model.Paths[0]
			} else {
				downloadDir = "./models"
			}
		}
	} else if req.URL != "" {
		// 使用旧格式(直接URL)
		downloadURL = req.URL
		source = "url"
		downloadDir = filepath.Dir(req.TargetPath)
		fileName = filepath.Base(req.TargetPath)
	} else {
		api.BadRequest(c, "缺少必要参数: 请提供 source/repoId 或 url")
		return
	}

	taskId, err := s.downloadMgr.CreateTask(downloadURL, downloadDir, fileName, source, repoId)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "创建下载失败", err.Error())
		return
	}

	task, exists := s.downloadMgr.GetTask(taskId)
	if !exists {
		api.ErrorWithDetails(c, types.ErrInternalError, "创建下载失败", "无法获取任务信息")
		return
	}

	// Get request ID from context
	requestID := "unknown"
	if id := c.GetString("requestId"); id != "" {
		requestID = id
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponse(mapDownloadTask(task), requestID))
}

func (s *Server) handleGetDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "下载ID不能为空")
		return
	}

	task, exists := s.downloadMgr.GetTask(id)
	if !exists {
		api.NotFound(c, "下载任务")
		return
	}

	api.Success(c, mapDownloadTask(task))
}

func (s *Server) handlePauseDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "下载ID不能为空")
		return
	}

	if err := s.downloadMgr.Pause(id); err != nil {
		api.NotFound(c, err.Error())
		return
	}

	api.SuccessWithMessage(c, "下载已暂停")
}

func (s *Server) handleResumeDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "下载ID不能为空")
		return
	}

	if err := s.downloadMgr.Resume(id); err != nil {
		api.NotFound(c, err.Error())
		return
	}

	api.SuccessWithMessage(c, "下载已恢复")
}

func (s *Server) handleDeleteDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "下载ID不能为空")
		return
	}

	if err := s.downloadMgr.Delete(id); err != nil {
		api.NotFound(c, err.Error())
		return
	}

	api.SuccessWithMessage(c, "下载任务已删除")
}

// handleListModelFiles handles requests to list model files from a repository
func (s *Server) handleListModelFiles(c *gin.Context) {
	// 使用查询参数而不是路径参数，以支持 repoId 中包含斜杠
	source := c.Query("source")
	repoID := c.Query("repoId")

	if source == "" || repoID == "" {
		api.BadRequest(c, "缺少必要参数: 需要 source 和 repoId 查询参数")
		return
	}

	// 目前只支持 HuggingFace
	if source != "huggingface" {
		api.BadRequest(c, "目前只支持 HuggingFace 源")
		return
	}

	files, err := s.repoClient.ListGGUFFiles(repoID)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "获取文件列表失败", err.Error())
		return
	}

	api.Success(c, files)
}

// handleSearchModels handles requests to search for models on HuggingFace
func (s *Server) handleSearchModels(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		api.BadRequest(c, "缺少必要参数: 需要 q 查询参数")
		return
	}

	// Parse limit parameter (default 20)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Parse format filter parameter (e.g., "gguf", "safetensors", "onnx")
	format := c.Query("format")

	result, err := s.repoClient.SearchHuggingFaceModels(query, limit, format)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "搜索模型失败", err.Error())
		return
	}

	api.Success(c, result)
}

// handleGetModelRepoConfig returns the current model repository configuration
func (s *Server) handleGetModelRepoConfig(c *gin.Context) {
	cfg := s.config.ConfigMgr.Get()
	api.Success(c, gin.H{
		"endpoint": cfg.ModelRepo.Endpoint,
		"token":    maskToken(cfg.ModelRepo.Token),
		"timeout":  cfg.ModelRepo.Timeout,
	})
}

// maskToken masks the token for security
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

// handleUpdateModelRepoConfig updates the model repository configuration
func (s *Server) handleUpdateModelRepoConfig(c *gin.Context) {
	var req struct {
		Endpoint string `json:"endpoint"`
		Token    string `json:"token"`
		Timeout  int    `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "无效的请求数据")
		return
	}

	cfg := s.config.ConfigMgr.Get()

	// Update endpoint if provided
	if req.Endpoint != "" {
		cfg.ModelRepo.Endpoint = req.Endpoint
	}

	// Update token if provided (allow empty string to clear token)
	if req.Token != "" {
		cfg.ModelRepo.Token = req.Token
	}

	// Update timeout if provided
	if req.Timeout > 0 {
		cfg.ModelRepo.Timeout = req.Timeout
	}

	// Save config
	if err := s.config.ConfigMgr.Save(cfg); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "保存配置失败", err.Error())
		return
	}

	// Update the repo client with new settings
	timeout := time.Duration(cfg.ModelRepo.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	s.repoClient = modelrepoclient.NewClientWithConfig(cfg.ModelRepo.Endpoint, cfg.ModelRepo.Token, timeout)

	api.Success(c, gin.H{
		"endpoint": cfg.ModelRepo.Endpoint,
		"token":    maskToken(cfg.ModelRepo.Token),
		"timeout":  cfg.ModelRepo.Timeout,
	})
}

// handleGetAvailableEndpoints returns available HuggingFace endpoints
func (s *Server) handleGetAvailableEndpoints(c *gin.Context) {
	endpoints := modelrepoclient.GetAvailableEndpoints()
	api.Success(c, endpoints)
}

func (s *Server) handleListProcesses(c *gin.Context) {
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

func (s *Server) handleGetProcess(c *gin.Context) {
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

func (s *Server) handleStopProcess(c *gin.Context) {
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

// handleLogStream streams log entries using Server-Sent Events
func (s *Server) handleLogStream(c *gin.Context) {
	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	// Get parameters
	fromBeginning := c.DefaultQuery("fromBeginning", "false") == "true"
	limit := 100
	if l := c.Query("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			limit = 100
		}
	}

	// Get log stream
	logStream := logger.GetLogStream()

	// Flush headers
	c.Writer.Flush()

	// Create channel for log entries
	logCh := logStream.Subscribe()
	defer logStream.Unsubscribe(logCh)

	// Send existing entries if requested
	if fromBeginning {
		// Try to read from log file first
		logEntries := []logger.StreamLogEntry{}

		// Get log file path from configuration
		if s.config != nil && s.config.ServerCfg != nil {
			logDir := s.config.ServerCfg.Log.Directory
			role := s.config.ServerCfg.Node.Role

			// Get latest log file path
			logPath, err := logger.GetLatestLogFile(logDir, role)
			if err == nil {
				// Read log file with empty filter to get all entries
				parsedEntries, err := logger.ReadLogFile(logPath, logger.LogFileFilter{})
				if err == nil && len(parsedEntries) > 0 {
					// Convert ParsedLogEntry to StreamLogEntry
					for _, entry := range parsedEntries {
						logEntries = append(logEntries, logger.StreamLogEntry{
							Timestamp: entry.Timestamp,
							Level:     entry.Level,
							Message:   entry.Message,
							Fields:    entry.Fields,
						})
					}
				}
			}
		}

		// If no file entries, use memory cache as fallback
		if len(logEntries) == 0 {
			logEntries = logStream.GetEntries(limit)
		}

		// Send historical logs
		if len(logEntries) > 0 {
			// Batch send historical logs to reduce network I/O
			var buf strings.Builder
			for _, entry := range logEntries {
				data := fmt.Sprintf("data: {\"timestamp\":\"%s\",\"level\":\"%s\",\"message\":\"%s\"}\n\n",
					entry.Timestamp.Format(time.RFC3339),
					entry.Level,
					entry.Message)
				buf.WriteString(data)
			}
			utils.WriteStringQuietly(c.Writer, buf.String())
			c.Writer.Flush()
		}
	}

	// Keep connection alive and send new entries
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-logCh:
			if !ok {
				return
			}
			s.sendSSE(c, &entry)
			c.Writer.Flush()
		case <-ticker.C:
			// Send keepalive comment
			c.SSEvent("keepalive", "")
			c.Writer.Flush()
		}
	}
}

// sendSSE sends a log entry as Server-Sent Event
func (s *Server) sendSSE(c *gin.Context, entry *logger.StreamLogEntry) {
	data := fmt.Sprintf("data: {\"timestamp\":\"%s\",\"level\":\"%s\",\"message\":\"%s\"}\n\n",
		entry.Timestamp.Format(time.RFC3339),
		entry.Level,
		entry.Message)
	utils.WriteStringQuietly(c.Writer, data)
}

// handleLogEntries returns recent log entries
func (s *Server) handleLogEntries(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil {
			limit = 100
		}
	}

	logStream := logger.GetLogStream()
	entries := logStream.GetEntries(limit)

	api.Success(c, gin.H{
		"entries": entries,
		"count":   len(entries),
	})
}

// handleLogFiles lists all available log files
func (s *Server) handleLogFiles(c *gin.Context) {
	cfg := s.config.ConfigMgr.Get()
	logDir := cfg.Log.Directory
	role := s.config.ServerCfg.Node.Role

	files, err := logger.ListLogFiles(logDir, role)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "获取日志文件列表失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"files": files,
		"count": len(files),
	})
}

// handleLogFileContent returns the content of a specific log file
func (s *Server) handleLogFileContent(c *gin.Context) {
	filename := c.Param("filename")

	// Security: ensure filename is safe
	if !isSafeFilename(filename) {
		api.BadRequest(c, "无效的文件名")
		return
	}

	cfg := s.config.ConfigMgr.Get()
	logDir := cfg.Log.Directory
	logPath := filepath.Join(logDir, filename)

	// Parse filters
	filter := logger.LogFileFilter{
		Level:  c.Query("level"),
		Search: c.Query("search"),
	}

	if offset := c.Query("offset"); offset != "" {
		if _, err := fmt.Sscanf(offset, "%d", &filter.Offset); err != nil {
			filter.Offset = 0
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if _, err := fmt.Sscanf(limit, "%d", &filter.Limit); err != nil {
			filter.Limit = 100
		}
	}

	// Read log file
	entries, err := logger.ReadLogFile(logPath, filter)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "读取日志文件失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"entries": entries,
		"count":   len(entries),
	})
}

// handleLogFileStats returns statistics about a log file
func (s *Server) handleLogFileStats(c *gin.Context) {
	filename := c.Param("filename")

	// Security: ensure filename is safe
	if !isSafeFilename(filename) {
		api.BadRequest(c, "无效的文件名")
		return
	}

	cfg := s.config.ConfigMgr.Get()
	logDir := cfg.Log.Directory
	logPath := filepath.Join(logDir, filename)

	stats, err := logger.GetLogFileStats(logPath)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "获取日志统计失败", err.Error())
		return
	}

	api.Success(c, stats)
}

// handleDeleteLogFile deletes a specific log file
func (s *Server) handleDeleteLogFile(c *gin.Context) {
	filename := c.Param("filename")

	// Security: ensure filename is safe
	if !isSafeFilename(filename) {
		api.BadRequest(c, "无效的文件名")
		return
	}

	// Prevent deleting the current day's log file
	cfg := s.config.ConfigMgr.Get()
	role := s.config.ServerCfg.Node.Role
	currentDate := time.Now().Format("2006-01-02")
	currentLogName := fmt.Sprintf("shepherd-%s-%s.log", role, currentDate)

	if filename == currentLogName {
		api.Forbidden(c, "不能删除当前日志文件")
		return
	}

	logDir := cfg.Log.Directory
	logPath := filepath.Join(logDir, filename)

	// Delete file
	if err := os.Remove(logPath); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "删除日志文件失败", err.Error())
		return
	}

	logger.Info("日志文件已删除", "filename", filename)

	api.Success(c, gin.H{
		"message": "日志文件已删除",
	})
}

// handleGetModelLoadConfig 获取模型加载配置
func (s *Server) handleGetModelLoadConfig(c *gin.Context) {
	modelID := c.Param("id")

	// 获取节点ID
	nodeID := "local"
	if s.nodeAdapter != nil {
		nodeID = s.nodeAdapter.GetNodeID()
	}

	// 从存储中获取配置
	store := s.storageMgr.GetStore()
	ctx := c.Request.Context()
	config, err := store.GetModelLoadConfig(ctx, nodeID, modelID)
	if err != nil {
		if err == storage.ErrModelLoadConfigNotFound {
			// 返回空配置，表示使用默认配置
			api.Success(c, gin.H{
				"exists": false,
				"config": nil,
			})
			return
		}
		api.ErrorWithDetails(c, types.ErrInternalError, "获取模型加载配置失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"exists": true,
		"config": config,
	})
}

// handleSaveModelLoadConfig 保存模型加载配置
func (s *Server) handleSaveModelLoadConfig(c *gin.Context) {
	modelID := c.Param("id")

	// 获取节点ID
	nodeID := "local"
	if s.nodeAdapter != nil {
		nodeID = s.nodeAdapter.GetNodeID()
	}

	// 解析请求参数
	var req struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求参数", err.Error())
		return
	}

	// 获取模型名称（从模型管理器）
	modelName := modelID
	if model, ok := s.modelMgr.GetModel(modelID); ok {
		modelName = model.Name
	}

	// 创建配置对象
	config := &storage.ModelLoadConfig{
		NodeID:    nodeID,
		ModelID:   modelID,
		ModelName: modelName,
		Config:    req.Config,
	}

	// 保存到存储
	store := s.storageMgr.GetStore()
	ctx := c.Request.Context()
	if err := store.SaveModelLoadConfig(ctx, config); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "保存模型加载配置失败", err.Error())
		return
	}

	logger.Info("模型加载配置已保存", "nodeId", nodeID, "modelId", modelID)

	api.Success(c, gin.H{
		"message": "配置已保存",
	})
}

// handleDeleteModelLoadConfig 删除模型加载配置
func (s *Server) handleDeleteModelLoadConfig(c *gin.Context) {
	modelID := c.Param("id")

	// 获取节点ID
	nodeID := "local"
	if s.nodeAdapter != nil {
		nodeID = s.nodeAdapter.GetNodeID()
	}

	// 从存储中删除配置
	store := s.storageMgr.GetStore()
	ctx := c.Request.Context()
	if err := store.DeleteModelLoadConfig(ctx, nodeID, modelID); err != nil {
		if err == storage.ErrModelLoadConfigNotFound {
			api.ErrorWithDetails(c, types.ErrInvalidRequest, "配置不存在", "模型加载配置不存在")
			return
		}
		api.ErrorWithDetails(c, types.ErrInternalError, "删除模型加载配置失败", err.Error())
		return
	}

	logger.Info("模型加载配置已删除", "nodeId", nodeID, "modelId", modelID)

	api.Success(c, gin.H{
		"message": "配置已删除",
	})
}

// isSafeFilename checks if a filename is safe (prevents directory traversal)
func isSafeFilename(filename string) bool {
	// Check for path traversal attempts
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return false
	}

	// Check file extension
	if !strings.HasSuffix(filename, ".log") {
		return false
	}

	// Check filename format (basic pattern match)
	// Format: shepherd-{mode}-{date} {time}.log or shepherd-{mode}-{date} {time}-{timestamp}-{reason}.log
	// Supports filenames with spaces and timestamps: shepherd-hybrid-2026-02-26 21-46-59.log
	pattern := regexp.MustCompile(`^shepherd-[a-z]+-\d{4}-\d{2}-\d{2}(?:\s\d{2}-\d{2}-\d{2})?(?:-\d{8}-\d{6}-[a-z]+)?\.log$`)
	return pattern.MatchString(filename)
}

func (s *Server) handleEvents(c *gin.Context) {
	s.wsMgr.HandleWebSocket(c)
}

func (s *Server) handleOpenAIChat(c *gin.Context) {
	s.handlers.OpenAI.HandleChatCompletions(c)
}

func (s *Server) handleOpenAIComplete(c *gin.Context) {
	s.handlers.OpenAI.HandleCompletions(c)
}

func (s *Server) handleOpenAIModels(c *gin.Context) {
	s.handlers.OpenAI.HandleModels(c)
}

func (s *Server) handleAnthropicMessages(c *gin.Context) {
	s.handlers.Anthropic.HandleMessages(c)
}

func (s *Server) handleOllamaChat(c *gin.Context) {
	s.handlers.Ollama.HandleChat(c)
}

func (s *Server) handleOllamaTags(c *gin.Context) {
	s.handlers.Ollama.HandleTags(c)
}
