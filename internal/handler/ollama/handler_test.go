package ollama

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

// createTestStorageMgr 创建测试用的存储管理器
func createTestStorageMgr(t *testing.T) *storage.Manager {
	storageCfg := storage.StorageConfig{
		Type: storage.StorageTypeMemory, // 测试使用内存存储
	}
	storageMgr, err := storage.NewManager(&storageCfg)
	require.NoError(t, err, "无法创建存储管理器")
	return storageMgr
}

func TestNewHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load() // 加载默认配置，忽略错误
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)

	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	handler := NewHandler(modelMgr)
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.ModelMgr)
	assert.NotNil(t, handler.Client)
}

func TestChatRequest(t *testing.T) {
	tests := []struct {
		name    string
		reqBody string
		wantErr bool
	}{
		{
			name:    "valid request",
			reqBody: `{"model": "test-model", "messages": [{"role": "user", "content": "Hello"}]}`,
			wantErr: false,
		},
		{
			name:    "empty model",
			reqBody: `{"messages": [{"role": "user", "content": "Hello"}]}`,
			wantErr: false,
		},
		{
			name:    "empty messages",
			reqBody: `{"model": "test-model", "messages": []}`,
			wantErr: false,
		},
		{
			name:    "with options",
			reqBody: `{"model": "test-model", "messages": [{"role": "user", "content": "Hello"}], "options": {"temperature": 0.7, "top_p": 0.9}}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			reqBody: `{"model": "test-model", "messages": [}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req ChatRequest
			err := json.Unmarshal([]byte(tt.reqBody), &req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChatResponse(t *testing.T) {
	resp := ChatResponse{
		Model:     "test-model",
		CreatedAt: "2024-01-01T00:00:00Z",
		Message: ChatMessage{
			Role:    "assistant",
			Content: "Hello! How can I help you?",
		},
		Done: false,
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded ChatResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Model, decoded.Model)
	assert.Equal(t, resp.Message.Content, decoded.Message.Content)
	assert.Equal(t, resp.Done, decoded.Done)
}

func TestGenerationParams(t *testing.T) {
	params := GenerationParams{
		Temperature:   0.7,
		TopP:          0.9,
		TopK:          40,
		NumPredict:    100,
		RepeatPenalty: 1.1,
		Stop:          []string{"stop", "end"},
	}

	data, err := json.Marshal(params)
	require.NoError(t, err)

	var decoded GenerationParams
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, params.Temperature, decoded.Temperature)
	assert.Equal(t, params.TopP, decoded.TopP)
	assert.Equal(t, params.TopK, decoded.TopK)
	assert.Equal(t, params.NumPredict, decoded.NumPredict)
	assert.Equal(t, params.RepeatPenalty, decoded.RepeatPenalty)
	assert.Equal(t, params.Stop, decoded.Stop)
}

func TestHandler_HandleChat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load() // 加载默认配置，忽略错误
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)
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
			reqBody:    `{"model": "nonexistent-model", "messages": [{"role": "user", "content": "Hello"}]}`,
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
			router.POST("/api/chat", handler.HandleChat)

			req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestHandler_HandleTags(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load() // 加载默认配置，忽略错误
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)
	handler := NewHandler(modelMgr)

	router := gin.New()
	router.GET("/api/tags", handler.HandleTags)

	req := httptest.NewRequest("GET", "/api/tags", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	models, ok := resp["models"].([]interface{})
	assert.True(t, ok)
	assert.Empty(t, models)
}

func TestHandler_findModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load() // 加载默认配置，忽略错误
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)
	handler := NewHandler(modelMgr)

	t.Run("no models loaded", func(t *testing.T) {
		_, err := handler.FindModel("test-model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model not found")
	})

	t.Run("empty model name", func(t *testing.T) {
		_, err := handler.FindModel("")
		assert.Error(t, err)
	})
}

func TestHandler_getModelPort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load() // 加载默认配置，忽略错误
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)
	handler := NewHandler(modelMgr)

	t.Run("model not loaded", func(t *testing.T) {
		_, err := handler.GetModelPort("nonexistent-model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model not loaded")
	})

	t.Run("empty model ID", func(t *testing.T) {
		_, err := handler.GetModelPort("")
		assert.Error(t, err)
	})
}

func TestHandler_sendError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load() // 加载默认配置，忽略错误
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)
	handler := NewHandler(modelMgr)

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		handler.sendError(c, http.StatusBadRequest, "test error message")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	errorMsg, ok := resp["error"].(string)
	assert.True(t, ok)
	assert.Equal(t, "test error message", errorMsg)
}
