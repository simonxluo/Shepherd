package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/handler/compat"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

type Handler struct {
	*compat.BaseHandler
}

func NewHandler(modelMgr *model.Manager) *Handler {
	return &Handler{
		BaseHandler: compat.NewBaseHandler(modelMgr),
	}
}

type MessageRequest struct {
	Model         string    `json:"model"`
	MaxTokens     int       `json:"max_tokens"`
	Messages      []Message `json:"messages"`
	System        string    `json:"system,omitempty"`
	Temperature   float64   `json:"temperature,omitempty"`
	TopP          float64   `json:"top_p,omitempty"`
	TopK          int       `json:"top_k,omitempty"`
	Stream        bool      `json:"stream,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MessageResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Model      string         `json:"model"`
	Usage      *Usage         `json:"usage,omitempty"`
	Error      *ErrorDetail   `json:"error,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// @Summary      Anthropic Messages
// @Description  Anthropic-compatible messages endpoint, internally converts to OpenAI format for forwarding
// @Tags         Anthropic
// @Accept       json
// @Produce      json
// @Param        request  body  MessageRequest  true  "Anthropic message request"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  MessageResponse
// @Failure      404  {object}  MessageResponse
// @Failure      502  {object}  MessageResponse
// @Router       /v1/messages [post]
func (h *Handler) HandleMessages(c *gin.Context) {
	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

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

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "invalid_request_error", err.Error())
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	openaiReq := h.convertToOpenAI(actualModelID, req)
	body, _ := json.Marshal(openaiReq)
	respBody, resp, err := h.ForwardRequestRaw(c, port, "/v1/chat/completions", actualModelID, body)
	if err != nil {
		h.sendError(c, http.StatusBadGateway, "internal_error", err.Error())
		logger.Errorf("转发请求到 llama.cpp 失败: %v", err)
		return
	}

	var openaiResp map[string]interface{}
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		c.Header("Content-Type", "application/json")
		c.Status(resp.StatusCode)
		utils.WriteQuietly(c.Writer, respBody)
		return
	}

	anthropicResp := h.convertResponse(openaiResp, req.Model)
	c.Header("Content-Type", "application/json")
	c.JSON(resp.StatusCode, anthropicResp)
}

func (h *Handler) convertToOpenAI(modelID string, anthropicReq MessageRequest) map[string]interface{} {
	messages := make([]map[string]interface{}, 0, len(anthropicReq.Messages)+1)

	if anthropicReq.System != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": anthropicReq.System,
		})
	}

	for _, msg := range anthropicReq.Messages {
		messages = append(messages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	openaiReq := map[string]interface{}{
		"model":      modelID,
		"messages":   messages,
		"stream":     anthropicReq.Stream,
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

	return openaiReq
}

func (h *Handler) convertResponse(openaiResp map[string]interface{}, model string) *MessageResponse {
	resp := &MessageResponse{
		ID:    generateID("msg"),
		Type:  "message",
		Role:  "assistant",
		Model: model,
	}

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

func (h *Handler) sendError(c *gin.Context, statusCode int, errorType, message string) {
	resp := &MessageResponse{
		Error: &ErrorDetail{
			Type:    errorType,
			Message: message,
		},
	}

	switch errorType {
	case "invalid_request":
		c.JSON(http.StatusBadRequest, resp)
	case "invalid_request_error":
		c.JSON(http.StatusNotFound, resp)
	default:
		c.JSON(statusCode, resp)
	}
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
