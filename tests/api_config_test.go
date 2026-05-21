package tests

import (
	"net/http"
	"testing"
)

func TestAPIGetConfig(t *testing.T) {
	env := SetupTestServer(t)

	w := DoRequest(env.Engine, http.MethodGet, "/api/config", nil)
	resp := AssertSuccess(t, w)

	// Verify config structure
	if resp.Data["server"] == nil {
		t.Error("expected server config section")
	}
	if resp.Data["storage"] == nil {
		t.Error("expected storage config section")
	}
	if resp.Data["models"] == nil {
		t.Error("expected models config section")
	}
	if resp.Data["node"] == nil {
		t.Error("expected node config section")
	}
	if resp.Data["role"] == nil {
		t.Error("expected role field")
	}

	// Verify server section has ports
	serverCfg, ok := resp.Data["server"].(map[string]interface{})
	if !ok {
		t.Fatal("expected server to be a map")
	}
	if serverCfg["web_port"] == nil {
		t.Error("expected web_port in server config")
	}
}

func TestAPIUpdateConfig(t *testing.T) {
	env := SetupTestServer(t)

	body := map[string]interface{}{
		"auto_scan":  true,
		"scan_paths": []string{"/tmp/models"},
	}

	w := DoRequest(env.Engine, http.MethodPut, "/api/config", body)
	resp := AssertSuccess(t, w)

	if resp.Data["message"] == nil {
		t.Error("expected message in response")
	}
}

func TestAPIUpdateConfigInvalidBody(t *testing.T) {
	env := SetupTestServer(t)

	// Send invalid JSON
	w := DoRequestRaw(env.Engine, http.MethodPut, "/api/config", []byte("not json"))
	AssertStatus(t, w, http.StatusBadRequest)
}
