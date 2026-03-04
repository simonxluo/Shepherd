// Package model provides model capability detection functionality
package model

import (
	"strings"

	"github.com/shepherd-project/shepherd/Shepherd/internal/gguf"
	"github.com/shepherd-project/shepherd/Shepherd/internal/storage"
)

// DetectCapabilities 自动检测模型能力
// 参考 LlamacppServer 的 resolveModelType() 实现
// 基于 GGUF 元数据中的模型名称、架构和 chat_template 进行检测
func DetectCapabilities(meta *gguf.Metadata) *storage.Capabilities {
	if meta == nil {
		return &storage.Capabilities{}
	}

	caps := &storage.Capabilities{}

	// 组合模型名称和架构用于关键词检测
	combined := strings.ToLower(meta.Name + " " + meta.Architecture)
	chatTemplate := strings.ToLower(meta.ChatTemplate)

	// Rerank 检测：通过模型名称和架构关键词
	caps.Rerank = containsAny(combined, []string{
		"rerank", "re-rank", "reranker", "ranker",
		"cross-encoder", "crossencoder", "cross_encoder",
	})

	// Embedding 检测：通过模型名称和架构关键词
	caps.Embedding = containsAny(combined, []string{
		"embedding", "embeddings", "text-embedding", "embed",
		"e5", "gte", "jina", "nomic", "mxbai", "arctic-embed", "bge",
	})

	// Tools 检测：通过 chat_template 关键词
	caps.Tools = containsAny(chatTemplate, []string{
		"tool_call", "tool_calls", "tools", "mcp", "function",
	})

	// Thinking 检测：通过 chat_template 和模型名称
	caps.Thinking = containsAny(chatTemplate, []string{
		"enable_thinking", "thinking", "reasoning",
	}) || containsAny(combined, []string{
		"deepseek-r1", "deepseek r1", "qwq", "qwq-32b",
	})

	// 互斥逻辑：rerank/embedding 模型不支持 thinking/tools
	if caps.Rerank || caps.Embedding {
		caps.Thinking = false
		caps.Tools = false
	}

	return caps
}

// containsAny 检查字符串是否包含任一关键词
func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
