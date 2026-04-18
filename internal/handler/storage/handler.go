// Package storage provides API handlers for storage configuration
package storage

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
)

// Handler handles storage API requests
type Handler struct {
	configManager *config.Manager
	storageMgr    *storage.Manager
}

// NewHandler creates a new storage handler
func NewHandler(configManager *config.Manager, storageMgr *storage.Manager) *Handler {
	return &Handler{
		configManager: configManager,
		storageMgr:    storageMgr,
	}
}

// GetStorageConfig returns current storage configuration
func (h *Handler) GetStorageConfig(c *gin.Context) {
	cfg := h.configManager.Get()
	handler.Success(c, gin.H{
		"config": cfg.Storage,
		"stats":  h.getStats(),
	})
}

// UpdateStorageConfig updates storage configuration
func (h *Handler) UpdateStorageConfig(c *gin.Context) {
	var req storage.StorageConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}

	// Validate storage type
	if req.Type != storage.StorageTypeMemory && req.Type != storage.StorageTypeSQLite {
		handler.BadRequest(c, "Storage type must be 'memory' or 'sqlite'")
		return
	}

	// Validate SQLite config if type is sqlite
	if req.Type == storage.StorageTypeSQLite && req.SQLite == nil {
		handler.BadRequest(c, "SQLite configuration is required when type is 'sqlite'")
		return
	}

	// Load current config
	cfg := h.configManager.Get()
	cfg.Storage = req

	// Save config
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save configuration", err.Error())
		return
	}

	// Note: Storage backend changes require server restart to take effect
	handler.Success(c, gin.H{
		"message": "Storage configuration updated. Restart the server for changes to take effect.",
		"config":  req,
	})
}

// GetStats returns storage statistics
func (h *Handler) GetStats(c *gin.Context) {
	handler.Success(c, h.getStats())
}

// getStats retrieves storage statistics
func (h *Handler) getStats() map[string]interface{} {
	if h.storageMgr == nil {
		return map[string]interface{}{
			"type":  "unknown",
			"error": "Storage manager not initialized",
		}
	}

	store := h.storageMgr.GetStore()

	// Type-specific stats
	switch s := store.(type) {
	case *storage.MemoryStore:
		return s.Stats()
	case *storage.SQLiteStore:
		stats, err := s.Stats()
		if err != nil {
			return map[string]interface{}{
				"type":  "sqlite",
				"error": err.Error(),
			}
		}
		return stats
	default:
		return map[string]interface{}{
			"type": "unknown",
		}
	}
}

// GetConversations returns all conversations (with pagination)
func (h *Handler) GetConversations(c *gin.Context) {
	if h.storageMgr == nil {
		handler.Error(c, types.ErrInternalError, "Storage not initialized")
		return
	}

	limit := 100
	offset := 0

	// Parse pagination parameters
	if l, ok := c.GetQuery("limit"); ok {
		if parsedLimit, err := parseQueryParam(l, 1, 1000); err == nil {
			limit = parsedLimit
		}
	}

	if o, ok := c.GetQuery("offset"); ok {
		if parsedOffset, err := parseQueryParam(o, 0, 1000000); err == nil {
			offset = parsedOffset
		}
	}

	store := h.storageMgr.GetStore()
	convs, err := store.ListConversations(c.Request.Context(), limit, offset)
	if err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to retrieve conversations", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"items":  convs,
		"count":  len(convs),
		"limit":  limit,
		"offset": offset,
	})
}

// GetConversation retrieves a specific conversation
func (h *Handler) GetConversation(c *gin.Context) {
	if h.storageMgr == nil {
		handler.Error(c, types.ErrInternalError, "Storage not initialized")
		return
	}

	id := c.Param("id")
	if id == "" {
		handler.BadRequest(c, "Conversation ID is required")
		return
	}

	store := h.storageMgr.GetStore()
	conv, err := store.GetConversation(c.Request.Context(), id)
	if err != nil {
		if err == storage.ErrConversationNotFound {
			handler.NotFound(c, "Conversation")
		} else {
			handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to retrieve conversation", err.Error())
		}
		return
	}

	// Get messages for this conversation
	messages, err := store.GetMessages(c.Request.Context(), id, 1000, 0)
	if err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to retrieve messages", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"conversation": conv,
		"messages":     messages,
	})
}

// DeleteConversation deletes a conversation and its messages
func (h *Handler) DeleteConversation(c *gin.Context) {
	if h.storageMgr == nil {
		handler.Error(c, types.ErrInternalError, "Storage not initialized")
		return
	}

	id := c.Param("id")
	if id == "" {
		handler.BadRequest(c, "Conversation ID is required")
		return
	}

	store := h.storageMgr.GetStore()
	if err := store.DeleteConversation(c.Request.Context(), id); err != nil {
		if err == storage.ErrConversationNotFound {
			handler.NotFound(c, "Conversation")
		} else {
			handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to delete conversation", err.Error())
		}
		return
	}

	handler.SuccessWithMessage(c, "Conversation deleted successfully")
}

// Helper function to parse query parameters with bounds
func parseQueryParam(s string, min, max int) (int, error) {
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, err
	}
	if val < min {
		return min, nil
	}
	if val > max {
		return max, nil
	}
	return val, nil
}
