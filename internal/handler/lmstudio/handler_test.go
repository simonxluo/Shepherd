package lmstudio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestStorageMgr(t *testing.T) *storage.Manager {
	storageCfg := storage.StorageConfig{
		Type: storage.StorageTypeMemory,
	}
	storageMgr, err := storage.NewManager(&storageCfg)
	require.NoError(t, err, "无法创建存储管理器")
	return storageMgr
}

func createTestModelMgr(t *testing.T) *model.Manager {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	return model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)
}

func TestNewHandler(t *testing.T) {
	modelMgr := createTestModelMgr(t)
	handler := NewHandler(modelMgr)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.ModelMgr)
	assert.NotNil(t, handler.Client)
}

func TestChatCompletionRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		reqBody := `{
			"model": "test-model",
			"messages": [{"role": "user", "content": "Hello"}],
			"stream": false,
			"temperature": 0.7
		}`
		var req ChatCompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)
		assert.Equal(t, "test-model", req.Model)
		assert.Len(t, req.Messages, 1)
		assert.False(t, req.Stream)
		assert.Equal(t, 0.7, req.Temperature)
	})

	t.Run("missing model", func(t *testing.T) {
		reqBody := `{"messages": [{"role": "user", "content": "Hello"}]}`
		var req ChatCompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)
		assert.Equal(t, "", req.Model)
	})

	t.Run("with tools", func(t *testing.T) {
		reqBody := `{
			"model": "test-model",
			"messages": [{"role": "user", "content": "Hello"}],
			"tools": [{"type": "function", "function": {"name": "get_weather", "description": "Get weather"}}]
		}`
		var req ChatCompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)
		assert.Len(t, req.Tools, 1)
		assert.Equal(t, "get_weather", req.Tools[0].Function.Name)
	})
}

func TestCompletionRequest(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		reqBody := `{
			"model": "test-model",
			"prompt": "Once upon a time",
			"max_tokens": 100,
			"temperature": 0.7
		}`
		var req CompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)
		assert.Equal(t, "test-model", req.Model)
		assert.Equal(t, 100, req.MaxTokens)
	})

	t.Run("prompt as array", func(t *testing.T) {
		reqBody := `{"model": "test-model", "prompt": ["Hello", "World"]}`
		var req CompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)
		assert.NotNil(t, req.Prompt)
	})
}

func TestEmbeddingRequest(t *testing.T) {
	t.Run("valid request with string input", func(t *testing.T) {
		reqBody := `{"model": "test-model", "input": "Hello world"}`
		var req EmbeddingRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)
		assert.Equal(t, "test-model", req.Model)
		assert.NotNil(t, req.Input)
	})

	t.Run("valid request with array input", func(t *testing.T) {
		reqBody := `{"model": "test-model", "input": ["Hello", "World"]}`
		var req EmbeddingRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)
		assert.NotNil(t, req.Input)
	})
}

func TestChatCompletionResponse(t *testing.T) {
	choices := []ChatCompletionChoice{
		{
			Index: 0,
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Hello!",
			},
			FinishReason: "stop",
		},
	}
	usage := &Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
	resp := NewChatCompletionResponse("chatcmpl-123", "test-model", choices, usage)
	assert.Equal(t, "chat.completion", resp.Object)
	assert.Equal(t, "test-model", resp.Model)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestEmbeddingResponse(t *testing.T) {
	data := []EmbeddingData{
		{Object: "embedding", Embedding: []float64{0.1, 0.2, 0.3}, Index: 0},
	}
	usage := &Usage{PromptTokens: 5, TotalTokens: 5}
	resp := NewEmbeddingResponse("test-model", data, usage)
	assert.Equal(t, "list", resp.Object)
	assert.Equal(t, "test-model", resp.Model)
	assert.Len(t, resp.Data, 1)
}

func TestModelsResponse(t *testing.T) {
	models := []Model{
		{ID: "model-1", Object: "model", Created: 123456, OwnedBy: "shepherd"},
	}
	resp := NewModelsResponse(models)
	assert.Equal(t, "list", resp.Object)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "model-1", resp.Data[0].ID)
}

func TestErrorResponse(t *testing.T) {
	resp := NewErrorResponse("Model not found", "model_not_found", "model", 404)
	assert.Equal(t, "Model not found", resp.Error.Message)
	assert.Equal(t, "model_not_found", resp.Error.Type)
	assert.Equal(t, "model", resp.Error.Param)
}

func TestHandler_HandleChatCompletions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	modelMgr := createTestModelMgr(t)
	handler := NewHandler(modelMgr)

	tests := []struct {
		name       string
		reqBody    string
		wantStatus int
	}{
		{
			name:       "missing model",
			reqBody:    `{"messages": [{"role": "user", "content": "Hello"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty messages",
			reqBody:    `{"model": "test", "messages": []}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "model not found",
			reqBody:    `{"model": "nonexistent", "messages": [{"role": "user", "content": "Hello"}]}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid json",
			reqBody:    `{"model": "test", "messages": [}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/v1/chat/completions", handler.HandleChatCompletions)
			req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestHandler_HandleCompletions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	modelMgr := createTestModelMgr(t)
	handler := NewHandler(modelMgr)

	t.Run("model not found", func(t *testing.T) {
		router := gin.New()
		router.POST("/v1/completions", handler.HandleCompletions)
		reqBody := `{"model": "nonexistent", "prompt": "Hello"}`
		req := httptest.NewRequest("POST", "/v1/completions", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("no models loaded fallback", func(t *testing.T) {
		router := gin.New()
		router.POST("/v1/completions", handler.HandleCompletions)
		reqBody := `{"prompt": "Hello"}`
		req := httptest.NewRequest("POST", "/v1/completions", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandler_HandleEmbeddings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	modelMgr := createTestModelMgr(t)
	handler := NewHandler(modelMgr)

	tests := []struct {
		name       string
		reqBody    string
		wantStatus int
	}{
		{
			name:       "missing model",
			reqBody:    `{"input": "Hello"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing input",
			reqBody:    `{"model": "test-model"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "model not found",
			reqBody:    `{"model": "nonexistent", "input": "Hello"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid json",
			reqBody:    `{"model": "test"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/v1/embeddings", handler.HandleEmbeddings)
			req := httptest.NewRequest("POST", "/v1/embeddings", strings.NewReader(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestHandler_HandleModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	modelMgr := createTestModelMgr(t)
	handler := NewHandler(modelMgr)

	router := gin.New()
	router.GET("/v1/models", handler.HandleModels)
	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response ModelsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "list", response.Object)
	assert.Empty(t, response.Data)
}

func TestHandler_sendError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	modelMgr := createTestModelMgr(t)
	handler := NewHandler(modelMgr)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		handler.sendError(c, http.StatusBadRequest, "invalid_request", "test error", "param")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "test error", resp.Error.Message)
	assert.Equal(t, "invalid_request", resp.Error.Type)
	assert.Equal(t, "param", resp.Error.Param)
}

func TestHandler_FindModel(t *testing.T) {
	modelMgr := createTestModelMgr(t)
	handler := NewHandler(modelMgr)

	_, err := handler.FindModel("nonexistent-model")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model not found")
}
