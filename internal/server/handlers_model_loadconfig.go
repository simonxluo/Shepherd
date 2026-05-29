package server

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/comm/types"
	api "github.com/simonxluo/Shepherd/internal/handler"
)

// HandleGetModelLoadConfig returns the saved load configuration for a model.
// @Summary      Get model load config
// @Description  Returns the saved default load configuration for a specific model on this node
// @Tags         Models
// @Produce      json
// @Param        id path string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /api/models/{id}/load-config [get]
func (s *Server) HandleGetModelLoadConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "model ID is required")
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
		api.BadRequest(c, "model ID is required")
		return
	}

	var req struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "invalid request format", err.Error())
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
		api.ErrorWithDetails(c, types.ErrInternalError, "failed to save model load config", err.Error())
		return
	}

	logger.Infof("model load config saved: modelId=%s, nodeId=%s", id, nodeID)

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
		api.BadRequest(c, "model ID is required")
		return
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	if err := s.storageMgr.GetStore().DeleteModelLoadConfig(ctx, nodeID, id); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "failed to delete model load config", err.Error())
		return
	}

	logger.Infof("model load config deleted: modelId=%s, nodeId=%s", id, nodeID)

	api.SuccessWithMessage(c, "model load config deleted")
}

// HandleListModelLoadConfigs returns all load configs (default + named) for a model.
// @Summary      List model load configs
// @Description  Returns all load configuration presets for a specific model
// @Tags         Models
// @Produce      json
// @Param        id path string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/load-configs [get]
func (s *Server) HandleListModelLoadConfigs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "model ID is required")
		return
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	configs, err := s.storageMgr.GetStore().ListModelLoadConfigs(ctx, nodeID, id)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "failed to list model load configs", err.Error())
		return
	}

	api.Success(c, gin.H{
		"modelId": id,
		"configs": configs,
	})
}

// HandleSaveNamedModelLoadConfig saves a named load config preset.
// @Summary      Save named load config
// @Description  Saves a named load configuration preset for a specific model
// @Tags         Models
// @Accept       json
// @Produce      json
// @Param        id path string true "Model ID"
// @Param        name path string true "Config preset name"
// @Param        request body object true "Load config to save"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/load-configs/{name} [put]
func (s *Server) HandleSaveNamedModelLoadConfig(c *gin.Context) {
	id := c.Param("id")
	name := c.Param("name")
	if id == "" {
		api.BadRequest(c, "model ID is required")
		return
	}
	if name == "" {
		api.BadRequest(c, "config name is required")
		return
	}

	var req struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorWithDetails(c, types.ErrInvalidRequest, "invalid request format", err.Error())
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
		api.ErrorWithDetails(c, types.ErrInternalError, "failed to save named config", err.Error())
		return
	}

	logger.Infof("named config saved: modelId=%s, name=%s, nodeId=%s", id, name, nodeID)

	api.Success(c, gin.H{
		"modelId": id,
		"name":    name,
		"config":  req.Config,
	})
}

// HandleDeleteNamedModelLoadConfig deletes a named load config preset.
// @Summary      Delete named load config
// @Description  Deletes a named load configuration preset for a specific model
// @Tags         Models
// @Produce      json
// @Param        id path string true "Model ID"
// @Param        name path string true "Config preset name"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/{id}/load-configs/{name} [delete]
func (s *Server) HandleDeleteNamedModelLoadConfig(c *gin.Context) {
	id := c.Param("id")
	name := c.Param("name")
	if id == "" {
		api.BadRequest(c, "model ID is required")
		return
	}
	if name == "" {
		api.BadRequest(c, "config name is required")
		return
	}

	nodeID := s.getNodeID()
	ctx := context.Background()

	if err := s.storageMgr.GetStore().DeleteNamedModelLoadConfig(ctx, nodeID, id, name); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "failed to delete named config", err.Error())
		return
	}

	logger.Infof("named config deleted: modelId=%s, name=%s, nodeId=%s", id, name, nodeID)

	api.SuccessWithMessage(c, "named config deleted")
}
