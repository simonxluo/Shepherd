// Package router centralizes all HTTP route registration for the Shepherd server.
package router

import (
	api "github.com/shepherd-project/shepherd/Shepherd/internal/handler"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler/anthropic"
	benchmarkapi "github.com/shepherd-project/shepherd/Shepherd/internal/handler/benchmark"
	compatibilityapi "github.com/shepherd-project/shepherd/Shepherd/internal/handler/compatibility"
	filesystemapi "github.com/shepherd-project/shepherd/Shepherd/internal/handler/filesystem"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler/ollama"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler/openai"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler/paths"
	storageapi "github.com/shepherd-project/shepherd/Shepherd/internal/handler/storage"
	"github.com/shepherd-project/shepherd/Shepherd/internal/langchain"
	"github.com/shepherd-project/shepherd/Shepherd/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
)

// Handlers contains all injected handler instances.
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

// ServerHandlers defines the interface for server-owned handler methods
// that are bound to the Server struct. The router calls these through this
// interface to decouple route definitions from the Server implementation.
type ServerHandlers interface {
	HandleEvents(c *gin.Context)
	HandleWebSocket(c *gin.Context)
	HandleServerInfo(c *gin.Context)
	HandleGetGPUs(c *gin.Context)
	HandleGetLlamacppBackends(c *gin.Context)
	HandleGetConfig(c *gin.Context)
	HandleUpdateConfig(c *gin.Context)
	HandleListModels(c *gin.Context)
	HandleListLoadedModels(c *gin.Context)
	HandleGetModelCapabilities(c *gin.Context)
	HandleSetModelCapabilities(c *gin.Context)
	HandleAutoDetectCapabilities(c *gin.Context)
	HandleEstimateVRAM(c *gin.Context)
	HandleGetModel(c *gin.Context)
	HandleLoadModel(c *gin.Context)
	HandleUnloadModel(c *gin.Context)
	HandleSetAlias(c *gin.Context)
	HandleSetFavourite(c *gin.Context)
	HandleGetModelLoadConfig(c *gin.Context)
	HandleSaveModelLoadConfig(c *gin.Context)
	HandleDeleteModelLoadConfig(c *gin.Context)
	HandleScanModels(c *gin.Context)
	HandleGetScanStatus(c *gin.Context)
	HandleListDownloads(c *gin.Context)
	HandleCreateDownload(c *gin.Context)
	HandleGetDownload(c *gin.Context)
	HandlePauseDownload(c *gin.Context)
	HandleResumeDownload(c *gin.Context)
	HandleDeleteDownload(c *gin.Context)
	HandleListModelFiles(c *gin.Context)
	HandleSearchModels(c *gin.Context)
	HandleGetModelRepoConfig(c *gin.Context)
	HandleUpdateModelRepoConfig(c *gin.Context)
	HandleGetAvailableEndpoints(c *gin.Context)
	HandleListProcesses(c *gin.Context)
	HandleGetProcess(c *gin.Context)
	HandleStopProcess(c *gin.Context)
	HandleLogStream(c *gin.Context)
	HandleLogEntries(c *gin.Context)
	HandleLogFiles(c *gin.Context)
	HandleLogFileContent(c *gin.Context)
	HandleLogFileStats(c *gin.Context)
	HandleDeleteLogFile(c *gin.Context)
	HandleOpenAIChat(c *gin.Context)
	HandleOpenAIComplete(c *gin.Context)
	HandleOpenAIModels(c *gin.Context)
	HandleAnthropicMessages(c *gin.Context)
	HandleOllamaChat(c *gin.Context)
	HandleOllamaTags(c *gin.Context)
}

// Config holds router-level configuration.
type Config struct {
	WebUIPath string
}

// Setup configures middleware and registers all routes on the given Gin engine.
// All route definitions live here; callers only need to provide the engine,
// handlers, and optional adapter instances.
func Setup(
	engine *gin.Engine,
	h *Handlers,
	sh ServerHandlers,
	cfg Config,
	nodeAdapter *api.NodeAdapter,
	langchainHandler *langchain.Handler,
) {
	setupMiddleware(engine)
	registerRoutes(engine, h, sh, cfg, nodeAdapter, langchainHandler)
}

// setupMiddleware configures global middleware on the engine.
func setupMiddleware(engine *gin.Engine) {
	engine.Use(
		middleware.RequestID(),
		middleware.RecoveryMiddleware(logger.GetLogger()),
		middleware.CORSMiddleware([]string{"*"}),
		middleware.LoggerMiddleware(logger.GetLogger()),
		middleware.ErrorHandler(logger.GetLogger()),
	)
}

// registerRoutes registers all application routes in one place.
func registerRoutes(
	engine *gin.Engine,
	h *Handlers,
	sh ServerHandlers,
	cfg Config,
	nodeAdapter *api.NodeAdapter,
	langchainHandler *langchain.Handler,
) {
	engine.GET("/api/events", sh.HandleEvents)
	engine.GET("/ws", sh.HandleWebSocket)

	apiGroup := engine.Group("/api")
	{
		apiGroup.GET("/info", sh.HandleServerInfo)
		apiGroup.GET("/system/gpus", sh.HandleGetGPUs)
		apiGroup.GET("/system/llamacpp-backends", sh.HandleGetLlamacppBackends)

		registerConfigRoutes(apiGroup, h, sh)
		registerConversationRoutes(apiGroup, h)
		registerModelRoutes(apiGroup, sh, h)
		registerModelScanRoutes(apiGroup, sh)
		registerDownloadRoutes(apiGroup, sh)
		registerRepoRoutes(apiGroup, sh)
		registerProcessRoutes(apiGroup, sh)
		registerLogRoutes(apiGroup, sh)
		registerSystemRoutes(apiGroup, h)

		apiGroup.GET("/model/device/list", h.Benchmark.GetDevices)
		apiGroup.GET("/models/param/benchmark/list", h.Benchmark.GetParams)
		apiGroup.GET("/llamacpp/list", h.Paths.GetLlamaCppPaths)
	}

	registerCompatibilityRoutes(engine, sh)

	if nodeAdapter != nil {
		nodeAdapter.RegisterRoutes(apiGroup)
	}

	if langchainHandler != nil {
		langchainHandler.RegisterRoutes(apiGroup)
	}

	registerStaticRoutes(engine, cfg)
}

func registerConfigRoutes(apiGroup *gin.RouterGroup, h *Handlers, sh ServerHandlers) {
	config := apiGroup.Group("/config")
	{
		config.GET("", sh.HandleGetConfig)
		config.PUT("", sh.HandleUpdateConfig)
	}
	{
		llamacpp := config.Group("/llamacpp/paths")
		{
			llamacpp.GET("", h.Paths.GetLlamaCppPaths)
			llamacpp.POST("", h.Paths.AddLlamaCppPath)
			llamacpp.PUT("", h.Paths.UpdateLlamaCppPath)
			llamacpp.DELETE("", h.Paths.RemoveLlamaCppPath)
			llamacpp.POST("/test", h.Paths.TestLlamaCppPath)
		}

		models := config.Group("/models/paths")
		{
			models.GET("", h.Paths.GetModelPaths)
			models.POST("", h.Paths.AddModelPath)
			models.PUT("", h.Paths.UpdateModelPath)
			models.DELETE("", h.Paths.RemoveModelPath)
		}

		storage := config.Group("/storage")
		{
			storage.GET("", h.Storage.GetStorageConfig)
			storage.PUT("", h.Storage.UpdateStorageConfig)
			storage.GET("/stats", h.Storage.GetStats)
		}

		compatibility := config.Group("/compatibility")
		{
			compatibility.GET("", h.Compatibility.GetCompatibility)
			compatibility.PUT("", h.Compatibility.UpdateCompatibility)
			compatibility.POST("/test", h.Compatibility.TestConnection)
		}
	}
}

func registerConversationRoutes(apiGroup *gin.RouterGroup, h *Handlers) {
	conversations := apiGroup.Group("/conversations")
	{
		conversations.GET("", h.Storage.GetConversations)
		conversations.GET("/:id", h.Storage.GetConversation)
		conversations.DELETE("/:id", h.Storage.DeleteConversation)
	}
}

func registerModelRoutes(apiGroup *gin.RouterGroup, sh ServerHandlers, h *Handlers) {
	models := apiGroup.Group("/models")
	{
		models.GET("", sh.HandleListModels)
		models.GET("/loaded", sh.HandleListLoadedModels)
		models.GET("/capabilities/get", sh.HandleGetModelCapabilities)
		models.POST("/capabilities/set", sh.HandleSetModelCapabilities)
		models.GET("/capabilities/auto-detect", sh.HandleAutoDetectCapabilities)
		models.POST("/vram/estimate", sh.HandleEstimateVRAM)

		benchmark := models.Group("/benchmark")
		{
			benchmark.POST("", h.Benchmark.Create)
			benchmark.GET("/tasks", h.Benchmark.List)
			benchmark.GET("/tasks/:benchmarkId", h.Benchmark.Get)
			benchmark.POST("/tasks/:benchmarkId/cancel", h.Benchmark.Cancel)
			benchmark.GET("/configs", h.Benchmark.ListConfigs)
			benchmark.POST("/configs", h.Benchmark.SaveConfig)
			benchmark.GET("/configs/:name", h.Benchmark.GetConfig)
			benchmark.DELETE("/configs/:name", h.Benchmark.DeleteConfig)
		}

		models.GET("/:id", sh.HandleGetModel)
		models.POST("/:id/load", sh.HandleLoadModel)
		models.POST("/:id/unload", sh.HandleUnloadModel)
		models.PUT("/:id/alias", sh.HandleSetAlias)
		models.PUT("/:id/favourite", sh.HandleSetFavourite)
		models.GET("/:id/load-config", sh.HandleGetModelLoadConfig)
		models.PUT("/:id/load-config", sh.HandleSaveModelLoadConfig)
		models.DELETE("/:id/load-config", sh.HandleDeleteModelLoadConfig)
	}
}

func registerModelScanRoutes(apiGroup *gin.RouterGroup, sh ServerHandlers) {
	modelScan := apiGroup.Group("/model/scan")
	{
		modelScan.POST("", sh.HandleScanModels)
		modelScan.GET("/status", sh.HandleGetScanStatus)
	}
}

func registerDownloadRoutes(apiGroup *gin.RouterGroup, sh ServerHandlers) {
	downloads := apiGroup.Group("/downloads")
	{
		downloads.GET("", sh.HandleListDownloads)
		downloads.POST("", sh.HandleCreateDownload)
		downloads.GET("/:id", sh.HandleGetDownload)
		downloads.POST("/:id/pause", sh.HandlePauseDownload)
		downloads.POST("/:id/resume", sh.HandleResumeDownload)
		downloads.DELETE("/:id", sh.HandleDeleteDownload)
	}
}

func registerRepoRoutes(apiGroup *gin.RouterGroup, sh ServerHandlers) {
	repo := apiGroup.Group("/repo")
	{
		repo.GET("/files", sh.HandleListModelFiles)
		repo.GET("/search", sh.HandleSearchModels)
		repo.GET("/config", sh.HandleGetModelRepoConfig)
		repo.PUT("/config", sh.HandleUpdateModelRepoConfig)
		repo.GET("/endpoints", sh.HandleGetAvailableEndpoints)
	}
}

func registerProcessRoutes(apiGroup *gin.RouterGroup, sh ServerHandlers) {
	processes := apiGroup.Group("/processes")
	{
		processes.GET("", sh.HandleListProcesses)
		processes.GET("/:id", sh.HandleGetProcess)
		processes.POST("/:id/stop", sh.HandleStopProcess)
	}
}

func registerLogRoutes(apiGroup *gin.RouterGroup, sh ServerHandlers) {
	logs := apiGroup.Group("/logs")
	{
		logs.GET("/stream", sh.HandleLogStream)
		logs.GET("/entries", sh.HandleLogEntries)
		logs.GET("/files", sh.HandleLogFiles)
		logs.GET("/files/:filename", sh.HandleLogFileContent)
		logs.GET("/files/:filename/stats", sh.HandleLogFileStats)
		logs.DELETE("/files/:filename", sh.HandleDeleteLogFile)
	}
}

func registerSystemRoutes(apiGroup *gin.RouterGroup, h *Handlers) {
	system := apiGroup.Group("/system")
	{
		system.GET("/filesystem", h.Filesystem.ListDirectory)
		system.POST("/filesystem/validate", h.Filesystem.ValidatePath)
	}
}

func registerCompatibilityRoutes(engine *gin.Engine, sh ServerHandlers) {
	openai := engine.Group("/v1")
	{
		openai.POST("/chat/completions", sh.HandleOpenAIChat)
		openai.POST("/completions", sh.HandleOpenAIComplete)
		openai.GET("/models", sh.HandleOpenAIModels)
	}

	anthropic := engine.Group("/v1")
	{
		anthropic.POST("/messages", sh.HandleAnthropicMessages)
	}

	ollama := engine.Group("/api")
	{
		ollama.POST("/chat", sh.HandleOllamaChat)
		ollama.POST("/tags", sh.HandleOllamaTags)
	}
}

func registerStaticRoutes(engine *gin.Engine, cfg Config) {
	if cfg.WebUIPath == "" {
		return
	}
	engine.Static("/assets", cfg.WebUIPath+"/assets")
	engine.Static("/favicon.svg", cfg.WebUIPath+"/favicon.svg")
	engine.StaticFile("/", cfg.WebUIPath+"/index.html")
}
