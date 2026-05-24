package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/simonxluo/Shepherd/internal/infra/storage/sqlcgen"
)

// sqlcModelLoadConfigToDomain 将 sqlc 生成的 ModelLoadConfig 转换为领域类型
func sqlcModelLoadConfigToDomain(row *sqlcgen.ModelLoadConfig) *ModelLoadConfig {
	c := &ModelLoadConfig{
		ID:        row.ID,
		NodeID:    row.NodeID,
		ModelID:   row.ModelID,
		ModelName: row.ModelName,
		Name:      row.Name,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}

	if row.Config != "" {
		if err := json.Unmarshal([]byte(row.Config), &c.Config); err != nil {
			c.Config = make(map[string]interface{})
		}
	}

	return c
}

// SaveModelLoadConfig saves or updates a model load configuration
func (s *SQLiteStore) SaveModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.ID == "" {
		config.ID = generateID("mlcfg")
	}

	now := timeNow()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal model load config: %w", err)
	}

	return s.queries.SaveModelLoadConfig(ctx, sqlcgen.SaveModelLoadConfigParams{
		ID:        config.ID,
		NodeID:    config.NodeID,
		ModelID:   config.ModelID,
		ModelName: config.ModelName,
		Config:    string(configJSON),
		CreatedAt: config.CreatedAt.Unix(),
		UpdatedAt: config.UpdatedAt.Unix(),
		Name:      config.Name,
	})
}

// GetModelLoadConfig retrieves a model load configuration by node ID and model ID
func (s *SQLiteStore) GetModelLoadConfig(ctx context.Context, nodeID, modelID string) (*ModelLoadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetModelLoadConfig(ctx, sqlcgen.GetModelLoadConfigParams{
		NodeID:  nodeID,
		ModelID: modelID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrModelLoadConfigNotFound
		}
		return nil, fmt.Errorf("failed to get model load config: %w", err)
	}

	return sqlcModelLoadConfigToDomain(&row), nil
}

// DeleteModelLoadConfig deletes a model load configuration by node ID and model ID
func (s *SQLiteStore) DeleteModelLoadConfig(ctx context.Context, nodeID, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.queries.DeleteModelLoadConfig(ctx, sqlcgen.DeleteModelLoadConfigParams{
		NodeID:  nodeID,
		ModelID: modelID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete model load config: %w", err)
	}

	return nil
}

// ListModelLoadConfigs returns all load configs (default + named) for a model on a node
func (s *SQLiteStore) ListModelLoadConfigs(ctx context.Context, nodeID, modelID string) ([]*ModelLoadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListModelLoadConfigs(ctx, sqlcgen.ListModelLoadConfigsParams{
		NodeID:  nodeID,
		ModelID: modelID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list model load configs: %w", err)
	}

	configs := make([]*ModelLoadConfig, 0, len(rows))
	for i := range rows {
		configs = append(configs, sqlcModelLoadConfigToDomain(&rows[i]))
	}

	return configs, nil
}

// SaveNamedModelLoadConfig saves a named load config preset
func (s *SQLiteStore) SaveNamedModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
	if config.Name == "" {
		return fmt.Errorf("named config requires a non-empty name")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if config.ID == "" {
		config.ID = generateID("mlcfg")
	}

	now := timeNow()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal model load config: %w", err)
	}

	return s.queries.SaveModelLoadConfig(ctx, sqlcgen.SaveModelLoadConfigParams{
		ID:        config.ID,
		NodeID:    config.NodeID,
		ModelID:   config.ModelID,
		ModelName: config.ModelName,
		Config:    string(configJSON),
		CreatedAt: config.CreatedAt.Unix(),
		UpdatedAt: config.UpdatedAt.Unix(),
		Name:      config.Name,
	})
}

// DeleteNamedModelLoadConfig deletes a named load config preset
func (s *SQLiteStore) DeleteNamedModelLoadConfig(ctx context.Context, nodeID, modelID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.queries.DeleteNamedModelLoadConfig(ctx, sqlcgen.DeleteNamedModelLoadConfigParams{
		NodeID:  nodeID,
		ModelID: modelID,
		Name:    name,
	})
	if err != nil {
		return fmt.Errorf("failed to delete named model load config: %w", err)
	}

	return nil
}
