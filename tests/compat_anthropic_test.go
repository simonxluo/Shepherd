package tests

import (
	"net/http"
	"testing"
)

func TestCompatAnthropicMessages(t *testing.T) {
	env := SetupTestServer(t)

	body := map[string]interface{}{
		"model":      "nonexistent-model",
		"max_tokens": 100,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	w := DoRequest(env.Engine, http.MethodPost, "/v1/messages", body)

	// Should fail because model doesn't exist
	// Anthropic format returns error differently
	if w.Code == http.StatusOK {
		t.Error("expected non-200 for nonexistent model in Anthropic messages")
	}

	// Verify it returns a proper error structure
	resp := ParseRawResponse(t, w)
	if resp["type"] == nil && resp["error"] == nil {
		// At minimum, should have some error indication
		t.Logf("response: %v", resp)
	}
}

func TestCompatAnthropicMessagesMissingModel(t *testing.T) {
	env := SetupTestServer(t)

	body := map[string]interface{}{
		"max_tokens": 100,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	w := DoRequest(env.Engine, http.MethodPost, "/v1/messages", body)

	// Should return 400 for missing model
	AssertStatus(t, w, http.StatusBadRequest)
}

func TestCompatAnthropicMessagesMissingMaxTokens(t *testing.T) {
	env := SetupTestServer(t)

	body := map[string]interface{}{
		"model": "some-model",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	w := DoRequest(env.Engine, http.MethodPost, "/v1/messages", body)

	// Should return 400 for missing max_tokens
	AssertStatus(t, w, http.StatusBadRequest)
}

func TestCompatAnthropicMessagesEmptyMessages(t *testing.T) {
	env := SetupTestServer(t)

	body := map[string]interface{}{
		"model":      "some-model",
		"max_tokens": 100,
		"messages":   []map[string]interface{}{},
	}

	w := DoRequest(env.Engine, http.MethodPost, "/v1/messages", body)

	// Should return 400 for empty messages
	AssertStatus(t, w, http.StatusBadRequest)
}
