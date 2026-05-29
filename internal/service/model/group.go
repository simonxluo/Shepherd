package model

import (
	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// ModelGroup defines a model group supporting llama-swap style model swapping
// and exclusive loading behavior.
//
// Fields:
//   - Swap: models in the same group are mutually exclusive — loading one auto-unloads others
//   - Exclusive: exclusive mode — loading a model in this group also unloads non-persistent groups
//   - Persistent: persistent groups are never unloaded by exclusive rules
type ModelGroup struct {
	ID         string
	Models     []string
	Swap       bool
	Exclusive  bool
	Persistent bool
}

// loadGroupsFromConfig loads model groups from the YAML configuration.
// Groups define swap/exclusive model loading behavior (llama-swap style).
func (m *Manager) loadGroupsFromConfig() {
	cfg := m.config
	if m.configMgr != nil {
		cfg = m.configMgr.Get()
	}

	if len(cfg.Model.Groups) == 0 {
		return
	}

	for _, gDef := range cfg.Model.Groups {
		if gDef.ID == "" || len(gDef.Models) == 0 {
			logger.Warnf("loadGroupsFromConfig: skipping invalid group definition: id=%q, models=%d", gDef.ID, len(gDef.Models))
			continue
		}
		m.groups[gDef.ID] = &ModelGroup{
			ID:         gDef.ID,
			Models:     gDef.Models,
			Swap:       gDef.Swap,
			Exclusive:  gDef.Exclusive,
			Persistent: gDef.Persistent,
		}
		logger.Infof("loadGroupsFromConfig: loaded group: id=%s, models=%v, swap=%v, exclusive=%v, persistent=%v",
			gDef.ID, gDef.Models, gDef.Swap, gDef.Exclusive, gDef.Persistent)
	}

	logger.Infof("loadGroupsFromConfig: loaded %d model group(s)", len(m.groups))
}

// findGroupForModel returns the group that contains the given model ID, or nil if not in any group.
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

// swapBeforeLoad executes group swap logic before loading a model.
// If the model belongs to a Swap group, it unloads other loaded models in that group.
// If the group is also marked Exclusive, it additionally unloads models from
// other non-Persistent groups.
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
			logger.Infof("组交换: 停止同组模型: stopping=%s, loading=%s, group=%s", id, modelID, group.ID)
			if err := m.Unload(id); err != nil {
				logger.Warnf("组交换: 停止模型失败: modelId=%s, error=%v", id, err)
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
			logger.Infof("互斥组: 停止非持久化模型: stopping=%s, group=%s", id, group.ID)
			if err := m.Unload(id); err != nil {
				logger.Warnf("互斥组: 停止模型失败: modelId=%s, error=%v", id, err)
			}
		}
	}

	return nil
}
