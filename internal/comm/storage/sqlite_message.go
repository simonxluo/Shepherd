package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/comm/storage/sqlcgen"
)

// sqlcMsgToDomain converts a sqlc-generated Message to the domain type.
func sqlcMsgToDomain(row *sqlcgen.Message) *Message {
	msg := &Message{
		ID:             row.ID,
		ConversationID: row.ConversationID,
		Role:           row.Role,
		Content:        row.Content,
		Name:           derefString(row.Name, ""),
		TokenCount:     int(derefInt64(row.TokenCount, 0)),
		CreatedAt:      time.Unix(row.CreatedAt, 0).UTC(),
	}

	if row.Metadata != nil && *row.Metadata != "" {
		if !utils.UnmarshalQuietly([]byte(*row.Metadata), &msg.Metadata, "消息元数据") {
			msg.Metadata = make(map[string]interface{})
		}
	}

	return msg
}

// CreateMessage creates a new message
func (s *SQLiteStore) CreateMessage(ctx context.Context, msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		msg.ID = generateID("msg")
	}

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = timeNow()
	}

	metadataJSON, _ := json.Marshal(msg.Metadata)
	name := msg.Name
	tokenCount := int64(msg.TokenCount)
	metadata := string(metadataJSON)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	err = qtx.CreateMessage(ctx, sqlcgen.CreateMessageParams{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		Role:           msg.Role,
		Content:        msg.Content,
		Name:           &name,
		TokenCount:     &tokenCount,
		CreatedAt:      msg.CreatedAt.Unix(),
		Metadata:       &metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	err = qtx.IncrementConversationMessageCount(ctx, sqlcgen.IncrementConversationMessageCountParams{
		UpdatedAt: timeNow().Unix(),
		ID:        msg.ConversationID,
	})
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return tx.Commit()
}

// GetMessages retrieves messages for a conversation
func (s *SQLiteStore) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.GetMessages(ctx, sqlcgen.GetMessagesParams{
		ConversationID: conversationID,
		Limit:          int64(limit),
		Offset:         int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*Message, 0, len(rows))
	for i := range rows {
		messages = append(messages, sqlcMsgToDomain(&rows[i]))
	}

	return messages, nil
}

// DeleteMessages deletes all messages for a conversation
func (s *SQLiteStore) DeleteMessages(ctx context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.queries.DeleteMessages(ctx, conversationID); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	if err := s.queries.ResetConversationMessageCount(ctx, sqlcgen.ResetConversationMessageCountParams{
		UpdatedAt: timeNow().Unix(),
		ID:        conversationID,
	}); err != nil {
		return err
	}

	return nil
}
