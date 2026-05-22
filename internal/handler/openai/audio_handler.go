package openai

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/service/model"
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
		Stream         bool    `json:"stream,omitempty"`
		// VoxCPM2 / 声音克隆扩展字段
		Instructions       string  `json:"instructions,omitempty"`
		RefAudio           string  `json:"ref_audio,omitempty"`
		RefText            string  `json:"ref_text,omitempty"`
		PromptAudio        string  `json:"prompt_audio,omitempty"`
		PromptText         string  `json:"prompt_text,omitempty"`
		MaxNewTokens       int     `json:"max_new_tokens,omitempty"`
		Seed               int64   `json:"seed,omitempty"`
		CfgValue           float64 `json:"cfg_value,omitempty"`
		InferenceTimesteps int     `json:"inference_timesteps,omitempty"`
		ExtraParams        any     `json:"extra_params,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	if req.Model == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if req.Input == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: input", "input")
		return
	}

	actualModelID, err := h.FindModel(req.Model)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	// 验证模型具备 TTS 能力
	caps := h.ModelMgr.GetModelCapabilities(actualModelID)
	if caps == nil || !caps.TTS {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_model", fmt.Sprintf("模型 %q 不支持 TTS（语音合成），请选择支持 TTS 的模型", req.Model), "model")
		return
	}

	// 验证后端支持 /v1/audio/speech 端点
	b := h.ModelMgr.GetBackendForModel(actualModelID)
	if b != nil {
		endpoints := b.SupportedEndpoints()
		if supported, ok := endpoints["/v1/audio/speech"]; !ok || !supported {
			h.SendOpenAIError(c, http.StatusBadRequest, "backend_not_supported",
				fmt.Sprintf("当前后端 %q 不支持 TTS 端点，请使用 vLLM-Omni 后端加载模型", b.Type()), "model")
			return
		}
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	if req.Stream {
		logger.Infof("TTS 流式请求: model=%s, port=%d", req.Model, port)
		h.ForwardStreamRequest(c, port, "/v1/audio/speech", actualModelID, &req)
	} else {
		h.ForwardBinaryRequest(c, port, "/v1/audio/speech", actualModelID, &req)
	}
}

// HandleCreateTranscription proxies POST /v1/audio/transcriptions (ASR) to the backend model.
// This accepts multipart/form-data with an audio file.
func (h *AudioHandler) HandleCreateTranscription(c *gin.Context) {
	modelName := c.PostForm("model")
	if modelName == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	actualModelID, err := h.FindModel(modelName)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	// 验证模型具备 ASR 能力
	caps := h.ModelMgr.GetModelCapabilities(actualModelID)
	if caps == nil || !caps.ASR {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_model", fmt.Sprintf("模型 %q 不支持 ASR（语音识别），请选择支持 ASR 的模型", modelName), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
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
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	actualModelID, err := h.FindModel(modelName)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
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

// HandleListVoices 代理 GET /v1/audio/voices 到后端，返回可用语音列表
func (h *AudioHandler) HandleListVoices(c *gin.Context) {
	modelName := c.Query("model")
	if modelName == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "缺少必要参数: model", "model")
		return
	}

	actualModelID, err := h.FindModel(modelName)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	h.ForwardGetRequest(c, port, "/v1/audio/voices")
}
