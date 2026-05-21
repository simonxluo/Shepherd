package tests

import (
	"net/http"
	"testing"
)

func TestCompatOllamaTags(t *testing.T) {
	env := SetupTestServer(t)

	// Ollama uses POST /api/tags
	w := DoRequest(env.Engine, http.MethodPost, "/api/tags", nil)
	AssertStatus(t, w, http.StatusOK)

	resp := ParseRawResponse(t, w)

	// Ollama format: {"models": [...]}
	models, ok := resp["models"].([]interface{})
	if !ok {
		t.Fatal("expected models to be an array in Ollama format")
	}
	// No models loaded
	if len(models) != 0 {
		t.Logf("found %d models in Ollama format", len(models))
	}
}

func TestCompatOllamaChatNoModel(t *testing.T) {
	env := SetupTestServer(t)

	body := map[string]interface{}{
		"model": "nonexistent-model",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	w := DoRequest(env.Engine, http.MethodPost, "/api/chat", body)

	// Should fail because model doesn't exist
	if w.Code == http.StatusOK {
		t.Error("expected non-200 for nonexistent model in Ollama chat")
	}
}
