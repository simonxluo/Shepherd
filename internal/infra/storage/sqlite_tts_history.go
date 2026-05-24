package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/simonxluo/Shepherd/internal/infra/storage/sqlcgen"
)

// sqlcTTSItemToDomain 将 sqlc 生成的 TtsHistory 转换为领域类型
func sqlcTTSItemToDomain(row *sqlcgen.TtsHistory) *TTSHistoryItem {
	item := &TTSHistoryItem{
		ID:        row.ID,
		Model:     row.Model,
		InputText: row.InputText,
		AudioPath: row.AudioPath,
		Format:    row.Format,
		Duration:  derefFloat64(row.Duration, 0),
		Favourite: derefInt64(row.Favourite, 0) != 0,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}

	if row.Params != nil && *row.Params != "" {
		if err := json.Unmarshal([]byte(*row.Params), &item.Params); err != nil {
			item.Params = make(map[string]interface{})
		}
	}

	return item
}

// CreateTTSHistory creates a new TTS history record
func (s *SQLiteStore) CreateTTSHistory(ctx context.Context, item *TTSHistoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = generateID("tts")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = timeNow()
	}

	paramsJSON, _ := json.Marshal(item.Params)
	duration := item.Duration
	favourite := int64(0)
	if item.Favourite {
		favourite = 1
	}
	params := string(paramsJSON)

	return s.queries.CreateTTSHistory(ctx, sqlcgen.CreateTTSHistoryParams{
		ID:        item.ID,
		Model:     item.Model,
		InputText: item.InputText,
		AudioPath: item.AudioPath,
		Format:    item.Format,
		Duration:  &duration,
		Favourite: &favourite,
		Params:    &params,
		CreatedAt: item.CreatedAt.Unix(),
	})
}

// GetTTSHistory retrieves a TTS history item by ID
func (s *SQLiteStore) GetTTSHistory(ctx context.Context, id string) (*TTSHistoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetTTSHistory(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTTSHistoryNotFound
		}
		return nil, fmt.Errorf("failed to get TTS history: %w", err)
	}

	return sqlcTTSItemToDomain(&row), nil
}

// ListTTSHistory lists TTS history items with pagination, ordered by created_at DESC
func (s *SQLiteStore) ListTTSHistory(ctx context.Context, limit, offset int, favouriteOnly *bool) ([]*TTSHistoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows []sqlcgen.TtsHistory
	var err error

	if favouriteOnly != nil && *favouriteOnly {
		rows, err = s.queries.ListTTSHistoryFav(ctx, sqlcgen.ListTTSHistoryFavParams{
			Limit:  int64(limit),
			Offset: int64(offset),
		})
	} else {
		rows, err = s.queries.ListTTSHistoryAll(ctx, sqlcgen.ListTTSHistoryAllParams{
			Limit:  int64(limit),
			Offset: int64(offset),
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list TTS history: %w", err)
	}

	items := make([]*TTSHistoryItem, 0, len(rows))
	for i := range rows {
		items = append(items, sqlcTTSItemToDomain(&rows[i]))
	}

	return items, nil
}

// UpdateTTSHistoryFavourite updates the favourite flag of a TTS history item
func (s *SQLiteStore) UpdateTTSHistoryFavourite(ctx context.Context, id string, favourite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	favInt := int64(0)
	if favourite {
		favInt = 1
	}

	err := s.queries.UpdateTTSHistoryFavourite(ctx, sqlcgen.UpdateTTSHistoryFavouriteParams{
		Favourite: &favInt,
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("failed to update TTS history favourite: %w", err)
	}

	return nil
}

// DeleteTTSHistory deletes a TTS history item by ID
func (s *SQLiteStore) DeleteTTSHistory(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.queries.DeleteTTSHistory(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete TTS history: %w", err)
	}

	return nil
}
