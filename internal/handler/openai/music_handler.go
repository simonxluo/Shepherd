package openai

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
)

type MusicHandler struct {
	*Handler
}

func NewMusicHandler(modelMgr *model.Manager) *MusicHandler {
	return &MusicHandler{
		Handler: NewHandler(modelMgr),
	}
}

// HandleCreateMusic proxies POST /v1/audio/music to the backend model.
// The backend returns raw audio binary data (same pattern as TTS).
func (h *MusicHandler) HandleCreateMusic(c *gin.Context) {
	var req struct {
		Model          string  `json:"model"`
		Prompt         string  `json:"prompt"`
		Duration       float64 `json:"duration,omitempty"`
		ResponseFormat string  `json:"response_format,omitempty"`
		Temperature    float64 `json:"temperature,omitempty"`
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

	// Validate model has Music capability
	caps := h.ModelMgr.GetModelCapabilities(actualModelID)
	if caps == nil || !caps.Music {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_model",
			fmt.Sprintf("Model %q does not support music generation, please select a model with music capability", req.Model), "model")
		return
	}

	// Validate backend supports /v1/audio/music endpoint
	b := h.ModelMgr.GetBackendForModel(actualModelID)
	if b != nil {
		endpoints := b.SupportedEndpoints()
		if supported, ok := endpoints["/v1/audio/music"]; !ok || !supported {
			h.SendOpenAIError(c, http.StatusBadRequest, "backend_not_supported",
				fmt.Sprintf("Backend %q does not support the music generation endpoint", b.Type()), "model")
			return
		}
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	h.ForwardBinaryRequest(c, port, "/v1/audio/music", actualModelID, &req)
}
