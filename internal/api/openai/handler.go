// Package openai provides OpenAI API compatibility layer
package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
)

// ModelIndex 加速模型查找的索引结构
type ModelIndex struct {
	byID     map[string]*model.Model  // 按 ID 索引
	byAlias  map[string]*model.Model  // 按别名索引
	byName   map[string]*model.Model  // 按名称索引
	mu       sync.RWMutex              // 保护索引的读写锁
}

// NewModelIndex 创建新的模型索引
func NewModelIndex() *ModelIndex {
	return &ModelIndex{
		byID:    make(map[string]*model.Model),
		byAlias: make(map[string]*model.Model),
		byName:  make(map[string]*model.Model),
	}
}

// Build 构建模型索引
func (idx *ModelIndex) Build(models []*model.Model) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 清空旧索引
	idx.byID = make(map[string]*model.Model)
	idx.byAlias = make(map[string]*model.Model)
	idx.byName = make(map[string]*model.Model)

	// 构建新索引
	for _, m := range models {
		idx.byID[m.ID] = m
		if m.Alias != "" {
			idx.byAlias[m.Alias] = m
		}
		idx.byName[m.Name] = m
	}
}

// Find 查找模型（按优先级：ID > 别名 > 名称）
func (idx *ModelIndex) Find(identifier string) (*model.Model, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 优先按 ID 查找
	if m, ok := idx.byID[identifier]; ok {
		return m, true
	}

	// 按别名查找
	if m, ok := idx.byAlias[identifier]; ok {
		return m, true
	}

	// 按名称查找（不区分大小写）
	for id, m := range idx.byName {
		if strings.EqualFold(id, identifier) {
			return m, true
		}
	}

	// 按名称模糊匹配（ID 包含搜索字符串）
	lowerIdentifier := strings.ToLower(identifier)
	for _, m := range idx.byID {
		if strings.Contains(strings.ToLower(m.ID), lowerIdentifier) {
			return m, true
		}
	}

	return nil, false
}

// Handler handles OpenAI API requests
type Handler struct {
	modelMgr  *model.Manager
	client     *http.Client
	modelIndex *ModelIndex
}

// NewHandler creates a new OpenAI API handler
func NewHandler(modelMgr *model.Manager) *Handler {
	h := &Handler{
		modelMgr: modelMgr,
		client: &http.Client{
			Timeout: 0, // No timeout for streaming responses
		},
		modelIndex: NewModelIndex(),
	}

	// 初始构建索引
	h.rebuildIndex()

	return h
}

// rebuildIndex 重建模型索引
func (h *Handler) rebuildIndex() {
	models := h.modelMgr.ListModels()
	h.modelIndex.Build(models)
}

// HandleChatCompletions handles chat completion requests
func (h *Handler) HandleChatCompletions(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	// Validate request
	if req.Model == "" {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Missing required parameter: model", "model")
		return
	}

	if len(req.Messages) == 0 {
		h.sendError(c, http.StatusBadRequest, "invalid_request", "Messages array is empty", "messages")
		return
	}

	// Find the actual model ID
	actualModelID, err := h.findModel(req.Model)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	// Get model port
	port, err := h.getModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	// Forward request to llama.cpp
	if req.Stream {
		h.forwardStreamRequest(c, actualModelID, port, "/v1/chat/completions", &req)
	} else {
		h.forwardRequest(c, actualModelID, port, "/v1/chat/completions", &req)
	}
}

// HandleCompletions handles legacy completion requests
func (h *Handler) HandleCompletions(c *gin.Context) {
	var req CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.sendError(c, http.StatusBadRequest, "invalid_request", err.Error(), "body")
		return
	}

	// If model is not specified, use the first loaded model
	if req.Model == "" {
		models := h.modelMgr.ListStatus()
		if len(models) == 0 {
			h.sendError(c, http.StatusNotFound, "model_not_found", "No models are currently loaded", "model")
			return
		}
		for modelID := range models {
			req.Model = modelID
			break
		}
	}

	// Find the actual model ID
	actualModelID, err := h.findModel(req.Model)
	if err != nil {
		h.sendError(c, http.StatusNotFound, "model_not_found", err.Error(), "model")
		return
	}

	// Get model port
	port, err := h.getModelPort(actualModelID)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	// Forward request to llama.cpp
	if req.Stream {
		h.forwardStreamRequest(c, actualModelID, port, "/v1/completions", &req)
	} else {
		h.forwardRequest(c, actualModelID, port, "/v1/completions", &req)
	}
}

// HandleModels handles the list models request
func (h *Handler) HandleModels(c *gin.Context) {
	statuses := h.modelMgr.ListStatus()
	models := h.modelMgr.ListModels()

	var openaiModels []Model
	for _, m := range models {
		// Only include loaded models
		if status, exists := statuses[m.ID]; exists && status.State == model.StateLoaded {
			openaiModels = append(openaiModels, Model{
				ID:      m.ID,
				Object:  "model",
				Created: m.ScannedAt.Unix(),
				OwnedBy: "shepherd",
			})
		}
	}

	response := NewModelsResponse(openaiModels)
	c.JSON(http.StatusOK, response)
}

// findModel finds a model by name or ID using index for O(1) lookup
func (h *Handler) findModel(modelName string) (string, error) {
	statuses := h.modelMgr.ListStatus()

	// First try exact match with ID
	if status, exists := statuses[modelName]; exists && status.State == model.StateLoaded {
		return modelName, nil
	}

	// 使用索引查找模型
	if m, ok := h.modelIndex.Find(modelName); ok {
		// 检查模型是否已加载
		if status, exists := statuses[m.ID]; exists && status.State == model.StateLoaded {
			return m.ID, nil
		}
	}

	return "", fmt.Errorf("model not found: %s", modelName)
}

// getModelPort returns the port for a loaded model
func (h *Handler) getModelPort(modelID string) (int, error) {
	status, exists := h.modelMgr.GetStatus(modelID)
	if !exists {
		return 0, fmt.Errorf("model not loaded: %s", modelID)
	}

	if status.State != model.StateLoaded {
		return 0, fmt.Errorf("model not in loaded state: %s", modelID)
	}

	if status.Port == 0 {
		return 0, fmt.Errorf("model port not available: %s", modelID)
	}

	return status.Port, nil
}

// forwardRequest forwards a non-streaming request to llama.cpp
func (h *Handler) forwardRequest(c *gin.Context, modelID string, port int, path string, req interface{}) {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewReader(body))
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	// Send request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		h.sendError(c, http.StatusBadGateway, "model_error", err.Error(), "")
		logger.Errorf("转发请求到 llama.cpp 失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	// Forward response
	c.Header("Content-Type", "application/json")
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	c.Writer.Write(respBody)
}

// forwardStreamRequest forwards a streaming request to llama.cpp
func (h *Handler) forwardStreamRequest(c *gin.Context, modelID string, port int, path string, req interface{}) {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, bytes.NewReader(body))
	if err != nil {
		h.sendError(c, http.StatusInternalServerError, "server_error", err.Error(), "")
		return
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.Request.Header.Get("Authorization"))

	// Send request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		h.sendError(c, http.StatusBadGateway, "model_error", err.Error(), "")
		logger.Errorf("转发流式请求到 llama.cpp 失败: %v", err)
		return
	}
	defer resp.Body.Close()

	// Set streaming headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.sendError(c, http.StatusInternalServerError, "server_error", "Streaming not supported", "")
		return
	}

	// Stream response
	c.Status(resp.StatusCode)

	reader := bufio.NewReader(resp.Body)

	for {
		// Check if client disconnected
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			logger.Errorf("读取流式响应失败: %v", err)
			return
		}

		// Write line to client
		c.Writer.Write([]byte(line))
		flusher.Flush()
	}
}

// sendError sends an error response
func (h *Handler) sendError(c *gin.Context, statusCode int, errorType, message, param string) {
	response := NewErrorResponse(message, errorType, param, statusCode)
	c.JSON(statusCode, response)
}
