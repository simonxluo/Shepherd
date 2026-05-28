// Package tts provides HTTP handlers for TTS generation history management.
package tts

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
)

// Handler handles TTS history API requests
type Handler struct {
	storageMgr *storage.Manager
	dataDir    string
}

// NewHandler creates a new TTS handler
func NewHandler(storageMgr *storage.Manager, dataDir string) *Handler {
	_ = os.MkdirAll(dataDir, 0755)
	return &Handler{storageMgr: storageMgr, dataDir: dataDir}
}

// createHistoryRequest is the JSON metadata for creating a history item
type createHistoryRequest struct {
	Model     string                 `json:"model"`
	InputText string                 `json:"inputText"`
	Format    string                 `json:"format"`
	Duration  float64                `json:"duration"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

// favouriteRequest is the request body for toggling favourite
type favouriteRequest struct {
	Favourite bool `json:"favourite"`
}

// ListHistory handles GET /api/tts/history
func (h *Handler) ListHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	var favouriteOnly *bool
	if fav := c.Query("favourite"); fav != "" {
		val := fav == "true" || fav == "1"
		favouriteOnly = &val
	}

	items, err := h.storageMgr.GetStore().ListTTSHistory(c.Request.Context(), limit, offset, favouriteOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if items == nil {
		items = []*storage.TTSHistoryItem{}
	}

	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

// GetHistory handles GET /api/tts/history/:id
func (h *Handler) GetHistory(c *gin.Context) {
	id := c.Param("id")

	item, err := h.storageMgr.GetStore().GetTTSHistory(c.Request.Context(), id)
	if err != nil {
		if err == storage.ErrTTSHistoryNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

// CreateHistory handles POST /api/tts/history (multipart: audio file + metadata JSON)
func (h *Handler) CreateHistory(c *gin.Context) {
	// Parse metadata from form field
	metadataStr := c.PostForm("metadata")
	if metadataStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metadata field is required"})
		return
	}

	var req createHistoryRequest
	if err := json.Unmarshal([]byte(metadataStr), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid metadata JSON: %v", err)})
		return
	}

	if req.Model == "" || req.InputText == "" || req.Format == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model, inputText, and format are required"})
		return
	}

	// Get uploaded audio file
	file, _, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("audio file is required: %v", err)})
		return
	}
	defer file.Close()

	// Generate ID and audio path upfront so we can save the file before creating the DB record
	id := fmt.Sprintf("tts-%d", time.Now().UnixNano())
	filename := fmt.Sprintf("%s.%s", id, req.Format)
	filePath := filepath.Join(h.dataDir, filename)

	// Save audio file first (no DB record yet, so failure is cheap)
	outFile, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save audio file: %v", err)})
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		_ = os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to write audio file: %v", err)})
		return
	}

	// Create history item with everything set in one shot
	item := &storage.TTSHistoryItem{
		ID:        id,
		Model:     req.Model,
		InputText: req.InputText,
		AudioPath: filename,
		Format:    req.Format,
		Duration:  req.Duration,
		Params:    req.Params,
	}

	if err := h.storageMgr.GetStore().CreateTTSHistory(c.Request.Context(), item); err != nil {
		_ = os.Remove(filePath) // rollback file on DB failure
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create history: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// ToggleFavourite handles PUT /api/tts/history/:id/favourite
func (h *Handler) ToggleFavourite(c *gin.Context) {
	id := c.Param("id")

	var req favouriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	err := h.storageMgr.GetStore().UpdateTTSHistoryFavourite(c.Request.Context(), id, req.Favourite)
	if err != nil {
		if err == storage.ErrTTSHistoryNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "favourite": req.Favourite})
}

// DeleteHistory handles DELETE /api/tts/history/:id
func (h *Handler) DeleteHistory(c *gin.Context) {
	id := c.Param("id")

	// Get item first to find audio file path
	item, err := h.storageMgr.GetStore().GetTTSHistory(c.Request.Context(), id)
	if err != nil {
		if err == storage.ErrTTSHistoryNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete from store
	if err := h.storageMgr.GetStore().DeleteTTSHistory(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove audio file (best effort)
	if item.AudioPath != "" {
		_ = os.Remove(filepath.Join(h.dataDir, item.AudioPath))
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

// ServeAudio handles GET /api/tts/audio/:id
func (h *Handler) ServeAudio(c *gin.Context) {
	id := c.Param("id")

	item, err := h.storageMgr.GetStore().GetTTSHistory(c.Request.Context(), id)
	if err != nil {
		if err == storage.ErrTTSHistoryNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filePath := filepath.Join(h.dataDir, item.AudioPath)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "audio file not found"})
		return
	}

	// Determine content type from format
	contentType := mime.TypeByExtension("." + item.Format)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", item.AudioPath))
	c.File(filePath)
}
