package model

import (
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// StartTTLChecker starts a background goroutine that periodically checks for idle models.
// Every 10 seconds it inspects all loaded models and unloads those that have exceeded
// their UnloadAfter threshold with no inflight requests.
// The goroutine is controlled by Manager.ctx and exits gracefully on Close().
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

// checkAndUnloadIdle inspects all loaded models and unloads those exceeding their TTL.
// A model is unloaded when:
//   - It is in StateLoaded
//   - TTL > 0 (TTL=0 means never auto-unload)
//   - No inflight requests (InflightCount == 0)
//   - Time since last request (or load time) exceeds UnloadAfter
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
		logger.Infof("TTL 过期，自动卸载模型: modelId=%s", id)
		if err := m.Unload(id); err != nil {
			logger.Warnf("TTL 自动卸载失败: modelId=%s, error=%v", id, err)
		}
	}
}
