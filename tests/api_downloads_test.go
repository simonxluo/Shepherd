package tests

import (
	"net/http"
	"testing"
)

func TestAPIListDownloads(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/downloads", nil)
	resp := AssertSuccess(t, w)

	downloads, ok := resp.Data["downloads"].([]interface{})
	if !ok {
		t.Fatal("expected downloads to be an array")
	}
	// No downloads active
	if len(downloads) != 0 {
		t.Errorf("expected 0 downloads, got %d", len(downloads))
	}

	total, ok := resp.Data["total"].(float64)
	if !ok {
		t.Fatal("expected total to be a number")
	}
	if total != 0 {
		t.Errorf("expected total=0, got %v", total)
	}
}

func TestAPIGetDownloadNotFound(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/downloads/nonexistent-id", nil)
	AssertStatus(t, w, http.StatusNotFound)
}
