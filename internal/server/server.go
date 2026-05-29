// Package server provides the HTTP server for the Shepherd application.
// It handles HTTP requests, routing, middleware, and serves the web UI.
package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/event"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	api "github.com/simonxluo/Shepherd/internal/handler"
	"github.com/simonxluo/Shepherd/internal/handler/anthropic"
	benchmarkapi "github.com/simonxluo/Shepherd/internal/handler/benchmark"
	chatapi "github.com/simonxluo/Shepherd/internal/handler/chat"
	compatibilityapi "github.com/simonxluo/Shepherd/internal/handler/compatibility"
	filesystemapi "github.com/simonxluo/Shepherd/internal/handler/filesystem"
	"github.com/simonxluo/Shepherd/internal/handler/lmstudio"
	mcpapi "github.com/simonxluo/Shepherd/internal/handler/mcp"
	"github.com/simonxluo/Shepherd/internal/handler/ollama"
	"github.com/simonxluo/Shepherd/internal/handler/openai"
	"github.com/simonxluo/Shepherd/internal/handler/paths"
	storageapi "github.com/simonxluo/Shepherd/internal/handler/storage"
	ttsapi "github.com/simonxluo/Shepherd/internal/handler/tts"
	"github.com/simonxluo/Shepherd/internal/infra/download"
	modelrepoclient "github.com/simonxluo/Shepherd/internal/infra/modelrepo"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/router"
	"github.com/simonxluo/Shepherd/internal/service/mcp"
	"github.com/simonxluo/Shepherd/internal/service/model"
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
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      string                 `json:"status"`
	IsLoaded    bool                   `json:"isLoaded"`
	Port        int                    `json:"port,omitempty"`
	ScannedAt   string                 `json:"scannedAt,omitempty"`   // 扫描时间（ISO 8601 格式）
	BackendType string                 `json:"backendType,omitempty"` // 推荐后端类型 (llamacpp/vllm/vllm_omni)
}

// Server represents the HTTP server
type Server struct {
	engine      *gin.Engine
	httpServer  *http.Server
	config      *Config
	handlers    *router.Handlers
	wsMgr       *event.Manager
	modelMgr    *model.Manager
	storageMgr  *storage.Manager
	downloadMgr *download.Manager
	nodeAdapter *api.NodeAdapter
	repoClient  *modelrepoclient.Client
	compatMgr   *compatibilityapi.ServerManager
	mcpService  *mcp.Service

	wsHub *WebSocketHub

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Config contains server configuration
type Config struct {
	WebPort       int
	AnthropicPort int
	Host          string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	WebUIPath     string
	ServerCfg     *config.Config
	ConfigMgr     *config.Manager // 配置管理器
	StorageMgr    *storage.Manager
	// Version information
	Version   string // 版本号
	BuildTime string // 构建时间
	GitCommit string // Git commit hash
}

// NewServer creates a new HTTP server
func NewServer(config *Config, modelMgr *model.Manager) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
		handlers: &router.Handlers{},
		modelMgr: modelMgr,
	}

	// Initialize storage manager
	var storageMgr *storage.Manager
	if config.StorageMgr != nil {
		storageMgr = config.StorageMgr
	} else {
		var err error
		storageMgr, err = storage.NewManager(&config.ServerCfg.Storage)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to initialize storage manager: %w", err)
		}
	}
	s.storageMgr = storageMgr

	// Create WebSocket manager
	s.wsMgr = event.NewManager(modelMgr.GetLoadedModelCount)

	// Load application config (used for download, model repo, compatibility, etc.)
	cfg := config.ConfigMgr.Get()

	// Create download manager with config from YAML
	dlCfg := cfg.Download
	s.downloadMgr = download.NewManager(download.DownloadConfig{
		MaxConcurrent: dlCfg.MaxConcurrent,
		ChunkSize:     int64(dlCfg.ChunkSize),
		Timeout:       time.Duration(dlCfg.Timeout) * time.Second,
		RetryCount:    dlCfg.RetryCount,
	}, download.ManagerDeps{
		Store:    storageMgr.GetStore(),
		EventMgr: s.wsMgr,
	})

	// Create WebSocket Hub
	s.wsHub = NewWebSocketHub()

	// Create model repository client with config
	timeout := time.Duration(cfg.ModelRepo.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	s.repoClient = modelrepoclient.NewClientWithConfig(cfg.ModelRepo.Endpoint, cfg.ModelRepo.Token, timeout)

	// Create compatibility server manager
	s.compatMgr = compatibilityapi.NewServerManager(modelMgr)

	// Create API handlers
	s.handlers.OpenAI = openai.NewHandler(modelMgr)
	s.handlers.Ollama = ollama.NewHandler(modelMgr)
	s.handlers.Anthropic = anthropic.NewHandler(modelMgr)
	s.handlers.LMStudio = lmstudio.NewHandler(modelMgr)
	s.handlers.Audio = openai.NewAudioHandler(modelMgr)
	s.handlers.Image = openai.NewImageHandler(modelMgr)
	s.handlers.Music = openai.NewMusicHandler(modelMgr)
	s.handlers.Paths = paths.NewHandler(config.ConfigMgr)
	s.handlers.Storage = storageapi.NewHandler(config.ConfigMgr, storageMgr)
	s.handlers.Compatibility = compatibilityapi.NewHandler(config.ConfigMgr, s.compatMgr)
	s.handlers.Filesystem = filesystemapi.NewHandler()
	s.handlers.Benchmark = benchmarkapi.NewHandler(logger.GetLogger(), storageMgr.GetStore(), modelMgr)
	s.handlers.Chat = chatapi.NewHandler(modelMgr)
	s.handlers.TTS = ttsapi.NewHandler(storageMgr, "./data/tts")

	// Initialize MCP service
	mcpSvc := mcp.NewService(config.ConfigMgr, modelMgr, storageMgr.GetStore())
	if err := mcpSvc.Initialize(); err != nil {
		logger.Warnf("MCP service initialization failed: %v", err)
	}
	s.mcpService = mcpSvc
	s.handlers.MCP = mcpapi.NewHandler(config.ConfigMgr, mcpSvc)

	// Auto-start compatibility servers if config says enabled
	if cfg.Compatibility.Ollama.Enabled {
		if err := s.compatMgr.StartOllamaServer(cfg.Compatibility.Ollama.Port); err != nil {
			logger.Warnf("自动启动 Ollama 兼容服务器失败: %v", err)
		}
	}
	if cfg.Compatibility.LMStudio.Enabled {
		if err := s.compatMgr.StartLMStudioServer(cfg.Compatibility.LMStudio.Port); err != nil {
			logger.Warnf("自动启动 LM Studio 兼容服务器失败: %v", err)
		}
	}

	// Setup Gin engine
	if config.WebUIPath == "" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	s.engine = gin.New()

	return s, nil
}

// SetupRoutes finalizes route registration. Must be called after all adapter
// registrations (RegisterNodeAdapter) and before Start.
func (s *Server) SetupRoutes() {
	router.Setup(s.engine, s.handlers, s, router.Config{WebUIPath: s.config.WebUIPath}, s.nodeAdapter)
}

// RegisterNodeAdapter registers the Node API adapter for later route setup.
func (s *Server) RegisterNodeAdapter(nodeAdapter *api.NodeAdapter) {
	s.nodeAdapter = nodeAdapter

	nodeAdapter.SetEventCallback(func(eventType string, data interface{}) {
		if s.wsHub != nil {
			s.wsHub.Emit(eventType, data)
		}
	})
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

	// Restore and resume interrupted downloads
	s.downloadMgr.RestoreAndResume()

	// Start WebSocket Hub (新增)
	go s.wsHub.Run()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.WebPort)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: 0, // SSE 长连接需要无限写超时
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

	// Step 4.1: Stop WebSocket Hub
	if s.wsHub != nil {
		logger.Info("停止 WebSocket Hub...")
		s.wsHub.Stop()
		logger.Info("WebSocket Hub 已停止")
	}

	// Step 4.5: Stop download manager
	logger.Info("停止下载管理器...")
	if s.downloadMgr != nil {
		if err := s.downloadMgr.Close(); err != nil {
			logger.Errorf("下载管理器关闭失败: %v", err)
		} else {
			logger.Info("下载管理器已停止")
		}
	}

	// Step 5: Stop compatibility servers (Ollama/LM Studio)
	if s.compatMgr != nil {
		logger.Info("停止兼容性服务器...")
		_ = s.compatMgr.StopOllamaServer()
		_ = s.compatMgr.StopLMStudioServer()
		logger.Info("兼容性服务器已停止")
	}

	// Step 5.5: Stop MCP service
	if s.mcpService != nil {
		logger.Info("停止 MCP 服务...")
		s.mcpService.Stop()
		logger.Info("MCP 服务已停止")
	}

	// 注意：storage manager 的关闭由 run_server.go 的 shutdown hook 统一管理，不在此处关闭

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
