package compat

import (
	"fmt"
	"strings"
	"sync"

	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
)

type ModelLookupIndex struct {
	byID        map[string]*model.Model
	byAlias     map[string]*model.Model
	byName      map[string]*model.Model
	mu          sync.RWMutex
	lastVersion int64
}

func NewModelLookupIndex() *ModelLookupIndex {
	return &ModelLookupIndex{
		byID:    make(map[string]*model.Model),
		byAlias: make(map[string]*model.Model),
		byName:  make(map[string]*model.Model),
	}
}

func (idx *ModelLookupIndex) Rebuild(models []*model.Model) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.byID = make(map[string]*model.Model)
	idx.byAlias = make(map[string]*model.Model)
	idx.byName = make(map[string]*model.Model)

	for _, m := range models {
		idx.byID[m.ID] = m
		if m.Alias != "" {
			idx.byAlias[m.Alias] = m
		}
		idx.byName[m.Name] = m
	}
}

func (idx *ModelLookupIndex) Find(identifier string) (*model.Model, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if m, ok := idx.byID[identifier]; ok {
		return m, true
	}

	if m, ok := idx.byAlias[identifier]; ok {
		return m, true
	}

	for name, m := range idx.byName {
		if strings.EqualFold(name, identifier) {
			return m, true
		}
	}

	lowerIdentifier := strings.ToLower(identifier)
	for _, m := range idx.byID {
		if strings.Contains(strings.ToLower(m.ID), lowerIdentifier) {
			return m, true
		}
	}

	return nil, false
}

func FindModelForAPI(modelMgr *model.Manager, idx *ModelLookupIndex, modelName string) (string, error) {
	statuses := modelMgr.ListStatus()

	if status, exists := statuses[modelName]; exists && status.State == model.StateLoaded {
		return modelName, nil
	}

	if m, ok := idx.Find(modelName); ok {
		if status, exists := statuses[m.ID]; exists && status.State == model.StateLoaded {
			return m.ID, nil
		}
		if _, err := modelMgr.EnsureLoaded(m.ID); err != nil {
			return "", fmt.Errorf("model %s not available: %w", modelName, err)
		}
		return m.ID, nil
	}

	return "", fmt.Errorf("model not found: %s", modelName)
}

func GetModelPort(modelMgr *model.Manager, modelID string) (int, error) {
	status, exists := modelMgr.GetStatus(modelID)
	if !exists {
		return 0, fmt.Errorf("model not loaded: %s", modelID)
	}

	if status.State != model.StateLoaded {
		return 0, fmt.Errorf("model not in loaded state: %s", modelID)
	}

	if status.Port == 0 {
		return 0, fmt.Errorf("model port not available: %s", modelID)
	}

	return status.Port, nil
}
