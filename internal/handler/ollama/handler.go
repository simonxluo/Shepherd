// Package ollama provides Ollama API compatibility layer
package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
)

// Handler handles Ollama API requests
type Handler struct {
	modelMgr *model.Manager
	client   *http.Client
}

// NewHandler creates a new Ollama API handler
func NewHandler(modelMgr *model.Manager) *Handler {
	return &Handler{
		modelMgr: modelMgr,
		client:   &http.Client{},
	}
}

// ChatRequest represents an Ollama chat request
type ChatRequest struct {
	Model    string            `json:"model"`
	Messages []ChatMessage     `json:"messages"`
	Stream   bool              `json:"stream,omitempty"`
	Options  *GenerationParams `json:"options,omitempty"`
	Format   string            `json:"format,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerationParams represents generation parameters
type GenerationParams struct {
	Temperature   float64  `json:"temperature,omitempty"`
	TopP          float64  `json:"top_p,omitempty"`
	TopK          int      `json:"top_k,omitempty"`
	NumPredict    int      `json:"num_predict,omitempty"`
	RepeatPenalty float64  `json:"repeat_penalty,omitempty"`
	Stop          []string `json:"stop,omitempty"`
}

// ChatResponse represents an Ollama chat response
type ChatResponse struct {
	Model     string      `json:"model"`
	CreatedAt string      `json:"created_at,omitempty"`
	Message   ChatMessage `json:"message,omitempty"`
	Done      bool        `json:"done,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// HandleChat handles Ollama chat completion requests
func (h *Handler) HandleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate request
	if req.Model == "" {
		h.sendError(c, http.StatusBadRequest, "model is required")
		return
	}

	if len(req.Messages) == 0 {
		h.sendError(c, http.StatusBadRequest, "messages array is empty")
		return
	}

	// Find the actual model ID
	actualModelID, err := h.findModel(req.Model)
	if err != nil {
		h.sendError(c, http.StatusNotFound, err.Error())
		return
	}

	// Get model port
	port, err := h.getModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to OpenAI format and forward
	h.forwardToOpenAI(c, actualModelID, port, req)
}

// HandleTags handles Ollama tags requests
func (h *Handler) HandleTags(c *gin.Context) {
	// Return empty tags list for now
	c.JSON(http.StatusOK, map[string]interface{}{
		"models": []interface{}{},
	})
}

// findModel finds a model by name or ID
func (h *Handler) findModel(modelName string) (string, error) {
	statuses := h.modelMgr.ListStatus()
	models := h.modelMgr.ListModels()

	// First try exact match with ID
	if status, exists := statuses[modelName]; exists && status.State == model.StateLoaded {
		return modelName, nil
	}

	// Try to find by name/alias
	for _, m := range models {
		status, exists := statuses[m.ID]
		if !exists || status.State != model.StateLoaded {
			continue
		}

		if m.Alias == modelName || m.Name == modelName ||
			strings.EqualFold(m.Name, modelName) ||
			strings.Contains(strings.ToLower(m.ID), strings.ToLower(modelName)) {
			return m.ID, nil
		}
	}

	return "", fmt.Errorf("model not found: %s", modelName)
}

// getModelPort returns the port for a loaded model
func (h *Handler) getModelPort(modelID string) (int, error) {
	status, exists := h.modelMgr.GetStatus(modelID)
	if !exists {
		return 0, fmt.Errorf("model not loaded: %s", modelID)
	}

	if status.State != model.StateLoaded {
		return 0, fmt.Errorf("model not in loaded state: %s", modelID)
	}

	if status.Port == 0 {
		return 0, fmt.Errorf("model port not available: %s", modelID)
	}

	return status.Port, nil
}

// forwardToOpenAI converts Ollama request to OpenAI format and forwards
func (h *Handler) forwardToOpenAI(c *gin.Context, modelID string, port int, ollamaReq ChatRequest) {
	// Convert to OpenAI format
	messages := make([]map[string]interface{}, len(ollamaReq.Messages))
	for i, msg := range ollamaReq.Messages {
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	openaiReq := map[string]interface{}{
		"model":    modelID,
		"messages": messages,
		"stream":   ollamaReq.Stream,
	}

	if ollamaReq.Options != nil {
		if ollamaReq.Options.Temperature > 0 {
			openaiReq["temperature"] = ollamaReq.Options.Temperature
		}
		if ollamaReq.Options.TopP > 0 {
			openaiReq["top_p"] = ollamaReq.Options.TopP
		}
		if ollamaReq.Options.TopK > 0 {
			openaiReq["top_k"] = ollamaReq.Options.TopK
		}
	}

	// Marshal request body
	body, err := json.Marshal(openaiReq)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewReader(body))
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		h.sendError(c, http.StatusBadGateway, err.Error())
		logger.Errorf("转发请求到 llama.cpp 失败: %v", err)
		return
	}
	defer utils.CloseQuietly(resp.Body)

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Forward response
	c.Header("Content-Type", "application/json")
	c.Status(resp.StatusCode)
	utils.WriteQuietly(c.Writer, respBody)
}

// sendError sends an error response
func (h *Handler) sendError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, map[string]interface{}{
		"error": message,
	})
}
