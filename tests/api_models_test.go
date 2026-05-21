package tests

import (
	"net/http"
	"testing"
)

func TestAPIListModels(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/models", nil)
	resp := AssertSuccess(t, w)

	// Models should be an array (empty since no model dirs configured)
	models, ok := resp.Data["models"].([]interface{})
	if !ok {
		t.Fatal("expected models to be an array")
	}
	// In test env, no models are scanned so it should be empty
	if len(models) != 0 {
		t.Logf("found %d models (expected 0 in test env)", len(models))
	}

	total, ok := resp.Data["total"].(float64)
	if !ok {
		t.Fatal("expected total to be a number")
	}
	if int(total) != len(models) {
		t.Errorf("total mismatch: total=%v, len(models)=%d", total, len(models))
	}
}

func TestAPIListLoadedModels(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/models/loaded", nil)
	resp := AssertSuccess(t, w)

	models, ok := resp.Data["models"].([]interface{})
	if !ok {
		t.Fatal("expected models to be an array")
	}
	if len(models) != 0 {
		t.Errorf("expected 0 loaded models, got %d", len(models))
	}
}
