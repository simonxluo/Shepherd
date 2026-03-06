// Package langchain provides LangChainGo integration for Shepherd
// 这个包提供了 LangChainGo 的自定义 LLM 实现，用于与 llama.cpp HTTP server 通信
package langchain

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// LlamaCPP 是 LangChainGo 的 LLM 实现，通过 HTTP 与 llama.cpp server 通信
type LlamaCPP struct {
	client      *http.Client
	serverURL   string // llama.cpp server URL (e.g., "http://127.0.0.1:8080")
	modelID     string // 模型 ID
	temperature float32
	maxTokens   int
	topP        float32
	topK        int
}

// NewLlamaCPP 创建新的 LlamaCPP LLM 实例
func NewLlamaCPP(serverURL, modelID string, opts ...Option) (*LlamaCPP, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("server URL cannot be empty")
	}

	llm := &LlamaCPP{
		client:      &http.Client{},
		serverURL:   strings.TrimRight(serverURL, "/"),
		modelID:     modelID,
		temperature: 0.7,
		maxTokens:   -1, // 使用模型默认值
		topP:        0.9,
		topK:        40,
	}

	// 应用选项
	for _, opt := range opts {
		opt(llm)
	}

	return llm, nil
}

// Option 是 LlamaCPP 的配置选项
type Option func(*LlamaCPP)

// WithTemperature 设置温度参数
func WithTemperature(temp float32) Option {
	return func(l *LlamaCPP) {
		l.temperature = temp
	}
}

// WithMaxTokens 设置最大 token 数
func WithMaxTokens(maxTokens int) Option {
	return func(l *LlamaCPP) {
		l.maxTokens = maxTokens
	}
}

// WithTopP 设置 top_p 参数
func WithTopP(topP float32) Option {
	return func(l *LlamaCPP) {
		l.topP = topP
	}
}

// WithTopK 设置 top_k 参数
func WithTopK(topK int) Option {
	return func(l *LlamaCPP) {
		l.topK = topK
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端
func WithHTTPClient(client *http.Client) Option {
	return func(l *LlamaCPP) {
		l.client = client
	}
}

// Call 实现 llms.Model 接口的简单文本生成方法
func (l *LlamaCPP) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	// 解析选项
	opts := llms.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	// 构建请求
	req := ChatCompletionRequest{
		Model: l.getModel(&opts),
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: l.getTemperature(&opts),
		MaxTokens:   l.getMaxTokens(&opts),
		TopP:        l.getTopP(&opts),
		TopK:        l.getTopK(&opts),
		Stream:      false,
	}

	// 发送请求
	resp, err := l.sendChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to send chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return resp.Choices[0].Message.Content, nil
}

// GenerateContent 实现 llms.Model 接口的高级内容生成方法
func (l *LlamaCPP) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	// 解析选项
	opts := llms.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	// 转换消息格式
	chatMessages := l.convertMessages(messages)

	// 构建请求
	req := ChatCompletionRequest{
		Model:       l.getModel(&opts),
		Messages:    chatMessages,
		Temperature: l.getTemperature(&opts),
		MaxTokens:   l.getMaxTokens(&opts),
		TopP:        l.getTopP(&opts),
		TopK:        l.getTopK(&opts),
		Stream:      false,
	}

	// 发送请求
	resp, err := l.sendChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send chat completion: %w", err)
	}

	// 转换响应
	contentResponse := &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content:    resp.Choices[0].Message.Content,
				StopReason: resp.Choices[0].FinishReason,
			},
		},
	}

	return contentResponse, nil
}

// convertMessages 将 LangChainGo 消息格式转换为 llama.cpp 格式
func (l *LlamaCPP) convertMessages(messages []llms.MessageContent) []ChatMessage {
	chatMessages := make([]ChatMessage, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case llms.ChatMessageTypeSystem:
			chatMessages = append(chatMessages, ChatMessage{
				Role:    "system",
				Content: l.extractTextContent(msg),
			})
		case llms.ChatMessageTypeHuman:
			chatMessages = append(chatMessages, ChatMessage{
				Role:    "user",
				Content: l.extractTextContent(msg),
			})
		case llms.ChatMessageTypeAI:
			chatMessages = append(chatMessages, ChatMessage{
				Role:    "assistant",
				Content: l.extractTextContent(msg),
			})
		case llms.ChatMessageTypeGeneric:
			chatMessages = append(chatMessages, ChatMessage{
				Role:    "user",
				Content: l.extractTextContent(msg),
			})
		case llms.ChatMessageTypeTool:
			// 工具消息可能需要特殊处理
			chatMessages = append(chatMessages, ChatMessage{
				Role:    "user",
				Content: l.extractTextContent(msg),
			})
		}
	}

	return chatMessages
}

// extractTextContent 从 MessageContent 中提取文本内容
func (l *LlamaCPP) extractTextContent(msg llms.MessageContent) string {
	var textParts []string

	for _, part := range msg.Parts {
		if text, ok := part.(llms.TextContent); ok {
			textParts = append(textParts, text.Text)
		}
	}

	return strings.Join(textParts, "\n")
}

// sendChatCompletion 发送聊天完成请求到 llama.cpp server
func (l *LlamaCPP) sendChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	url := fmt.Sprintf("%s/v1/chat/completions", l.serverURL)

	// 序列化请求
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	httpResp, err := l.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer utils.CloseQuietly(httpResp.Body)

	// 读取响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查状态码
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// 解析响应
	var resp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// 辅助方法：从选项中获取参数
func (l *LlamaCPP) getModel(opts *llms.CallOptions) string {
	if opts != nil && opts.Model != "" {
		return opts.Model
	}
	return l.modelID
}

func (l *LlamaCPP) getTemperature(opts *llms.CallOptions) float32 {
	if opts != nil && opts.Temperature != 0 {
		return float32(opts.Temperature)
	}
	return l.temperature
}

func (l *LlamaCPP) getMaxTokens(opts *llms.CallOptions) int {
	if opts != nil && opts.MaxTokens != 0 {
		return opts.MaxTokens
	}
	return l.maxTokens
}

func (l *LlamaCPP) getTopP(opts *llms.CallOptions) float32 {
	if opts != nil && opts.TopP != 0 {
		return float32(opts.TopP)
	}
	return l.topP
}

func (l *LlamaCPP) getTopK(opts *llms.CallOptions) int {
	// LangChainGo 可能没有 TopK 选项
	return l.topK
}

// ===== 请求/响应类型定义 =====

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float32       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	TopP        float32       `json:"top_p,omitempty"`
	TopK        int           `json:"top_k,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []ChatChoice        `json:"choices"`
	Usage   ChatCompletionUsage `json:"usage"`
}

// ChatChoice 聊天选择
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatCompletionUsage token 使用情况
type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ===== 流式生成支持 =====

// GenerateContentStream 实现流式内容生成（可选）
func (l *LlamaCPP) GenerateContentStream(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (<-chan *llms.ContentResponse, error) {
	// 解析选项
	opts := llms.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	// 转换消息格式
	chatMessages := l.convertMessages(messages)

	// 构建请求
	req := ChatCompletionRequest{
		Model:       l.getModel(&opts),
		Messages:    chatMessages,
		Temperature: l.getTemperature(&opts),
		MaxTokens:   l.getMaxTokens(&opts),
		TopP:        l.getTopP(&opts),
		TopK:        l.getTopK(&opts),
		Stream:      true, // 启用流式
	}

	// 创建响应通道
	respChan := make(chan *llms.ContentResponse)

	// 在 goroutine 中处理流式响应
	go func() {
		defer close(respChan)

		err := l.sendStreamingChatCompletion(ctx, req, respChan)
		if err != nil {
			// 发送错误响应
			respChan <- &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{
						Content:    "",
						StopReason: "error",
					},
				},
			}
		}
	}()

	return respChan, nil
}

// sendStreamingChatCompletion 发送流式聊天完成请求
func (l *LlamaCPP) sendStreamingChatCompletion(ctx context.Context, req ChatCompletionRequest, respChan chan<- *llms.ContentResponse) error {
	url := fmt.Sprintf("%s/v1/chat/completions", l.serverURL)

	// 序列化请求
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	httpResp, err := l.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer utils.CloseQuietly(httpResp.Body)

	// 检查状态码
	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("server returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// 读取流式响应
	reader := bufio.NewReader(httpResp.Body)

	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 读取一行
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		// 跳过空行
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析 SSE 格式
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 检查结束标记
		if data == "[DONE]" {
			return nil
		}

		// 解析 JSON
		var streamResp ChatCompletionResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			// 忽略解析错误，继续处理下一行
			continue
		}

		// 发送响应到通道
		if len(streamResp.Choices) > 0 {
			respChan <- &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{
						Content:    streamResp.Choices[0].Message.Content,
						StopReason: streamResp.Choices[0].FinishReason,
					},
				},
			}
		}
	}
}
