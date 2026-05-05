package lmstudio

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler/compat"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
)

type Handler struct {
	*compat.BaseHandler
}

func NewHandler(modelMgr *model.Manager) *Handler {
	return &Handler{
		BaseHandler: compat.NewBaseHandler(modelMgr),
	}
}

// @Summary      LM Studio Chat Completions
// @Description  LM Studio 兼容的聊天补全接口，支持流式和非流式响应
// @Tags         LMStudio
// @Accept       json
// @Produce      json
// @Param        request  body  ChatCompletionRequest  true  "Chat completion request"
// @Success      200  {object}  ChatCompletionResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      502  {object}  ErrorResponse
// @Router       /v1/chat/completions [post]
func (h *Handler) HandleChatCompletions(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if len(req.Messages) == 0 {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Messages array is empty", "messages")
		return
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	if req.Stream {
		h.ForwardStreamRequest(c, port, "/v1/chat/completions", actualModelID, &req)
	} else {
		h.ForwardRequest(c, port, "/v1/chat/completions", actualModelID, &req)
	}
}

// @Summary      LM Studio Completions
// @Description  LM Studio 兼容的文本补全接口
// @Tags         LMStudio
// @Accept       json
// @Produce      json
// @Param        request  body  CompletionRequest  true  "Completion request"
// @Success      200  {object}  CompletionResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      502  {object}  ErrorResponse
// @Router       /v1/completions [post]
func (h *Handler) HandleCompletions(c *gin.Context) {
	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		models := h.ModelMgr.ListStatus()
		if len(models) == 0 {
			h.sendError(c, http.StatusNotFound, "model_not_found", "No models are currently loaded", "model")
			return
		}
		for modelID := range models {
			req.Model = modelID
			break
		}
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	if req.Stream {
		h.ForwardStreamRequest(c, port, "/v1/completions", actualModelID, &req)
	} else {
		h.ForwardRequest(c, port, "/v1/completions", actualModelID, &req)
	}
}

// @Summary      List Models
// @Description  获取已加载的模型列表（LM Studio 格式）
// @Tags         LMStudio
// @Produce      json
// @Success      200  {object}  ModelsResponse
// @Router       /v1/models [get]
func (h *Handler) HandleModels(c *gin.Context) {
	statuses := h.ModelMgr.ListStatus()
	models := h.ModelMgr.ListModels()

	var lmstudioModels []Model
	for _, m := range models {
		if status, exists := statuses[m.ID]; exists && status.State == model.StateLoaded {
			lmstudioModels = append(lmstudioModels, Model{
				ID:      m.ID,
				Object:  "model",
				Created: m.ScannedAt.Unix(),
				OwnedBy: "shepherd",
			})
		}
	}

	response := NewModelsResponse(lmstudioModels)
	c.JSON(http.StatusOK, response)
}

// @Summary      LM Studio Embeddings
// @Description  LM Studio 兼容的嵌入接口，内部转发到 llama.cpp
// @Tags         LMStudio
// @Accept       json
// @Produce      json
// @Param        request  body  EmbeddingRequest  true  "Embedding request"
// @Success      200  {object}  EmbeddingResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      502  {object}  ErrorResponse
// @Router       /v1/embeddings [post]
func (h *Handler) HandleEmbeddings(c *gin.Context) {
	var req EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if req.Input == nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: input", "input")
		return
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	h.ForwardRequest(c, port, "/v1/embeddings", actualModelID, &req)
}

func (h *Handler) sendError(c *gin.Context, statusCode int, errorType, message, param string) {
	response := NewErrorResponse(message, errorType, param, statusCode)
	c.JSON(statusCode, response)
}

// Types

type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []ChatMessage          `json:"messages"`
	Stream           bool                   `json:"stream,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	TopK             int                    `json:"top_k,omitempty"`
	N                int                    `json:"n,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Seed             int                    `json:"seed,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	RepeatPenalty    float64                `json:"repeat_penalty,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	Tools            []Tool                 `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	Extra            map[string]interface{} `json:"-"`
}

type ChatMessage struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	Name         string        `json:"name,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
}

type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type ToolCall struct {
	ID       string        `json:"id,omitempty"`
	Type     string        `json:"type,omitempty"`
	Function *FunctionCall `json:"function,omitempty"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function,omitempty"`
}

type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type,omitempty"`
}

type CompletionRequest struct {
	Model            string                 `json:"model"`
	Prompt           interface{}            `json:"prompt"`
	Stream           bool                   `json:"stream,omitempty"`
	Suffix           string                 `json:"suffix,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	TopK             int                    `json:"top_k,omitempty"`
	N                int                    `json:"n,omitempty"`
	LogProbs         int                    `json:"logprobs,omitempty"`
	Echo             bool                   `json:"echo,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	RepeatPenalty    float64                `json:"repeat_penalty,omitempty"`
	Seed             int                    `json:"seed,omitempty"`
	Extra            map[string]interface{} `json:"-"`
}

type EmbeddingRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"`
}

type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	Choices           []ChatCompletionChoice `json:"choices"`
	Usage             *Usage                 `json:"usage,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason,omitempty"`
	LogProbs     interface{} `json:"logprobs,omitempty"`
}

type CompletionResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Created           int64              `json:"created"`
	Model             string             `json:"model"`
	Choices           []CompletionChoice `json:"choices"`
	Usage             *Usage             `json:"usage,omitempty"`
	SystemFingerprint string             `json:"system_fingerprint,omitempty"`
}

type CompletionChoice struct {
	Index        int         `json:"index"`
	Text         string      `json:"text"`
	FinishReason string      `json:"finish_reason,omitempty"`
	LogProbs     interface{} `json:"logprobs,omitempty"`
}

type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  *Usage          `json:"usage,omitempty"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

func NewModelsResponse(models []Model) *ModelsResponse {
	return &ModelsResponse{
		Object: "list",
		Data:   models,
	}
}

func NewErrorResponse(message, errorType, param string, statusCode int) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errorType,
			Param:   param,
			Code:    string(rune(statusCode)),
		},
	}
}
