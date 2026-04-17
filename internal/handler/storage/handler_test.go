package storage

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
	"github.com/shepherd-project/shepherd/Shepherd/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandler(t *testing.T) (*Handler, *config.Manager, *storage.Manager, func()) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.config.yaml")

	cfgMgr := config.NewManager()

	storageCfg := &storage.StorageConfig{
		Type: storage.StorageTypeMemory,
	}
	storageMgr, err := storage.NewManager(storageCfg)
	require.NoError(t, err)

	handler := NewHandler(cfgMgr, storageMgr)

	cleanup := func() {
		storageMgr.Close()
		os.Remove(configPath)
	}

	return handler, cfgMgr, storageMgr, cleanup
}

func TestNewHandler(t *testing.T) {
	handler, _, _, cleanup := setupTestHandler(t)
	defer cleanup()

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.configManager)
	assert.NotNil(t, handler.storageMgr)
}

func TestHandler_GetStorageConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/storage/config", handler.GetStorageConfig)

	req := httptest.NewRequest("GET", "/api/storage/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
}

func TestHandler_UpdateStorageConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.PUT("/api/storage/config", handler.UpdateStorageConfig)

	tests := []struct {
		name        string
		reqBody     interface{}
		wantStatus  int
		wantSuccess bool
	}{
		{
			name: "valid memory config",
			reqBody: storage.StorageConfig{
				Type: storage.StorageTypeMemory,
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "valid sqlite config",
			reqBody: storage.StorageConfig{
				Type:   storage.StorageTypeSQLite,
				SQLite: &storage.SQLiteConfig{Path: "/tmp/test.db"},
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "invalid storage type",
			reqBody:     storage.StorageConfig{Type: "invalid"},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:        "sqlite without config",
			reqBody:     storage.StorageConfig{Type: storage.StorageTypeSQLite},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:        "invalid json",
			reqBody:     "invalid",
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("PUT", "/api/storage/config", bytes.NewReader(body))
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

func TestHandler_GetStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/storage/stats", handler.GetStats)

	req := httptest.NewRequest("GET", "/api/storage/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	assert.NotNil(t, resp["data"])
}

func TestHandler_GetConversations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/storage/conversations", handler.GetConversations)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{
			name:       "default pagination",
			query:      "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom limit",
			query:      "?limit=50",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom limit and offset",
			query:      "?limit=10&offset=20",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/storage/conversations"+tt.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.True(t, resp["success"].(bool))
		})
	}
}

func TestHandler_GetConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, storageMgr, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/storage/conversations/:id", handler.GetConversation)

	tests := []struct {
		name       string
		id         string
		setup      func()
		wantStatus int
	}{
		{
			name:       "empty id",
			id:         "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non-existent conversation",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "existing conversation",
			id:   "test-conv-1",
			setup: func() {
				store := storageMgr.GetStore()
				store.CreateConversation(nil, &storage.Conversation{
					ID:    "test-conv-1",
					Model: "test-model",
					Title: "Test Conversation",
				})
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			url := "/api/storage/conversations/" + tt.id
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestHandler_DeleteConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, _, storageMgr, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.New()
	router.DELETE("/api/storage/conversations/:id", handler.DeleteConversation)

	tests := []struct {
		name       string
		id         string
		setup      func()
		wantStatus int
	}{
		{
			name:       "empty id",
			id:         "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non-existent conversation",
			id:         "non-existent",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "existing conversation",
			id:   "test-conv-1",
			setup: func() {
				store := storageMgr.GetStore()
				store.CreateConversation(nil, &storage.Conversation{
					ID:    "test-conv-1",
					Model: "test-model",
					Title: "Test Conversation",
				})
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			url := "/api/storage/conversations/" + tt.id
			req := httptest.NewRequest("DELETE", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestParseQueryParam(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		min     int
		max     int
		want    int
		wantErr bool
	}{
		{
			name:    "valid value",
			input:   "50",
			min:     1,
			max:     100,
			want:    50,
			wantErr: false,
		},
		{
			name:    "below min",
			input:   "0",
			min:     10,
			max:     100,
			want:    10,
			wantErr: false,
		},
		{
			name:    "above max",
			input:   "200",
			min:     1,
			max:     100,
			want:    100,
			wantErr: false,
		},
		{
			name:    "invalid input",
			input:   "abc",
			min:     1,
			max:     100,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseQueryParam(tt.input, tt.min, tt.max)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
