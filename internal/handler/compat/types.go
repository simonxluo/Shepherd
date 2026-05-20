// Package compat provides shared types and utilities for OpenAI-compatible API handlers.
package compat

import "strconv"

// ChatCompletionRequest represents a chat completion request (OpenAI-compatible).
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []ChatMessage          `json:"messages"`
	Stream           bool                   `json:"stream,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	TopK             int                    `json:"top_k,omitempty"`
	N                int                    `json:"n,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Seed             int                    `json:"seed,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	RepeatPenalty    float64                `json:"repeat_penalty,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	ResponseFormat   *ResponseFormat        `json:"response_format,omitempty"`
	Tools            []Tool                 `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	Extra            map[string]interface{} `json:"-"`
}

// ChatMessage represents a chat message.
type ChatMessage struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	Name         string        `json:"name,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
}

// FunctionCall represents a function call in a message.
type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ToolCall represents a tool call in a message.
type ToolCall struct {
	ID       string        `json:"id,omitempty"`
	Type     string        `json:"type,omitempty"`
	Function *FunctionCall `json:"function,omitempty"`
}

// Tool represents a tool definition.
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function,omitempty"`
}

// Function represents a function definition within a tool.
type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// ResponseFormat specifies the response format.
type ResponseFormat struct {
	Type string `json:"type,omitempty"`
}

// CompletionRequest represents a legacy completion request.
type CompletionRequest struct {
	Model            string                 `json:"model"`
	Prompt           interface{}            `json:"prompt"`
	Stream           bool                   `json:"stream,omitempty"`
	Suffix           string                 `json:"suffix,omitempty"`
	MaxTokens        int                    `json:"max_tokens,omitempty"`
	Temperature      float64                `json:"temperature,omitempty"`
	TopP             float64                `json:"top_p,omitempty"`
	TopK             int                    `json:"top_k,omitempty"`
	N                int                    `json:"n,omitempty"`
	LogProbs         int                    `json:"logprobs,omitempty"`
	Echo             bool                   `json:"echo,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	FrequencyPenalty float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                `json:"presence_penalty,omitempty"`
	RepeatPenalty    float64                `json:"repeat_penalty,omitempty"`
	Seed             int                    `json:"seed,omitempty"`
	Extra            map[string]interface{} `json:"-"`
}

// ChatCompletionResponse represents a chat completion response.
type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	Choices           []ChatCompletionChoice `json:"choices"`
	Usage             *Usage                 `json:"usage,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
}

// ChatCompletionChoice represents a choice in chat completion.
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason,omitempty"`
	LogProbs     interface{} `json:"logprobs,omitempty"`
}

// CompletionResponse represents a legacy completion response.
type CompletionResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Created           int64              `json:"created"`
	Model             string             `json:"model"`
	Choices           []CompletionChoice `json:"choices"`
	Usage             *Usage             `json:"usage,omitempty"`
	SystemFingerprint string             `json:"system_fingerprint,omitempty"`
}

// CompletionChoice represents a choice in legacy completion.
type CompletionChoice struct {
	Index        int         `json:"index"`
	Text         string      `json:"text"`
	FinishReason string      `json:"finish_reason,omitempty"`
	LogProbs     interface{} `json:"logprobs,omitempty"`
}

// ModelsResponse represents the list of available models.
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model represents an available model.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LogProbs represents log probabilities.
type LogProbs struct {
	Tokens        []string     `json:"tokens,omitempty"`
	TokenLogprobs []float64    `json:"token_logprobs,omitempty"`
	TopLogprobs   []TopLogprob `json:"top_logprobs,omitempty"`
	TextOffset    []int        `json:"text_offset,omitempty"`
}

// TopLogprob represents top log probabilities.
type TopLogprob struct {
	Token   string  `json:"token,omitempty"`
	Logprob float64 `json:"logprob,omitempty"`
	Bytes   []byte  `json:"bytes,omitempty"`
}

// ErrorResponse represents an error response (OpenAI format).
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// EmbeddingRequest represents an embedding request.
type EmbeddingRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"`
}

// EmbeddingResponse represents an embedding response.
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  *Usage          `json:"usage,omitempty"`
}

// EmbeddingData represents a single embedding result.
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// NewModelsResponse creates a new models response.
func NewModelsResponse(models []Model) *ModelsResponse {
	return &ModelsResponse{
		Object: "list",
		Data:   models,
	}
}

// NewErrorResponse creates a new error response.
func NewErrorResponse(message, errorType, param string, statusCode int) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errorType,
			Param:   param,
			Code:    strconv.Itoa(statusCode),
		},
	}
}
