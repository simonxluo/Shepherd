package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/storage"
)

// runStoreTestSuite runs the full Store interface compliance tests against any backend.
func runStoreTestSuite(t *testing.T, store storage.Store) {
	t.Helper()
	ctx := context.Background()

	t.Run("Conversations", func(t *testing.T) {
		testConversationCRUD(t, ctx, store)
	})
	t.Run("Messages", func(t *testing.T) {
		testMessageCRUD(t, ctx, store)
	})
	t.Run("Benchmarks", func(t *testing.T) {
		testBenchmarkCRUD(t, ctx, store)
	})
	t.Run("BenchmarkConfigs", func(t *testing.T) {
		testBenchmarkConfigCRUD(t, ctx, store)
	})
	t.Run("ModelLoadConfigs", func(t *testing.T) {
		testModelLoadConfigCRUD(t, ctx, store)
	})
	t.Run("LaunchProfiles", func(t *testing.T) {
		testLaunchProfileCRUD(t, ctx, store)
	})
	t.Run("ModelMetadata", func(t *testing.T) {
		testModelMetadataCRUD(t, ctx, store)
	})
	t.Run("TTSHistory", func(t *testing.T) {
		testTTSHistoryCRUD(t, ctx, store)
	})
	t.Run("DownloadTasks", func(t *testing.T) {
		testDownloadTaskCRUD(t, ctx, store)
	})
}

func testConversationCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	conv := &storage.Conversation{
		Model:        "test-model",
		Title:        "Test Conversation",
		SystemPrompt: "You are a test assistant",
		Metadata:     map[string]interface{}{"key": "value"},
	}

	err := store.CreateConversation(ctx, conv)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if conv.ID == "" {
		t.Fatal("CreateConversation: ID not generated")
	}

	got, err := store.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if got.Model != "test-model" {
		t.Errorf("GetConversation: model = %q, want %q", got.Model, "test-model")
	}
	if got.Title != "Test Conversation" {
		t.Errorf("GetConversation: title = %q, want %q", got.Title, "Test Conversation")
	}

	list, err := store.ListConversations(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("ListConversations: empty result")
	}

	conv.Title = "Updated Title"
	err = store.UpdateConversation(ctx, conv)
	if err != nil {
		t.Fatalf("UpdateConversation: %v", err)
	}
	got, _ = store.GetConversation(ctx, conv.ID)
	if got.Title != "Updated Title" {
		t.Errorf("UpdateConversation: title = %q, want %q", got.Title, "Updated Title")
	}

	err = store.DeleteConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}
	_, err = store.GetConversation(ctx, conv.ID)
	if err == nil {
		t.Error("GetConversation after delete: expected error, got nil")
	}
}

func testMessageCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	conv := &storage.Conversation{Model: "test-model", Title: "Msg Test"}
	_ = store.CreateConversation(ctx, conv)

	msg := &storage.Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "Hello",
		Metadata:       map[string]interface{}{"test": true},
	}

	err := store.CreateMessage(ctx, msg)
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("CreateMessage: ID not generated")
	}

	got, _ := store.GetConversation(ctx, conv.ID)
	if got.MessageCount != 1 {
		t.Errorf("MessageCount after CreateMessage: got %d, want 1", got.MessageCount)
	}

	msgs, err := store.GetMessages(ctx, conv.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("GetMessages: len = %d, want 1", len(msgs))
	}
	if msgs[0].Content != "Hello" {
		t.Errorf("GetMessages: content = %q, want %q", msgs[0].Content, "Hello")
	}

	err = store.DeleteMessages(ctx, conv.ID)
	if err != nil {
		t.Fatalf("DeleteMessages: %v", err)
	}
	msgs, _ = store.GetMessages(ctx, conv.ID, 10, 0)
	if len(msgs) != 0 {
		t.Errorf("GetMessages after delete: len = %d, want 0", len(msgs))
	}

	_ = store.DeleteConversation(ctx, conv.ID)
}

func testBenchmarkCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	now := time.Now().UTC()
	b := &storage.Benchmark{
		ID:        fmt.Sprintf("bench-%d", time.Now().UnixNano()),
		ModelID:   "model-1",
		ModelName: "Test Model",
		Status:    "running",
		CreatedAt: now,
		Config:    map[string]interface{}{"threads": 4},
	}

	err := store.CreateBenchmark(ctx, b)
	if err != nil {
		t.Fatalf("CreateBenchmark: %v", err)
	}

	got, err := store.GetBenchmark(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetBenchmark: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("GetBenchmark: status = %q, want %q", got.Status, "running")
	}

	b.Status = "completed"
	finished := time.Now().UTC()
	b.FinishedAt = &finished
	err = store.UpdateBenchmark(ctx, b)
	if err != nil {
		t.Fatalf("UpdateBenchmark: %v", err)
	}

	list, err := store.ListBenchmarks(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("ListBenchmarks: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("ListBenchmarks: empty")
	}

	err = store.DeleteBenchmark(ctx, b.ID)
	if err != nil {
		t.Fatalf("DeleteBenchmark: %v", err)
	}
}

func testBenchmarkConfigCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	cfg := &storage.BenchmarkConfig{
		Name:         fmt.Sprintf("test-config-%d", time.Now().UnixNano()),
		ModelID:      "model-1",
		ModelName:    "Test Model",
		LlamaCppPath: "/usr/local/bin/llama-server",
		Devices:      []string{"cuda:0"},
		Params:       map[string]string{"threads": "4"},
		CreatedAt:    time.Now().UTC(),
	}

	err := store.CreateBenchmarkConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("CreateBenchmarkConfig: %v", err)
	}

	got, err := store.GetBenchmarkConfig(ctx, cfg.Name)
	if err != nil {
		t.Fatalf("GetBenchmarkConfig: %v", err)
	}
	if got.LlamaCppPath != "/usr/local/bin/llama-server" {
		t.Errorf("GetBenchmarkConfig: path = %q", got.LlamaCppPath)
	}

	cfg.LlamaCppPath = "/opt/llama/bin/server"
	err = store.UpdateBenchmarkConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("UpdateBenchmarkConfig: %v", err)
	}

	err = store.DeleteBenchmarkConfig(ctx, cfg.Name)
	if err != nil {
		t.Fatalf("DeleteBenchmarkConfig: %v", err)
	}
}

func testModelLoadConfigCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	cfg := &storage.ModelLoadConfig{
		NodeID:    "node-1",
		ModelID:   "model-1",
		ModelName: "Test Model",
		Config:    map[string]interface{}{"gpu_layers": 32},
	}

	err := store.SaveModelLoadConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("SaveModelLoadConfig: %v", err)
	}

	got, err := store.GetModelLoadConfig(ctx, "node-1", "model-1")
	if err != nil {
		t.Fatalf("GetModelLoadConfig: %v", err)
	}
	if got.ModelName != "Test Model" {
		t.Errorf("GetModelLoadConfig: name = %q", got.ModelName)
	}

	named := &storage.ModelLoadConfig{
		NodeID:    "node-1",
		ModelID:   "model-1",
		ModelName: "Test Model",
		Name:      "fast",
		Config:    map[string]interface{}{"gpu_layers": 64},
	}
	err = store.SaveNamedModelLoadConfig(ctx, named)
	if err != nil {
		t.Fatalf("SaveNamedModelLoadConfig: %v", err)
	}

	list, err := store.ListModelLoadConfigs(ctx, "node-1", "model-1")
	if err != nil {
		t.Fatalf("ListModelLoadConfigs: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("ListModelLoadConfigs: len = %d, want >= 2", len(list))
	}

	err = store.DeleteNamedModelLoadConfig(ctx, "node-1", "model-1", "fast")
	if err != nil {
		t.Fatalf("DeleteNamedModelLoadConfig: %v", err)
	}

	err = store.DeleteModelLoadConfig(ctx, "node-1", "model-1")
	if err != nil {
		t.Fatalf("DeleteModelLoadConfig: %v", err)
	}
}

func testLaunchProfileCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	profile := &storage.LaunchProfile{
		Name:     "test-profile",
		PluginID: "llamacpp",
		Params:   map[string]interface{}{"threads": 8},
		Env:      []string{"CUDA_VISIBLE_DEVICES=0"},
	}

	err := store.CreateLaunchProfile(ctx, profile)
	if err != nil {
		t.Fatalf("CreateLaunchProfile: %v", err)
	}
	if profile.ID == "" {
		t.Fatal("CreateLaunchProfile: ID not generated")
	}

	got, err := store.GetLaunchProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("GetLaunchProfile: %v", err)
	}
	if got.Name != "test-profile" {
		t.Errorf("GetLaunchProfile: name = %q", got.Name)
	}

	list, err := store.ListLaunchProfiles(ctx, "llamacpp", "")
	if err != nil {
		t.Fatalf("ListLaunchProfiles: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("ListLaunchProfiles: empty")
	}

	profile.Name = "updated-profile"
	err = store.UpdateLaunchProfile(ctx, profile)
	if err != nil {
		t.Fatalf("UpdateLaunchProfile: %v", err)
	}

	err = store.DeleteLaunchProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("DeleteLaunchProfile: %v", err)
	}
}

func testModelMetadataCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	now := time.Now().UTC()
	metadata := &storage.ModelMetadata{
		ModelID:     fmt.Sprintf("model-meta-%d", time.Now().UnixNano()),
		NodeID:      "node-1",
		StoragePath: "/models/test.gguf",
		Alias:       "test-alias",
		Favourite:   true,
		Tags:        []string{"chat", "code"},
		Description: "A test model",
		LoadCount:   5,
		LastLoaded:  &now,
		TotalTokens: 1000,
		Capabilities: &storage.Capabilities{
			Thinking: true,
			Tools:    true,
		},
	}

	err := store.SaveModelMetadata(ctx, metadata)
	if err != nil {
		t.Fatalf("SaveModelMetadata (insert): %v", err)
	}

	got, err := store.GetModelMetadata(ctx, metadata.ModelID)
	if err != nil {
		t.Fatalf("GetModelMetadata: %v", err)
	}
	if got.Alias != "test-alias" {
		t.Errorf("GetModelMetadata: alias = %q", got.Alias)
	}
	if !got.Favourite {
		t.Error("GetModelMetadata: favourite = false, want true")
	}

	metadata.Alias = "updated-alias"
	metadata.LoadCount = 10
	err = store.SaveModelMetadata(ctx, metadata)
	if err != nil {
		t.Fatalf("SaveModelMetadata (update): %v", err)
	}

	got, _ = store.GetModelMetadata(ctx, metadata.ModelID)
	if got.Alias != "updated-alias" {
		t.Errorf("After update: alias = %q, want %q", got.Alias, "updated-alias")
	}

	all, err := store.GetAllModelMetadata(ctx)
	if err != nil {
		t.Fatalf("GetAllModelMetadata: %v", err)
	}
	if _, ok := all[metadata.ModelID]; !ok {
		t.Error("GetAllModelMetadata: model not found")
	}

	list, err := store.ListModelMetadata(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListModelMetadata: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("ListModelMetadata: empty")
	}

	err = store.DeleteModelMetadata(ctx, metadata.ModelID)
	if err != nil {
		t.Fatalf("DeleteModelMetadata: %v", err)
	}
}

func testTTSHistoryCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	item := &storage.TTSHistoryItem{
		Model:     "tts-model",
		InputText: "Hello world",
		AudioPath: "/audio/test.wav",
		Format:    "wav",
		Duration:  2.5,
		Favourite: false,
		Params:    map[string]interface{}{"speed": 1.0},
	}

	err := store.CreateTTSHistory(ctx, item)
	if err != nil {
		t.Fatalf("CreateTTSHistory: %v", err)
	}
	if item.ID == "" {
		t.Fatal("CreateTTSHistory: ID not generated")
	}

	got, err := store.GetTTSHistory(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetTTSHistory: %v", err)
	}
	if got.InputText != "Hello world" {
		t.Errorf("GetTTSHistory: text = %q", got.InputText)
	}

	err = store.UpdateTTSHistoryFavourite(ctx, item.ID, true)
	if err != nil {
		t.Fatalf("UpdateTTSHistoryFavourite: %v", err)
	}

	trueVal := true
	list, err := store.ListTTSHistory(ctx, 10, 0, &trueVal)
	if err != nil {
		t.Fatalf("ListTTSHistory favourite: %v", err)
	}
	if len(list) == 0 {
		t.Error("ListTTSHistory favourite: empty")
	}

	err = store.DeleteTTSHistory(ctx, item.ID)
	if err != nil {
		t.Fatalf("DeleteTTSHistory: %v", err)
	}
}

func testDownloadTaskCRUD(t *testing.T, ctx context.Context, store storage.Store) {
	task := &storage.DownloadTask{
		URL:       "https://example.com/model.gguf",
		Path:      "/models/",
		FileName:  "model.gguf",
		State:     "idle",
		CreatedAt: time.Now().UTC(),
	}

	err := store.CreateDownloadTask(ctx, task)
	if err != nil {
		t.Fatalf("CreateDownloadTask: %v", err)
	}
	if task.ID == "" {
		t.Fatal("CreateDownloadTask: ID not generated")
	}

	got, err := store.GetDownloadTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetDownloadTask: %v", err)
	}
	if got.State != "idle" {
		t.Errorf("GetDownloadTask: state = %q", got.State)
	}

	task.State = "downloading"
	task.DownloadedBytes = 1024
	task.TotalBytes = 4096
	err = store.UpdateDownloadTask(ctx, task)
	if err != nil {
		t.Fatalf("UpdateDownloadTask: %v", err)
	}

	active, err := store.ListActiveDownloadTasks(ctx)
	if err != nil {
		t.Fatalf("ListActiveDownloadTasks: %v", err)
	}
	if len(active) == 0 {
		t.Error("ListActiveDownloadTasks: empty (should have downloading task)")
	}

	list, err := store.ListDownloadTasks(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListDownloadTasks: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("ListDownloadTasks: empty")
	}

	err = store.DeleteDownloadTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("DeleteDownloadTask: %v", err)
	}
}

// TestMemoryStore runs the full test suite against the memory backend.
func TestMemoryStore(t *testing.T) {
	store, err := storage.NewMemoryStore()
	if err != nil {
		t.Fatalf("NewMemoryStore: %v", err)
	}
	defer store.Close()
	runStoreTestSuite(t, store)
}

// TestSQLiteStore runs the full test suite against the SQLite backend.
func TestSQLiteStore(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewSQLiteStore(&storage.SQLiteConfig{
		Path: dir + "/test.db",
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()
	runStoreTestSuite(t, store)
}
