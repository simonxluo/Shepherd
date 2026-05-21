package tests

import (
	"fmt"
	"net/http"
	"testing"
)

func TestAPIConversationsCRUD(t *testing.T) {
	env := SetupTestServer(t)

	// Create conversation
	createBody := map[string]interface{}{
		"model": "test-model",
		"title": "Integration Test Conversation",
	}
	w := DoRequest(env.Engine, http.MethodPost, "/api/conversations", createBody)
	resp := AssertSuccess(t, w)

	convData, ok := resp.Data["conversation"].(map[string]interface{})
	if !ok {
		t.Fatal("expected conversation object in response")
	}
	convID, ok := convData["id"].(string)
	if !ok || convID == "" {
		t.Fatal("expected conversation id to be a non-empty string")
	}

	// List conversations
	w = DoRequest(env.Engine, http.MethodGet, "/api/conversations", nil)
	resp = AssertSuccess(t, w)

	items, ok := resp.Data["items"].([]interface{})
	if !ok {
		t.Fatal("expected items to be an array")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(items))
	}

	// Get single conversation
	w = DoRequest(env.Engine, http.MethodGet, fmt.Sprintf("/api/conversations/%s", convID), nil)
	resp = AssertSuccess(t, w)

	gotConv, ok := resp.Data["conversation"].(map[string]interface{})
	if !ok {
		t.Fatal("expected conversation object")
	}
	if gotConv["title"] != "Integration Test Conversation" {
		t.Errorf("expected title 'Integration Test Conversation', got '%v'", gotConv["title"])
	}

	// Update conversation
	updateBody := map[string]interface{}{
		"title": "Updated Title",
	}
	w = DoRequest(env.Engine, http.MethodPut, fmt.Sprintf("/api/conversations/%s", convID), updateBody)
	resp = AssertSuccess(t, w)

	updatedConv, ok := resp.Data["conversation"].(map[string]interface{})
	if !ok {
		t.Fatal("expected conversation object after update")
	}
	if updatedConv["title"] != "Updated Title" {
		t.Errorf("expected updated title, got '%v'", updatedConv["title"])
	}

	// Create message in conversation
	msgBody := map[string]interface{}{
		"role":    "user",
		"content": "Hello from integration test",
	}
	w = DoRequest(env.Engine, http.MethodPost, fmt.Sprintf("/api/conversations/%s/messages", convID), msgBody)
	resp = AssertSuccess(t, w)

	msgData, ok := resp.Data["message"].(map[string]interface{})
	if !ok {
		t.Fatal("expected message object in response")
	}
	if msgData["content"] != "Hello from integration test" {
		t.Errorf("expected message content, got '%v'", msgData["content"])
	}

	// Delete conversation
	w = DoRequest(env.Engine, http.MethodDelete, fmt.Sprintf("/api/conversations/%s", convID), nil)
	AssertSuccess(t, w)

	// Verify deletion
	w = DoRequest(env.Engine, http.MethodGet, fmt.Sprintf("/api/conversations/%s", convID), nil)
	AssertStatus(t, w, http.StatusNotFound)
}

func TestAPIConversationNotFound(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/conversations/nonexistent-id", nil)
	AssertStatus(t, w, http.StatusNotFound)
}

func TestAPICreateConversationMissingModel(t *testing.T) {
	env := SetupTestServer(t)

	// model is required
	body := map[string]interface{}{
		"title": "No Model",
	}
	w := DoRequest(env.Engine, http.MethodPost, "/api/conversations", body)
	AssertStatus(t, w, http.StatusBadRequest)
}

func TestAPICreateMessageInvalidRole(t *testing.T) {
	env := SetupTestServer(t)

	// Create a conversation first
	createBody := map[string]interface{}{
		"model": "test-model",
		"title": "Test",
	}
	w := DoRequest(env.Engine, http.MethodPost, "/api/conversations", createBody)
	resp := AssertSuccess(t, w)
	convData := resp.Data["conversation"].(map[string]interface{})
	convID := convData["id"].(string)

	// Send message with invalid role
	msgBody := map[string]interface{}{
		"role":    "invalid-role",
		"content": "test",
	}
	w = DoRequest(env.Engine, http.MethodPost, fmt.Sprintf("/api/conversations/%s/messages", convID), msgBody)
	AssertStatus(t, w, http.StatusBadRequest)
}
