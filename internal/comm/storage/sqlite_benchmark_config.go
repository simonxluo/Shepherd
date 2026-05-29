package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/comm/storage/sqlcgen"
)

// sqlcBenchmarkConfigToDomain converts a sqlc-generated BenchmarkConfig to the domain type.
func sqlcBenchmarkConfigToDomain(row *sqlcgen.BenchmarkConfig) *BenchmarkConfig {
	c := &BenchmarkConfig{
		Name:         row.Name,
		ModelID:      row.ModelID,
		ModelName:    row.ModelName,
		LlamaCppPath: row.LlamacppPath,
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
	}

	if row.Devices != nil && *row.Devices != "" {
		if !utils.UnmarshalQuietly([]byte(*row.Devices), &c.Devices, "设备配置") {
			c.Devices = []string{}
		}
	}

	if row.Params != nil && *row.Params != "" {
		if !utils.UnmarshalQuietly([]byte(*row.Params), &c.Params, "模型参数") {
			c.Params = make(map[string]string)
		}
	}

	return c
}

// CreateBenchmarkConfig creates a new benchmark configuration
func (s *SQLiteStore) CreateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	devicesJSON, _ := json.Marshal(config.Devices)
	paramsJSON, _ := json.Marshal(config.Params)
	devices := string(devicesJSON)
	params := string(paramsJSON)

	return s.queries.CreateBenchmarkConfig(ctx, sqlcgen.CreateBenchmarkConfigParams{
		Name:         config.Name,
		ModelID:      config.ModelID,
		ModelName:    config.ModelName,
		LlamacppPath: config.LlamaCppPath,
		Devices:      &devices,
		Params:       &params,
		CreatedAt:    config.CreatedAt.Unix(),
	})
}

// GetBenchmarkConfig retrieves a benchmark config by name
func (s *SQLiteStore) GetBenchmarkConfig(ctx context.Context, name string) (*BenchmarkConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetBenchmarkConfig(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBenchmarkConfigNotFound
		}
		return nil, err
	}

	return sqlcBenchmarkConfigToDomain(&row), nil
}

// ListBenchmarkConfigs lists all benchmark configurations
func (s *SQLiteStore) ListBenchmarkConfigs(ctx context.Context, limit, offset int) ([]*BenchmarkConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListBenchmarkConfigs(ctx, sqlcgen.ListBenchmarkConfigsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}

	configs := make([]*BenchmarkConfig, 0, len(rows))
	for i := range rows {
		configs = append(configs, sqlcBenchmarkConfigToDomain(&rows[i]))
	}

	return configs, nil
}

// UpdateBenchmarkConfig updates an existing benchmark configuration
func (s *SQLiteStore) UpdateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	devicesJSON, _ := json.Marshal(config.Devices)
	paramsJSON, _ := json.Marshal(config.Params)
	devices := string(devicesJSON)
	params := string(paramsJSON)

	return s.queries.UpdateBenchmarkConfig(ctx, sqlcgen.UpdateBenchmarkConfigParams{
		ModelID:      config.ModelID,
		ModelName:    config.ModelName,
		LlamacppPath: config.LlamaCppPath,
		Devices:      &devices,
		Params:       &params,
		Name:         config.Name,
	})
}

// DeleteBenchmarkConfig deletes a benchmark configuration by name
func (s *SQLiteStore) DeleteBenchmarkConfig(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.queries.DeleteBenchmarkConfig(ctx, name)
}
