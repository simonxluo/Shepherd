// Package model provides tests for model capability detection
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/shepherd-project/shepherd/Shepherd/internal/gguf"
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
