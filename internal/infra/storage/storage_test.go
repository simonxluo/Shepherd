// Package storage provides tests for storage implementations
package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryStore tests the in-memory storage implementation
func TestMemoryStore(t *testing.T) {
	store, err := NewMemoryStore()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test conversation creation
	conv := &Conversation{
		ID:           "test-conv-1",
		Model:        "llama-3.2",
		Title:        "Test Conversation",
		SystemPrompt: "You are a helpful assistant",
		Metadata:     map[string]interface{}{"key": "value"},
	}

	err = store.CreateConversation(ctx, conv)
	require.NoError(t, err)
	assert.NotEmpty(t, conv.ID)

	// Test conversation retrieval
	retrieved, err := store.GetConversation(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, retrieved.ID)
	assert.Equal(t, conv.Model, retrieved.Model)
	assert.Equal(t, conv.Title, retrieved.Title)
	assert.Equal(t, 0, retrieved.MessageCount)

	// Test message creation
	msg := &Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "Hello, world!",
		TokenCount:     5,
	}

	err = store.CreateMessage(ctx, msg)
	require.NoError(t, err)
	assert.NotEmpty(t, msg.ID)

	// Verify conversation message count was updated
	retrieved, err = store.GetConversation(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.MessageCount)

	// Test message retrieval
	messages, err := store.GetMessages(ctx, conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "Hello, world!", messages[0].Content)

	// Test conversation listing
	conv2 := &Conversation{
		Model: "llama-3.2",
		Title: "Another Conversation",
	}
	err = store.CreateConversation(ctx, conv2)
	require.NoError(t, err)

	convs, err := store.ListConversations(ctx, 10, 0)
	require.NoError(t, err)
	assert.Len(t, convs, 2)

	// Test conversation deletion
	err = store.DeleteConversation(ctx, conv.ID)
	require.NoError(t, err)

	_, err = store.GetConversation(ctx, conv.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrConversationNotFound, err)

	// Verify messages were deleted
	messages, err = store.GetMessages(ctx, conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 0)
}

// TestSQLiteStore tests the SQLite storage implementation
func TestSQLiteStore(t *testing.T) {
	// Use in-memory database for testing
	config := &SQLiteConfig{
		Path:      ":memory:",
		EnableWAL: false,
		Pragmas:   map[string]string{"synchronous": "OFF"},
	}

	store, err := NewSQLiteStore(config)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test conversation creation
	conv := &Conversation{
		Model:        "llama-3.2",
		Title:        "Test Conversation",
		SystemPrompt: "You are a helpful assistant",
		Metadata:     map[string]interface{}{"key": "value"},
	}

	err = store.CreateConversation(ctx, conv)
	require.NoError(t, err)
	assert.NotEmpty(t, conv.ID)

	// Test conversation retrieval
	retrieved, err := store.GetConversation(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, retrieved.ID)
	assert.Equal(t, conv.Model, retrieved.Model)
	assert.Equal(t, conv.Title, retrieved.Title)
	assert.False(t, retrieved.CreatedAt.IsZero())
	assert.False(t, retrieved.UpdatedAt.IsZero())

	// Test message creation
	msg := &Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "Hello, world!",
		TokenCount:     5,
		Metadata:       map[string]interface{}{"meta": "data"},
	}

	err = store.CreateMessage(ctx, msg)
	require.NoError(t, err)
	assert.NotEmpty(t, msg.ID)
	assert.False(t, msg.CreatedAt.IsZero())

	// Verify conversation message count was updated
	retrieved, err = store.GetConversation(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.MessageCount)

	// Test message retrieval
	messages, err := store.GetMessages(ctx, conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "Hello, world!", messages[0].Content)
	assert.False(t, messages[0].CreatedAt.IsZero())

	// Add assistant response
	assistantMsg := &Message{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        "Hi there!",
		TokenCount:     2,
	}
	err = store.CreateMessage(ctx, assistantMsg)
	require.NoError(t, err)

	// Verify message order
	messages, err = store.GetMessages(ctx, conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "assistant", messages[1].Role)

	// Test conversation listing with pagination
	conv2 := &Conversation{
		Model: "llama-3.2",
		Title: "Second Conversation",
	}
	err = store.CreateConversation(ctx, conv2)
	require.NoError(t, err)

	// Get first page
	convs, err := store.ListConversations(ctx, 1, 0)
	require.NoError(t, err)
	assert.Len(t, convs, 1)

	// Get second page
	convs, err = store.ListConversations(ctx, 1, 1)
	require.NoError(t, err)
	assert.Len(t, convs, 1)

	// Test conversation update
	conv.Title = "Updated Title"
	err = store.UpdateConversation(ctx, conv)
	require.NoError(t, err)

	retrieved, err = store.GetConversation(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)

	// Test conversation deletion
	err = store.DeleteConversation(ctx, conv.ID)
	require.NoError(t, err)

	_, err = store.GetConversation(ctx, conv.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrConversationNotFound, err)

	// Verify messages were cascade deleted
	messages, err = store.GetMessages(ctx, conv.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, messages, 0)

	// Verify second conversation still exists
	_, err = store.GetConversation(ctx, conv2.ID)
	require.NoError(t, err)
}

// TestStorageManager tests the storage manager
func TestStorageManager(t *testing.T) {
	// Test memory storage manager
	memoryConfig := &StorageConfig{
		Type: StorageTypeMemory,
	}

	mgr, err := NewManager(memoryConfig)
	require.NoError(t, err)
	require.NotNil(t, mgr.GetStore())
	mgr.Close()

	// Test SQLite storage manager
	sqliteConfig := &StorageConfig{
		Type: StorageTypeSQLite,
		SQLite: &SQLiteConfig{
			Path:      ":memory:",
			EnableWAL: false,
		},
	}

	mgr, err = NewManager(sqliteConfig)
	require.NoError(t, err)
	require.NotNil(t, mgr.GetStore())
	mgr.Close()

	// Test invalid storage type
	invalidConfig := &StorageConfig{
		Type: StorageType("invalid"),
	}

	_, err = NewManager(invalidConfig)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStorageType, err)

	// Test missing SQLite config
	missingSQLiteConfig := &StorageConfig{
		Type: StorageTypeSQLite,
	}

	_, err = NewManager(missingSQLiteConfig)
	assert.Error(t, err)
	assert.Equal(t, ErrMissingSQLiteConfig, err)
}

// TestConcurrentOperations tests concurrent storage operations
func TestConcurrentOperations(t *testing.T) {
	store, err := NewMemoryStore()
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Create multiple conversations concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			conv := &Conversation{
				Model: "llama-3.2",
				Title: fmt.Sprintf("Conversation %d", index),
			}
			err := store.CreateConversation(ctx, conv)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all conversations were created
	convs, err := store.ListConversations(ctx, 100, 0)
	require.NoError(t, err)
	assert.Len(t, convs, 10)
}

// TestGenerateID tests ID generation
func TestGenerateID(t *testing.T) {
	id1 := generateID("test")
	id2 := generateID("test")

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Contains(t, id1, "test-")
	assert.Contains(t, id2, "test-")
}
