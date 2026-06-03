package server

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/comm/types"
	api "github.com/simonxluo/Shepherd/internal/handler"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

func applyLaunchProfileToLoadRequest(req *model.LoadRequest, profile *storage.LaunchProfile) {
	if profile == nil {
		return
	}
	if req.PluginID == "" {
		req.PluginID = profile.PluginID
	}
	if req.ExtraParams == "" {
		req.ExtraParams = profile.ExtraArgs
	}
	if len(req.EnvVars) == 0 {
		req.EnvVars = append([]string(nil), profile.Env...)
	}
	for key, value := range profile.Params {
		switch key {
		case "ctxSize":
			if req.CtxSize == 0 {
				req.CtxSize = intValue(value)
			}
		case "batchSize":
			if req.BatchSize == 0 {
				req.BatchSize = intValue(value)
			}
		case "threads":
			if req.Threads == 0 {
				req.Threads = intValue(value)
			}
		case "gpuLayers":
			if req.GPULayers == 0 {
				req.GPULayers = intValue(value)
			}
		case "temperature":
			if req.Temperature == 0 {
				req.Temperature = floatValue(value)
			}
		case "topP":
			if req.TopP == 0 {
				req.TopP = floatValue(value)
			}
		case "topK":
			if req.TopK == 0 {
				req.TopK = intValue(value)
			}
		case "repeatPenalty":
			if req.RepeatPenalty == 0 {
				req.RepeatPenalty = floatValue(value)
			}
		case "seed":
			if req.Seed == 0 {
				req.Seed = intValue(value)
			}
		case "nPredict":
			if req.NPredict == 0 {
				req.NPredict = intValue(value)
			}
		case "parallelSlots":
			if req.ParallelSlots == 0 {
				req.ParallelSlots = intValue(value)
			}
		case "timeout":
			if req.Timeout == 0 {
				req.Timeout = intValue(value)
			}
		case "mmprojPath":
			if req.MmprojPath == "" {
				req.MmprojPath, _ = value.(string)
			}
		}
	}
}

func intValue(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

func floatValue(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	default:
		return 0
	}
}

// HandleListLaunchProfiles lists reusable launch profiles.
func (s *Server) HandleListLaunchProfiles(c *gin.Context) {
	profiles, err := s.storageMgr.GetStore().ListLaunchProfiles(context.Background(), c.Query("backendType"), c.Query("modelScope"))
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "Failed to list launch profiles", err.Error())
		return
	}
	api.Success(c, gin.H{"profiles": profiles, "count": len(profiles)})
}

// HandleCreateLaunchProfile creates a reusable launch profile.
func (s *Server) HandleCreateLaunchProfile(c *gin.Context) {
	var profile storage.LaunchProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		api.BadRequest(c, "Invalid request body")
		return
	}
	if profile.Name == "" {
		api.BadRequest(c, "Profile name is required")
		return
	}
	if profile.PluginID == "" {
		profile.PluginID = string(backend.IDLlamaCpp)
	}
	if profile.Params == nil {
		profile.Params = map[string]interface{}{}
	}
	if plugin, ok := backend.Default().Get(backend.ID(profile.PluginID)); ok {
		validation := plugin.ValidateParams(backend.RawParams(profile.Params))
		if !validation.Valid {
			api.ErrorWithDetails(c, types.ErrInvalidRequest, "Invalid profile params", strings.Join(validation.Errors, "; "))
			return
		}
	}
	if err := s.storageMgr.GetStore().CreateLaunchProfile(context.Background(), &profile); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "Failed to create launch profile", err.Error())
		return
	}
	api.Success(c, gin.H{"profile": profile})
}

// HandleGetLaunchProfile returns one launch profile.
func (s *Server) HandleGetLaunchProfile(c *gin.Context) {
	profile, err := s.storageMgr.GetStore().GetLaunchProfile(context.Background(), c.Param("id"))
	if err != nil {
		api.NotFound(c, "Launch profile")
		return
	}
	api.Success(c, gin.H{"profile": profile})
}

// HandleUpdateLaunchProfile updates a launch profile.
func (s *Server) HandleUpdateLaunchProfile(c *gin.Context) {
	var profile storage.LaunchProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		api.BadRequest(c, "Invalid request body")
		return
	}
	profile.ID = c.Param("id")
	if profile.Name == "" {
		api.BadRequest(c, "Profile name is required")
		return
	}
	if profile.PluginID == "" {
		profile.PluginID = string(backend.IDLlamaCpp)
	}
	if profile.Params == nil {
		profile.Params = map[string]interface{}{}
	}
	if plugin, ok := backend.Default().Get(backend.ID(profile.PluginID)); ok {
		validation := plugin.ValidateParams(backend.RawParams(profile.Params))
		if !validation.Valid {
			api.ErrorWithDetails(c, types.ErrInvalidRequest, "Invalid profile params", strings.Join(validation.Errors, "; "))
			return
		}
	}
	if err := s.storageMgr.GetStore().UpdateLaunchProfile(context.Background(), &profile); err != nil {
		api.NotFound(c, "Launch profile")
		return
	}
	api.Success(c, gin.H{"profile": profile})
}

// HandleDeleteLaunchProfile deletes a launch profile.
func (s *Server) HandleDeleteLaunchProfile(c *gin.Context) {
	if err := s.storageMgr.GetStore().DeleteLaunchProfile(context.Background(), c.Param("id")); err != nil {
		api.NotFound(c, "Launch profile")
		return
	}
	api.Success(c, gin.H{"message": "Launch profile deleted"})
}
