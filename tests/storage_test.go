package tests

import (
	"context"
	"testing"

	"github.com/simonxluo/Shepherd/internal/comm/storage"
)

func TestStorageConversationCRUD(t *testing.T) {
	store, err := storage.NewMemoryStore()
	if err != nil {
		t.Fatalf("failed to create memory store: %v", err)
	}
	ctx := context.Background()

	// Create
	conv := &storage.Conversation{
		Model: "test-model",
		Title: "Test Conversation",
	}
	if err := store.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}
	if conv.ID == "" {
		t.Fatal("expected ID to be generated")
	}

	// Get
	got, err := store.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}
	if got.Title != "Test Conversation" {
		t.Errorf("expected title 'Test Conversation', got '%s'", got.Title)
	}
	if got.Model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", got.Model)
	}

	// List
	convs, err := store.ListConversations(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}

	// Update
	got.Title = "Updated Title"
	if err := store.UpdateConversation(ctx, got); err != nil {
		t.Fatalf("UpdateConversation failed: %v", err)
	}
	updated, _ := store.GetConversation(ctx, conv.ID)
	if updated.Title != "Updated Title" {
		t.Errorf("expected updated title, got '%s'", updated.Title)
	}

	// Delete
	if err := store.DeleteConversation(ctx, conv.ID); err != nil {
		t.Fatalf("DeleteConversation failed: %v", err)
	}
	_, err = store.GetConversation(ctx, conv.ID)
	if err != storage.ErrConversationNotFound {
		t.Errorf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestStorageConversationNotFound(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	_, err := store.GetConversation(ctx, "nonexistent")
	if err != storage.ErrConversationNotFound {
		t.Errorf("expected ErrConversationNotFound, got %v", err)
	}
}

func TestStorageMessageCRUD(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	// Create conversation first
	conv := &storage.Conversation{Model: "test-model", Title: "Test"}
	store.CreateConversation(ctx, conv)

	// Create message
	msg := &storage.Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "Hello, world!",
	}
	if err := store.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("expected message ID to be generated")
	}

	// Get messages
	msgs, err := store.GetMessages(ctx, conv.ID, 100, 0)
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got '%s'", msgs[0].Content)
	}

	// Check message count updated
	updatedConv, _ := store.GetConversation(ctx, conv.ID)
	if updatedConv.MessageCount != 1 {
		t.Errorf("expected message count 1, got %d", updatedConv.MessageCount)
	}

	// Delete messages
	if err := store.DeleteMessages(ctx, conv.ID); err != nil {
		t.Fatalf("DeleteMessages failed: %v", err)
	}
	msgs, _ = store.GetMessages(ctx, conv.ID, 100, 0)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs))
	}
}

func TestStorageBenchmarkCRUD(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	// Create
	bench := &storage.Benchmark{
		ModelID:   "model-1",
		ModelName: "Test Model",
		Status:    "running",
	}
	if err := store.CreateBenchmark(ctx, bench); err != nil {
		t.Fatalf("CreateBenchmark failed: %v", err)
	}
	if bench.ID == "" {
		t.Fatal("expected benchmark ID")
	}

	// Get
	got, err := store.GetBenchmark(ctx, bench.ID)
	if err != nil {
		t.Fatalf("GetBenchmark failed: %v", err)
	}
	if got.ModelID != "model-1" {
		t.Errorf("expected model-1, got %s", got.ModelID)
	}

	// List
	benchmarks, err := store.ListBenchmarks(ctx, "", 100, 0)
	if err != nil {
		t.Fatalf("ListBenchmarks failed: %v", err)
	}
	if len(benchmarks) != 1 {
		t.Fatalf("expected 1 benchmark, got %d", len(benchmarks))
	}

	// List with filter
	filtered, _ := store.ListBenchmarks(ctx, "model-1", 100, 0)
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered benchmark, got %d", len(filtered))
	}
	empty, _ := store.ListBenchmarks(ctx, "nonexistent", 100, 0)
	if len(empty) != 0 {
		t.Errorf("expected 0 filtered benchmarks, got %d", len(empty))
	}

	// Update
	got.Status = "completed"
	if err := store.UpdateBenchmark(ctx, got); err != nil {
		t.Fatalf("UpdateBenchmark failed: %v", err)
	}
	updated, _ := store.GetBenchmark(ctx, bench.ID)
	if updated.Status != "completed" {
		t.Errorf("expected status completed, got %s", updated.Status)
	}

	// Delete
	if err := store.DeleteBenchmark(ctx, bench.ID); err != nil {
		t.Fatalf("DeleteBenchmark failed: %v", err)
	}
	_, err = store.GetBenchmark(ctx, bench.ID)
	if err != storage.ErrBenchmarkNotFound {
		t.Errorf("expected ErrBenchmarkNotFound, got %v", err)
	}
}

func TestStorageBenchmarkConfigCRUD(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	// Create
	cfg := &storage.BenchmarkConfig{
		Name:         "test-config",
		ModelID:      "model-1",
		ModelName:    "Test Model",
		LlamaCppPath: "/usr/local/bin/llama-server",
	}
	if err := store.CreateBenchmarkConfig(ctx, cfg); err != nil {
		t.Fatalf("CreateBenchmarkConfig failed: %v", err)
	}

	// Get
	got, err := store.GetBenchmarkConfig(ctx, "test-config")
	if err != nil {
		t.Fatalf("GetBenchmarkConfig failed: %v", err)
	}
	if got.ModelID != "model-1" {
		t.Errorf("expected model-1, got %s", got.ModelID)
	}

	// List
	configs, err := store.ListBenchmarkConfigs(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListBenchmarkConfigs failed: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}

	// Update
	got.LlamaCppPath = "/new/path"
	if err := store.UpdateBenchmarkConfig(ctx, got); err != nil {
		t.Fatalf("UpdateBenchmarkConfig failed: %v", err)
	}
	updated, _ := store.GetBenchmarkConfig(ctx, "test-config")
	if updated.LlamaCppPath != "/new/path" {
		t.Errorf("expected updated path, got '%s'", updated.LlamaCppPath)
	}

	// Delete
	if err := store.DeleteBenchmarkConfig(ctx, "test-config"); err != nil {
		t.Fatalf("DeleteBenchmarkConfig failed: %v", err)
	}
	_, err = store.GetBenchmarkConfig(ctx, "test-config")
	if err != storage.ErrBenchmarkConfigNotFound {
		t.Errorf("expected ErrBenchmarkConfigNotFound, got %v", err)
	}
}

func TestStorageModelLoadConfigCRUD(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	// Save
	cfg := &storage.ModelLoadConfig{
		NodeID:    "node-1",
		ModelID:   "model-1",
		ModelName: "Test Model",
		Name:      "",
		Config:    map[string]interface{}{"gpu_layers": 99},
	}
	if err := store.SaveModelLoadConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveModelLoadConfig failed: %v", err)
	}

	// Get
	got, err := store.GetModelLoadConfig(ctx, "node-1", "model-1")
	if err != nil {
		t.Fatalf("GetModelLoadConfig failed: %v", err)
	}
	if got.ModelName != "Test Model" {
		t.Errorf("expected 'Test Model', got '%s'", got.ModelName)
	}

	// List
	configs, err := store.ListModelLoadConfigs(ctx, "node-1", "model-1")
	if err != nil {
		t.Fatalf("ListModelLoadConfigs failed: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}

	// Delete
	if err := store.DeleteModelLoadConfig(ctx, "node-1", "model-1"); err != nil {
		t.Fatalf("DeleteModelLoadConfig failed: %v", err)
	}
	_, err = store.GetModelLoadConfig(ctx, "node-1", "model-1")
	if err != storage.ErrModelLoadConfigNotFound {
		t.Errorf("expected ErrModelLoadConfigNotFound, got %v", err)
	}
}

func TestStorageNamedModelLoadConfig(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	cfg := &storage.ModelLoadConfig{
		NodeID:    "node-1",
		ModelID:   "model-1",
		ModelName: "Test Model",
		Name:      "fast-preset",
		Config:    map[string]interface{}{"gpu_layers": 50},
	}
	if err := store.SaveNamedModelLoadConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveNamedModelLoadConfig failed: %v", err)
	}

	configs, _ := store.ListModelLoadConfigs(ctx, "node-1", "model-1")
	if len(configs) != 1 {
		t.Fatalf("expected 1 named config, got %d", len(configs))
	}
	if configs[0].Name != "fast-preset" {
		t.Errorf("expected name 'fast-preset', got '%s'", configs[0].Name)
	}

	// Delete named
	if err := store.DeleteNamedModelLoadConfig(ctx, "node-1", "model-1", "fast-preset"); err != nil {
		t.Fatalf("DeleteNamedModelLoadConfig failed: %v", err)
	}
	configs, _ = store.ListModelLoadConfigs(ctx, "node-1", "model-1")
	if len(configs) != 0 {
		t.Errorf("expected 0 configs after delete, got %d", len(configs))
	}
}

func TestStorageModelMetadataCRUD(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	// Save
	metadata := &storage.ModelMetadata{
		ModelID:     "model-1",
		Alias:       "my-model",
		Favourite:   true,
		Tags:        []string{"chat", "english"},
		Description: "A test model",
	}
	if err := store.SaveModelMetadata(ctx, metadata); err != nil {
		t.Fatalf("SaveModelMetadata failed: %v", err)
	}

	// Get
	got, err := store.GetModelMetadata(ctx, "model-1")
	if err != nil {
		t.Fatalf("GetModelMetadata failed: %v", err)
	}
	if got.Alias != "my-model" {
		t.Errorf("expected alias 'my-model', got '%s'", got.Alias)
	}
	if !got.Favourite {
		t.Error("expected favourite=true")
	}

	// List
	metadatas, err := store.ListModelMetadata(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListModelMetadata failed: %v", err)
	}
	if len(metadatas) != 1 {
		t.Fatalf("expected 1 metadata, got %d", len(metadatas))
	}

	// GetAll
	allMeta, err := store.GetAllModelMetadata(ctx)
	if err != nil {
		t.Fatalf("GetAllModelMetadata failed: %v", err)
	}
	if len(allMeta) != 1 {
		t.Fatalf("expected 1 in GetAll, got %d", len(allMeta))
	}

	// Delete
	if err := store.DeleteModelMetadata(ctx, "model-1"); err != nil {
		t.Fatalf("DeleteModelMetadata failed: %v", err)
	}
	_, err = store.GetModelMetadata(ctx, "model-1")
	if err != storage.ErrModelMetadataNotFound {
		t.Errorf("expected ErrModelMetadataNotFound, got %v", err)
	}
}

func TestStorageTTSHistoryCRUD(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	// Create
	item := &storage.TTSHistoryItem{
		Model:     "tts-model",
		InputText: "Hello world",
		AudioPath: "/tmp/audio.wav",
		Format:    "wav",
		Duration:  2.5,
	}
	if err := store.CreateTTSHistory(ctx, item); err != nil {
		t.Fatalf("CreateTTSHistory failed: %v", err)
	}
	if item.ID == "" {
		t.Fatal("expected ID to be generated")
	}

	// Get
	got, err := store.GetTTSHistory(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetTTSHistory failed: %v", err)
	}
	if got.InputText != "Hello world" {
		t.Errorf("expected 'Hello world', got '%s'", got.InputText)
	}

	// List
	items, err := store.ListTTSHistory(ctx, 100, 0, nil)
	if err != nil {
		t.Fatalf("ListTTSHistory failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	// Update favourite
	if err := store.UpdateTTSHistoryFavourite(ctx, item.ID, true); err != nil {
		t.Fatalf("UpdateTTSHistoryFavourite failed: %v", err)
	}
	got, _ = store.GetTTSHistory(ctx, item.ID)
	if !got.Favourite {
		t.Error("expected favourite=true after update")
	}

	// Delete
	if err := store.DeleteTTSHistory(ctx, item.ID); err != nil {
		t.Fatalf("DeleteTTSHistory failed: %v", err)
	}
	_, err = store.GetTTSHistory(ctx, item.ID)
	if err != storage.ErrTTSHistoryNotFound {
		t.Errorf("expected ErrTTSHistoryNotFound, got %v", err)
	}
}

func TestStorageEmptyList(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	convs, err := store.ListConversations(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListConversations on empty store failed: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("expected empty list, got %d items", len(convs))
	}

	benchmarks, err := store.ListBenchmarks(ctx, "", 100, 0)
	if err != nil {
		t.Fatalf("ListBenchmarks on empty store failed: %v", err)
	}
	if len(benchmarks) != 0 {
		t.Errorf("expected empty list, got %d items", len(benchmarks))
	}
}

func TestStoragePagination(t *testing.T) {
	store, _ := storage.NewMemoryStore()
	ctx := context.Background()

	// Create 5 conversations
	for i := 0; i < 5; i++ {
		conv := &storage.Conversation{Model: "model", Title: "Conv"}
		store.CreateConversation(ctx, conv)
	}

	// Get first 2
	page1, _ := store.ListConversations(ctx, 2, 0)
	if len(page1) != 2 {
		t.Errorf("expected 2 conversations in page 1, got %d", len(page1))
	}

	// Get next 2
	page2, _ := store.ListConversations(ctx, 2, 2)
	if len(page2) != 2 {
		t.Errorf("expected 2 conversations in page 2, got %d", len(page2))
	}

	// Get last 1
	page3, _ := store.ListConversations(ctx, 2, 4)
	if len(page3) != 1 {
		t.Errorf("expected 1 conversation in page 3, got %d", len(page3))
	}

	// Out of range offset
	page4, _ := store.ListConversations(ctx, 2, 10)
	if len(page4) != 0 {
		t.Errorf("expected 0 conversations for out-of-range offset, got %d", len(page4))
	}
}
