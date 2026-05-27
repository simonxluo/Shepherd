package server

import (
	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/types"
	api "github.com/simonxluo/Shepherd/internal/handler"
)

// HandleListRuntimeInstances lists runtime instances.
func (s *Server) HandleListRuntimeInstances(c *gin.Context) {
	instances := s.modelMgr.ListRuntimeInstances()
	api.Success(c, gin.H{"instances": instances, "count": len(instances)})
}

// HandleGetRuntimeInstance returns one runtime instance.
func (s *Server) HandleGetRuntimeInstance(c *gin.Context) {
	instance, ok := s.modelMgr.GetRuntimeInstance(c.Param("id"))
	if !ok {
		api.NotFound(c, "Runtime instance")
		return
	}
	api.Success(c, gin.H{"instance": instance})
}

// HandleStopRuntimeInstance stops a runtime instance.
func (s *Server) HandleStopRuntimeInstance(c *gin.Context) {
	instance, ok := s.modelMgr.GetRuntimeInstance(c.Param("id"))
	if !ok {
		api.NotFound(c, "Runtime instance")
		return
	}
	if err := s.modelMgr.Unload(instance.ModelID); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "Failed to stop runtime instance", err.Error())
		return
	}
	api.Success(c, gin.H{"message": "Runtime instance stopped", "instanceId": instance.InstanceID})
}
