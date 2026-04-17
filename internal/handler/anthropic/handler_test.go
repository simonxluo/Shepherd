package anthropic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
	"github.com/shepherd-project/shepherd/Shepherd/internal/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/storage"
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
	assert.NotNil(t, handler.modelMgr)
	assert.NotNil(t, handler.client)
}

func TestMessageRequest(t *testing.T) {
	tests := []struct {
		name    string
		reqBody string
		wantErr bool
	}{
		{
			name:    "valid request",
			reqBody: `{"model": "test-model", "max_tokens": 100, "messages": [{"role": "user", "content": "Hello"}]}`,
			wantErr: false,
		},
		{
			name:    "with system message",
			reqBody: `{"model": "test-model", "max_tokens": 100, "messages": [{"role": "user", "content": "Hello"}], "system": "You are a helpful assistant"}`,
			wantErr: false,
		},
		{
			name:    "with generation params",
			reqBody: `{"model": "test-model", "max_tokens": 100, "messages": [{"role": "user", "content": "Hello"}], "temperature": 0.7, "top_p": 0.9, "top_k": 40}`,
			wantErr: false,
		},
		{
			name:    "with stop sequences",
			reqBody: `{"model": "test-model", "max_tokens": 100, "messages": [{"role": "user", "content": "Hello"}], "stop_sequences": ["stop", "end"]}`,
			wantErr: false,
		},
		{
			name:    "with stream",
			reqBody: `{"model": "test-model", "max_tokens": 100, "messages": [{"role": "user", "content": "Hello"}], "stream": true}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			reqBody: `{"model": "test-model", "max_tokens": 100, "messages": [}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req MessageRequest
			err := json.Unmarshal([]byte(tt.reqBody), &req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello, how are you?",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.Role, decoded.Role)
	assert.Equal(t, msg.Content, decoded.Content)
}

func TestMessageResponse(t *testing.T) {
	resp := MessageResponse{
		ID:    "msg_123",
		Type:  "message",
		Role:  "assistant",
		Model: "test-model",
		Content: []ContentBlock{
			{Type: "text", Text: "Hello! How can I help you?"},
		},
		StopReason: "end_turn",
		Usage: &Usage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded MessageResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.ID, decoded.ID)
	assert.Equal(t, resp.Type, decoded.Type)
	assert.Equal(t, resp.Role, decoded.Role)
	assert.Equal(t, resp.Model, decoded.Model)
	assert.Len(t, decoded.Content, 1)
	assert.Equal(t, resp.Content[0].Text, decoded.Content[0].Text)
	assert.Equal(t, resp.StopReason, decoded.StopReason)
	assert.Equal(t, resp.Usage.InputTokens, decoded.Usage.InputTokens)
	assert.Equal(t, resp.Usage.OutputTokens, decoded.Usage.OutputTokens)
}

func TestContentBlock(t *testing.T) {
	tests := []struct {
		name    string
		block   ContentBlock
		wantErr bool
	}{
		{
			name:    "text block",
			block:   ContentBlock{Type: "text", Text: "Hello"},
			wantErr: false,
		},
		{
			name:    "thinking block",
			block:   ContentBlock{Type: "thinking"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.block)
			require.NoError(t, err)

			var decoded ContentBlock
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.block.Type, decoded.Type)
			assert.Equal(t, tt.block.Text, decoded.Text)
		})
	}
}

func TestUsage(t *testing.T) {
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 50,
	}

	data, err := json.Marshal(usage)
	require.NoError(t, err)

	var decoded Usage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, usage.InputTokens, decoded.InputTokens)
	assert.Equal(t, usage.OutputTokens, decoded.OutputTokens)
}

func TestErrorDetail(t *testing.T) {
	errDetail := ErrorDetail{
		Type:    "invalid_request",
		Message: "Invalid request body",
	}

	data, err := json.Marshal(errDetail)
	require.NoError(t, err)

	var decoded ErrorDetail
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, errDetail.Type, decoded.Type)
	assert.Equal(t, errDetail.Message, decoded.Message)
}

func TestHandler_HandleMessages(t *testing.T) {
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
			reqBody:    `{"max_tokens": 100, "messages": [{"role": "user", "content": "Hello"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing max_tokens",
			reqBody:    `{"model": "test", "messages": [{"role": "user", "content": "Hello"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty messages",
			reqBody:    `{"model": "test", "max_tokens": 100, "messages": []}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "model not found",
			reqBody:    `{"model": "nonexistent-model", "max_tokens": 100, "messages": [{"role": "user", "content": "Hello"}]}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid json",
			reqBody:    `{"model": "test", "max_tokens": 100, "messages": [}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/v1/messages", handler.HandleMessages)

			req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
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
		_, err := handler.findModel("test-model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model not found")
	})

	t.Run("empty model name", func(t *testing.T) {
		_, err := handler.findModel("")
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
		_, err := handler.getModelPort("nonexistent-model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model not loaded")
	})

	t.Run("empty model ID", func(t *testing.T) {
		_, err := handler.getModelPort("")
		assert.Error(t, err)
	})
}

func TestHandler_convertResponse(t *testing.T) {
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
		openaiResp map[string]interface{}
		model      string
		validate   func(t *testing.T, resp *MessageResponse)
	}{
		{
			name: "valid response with content",
			openaiResp: map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"content": "Hello! How can I help?",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10.0,
					"completion_tokens": 20.0,
				},
			},
			model: "test-model",
			validate: func(t *testing.T, resp *MessageResponse) {
				assert.Equal(t, "message", resp.Type)
				assert.Equal(t, "assistant", resp.Role)
				assert.Equal(t, "test-model", resp.Model)
				assert.Len(t, resp.Content, 1)
				assert.Equal(t, "Hello! How can I help?", resp.Content[0].Text)
				assert.Equal(t, "stop", resp.StopReason)
				assert.Equal(t, 10, resp.Usage.InputTokens)
				assert.Equal(t, 20, resp.Usage.OutputTokens)
			},
		},
		{
			name: "response without choices",
			openaiResp: map[string]interface{}{
				"usage": map[string]interface{}{},
			},
			model: "test-model",
			validate: func(t *testing.T, resp *MessageResponse) {
				assert.Equal(t, "message", resp.Type)
				assert.Empty(t, resp.Content)
			},
		},
		{
			name:       "empty response",
			openaiResp: map[string]interface{}{},
			model:      "test-model",
			validate: func(t *testing.T, resp *MessageResponse) {
				assert.Equal(t, "message", resp.Type)
				assert.Equal(t, "test-model", resp.Model)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := handler.convertResponse(tt.openaiResp, tt.model)
			tt.validate(t, resp)
		})
	}
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

	tests := []struct {
		name     string
		errType  string
		wantCode int
	}{
		{
			name:     "invalid_request error",
			errType:  "invalid_request",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid_request_error error",
			errType:  "invalid_request_error",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "other error type",
			errType:  "internal_error",
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				handler.sendError(c, tt.wantCode, tt.errType, "test error message")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)

			var resp MessageResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.NotNil(t, resp.Error)
			assert.Equal(t, tt.errType, resp.Error.Type)
			assert.Equal(t, "test error message", resp.Error.Message)
		})
	}
}

func TestGenerateID(t *testing.T) {
	id := generateID("msg")
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "msg")
	assert.Contains(t, id, "_")
}
