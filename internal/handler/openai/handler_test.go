package openai

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

func TestChatCompletionRequest(t *testing.T) {
	t.Run("Valid request", func(t *testing.T) {
		reqBody := `{
			"model": "test-model",
			"messages": [
				{"role": "user", "content": "Hello"}
			],
			"stream": false
		}`

		var req ChatCompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)

		assert.Equal(t, "test-model", req.Model)
		assert.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)
		assert.Equal(t, "Hello", req.Messages[0].Content)
		assert.False(t, req.Stream)
	})

	t.Run("Missing model", func(t *testing.T) {
		reqBody := `{
			"messages": [
				{"role": "user", "content": "Hello"}
			]
		}`

		var req ChatCompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)

		assert.Equal(t, "", req.Model)
	})

	t.Run("With extra fields", func(t *testing.T) {
		reqBody := `{
			"model": "test-model",
			"messages": [
				{"role": "user", "content": "Hello"}
			],
			"custom_field": "custom_value"
		}`

		var req ChatCompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)

		assert.Equal(t, "test-model", req.Model)
		assert.Contains(t, req.Extra, "custom_field")
	})
}

func TestCompletionRequest(t *testing.T) {
	t.Run("Valid request", func(t *testing.T) {
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
		assert.Equal(t, "Once upon a time", req.Prompt)
		assert.Equal(t, 100, req.MaxTokens)
		assert.Equal(t, 0.7, req.Temperature)
	})

	t.Run("Prompt as array", func(t *testing.T) {
		reqBody := `{
			"model": "test-model",
			"prompt": ["Hello", "World"]
		}`

		var req CompletionRequest
		err := json.Unmarshal([]byte(reqBody), &req)
		require.NoError(t, err)

		assert.NotNil(t, req.Prompt)
	})
}

func TestChatCompletionResponse(t *testing.T) {
	t.Run("Create response", func(t *testing.T) {
		choices := []ChatCompletionChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "Hello! How can I help you?",
				},
				FinishReason: "stop",
			},
		}

		usage := &Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		}

		response := NewChatCompletionResponse("chatcmpl-123", "test-model", choices, usage)

		assert.Equal(t, "chat.completion", response.Object)
		assert.Equal(t, "test-model", response.Model)
		assert.Len(t, response.Choices, 1)
		assert.Equal(t, "assistant", response.Choices[0].Message.Role)
		assert.Equal(t, 10, response.Usage.PromptTokens)
	})
}

func TestModelsResponse(t *testing.T) {
	t.Run("Create models response", func(t *testing.T) {
		models := []Model{
			{ID: "model-1", Object: "model", Created: 123456, OwnedBy: "shepherd"},
			{ID: "model-2", Object: "model", Created: 123457, OwnedBy: "shepherd"},
		}

		response := NewModelsResponse(models)

		assert.Equal(t, "list", response.Object)
		assert.Len(t, response.Data, 2)
		assert.Equal(t, "model-1", response.Data[0].ID)
	})
}

func TestErrorResponse(t *testing.T) {
	t.Run("Create error response", func(t *testing.T) {
		response := NewErrorResponse("Model not found", "model_not_found", "model", 404)

		assert.Equal(t, "Model not found", response.Error.Message)
		assert.Equal(t, "model_not_found", response.Error.Type)
		assert.Equal(t, "model", response.Error.Param)
	})
}

func TestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load() // 加载默认配置，忽略错误
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	handler := NewHandler(modelMgr)

	t.Run("HandleModels - no models loaded", func(t *testing.T) {
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
	})

	t.Run("HandleChatCompletions - missing model", func(t *testing.T) {
		router := gin.New()
		router.POST("/v1/chat/completions", handler.HandleChatCompletions)

		reqBody := `{
			"messages": [{"role": "user", "content": "Hello"}]
		}`
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("HandleChatCompletions - empty messages", func(t *testing.T) {
		router := gin.New()
		router.POST("/v1/chat/completions", handler.HandleChatCompletions)

		reqBody := `{"model": "test", "messages": []}`
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestFindModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	_, _ = cfgMgr.Load() // 加载默认配置，忽略错误
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)
	storageMgr := createTestStorageMgr(t)
	modelMgr := model.NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	handler := NewHandler(modelMgr)

	t.Run("No models loaded", func(t *testing.T) {
		_, err := handler.FindModel("test-model")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model not found")
	})
}
