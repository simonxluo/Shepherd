package openai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
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
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if req.Prompt == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: prompt", "prompt")
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

	h.ForwardRequest(c, port, "/v1/images/generations", actualModelID, &req)
}
