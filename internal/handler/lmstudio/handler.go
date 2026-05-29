package lmstudio

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

// @Summary      LM Studio Chat Completions
// @Description  LM Studio-compatible chat completions endpoint, supports streaming and non-streaming responses
// @Tags         LMStudio
// @Accept       json
// @Produce      json
// @Param        request  body  compat.ChatCompletionRequest  true  "Chat completion request"
// @Success      200  {object}  compat.ChatCompletionResponse
// @Failure      400  {object}  compat.ErrorResponse
// @Failure      404  {object}  compat.ErrorResponse
// @Failure      502  {object}  compat.ErrorResponse
// @Router       /v1/chat/completions [post]
func (h *Handler) HandleChatCompletions(c *gin.Context) {
	var req compat.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if len(req.Messages) == 0 {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Messages array is empty", "messages")
		return
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	if req.Stream {
		h.ForwardStreamRequest(c, port, "/v1/chat/completions", actualModelID, &req)
	} else {
		h.ForwardRequest(c, port, "/v1/chat/completions", actualModelID, &req)
	}
}

// @Summary      LM Studio Completions
// @Description  LM Studio-compatible text completions endpoint
// @Tags         LMStudio
// @Accept       json
// @Produce      json
// @Param        request  body  compat.CompletionRequest  true  "Completion request"
// @Success      200  {object}  compat.CompletionResponse
// @Failure      400  {object}  compat.ErrorResponse
// @Failure      404  {object}  compat.ErrorResponse
// @Failure      502  {object}  compat.ErrorResponse
// @Router       /v1/completions [post]
func (h *Handler) HandleCompletions(c *gin.Context) {
	var req compat.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		models := h.ModelMgr.ListStatus()
		if len(models) == 0 {
			h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", "No models are currently loaded", "model")
			return
		}
		for modelID := range models {
			req.Model = modelID
			break
		}
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	if req.Stream {
		h.ForwardStreamRequest(c, port, "/v1/completions", actualModelID, &req)
	} else {
		h.ForwardRequest(c, port, "/v1/completions", actualModelID, &req)
	}
}

// @Summary      List Models
// @Description  List loaded models in LM Studio format
// @Tags         LMStudio
// @Produce      json
// @Success      200  {object}  compat.ModelsResponse
// @Router       /v1/models [get]
func (h *Handler) HandleModels(c *gin.Context) {
	models := h.ListLoadedModels("shepherd")
	c.JSON(http.StatusOK, models)
}

// @Summary      LM Studio Embeddings
// @Description  LM Studio-compatible embeddings endpoint, proxied to llama.cpp backend
// @Tags         LMStudio
// @Accept       json
// @Produce      json
// @Param        request  body  compat.EmbeddingRequest  true  "Embedding request"
// @Success      200  {object}  compat.EmbeddingResponse
// @Failure      400  {object}  compat.ErrorResponse
// @Failure      404  {object}  compat.ErrorResponse
// @Failure      502  {object}  compat.ErrorResponse
// @Router       /v1/embeddings [post]
func (h *Handler) HandleEmbeddings(c *gin.Context) {
	var req compat.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if req.Input == nil {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: input", "input")
		return
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	h.ForwardRequest(c, port, "/v1/embeddings", actualModelID, &req)
}
