package openai

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

// @Summary      OpenAI Chat Completions
// @Description  OpenAI 兼容的聊天补全接口，支持流式和非流式响应
// @Tags         OpenAI
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

	if req.Stream {
		h.StreamWithLazyLoad(c, req.Model, "/v1/chat/completions", &req)
	} else {
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

		h.ForwardRequest(c, port, "/v1/chat/completions", actualModelID, &req)
	}
}

// @Summary      OpenAI Completions
// @Description  OpenAI 兼容的文本补全接口
// @Tags         OpenAI
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
// @Description  获取已加载的模型列表（OpenAI 格式）
// @Tags         OpenAI
// @Produce      json
// @Success      200  {object}  ModelsResponse
// @Router       /v1/models [get]
func (h *Handler) HandleModels(c *gin.Context) {
	models := h.ListLoadedModels("shepherd")
	c.JSON(http.StatusOK, models)
}
