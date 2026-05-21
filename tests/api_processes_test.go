package tests

import (
	"net/http"
	"testing"
)

func TestAPIListProcesses(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/processes", nil)
	resp := AssertSuccess(t, w)

	// Verify response structure
	stats, ok := resp.Data["stats"].(map[string]interface{})
	if !ok {
		t.Fatal("expected stats to be a map")
	}

	running, ok := stats["running"].(float64)
	if !ok {
		t.Fatal("expected running to be a number")
	}
	if running != 0 {
		t.Errorf("expected 0 running processes, got %v", running)
	}

	loading, ok := stats["loading"].(float64)
	if !ok {
		t.Fatal("expected loading to be a number")
	}
	if loading != 0 {
		t.Errorf("expected 0 loading processes, got %v", loading)
	}

	total, ok := stats["total"].(float64)
	if !ok {
		t.Fatal("expected total to be a number")
	}
	if total != 0 {
		t.Errorf("expected 0 total processes, got %v", total)
	}
}

func TestAPIGetProcessNotFound(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/processes/nonexistent-id", nil)
	AssertStatus(t, w, http.StatusNotFound)
}
