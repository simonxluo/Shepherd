package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
	api "github.com/shepherd-project/shepherd/Shepherd/internal/handler"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/gguf"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model/backend"
)

// loadedModelInfo 用于 /api/models/loaded 响应的已加载模型信息
type loadedModelInfo struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Alias        string                `json:"alias,omitempty"`
	State        string                `json:"state"`
	ProcessID    string                `json:"processId"`
	Port         int                   `json:"port"`
	CtxSize      int                   `json:"ctxSize"`
	BackendType  string                `json:"backendType,omitempty"`
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
				ID:          id,
				Name:        st.Name,
				State:       st.State.String(),
				ProcessID:   st.ProcessID,
				Port:        st.Port,
				CtxSize:     st.CtxSize,
				BackendType: st.BackendType,
			}
			if !st.LoadedAt.IsZero() {
				info.LoadedAt = st.LoadedAt.Format(time.RFC3339)
			}

			// 获取模型别名和 capabilities
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

	// Validate draft model if SpecDecoding is set with draft type
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

	// 不再强制设置 vllm_omni 后端，交给 Resolve 的能力感知路由处理：
	// - 有 vllm_omni 配置时，能力感知路由会优先选择它
	// - 无 vllm_omni 配置时，GGUF 模型会 fallback 到 llama.cpp
	// 前端显式指定 BackendType 时仍然会被尊重

	result, err := s.modelMgr.LoadAsync(&req)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "模型加载失败", err.Error())
		return
	}

	respData := gin.H{
		"id":     id,
		"status": "loading",
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

// HandleGetModelLoadConfig returns the saved load configuration for a model.
// @Summary      Get model load config
// @Description  Returns the saved load configuration for a specific model on this node
// @Tags         Models
// @Produce      json
// @Param        id path string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Router       /api/models/{id}/load-config [get]
func (s *Server) HandleGetModelLoadConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	config, err := s.storageMgr.GetStore().GetModelLoadConfig(ctx, nodeID, id)
	if err != nil {
		api.Success(c, gin.H{
			"modelId": id,
			"config":  nil,
		})
		return
	}

	api.Success(c, gin.H{
		"modelId":   id,
		"nodeId":    nodeID,
		"config":    config.Config,
		"createdAt": config.CreatedAt.Format(time.RFC3339),
		"updatedAt": config.UpdatedAt.Format(time.RFC3339),
	})
}

// HandleSaveModelLoadConfig saves the load configuration for a model.
// @Summary      Save model load config
// @Description  Saves load configuration for a specific model on this node
// @Tags         Models
// @Accept       json
// @Produce      json
// @Param        id path string true "Model ID"
// @Param        request body object true "Load config to save"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/load-config [put]
func (s *Server) HandleSaveModelLoadConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	var req struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求格式", err.Error())
		return
	}

	m, exists := s.modelMgr.GetModel(id)
	modelName := ""
	if exists {
		modelName = m.Name
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	loadConfig := &storage.ModelLoadConfig{
		NodeID:    nodeID,
		ModelID:   id,
		ModelName: modelName,
		Config:    req.Config,
	}

	if err := s.storageMgr.GetStore().SaveModelLoadConfig(ctx, loadConfig); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "保存模型加载配置失败", err.Error())
		return
	}

	logger.Infof("模型加载配置已保存: modelId=%s, nodeId=%s", id, nodeID)

	api.Success(c, gin.H{
		"modelId": id,
		"nodeId":  nodeID,
		"config":  req.Config,
	})
}

// HandleDeleteModelLoadConfig deletes the saved load configuration for a model.
// @Summary      Delete model load config
// @Description  Deletes the saved load configuration for a specific model on this node
// @Tags         Models
// @Produce      json
// @Param        id path string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/load-config [delete]
func (s *Server) HandleDeleteModelLoadConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	if err := s.storageMgr.GetStore().DeleteModelLoadConfig(ctx, nodeID, id); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "删除模型加载配置失败", err.Error())
		return
	}

	logger.Infof("模型加载配置已删除: modelId=%s, nodeId=%s", id, nodeID)

	api.SuccessWithMessage(c, "模型加载配置已删除")
}

// HandleListModelLoadConfigs returns all load configs (default + named) for a model.
func (s *Server) HandleListModelLoadConfigs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	configs, err := s.storageMgr.GetStore().ListModelLoadConfigs(ctx, nodeID, id)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "获取模型加载配置列表失败", err.Error())
		return
	}

	api.Success(c, gin.H{
		"modelId": id,
		"configs": configs,
	})
}

// HandleSaveNamedModelLoadConfig saves a named load config preset.
func (s *Server) HandleSaveNamedModelLoadConfig(c *gin.Context) {
	id := c.Param("id")
	name := c.Param("name")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}
	if name == "" {
		api.BadRequest(c, "配置名称不能为空")
		return
	}

	var req struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "无效的请求格式", err.Error())
		return
	}

	m, exists := s.modelMgr.GetModel(id)
	modelName := ""
	if exists {
		modelName = m.Name
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	loadConfig := &storage.ModelLoadConfig{
		NodeID:    nodeID,
		ModelID:   id,
		ModelName: modelName,
		Name:      name,
		Config:    req.Config,
	}

	if err := s.storageMgr.GetStore().SaveNamedModelLoadConfig(ctx, loadConfig); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "保存命名配置失败", err.Error())
		return
	}

	logger.Infof("命名配置已保存: modelId=%s, name=%s, nodeId=%s", id, name, nodeID)

	api.Success(c, gin.H{
		"modelId": id,
		"name":    name,
		"config":  req.Config,
	})
}

// HandleDeleteNamedModelLoadConfig deletes a named load config preset.
func (s *Server) HandleDeleteNamedModelLoadConfig(c *gin.Context) {
	id := c.Param("id")
	name := c.Param("name")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}
	if name == "" {
		api.BadRequest(c, "配置名称不能为空")
		return
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	if err := s.storageMgr.GetStore().DeleteNamedModelLoadConfig(ctx, nodeID, id, name); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "删除命名配置失败", err.Error())
		return
	}

	logger.Infof("命名配置已删除: modelId=%s, name=%s, nodeId=%s", id, name, nodeID)

	api.SuccessWithMessage(c, "命名配置已删除")
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
	// 根据能力 + 格式 + 后端可用性决定推荐后端
	caps := s.modelMgr.GetModelCapabilities(m.ID)
	vllmOmniConfigured := s.config != nil && s.config.ServerCfg != nil &&
		s.config.ServerCfg.Backends.VLLMOmni != nil &&
		s.config.ServerCfg.Backends.VLLMOmni.Enabled
	if caps != nil && (caps.TTS || caps.ASR) && vllmOmniConfigured {
		dto.BackendType = "vllm_omni"
	} else if caps != nil && (caps.TTS || caps.ASR) && !vllmOmniConfigured {
		// vllm_omni 未配置，GGUF TTS/ASR 模型 fallback 到 llama.cpp
		dto.BackendType = "llamacpp"
	} else if backend.IsSafeTensorsModel(modelPath) || filepath.Ext(modelPath) == "" {
		dto.BackendType = "vllm"
	} else {
		dto.BackendType = "llamacpp"
	}

	if m.Metadata != nil {
		dto.Metadata = map[string]interface{}{
			"architecture":       m.Metadata.Architecture,
			"quantization":       m.Metadata.GetQuantizationString(),
			"contextLength":      m.Metadata.ContextLength,
			"parameters":         m.Metadata.Parameters,
			"fileTypeDescriptor": m.Metadata.FileTypeDescriptor,
			"url":                m.Metadata.URL,
			"author":             m.Metadata.Author,
			"embeddingLength":    m.Metadata.EmbeddingLength,
			"layerCount":         m.Metadata.BlockSize,
			"headCount":          m.Metadata.HeadCount,
			"license":            m.Metadata.License,
			"bitsPerWeight":      m.Metadata.BitsPerWeight,
			"fileSize":           m.Metadata.FileSize,
			"modelSize":          m.Metadata.ModelSize,
			"headCountKV":        m.Metadata.HeadCountKV,
			"tokenCount":         m.Metadata.TokenCount,
			"poolingType":        m.Metadata.PoolingType,
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
