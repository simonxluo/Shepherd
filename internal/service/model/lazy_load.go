package model

import (
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
// Uses a default CtxSize of 4096. Intended for protocol compatibility layers
// to automatically load models upon receiving inference requests.
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

	req := &LoadRequest{
		ModelID: modelID,
		CtxSize: 4096,
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
