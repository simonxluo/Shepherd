// Package model provides tests for model capability detection
package model

import (
	"testing"

	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/gguf"
	"github.com/stretchr/testify/assert"
)

// TestDetectCapabilities_DeepSeekR1 测试 DeepSeek-R1 思考模型检测
func TestDetectCapabilities_DeepSeekR1(t *testing.T) {
	meta := &gguf.Metadata{
		Name:         "deepseek-r1-qwen-7b",
		Architecture: "deepseek2",
		ChatTemplate: "...enable_thinking...",
	}
	caps := DetectCapabilities(meta)

	assert.True(t, caps.Thinking, "DeepSeek-R1 should have thinking capability")
	assert.False(t, caps.Rerank, "DeepSeek-R1 should not have rerank capability")
	assert.False(t, caps.Embedding, "DeepSeek-R1 should not have embedding capability")
}

// TestDetectCapabilities_BGE_M3 测试 BGE-M3 嵌入模型检测
func TestDetectCapabilities_BGE_M3(t *testing.T) {
	meta := &gguf.Metadata{
		Name:         "bge-m3",
		Architecture: "bge",
		ChatTemplate: "",
	}
	caps := DetectCapabilities(meta)

	assert.True(t, caps.Embedding, "BGE-M3 should have embedding capability")
	assert.False(t, caps.Thinking, "BGE-M3 should not have thinking capability")
	assert.False(t, caps.Tools, "BGE-M3 should not have tools capability")
}

// TestDetectCapabilities_BGEReranker 测试 BGE Reranker 检测
func TestDetectCapabilities_BGEReranker(t *testing.T) {
	meta := &gguf.Metadata{
		Name:         "bge-reranker-v2-m3",
		Architecture: "bge",
		ChatTemplate: "",
	}
	caps := DetectCapabilities(meta)

	assert.True(t, caps.Rerank, "BGE Reranker should have rerank capability")
	assert.False(t, caps.Thinking, "BGE Reranker should not have thinking capability")
	assert.False(t, caps.Tools, "BGE Reranker should not have tools capability")
}

// TestDetectCapabilities_GPT4o 测试 GPT-4o 工具调用检测
func TestDetectCapabilities_GPT4o(t *testing.T) {
	meta := &gguf.Metadata{
		Name:         "gpt-4o",
		Architecture: "gpt2",
		ChatTemplate: "...tool_calls...function...",
	}
	caps := DetectCapabilities(meta)

	assert.True(t, caps.Tools, "GPT-4o should have tools capability")
	assert.False(t, caps.Rerank, "GPT-4o should not have rerank capability")
	assert.False(t, caps.Embedding, "GPT-4o should not have embedding capability")
}

// TestDetectCapabilities_QWQ 测试 QWQ 思考模型检测
func TestDetectCapabilities_QWQ(t *testing.T) {
	meta := &gguf.Metadata{
		Name:         "qwq-32b-preview",
		Architecture: "qwen2",
		ChatTemplate: "...reasoning...",
	}
	caps := DetectCapabilities(meta)

	assert.True(t, caps.Thinking, "QWQ should have thinking capability")
	assert.False(t, caps.Rerank, "QWQ should not have rerank capability")
	assert.False(t, caps.Embedding, "QWQ should not have embedding capability")
}

// TestDetectCapabilities_NilMetadata 测试空元数据处理
func TestDetectCapabilities_NilMetadata(t *testing.T) {
	caps := DetectCapabilities(nil)

	assert.False(t, caps.Thinking, "Nil metadata should not have thinking capability")
	assert.False(t, caps.Tools, "Nil metadata should not have tools capability")
	assert.False(t, caps.Rerank, "Nil metadata should not have rerank capability")
	assert.False(t, caps.Embedding, "Nil metadata should not have embedding capability")
}

// TestDetectCapabilities_EmptyMetadata 测试空元数据处理
func TestDetectCapabilities_EmptyMetadata(t *testing.T) {
	meta := &gguf.Metadata{
		Name:         "",
		Architecture: "",
		ChatTemplate: "",
	}
	caps := DetectCapabilities(meta)

	assert.False(t, caps.Thinking, "Empty metadata should not have thinking capability")
	assert.False(t, caps.Tools, "Empty metadata should not have tools capability")
	assert.False(t, caps.Rerank, "Empty metadata should not have rerank capability")
	assert.False(t, caps.Embedding, "Empty metadata should not have embedding capability")
}

// TestDetectCapabilities_E5Embedding 测试 E5 嵌入模型检测
func TestDetectCapabilities_E5Embedding(t *testing.T) {
	meta := &gguf.Metadata{
		Name:         "e5-base-v2",
		Architecture: "bert",
		ChatTemplate: "",
	}
	caps := DetectCapabilities(meta)

	assert.True(t, caps.Embedding, "E5 should have embedding capability")
	assert.False(t, caps.Tools, "E5 should not have tools capability")
}

// TestDetectCapabilities_ContainsAny 测试关键词匹配辅助函数
func TestDetectCapabilities_ContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		keywords []string
		expected bool
	}{
		{
			name:     "Match found",
			s:        "deepseek-r1-qwen-7b",
			keywords: []string{"deepseek-r1", "qwq"},
			expected: true,
		},
		{
			name:     "No match",
			s:        "gpt-4o",
			keywords: []string{"deepseek-r1", "qwq"},
			expected: false,
		},
		{
			name:     "Empty string",
			s:        "",
			keywords: []string{"test"},
			expected: false,
		},
		{
			name:     "Empty keywords",
			s:        "test",
			keywords: []string{},
			expected: false,
		},
		{
			name:     "Case sensitive match",
			s:        "deepseek-r1",
			keywords: []string{"deepseek-r1"},
			expected: true,
		},
		{
			name:     "Case insensitive (lowercase input)",
			s:        "deepseek-r1",
			keywords: []string{"DEEPSEEK-R1"},
			expected: false, // containsAny is case-sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.keywords)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestThinkingDetectionFromChatTemplate 测试从 chat_template 检测 thinking 能力
func TestThinkingDetectionFromChatTemplate(t *testing.T) {
	tests := []struct {
		name         string
		chatTemplate string
		expected     bool
	}{
		{
			name:         "enable_thinking keyword",
			chatTemplate: "{% if enable_thinking %}...",
			expected:     true,
		},
		{
			name:         "thinking keyword",
			chatTemplate: "...{{ thinking }}...",
			expected:     true,
		},
		{
			name:         "reasoning keyword",
			chatTemplate: "...{% set reasoning = true %}...",
			expected:     true,
		},
		{
			name:         "no thinking keywords",
			chatTemplate: "...standard chat template...",
			expected:     false,
		},
		{
			name:         "empty chat template",
			chatTemplate: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gguf.Metadata{
				Name:         "test-model",
				Architecture: "llama",
				ChatTemplate: tt.chatTemplate,
			}
			caps := DetectCapabilities(meta)
			assert.Equal(t, tt.expected, caps.Thinking, "thinking capability mismatch")
		})
	}
}

// TestThinkingDetectionFromModelName 测试从模型名称检测 thinking 能力
func TestThinkingDetectionFromModelName(t *testing.T) {
	tests := []struct {
		name          string
		modelName     string
		architecture  string
		expectedThink bool
	}{
		{
			name:          "deepseek-r1 in name",
			modelName:     "deepseek-r1-qwen-7b",
			architecture:  "llama",
			expectedThink: true,
		},
		{
			name:          "deepseek r1 with space",
			modelName:     "deepseek r1 671b",
			architecture:  "llama",
			expectedThink: true,
		},
		{
			name:          "qwq model",
			modelName:     "qwq-32b-preview",
			architecture:  "qwen2",
			expectedThink: true,
		},
		{
			name:          "qwq-32b variant",
			modelName:     "qwq-32b",
			architecture:  "qwen2",
			expectedThink: true,
		},
		{
			name:          "case insensitive deepseek-r1",
			modelName:     "DeepSeek-R1-Distill-Qwen-7B",
			architecture:  "llama",
			expectedThink: true,
		},
		{
			name:          "non-thinking model",
			modelName:     "llama-2-7b-chat",
			architecture:  "llama",
			expectedThink: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gguf.Metadata{
				Name:         tt.modelName,
				Architecture: tt.architecture,
				ChatTemplate: "",
			}
			caps := DetectCapabilities(meta)
			assert.Equal(t, tt.expectedThink, caps.Thinking, "thinking capability mismatch")
		})
	}
}

// TestToolsDetectionFromChatTemplate 测试从 chat_template 检测 tools 能力
func TestToolsDetectionFromChatTemplate(t *testing.T) {
	tests := []struct {
		name         string
		chatTemplate string
		expected     bool
	}{
		{
			name:         "tool_call keyword",
			chatTemplate: "...{% if tool_call %}...",
			expected:     true,
		},
		{
			name:         "tool_calls keyword",
			chatTemplate: "...{{ tool_calls }}...",
			expected:     true,
		},
		{
			name:         "tools keyword",
			chatTemplate: "...{% for tool in tools %}...",
			expected:     true,
		},
		{
			name:         "mcp keyword",
			chatTemplate: "...mcp protocol...",
			expected:     true,
		},
		{
			name:         "function keyword",
			chatTemplate: "...function calling...",
			expected:     true,
		},
		{
			name:         "no tools keywords",
			chatTemplate: "...standard template...",
			expected:     false,
		},
		{
			name:         "empty chat template",
			chatTemplate: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gguf.Metadata{
				Name:         "test-model",
				Architecture: "llama",
				ChatTemplate: tt.chatTemplate,
			}
			caps := DetectCapabilities(meta)
			assert.Equal(t, tt.expected, caps.Tools, "tools capability mismatch")
		})
	}
}

// TestEmbeddingDetectionFromModelName 测试从模型名称检测 embedding 能力
func TestEmbeddingDetectionFromModelName(t *testing.T) {
	tests := []struct {
		name              string
		modelName         string
		architecture      string
		expectedEmbedding bool
	}{
		{
			name:              "embedding keyword",
			modelName:         "text-embedding-ada-002",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "embeddings keyword",
			modelName:         "all-embeddings-v1",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "e5 model",
			modelName:         "e5-large-v2",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "gte model",
			modelName:         "gte-base-en-v1.5",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "jina model",
			modelName:         "jina-embeddings-v2",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "nomic model",
			modelName:         "nomic-embed-text-v1",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "mxbai model",
			modelName:         "mxbai-embed-large-v1",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "arctic-embed model",
			modelName:         "snowflake-arctic-embed-l",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "bge model",
			modelName:         "bge-large-en-v1.5",
			architecture:      "bert",
			expectedEmbedding: true,
		},
		{
			name:              "non-embedding model",
			modelName:         "llama-2-7b-chat",
			architecture:      "llama",
			expectedEmbedding: false,
		},
		{
			name:              "case insensitive embedding",
			modelName:         "TEXT-EMBEDDING-LARGE",
			architecture:      "bert",
			expectedEmbedding: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gguf.Metadata{
				Name:         tt.modelName,
				Architecture: tt.architecture,
				ChatTemplate: "",
			}
			caps := DetectCapabilities(meta)
			assert.Equal(t, tt.expectedEmbedding, caps.Embedding, "embedding capability mismatch")
		})
	}
}

// TestRerankDetectionFromModelName 测试从模型名称检测 rerank 能力
func TestRerankDetectionFromModelName(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		architecture   string
		expectedRerank bool
	}{
		{
			name:           "rerank keyword",
			modelName:      "bge-rerank-v2",
			architecture:   "bert",
			expectedRerank: true,
		},
		{
			name:           "reranker keyword",
			modelName:      "bge-reranker-v2-m3",
			architecture:   "bert",
			expectedRerank: true,
		},
		{
			name:           "re-rank keyword",
			modelName:      "cross-re-rank-model",
			architecture:   "bert",
			expectedRerank: true,
		},
		{
			name:           "ranker keyword",
			modelName:      "text-ranker-v1",
			architecture:   "bert",
			expectedRerank: true,
		},
		{
			name:           "cross-encoder keyword",
			modelName:      "ms-marco-cross-encoder",
			architecture:   "bert",
			expectedRerank: true,
		},
		{
			name:           "crossencoder keyword",
			modelName:      "stsb-crossencoder",
			architecture:   "bert",
			expectedRerank: true,
		},
		{
			name:           "cross_encoder keyword",
			modelName:      "cross_encoder_v1",
			architecture:   "bert",
			expectedRerank: true,
		},
		{
			name:           "non-rerank model",
			modelName:      "llama-2-7b-chat",
			architecture:   "llama",
			expectedRerank: false,
		},
		{
			name:           "case insensitive reranker",
			modelName:      "BGE-RERANKER-V2-M3",
			architecture:   "bert",
			expectedRerank: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gguf.Metadata{
				Name:         tt.modelName,
				Architecture: tt.architecture,
				ChatTemplate: "",
			}
			caps := DetectCapabilities(meta)
			assert.Equal(t, tt.expectedRerank, caps.Rerank, "rerank capability mismatch")
		})
	}
}

// TestMutexConstraints 测试互斥约束：embedding/rerank 启用时 thinking 和 tools 必须为 false
func TestMutexConstraints(t *testing.T) {
	tests := []struct {
		name              string
		modelName         string
		chatTemplate      string
		expectedThinking  bool
		expectedTools     bool
		expectedEmbedding bool
		expectedRerank    bool
	}{
		{
			name:              "embedding model with thinking template - thinking should be false",
			modelName:         "bge-m3-embedding",
			chatTemplate:      "...enable_thinking...",
			expectedThinking:  false, // mutex: embedding disables thinking
			expectedTools:     false, // mutex: embedding disables tools
			expectedEmbedding: true,
			expectedRerank:    false,
		},
		{
			name:              "rerank model with tools template - tools should be false",
			modelName:         "ms-marco-reranker-v2",
			chatTemplate:      "...tool_calls...",
			expectedThinking:  false, // mutex: rerank disables thinking
			expectedTools:     false, // mutex: rerank disables tools
			expectedEmbedding: false,
			expectedRerank:    true,
		},
		{
			name:              "normal chat model with thinking and tools",
			modelName:         "llama-3-70b-chat",
			chatTemplate:      "...enable_thinking...tool_calls...",
			expectedThinking:  true,
			expectedTools:     true,
			expectedEmbedding: false,
			expectedRerank:    false,
		},
		{
			name:              "embedding model with deepseek-r1 name - thinking should be false",
			modelName:         "deepseek-r1-embedding",
			chatTemplate:      "",
			expectedThinking:  false, // mutex: embedding disables thinking
			expectedTools:     false,
			expectedEmbedding: true,
			expectedRerank:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gguf.Metadata{
				Name:         tt.modelName,
				Architecture: "llama",
				ChatTemplate: tt.chatTemplate,
			}
			caps := DetectCapabilities(meta)

			assert.Equal(t, tt.expectedThinking, caps.Thinking, "thinking capability mismatch")
			assert.Equal(t, tt.expectedTools, caps.Tools, "tools capability mismatch")
			assert.Equal(t, tt.expectedEmbedding, caps.Embedding, "embedding capability mismatch")
			assert.Equal(t, tt.expectedRerank, caps.Rerank, "rerank capability mismatch")
		})
	}
}

// TestCombinedThinkingDetection 测试 thinking 同时从 template 和 name 检测
func TestCombinedThinkingDetection(t *testing.T) {
	tests := []struct {
		name          string
		modelName     string
		chatTemplate  string
		expectedThink bool
	}{
		{
			name:          "both template and name have thinking keywords",
			modelName:     "deepseek-r1-7b",
			chatTemplate:  "...enable_thinking...",
			expectedThink: true,
		},
		{
			name:          "only template has thinking keyword",
			modelName:     "llama-2-7b",
			chatTemplate:  "...enable_thinking...",
			expectedThink: true,
		},
		{
			name:          "only name has thinking keyword",
			modelName:     "deepseek-r1-7b",
			chatTemplate:  "",
			expectedThink: true,
		},
		{
			name:          "neither has thinking keyword",
			modelName:     "llama-2-7b",
			chatTemplate:  "",
			expectedThink: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gguf.Metadata{
				Name:         tt.modelName,
				Architecture: "llama",
				ChatTemplate: tt.chatTemplate,
			}
			caps := DetectCapabilities(meta)
			assert.Equal(t, tt.expectedThink, caps.Thinking, "thinking capability mismatch")
		})
	}
}

// TestArchitectureInDetection 测试架构字段参与检测
func TestArchitectureInDetection(t *testing.T) {
	tests := []struct {
		name              string
		modelName         string
		architecture      string
		expectedEmbedding bool
	}{
		{
			name:              "embedding keyword in architecture",
			modelName:         "model-v1",
			architecture:      "embedding",
			expectedEmbedding: true,
		},
		{
			name:              "e5 in architecture",
			modelName:         "model-v1",
			architecture:      "e5-base",
			expectedEmbedding: true,
		},
		{
			name:              "bge in architecture",
			modelName:         "model-v1",
			architecture:      "bge-large",
			expectedEmbedding: true,
		},
		{
			name:              "rerank in architecture",
			modelName:         "model-v1",
			architecture:      "reranker",
			expectedEmbedding: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &gguf.Metadata{
				Name:         tt.modelName,
				Architecture: tt.architecture,
				ChatTemplate: "",
			}
			caps := DetectCapabilities(meta)
			assert.Equal(t, tt.expectedEmbedding, caps.Embedding, "embedding capability mismatch")
		})
	}
}
