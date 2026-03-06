// Package anthropic provides Anthropic API compatibility layer
package anthropic

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
)

// Handler handles Anthropic API requests
type Handler struct {
	modelMgr *model.Manager
	client   *http.Client
}

// NewHandler creates a new Anthropic API handler
func NewHandler(modelMgr *model.Manager) *Handler {
	return &Handler{
		modelMgr: modelMgr,
		client:   &http.Client{},
	}
}

// MessageRequest represents an Anthropic messages API request
type MessageRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	Messages  []Message     `json:"messages"`
	System    string        `json:"system,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
	TopP      float64       `json:"top_p,omitempty"`
	TopK      int           `json:"top_k,omitempty"`
	Stream    bool          `json:"stream,omitempty"`
	StopSequences []string `json:"stop_sequences,omitempty"`
}

// Message represents a message in Anthropic format
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MessageResponse represents an Anthropic messages API response
type MessageResponse struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Content      []ContentBlock `json:"content,omitempty"`
	StopReason   string        `json:"stop_reason,omitempty"`
	Model        string        `json:"model"`
	Usage        *Usage        `json:"usage,omitempty"`
	Error        *ErrorDetail  `json:"error,omitempty"`
}

// ContentBlock represents a content block
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Usage represents token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// HandleMessages handles Anthropic messages API requests
func (h *Handler) HandleMessages(c *gin.Context) {
	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Validate request
	if req.Model == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "model is required")
		return
	}

	if req.MaxTokens == 0 {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "max_tokens is required")
		return
	}

	if len(req.Messages) == 0 {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "messages array is empty")
		return
	}

	// Find the actual model ID
	actualModelID, err := h.findModel(req.Model)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "invalid_request_error", err.Error())
		return
	}

	// Get model port
	port, err := h.getModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Convert to OpenAI format and forward
	h.forwardToOpenAI(c, actualModelID, port, req)
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

		if m.Alias == modelName || m.Name == modelName {
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

// forwardToOpenAI converts Anthropic request to OpenAI format and forwards
func (h *Handler) forwardToOpenAI(c *gin.Context, modelID string, port int, anthropicReq MessageRequest) {
	// Convert Anthropic messages to OpenAI format
	messages := make([]map[string]interface{}, 0, len(anthropicReq.Messages)+1)

	// Add system message if present
	if anthropicReq.System != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": anthropicReq.System,
		})
	}

	// Add user messages
	for _, msg := range anthropicReq.Messages {
		messages = append(messages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	openaiReq := map[string]interface{}{
		"model":    modelID,
		"messages": messages,
		"stream":   anthropicReq.Stream,
		"max_tokens": anthropicReq.MaxTokens,
	}

	if anthropicReq.Temperature > 0 {
		openaiReq["temperature"] = anthropicReq.Temperature
	}
	if anthropicReq.TopP > 0 {
		openaiReq["top_p"] = anthropicReq.TopP
	}
	if anthropicReq.TopK > 0 {
		openaiReq["top_k"] = anthropicReq.TopK
	}

	// Marshal request body
	body, err := json.Marshal(openaiReq)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewReader(body))
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		h.sendError(c, http.StatusBadGateway, "internal_error", err.Error())
		logger.Errorf("转发请求到 llama.cpp 失败: %v", err)
		return
	}
	defer utils.CloseQuietly(resp.Body)

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Convert OpenAI response to Anthropic format
	var openaiResp map[string]interface{}
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		// Forward as-is if conversion fails
		c.Header("Content-Type", "application/json")
		c.Status(resp.StatusCode)
		utils.WriteQuietly(c.Writer, respBody)
		return
	}

	// Convert response format
	anthropicResp := h.convertResponse(openaiResp, anthropicReq.Model)

	c.Header("Content-Type", "application/json")
	c.JSON(resp.StatusCode, anthropicResp)
}

// convertResponse converts OpenAI response to Anthropic format
func (h *Handler) convertResponse(openaiResp map[string]interface{}, model string) *MessageResponse {
	resp := &MessageResponse{
		ID:     generateID("msg"),
		Type:   "message",
		Role:   "assistant",
		Model:  model,
	}

	// Extract choices
	if choices, ok := openaiResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if firstChoice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := firstChoice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					resp.Content = []ContentBlock{
						{Type: "text", Text: content},
					}
				}
			}
			if finishReason, ok := firstChoice["finish_reason"].(string); ok {
				resp.StopReason = finishReason
			}
		}
	}

	// Extract usage
	if usage, ok := openaiResp["usage"].(map[string]interface{}); ok {
		resp.Usage = &Usage{}
		if inputTokens, ok := usage["prompt_tokens"].(float64); ok {
			resp.Usage.InputTokens = int(inputTokens)
		}
		if outputTokens, ok := usage["completion_tokens"].(float64); ok {
			resp.Usage.OutputTokens = int(outputTokens)
		}
	}

	return resp
}

// sendError sends an error response in Anthropic format
func (h *Handler) sendError(c *gin.Context, statusCode int, errorType, message string) {
	resp := &MessageResponse{
		Error: &ErrorDetail{
			Type:    errorType,
			Message: message,
		},
	}

	// Set appropriate status code based on error type
	switch errorType {
	case "invalid_request":
		c.JSON(http.StatusBadRequest, resp)
	case "invalid_request_error":
		c.JSON(http.StatusNotFound, resp)
	default:
		c.JSON(statusCode, resp)
	}
}

// generateID generates a unique ID for Anthropic responses
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, "id")
}
