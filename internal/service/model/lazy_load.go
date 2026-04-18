package model

import (
	"fmt"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

func (m *Manager) EnsureLoaded(modelID string) (int, error) {
	m.mu.RLock()
	status, exists := m.statuses[modelID]
	m.mu.RUnlock()

	if exists && status.State == StateLoaded {
		return status.Port, nil
	}

	if exists && status.State == StateLoading {
		logger.Info("模型正在加载中，等待完成", "modelId", modelID)
		status.LoadWait.Wait()
		m.mu.RLock()
		status = m.statuses[modelID]
		m.mu.RUnlock()
		if status != nil && status.State == StateLoaded {
			return status.Port, nil
		}
		return 0, fmt.Errorf("model %s load failed", modelID)
	}

	logger.Info("惰性加载模型", "modelId", modelID)

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

	logger.Info("等待模型加载完成", "modelId", modelID)
	status.LoadWait.Wait()

	m.mu.RLock()
	status = m.statuses[modelID]
	m.mu.RUnlock()
	if status != nil && status.State == StateLoaded {
		return status.Port, nil
	}
	return 0, fmt.Errorf("model %s load failed", modelID)
}
