// Package storage provides in-memory storage implementation
package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryStore implements Store interface with in-memory storage
type MemoryStore struct {
	mu               sync.RWMutex
	conversations    map[string]*Conversation
	messages         map[string][]*Message // conversation_id -> messages
	messagesByID     map[string]*Message
	benchmarks       map[string]*Benchmark
	benchmarkConfigs map[string]*BenchmarkConfig
	modelLoadConfigs map[string]*ModelLoadConfig // key: "nodeID:modelID"
	modelMetadata    map[string]*ModelMetadata   // key: modelID
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() (*MemoryStore, error) {
	return &MemoryStore{
		conversations:    make(map[string]*Conversation),
		messages:         make(map[string][]*Message),
		messagesByID:     make(map[string]*Message),
		benchmarks:       make(map[string]*Benchmark),
		benchmarkConfigs: make(map[string]*BenchmarkConfig),
		modelLoadConfigs: make(map[string]*ModelLoadConfig),
		modelMetadata:    make(map[string]*ModelMetadata),
	}, nil
}

// CreateConversation creates a new conversation
func (s *MemoryStore) CreateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conv.ID == "" {
		conv.ID = generateID("conv")
	}

	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = time.Now()
	}

	if conv.UpdatedAt.IsZero() {
		conv.UpdatedAt = time.Now()
	}

	s.conversations[conv.ID] = conv
	s.messages[conv.ID] = []*Message{}

	return nil
}

// GetConversation retrieves a conversation by ID
func (s *MemoryStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, exists := s.conversations[id]
	if !exists {
		return nil, ErrConversationNotFound
	}

	// Return a copy to avoid race conditions
	convCopy := *conv
	return &convCopy, nil
}

// ListConversations lists all conversations
func (s *MemoryStore) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	convs := make([]*Conversation, 0, len(s.conversations))

	for _, conv := range s.conversations {
		convs = append(convs, conv)
	}

	// Simple pagination (in production, should sort by updated_at)
	if offset >= len(convs) {
		return []*Conversation{}, nil
	}

	end := offset + limit
	if end > len(convs) {
		end = len(convs)
	}

	return convs[offset:end], nil
}

// UpdateConversation updates an existing conversation
func (s *MemoryStore) UpdateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.conversations[conv.ID]; !exists {
		return ErrConversationNotFound
	}

	conv.UpdatedAt = time.Now()
	s.conversations[conv.ID] = conv

	return nil
}

// DeleteConversation deletes a conversation and its messages
func (s *MemoryStore) DeleteConversation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.conversations[id]; !exists {
		return ErrConversationNotFound
	}

	delete(s.conversations, id)

	// Delete associated messages
	for _, msg := range s.messages[id] {
		delete(s.messagesByID, msg.ID)
	}
	delete(s.messages, id)

	return nil
}

// CreateMessage creates a new message
func (s *MemoryStore) CreateMessage(ctx context.Context, msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		msg.ID = generateID("msg")
	}

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	s.messagesByID[msg.ID] = msg
	s.messages[msg.ConversationID] = append(s.messages[msg.ConversationID], msg)

	// Update conversation message count and timestamp
	if conv, exists := s.conversations[msg.ConversationID]; exists {
		conv.MessageCount = len(s.messages[msg.ConversationID])
		conv.UpdatedAt = time.Now()
	}

	return nil
}

// GetMessages retrieves messages for a conversation
func (s *MemoryStore) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messages, exists := s.messages[conversationID]
	if !exists {
		return []*Message{}, nil
	}

	if offset >= len(messages) {
		return []*Message{}, nil
	}

	end := offset + limit
	if end > len(messages) {
		end = len(messages)
	}

	return messages[offset:end], nil
}

// DeleteMessages deletes all messages for a conversation
func (s *MemoryStore) DeleteMessages(ctx context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	messages, exists := s.messages[conversationID]
	if !exists {
		return nil
	}

	for _, msg := range messages {
		delete(s.messagesByID, msg.ID)
	}

	delete(s.messages, conversationID)

	// Update conversation message count
	if conv, exists := s.conversations[conversationID]; exists {
		conv.MessageCount = 0
		conv.UpdatedAt = time.Now()
	}

	return nil
}

// Benchmark operations (MemoryStore implementation)

func (s *MemoryStore) CreateBenchmark(ctx context.Context, benchmark *Benchmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if benchmark.ID == "" {
		benchmark.ID = generateID("bench")
	}

	if benchmark.CreatedAt.IsZero() {
		benchmark.CreatedAt = time.Now()
	}

	s.benchmarks[benchmark.ID] = benchmark
	return nil
}

func (s *MemoryStore) GetBenchmark(ctx context.Context, id string) (*Benchmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, exists := s.benchmarks[id]
	if !exists {
		return nil, ErrBenchmarkNotFound
	}

	bCopy := *b
	return &bCopy, nil
}

func (s *MemoryStore) ListBenchmarks(ctx context.Context, modelID string, limit, offset int) ([]*Benchmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Benchmark
	for _, b := range s.benchmarks {
		if modelID == "" || b.ModelID == modelID {
			result = append(result, b)
		}
	}

	if offset >= len(result) {
		return []*Benchmark{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

func (s *MemoryStore) UpdateBenchmark(ctx context.Context, benchmark *Benchmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.benchmarks[benchmark.ID]; !exists {
		return ErrBenchmarkNotFound
	}

	s.benchmarks[benchmark.ID] = benchmark
	return nil
}

func (s *MemoryStore) DeleteBenchmark(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.benchmarks[id]; !exists {
		return ErrBenchmarkNotFound
	}

	delete(s.benchmarks, id)
	return nil
}

// BenchmarkConfig operations (MemoryStore implementation)

func (s *MemoryStore) CreateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.Name == "" {
		return fmt.Errorf("config name cannot be empty")
	}

	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	s.benchmarkConfigs[config.Name] = config
	return nil
}

func (s *MemoryStore) GetBenchmarkConfig(ctx context.Context, name string) (*BenchmarkConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, exists := s.benchmarkConfigs[name]
	if !exists {
		return nil, ErrBenchmarkConfigNotFound
	}

	cCopy := *c
	return &cCopy, nil
}

func (s *MemoryStore) ListBenchmarkConfigs(ctx context.Context, limit, offset int) ([]*BenchmarkConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*BenchmarkConfig
	for _, c := range s.benchmarkConfigs {
		result = append(result, c)
	}

	if offset >= len(result) {
		return []*BenchmarkConfig{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

func (s *MemoryStore) UpdateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.benchmarkConfigs[config.Name]; !exists {
		return ErrBenchmarkConfigNotFound
	}

	s.benchmarkConfigs[config.Name] = config
	return nil
}

func (s *MemoryStore) DeleteBenchmarkConfig(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.benchmarkConfigs[name]; !exists {
		return ErrBenchmarkConfigNotFound
	}

	delete(s.benchmarkConfigs, name)
	return nil
}

// ModelLoadConfig operations

// SaveModelLoadConfig saves or updates a model load configuration
func (s *MemoryStore) SaveModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not provided
	if config.ID == "" {
		config.ID = generateID("mlcfg")
	}

	// Set timestamps
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// Use composite key: nodeID:modelID
	key := config.NodeID + ":" + config.ModelID
	s.modelLoadConfigs[key] = config

	return nil
}

// GetModelLoadConfig retrieves a model load configuration by node ID and model ID
func (s *MemoryStore) GetModelLoadConfig(ctx context.Context, nodeID, modelID string) (*ModelLoadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := nodeID + ":" + modelID
	config, exists := s.modelLoadConfigs[key]
	if !exists {
		return nil, ErrModelLoadConfigNotFound
	}

	return config, nil
}

// DeleteModelLoadConfig deletes a model load configuration by node ID and model ID
func (s *MemoryStore) DeleteModelLoadConfig(ctx context.Context, nodeID, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := nodeID + ":" + modelID
	if _, exists := s.modelLoadConfigs[key]; !exists {
		return ErrModelLoadConfigNotFound
	}

	delete(s.modelLoadConfigs, key)
	return nil
}

// ModelMetadata operations

// SaveModelMetadata saves or updates model metadata
func (s *MemoryStore) SaveModelMetadata(ctx context.Context, metadata *ModelMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = now
	}
	metadata.UpdatedAt = now

	s.modelMetadata[metadata.ModelID] = metadata
	return nil
}

// GetModelMetadata retrieves metadata for a single model
func (s *MemoryStore) GetModelMetadata(ctx context.Context, modelID string) (*ModelMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata, exists := s.modelMetadata[modelID]
	if !exists {
		return nil, ErrModelMetadataNotFound
	}

	// Return a copy
	metadataCopy := *metadata
	return &metadataCopy, nil
}

// ListModelMetadata lists model metadata with pagination
func (s *MemoryStore) ListModelMetadata(ctx context.Context, limit, offset int) ([]*ModelMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ModelMetadata
	for _, metadata := range s.modelMetadata {
		result = append(result, metadata)
	}

	if offset >= len(result) {
		return []*ModelMetadata{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

// DeleteModelMetadata deletes metadata for a model
func (s *MemoryStore) DeleteModelMetadata(ctx context.Context, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.modelMetadata[modelID]; !exists {
		return ErrModelMetadataNotFound
	}

	delete(s.modelMetadata, modelID)
	return nil
}

// GetAllModelMetadata retrieves all model metadata as a map
func (s *MemoryStore) GetAllModelMetadata(ctx context.Context) (map[string]*ModelMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*ModelMetadata, len(s.modelMetadata))
	for k, v := range s.modelMetadata {
		metadataCopy := *v
		result[k] = &metadataCopy
	}

	return result, nil
}

// Close closes the store (no-op for memory store)
func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conversations = make(map[string]*Conversation)
	s.messages = make(map[string][]*Message)
	s.messagesByID = make(map[string]*Message)
	s.benchmarks = make(map[string]*Benchmark)
	s.benchmarkConfigs = make(map[string]*BenchmarkConfig)
	s.modelLoadConfigs = make(map[string]*ModelLoadConfig)

	return nil
}

// Stats returns statistics about the store
func (s *MemoryStore) Stats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalMessages := 0
	for _, msgs := range s.messages {
		totalMessages += len(msgs)
	}

	return map[string]interface{}{
		"conversations": len(s.conversations),
		"messages":      totalMessages,
		"type":          "memory",
	}
}

// generateID generates a unique ID with a prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
