package model

import (
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

func (m *Manager) StartTTLChecker() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.checkAndUnloadIdle()
			}
		}
	}()
}

func (m *Manager) checkAndUnloadIdle() {
	m.mu.RLock()
	var toUnload []string
	now := time.Now()
	for id, status := range m.statuses {
		if status.State != StateLoaded {
			continue
		}
		status.mu.Lock()
		ttl := status.UnloadAfter
		lastReq := status.LastRequestTime
		status.mu.Unlock()

		if ttl <= 0 {
			continue
		}
		if status.GetInflightCount() != 0 {
			continue
		}
		if lastReq.IsZero() {
			status.mu.Lock()
			lastReq = status.LoadedAt
			status.mu.Unlock()
		}
		if !lastReq.IsZero() && now.Sub(lastReq) > ttl {
			toUnload = append(toUnload, id)
		}
	}
	m.mu.RUnlock()

	for _, id := range toUnload {
		logger.Info("TTL 过期，自动卸载模型", "modelId", id)
		if err := m.Unload(id); err != nil {
			logger.Warn("TTL 自动卸载失败", "modelId", id, "error", err)
		}
	}
}
