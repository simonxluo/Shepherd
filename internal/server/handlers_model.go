package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/comm/types"
	api "github.com/simonxluo/Shepherd/internal/handler"
	"github.com/simonxluo/Shepherd/internal/infra/gguf"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

// loadedModelInfo is used for /api/models/loaded response payload
type loadedModelInfo struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Alias        string                `json:"alias,omitempty"`
	State        string                `json:"state"`
	ProcessID    string                `json:"processId"`
	Port         int                   `json:"port"`
	CtxSize      int                   `json:"ctxSize"`
	PluginID     string                `json:"pluginId,omitempty"`
	LoadedAt     string                `json:"loadedAt,omitempty"`
	Capabilities *storage.Capabilities `json:"capabilities,omitempty"`
}

// HandleListModels returns all available models.
// @Summary      List all models
// @Description  Returns a list of all scanned models with their metadata and status
// @Tags         Models
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/models [get]
func (s *Server) HandleListModels(c *gin.Context) {
	models := s.modelMgr.ListModels()

	dtos := make([]ModelDTO, 0, len(models))
	statuses := s.modelMgr.ListStatus()

	for _, m := range models {
		dto := s.toModelDTO(m, statuses)
		dtos = append(dtos, dto)
	}

	api.Success(c, gin.H{
		"models": dtos,
		"total":  len(dtos),
	})
}

// HandleListLoadedModels returns all currently loaded models.
// @Summary      List loaded models
// @Description  Returns a list of models that are currently loaded or loading
// @Tags         Models
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/models/loaded [get]
func (s *Server) HandleListLoadedModels(c *gin.Context) {
	statuses := s.modelMgr.ListStatus()

	loaded := make([]loadedModelInfo, 0)
	for id, st := range statuses {
		if st.State == model.StateLoaded || st.State == model.StateLoading {
			info := loadedModelInfo{
				ID:        id,
				Name:      st.Name,
				State:     st.State.String(),
				ProcessID: st.ProcessID,
				Port:      st.Port,
				CtxSize:   st.CtxSize,
				PluginID:  st.PluginID,
			}
			if !st.LoadedAt.IsZero() {
				info.LoadedAt = st.LoadedAt.Format(time.RFC3339)
			}

			// Get model alias and capabilities
			if m, exists := s.modelMgr.GetModel(id); exists {
				info.Alias = m.Alias
			}
			if caps := s.modelMgr.GetModelCapabilities(id); caps != nil {
				info.Capabilities = caps
			}

			loaded = append(loaded, info)
		}
	}

	api.Success(c, gin.H{
		"models": loaded,
		"total":  len(loaded),
	})
}

// HandleGetModel returns details of a specific model.
// @Summary      Get model details
// @Description  Returns detailed information about a specific model by ID
// @Tags         Models
// @Produce      json
// @Param        id path string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /api/models/{id} [get]
func (s *Server) HandleGetModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	m, exists := s.modelMgr.GetModel(id)
	if !exists {
		api.NotFound(c, "模型")
		return
	}

	statuses := s.modelMgr.ListStatus()
	dto := s.toModelDTO(m, statuses)

	api.Success(c, dto)
}

// HandleLoadModel initiates loading a model into memory.
// @Summary      Load a model
// @Description  Initiates asynchronous loading of a model with specified parameters
// @Tags         Models
// @Accept       json
// @Produce      json
// @Param        id path string true "Model ID"
// @Param        request body object true "Load configuration parameters"
// @Success      202 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/load [post]
func (s *Server) HandleLoadModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	var req model.LoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求格式", err.Error())
		return
	}

	req.ModelID = id

	if req.ProfileID != "" {
		profile, err := s.storageMgr.GetStore().GetLaunchProfile(c.Request.Context(), req.ProfileID)
		if err != nil {
			api.NotFound(c, "Launch profile")
			return
		}
		applyLaunchProfileToLoadRequest(&req, profile)
	}

	// Backend-specific parameter processing.
	// Determine the effective backend early and branch all backend-specific
	// logic (validation, spec-decoding, defaults) into a single block.
	// This avoids scattering backend-type checks across the handler and
	// prevents llama.cpp parameters from leaking into vLLM/vLLM-Omni.
	// Validate backend-specific parameters via the plugin.
	pluginID := backend.ID(req.PluginID)
	if pluginID == "" {
		pluginID = backend.IDLlamaCpp
	}
	if plugin, ok := backend.Default().Get(pluginID); ok {
		rawParams := model.LoadRequestToRawParams(&req)
		validation := plugin.ValidateParams(rawParams)
		if !validation.Valid {
			api.ErrorWithDetails(c, types.ErrInvalidRequest, "invalid parameters", strings.Join(validation.Errors, "; "))
			return
		}
	}

	// Spec-decoding draft model validation (llamacpp-specific).
	if pluginID == backend.IDLlamaCpp {
		if req.SpecDecoding != nil && (req.SpecDecoding.SpecType == "draft" || req.SpecDecoding.SpecType == "eagle3") && req.SpecDecoding.SpecDraftModelID != "" {
			if req.SpecDecoding.SpecDraftModelID == id {
				api.BadRequest(c, "Draft模型不能与主模型相同")
				return
			}
			draftModel, exists := s.modelMgr.GetModel(req.SpecDecoding.SpecDraftModelID)
			if !exists {
				api.BadRequest(c, fmt.Sprintf("Draft模型未找到: %s", req.SpecDecoding.SpecDraftModelID))
				return
			}
			draftPath := draftModel.Path
			if len(draftModel.ShardFiles) > 0 {
				draftPath = draftModel.ShardFiles[0]
			}
			if _, err := os.Stat(draftPath); err != nil {
				api.BadRequest(c, fmt.Sprintf("Draft模型文件不可访问: %s", draftPath))
				return
			}
			mainModel, mainExists := s.modelMgr.GetModel(id)
			if !mainExists {
				api.BadRequest(c, "主模型未找到")
				return
			}
			mainPath := mainModel.Path
			if len(mainModel.ShardFiles) > 0 {
				mainPath = mainModel.ShardFiles[0]
			}
			mainArch := getArchitecture(mainPath)
			draftArch := getArchitecture(draftPath)
			if mainArch == "" || draftArch == "" {
				api.BadRequest(c, "无法读取模型架构信息，请确保模型文件有效")
				return
			}
			if !strings.EqualFold(mainArch, draftArch) {
				api.BadRequest(c, fmt.Sprintf("Draft模型架构(%s)与主模型架构(%s)不匹配", draftArch, mainArch))
				return
			}
			req.SpecDecoding.SpecDraftModelPath = draftPath
			logger.Infof("draft model resolved: modelId=%s, draftModelId=%s, draftPath=%s, arch=%s", id, req.SpecDecoding.SpecDraftModelID, draftPath, draftArch)
		}

		// Apply llama.cpp-specific defaults (zero values mean "not set").
		if req.CtxSize == 0 {
			req.CtxSize = 512
		}
		if req.BatchSize == 0 {
			req.BatchSize = 512
		}
		if req.Threads == 0 {
			req.Threads = -1
		}
		if req.Temperature == 0 {
			req.Temperature = 0.7
		}
		if req.TopP == 0 {
			req.TopP = 0.95
		}
		if req.TopK == 0 {
			req.TopK = 40
		}
		if req.RepeatPenalty == 0 {
			req.RepeatPenalty = 1.1
		}
	}

	// No longer force vllm_omni backend — let Resolve's capability-aware routing handle it:
	// - If vllm_omni is configured, capability-aware routing will prefer it
	// - If not configured, GGUF models fall back to llama.cpp
	// Explicit PluginID from frontend is still respected

	result, err := s.modelMgr.LoadAsync(&req)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "模型加载失败", err.Error())
		return
	}

	respData := gin.H{
		"id":         id,
		"instanceId": result.InstanceID,
		"status":     "loading",
	}

	if result.AlreadyLoaded {
		respData["status"] = "running"
		respData["port"] = result.Port
	}

	if result.Port > 0 {
		respData["port"] = result.Port
	}

	api.Accepted(c, respData)
}

// HandleUnloadModel unloads a model from memory.
// @Summary      Unload a model
// @Description  Unloads a currently loaded model and stops its process
// @Tags         Models
// @Produce      json
// @Param        id path string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/unload [post]
func (s *Server) HandleUnloadModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	if err := s.modelMgr.Unload(id); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "模型卸载失败", err.Error())
		return
	}

	if s.wsHub != nil {
		s.wsHub.Emit("model_unload", gin.H{"modelId": id})
	}

	api.SuccessWithMessage(c, "模型已卸载")
}

// HandleSetAlias sets an alias for a model.
// @Summary      Set model alias
// @Description  Sets or updates the alias name for a model
// @Tags         Models
// @Accept       json
// @Produce      json
// @Param        id path string true "Model ID"
// @Param        request body object true "Alias request with alias field"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/alias [put]
func (s *Server) HandleSetAlias(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	var req struct {
		Alias string `json:"alias"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求格式", err.Error())
		return
	}

	if err := s.modelMgr.SetAlias(id, req.Alias); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "设置别名失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"modelId": id,
		"alias":   req.Alias,
	})
}

// HandleSetFavourite sets the favourite status for a model.
// @Summary      Set model favourite status
// @Description  Sets or removes a model from the favourites list
// @Tags         Models
// @Accept       json
// @Produce      json
// @Param        id path string true "Model ID"
// @Param        request body object true "Favourite request with favourite field"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/favourite [put]
func (s *Server) HandleSetFavourite(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	var req struct {
		Favourite bool `json:"favourite"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求格式", err.Error())
		return
	}

	if err := s.modelMgr.SetFavourite(id, req.Favourite); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "设置收藏失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"modelId":   id,
		"favourite": req.Favourite,
	})
}

// HandleGetModelCapabilities returns the capabilities of a model.
// @Summary      Get model capabilities
// @Description  Returns the capabilities (thinking, tools, rerank, embedding, etc.) of a model
// @Tags         Models
// @Produce      json
// @Param        modelId query string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /api/models/capabilities/get [get]
func (s *Server) HandleGetModelCapabilities(c *gin.Context) {
	modelID := c.Query("modelId")
	if modelID == "" {
		api.BadRequest(c, "缺少必要参数: modelId")
		return
	}

	ctx := context.Background()
	meta, err := s.storageMgr.GetStore().GetModelMetadata(ctx, modelID)
	if err != nil {
		logger.Infof("HandleGetModelCapabilities: GetModelMetadata failed: modelId=%s, err=%v, storeType=%T", modelID, err, s.storageMgr.GetStore())
		api.Success(c, gin.H{
			"modelId":      modelID,
			"capabilities": &storage.Capabilities{},
		})
		return
	}

	caps := meta.Capabilities
	if caps == nil {
		caps = &storage.Capabilities{}
	}

	logger.Infof("HandleGetModelCapabilities: modelId=%s, caps=%+v", modelID, caps)

	api.Success(c, gin.H{
		"modelId":      modelID,
		"capabilities": caps,
	})
}

// HandleSetModelCapabilities sets the capabilities of a model.
// @Summary      Set model capabilities
// @Description  Sets the capabilities for a model (thinking, tools, rerank, embedding, TTS, ASR, etc.)
// @Tags         Models
// @Accept       json
// @Produce      json
// @Param        request body object true "Capabilities request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/capabilities/set [post]
func (s *Server) HandleSetModelCapabilities(c *gin.Context) {
	var req struct {
		ModelID         string `json:"modelId"`
		Thinking        bool   `json:"thinking"`
		Tools           bool   `json:"tools"`
		Rerank          bool   `json:"rerank"`
		Embedding       bool   `json:"embedding"`
		TTS             bool   `json:"tts"`
		ASR             bool   `json:"asr"`
		ImageGeneration bool   `json:"imageGeneration"`
		Music           bool   `json:"music"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求格式", err.Error())
		return
	}

	if req.ModelID == "" {
		api.BadRequest(c, "缺少必要参数: modelId")
		return
	}

	caps := &storage.Capabilities{
		Thinking:        req.Thinking,
		Tools:           req.Tools,
		Rerank:          req.Rerank,
		Embedding:       req.Embedding,
		TTS:             req.TTS,
		ASR:             req.ASR,
		ImageGeneration: req.ImageGeneration,
		Music:           req.Music,
	}
	caps.ApplyConstraints()

	ctx := context.Background()
	meta, err := s.storageMgr.GetStore().GetModelMetadata(ctx, req.ModelID)
	if err != nil {
		meta = &storage.ModelMetadata{
			ModelID: req.ModelID,
		}
	}
	meta.Capabilities = caps

	if err := s.storageMgr.GetStore().SaveModelMetadata(ctx, meta); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "保存模型能力失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"modelId":      req.ModelID,
		"capabilities": caps,
	})
}

// HandleAutoDetectCapabilities auto-detects model capabilities.
// @Summary      Auto-detect model capabilities
// @Description  Automatically detects model capabilities based on model metadata and architecture
// @Tags         Models
// @Produce      json
// @Param        modelId query string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/capabilities/auto-detect [get]
func (s *Server) HandleAutoDetectCapabilities(c *gin.Context) {
	modelID := c.Query("modelId")
	if modelID == "" {
		api.BadRequest(c, "缺少必要参数: modelId")
		return
	}

	caps, err := s.modelMgr.AutoDetectCapabilities(modelID)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "自动检测模型能力失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"modelId":      modelID,
		"capabilities": caps,
	})
}

// HandleEstimateVRAM estimates VRAM usage for loading a model.
// @Summary      Estimate VRAM usage
// @Description  Estimates the VRAM required to load a model with specified context size
// @Tags         Models
// @Accept       json
// @Produce      json
// @Param        request body object true "VRAM estimation request with modelId and ctxSize"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /api/models/vram/estimate [post]
func (s *Server) HandleEstimateVRAM(c *gin.Context) {
	var req struct {
		ModelID string `json:"modelId"`
		CtxSize int    `json:"ctxSize"`
		GPUType string `json:"gpuType"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求格式", err.Error())
		return
	}

	if req.ModelID == "" {
		api.BadRequest(c, "缺少必要参数: modelId")
		return
	}

	m, exists := s.modelMgr.GetModel(req.ModelID)
	if !exists {
		api.NotFound(c, "模型")
		return
	}

	ctxSize := req.CtxSize
	if ctxSize == 0 {
		ctxSize = 4096
	}

	modelSize := m.Size
	if m.TotalSize > 0 {
		modelSize = m.TotalSize
	}

	bitsPerWeight := 4.0
	if m.Metadata != nil && m.Metadata.BitsPerWeight > 0 {
		bitsPerWeight = m.Metadata.BitsPerWeight
	}

	kvCachePerToken := float64(0)
	if m.Metadata != nil {
		embeddingLen := m.Metadata.EmbeddingLength
		headCountKV := m.Metadata.HeadCountKV
		if embeddingLen > 0 {
			if headCountKV > 0 {
				kvCachePerToken = float64(2*embeddingLen*headCountKV) * bitsPerWeight / 8
			} else {
				kvCachePerToken = float64(2*embeddingLen) * bitsPerWeight / 8
			}
		}
	}
	if kvCachePerToken == 0 {
		kvCachePerToken = float64(modelSize) * 0.0001
	}

	kvCacheBytes := kvCachePerToken * float64(ctxSize)
	totalVRAM := float64(modelSize) + kvCacheBytes
	overhead := totalVRAM * 0.05
	totalVRAM += overhead

	api.Success(c, gin.H{
		"modelId":         req.ModelID,
		"modelSizeBytes":  modelSize,
		"modelSizeGB":     fmt.Sprintf("%.2f", float64(modelSize)/1024/1024/1024),
		"kvCacheBytes":    uint64(kvCacheBytes),
		"kvCacheGB":       fmt.Sprintf("%.2f", kvCacheBytes/1024/1024/1024),
		"estimatedVRAM":   uint64(totalVRAM),
		"estimatedVRAMGB": fmt.Sprintf("%.2f", totalVRAM/1024/1024/1024),
		"ctxSize":         ctxSize,
		"bitsPerWeight":   bitsPerWeight,
	})
}

// HandleScanModels triggers a model scan across configured paths.
// @Summary      Scan for models
// @Description  Triggers a scan of all configured model paths to discover GGUF models
// @Tags         Scan
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/model/scan [post]
func (s *Server) HandleScanModels(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	result, err := s.modelMgr.Scan(ctx)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "模型扫描失败", err.Error())
		return
	}

	scanErrors := make([]gin.H, 0, len(result.Errors))
	for _, e := range result.Errors {
		scanErrors = append(scanErrors, gin.H{
			"path":  e.Path,
			"error": e.Error,
		})
	}

	if s.wsHub != nil {
		s.wsHub.Emit("scan_complete", gin.H{
			"totalModels": len(result.Models),
			"errors":      len(result.Errors),
			"duration":    result.Duration.String(),
		})
	}

	api.Success(c, gin.H{
		"models":       len(result.Models),
		"errors":       scanErrors,
		"scannedAt":    result.ScannedAt.Format(time.RFC3339),
		"duration":     result.Duration.String(),
		"totalFiles":   result.TotalFiles,
		"matchedFiles": result.MatchedFiles,
	})
}

// HandleGetScanStatus returns the current model scan status.
// @Summary      Get scan status
// @Description  Returns the current status of model scanning including progress and errors
// @Tags         Scan
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/model/scan/status [get]
func (s *Server) HandleGetScanStatus(c *gin.Context) {
	status := s.modelMgr.GetScanStatus()

	resp := gin.H{
		"scanning": status.Scanning,
		"progress": status.Progress,
	}

	if status.Scanning {
		resp["currentPath"] = status.CurrentPath
		resp["startedAt"] = status.StartedAt.Format(time.RFC3339)
	}

	if len(status.Errors) > 0 {
		errs := make([]gin.H, 0, len(status.Errors))
		for _, e := range status.Errors {
			errs = append(errs, gin.H{
				"path":  e.Path,
				"error": e.Error,
			})
		}
		resp["errors"] = errs
	}

	api.Success(c, resp)
}

func (s *Server) getNodeID() string {
	if s.config != nil && s.config.ServerCfg != nil && s.config.ServerCfg.Node.ID != "" {
		return s.config.ServerCfg.Node.ID
	}
	return "local"
}

func (s *Server) toModelDTO(m *model.Model, statuses map[string]*model.ModelStatus) ModelDTO {
	status := "stopped"
	isLoaded := false
	if st, ok := statuses[m.ID]; ok {
		status = st.State.String()
		isLoaded = st.State == model.StateLoaded
	}

	scannedAt := ""
	if !m.ScannedAt.IsZero() {
		scannedAt = m.ScannedAt.Format(time.RFC3339)
	}

	dto := ModelDTO{
		ID:          m.ID,
		Name:        m.Name,
		DisplayName: m.DisplayName,
		Alias:       m.Alias,
		Path:        m.Path,
		PathPrefix:  m.PathPrefix,
		Size:        m.Size,
		TotalSize:   m.TotalSize,
		ShardCount:  m.ShardCount,
		ShardFiles:  m.ShardFiles,
		MmprojPath:  m.MmprojPath,
		Favourite:   m.Favourite,
		Tags:        m.Tags,
		Status:      status,
		IsLoaded:    isLoaded,
		ScannedAt:   scannedAt,
	}

	if st, ok := statuses[m.ID]; ok && st.Port > 0 {
		dto.Port = st.Port
	}

	// Auto-detect recommended backend type from model path + capabilities
	modelPath := m.Path
	if len(m.ShardFiles) > 0 {
		modelPath = m.ShardFiles[0]
	}
	// Delegate to the backend registry's resolution chain so the rules
	// (CapabilityHint → FormatAutoDetect → DefaultForGGUF) drive the
	// recommendation. Falls back to llamacpp when no rule matches.
	var capHint *backend.CapabilityHint
	if caps := s.modelMgr.GetModelCapabilities(m.ID); caps != nil {
		capHint = &backend.CapabilityHint{
			TTS:             caps.TTS,
			ASR:             caps.ASR,
			ImageGeneration: caps.ImageGeneration,
		}
	}
	if plugin, _, err := backend.Default().Resolve(modelPath, "", capHint); err == nil && plugin != nil {
		dto.PluginID = string(plugin.ID())
	} else {
		dto.PluginID = string(backend.IDLlamaCpp)
	}

	if m.Metadata != nil {
		dto.Metadata = map[string]interface{}{
			// Basic
			"architecture":       m.Metadata.Architecture,
			"quantization":       m.Metadata.GetQuantizationString(),
			"parameters":         m.Metadata.Parameters,
			"fileTypeDescriptor": m.Metadata.FileTypeDescriptor,
			"url":                m.Metadata.URL,
			"author":             m.Metadata.Author,
			"license":            m.Metadata.License,

			// Model structure
			"contextLength":     m.Metadata.ContextLength,
			"embeddingLength":   m.Metadata.EmbeddingLength,
			"feedForwardLength": m.Metadata.FeedForwardLength,
			"blockCount":        m.Metadata.BlockCount,
			"headCount":         m.Metadata.HeadCount,
			"headCountKV":       m.Metadata.HeadCountKV,
			"layerNormRmsEps":   m.Metadata.LayerNormRMS_EPS,

			// Tokenizer
			"tokenCount":     m.Metadata.TokenCount,
			"tokenizerModel": m.Metadata.TokenizerModel,
			"bosTokenId":     m.Metadata.BosTokenID,
			"eosTokenId":     m.Metadata.EosTokenID,
			"padTokenId":     m.Metadata.PadTokenID,
			"uncTokenId":     m.Metadata.UncTokenID,

			// RoPE
			"ropeDim":       m.Metadata.RopeDim,
			"ropeFreqBase":  m.Metadata.RopeFreqBase,
			"ropeFreqScale": m.Metadata.RopeFreqScale,

			// File info
			"bitsPerWeight": m.Metadata.BitsPerWeight,
			"fileSize":      m.Metadata.FileSize,
			"modelSize":     m.Metadata.ModelSize,
			"poolingType":   m.Metadata.PoolingType,
			"littleEndian":  m.Metadata.LittleEndian,
			"chatTemplate":  m.Metadata.ChatTemplate,
		}
	}

	return dto
}

func getArchitecture(modelPath string) string {
	parser, err := gguf.NewParser(modelPath)
	if err != nil {
		return ""
	}
	defer func() { _ = parser.Close() }()
	return parser.GetArchitecture()
}
