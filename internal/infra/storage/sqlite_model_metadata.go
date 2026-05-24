package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/infra/storage/sqlcgen"
)

// sqlcModelMetadataToDomain 将 sqlc 生成的 ModelMetadatum 转换为领域类型
func sqlcModelMetadataToDomain(row *sqlcgen.ModelMetadatum) *ModelMetadata {
	m := &ModelMetadata{
		ModelID:     row.ModelID,
		NodeID:      derefString(row.NodeID, ""),
		StoragePath: derefString(row.StoragePath, ""),
		Alias:       derefString(row.Alias, ""),
		Favourite:   derefInt64(row.Favourite, 0) != 0,
		Description: derefString(row.Description, ""),
		LoadCount:   int(derefInt64(row.LoadCount, 0)),
		TotalTokens: derefInt64(row.TotalTokens, 0),
		CreatedAt:   time.Unix(row.CreatedAt, 0),
		UpdatedAt:   time.Unix(row.UpdatedAt, 0),
	}

	if row.Tags != nil && *row.Tags != "" {
		if !utils.UnmarshalQuietly([]byte(*row.Tags), &m.Tags, "模型标签") {
			m.Tags = []string{}
		}
	}

	if row.Capabilities != nil && *row.Capabilities != "" {
		if !utils.UnmarshalQuietly([]byte(*row.Capabilities), &m.Capabilities, "模型能力") {
			m.Capabilities = &Capabilities{}
		}
	}

	if row.LastLoaded != nil {
		ld := time.Unix(*row.LastLoaded, 0)
		m.LastLoaded = &ld
	}

	return m
}

// SaveModelMetadata saves or updates model metadata
func (s *SQLiteStore) SaveModelMetadata(ctx context.Context, metadata *ModelMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := timeNow()

	// 检查是否已存在记录
	existingCreatedAt, err := s.queries.GetModelMetadataCreatedAt(ctx, metadata.ModelID)
	if err == sql.ErrNoRows {
		// 新建
		if metadata.CreatedAt.IsZero() {
			metadata.CreatedAt = now
		}
		metadata.UpdatedAt = now

		tagsJSON, _ := json.Marshal(metadata.Tags)
		capsJSON, _ := json.Marshal(metadata.Capabilities)
		tags := string(tagsJSON)
		caps := string(capsJSON)
		nodeID := metadata.NodeID
		storagePath := metadata.StoragePath
		alias := metadata.Alias
		favourite := int64(0)
		if metadata.Favourite {
			favourite = 1
		}
		description := metadata.Description
		loadCount := int64(metadata.LoadCount)
		totalTokens := metadata.TotalTokens

		return s.queries.InsertModelMetadata(ctx, sqlcgen.InsertModelMetadataParams{
			ModelID:      metadata.ModelID,
			NodeID:       &nodeID,
			StoragePath:  &storagePath,
			Alias:        &alias,
			Favourite:    &favourite,
			Tags:         &tags,
			Description:  &description,
			LoadCount:    &loadCount,
			LastLoaded:   timeToUnixPtr(metadata.LastLoaded),
			TotalTokens:  &totalTokens,
			Capabilities: &caps,
			CreatedAt:    metadata.CreatedAt.Unix(),
			UpdatedAt:    metadata.UpdatedAt.Unix(),
		})
	} else if err != nil {
		return fmt.Errorf("failed to query model metadata: %w", err)
	}

	// 更新已有记录
	metadata.CreatedAt = time.Unix(existingCreatedAt, 0)
	metadata.UpdatedAt = now

	tagsJSON, _ := json.Marshal(metadata.Tags)
	capsJSON, _ := json.Marshal(metadata.Capabilities)
	tags := string(tagsJSON)
	caps := string(capsJSON)
	nodeID := metadata.NodeID
	storagePath := metadata.StoragePath
	alias := metadata.Alias
	favourite := int64(0)
	if metadata.Favourite {
		favourite = 1
	}
	description := metadata.Description
	loadCount := int64(metadata.LoadCount)
	totalTokens := metadata.TotalTokens

	return s.queries.UpdateModelMetadata(ctx, sqlcgen.UpdateModelMetadataParams{
		NodeID:       &nodeID,
		StoragePath:  &storagePath,
		Alias:        &alias,
		Favourite:    &favourite,
		Tags:         &tags,
		Description:  &description,
		LoadCount:    &loadCount,
		LastLoaded:   timeToUnixPtr(metadata.LastLoaded),
		TotalTokens:  &totalTokens,
		Capabilities: &caps,
		UpdatedAt:    metadata.UpdatedAt.Unix(),
		ModelID:      metadata.ModelID,
	})
}

// GetModelMetadata retrieves metadata for a single model
func (s *SQLiteStore) GetModelMetadata(ctx context.Context, modelID string) (*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row, err := s.queries.GetModelMetadata(ctx, modelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrModelMetadataNotFound
		}
		return nil, fmt.Errorf("failed to get model metadata: %w", err)
	}

	return sqlcModelMetadataToDomain(&row), nil
}

// ListModelMetadata lists model metadata with pagination
func (s *SQLiteStore) ListModelMetadata(ctx context.Context, limit, offset int) ([]*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.queries.ListModelMetadata(ctx, sqlcgen.ListModelMetadataParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list model metadata: %w", err)
	}

	metadatas := make([]*ModelMetadata, 0, len(rows))
	for i := range rows {
		metadatas = append(metadatas, sqlcModelMetadataToDomain(&rows[i]))
	}

	return metadatas, nil
}

// DeleteModelMetadata deletes metadata for a model
func (s *SQLiteStore) DeleteModelMetadata(ctx context.Context, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.queries.DeleteModelMetadata(ctx, modelID)
	if err != nil {
		return fmt.Errorf("failed to delete model metadata: %w", err)
	}

	return nil
}

// GetAllModelMetadata retrieves all model metadata as a map
func (s *SQLiteStore) GetAllModelMetadata(ctx context.Context) (map[string]*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.queries.GetAllModelMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all model metadata: %w", err)
	}

	result := make(map[string]*ModelMetadata, len(rows))
	for i := range rows {
		m := sqlcModelMetadataToDomain(&rows[i])
		result[m.ModelID] = m
	}

	return result, nil
}
