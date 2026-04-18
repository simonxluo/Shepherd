package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelStatusTokenCounting(t *testing.T) {
	status := &ModelStatus{ID: "test-model"}

	status.AddTokens(10, 20)
	p, c := status.GetTokenCounts()
	assert.Equal(t, int64(10), p)
	assert.Equal(t, int64(20), c)

	status.AddTokens(5, 15)
	p, c = status.GetTokenCounts()
	assert.Equal(t, int64(15), p)
	assert.Equal(t, int64(35), c)
}

func TestExtractTokenUsage(t *testing.T) {
	mgr := &Manager{
		statuses: make(map[string]*ModelStatus),
	}
	status := &ModelStatus{ID: "test-model"}
	mgr.statuses["test-model"] = status

	body, _ := json.Marshal(map[string]interface{}{
		"id":      "chatcmpl-123",
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   "test-model",
		"choices": []map[string]interface{}{},
		"usage": map[string]int{
			"prompt_tokens":     50,
			"completion_tokens": 100,
			"total_tokens":      150,
		},
	})

	extractTokenUsage(mgr, "test-model", body)

	p, c := status.GetTokenCounts()
	assert.Equal(t, int64(50), p)
	assert.Equal(t, int64(100), c)
}

func TestExtractTokenUsageNoUsage(t *testing.T) {
	mgr := &Manager{
		statuses: make(map[string]*ModelStatus),
	}
	status := &ModelStatus{ID: "test-model"}
	mgr.statuses["test-model"] = status

	body, _ := json.Marshal(map[string]interface{}{
		"id":      "chatcmpl-123",
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   "test-model",
		"choices": []map[string]interface{}{},
	})

	extractTokenUsage(mgr, "test-model", body)

	p, c := status.GetTokenCounts()
	assert.Equal(t, int64(0), p)
	assert.Equal(t, int64(0), c)
}

func TestExtractTokenUsageUnknownModel(t *testing.T) {
	mgr := &Manager{
		statuses: make(map[string]*ModelStatus),
	}

	body, _ := json.Marshal(map[string]interface{}{
		"usage": map[string]int{
			"prompt_tokens":     50,
			"completion_tokens": 100,
		},
	})

	extractTokenUsage(mgr, "nonexistent", body)
}

func TestGetModelTokenCounts(t *testing.T) {
	mgr := &Manager{
		statuses: make(map[string]*ModelStatus),
	}
	status := &ModelStatus{ID: "test-model"}
	status.AddTokens(100, 200)
	mgr.statuses["test-model"] = status

	p, c, found := mgr.GetModelTokenCounts("test-model")
	assert.True(t, found)
	assert.Equal(t, int64(100), p)
	assert.Equal(t, int64(200), c)

	_, _, found = mgr.GetModelTokenCounts("nonexistent")
	assert.False(t, found)
}

func extractTokenUsage(mgr *Manager, modelID string, body []byte) {
	if status, exists := mgr.GetStatusRef(modelID); exists {
		var resp struct {
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(body, &resp) == nil && resp.Usage != nil {
			status.AddTokens(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		}
	}
}
