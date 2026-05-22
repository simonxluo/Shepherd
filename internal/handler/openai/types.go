// Package openai provides OpenAI API compatibility layer
package openai

import (
	"encoding/json"

	"github.com/simonxluo/Shepherd/internal/handler/compat"
)

// Re-export compat types for backward compatibility within the package and swagger docs.
type ChatCompletionRequest = compat.ChatCompletionRequest
type ChatMessage = compat.ChatMessage
type FunctionCall = compat.FunctionCall
type ToolCall = compat.ToolCall
type Tool = compat.Tool
type Function = compat.Function
type ChatCompletionResponseFormat = compat.ResponseFormat
type CompletionRequest = compat.CompletionRequest
type ChatCompletionResponse = compat.ChatCompletionResponse
type ChatCompletionChoice = compat.ChatCompletionChoice
type CompletionResponse = compat.CompletionResponse
type CompletionChoice = compat.CompletionChoice
type ModelsResponse = compat.ModelsResponse
type Model = compat.Model
type Usage = compat.Usage
type LogProbs = compat.LogProbs
type TopLogprob = compat.TopLogprob
type ErrorResponse = compat.ErrorResponse
type ErrorDetail = compat.ErrorDetail

var NewModelsResponse = compat.NewModelsResponse
var NewErrorResponse = compat.NewErrorResponse

// ChatCompletionRequestWithExtra wraps ChatCompletionRequest with custom JSON unmarshaling
// to capture extra/unknown fields.
type ChatCompletionRequestWithExtra struct {
	compat.ChatCompletionRequest
}

// UnmarshalJSON handles custom JSON unmarshaling for ChatCompletionRequestWithExtra
func (r *ChatCompletionRequestWithExtra) UnmarshalJSON(data []byte) error {
	// Create a type alias to avoid recursion
	type Alias compat.ChatCompletionRequest
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(&r.ChatCompletionRequest),
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
		"tools": true, "tool_choice": true,
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
