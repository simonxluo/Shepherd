package paths

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandler(t *testing.T) (*Handler, *config.Manager, func()) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.config.yaml")

	cfg := config.DefaultConfig()
	cfg.Llamacpp.Paths = []config.LlamacppPath{}
	cfg.Model.PathConfigs = []config.ModelPath{}

	cfgMgr := config.NewManager()
	require.NotNil(t, cfgMgr)

	handler := NewHandler(cfgMgr)

	cleanup := func() {
		os.Remove(configPath)
	}

	return handler, cfgMgr, cleanup
}

func TestNewHandler(t *testing.T) {
	_, cfgMgr, cleanup := setupTestHandler(t)
	defer cleanup()

	handler := NewHandler(cfgMgr)
	assert.NotNil(t, handler)
	assert.Equal(t, cfgMgr, handler.configManager)
}

func TestHandler_GetLlamaCppPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/config/llamacpp/paths", handler.GetLlamaCppPaths)

	req := httptest.NewRequest("GET", "/api/config/llamacpp/paths", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.NotNil(t, data["items"])
}

func TestHandler_AddLlamaCppPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	tempLlamaPath := t.TempDir()

	router := gin.New()
	router.POST("/api/config/llamacpp/paths", handler.AddLlamaCppPath)

	tests := []struct {
		name        string
		reqBody     map[string]interface{}
		wantStatus  int
		wantSuccess bool
	}{
		{
			name: "valid path",
			reqBody: map[string]interface{}{
				"path":        tempLlamaPath,
				"name":        "Default",
				"description": "Default llama.cpp installation",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "empty path",
			reqBody: map[string]interface{}{
				"path": "",
				"name": "Empty",
			},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:        "invalid request body",
			reqBody:     map[string]interface{}{},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/api/config/llamacpp/paths", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tt.wantSuccess, resp["success"])
		})
	}
}

func TestHandler_UpdateLlamaCppPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	tempOldPath := t.TempDir()

	// First add a path via POST
	addRouter := gin.New()
	addRouter.POST("/api/config/llamacpp/paths", handler.AddLlamaCppPath)

	addBody, _ := json.Marshal(map[string]interface{}{
		"path":        tempOldPath,
		"name":        "OldLlamaPath",
		"description": "Original llama.cpp path",
	})
	addReq := httptest.NewRequest("POST", "/api/config/llamacpp/paths", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	addRouter.ServeHTTP(addW, addReq)

	// Verify add succeeded
	require.Equal(t, 200, addW.Code)

	// Now test update
	router := gin.New()
	router.PUT("/api/config/llamacpp/paths", handler.UpdateLlamaCppPath)

	tests := []struct {
		name        string
		reqBody     map[string]interface{}
		wantStatus  int
		wantSuccess bool
	}{
		{
			name: "update existing path with originalPath",
			reqBody: map[string]interface{}{
				"originalPath": tempOldPath,
				"path":         tempOldPath,
				"name":         "UpdatedLlamaPath",
				"description":  "Updated llama.cpp path",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "update existing path by name",
			reqBody: map[string]interface{}{
				"path":        tempOldPath,
				"name":        "UpdatedLlamaPath",
				"description": "Updated by name",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "update non-existent path",
			reqBody: map[string]interface{}{
				"path": t.TempDir(),
				"name": "NonExistent",
			},
			wantStatus:  http.StatusNotFound,
			wantSuccess: false,
		},
		{
			name: "empty path",
			reqBody: map[string]interface{}{
				"name": "NoPath",
			},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("PUT", "/api/config/llamacpp/paths", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tt.wantSuccess, resp["success"])
		})
	}
}

func TestHandler_RemoveLlamaCppPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, cfgMgr, cleanup := setupTestHandler(t)
	defer cleanup()

	// 先添加一个路径
	cfg := cfgMgr.Get()
	cfg.Llamacpp.Paths = append(cfg.Llamacpp.Paths, config.LlamacppPath{
		Path: "/test/path",
		Name: "Test",
	})
	cfgMgr.Save(cfg)

	router := gin.New()
	router.DELETE("/api/config/llamacpp/paths", handler.RemoveLlamaCppPath)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantSuccess bool
	}{
		{
			name:        "remove existing path",
			path:        "/test/path",
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "remove non-existent path",
			path:        "/nonexistent/path",
			wantStatus:  http.StatusNotFound,
			wantSuccess: false,
		},
		{
			name:        "empty path parameter",
			path:        "",
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/config/llamacpp/paths"
			if tt.path != "" {
				url += "?path=" + tt.path
			}

			req := httptest.NewRequest("DELETE", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tt.wantSuccess, resp["success"])
		})
	}
}

func TestHandler_TestLlamaCppPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.POST("/api/config/llamacpp/test", handler.TestLlamaCppPath)

	tests := []struct {
		name       string
		reqBody    map[string]string
		wantStatus int
		wantValid  bool
	}{
		{
			name:       "test existing directory",
			reqBody:    map[string]string{"path": "/tmp"},
			wantStatus: http.StatusOK,
			wantValid:  true,
		},
		{
			name:       "test non-existent path",
			reqBody:    map[string]string{"path": "/nonexistent/path/12345"},
			wantStatus: http.StatusOK,
			wantValid:  false,
		},
		{
			name:       "empty path",
			reqBody:    map[string]string{"path": ""},
			wantStatus: http.StatusBadRequest,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/api/config/llamacpp/test", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			if tt.wantStatus == http.StatusOK {
				assert.True(t, resp["success"].(bool))
				// 新响应格式: data 字段包含实际数据
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, tt.wantValid, data["valid"])
			} else {
				assert.False(t, resp["success"].(bool))
			}
		})
	}
}

func TestHandler_GetModelPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/config/models/paths", handler.GetModelPaths)

	req := httptest.NewRequest("GET", "/api/config/models/paths", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	require.NotNil(t, resp["success"], "response should have success field, body: %s", w.Body.String())
	assert.True(t, resp["success"].(bool))
	require.NotNil(t, resp["data"])
	data := resp["data"].(map[string]interface{})
	_ = data["items"]
}

func TestHandler_AddModelPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	tempModelPath := t.TempDir()

	router := gin.New()
	router.POST("/api/config/models/paths", handler.AddModelPath)

	tests := []struct {
		name        string
		reqBody     map[string]interface{}
		wantStatus  int
		wantSuccess bool
	}{
		{
			name: "valid path",
			reqBody: map[string]interface{}{
				"path":        tempModelPath,
				"name":        "Models",
				"description": "Model directory",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "empty path",
			reqBody: map[string]interface{}{
				"path": "",
				"name": "Empty",
			},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/api/config/models/paths", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Debug: print response for 500 errors
			if w.Code != tt.wantStatus {
				t.Logf("Status: %d (expected %d)", w.Code, tt.wantStatus)
				t.Logf("Response: %s", w.Body.String())
			}

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tt.wantSuccess, resp["success"])
		})
	}
}

func TestHandler_UpdateModelPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	tempOldPath := t.TempDir()

	// First add a path via POST
	addRouter := gin.New()
	addRouter.POST("/api/config/models/paths", handler.AddModelPath)

	addBody, _ := json.Marshal(map[string]interface{}{
		"path":        tempOldPath,
		"name":        "OldPath",
		"description": "Original description",
	})
	addReq := httptest.NewRequest("POST", "/api/config/models/paths", bytes.NewReader(addBody))
	addReq.Header.Set("Content-Type", "application/json")
	addW := httptest.NewRecorder()
	addRouter.ServeHTTP(addW, addReq)

	// Verify add succeeded
	require.Equal(t, 200, addW.Code)

	// Now test update
	router := gin.New()
	router.PUT("/api/config/models/paths", handler.UpdateModelPath)

	tests := []struct {
		name        string
		reqBody     map[string]interface{}
		wantStatus  int
		wantSuccess bool
	}{
		{
			name: "update existing path",
			reqBody: map[string]interface{}{
				"originalPath": tempOldPath, // Now we include originalPath
				"path":         tempOldPath,
				"name":         "UpdatedPath",
				"description":  "Updated description",
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "update non-existent path",
			reqBody: map[string]interface{}{
				"path": t.TempDir(),
				"name": "NonExistent",
			},
			wantStatus:  http.StatusNotFound,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("PUT", "/api/config/models/paths", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Print response for debugging
			t.Logf("Status: %d, Body: %s", w.Code, w.Body.String())

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tt.wantSuccess, resp["success"])
		})
	}
}

func TestHandler_RemoveModelPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, cfgMgr, cleanup := setupTestHandler(t)
	defer cleanup()

	// 先添加一个路径
	cfg := cfgMgr.Get()
	cfg.Model.PathConfigs = append(cfg.Model.PathConfigs, config.ModelPath{
		Path: "/test/model/path",
		Name: "TestModels",
	})
	cfgMgr.Save(cfg)

	router := gin.New()
	router.DELETE("/api/config/models/paths", handler.RemoveModelPath)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantSuccess bool
	}{
		{
			name:        "remove existing path",
			path:        "/test/model/path",
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "remove non-existent path",
			path:        "/nonexistent/path",
			wantStatus:  http.StatusNotFound,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/config/models/paths"
			if tt.path != "" {
				url += "?path=" + tt.path
			}

			req := httptest.NewRequest("DELETE", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tt.wantSuccess, resp["success"])
		})
	}
}

func TestHandler_validateAndNormalizePath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid existing directory",
			path:    "/tmp",
			wantErr: false,
		},
		{
			name:    "non-existent path",
			path:    "/nonexistent/path/12345",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "file instead of directory",
			path:    "/etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.validateAndNormalizePath(tt.path)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.True(t, filepath.IsAbs(result))
			}
		})
	}
}
