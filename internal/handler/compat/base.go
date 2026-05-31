package compat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

type BaseHandler struct {
	ModelMgr   *model.Manager
	Client     *http.Client
	ModelIndex *ModelLookupIndex
}

// maxResponseSize limits the proxied response body to a maximum of 1GB
const maxResponseSize = 1 << 30

func NewBaseHandler(modelMgr *model.Manager) *BaseHandler {
	b := &BaseHandler{
		ModelMgr: modelMgr,
		Client: &http.Client{
			Timeout: 0,
		},
		ModelIndex: NewModelLookupIndex(),
	}
	b.RebuildIndex()
	return b
}

func (b *BaseHandler) RebuildIndex() {
	models := b.ModelMgr.ListModels()
	b.ModelIndex.Rebuild(models)
}

func (b *BaseHandler) FindModel(modelName string) (string, error) {
	// Only rebuild the index when the model manager's version has changed
	currentVersion := b.ModelMgr.Version()
	if currentVersion != b.ModelIndex.lastVersion {
		b.RebuildIndex()
		b.ModelIndex.lastVersion = currentVersion
	}
	return FindModelForAPI(b.ModelMgr, b.ModelIndex, modelName)
}

// GetServedModelName returns the model name recognized by the backend service.
// vLLM backends use the model path as identifier, llama.cpp uses the model name.
func (b *BaseHandler) GetServedModelName(modelID string) string {
	if m, ok := b.ModelMgr.GetModel(modelID); ok && m != nil {
		status, exists := b.ModelMgr.GetStatus(modelID)
		if exists && (status.BackendType == "vllm" || status.BackendType == "vllm_omni") {
			return m.Path
		}
		return m.Name
	}
	return modelID
}

// replaceModelField replaces the model field in request body with the backend-recognized model name.
func replaceModelField(reqBody []byte, servedModelName string) []byte {
	var payload map[string]interface{}
	if err := json.Unmarshal(reqBody, &payload); err != nil {
		return reqBody
	}
	if _, hasModel := payload["model"]; hasModel {
		payload["model"] = servedModelName
		newBody, err := json.Marshal(payload)
		if err != nil {
			return reqBody
		}
		return newBody
	}
	return reqBody
}

// truncateString 截断字符串到指定长度，超出部分用 "..." 表示
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// buildRequestSummary 从 JSON 请求体中提取关键字段构建简短摘要
func buildRequestSummary(params map[string]interface{}) string {
	parts := []string{}

	// input 文本字段（TTS）
	if v, ok := params["input"].(string); ok && v != "" {
		parts = append(parts, fmt.Sprintf("input=%q", truncateString(v, 100)))
	}

	// messages 数量（Chat）
	if msgs, ok := params["messages"].([]interface{}); ok {
		parts = append(parts, fmt.Sprintf("messages=%d", len(msgs)))
	}

	// prompt 文本
	if v, ok := params["prompt"].(string); ok && v != "" {
		parts = append(parts, fmt.Sprintf("prompt=%q", truncateString(v, 100)))
	}

	// stream 标志
	if v, ok := params["stream"].(bool); ok {
		parts = append(parts, fmt.Sprintf("stream=%v", v))
	}

	// voice（语音名称）
	if v, ok := params["voice"].(string); ok && v != "" {
		parts = append(parts, fmt.Sprintf("voice=%q", v))
	}

	// ref_audio（参考音频，仅显示有无）
	if v, ok := params["ref_audio"].(string); ok && v != "" {
		parts = append(parts, fmt.Sprintf("ref_audio=%q", truncateString(v, 80)))
	}

	// ref_text（参考音频文本）
	if v, ok := params["ref_text"].(string); ok && v != "" {
		parts = append(parts, fmt.Sprintf("ref_text=%q", truncateString(v, 100)))
	}

	// temperature
	if v, ok := params["temperature"].(float64); ok {
		parts = append(parts, fmt.Sprintf("temperature=%.2f", v))
	}

	// language
	if v, ok := params["language"].(string); ok && v != "" {
		parts = append(parts, fmt.Sprintf("language=%q", v))
	}

	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// buildMultipartSummary 从 multipart 表单字段构建简短摘要
func buildMultipartSummary(formFields map[string]string) string {
	parts := []string{}

	keys := make([]string, 0, len(formFields))
	for k := range formFields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := formFields[k]
		if k == "model" {
			continue // model 已在外部日志中体现
		}
		parts = append(parts, fmt.Sprintf("%s=%q", k, truncateString(v, 100)))
	}

	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// logForwardRequest 记录转发到后端的 JSON 请求日志
func logForwardRequest(modelID string, port int, path string, reqBody []byte) {
	var params map[string]interface{}
	if json.Unmarshal(reqBody, &params) == nil {
		summary := buildRequestSummary(params)
		logger.Infof("转发请求 -> model=%s port=%d path=%s%s", modelID, port, path, summary)
	}
}

// logForwardMultipart 记录转发到后端的 multipart 请求日志
func logForwardMultipart(modelID string, port int, path string, formFields map[string]string) {
	summary := buildMultipartSummary(formFields)
	logger.Infof("转发请求 -> model=%s port=%d path=%s%s", modelID, port, path, summary)
}

func (b *BaseHandler) GetModelPort(modelID string) (int, error) {
	return GetModelPort(b.ModelMgr, modelID)
}

func (b *BaseHandler) ForwardRequest(c *gin.Context, port int, path string, modelID string, req interface{}) {
	if status, exists := b.ModelMgr.GetStatusRef(modelID); exists {
		if !status.AcquireSlot() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "model is at concurrent request limit"})
			return
		}
		defer status.ReleaseSlot()
		status.InflightAdd()
		defer status.InflightDone()
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	body, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	body = replaceModelField(body, b.GetServedModelName(modelID))

	logForwardRequest(modelID, port, path, body)

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	resp, err := b.Client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		logger.Errorf("转发请求到后端失败: %v", err)
		return
	}
	defer utils.CloseQuietly(resp.Body)

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	b.extractTokenUsage(modelID, respBody)

	c.Header("Content-Type", "application/json")
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	utils.WriteQuietly(c.Writer, respBody)
}

func (b *BaseHandler) ForwardStreamRequest(c *gin.Context, port int, path string, modelID string, req interface{}) {
	if status, exists := b.ModelMgr.GetStatusRef(modelID); exists {
		if !status.AcquireSlot() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "model is at concurrent request limit"})
			return
		}
		defer status.ReleaseSlot()
		status.InflightAdd()
		defer status.InflightDone()
	}

	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 0,
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
			resp.Header.Set("X-Accel-Buffering", "no")
			resp.Header.Set("Cache-Control", "no-cache")
			resp.Header.Set("Connection", "keep-alive")
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Errorf("转发流式请求到后端失败: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
	}

	c.Request.URL.Path = path
	c.Request.Host = fmt.Sprintf("127.0.0.1:%d", port)

	if req != nil {
		body, err := json.Marshal(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		body = replaceModelField(body, b.GetServedModelName(modelID))
		logForwardRequest(modelID, port, path, body)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Request.ContentLength = int64(len(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func (b *BaseHandler) ForwardRequestRaw(c *gin.Context, port int, path string, modelID string, body []byte) ([]byte, *http.Response, error) {
	if status, exists := b.ModelMgr.GetStatusRef(modelID); exists {
		if !status.AcquireSlot() {
			return nil, nil, fmt.Errorf("model is at concurrent request limit")
		}
		defer status.ReleaseSlot()
		status.InflightAdd()
		defer status.InflightDone()
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	logForwardRequest(modelID, port, path, body)

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	resp, err := b.Client.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer utils.CloseQuietly(resp.Body)

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, resp, err
	}

	return respBody, resp, nil
}

func (b *BaseHandler) StreamWithLazyLoad(c *gin.Context, modelName string, path string, req interface{}) {
	actualModelID := ""

	if status, exists := b.ModelMgr.GetStatusRef(modelName); exists {
		actualModelID = modelName
		if status.State == model.StateLoaded {
			port, err := b.GetModelPort(actualModelID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			b.ForwardStreamRequest(c, port, path, actualModelID, req)
			return
		}
	} else {
		// 刷新索引后再查找，确保扫描/加载/别名变更后的模型能被解析
		currentVersion := b.ModelMgr.Version()
		if currentVersion != b.ModelIndex.lastVersion {
			b.RebuildIndex()
			b.ModelIndex.lastVersion = currentVersion
		}
		if m, ok := b.ModelIndex.Find(modelName); ok {
			actualModelID = m.ID
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model not found: %s", modelName)})
			return
		}
	}

	status, exists := b.ModelMgr.GetStatusRef(actualModelID)
	if exists && status.State == model.StateLoaded {
		port, err := b.GetModelPort(actualModelID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		b.ForwardStreamRequest(c, port, path, actualModelID, req)
		return
	}

	loadDone := make(chan struct{})
	go func() {
		defer close(loadDone)
		if exists && status.State == model.StateLoading {
			status.LoadWait.Wait()
		} else {
			b.ModelMgr.EnsureLoaded(actualModelID)
		}
	}()

	loadCtx, loadCancel := context.WithCancel(context.Background())
	go func() {
		<-loadDone
		loadCancel()
	}()
	sendLoadingSSE(c, actualModelID, loadCtx)

	port, err := b.GetModelPort(actualModelID)
	if err != nil {
		c.Writer.Write([]byte(fmt.Sprintf("data: {\"error\":\"%s\"}\n\n", err.Error())))
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}
		return
	}

	b.ForwardStreamRequest(c, port, path, actualModelID, req)
}

func (b *BaseHandler) extractTokenUsage(modelID string, body []byte) {
	if status, exists := b.ModelMgr.GetStatusRef(modelID); exists {
		var resp struct {
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(body, &resp) == nil && resp.Usage != nil {
			status.AddTokens(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		}
	}
}

// ForwardBinaryRequest forwards a JSON request and returns the raw binary response
// without assuming JSON content type. Used for TTS audio responses.
func (b *BaseHandler) ForwardBinaryRequest(c *gin.Context, port int, path string, modelID string, req interface{}) {
	if status, exists := b.ModelMgr.GetStatusRef(modelID); exists {
		if !status.AcquireSlot() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "model is at concurrent request limit"})
			return
		}
		defer status.ReleaseSlot()
		status.InflightAdd()
		defer status.InflightDone()
	}

	reqURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	var bodyReader io.Reader
	if req != nil {
		body, err := json.Marshal(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		body = replaceModelField(body, b.GetServedModelName(modelID))
		logForwardRequest(modelID, port, path, body)
		bodyReader = bytes.NewReader(body)
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", reqURL, bodyReader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	resp, err := b.Client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer utils.CloseQuietly(resp.Body)

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	utils.WriteQuietly(c.Writer, respBody)
}

// ForwardMultipartRequest forwards a multipart/form-data request to the backend.
// Used for ASR audio file uploads.
func (b *BaseHandler) ForwardMultipartRequest(c *gin.Context, port int, path string, modelID string, formFields map[string]string) {
	if status, exists := b.ModelMgr.GetStatusRef(modelID); exists {
		if !status.AcquireSlot() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "model is at concurrent request limit"})
			return
		}
		defer status.ReleaseSlot()
		status.InflightAdd()
		defer status.InflightDone()
	}

	reqURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	logForwardMultipart(modelID, port, path, formFields)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Copy form fields
	for key, value := range formFields {
		if err := writer.WriteField(key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Copy file from the incoming request
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		part, err := writer.CreateFormFile("file", header.Filename)
		if err != nil {
			utils.CloseQuietly(file)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		io.Copy(part, file)
		utils.CloseQuietly(file)
	}

	writer.Close()

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", reqURL, &body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	resp, err := b.Client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer utils.CloseQuietly(resp.Body)

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	utils.WriteQuietly(c.Writer, respBody)
}

// SendOpenAIError sends an error response in OpenAI API format.
func (b *BaseHandler) SendOpenAIError(c *gin.Context, status int, errType, msg, param string) {
	response := NewErrorResponse(msg, errType, param, status)
	c.JSON(status, response)
}

// SendSimpleError sends a simple JSON error response with {"error": msg}.
func (b *BaseHandler) SendSimpleError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// ListLoadedModels returns loaded models in OpenAI ModelsResponse format.
func (b *BaseHandler) ListLoadedModels(ownedBy string) *ModelsResponse {
	statuses := b.ModelMgr.ListStatus()
	models := b.ModelMgr.ListModels()

	var result []Model
	for _, m := range models {
		if status, exists := statuses[m.ID]; exists && status.State == model.StateLoaded {
			result = append(result, Model{
				ID:      m.ID,
				Object:  "model",
				Created: m.ScannedAt.Unix(),
				OwnedBy: ownedBy,
			})
		}
	}

	return NewModelsResponse(result)
}

// ForwardGetRequest proxies a GET request to the backend model service.
func (b *BaseHandler) ForwardGetRequest(c *gin.Context, port int, path string) {
	reqURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "GET", reqURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpReq.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	resp, err := b.Client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer utils.CloseQuietly(resp.Body)

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	utils.WriteQuietly(c.Writer, respBody)
}

func (b *BaseHandler) ForwardDeleteRequest(c *gin.Context, port int, path string) {
	reqURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "DELETE", reqURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpReq.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	resp, err := b.Client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer utils.CloseQuietly(resp.Body)

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	utils.WriteQuietly(c.Writer, respBody)
}
