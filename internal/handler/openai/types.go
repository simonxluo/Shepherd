// Package openai provides OpenAI API compatibility layer
package openai

import (
	"encoding/json"
	"strconv"
)

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model            string                        `json:"model"`
	Messages         []ChatMessage                 `json:"messages"`
	Stream           bool                          `json:"stream,omitempty"`
	Temperature      float64                       `json:"temperature,omitempty"`
	TopP             float64                       `json:"top_p,omitempty"`
	TopK             int                           `json:"top_k,omitempty"`
	N                int                           `json:"n,omitempty"`
	MaxTokens        int                           `json:"max_tokens,omitempty"`
	Seed             int                           `json:"seed,omitempty"`
	FrequencyPenalty float64                       `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64                       `json:"presence_penalty,omitempty"`
	RepeatPenalty    float64                       `json:"repeat_penalty,omitempty"`
	Stop             []string                      `json:"stop,omitempty"`
	ResponseFormat   *ChatCompletionResponseFormat `json:"response_format,omitempty"`
	Extra            map[string]interface{}        `json:"-"`
}

// UnmarshalJSON handles custom JSON unmarshaling for ChatCompletionRequest
func (r *ChatCompletionRequest) UnmarshalJSON(data []byte) error {
	// Create a type alias to avoid recursion
	type Alias ChatCompletionRequest
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	// Use a map to capture extra fields
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Unmarshal into the aux struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Capture extra fields
	r.Extra = make(map[string]interface{})
	knownFields := map[string]bool{
		"model": true, "messages": true, "stream": true,
		"temperature": true, "top_p": true, "top_k": true,
		"n": true, "max_tokens": true, "seed": true,
		"frequency_penalty": true, "presence_penalty": true,
		"repeat_penalty": true, "stop": true, "response_format": true,
	}

	for key, value := range raw {
		if !knownFields[key] {
			var extra interface{}
			if err := json.Unmarshal(value, &extra); err == nil {
				r.Extra[key] = extra
			}
		}
	}

	return nil
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	Name         string        `json:"name,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ToolCall represents a tool call
type ToolCall struct {
	ID       string        `json:"id,omitempty"`
	Type     string        `json:"type,omitempty"`
	Function *FunctionCall `json:"function,omitempty"`
}

// ChatCompletionResponseFormat specifies the format
type ChatCompletionResponseFormat struct {
	Type string `json:"type,omitempty"`
}

// CompletionRequest represents a legacy completion request
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

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	Choices           []ChatCompletionChoice `json:"choices"`
	Usage             *Usage                 `json:"usage,omitempty"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
}

// ChatCompletionChoice represents a choice in chat completion
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason,omitempty"`
	LogProbs     *LogProbs   `json:"logprobs,omitempty"`
}

// CompletionResponse represents a legacy completion response
type CompletionResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Created           int64              `json:"created"`
	Model             string             `json:"model"`
	Choices           []CompletionChoice `json:"choices"`
	Usage             *Usage             `json:"usage,omitempty"`
	SystemFingerprint string             `json:"system_fingerprint,omitempty"`
}

// CompletionChoice represents a choice in legacy completion
type CompletionChoice struct {
	Index        int       `json:"index"`
	Text         string    `json:"text"`
	FinishReason string    `json:"finish_reason,omitempty"`
	LogProbs     *LogProbs `json:"logprobs,omitempty"`
}

// ModelsResponse represents the list of available models
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model represents an available model
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LogProbs represents log probabilities
type LogProbs struct {
	Tokens        []string     `json:"tokens,omitempty"`
	TokenLogprobs []float64    `json:"token_logprobs,omitempty"`
	TopLogprobs   []TopLogprob `json:"top_logprobs,omitempty"`
	TextOffset    []int        `json:"text_offset,omitempty"`
}

// TopLogprob represents top log probabilities
type TopLogprob struct {
	Token   string  `json:"token,omitempty"`
	Logprob float64 `json:"logprob,omitempty"`
	Bytes   []byte  `json:"bytes,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// NewModelsResponse creates a new models response
func NewModelsResponse(models []Model) *ModelsResponse {
	return &ModelsResponse{
		Object: "list",
		Data:   models,
	}
}

// NewErrorResponse creates a new error response
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
