package server

import (
	"context"
	"fmt"
	"net/http"
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

	if req.DraftModelID != "" {
		if req.DraftModelID == id {
			api.BadRequest(c, "Draft模型不能与主模型相同")
			return
		}
		draftModel, exists := s.modelMgr.GetModel(req.DraftModelID)
		if !exists {
			api.BadRequest(c, fmt.Sprintf("Draft模型未找到: %s", req.DraftModelID))
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
		req.DraftModelPath = draftPath
		logger.Infof("draft model resolved: modelId=%s, draftModelId=%s, draftPath=%s, arch=%s", id, req.DraftModelID, draftPath, draftArch)
	}

	if req.CtxSize == 0 {
		req.CtxSize = 512
	}
	if req.BatchSize == 0 {
		req.BatchSize = 512
	}
	if req.Threads == 0 {
		req.Threads = 4
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

	// 根据模型能力自动设置后端类型（如果前端未指定）
	if req.BackendType == "" {
		caps := s.modelMgr.GetModelCapabilities(id)
		mdl, _ := s.modelMgr.GetModel(id)
		if caps != nil && mdl != nil {
			mPath := mdl.Path
			if len(mdl.ShardFiles) > 0 {
				mPath = mdl.ShardFiles[0]
			}
			if !backend.IsGGUFModel(mPath) && (caps.TTS || caps.ASR) {
				req.BackendType = "vllm_omni"
				logger.Infof("根据模型能力自动设置后端类型: modelId=%s, backendType=vllm_omni", id)
			}
		}
	}

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

	requestID := c.GetString("requestId")
	if requestID == "" {
		requestID = "unknown"
	}
	c.JSON(http.StatusAccepted, types.NewSuccessResponse(respData, requestID))
}

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
	status := "unloaded"
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
		Status:      status,
		IsLoaded:    isLoaded,
		ScannedAt:   scannedAt,
	}

	// Auto-detect recommended backend type from model path + capabilities
	modelPath := m.Path
	if len(m.ShardFiles) > 0 {
		modelPath = m.ShardFiles[0]
	}
	if backend.IsGGUFModel(modelPath) {
		dto.BackendType = "llamacpp"
	} else if backend.IsSafeTensorsModel(modelPath) || filepath.Ext(modelPath) == "" {
		// safetensors 或目录格式模型，根据能力决定后端类型
		caps := s.modelMgr.GetModelCapabilities(m.ID)
		if caps != nil && (caps.TTS || caps.ASR) {
			dto.BackendType = "vllm_omni"
		} else {
			dto.BackendType = "vllm"
		}
	}

	if m.Metadata != nil {
		dto.Metadata = map[string]interface{}{
			"architecture":  m.Metadata.Architecture,
			"quantization":  m.Metadata.GetQuantizationString(),
			"contextLength": m.Metadata.ContextLength,
			"parameters":    m.Metadata.Parameters,
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
