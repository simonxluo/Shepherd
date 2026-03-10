// Package langchain provides HTTP handlers for LangChainGo integration
package langchain

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/types"
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
