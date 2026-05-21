package tests

import (
	"net/http"
	"testing"
)

func TestCompatOpenAIModels(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/v1/models", nil)
	AssertStatus(t, w, http.StatusOK)

	// OpenAI models endpoint returns a non-standard format (OpenAI API format)
	resp := ParseRawResponse(t, w)

	// OpenAI format: {"object": "list", "data": [...] or null}
	if resp["object"] != "list" {
		t.Errorf("expected object='list', got '%v'", resp["object"])
	}

	// data can be null (nil slice serializes to null) or an empty array
	if data, ok := resp["data"].([]interface{}); ok {
		// No models loaded, so should be empty
		if len(data) != 0 {
			t.Logf("found %d models in OpenAI format", len(data))
		}
	} else if resp["data"] != nil {
		t.Errorf("expected data to be null or array, got %T", resp["data"])
	}
	// data == nil is acceptable (no loaded models)
}

func TestCompatOpenAIChatCompletionsNoModel(t *testing.T) {
	env := SetupTestServer(t)

	body := map[string]interface{}{
		"model": "nonexistent-model",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	w := DoRequest(env.Engine, http.MethodPost, "/v1/chat/completions", body)

	// Should fail because the model doesn't exist
	if w.Code == http.StatusOK {
		t.Error("expected non-200 status for nonexistent model")
	}
}
