package langchain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// TestLlamaCPPNew 测试 LlamaCPP 实例创建
func TestLlamaCPPNew(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		modelID   string
		opts      []Option
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid creation",
			serverURL: "http://localhost:8080",
			modelID:   "test-model",
			opts:      []Option{WithTemperature(0.8), WithMaxTokens(100)},
			wantErr:   false,
		},
		{
			name:      "empty server URL",
			serverURL: "",
			modelID:   "test-model",
			wantErr:   true,
			errMsg:    "server URL cannot be empty",
		},
		{
			name:      "default values",
			serverURL: "http://localhost:8080",
			modelID:   "test-model",
			opts:      nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm, err := NewLlamaCPP(tt.serverURL, tt.modelID, tt.opts...)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, llm)
			} else {
				require.NoError(t, err)
				require.NotNil(t, llm)
				assert.Equal(t, tt.serverURL, llm.serverURL)
				assert.Equal(t, tt.modelID, llm.modelID)

				// 检查默认值
				if tt.opts == nil {
					assert.Equal(t, float32(0.7), llm.temperature)
					assert.Equal(t, -1, llm.maxTokens)
					assert.Equal(t, float32(0.9), llm.topP)
					assert.Equal(t, 40, llm.topK)
				}
			}
		})
	}
}

// TestLlamaCPPOptions 测试选项功能
func TestLlamaCPPOptions(t *testing.T) {
	serverURL := "http://localhost:8080"
	modelID := "test-model"

	tests := []struct {
		name       string
		opts       []Option
		wantTemp   float32
		wantMaxTok int
		wantTopP   float32
		wantTopK   int
	}{
		{
			name: "all options",
			opts: []Option{
				WithTemperature(0.5),
				WithMaxTokens(200),
				WithTopP(0.8),
				WithTopK(50),
			},
			wantTemp:   0.5,
			wantMaxTok: 200,
			wantTopP:   0.8,
			wantTopK:   50,
		},
		{
			name: "partial options",
			opts: []Option{
				WithTemperature(1.0),
				WithMaxTokens(500),
			},
			wantTemp:   1.0,
			wantMaxTok: 500,
			wantTopP:   0.9, // 默认值
			wantTopK:   40,  // 默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm, err := NewLlamaCPP(serverURL, modelID, tt.opts...)
			require.NoError(t, err)

			assert.Equal(t, tt.wantTemp, llm.temperature)
			assert.Equal(t, tt.wantMaxTok, llm.maxTokens)
			assert.Equal(t, tt.wantTopP, llm.topP)
			assert.Equal(t, tt.wantTopK, llm.topK)
		})
	}
}

// TestConvertMessages 测试消息转换
func TestConvertMessages(t *testing.T) {
	llm, err := NewLlamaCPP("http://localhost:8080", "test-model")
	require.NoError(t, err)

	tests := []struct {
		name     string
		messages []llms.MessageContent
		wantLen  int
	}{
		{
			name: "single message",
			messages: []llms.MessageContent{
				{
					Role: llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{
						llms.TextPart("Hello"),
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "multiple messages",
			messages: []llms.MessageContent{
				{
					Role: llms.ChatMessageTypeSystem,
					Parts: []llms.ContentPart{
						llms.TextPart("You are a helpful assistant"),
					},
				},
				{
					Role: llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{
						llms.TextPart("Hello"),
					},
				},
				{
					Role: llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{
						llms.TextPart("Hi there!"),
					},
				},
			},
			wantLen: 3,
		},
		{
			name: "tool message",
			messages: []llms.MessageContent{
				{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.TextPart("Tool result"),
					},
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatMessages := llm.convertMessages(tt.messages)
			assert.Equal(t, tt.wantLen, len(chatMessages))

			// 验证角色映射
			if len(tt.messages) > 0 {
				switch tt.messages[0].Role {
				case llms.ChatMessageTypeSystem:
					assert.Equal(t, "system", chatMessages[0].Role)
				case llms.ChatMessageTypeHuman:
					assert.Equal(t, "user", chatMessages[0].Role)
				case llms.ChatMessageTypeAI:
					assert.Equal(t, "assistant", chatMessages[0].Role)
				case llms.ChatMessageTypeTool:
					assert.Equal(t, "user", chatMessages[0].Role)
				}
			}
		})
	}
}

// TestExtractTextContent 测试文本内容提取
func TestExtractTextContent(t *testing.T) {
	llm, err := NewLlamaCPP("http://localhost:8080", "test-model")
	require.NoError(t, err)

	tests := []struct {
		name     string
		msg      llms.MessageContent
		wantText string
	}{
		{
			name: "single text part",
			msg: llms.MessageContent{
				Role: llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{
					llms.TextPart("Hello"),
				},
			},
			wantText: "Hello",
		},
		{
			name: "multiple text parts",
			msg: llms.MessageContent{
				Role: llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{
					llms.TextPart("Line 1"),
					llms.TextPart("Line 2"),
					llms.TextPart("Line 3"),
				},
			},
			wantText: "Line 1\nLine 2\nLine 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := llm.extractTextContent(tt.msg)
			assert.Equal(t, tt.wantText, text)
		})
	}
}

// TestLlamaCPPGetModel 测试从选项获取模型
func TestLlamaCPPGetModel(t *testing.T) {
	llm, err := NewLlamaCPP("http://localhost:8080", "default-model")
	require.NoError(t, err)

	tests := []struct {
		name     string
		llmModel string
		opts     *llms.CallOptions
		want     string
	}{
		{
			name:     "no option",
			llmModel: "default-model",
			opts:     &llms.CallOptions{},
			want:     "default-model",
		},
		{
			name:     "with option",
			llmModel: "default-model",
			opts:     &llms.CallOptions{Model: "custom-model"},
			want:     "custom-model",
		},
		{
			name:     "nil option",
			llmModel: "default-model",
			opts:     nil,
			want:     "default-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := llm.getModel(tt.opts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestLlamaCPPGetTemperature 测试从选项获取温度
func TestLlamaCPPGetTemperature(t *testing.T) {
	llm, err := NewLlamaCPP("http://localhost:8080", "test-model", WithTemperature(0.7))
	require.NoError(t, err)

	tests := []struct {
		name    string
		llmTemp float32
		opts    *llms.CallOptions
		want    float32
	}{
		{
			name:    "no option",
			llmTemp: 0.7,
			opts:    &llms.CallOptions{},
			want:    0.7,
		},
		{
			name:    "with option",
			llmTemp: 0.7,
			opts:    &llms.CallOptions{Temperature: 0.9},
			want:    0.9,
		},
		{
			name:    "nil option",
			llmTemp: 0.7,
			opts:    nil,
			want:    0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm.temperature = tt.llmTemp
			got := llm.getTemperature(tt.opts)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestChatCompletionRequestSerialization 测试请求序列化
func TestChatCompletionRequestSerialization(t *testing.T) {
	req := ChatCompletionRequest{
		Model:       "test-model",
		Temperature: 0.7,
		MaxTokens:   100,
		TopP:        0.9,
		TopK:        40,
		Stream:      false,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	}

	// 测试 JSON 序列化不会出错
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-model")
	assert.Contains(t, string(data), "Hello")

	// 测试反序列化
	var unmarshaled ChatCompletionRequest
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, req.Model, unmarshaled.Model)
	assert.Equal(t, req.Temperature, unmarshaled.Temperature)
	assert.Equal(t, len(req.Messages), len(unmarshaled.Messages))
}

// TestNewManager 测试管理器创建
func TestNewManager(t *testing.T) {
	// 注意：这个测试需要 mock model.Manager
	// 这里我们只测试 nil 检查
	manager := NewManager(nil, nil)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.llmInstances)
}

// TestManagerGetStats 测试统计信息
func TestManagerGetStats(t *testing.T) {
	manager := NewManager(nil, nil)

	// 由于 modelMgr 为 nil，直接测试 manager 结构
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.llmInstances)
}

// BenchmarkConvertMessages 性能测试
func BenchmarkConvertMessages(b *testing.B) {
	llm, _ := NewLlamaCPP("http://localhost:8080", "test-model")

	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart("You are a helpful assistant"),
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart("Explain quantum computing"),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		llm.convertMessages(messages)
	}
}
