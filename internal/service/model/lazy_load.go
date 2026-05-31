package model

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// EnsureLoaded ensures the specified model is in a loaded state (lazy loading).
//
// Behavior:
//   - If already loaded: returns its listening port immediately
//   - If currently loading: blocks until loading completes
//   - If not loaded: triggers an async load and blocks until complete
//
// Attempts to restore a previously saved load configuration from storage.
// Falls back to a default CtxSize of 4096 if no saved config exists.
// Returns the model's listening port or an error if loading fails.
func (m *Manager) EnsureLoaded(modelID string) (int, error) {
	m.mu.RLock()
	status, exists := m.statuses[modelID]
	m.mu.RUnlock()

	if exists && status.State == StateLoaded {
		return status.Port, nil
	}

	if exists && status.State == StateLoading {
		logger.Infof("模型正在加载中，等待完成: modelId=%s", modelID)
		status.LoadWait.Wait()
		m.mu.RLock()
		status = m.statuses[modelID]
		m.mu.RUnlock()
		if status != nil && status.State == StateLoaded {
			return status.Port, nil
		}
		return 0, fmt.Errorf("model %s load failed", modelID)
	}

	logger.Infof("惰性加载模型: modelId=%s", modelID)

	req := m.loadSavedConfig(modelID)
	if req == nil {
		req = &LoadRequest{
			ModelID: modelID,
			CtxSize: 4096,
		}
	}

	result, err := m.LoadAsync(req)
	if err != nil {
		return 0, err
	}

	if result.AlreadyLoaded {
		return result.Port, nil
	}

	m.mu.RLock()
	status = m.statuses[modelID]
	m.mu.RUnlock()

	if status == nil {
		return 0, fmt.Errorf("model %s status not found after load initiation", modelID)
	}

	logger.Infof("等待模型加载完成: modelId=%s", modelID)
	status.LoadWait.Wait()

	m.mu.RLock()
	status = m.statuses[modelID]
	m.mu.RUnlock()
	if status != nil && status.State == StateLoaded {
		return status.Port, nil
	}
	return 0, fmt.Errorf("model %s load failed", modelID)
}

// loadSavedConfig attempts to load a previously saved model load configuration from storage.
// Returns nil if no config is found or if deserialization fails.
func (m *Manager) loadSavedConfig(modelID string) *LoadRequest {
	if m.storageMgr == nil {
		return nil
	}

	store := m.storageMgr.GetStore()
	if store == nil {
		return nil
	}

	// 获取节点 ID，与 handler 中的 getNodeID() 逻辑一致
	nodeID := "local"
	if m.config != nil && m.config.Node.ID != "" {
		nodeID = m.config.Node.ID
	}

	cfg, err := store.GetModelLoadConfig(context.Background(), nodeID, modelID)
	if err != nil {
		return nil
	}

	// 将 Config (map[string]interface{}) 序列化后反序列化为 LoadRequest
	jsonBytes, err := json.Marshal(cfg.Config)
	if err != nil {
		return nil
	}

	var req LoadRequest
	if err := json.Unmarshal(jsonBytes, &req); err != nil {
		return nil
	}

	req.ModelID = modelID
	if req.CtxSize == 0 {
		req.CtxSize = 4096
	}

	logger.Infof("从存储恢复模型加载配置: modelId=%s, ctxSize=%d", modelID, req.CtxSize)
	return &req
}
