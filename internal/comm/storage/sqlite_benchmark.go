package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/storage/sqlcgen"
)

// sqlcBenchmarkToDomain converts a sqlc-generated Benchmark to the domain type.
func sqlcBenchmarkToDomain(row *sqlcgen.Benchmark) *Benchmark {
	b := &Benchmark{
		ID:        row.ID,
		ModelID:   row.ModelID,
		ModelName: row.ModelName,
		Status:    row.Status,
		Command:   derefString(row.Command, ""),
		Error:     derefString(row.Error, ""),
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}

	if row.Config != nil && *row.Config != "" {
		if err := json.Unmarshal([]byte(*row.Config), &b.Config); err != nil {
			b.Config = make(map[string]interface{})
		}
	}

	if row.Metrics != nil && *row.Metrics != "" {
		if err := json.Unmarshal([]byte(*row.Metrics), &b.Metrics); err != nil {
			b.Metrics = make(map[string]interface{})
		}
	}

	if row.StartedAt != nil {
		t := time.Unix(*row.StartedAt, 0).UTC()
		b.StartedAt = &t
	}

	if row.FinishedAt != nil {
		t := time.Unix(*row.FinishedAt, 0).UTC()
		b.FinishedAt = &t
	}

	return b
}

// CreateBenchmark creates a new benchmark task
func (s *SQLiteStore) CreateBenchmark(ctx context.Context, benchmark *Benchmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configJSON, err := json.Marshal(benchmark.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark config: %w", err)
	}

	metricsJSON, err := json.Marshal(benchmark.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark metrics: %w", err)
	}

	command := benchmark.Command
	config := string(configJSON)
	metrics := string(metricsJSON)
	errorMsg := benchmark.Error

	return s.queries.CreateBenchmark(ctx, sqlcgen.CreateBenchmarkParams{
		ID:         benchmark.ID,
		ModelID:    benchmark.ModelID,
		ModelName:  benchmark.ModelName,
		Status:     benchmark.Status,
		Command:    &command,
		Config:     &config,
		Metrics:    &metrics,
		Error:      &errorMsg,
		CreatedAt:  benchmark.CreatedAt.Unix(),
		StartedAt:  timeToUnixPtr(benchmark.StartedAt),
		FinishedAt: timeToUnixPtr(benchmark.FinishedAt),
	})
}

// GetBenchmark retrieves a benchmark by ID
func (s *SQLiteStore) GetBenchmark(ctx context.Context, id string) (*Benchmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetBenchmark(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBenchmarkNotFound
		}
		return nil, err
	}

	return sqlcBenchmarkToDomain(&row), nil
}

// ListBenchmarks lists benchmarks with optional filtering
func (s *SQLiteStore) ListBenchmarks(ctx context.Context, modelID string, limit, offset int) ([]*Benchmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows []sqlcgen.Benchmark
	var err error

	if modelID != "" {
		rows, err = s.queries.ListBenchmarks(ctx, sqlcgen.ListBenchmarksParams{
			ModelID: modelID,
			Limit:   int64(limit),
			Offset:  int64(offset),
		})
	} else {
		rows, err = s.queries.ListBenchmarksAll(ctx, sqlcgen.ListBenchmarksAllParams{
			Limit:  int64(limit),
			Offset: int64(offset),
		})
	}

	if err != nil {
		return nil, err
	}

	benchmarks := make([]*Benchmark, 0, len(rows))
	for i := range rows {
		benchmarks = append(benchmarks, sqlcBenchmarkToDomain(&rows[i]))
	}

	return benchmarks, nil
}

// UpdateBenchmark updates an existing benchmark
func (s *SQLiteStore) UpdateBenchmark(ctx context.Context, benchmark *Benchmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configJSON, err := json.Marshal(benchmark.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark config: %w", err)
	}

	metricsJSON, err := json.Marshal(benchmark.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark metrics: %w", err)
	}

	command := benchmark.Command
	config := string(configJSON)
	metrics := string(metricsJSON)
	errorMsg := benchmark.Error

	return s.queries.UpdateBenchmark(ctx, sqlcgen.UpdateBenchmarkParams{
		ModelID:    benchmark.ModelID,
		ModelName:  benchmark.ModelName,
		Status:     benchmark.Status,
		Command:    &command,
		Config:     &config,
		Metrics:    &metrics,
		Error:      &errorMsg,
		StartedAt:  timeToUnixPtr(benchmark.StartedAt),
		FinishedAt: timeToUnixPtr(benchmark.FinishedAt),
		ID:         benchmark.ID,
	})
}

// DeleteBenchmark deletes a benchmark by ID
func (s *SQLiteStore) DeleteBenchmark(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.queries.DeleteBenchmark(ctx, id)
}
