package tests

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/types"
	api "github.com/simonxluo/Shepherd/internal/handler"
	"github.com/simonxluo/Shepherd/internal/infra/download"
	"github.com/simonxluo/Shepherd/internal/infra/storage"
	"github.com/simonxluo/Shepherd/internal/router"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

// testServerHandlers implements router.ServerHandlers for testing.
// It provides real handler implementations that work against in-memory stores.
type testServerHandlers struct {
	config     *config.Config
	configMgr  *config.Manager
	modelMgr   *model.Manager
	storageMgr *storage.Manager
	downloadMgr *download.Manager
	handlers   *router.Handlers
}

// Verify interface implementation at compile time.
var _ router.ServerHandlers = (*testServerHandlers)(nil)

func newTestServerHandlers(
	cfg *config.Config,
	cfgMgr *config.Manager,
	modelMgr *model.Manager,
	storageMgr *storage.Manager,
	handlers *router.Handlers,
) *testServerHandlers {
	return &testServerHandlers{
		config:      cfg,
		configMgr:   cfgMgr,
		modelMgr:    modelMgr,
		storageMgr:  storageMgr,
		downloadMgr: download.NewManager(download.DownloadConfig{MaxConcurrent: 3}, download.ManagerDeps{}),
		handlers:    handlers,
	}
}

func (s *testServerHandlers) HandleEvents(c *gin.Context) {
	// SSE not testable via httptest; return immediately
	c.String(http.StatusOK, "event stream")
}

func (s *testServerHandlers) HandleGetResources(c *gin.Context) {
	api.Success(c, gin.H{"resources": nil})
}

func (s *testServerHandlers) HandleWebSocket(c *gin.Context) {
	c.String(http.StatusOK, "websocket")
}

func (s *testServerHandlers) HandleServerInfo(c *gin.Context) {
	api.Success(c, gin.H{
		"version":   "test",
		"buildTime": "2024-01-01T00:00:00Z",
		"gitCommit": "abc123",
		"name":      "Shepherd",
		"status":    "running",
		"role":      s.config.Node.Role,
		"ports": gin.H{
			"web":       s.config.Server.WebPort,
			"anthropic": s.config.Server.AnthropicPort,
			"ollama":    s.config.Server.OllamaPort,
			"lmstudio":  s.config.Server.LMStudioPort,
		},
	})
}

func (s *testServerHandlers) HandleGetGPUs(c *gin.Context) {
	api.Success(c, gin.H{
		"devices": []string{},
		"gpus":    []gin.H{},
		"count":   0,
	})
}

func (s *testServerHandlers) HandleGetLlamacppBackends(c *gin.Context) {
	api.Success(c, gin.H{
		"backends":          []gin.H{},
		"inferenceBackends": []gin.H{},
		"count":             0,
	})
}

func (s *testServerHandlers) HandleGetConfig(c *gin.Context) {
	cfg := s.config
	api.Success(c, gin.H{
		"role": cfg.Node.Role,
		"server": gin.H{
			"host":           cfg.Server.Host,
			"web_port":       cfg.Server.WebPort,
			"anthropic_port": cfg.Server.AnthropicPort,
			"ollama_port":    cfg.Server.OllamaPort,
			"lm_studio_port": cfg.Server.LMStudioPort,
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

func (s *testServerHandlers) HandleUpdateConfig(c *gin.Context) {
	var req struct {
		Mode      string   `json:"mode"`
		WebPort   int      `json:"web_port"`
		AutoScan  bool     `json:"auto_scan"`
		ScanPaths []string `json:"scan_paths"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "Invalid request format", err.Error())
		return
	}

	api.Success(c, gin.H{
		"message":          "Configuration updated",
		"restart_required": false,
	})
}

func (s *testServerHandlers) HandleListModels(c *gin.Context) {
	models := s.modelMgr.ListModels()
	dtos := make([]gin.H, 0, len(models))
	for _, m := range models {
		dtos = append(dtos, gin.H{
			"id":     m.ID,
			"name":   m.Name,
			"path":   m.Path,
			"size":   m.Size,
			"status": "idle",
		})
	}
	api.Success(c, gin.H{
		"models": dtos,
		"total":  len(dtos),
	})
}

func (s *testServerHandlers) HandleListLoadedModels(c *gin.Context) {
	api.Success(c, gin.H{
		"models": []interface{}{},
		"total":  0,
	})
}

func (s *testServerHandlers) HandleGetModelCapabilities(c *gin.Context) {
	modelID := c.Query("modelId")
	if modelID == "" {
		api.BadRequest(c, "modelId is required")
		return
	}
	caps := s.modelMgr.GetModelCapabilities(modelID)
	api.Success(c, gin.H{"capabilities": caps})
}

func (s *testServerHandlers) HandleSetModelCapabilities(c *gin.Context) {
	api.Success(c, gin.H{"message": "capabilities updated"})
}

func (s *testServerHandlers) HandleAutoDetectCapabilities(c *gin.Context) {
	api.Success(c, gin.H{"capabilities": nil})
}

func (s *testServerHandlers) HandleEstimateVRAM(c *gin.Context) {
	api.Success(c, gin.H{"estimate": 0})
}

func (s *testServerHandlers) HandleGetModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "model id is required")
		return
	}
	api.NotFound(c, "Model")
}

func (s *testServerHandlers) HandleLoadModel(c *gin.Context) {
	api.Success(c, gin.H{"message": "load initiated"})
}

func (s *testServerHandlers) HandleUnloadModel(c *gin.Context) {
	api.Success(c, gin.H{"message": "unload initiated"})
}

func (s *testServerHandlers) HandleSetAlias(c *gin.Context) {
	api.Success(c, gin.H{"message": "alias set"})
}

func (s *testServerHandlers) HandleSetFavourite(c *gin.Context) {
	api.Success(c, gin.H{"message": "favourite updated"})
}

func (s *testServerHandlers) HandleGetModelLoadConfig(c *gin.Context) {
	api.Success(c, gin.H{"config": nil})
}

func (s *testServerHandlers) HandleSaveModelLoadConfig(c *gin.Context) {
	api.Success(c, gin.H{"message": "config saved"})
}

func (s *testServerHandlers) HandleDeleteModelLoadConfig(c *gin.Context) {
	api.Success(c, gin.H{"message": "config deleted"})
}

func (s *testServerHandlers) HandleListModelLoadConfigs(c *gin.Context) {
	api.Success(c, gin.H{"configs": []interface{}{}})
}

func (s *testServerHandlers) HandleSaveNamedModelLoadConfig(c *gin.Context) {
	api.Success(c, gin.H{"message": "named config saved"})
}

func (s *testServerHandlers) HandleDeleteNamedModelLoadConfig(c *gin.Context) {
	api.Success(c, gin.H{"message": "named config deleted"})
}

func (s *testServerHandlers) HandleScanModels(c *gin.Context) {
	api.Success(c, gin.H{"message": "scan initiated"})
}

func (s *testServerHandlers) HandleGetScanStatus(c *gin.Context) {
	status := s.modelMgr.GetScanStatus()
	api.Success(c, gin.H{
		"scanning": status.Scanning,
		"progress": status.Progress,
	})
}

func (s *testServerHandlers) HandleListDownloads(c *gin.Context) {
	tasks := s.downloadMgr.ListTasks()
	downloads := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		downloads = append(downloads, gin.H{
			"id":    t.ID,
			"state": t.State.String(),
		})
	}
	api.Success(c, gin.H{
		"downloads": downloads,
		"total":     len(downloads),
	})
}

func (s *testServerHandlers) HandleCreateDownload(c *gin.Context) {
	api.Success(c, gin.H{"message": "download created"})
}

func (s *testServerHandlers) HandleGetDownload(c *gin.Context) {
	api.NotFound(c, "Download")
}

func (s *testServerHandlers) HandlePauseDownload(c *gin.Context) {
	api.Success(c, gin.H{"message": "download paused"})
}

func (s *testServerHandlers) HandleResumeDownload(c *gin.Context) {
	api.Success(c, gin.H{"message": "download resumed"})
}

func (s *testServerHandlers) HandleDeleteDownload(c *gin.Context) {
	api.Success(c, gin.H{"message": "download deleted"})
}

func (s *testServerHandlers) HandleListModelFiles(c *gin.Context) {
	api.Success(c, gin.H{"files": []interface{}{}})
}

func (s *testServerHandlers) HandleSearchModels(c *gin.Context) {
	api.Success(c, gin.H{"models": []interface{}{}})
}

func (s *testServerHandlers) HandleGetModelRepoConfig(c *gin.Context) {
	api.Success(c, gin.H{"config": s.config.ModelRepo})
}

func (s *testServerHandlers) HandleUpdateModelRepoConfig(c *gin.Context) {
	api.Success(c, gin.H{"message": "repo config updated"})
}

func (s *testServerHandlers) HandleGetAvailableEndpoints(c *gin.Context) {
	api.Success(c, gin.H{"endpoints": []interface{}{}})
}

func (s *testServerHandlers) HandleListProcesses(c *gin.Context) {
	processMgr := s.modelMgr.GetProcessManager()
	running, loading := processMgr.ListAll()

	type ProcessInfo struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		PID     int    `json:"pid"`
		Port    int    `json:"port"`
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

func (s *testServerHandlers) HandleGetProcess(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "Process ID is required")
		return
	}

	processMgr := s.modelMgr.GetProcessManager()
	proc, exists := processMgr.Get(id)
	if !exists {
		api.NotFound(c, "Process")
		return
	}

	api.Success(c, gin.H{
		"process": gin.H{
			"id":      proc.ID,
			"name":    proc.Name,
			"running": proc.IsRunning(),
		},
	})
}

func (s *testServerHandlers) HandleStopProcess(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "Process ID is required")
		return
	}
	api.Success(c, gin.H{"message": "process stopped", "id": id})
}

func (s *testServerHandlers) HandleLogStreamText(c *gin.Context) {
	c.String(http.StatusOK, "log stream")
}

func (s *testServerHandlers) HandleOpenAIChat(c *gin.Context) {
	s.handlers.OpenAI.HandleChatCompletions(c)
}

func (s *testServerHandlers) HandleOpenAIComplete(c *gin.Context) {
	s.handlers.OpenAI.HandleCompletions(c)
}

func (s *testServerHandlers) HandleOpenAIModels(c *gin.Context) {
	s.handlers.OpenAI.HandleModels(c)
}

func (s *testServerHandlers) HandleAnthropicMessages(c *gin.Context) {
	s.handlers.Anthropic.HandleMessages(c)
}

func (s *testServerHandlers) HandleOllamaChat(c *gin.Context) {
	s.handlers.Ollama.HandleChat(c)
}

func (s *testServerHandlers) HandleOllamaTags(c *gin.Context) {
	s.handlers.Ollama.HandleTags(c)
}

func (s *testServerHandlers) HandleLMStudioChat(c *gin.Context) {
	s.handlers.LMStudio.HandleChatCompletions(c)
}

func (s *testServerHandlers) HandleLMStudioComplete(c *gin.Context) {
	s.handlers.LMStudio.HandleCompletions(c)
}

func (s *testServerHandlers) HandleLMStudioModels(c *gin.Context) {
	s.handlers.LMStudio.HandleModels(c)
}

func (s *testServerHandlers) HandleLMStudioEmbeddings(c *gin.Context) {
	s.handlers.LMStudio.HandleEmbeddings(c)
}

func (s *testServerHandlers) HandleCreateSpeech(c *gin.Context) {
	s.handlers.Audio.HandleCreateSpeech(c)
}

func (s *testServerHandlers) HandleCreateTranscription(c *gin.Context) {
	s.handlers.Audio.HandleCreateTranscription(c)
}

func (s *testServerHandlers) HandleCreateTranslation(c *gin.Context) {
	s.handlers.Audio.HandleCreateTranslation(c)
}

func (s *testServerHandlers) HandleCreateImage(c *gin.Context) {
	s.handlers.Image.HandleCreateImage(c)
}

func (s *testServerHandlers) HandleCreateMusic(c *gin.Context) {
	s.handlers.Music.HandleCreateMusic(c)
}

func (s *testServerHandlers) HandleListVoices(c *gin.Context) {
	s.handlers.Audio.HandleListVoices(c)
}
