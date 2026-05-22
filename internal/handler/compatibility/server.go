package compatibility

import (
	"context"
	"fmt"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/handler/lmstudio"
	"github.com/simonxluo/Shepherd/internal/handler/ollama"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

type ServerManager struct {
	ollamaServer   *http.Server
	lmstudioServer *http.Server
	modelMgr       *model.Manager
	mu             sync.RWMutex
}

func NewServerManager(modelMgr *model.Manager) *ServerManager {
	return &ServerManager{
		modelMgr: modelMgr,
	}
}

// StartOllamaServer starts the Ollama compatibility server on the specified port
func (sm *ServerManager) StartOllamaServer(port int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.ollamaServer != nil {
		return fmt.Errorf("Ollama server already running")
	}

	// Create a minimal Gin engine
	engine := gin.New()
	engine.Use(gin.Recovery())

	ollamaHandler := ollama.NewHandler(sm.modelMgr)
	api := engine.Group("/api")
	{
		api.GET("/tags", func(c *gin.Context) {
			ollamaHandler.HandleTags(c)
		})
		api.POST("/chat", func(c *gin.Context) {
			ollamaHandler.HandleChat(c)
		})
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%d", port)
	sm.ollamaServer = &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in background
	go func() {
		logger.Infof("启动 Ollama 兼容服务器，监听 %s", addr)
		if err := sm.ollamaServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Ollama 服务器错误: %v", err)
		}
		logger.Info("Ollama 服务器已停止")
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// StopOllamaServer stops the Ollama compatibility server
func (sm *ServerManager) StopOllamaServer() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.ollamaServer == nil {
		return nil
	}

	logger.Info("停止 Ollama 服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sm.ollamaServer.Shutdown(ctx); err != nil {
		logger.Errorf("Ollama 服务器关闭失败: %v", err)
		utils.CloseQuietly(sm.ollamaServer)
	}

	sm.ollamaServer = nil
	logger.Info("Ollama 服务器已停止")
	return nil
}

// StartLMStudioServer starts the LM Studio compatibility server on the specified port
func (sm *ServerManager) StartLMStudioServer(port int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.lmstudioServer != nil {
		return fmt.Errorf("LM Studio server already running")
	}

	// Create a minimal Gin engine
	engine := gin.New()
	engine.Use(gin.Recovery())

	// Setup OpenAI compatible routes for LM Studio
	lmstudioHandler := lmstudio.NewHandler(sm.modelMgr)
	v1 := engine.Group("/v1")
	{
		v1.GET("/models", func(c *gin.Context) {
			lmstudioHandler.HandleModels(c)
		})
		v1.POST("/chat/completions", func(c *gin.Context) {
			lmstudioHandler.HandleChatCompletions(c)
		})
		v1.POST("/completions", func(c *gin.Context) {
			lmstudioHandler.HandleCompletions(c)
		})
		v1.POST("/embeddings", func(c *gin.Context) {
			lmstudioHandler.HandleEmbeddings(c)
		})
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%d", port)
	sm.lmstudioServer = &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in background
	go func() {
		logger.Infof("启动 LM Studio 兼容服务器，监听 %s", addr)
		if err := sm.lmstudioServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("LM Studio 服务器错误: %v", err)
		}
		logger.Info("LM Studio 服务器已停止")
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// StopLMStudioServer stops the LM Studio compatibility server
func (sm *ServerManager) StopLMStudioServer() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.lmstudioServer == nil {
		return nil
	}

	logger.Info("停止 LM Studio 服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sm.lmstudioServer.Shutdown(ctx); err != nil {
		logger.Errorf("LM Studio 服务器关闭失败: %v", err)
		utils.CloseQuietly(sm.lmstudioServer)
	}

	sm.lmstudioServer = nil
	logger.Info("LM Studio 服务器已停止")
	return nil
}

// IsOllamaRunning returns whether the Ollama server is running
func (sm *ServerManager) IsOllamaRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.ollamaServer != nil
}

// IsLMStudioRunning returns whether the LM Studio server is running
func (sm *ServerManager) IsLMStudioRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.lmstudioServer != nil
}
