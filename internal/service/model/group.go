package model

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

type ModelGroup struct {
	ID         string
	Models     []string
	Swap       bool
	Exclusive  bool
	Persistent bool
}

func (m *Manager) SetGroups(groups []*ModelGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups = make(map[string]*ModelGroup)
	for _, g := range groups {
		m.groups[g.ID] = g
	}
}

func (m *Manager) GetGroups() map[string]*ModelGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*ModelGroup, len(m.groups))
	for k, v := range m.groups {
		result[k] = v
	}
	return result
}

func (m *Manager) findGroupForModel(modelID string) *ModelGroup {
	for _, g := range m.groups {
		for _, id := range g.Models {
			if id == modelID {
				return g
			}
		}
	}
	return nil
}

func (m *Manager) swapBeforeLoad(modelID string) error {
	group := m.findGroupForModel(modelID)
	if group == nil || !group.Swap {
		return nil
	}

	for _, id := range group.Models {
		if id == modelID {
			continue
		}
		m.mu.RLock()
		status, exists := m.statuses[id]
		m.mu.RUnlock()
		if exists && status.State == StateLoaded {
			logger.Info("组交换: 停止同组模型", "stopping", id, "loading", modelID, "group", group.ID)
			if err := m.Unload(id); err != nil {
				logger.Warn("组交换: 停止模型失败", "modelId", id, "error", err)
			}
		}
	}

	if group.Exclusive {
		m.mu.RLock()
		var toStop []string
		for gid, g := range m.groups {
			if gid == group.ID {
				continue
			}
			if g.Persistent {
				continue
			}
			for _, id := range g.Models {
				status, exists := m.statuses[id]
				if exists && status.State == StateLoaded {
					toStop = append(toStop, id)
				}
			}
		}
		m.mu.RUnlock()

		for _, id := range toStop {
			logger.Info("互斥组: 停止非持久化模型", "stopping", id, "group", group.ID)
			if err := m.Unload(id); err != nil {
				logger.Warn("互斥组: 停止模型失败", "modelId", id, "error", err)
			}
		}
	}

	return nil
}
