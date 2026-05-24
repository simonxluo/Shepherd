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
	modelLoadConfigs map[string]*ModelLoadConfig // key: "nodeID:modelID:name"
	modelMetadata    map[string]*ModelMetadata   // key: modelID
	ttsHistory       map[string]*TTSHistoryItem  // key: id
	downloadTasks    map[string]*DownloadTask    // key: id
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
		ttsHistory:       make(map[string]*TTSHistoryItem),
		downloadTasks:    make(map[string]*DownloadTask),
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

	// Use composite key: nodeID:modelID:name
	name := config.Name
	key := config.NodeID + ":" + config.ModelID + ":" + name
	s.modelLoadConfigs[key] = config

	return nil
}

// GetModelLoadConfig retrieves a model load configuration by node ID and model ID
func (s *MemoryStore) GetModelLoadConfig(ctx context.Context, nodeID, modelID string) (*ModelLoadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := nodeID + ":" + modelID + ":"
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

	key := nodeID + ":" + modelID + ":"
	if _, exists := s.modelLoadConfigs[key]; !exists {
		return ErrModelLoadConfigNotFound
	}

	delete(s.modelLoadConfigs, key)
	return nil
}

// ListModelLoadConfigs returns all load configs for a model on a node
func (s *MemoryStore) ListModelLoadConfigs(ctx context.Context, nodeID, modelID string) ([]*ModelLoadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := nodeID + ":" + modelID + ":"
	var result []*ModelLoadConfig

	for key, cfg := range s.modelLoadConfigs {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			cfgCopy := *cfg
			result = append(result, &cfgCopy)
		} else if key == prefix {
			cfgCopy := *cfg
			result = append(result, &cfgCopy)
		}
	}

	return result, nil
}

// SaveNamedModelLoadConfig saves a named load config preset
func (s *MemoryStore) SaveNamedModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
	if config.Name == "" {
		return fmt.Errorf("named config requires a non-empty name")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if config.ID == "" {
		config.ID = generateID("mlcfg")
	}

	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	key := config.NodeID + ":" + config.ModelID + ":" + config.Name
	s.modelLoadConfigs[key] = config

	return nil
}

// DeleteNamedModelLoadConfig deletes a named load config preset
func (s *MemoryStore) DeleteNamedModelLoadConfig(ctx context.Context, nodeID, modelID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := nodeID + ":" + modelID + ":" + name
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

//  TTS History Operations 

// CreateTTSHistory creates a new TTS history record
func (s *MemoryStore) CreateTTSHistory(ctx context.Context, item *TTSHistoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = generateID("tts")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	s.ttsHistory[item.ID] = item
	return nil
}

// GetTTSHistory retrieves a TTS history item by ID
func (s *MemoryStore) GetTTSHistory(ctx context.Context, id string) (*TTSHistoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.ttsHistory[id]
	if !exists {
		return nil, ErrTTSHistoryNotFound
	}

	itemCopy := *item
	return &itemCopy, nil
}

// ListTTSHistory lists TTS history items with pagination
func (s *MemoryStore) ListTTSHistory(ctx context.Context, limit, offset int, favouriteOnly *bool) ([]*TTSHistoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*TTSHistoryItem
	for _, item := range s.ttsHistory {
		if favouriteOnly != nil && *favouriteOnly && !item.Favourite {
			continue
		}
		result = append(result, item)
	}

	// Sort by created_at desc (simple bubble for in-memory)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if offset >= len(result) {
		return []*TTSHistoryItem{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

// UpdateTTSHistoryFavourite updates the favourite flag of a TTS history item
func (s *MemoryStore) UpdateTTSHistoryFavourite(ctx context.Context, id string, favourite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.ttsHistory[id]
	if !exists {
		return ErrTTSHistoryNotFound
	}

	item.Favourite = favourite
	return nil
}

// DeleteTTSHistory deletes a TTS history item by ID
func (s *MemoryStore) DeleteTTSHistory(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.ttsHistory[id]; !exists {
		return ErrTTSHistoryNotFound
	}

	delete(s.ttsHistory, id)
	return nil
}

// Download Task Operations

// CreateDownloadTask creates a new download task record
func (s *MemoryStore) CreateDownloadTask(ctx context.Context, task *DownloadTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		task.ID = generateID("dl")
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	taskCopy := *task
	s.downloadTasks[task.ID] = &taskCopy
	return nil
}

// GetDownloadTask retrieves a download task by ID
func (s *MemoryStore) GetDownloadTask(ctx context.Context, id string) (*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.downloadTasks[id]
	if !exists {
		return nil, ErrDownloadTaskNotFound
	}

	taskCopy := *task
	return &taskCopy, nil
}

// ListDownloadTasks lists download tasks with pagination, ordered by created_at DESC
func (s *MemoryStore) ListDownloadTasks(ctx context.Context, limit, offset int) ([]*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*DownloadTask
	for _, task := range s.downloadTasks {
		taskCopy := *task
		result = append(result, &taskCopy)
	}

	// Sort by created_at desc
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if offset >= len(result) {
		return []*DownloadTask{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

// UpdateDownloadTask updates all mutable fields of a download task
func (s *MemoryStore) UpdateDownloadTask(ctx context.Context, task *DownloadTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.downloadTasks[task.ID]; !exists {
		return ErrDownloadTaskNotFound
	}

	taskCopy := *task
	s.downloadTasks[task.ID] = &taskCopy
	return nil
}

// DeleteDownloadTask deletes a download task by ID
func (s *MemoryStore) DeleteDownloadTask(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.downloadTasks[id]; !exists {
		return ErrDownloadTaskNotFound
	}

	delete(s.downloadTasks, id)
	return nil
}

// ListActiveDownloadTasks returns all download tasks with active states
func (s *MemoryStore) ListActiveDownloadTasks(ctx context.Context) ([]*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeStates := map[string]bool{
		"idle":        true,
		"preparing":   true,
		"downloading": true,
		"merging":     true,
		"verifying":   true,
		"paused":      true,
	}

	var result []*DownloadTask
	for _, task := range s.downloadTasks {
		if activeStates[task.State] {
			taskCopy := *task
			result = append(result, &taskCopy)
		}
	}

	// Sort by created_at desc
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
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

