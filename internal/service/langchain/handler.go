// Package langchain provides HTTP handlers for LangChainGo integration
package langchain

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
	"github.com/tmc/langchaingo/llms"
)

// Handler LangChainGo API 处理器
type Handler struct {
	manager *Manager
	log     *logger.Logger
}

// NewHandler 创建新的 LangChainGo API 处理器
func NewHandler(manager *Manager, log *logger.Logger) *Handler {
	return &Handler{
		manager: manager,
		log:     log,
	}
}

// ===== 请求/响应类型 =====

// SimplePromptRequest 简单提示请求
type SimplePromptRequest struct {
	ModelID string                 `json:"model_id" binding:"required"`
	Prompt  string                 `json:"prompt" binding:"required"`
	Input   map[string]interface{} `json:"input,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// ChatPromptRequest 聊天提示请求
type ChatPromptRequest struct {
	ModelID  string                 `json:"model_id" binding:"required"`
	Messages []ChatMessageRequest   `json:"messages" binding:"required,min=1"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// ChatMessageRequest 聊天消息请求
type ChatMessageRequest struct {
	Role    string `json:"role" binding:"required,oneof=system user assistant tool"`
	Content string `json:"content" binding:"required"`
}

// ===== API 端点 =====

// HandleSimplePrompt 处理简单提示请求
// POST /api/langchain/prompt
func (h *Handler) HandleSimplePrompt(c *gin.Context) {
	var req SimplePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Errorf("解析简单提示请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 将请求中的选项转换为 LangChainGo 选项
	var opts []Option
	if temp, ok := req.Options["temperature"].(float64); ok {
		opts = append(opts, WithTemperature(float32(temp)))
	}
	if maxTokens, ok := req.Options["max_tokens"].(float64); ok {
		opts = append(opts, WithMaxTokens(int(maxTokens)))
	}
	if topP, ok := req.Options["top_p"].(float64); ok {
		opts = append(opts, WithTopP(float32(topP)))
	}
	if topK, ok := req.Options["top_k"].(float64); ok {
		opts = append(opts, WithTopK(int(topK)))
	}

	// 生成文本
	response, err := h.manager.SimplePrompt(c.Request.Context(), req.ModelID, req.Prompt, req.Input, opts...)
	if err != nil {
		h.log.Errorf("生成文本失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate text",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"model_id": req.ModelID,
		"response": response,
	})
}

// HandleChatPrompt 处理聊天提示请求
// POST /api/langchain/chat
func (h *Handler) HandleChatPrompt(c *gin.Context) {
	var req ChatPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Errorf("解析聊天提示请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 转换消息格式
	messages := make([]llms.MessageContent, len(req.Messages))
	for i, msg := range req.Messages {
		var role llms.ChatMessageType
		switch msg.Role {
		case "system":
			role = llms.ChatMessageTypeSystem
		case "user":
			role = llms.ChatMessageTypeHuman
		case "assistant":
			role = llms.ChatMessageTypeAI
		case "tool":
			role = llms.ChatMessageTypeTool
		default:
			role = llms.ChatMessageTypeGeneric
		}

		messages[i] = llms.MessageContent{
			Role: role,
			Parts: []llms.ContentPart{
				llms.TextPart(msg.Content),
			},
		}
	}

	// 将请求中的选项转换为 LangChainGo 选项
	var opts []llms.CallOption
	if model, ok := req.Options["model"].(string); ok && model != "" {
		opts = append(opts, llms.WithModel(model))
	}
	if temp, ok := req.Options["temperature"].(float64); ok {
		opts = append(opts, llms.WithTemperature(temp))
	}
	if maxTokens, ok := req.Options["max_tokens"].(float64); ok {
		opts = append(opts, llms.WithMaxTokens(int(maxTokens)))
	}
	if topP, ok := req.Options["top_p"].(float64); ok {
		opts = append(opts, llms.WithTopP(topP))
	}

	// 生成响应
	response, err := h.manager.ChatPrompt(c.Request.Context(), req.ModelID, messages, opts...)
	if err != nil {
		h.log.Errorf("生成聊天响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate chat response",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"model_id": req.ModelID,
		"response": response,
	})
}

// HandleStreamPrompt 处理流式提示请求
// POST /api/langchain/stream
func (h *Handler) HandleStreamPrompt(c *gin.Context) {
	var req ChatPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Errorf("解析流式提示请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 转换消息格式
	messages := make([]llms.MessageContent, len(req.Messages))
	for i, msg := range req.Messages {
		var role llms.ChatMessageType
		switch msg.Role {
		case "system":
			role = llms.ChatMessageTypeSystem
		case "user":
			role = llms.ChatMessageTypeHuman
		case "assistant":
			role = llms.ChatMessageTypeAI
		case "tool":
			role = llms.ChatMessageTypeTool
		default:
			role = llms.ChatMessageTypeGeneric
		}

		messages[i] = llms.MessageContent{
			Role: role,
			Parts: []llms.ContentPart{
				llms.TextPart(msg.Content),
			},
		}
	}

	// 将请求中的选项转换为 LangChainGo 选项
	var opts []llms.CallOption
	if model, ok := req.Options["model"].(string); ok && model != "" {
		opts = append(opts, llms.WithModel(model))
	}
	if temp, ok := req.Options["temperature"].(float64); ok {
		opts = append(opts, llms.WithTemperature(temp))
	}
	if maxTokens, ok := req.Options["max_tokens"].(float64); ok {
		opts = append(opts, llms.WithMaxTokens(int(maxTokens)))
	}
	if topP, ok := req.Options["top_p"].(float64); ok {
		opts = append(opts, llms.WithTopP(topP))
	}

	// 开始流式生成
	respChan, err := h.manager.StreamPrompt(c.Request.Context(), req.ModelID, messages, opts...)
	if err != nil {
		h.log.Errorf("启动流式生成失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to start streaming",
			"details": err.Error(),
		})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no")

	// 流式发送响应
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming not supported",
		})
		return
	}

	for response := range respChan {
		if len(response.Choices) > 0 {
			// 发送 SSE 格式数据
			c.SSEvent("message", response.Choices[0].Content)
			flusher.Flush()
		}
	}

	// 发送结束标记
	c.SSEvent("end", "done")
	flusher.Flush()
}

// HandleListModels 处理列出模型请求
// GET /api/langchain/models
func (h *Handler) HandleListModels(c *gin.Context) {
	models := h.manager.ListModels()

	c.JSON(http.StatusOK, gin.H{
		"models": models,
		"total":  len(models),
	})
}

// HandleGetStats 处理获取统计信息请求
// GET /api/langchain/stats
func (h *Handler) HandleGetStats(c *gin.Context) {
	stats := h.manager.GetStats()

	c.JSON(http.StatusOK, stats)
}

// ===== 路由注册 =====

// RegisterRoutes 注册 LangChainGo API 路由
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	langchain := router.Group("/langchain")
	{
		langchain.POST("/prompt", h.HandleSimplePrompt)
		langchain.POST("/chat", h.HandleChatPrompt)
		langchain.POST("/stream", h.HandleStreamPrompt)
		langchain.GET("/models", h.HandleListModels)
		langchain.GET("/stats", h.HandleGetStats)

		// Chat-specific endpoints
		chat := langchain.Group("/chat")
		{
			chat.GET("/models", h.HandleChatModels)
			chat.POST("/completions", h.HandleChatCompletions)
		}
	}

	h.log.Infof("LangChainGo API 路由已注册: /api/langchain/*")
}

// ===== 错误处理 =====

// ErrorWithDetails 发送带详细信息的错误响应
func ErrorWithDetails(c *gin.Context, code types.ErrorCode, message, details string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   message,
		"code":    string(code),
		"details": details,
	})
}

// ===== Chat-specific endpoints =====

// msgToParts converts a ChatCompletionsMsg.Content (string or []ContentPart) to []llms.ContentPart
func msgToParts(content interface{}) ([]llms.ContentPart, error) {
	switch v := content.(type) {
	case string:
		return []llms.ContentPart{llms.TextPart(v)}, nil
	case []interface{}:
		parts := make([]llms.ContentPart, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid content part: expected object")
			}
			partType, _ := m["type"].(string)
			switch partType {
			case "text":
				text, _ := m["text"].(string)
				parts = append(parts, llms.TextPart(text))
			case "image_url":
				imgMap, _ := m["image_url"].(map[string]interface{})
				url, _ := imgMap["url"].(string)
				parts = append(parts, llms.ImageURLPart(url))
			default:
				return nil, fmt.Errorf("unknown content part type: %s", partType)
			}
		}
		return parts, nil
	default:
		return nil, fmt.Errorf("invalid message content type: expected string or array")
	}
}

// ChatCompletionsRequest OpenAI-compatible chat completion request for the chat UI
type ChatCompletionsRequest struct {
	Model       string                 `json:"model"`
	Messages    []ChatCompletionsMsg   `json:"messages"`
	Stream      bool                   `json:"stream,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
}

// ChatCompletionsMsg message in a chat completion request
type ChatCompletionsMsg struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentPart for multimodal
}

// ContentPart represents a single content part in a multimodal message
type ContentPart struct {
	Type     string    `json:"type"`               // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // for type "text"
	ImageURL *ImageURL `json:"image_url,omitempty"` // for type "image_url"
}

// ImageURL represents an image URL in a content part
type ImageURL struct {
	URL string `json:"url"` // base64 data URL or HTTP URL
}

// HandleChatModels returns all models with their loaded status for the chat UI
// GET /api/langchain/chat/models
func (h *Handler) HandleChatModels(c *gin.Context) {
	models := h.manager.GetChatModels()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"models":  models,
		"total":   len(models),
	})
}

// HandleChatCompletions handles OpenAI-compatible streaming chat completions
// POST /api/langchain/chat/completions
func (h *Handler) HandleChatCompletions(c *gin.Context) {
	var req ChatCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages is required"})
		return
	}

	// Ensure the model is loaded (auto-load if needed)
	port, err := h.manager.EnsureModelLoaded(req.Model)
	if err != nil {
		h.log.Errorf("Failed to ensure model loaded: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load model",
			"details": err.Error(),
		})
		return
	}

	// Build the upstream llama.cpp URL
	upstreamURL := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)

	// Forward the request body, ensuring stream:true
	forwardBody := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature > 0 {
		forwardBody["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		forwardBody["max_tokens"] = req.MaxTokens
	}
	if req.TopP > 0 {
		forwardBody["top_p"] = req.TopP
	}
	if len(req.Stop) > 0 {
		forwardBody["stop"] = req.Stop
	}

	bodyBytes, err := json.Marshal(forwardBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal request"})
		return
	}

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", upstreamURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upstream request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	h.log.Infof("Chat completions: forwarding to %s (model=%s, temp=%.2f, max_tokens=%d, top_p=%.2f)",
		upstreamURL, req.Model, req.Temperature, req.MaxTokens, req.TopP)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to connect to model server",
			"details": err.Error(),
		})
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		h.log.Errorf("Upstream returned %d: %s", httpResp.StatusCode, string(respBody))
		c.JSON(httpResp.StatusCode, gin.H{
			"error":   "Model server error",
			"details": string(respBody),
		})
		return
	}

	h.log.Infof("Upstream responded 200, Content-Type=%s, streaming back to client", httpResp.Header.Get("Content-Type"))

	// Stream the SSE response back to the client as-is (OpenAI format)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	buf := make([]byte, 4096)
	for {
		n, readErr := httpResp.Body.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			flusher.Flush()
		}
		if readErr != nil {
			break
		}
	}

	// Ensure [DONE] is sent
	c.Writer.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

// HandleChatCompletionsNonStream handles non-streaming chat completions
func (h *Handler) HandleChatCompletionsNonStream(c *gin.Context) {
	var req ChatCompletionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	// Ensure model is loaded
	if _, err := h.manager.EnsureModelLoaded(req.Model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to load model",
			"details": err.Error(),
		})
		return
	}

	// Convert messages to langchain format
	messages := make([]llms.MessageContent, len(req.Messages))
	for i, msg := range req.Messages {
		var role llms.ChatMessageType
		switch msg.Role {
		case "system":
			role = llms.ChatMessageTypeSystem
		case "user":
			role = llms.ChatMessageTypeHuman
		case "assistant":
			role = llms.ChatMessageTypeAI
		default:
			role = llms.ChatMessageTypeGeneric
		}
		parts, err := msgToParts(msg.Content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		messages[i] = llms.MessageContent{
			Role:  role,
			Parts: parts,
		}
	}

	// Build options
	var opts []llms.CallOption
	if req.Temperature > 0 {
		opts = append(opts, llms.WithTemperature(req.Temperature))
	}
	if req.MaxTokens > 0 {
		opts = append(opts, llms.WithMaxTokens(req.MaxTokens))
	}

	response, err := h.manager.ChatPrompt(c.Request.Context(), req.Model, messages, opts...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate response",
			"details": err.Error(),
		})
		return
	}

	content := ""
	if len(response.Choices) > 0 {
		content = response.Choices[0].Content
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixMilli()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
	})
}
