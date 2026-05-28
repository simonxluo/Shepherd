// Package storage provides API handlers for storage configuration
package storage

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/types"
	"github.com/simonxluo/Shepherd/internal/handler"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
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

// GetStorageConfig returns current storage configuration.
// @Summary      Get storage config
// @Description  Returns the current storage configuration and statistics
// @Tags         Storage
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/config/storage [get]
func (h *Handler) GetStorageConfig(c *gin.Context) {
	cfg := h.configManager.Get()
	handler.Success(c, gin.H{
		"config": cfg.Storage,
		"stats":  h.getStats(),
	})
}

// UpdateStorageConfig updates storage configuration.
// @Summary      Update storage config
// @Description  Updates the storage configuration (requires server restart to take effect)
// @Tags         Storage
// @Accept       json
// @Produce      json
// @Param        request body object true "Storage configuration"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/config/storage [put]
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

// GetStats returns storage statistics.
// @Summary      Get storage stats
// @Description  Returns storage statistics including type-specific metrics
// @Tags         Storage
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/config/storage/stats [get]
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

// GetConversations returns all conversations with pagination.
// @Summary      List conversations
// @Description  Returns all conversations with pagination support
// @Tags         Conversations
// @Produce      json
// @Param        limit query int false "Maximum number of results (default 100)"
// @Param        offset query int false "Offset for pagination (default 0)"
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/conversations [get]
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

// GetConversation retrieves a specific conversation with its messages.
// @Summary      Get conversation
// @Description  Returns a specific conversation with all its messages
// @Tags         Conversations
// @Produce      json
// @Param        id path string true "Conversation ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/conversations/{id} [get]
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

// DeleteConversation deletes a conversation and its messages.
// @Summary      Delete conversation
// @Description  Deletes a conversation and all associated messages
// @Tags         Conversations
// @Produce      json
// @Param        id path string true "Conversation ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/conversations/{id} [delete]
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

// CreateConversationRequest create conversation request body
type CreateConversationRequest struct {
	Model        string `json:"model" binding:"required"`
	Title        string `json:"title"`
	SystemPrompt string `json:"systemPrompt"`
}

// UpdateConversationRequest update conversation request body
type UpdateConversationRequest struct {
	Title        string `json:"title"`
	SystemPrompt string `json:"systemPrompt"`
}

// CreateMessageRequest create message request body
type CreateMessageRequest struct {
	Role       string                 `json:"role" binding:"required,oneof=system user assistant"`
	Content    string                 `json:"content" binding:"required"`
	TokenCount int                    `json:"tokenCount"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// CreateConversation creates a new conversation.
// @Summary      Create conversation
// @Description  Creates a new conversation with a model, title, and optional system prompt
// @Tags         Conversations
// @Accept       json
// @Produce      json
// @Param        request body CreateConversationRequest true "Conversation creation request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/conversations [post]
func (h *Handler) CreateConversation(c *gin.Context) {
	if h.storageMgr == nil {
		handler.Error(c, types.ErrInternalError, "Storage not initialized")
		return
	}

	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	conv := &storage.Conversation{
		Model:        req.Model,
		Title:        req.Title,
		SystemPrompt: req.SystemPrompt,
	}

	store := h.storageMgr.GetStore()
	if err := store.CreateConversation(c.Request.Context(), conv); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to create conversation", err.Error())
		return
	}

	handler.Success(c, gin.H{"conversation": conv})
}

// UpdateConversation updates an existing conversation.
// @Summary      Update conversation
// @Description  Updates the title or system prompt of an existing conversation
// @Tags         Conversations
// @Accept       json
// @Produce      json
// @Param        id path string true "Conversation ID"
// @Param        request body UpdateConversationRequest true "Conversation update request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/conversations/{id} [put]
func (h *Handler) UpdateConversation(c *gin.Context) {
	if h.storageMgr == nil {
		handler.Error(c, types.ErrInternalError, "Storage not initialized")
		return
	}

	id := c.Param("id")
	if id == "" {
		handler.BadRequest(c, "Conversation ID is required")
		return
	}

	var req UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
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

	if req.Title != "" {
		conv.Title = req.Title
	}
	if req.SystemPrompt != "" {
		conv.SystemPrompt = req.SystemPrompt
	}
	conv.UpdatedAt = time.Now()

	if err := store.UpdateConversation(c.Request.Context(), conv); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to update conversation", err.Error())
		return
	}

	handler.Success(c, gin.H{"conversation": conv})
}

// CreateMessage adds a message to a conversation.
// @Summary      Create message
// @Description  Adds a new message to a conversation
// @Tags         Conversations
// @Accept       json
// @Produce      json
// @Param        id path string true "Conversation ID"
// @Param        request body CreateMessageRequest true "Message creation request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/conversations/{id}/messages [post]
func (h *Handler) CreateMessage(c *gin.Context) {
	if h.storageMgr == nil {
		handler.Error(c, types.ErrInternalError, "Storage not initialized")
		return
	}

	convID := c.Param("id")
	if convID == "" {
		handler.BadRequest(c, "Conversation ID is required")
		return
	}

	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	msg := &storage.Message{
		ConversationID: convID,
		Role:           req.Role,
		Content:        req.Content,
		TokenCount:     req.TokenCount,
		Metadata:       req.Metadata,
	}

	store := h.storageMgr.GetStore()
	if err := store.CreateMessage(c.Request.Context(), msg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to create message", err.Error())
		return
	}

	handler.Success(c, gin.H{"message": msg})
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
