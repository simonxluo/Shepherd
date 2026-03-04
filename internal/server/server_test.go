package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
	"github.com/shepherd-project/shepherd/Shepherd/internal/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test server with model manager
func createTestServer(t *testing.T) *Server {
	cfg := config.DefaultConfig()
	configMgr := config.NewManager()
	_, _ = configMgr.Load() // 加载默认配置，忽略错误

	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	// 创建存储管理器
	storageCfg := storage.StorageConfig{
		Type: storage.StorageTypeMemory, // 测试使用内存存储
	}
	storageMgr, err := storage.NewManager(&storageCfg)
	require.NoError(t, err, "无法创建存储管理器")

	modelMgr := model.NewManager(cfg, configMgr, procMgr, portAllocator, storageMgr)

	serverConfig := &Config{
		WebPort:       8080,
		AnthropicPort: 8070,
		OllamaPort:    11434,
		LMStudioPort:  1234,
		Host:          "0.0.0.0",
		ReadTimeout:   60 * time.Second,
		WriteTimeout:  60 * time.Second,
		ServerCfg:     cfg,
		ConfigMgr:     configMgr,
	}

	server, err := NewServer(serverConfig, modelMgr)
	require.NoError(t, err)
	return server
}

func TestNewServer(t *testing.T) {
	server := createTestServer(t)

	assert.NotNil(t, server)
	assert.NotNil(t, server.engine)
	assert.NotNil(t, server.handlers)
	assert.NotNil(t, server.wsMgr)
	assert.NotNil(t, server.modelMgr)
	assert.NotNil(t, server.config)
}

func TestServerHandleServerInfo(t *testing.T) {
	server := createTestServer(t)
	router := server.GetEngine()

	req := httptest.NewRequest("GET", "/api/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "1.0.0", data["version"])
	assert.Equal(t, "Shepherd", data["name"])
	assert.Equal(t, "running", data["status"])

	ports, ok := data["ports"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, float64(8080), ports["web"])
	assert.Equal(t, float64(8070), ports["anthropic"])
}

func TestServerCORSMiddleware(t *testing.T) {
	server := createTestServer(t)
	router := server.GetEngine()

	req := httptest.NewRequest("OPTIONS", "/api/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
}

func TestServerRoutes(t *testing.T) {
	server := createTestServer(t)
	router := server.GetEngine()

	tests := []struct {
		name         string
		method       string
		path         string
		expectedCode int
	}{
		// Server info
		{"Server info", "GET", "/api/info", http.StatusOK},

		// Config routes
		{"Get config", "GET", "/api/config", http.StatusOK},
		{"Update config", "PUT", "/api/config", http.StatusOK},

		// Model routes - 注意：测试中没有预加载模型，所以 test-id 返回错误状态码是正确的
		{"List models", "GET", "/api/models", http.StatusOK},
		{"Get model", "GET", "/api/models/test-id", http.StatusNotFound},                       // 模型不存在
		{"Load model", "POST", "/api/models/test-id/load", http.StatusInternalServerError},     // 模型不存在时加载失败
		{"Unload model", "POST", "/api/models/test-id/unload", http.StatusInternalServerError}, // 模型不存在时卸载失败
		{"Set alias", "PUT", "/api/models/test-id/alias", http.StatusBadRequest},               // 缺少请求体
		{"Set favourite", "PUT", "/api/models/test-id/favourite", http.StatusBadRequest},       // 缺少请求体

		// Scan routes
		{"Scan models", "POST", "/api/model/scan", http.StatusOK},
		{"Scan status", "GET", "/api/model/scan/status", http.StatusOK},

		// Download routes
		{"List downloads", "GET", "/api/downloads", http.StatusOK},
		{"Create download", "POST", "/api/downloads", http.StatusCreated},                 // 201 Created
		{"Get download", "GET", "/api/downloads/test-id", http.StatusNotFound},            // 下载不存在
		{"Pause download", "POST", "/api/downloads/test-id/pause", http.StatusNotFound},   // 下载不存在
		{"Resume download", "POST", "/api/downloads/test-id/resume", http.StatusNotFound}, // 下载不存在
		{"Delete download", "DELETE", "/api/downloads/test-id", http.StatusNotFound},      // 下载不存在

		// Process routes
		{"List processes", "GET", "/api/processes", http.StatusOK},
		{"Get process", "GET", "/api/processes/test-id", http.StatusNotFound},                   // 进程不存在
		{"Stop process", "POST", "/api/processes/test-id/stop", http.StatusInternalServerError}, // 进程停止失败（进程不存在是内部错误）

		// Repo routes
		{"Search repo without query", "GET", "/api/repo/search", http.StatusBadRequest},                       // 缺少 q 参数
		{"Search repo with query", "GET", "/api/repo/search?q=qwen&limit=10", http.StatusInternalServerError}, // 网络请求可能失败
		{"Get repo config", "GET", "/api/repo/config", http.StatusOK},
		{"Get repo endpoints", "GET", "/api/repo/endpoints", http.StatusOK},

		// Note: SSE endpoint /api/events is tested separately

		// OpenAI API
		{"OpenAI models", "GET", "/v1/models", http.StatusOK},

		// Ollama API
		{"Ollama tags", "POST", "/api/tags", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body string
			// 为需要请求体的端点提供空 JSON 对象
			if strings.Contains(tt.path, "/api/config") && tt.method == "PUT" {
				body = "{}"
			} else if strings.Contains(tt.path, "/api/scan") && tt.method == "POST" {
				body = "{}"
			} else if strings.Contains(tt.path, "/api/downloads") && tt.method == "POST" {
				// Create download 需要特定的参数，但我们只测试路由存在
				body = `{"url": "http://example.com/model.gguf", "target_path": "/tmp"}`
			}

			var req *http.Request
			if body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestServerSSEEndpoint(t *testing.T) {
	server := createTestServer(t)
	router := server.GetEngine()

	// Create a request with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Run in goroutine since SSE will hang until timeout
	done := make(chan bool)
	go func() {
		router.ServeHTTP(w, req)
		done <- true
	}()

	// Wait for request to complete (or timeout)
	select {
	case <-done:
		// Check that we got the proper SSE headers
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/event-stream")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("SSE request did not complete within timeout")
	}
}

func TestServerStartStop(t *testing.T) {
	cfg := config.DefaultConfig()
	configMgr := config.NewManager()
	_, _ = configMgr.Load()

	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	// 创建存储管理器
	storageCfg := storage.StorageConfig{
		Type: storage.StorageTypeMemory, // 测试使用内存存储
	}
	storageMgr, err := storage.NewManager(&storageCfg)
	require.NoError(t, err, "无法创建存储管理器")

	modelMgr := model.NewManager(cfg, configMgr, procMgr, portAllocator, storageMgr)

	serverConfig := &Config{
		WebPort:   18080, // Use non-standard port for testing
		Host:      "127.0.0.1",
		ServerCfg: cfg,
		ConfigMgr: configMgr,
	}

	server, err := NewServer(serverConfig, modelMgr)
	require.NoError(t, err)

	// Start server
	err = server.Start()
	assert.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Try to connect
	resp, err := http.Get("http://127.0.0.1:18080/api/info")
	if err == nil {
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Stop server
	err = server.Stop()
	assert.NoError(t, err)
}

func TestServerMiddleware(t *testing.T) {
	server := createTestServer(t)
	router := server.GetEngine()

	// Test that recovery middleware works
	req := httptest.NewRequest("GET", "/api/invalid-route", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 404 for invalid routes
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServerConfigDefaults(t *testing.T) {
	server := createTestServer(t)

	assert.NotNil(t, server)
	assert.NotNil(t, server.engine)
}

func BenchmarkServerRequest(b *testing.B) {
	cfg := config.DefaultConfig()
	configMgr := config.NewManager()
	_, _ = configMgr.Load()

	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	// 创建存储管理器
	storageCfg := storage.StorageConfig{
		Type: storage.StorageTypeMemory, // 测试使用内存存储
	}
	storageMgr, err := storage.NewManager(&storageCfg)
	if err != nil {
		b.Fatal(err)
	}

	modelMgr := model.NewManager(cfg, configMgr, procMgr, portAllocator, storageMgr)

	serverConfig := &Config{
		WebPort:   8080,
		Host:      "0.0.0.0",
		ServerCfg: cfg,
		ConfigMgr: configMgr,
	}

	server, err := NewServer(serverConfig, modelMgr)
	if err != nil {
		b.Fatal(err)
	}
	router := server.GetEngine()

	req := httptest.NewRequest("GET", "/api/info", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
