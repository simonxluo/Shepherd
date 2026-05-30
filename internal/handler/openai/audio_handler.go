package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

type AudioHandler struct {
	*Handler
	storageMgr *storage.Manager
	ttsDataDir string
}

func NewAudioHandler(modelMgr *model.Manager, storageMgr *storage.Manager, ttsDataDir string) *AudioHandler {
	return &AudioHandler{
		Handler:    NewHandler(modelMgr),
		storageMgr: storageMgr,
		ttsDataDir: ttsDataDir,
	}
}

// resolveTTSAudioURL converts a frontend /api/tts/audio/<id> path to a file:// absolute URL
// that vLLM accepts. Returns the original string if the pattern doesn't match or lookup fails.
func (h *AudioHandler) resolveTTSAudioURL(audioURL string) string {
	const prefix = "/api/tts/audio/"
	if !strings.HasPrefix(audioURL, prefix) {
		return audioURL
	}

	id := strings.TrimPrefix(audioURL, prefix)
	if id == "" {
		return audioURL
	}

	if h.storageMgr == nil {
		logger.Warnf("TTS 音频路径解析失败: storageMgr 未初始化, id=%s", id)
		return audioURL
	}

	item, err := h.storageMgr.GetStore().GetTTSHistory(context.Background(), id)
	if err != nil {
		logger.Warnf("TTS 音频路径解析失败: 查找历史记录失败, id=%s, err=%v", id, err)
		return audioURL
	}

	absPath, err := filepath.Abs(filepath.Join(h.ttsDataDir, item.AudioPath))
	if err != nil {
		logger.Warnf("TTS 音频路径解析失败: 获取绝对路径失败, id=%s, err=%v", id, err)
		return audioURL
	}

	fileURL := "file://" + absPath
	logger.Infof("TTS 参考音频路径解析: %s -> %s", audioURL, fileURL)
	return fileURL
}

// prepareTTSRequest reads the raw JSON body, validates required fields, resolves
// audio paths, and returns the prepared payload map for forwarding to vLLM.
func (h *AudioHandler) prepareTTSRequest(c *gin.Context) (map[string]interface{}, string, bool) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return nil, "", false
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return nil, "", false
	}

	// Validate required fields
	modelName, _ := payload["model"].(string)
	if modelName == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return nil, "", false
	}

	input, _ := payload["input"].(string)
	if input == "" {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: input", "input")
		return nil, "", false
	}

	// Resolve model
	actualModelID, err := h.FindModel(modelName)
	if err != nil {
		h.SendOpenAIError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return nil, "", false
	}

	// Verify TTS capability
	caps := h.ModelMgr.GetModelCapabilities(actualModelID)
	if caps == nil || !caps.TTS {
		h.SendOpenAIError(c, http.StatusBadRequest, "invalid_model",
			fmt.Sprintf("模型 %q 不支持 TTS（语音合成），请选择支持 TTS 的模型", modelName), "model")
		return nil, "", false
	}

	// Verify backend supports /v1/audio/speech
	b := h.ModelMgr.GetBackendForModel(actualModelID)
	if b != nil {
		endpoints := b.SupportedEndpoints()
		if supported, ok := endpoints["/v1/audio/speech"]; !ok || !supported {
			h.SendOpenAIError(c, http.StatusBadRequest, "backend_not_supported",
				fmt.Sprintf("当前后端 %q 不支持 TTS 端点，请使用 vLLM-Omni 后端加载模型", b.Type()), "model")
			return nil, "", false
		}
	}

	// Resolve audio paths: /api/tts/audio/<id> -> file://<abs>
	if v, ok := payload["ref_audio"].(string); ok && v != "" {
		payload["ref_audio"] = h.resolveTTSAudioURL(v)
	}

	return payload, actualModelID, true
}

// HandleCreateSpeech proxies POST /v1/audio/speech (TTS) to the backend model.
// The vLLM-Omni backend returns raw audio binary data.
func (h *AudioHandler) HandleCreateSpeech(c *gin.Context) {
	payload, actualModelID, ok := h.prepareTTSRequest(c)
	if !ok {
		return
	}

	port, err := h.GetModelPort(actualModelID)
	if err != nil {
		h.SendOpenAIError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	isStream, _ := payload["stream"].(bool)
	if isStream {
		modelName, _ := payload["model"].(string)
		logger.Infof("TTS 流式请求: model=%s, port=%d", modelName, port)
		h.ForwardStreamRequest(c, port, "/v1/audio/speech", actualModelID, payload)
	} else {
		h.ForwardBinaryRequest(c, port, "/v1/audio/speech", actualModelID, payload)
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

	// Verify that the model has ASR capabilities
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

// HandleListVoices proxies GET /v1/audio/voices to the backend, returning available voices.
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
