// Package tests provides integration test infrastructure for the Shepherd project.
// It creates an in-process test server using httptest, MemoryStore, and real route
// registration to exercise all API handlers without starting a real HTTP server.
package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/event"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/infra/taskmanager"
	benchmarkapi "github.com/simonxluo/Shepherd/internal/handler/benchmark"
	chatapi "github.com/simonxluo/Shepherd/internal/handler/chat"
	compatibilityapi "github.com/simonxluo/Shepherd/internal/handler/compatibility"
	filesystemapi "github.com/simonxluo/Shepherd/internal/handler/filesystem"
	"github.com/simonxluo/Shepherd/internal/handler/lmstudio"
	"github.com/simonxluo/Shepherd/internal/handler/ollama"
	"github.com/simonxluo/Shepherd/internal/handler/openai"
	"github.com/simonxluo/Shepherd/internal/handler/paths"
	storageapi "github.com/simonxluo/Shepherd/internal/handler/storage"
	ttsapi "github.com/simonxluo/Shepherd/internal/handler/tts"
	"github.com/simonxluo/Shepherd/internal/handler/anthropic"
	"github.com/simonxluo/Shepherd/internal/infra/port"
	"github.com/simonxluo/Shepherd/internal/infra/process"
	"github.com/simonxluo/Shepherd/internal/infra/storage"
	"github.com/simonxluo/Shepherd/internal/router"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

// TestEnv holds all components needed for integration testing.
type TestEnv struct {
	Engine     *gin.Engine
	Store      storage.Store
	StorageMgr *storage.Manager
	ModelMgr   *model.Manager
	ConfigMgr  *config.Manager
	Config     *config.Config
}

// SetupTestServer creates a fully wired test environment with:
// - MemoryStore for data isolation
// - Real Model Manager (with empty models)
// - All routes registered via router.Setup
// - No real HTTP server (use DoRequest for in-process testing)
func SetupTestServer(t *testing.T) *TestEnv {
	t.Helper()

	gin.SetMode(gin.TestMode)

	// Use default config (test-aware: empty model paths, no auto-scan)
	t.Setenv("SHEPHERD_TEST", "1")
	cfg := config.DefaultConfig()

	// Create a temporary config file for config manager
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "server.config.yaml")
	if err := os.WriteFile(configPath, []byte("server:\n  web_port: 9190\n"), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}
	cfgMgr := config.NewManagerWithPath(configPath)

	// Create storage manager with in-memory store
	storageMgr, err := storage.NewManager(&cfg.Storage)
	if err != nil {
		t.Fatalf("failed to create storage manager: %v", err)
	}
	t.Cleanup(func() { storageMgr.Close() })

	store := storageMgr.GetStore()

	// Create process manager and port allocator
	procMgr := process.NewManager()
	portAlloc := port.NewPortAllocator(18000, 19000)

	// Create model manager
	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAlloc, storageMgr)
	t.Cleanup(func() { modelMgr.Close() })

	// Create event manager (needed by some handlers indirectly)
	_ = event.NewManager(modelMgr.GetLoadedModelCount)

	// Suppress logger output during tests
	logger.GetLogger()

	// Create compatibility server manager
	compatServerManager := compatibilityapi.NewServerManager(modelMgr)

	// Create all handlers
	handlers := &router.Handlers{
		OpenAI:        openai.NewHandler(modelMgr),
		Ollama:        ollama.NewHandler(modelMgr),
		Anthropic:     anthropic.NewHandler(modelMgr),
		LMStudio:      lmstudio.NewHandler(modelMgr),
		Audio:         openai.NewAudioHandler(modelMgr),
		Image:         openai.NewImageHandler(modelMgr),
		Music:         openai.NewMusicHandler(modelMgr),
		Paths:         paths.NewHandler(cfgMgr),
		Storage:       storageapi.NewHandler(cfgMgr, storageMgr),
		Compatibility: compatibilityapi.NewHandler(cfgMgr, compatServerManager),
		Filesystem:    filesystemapi.NewHandler(),
		Benchmark:     benchmarkapi.NewHandler(logger.GetLogger(), store, taskmanager.NewManager(nil), event.NewManager(nil)),
		Chat:          chatapi.NewHandler(modelMgr),
		TTS:           ttsapi.NewHandler(storageMgr, filepath.Join(tmpDir, "tts")),
	}

	// Create a mock server handler that implements router.ServerHandlers
	serverHandlers := newTestServerHandlers(cfg, cfgMgr, modelMgr, storageMgr, handlers)

	// Create Gin engine and register all routes
	engine := gin.New()
	router.Setup(engine, handlers, serverHandlers, router.Config{}, nil)

	return &TestEnv{
		Engine:     engine,
		Store:      store,
		StorageMgr: storageMgr,
		ModelMgr:   modelMgr,
		ConfigMgr:  cfgMgr,
		Config:     cfg,
	}
}

// DoRequest performs an HTTP request against the test engine and returns the recorder.
func DoRequest(engine *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			panic("failed to marshal request body: " + err.Error())
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// DoRequestRaw performs a request with a raw []byte body.
func DoRequestRaw(engine *gin.Engine, method, path string, body []byte) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// APIResponse represents the standard API response structure.
type APIResponse struct {
	Success  bool                   `json:"success"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Error    *APIError              `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// APIError represents the error portion of an API response.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ParseResponse parses a recorder's body into an APIResponse.
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder) *APIResponse {
	t.Helper()
	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v\nbody: %s", err, w.Body.String())
	}
	return &resp
}

// ParseRawResponse parses a recorder's body into a generic map (for non-standard responses).
func ParseRawResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v\nbody: %s", err, w.Body.String())
	}
	return resp
}

// AssertStatus checks that the response has the expected HTTP status code.
func AssertStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("expected status %d, got %d\nbody: %s", expected, w.Code, w.Body.String())
	}
}

// AssertSuccess checks that the response indicates success.
func AssertSuccess(t *testing.T, w *httptest.ResponseRecorder) *APIResponse {
	t.Helper()
	AssertStatus(t, w, http.StatusOK)
	resp := ParseResponse(t, w)
	if !resp.Success {
		t.Errorf("expected success=true, got false. Error: %+v", resp.Error)
	}
	return resp
}
