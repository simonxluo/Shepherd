// Package api provides shared helpers for API compatibility layers (OpenAI, Anthropic, Ollama).
package handler

import (
	"fmt"
	"strings"
	"sync"

	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
)

// ModelLookupIndex accelerates model lookups across API compatibility handlers.
// It builds an in-memory index for O(1) lookups by ID, alias, or name.
type ModelLookupIndex struct {
	byID    map[string]*model.Model
	byAlias map[string]*model.Model
	byName  map[string]*model.Model
	mu      sync.RWMutex
}

// NewModelLookupIndex creates a new ModelLookupIndex.
func NewModelLookupIndex() *ModelLookupIndex {
	return &ModelLookupIndex{
		byID:    make(map[string]*model.Model),
		byAlias: make(map[string]*model.Model),
		byName:  make(map[string]*model.Model),
	}
}

// Rebuild rebuilds the index from the given model list.
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

// Find looks up a model by identifier with priority: ID > alias > name (case-insensitive) > ID substring (case-insensitive).
func (idx *ModelLookupIndex) Find(identifier string) (*model.Model, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Exact ID match
	if m, ok := idx.byID[identifier]; ok {
		return m, true
	}

	// Exact alias match
	if m, ok := idx.byAlias[identifier]; ok {
		return m, true
	}

	// Case-insensitive name match
	for name, m := range idx.byName {
		if strings.EqualFold(name, identifier) {
			return m, true
		}
	}

	// Case-insensitive ID substring match
	lowerIdentifier := strings.ToLower(identifier)
	for _, m := range idx.byID {
		if strings.Contains(strings.ToLower(m.ID), lowerIdentifier) {
			return m, true
		}
	}

	return nil, false
}

// FindModelForAPI finds a loaded model by name, alias, or ID.
// It returns the resolved model ID.
//
// Lookup priority:
//  1. Exact ID match (must be in loaded state)
//  2. Alias or name match via index (must be in loaded state)
//
// Returns an error if the model is not found or not loaded.
func FindModelForAPI(modelMgr *model.Manager, idx *ModelLookupIndex, modelName string) (string, error) {
	statuses := modelMgr.ListStatus()

	// First try exact match with ID
	if status, exists := statuses[modelName]; exists && status.State == model.StateLoaded {
		return modelName, nil
	}

	// Use index for broader lookup
	if m, ok := idx.Find(modelName); ok {
		if status, exists := statuses[m.ID]; exists && status.State == model.StateLoaded {
			return m.ID, nil
		}
	}

	return "", fmt.Errorf("model not found: %s", modelName)
}

// GetModelPort returns the port for a loaded model.
// It verifies the model exists, is in loaded state, and has a valid port.
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
