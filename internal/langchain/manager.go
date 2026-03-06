// Package langchain provides LangChainGo integration manager
package langchain

import (
	"context"
	"fmt"
	"sync"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

// Manager 管理 LangChainGo LLM 实例
type Manager struct {
	modelMgr    *model.Manager
	llmInstances map[string]*LlamaCPP // 模型ID -> LLM 实例
	mu          sync.RWMutex
	logger      *logger.Logger
}

// NewManager 创建新的 LangChainGo 管理器
func NewManager(modelMgr *model.Manager, log *logger.Logger) *Manager {
	return &Manager{
		modelMgr:    modelMgr,
		llmInstances: make(map[string]*LlamaCPP),
		logger:      log,
	}
}

// GetLLM 获取指定模型的 LLM 实例
// 如果实例不存在，会自动创建
func (m *Manager) GetLLM(modelID string, opts ...Option) (*LlamaCPP, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if llm, exists := m.llmInstances[modelID]; exists {
		return llm, nil
	}

	// 获取模型状态
	status, exists := m.modelMgr.GetStatus(modelID)
	if !exists {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	if status.State != model.StateLoaded {
		return nil, fmt.Errorf("model not loaded: %s (current state: %s)", modelID, status.State)
	}

	if status.Port == 0 {
		return nil, fmt.Errorf("model port not available: %s", modelID)
	}

	// 构建服务器 URL
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", status.Port)

	// 创建新的 LLM 实例
	llm, err := NewLlamaCPP(serverURL, modelID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM instance: %w", err)
	}

	// 缓存实例
	m.llmInstances[modelID] = llm

	m.logger.Infof("LangChainGo LLM 实例已创建: model=%s, url=%s", modelID, serverURL)

	return llm, nil
}

// RemoveLLM 移除指定模型的 LLM 实例（用于模型卸载时）
func (m *Manager) RemoveLLM(modelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.llmInstances[modelID]; exists {
		delete(m.llmInstances, modelID)
		m.logger.Infof("LangChainGo LLM 实例已移除: model=%s", modelID)
	}
}

// ClearAll 清空所有 LLM 实例
func (m *Manager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.llmInstances = make(map[string]*LlamaCPP)
	m.logger.Info("所有 LangChainGo LLM 实例已清空")
}

// ListModels 列出所有可用的模型
func (m *Manager) ListModels() []string {
	statuses := m.modelMgr.ListStatus()

	models := make([]string, 0, len(statuses))
	for modelID := range statuses {
		models = append(models, modelID)
	}

	return models
}

// ===== 高级功能 =====

// SimplePrompt 使用简单的提示模板生成文本
func (m *Manager) SimplePrompt(ctx context.Context, modelID, promptTemplate string, input map[string]interface{}, opts ...Option) (string, error) {
	// 获取 LLM 实例
	llm, err := m.GetLLM(modelID, opts...)
	if err != nil {
		return "", err
	}

	// 创建提示模板
	template := prompts.NewPromptTemplate(promptTemplate, []string{})

	// 如果有输入变量，提取变量名
	vars := make([]string, 0, len(input))
	for k := range input {
		vars = append(vars, k)
	}
	template = prompts.NewPromptTemplate(promptTemplate, vars)

	// 格式化提示
	prompt, err := template.Format(input)
	if err != nil {
		return "", fmt.Errorf("failed to format prompt: %w", err)
	}

	// 生成文本
	response, err := llm.Call(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return response, nil
}

// ChatPrompt 使用聊天提示模板生成对话
func (m *Manager) ChatPrompt(ctx context.Context, modelID string, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	// 获取 LLM 实例
	llm, err := m.GetLLM(modelID)
	if err != nil {
		return nil, err
	}

	// 生成响应
	response, err := llm.GenerateContent(ctx, messages, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate chat response: %w", err)
	}

	return response, nil
}

// StreamPrompt 流式生成文本
func (m *Manager) StreamPrompt(ctx context.Context, modelID string, messages []llms.MessageContent, opts ...llms.CallOption) (<-chan *llms.ContentResponse, error) {
	// 获取 LLM 实例
	llm, err := m.GetLLM(modelID)
	if err != nil {
		return nil, err
	}

	// 流式生成
	respChan, err := llm.GenerateContentStream(ctx, messages, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start streaming: %w", err)
	}

	return respChan, nil
}

// ===== 统计信息 =====

// GetStats 获取统计信息
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := m.modelMgr.ListStatus()
	loadedModels := make([]string, 0)
	for modelID, status := range statuses {
		if status.State == model.StateLoaded {
			loadedModels = append(loadedModels, modelID)
		}
	}

	return map[string]interface{}{
		"total_models":       len(statuses),
		"loaded_models":      len(loadedModels),
		"llm_instances":      len(m.llmInstances),
		"cached_instances":   m.llmInstances,
		"available_models":   loadedModels,
	}
}
