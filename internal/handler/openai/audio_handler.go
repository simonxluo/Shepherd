package openai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
)

type AudioHandler struct {
	*Handler
}

func NewAudioHandler(modelMgr *model.Manager) *AudioHandler {
	return &AudioHandler{
		Handler: NewHandler(modelMgr),
	}
}

// HandleCreateSpeech proxies POST /v1/audio/speech (TTS) to the backend model.
// The vLLM-Omni backend returns raw audio binary data.
func (h *AudioHandler) HandleCreateSpeech(c *gin.Context) {
	var req struct {
		Model          string  `json:"model"`
		Input          string  `json:"input"`
		Voice          string  `json:"voice,omitempty"`
		ResponseFormat string  `json:"response_format,omitempty"`
		Speed          float64 `json:"speed,omitempty"`
		Language       string  `json:"language,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if req.Input == "" {
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

	h.ForwardBinaryRequest(c, port, "/v1/audio/speech", actualModelID, &req)
}

// HandleCreateTranscription proxies POST /v1/audio/transcriptions (ASR) to the backend model.
// This accepts multipart/form-data with an audio file.
func (h *AudioHandler) HandleCreateTranscription(c *gin.Context) {
	modelName := c.PostForm("model")
	if modelName == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	actualModelID, err := h.FindModel(modelName)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	formFields := map[string]string{}
	for _, field := range []string{
		"model", "language", "prompt", "response_format",
		"temperature", "timestamp_granularities[]",
	} {
		if v := c.PostForm(field); v != "" {
			formFields[field] = v
		}
	}

	h.ForwardMultipartRequest(c, port, "/v1/audio/transcriptions", actualModelID, formFields)
}

// HandleCreateTranslation proxies POST /v1/audio/translations to the backend model.
func (h *AudioHandler) HandleCreateTranslation(c *gin.Context) {
	modelName := c.PostForm("model")
	if modelName == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	actualModelID, err := h.FindModel(modelName)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	formFields := map[string]string{}
	for _, field := range []string{
		"model", "prompt", "response_format", "temperature",
	} {
		if v := c.PostForm(field); v != "" {
			formFields[field] = v
		}
	}

	h.ForwardMultipartRequest(c, port, "/v1/audio/translations", actualModelID, formFields)
}
