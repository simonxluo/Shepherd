package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/comm/storage/sqlcgen"
)

// sqlcConvToDomain converts a sqlc-generated Conversation to the domain type.
func sqlcConvToDomain(row *sqlcgen.Conversation) *Conversation {
	conv := &Conversation{
		ID:        row.ID,
		Model:     row.Model,
		Title:     derefString(row.Title, ""),
		SystemPrompt: derefString(row.SystemPrompt, ""),
		MessageCount: int(derefInt64(row.MessageCount, 0)),
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}

	if row.Metadata != nil && *row.Metadata != "" {
		if !utils.UnmarshalQuietly([]byte(*row.Metadata), &conv.Metadata, "会话元数据") {
			conv.Metadata = make(map[string]interface{})
		}
	}

	return conv
}

// CreateConversation creates a new conversation
func (s *SQLiteStore) CreateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conv.ID == "" {
		conv.ID = generateID("conv")
	}

	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = timeNow()
	}

	if conv.UpdatedAt.IsZero() {
		conv.UpdatedAt = timeNow()
	}

	metadataJSON, _ := json.Marshal(conv.Metadata)
	title := conv.Title
	systemPrompt := conv.SystemPrompt
	messageCount := int64(conv.MessageCount)
	metadata := string(metadataJSON)

	return s.queries.CreateConversation(ctx, sqlcgen.CreateConversationParams{
		ID:           conv.ID,
		Model:        conv.Model,
		Title:        &title,
		SystemPrompt: &systemPrompt,
		MessageCount: &messageCount,
		CreatedAt:    conv.CreatedAt.Unix(),
		UpdatedAt:    conv.UpdatedAt.Unix(),
		Metadata:     &metadata,
	})
}

// GetConversation retrieves a conversation by ID
func (s *SQLiteStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetConversation(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrConversationNotFound
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return sqlcConvToDomain(&row), nil
}

// ListConversations lists all conversations
func (s *SQLiteStore) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListConversations(ctx, sqlcgen.ListConversationsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	convs := make([]*Conversation, 0, len(rows))
	for i := range rows {
		convs = append(convs, sqlcConvToDomain(&rows[i]))
	}

	return convs, nil
}

// UpdateConversation updates an existing conversation
func (s *SQLiteStore) UpdateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv.UpdatedAt = timeNow()

	metadataJSON, _ := json.Marshal(conv.Metadata)
	title := conv.Title
	systemPrompt := conv.SystemPrompt
	messageCount := int64(conv.MessageCount)
	metadata := string(metadataJSON)

	err := s.queries.UpdateConversation(ctx, sqlcgen.UpdateConversationParams{
		Model:        conv.Model,
		Title:        &title,
		SystemPrompt: &systemPrompt,
		MessageCount: &messageCount,
		UpdatedAt:    conv.UpdatedAt.Unix(),
		Metadata:     &metadata,
		ID:           conv.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return nil
}

// DeleteConversation deletes a conversation and its messages
func (s *SQLiteStore) DeleteConversation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.queries.DeleteConversation(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	return nil
}
