package tests

import (
	"net/http"
	"testing"
)

func TestAPIServerInfo(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/info", nil)
	resp := AssertSuccess(t, w)

	// Verify response contains expected fields
	if resp.Data["name"] != "Shepherd" {
		t.Errorf("expected name 'Shepherd', got '%v'", resp.Data["name"])
	}
	if resp.Data["status"] != "running" {
		t.Errorf("expected status 'running', got '%v'", resp.Data["status"])
	}
	if resp.Data["version"] == nil {
		t.Error("expected version to be present")
	}

	// Verify ports
	ports, ok := resp.Data["ports"].(map[string]interface{})
	if !ok {
		t.Fatal("expected ports to be a map")
	}
	if ports["web"] == nil {
		t.Error("expected web port to be present")
	}
}

func TestAPIGetGPUs(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/system/gpus", nil)
	resp := AssertSuccess(t, w)

	// Verify response structure
	if resp.Data["devices"] == nil {
		t.Error("expected devices field")
	}
	if resp.Data["gpus"] == nil {
		t.Error("expected gpus field")
	}

	count, ok := resp.Data["count"].(float64)
	if !ok {
		t.Fatal("expected count to be a number")
	}
	if count < 0 {
		t.Errorf("expected count >= 0, got %v", count)
	}
}

func TestAPIGetLlamacppBackends(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/system/llamacpp-backends", nil)
	resp := AssertSuccess(t, w)

	if resp.Data["backends"] == nil {
		t.Error("expected backends field")
	}
	if resp.Data["inferenceBackends"] == nil {
		t.Error("expected inferenceBackends field")
	}
}
