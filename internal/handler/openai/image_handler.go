package openai

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

type ImageHandler struct {
	*Handler
}

func NewImageHandler(modelMgr *model.Manager) *ImageHandler {
	return &ImageHandler{
		Handler: NewHandler(modelMgr),
	}
}

// HandleCreateImage proxies POST /v1/images/generations to the backend model.
// The vLLM-Omni backend returns image data (base64 or URL).
func (h *ImageHandler) HandleCreateImage(c *gin.Context) {
	var req struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt"`
		N              int    `json:"n,omitempty"`
		Size           string `json:"size,omitempty"`
		ResponseFormat string `json:"response_format,omitempty"`
		Quality        string `json:"quality,omitempty"`
		Style          string `json:"style,omitempty"`
		User           string `json:"user,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if req.Prompt == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: prompt", "prompt")
		return
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	caps := h.ModelMgr.GetModelCapabilities(actualModelID)
	if caps == nil || !caps.ImageGeneration {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_model",
			fmt.Sprintf("Model %q does not support image generation, please select a model with image generation capability", req.Model), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	h.ForwardRequest(c, port, "/v1/images/generations", actualModelID, &req)
}
