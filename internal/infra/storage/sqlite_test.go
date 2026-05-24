package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(&SQLiteConfig{Path: path})
	if err != nil {
		t.Fatalf("创建 SQLite 存储失败: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSQLiteConversationCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	conv := &Conversation{Model: "test-model", Title: "测试会话"}
	if err := s.CreateConversation(ctx, conv); err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if conv.ID == "" {
		t.Fatal("期望自动生成 ID")
	}

	got, err := s.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("GetConversation: %v", err)
	}
	if got.Title != "测试会话" {
		t.Errorf("Title = %q, want %q", got.Title, "测试会话")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt 不应为零值")
	}

	convs, err := s.ListConversations(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convs) != 1 {
		t.Fatalf("len = %d, want 1", len(convs))
	}

	got.Title = "更新标题"
	if err := s.UpdateConversation(ctx, got); err != nil {
		t.Fatalf("UpdateConversation: %v", err)
	}
	updated, _ := s.GetConversation(ctx, conv.ID)
	if updated.Title != "更新标题" {
		t.Errorf("Title = %q, want %q", updated.Title, "更新标题")
	}

	if err := s.DeleteConversation(ctx, conv.ID); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}
	_, err = s.GetConversation(ctx, conv.ID)
	if err != ErrConversationNotFound {
		t.Errorf("err = %v, want ErrConversationNotFound", err)
	}
}

func TestSQLiteMessageCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	conv := &Conversation{Model: "test-model", Title: "测试"}
	s.CreateConversation(ctx, conv)

	msg := &Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        "你好世界",
	}
	if err := s.CreateMessage(ctx, msg); err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	msgs, err := s.GetMessages(ctx, conv.ID, 100, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Content != "你好世界" {
		t.Fatalf("消息内容不匹配: %v", msgs)
	}

	updatedConv, _ := s.GetConversation(ctx, conv.ID)
	if updatedConv.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1", updatedConv.MessageCount)
	}

	if err := s.DeleteMessages(ctx, conv.ID); err != nil {
		t.Fatalf("DeleteMessages: %v", err)
	}
	msgs, _ = s.GetMessages(ctx, conv.ID, 100, 0)
	if len(msgs) != 0 {
		t.Errorf("删除后消息数 = %d, want 0", len(msgs))
	}
}

func TestSQLiteBenchmarkCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	bench := &Benchmark{
		ID:        "bench-test-1",
		ModelID:   "model-1",
		ModelName: "测试模型",
		Status:    "running",
		CreatedAt: now,
	}
	if err := s.CreateBenchmark(ctx, bench); err != nil {
		t.Fatalf("CreateBenchmark: %v", err)
	}

	got, err := s.GetBenchmark(ctx, bench.ID)
	if err != nil {
		t.Fatalf("GetBenchmark: %v", err)
	}
	if got.ModelID != "model-1" {
		t.Errorf("ModelID = %q, want %q", got.ModelID, "model-1")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt 不应为零值")
	}

	benchmarks, err := s.ListBenchmarks(ctx, "", 100, 0)
	if err != nil {
		t.Fatalf("ListBenchmarks: %v", err)
	}
	if len(benchmarks) != 1 {
		t.Fatalf("len = %d, want 1", len(benchmarks))
	}

	got.Status = "completed"
	finished := time.Now().UTC()
	got.FinishedAt = &finished
	if err := s.UpdateBenchmark(ctx, got); err != nil {
		t.Fatalf("UpdateBenchmark: %v", err)
	}
	updated, _ := s.GetBenchmark(ctx, bench.ID)
	if updated.Status != "completed" {
		t.Errorf("Status = %q, want completed", updated.Status)
	}

	if err := s.DeleteBenchmark(ctx, bench.ID); err != nil {
		t.Fatalf("DeleteBenchmark: %v", err)
	}
	_, err = s.GetBenchmark(ctx, bench.ID)
	if err != ErrBenchmarkNotFound {
		t.Errorf("err = %v, want ErrBenchmarkNotFound", err)
	}
}

func TestSQLiteBenchmarkConfigCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	cfg := &BenchmarkConfig{
		Name:         "test-config",
		ModelID:      "model-1",
		ModelName:    "测试模型",
		LlamaCppPath: "/usr/local/bin/llama-server",
		Devices:      []string{"cuda:0"},
		Params:       map[string]string{"threads": "4"},
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.CreateBenchmarkConfig(ctx, cfg); err != nil {
		t.Fatalf("CreateBenchmarkConfig: %v", err)
	}

	got, err := s.GetBenchmarkConfig(ctx, "test-config")
	if err != nil {
		t.Fatalf("GetBenchmarkConfig: %v", err)
	}
	if got.ModelID != "model-1" {
		t.Errorf("ModelID = %q, want %q", got.ModelID, "model-1")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt 不应为零值")
	}
	if len(got.Devices) != 1 || got.Devices[0] != "cuda:0" {
		t.Errorf("Devices = %v, want [cuda:0]", got.Devices)
	}

	configs, err := s.ListBenchmarkConfigs(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListBenchmarkConfigs: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("len = %d, want 1", len(configs))
	}

	got.LlamaCppPath = "/new/path"
	if err := s.UpdateBenchmarkConfig(ctx, got); err != nil {
		t.Fatalf("UpdateBenchmarkConfig: %v", err)
	}
	updated, _ := s.GetBenchmarkConfig(ctx, "test-config")
	if updated.LlamaCppPath != "/new/path" {
		t.Errorf("LlamaCppPath = %q, want /new/path", updated.LlamaCppPath)
	}

	if err := s.DeleteBenchmarkConfig(ctx, "test-config"); err != nil {
		t.Fatalf("DeleteBenchmarkConfig: %v", err)
	}
	_, err = s.GetBenchmarkConfig(ctx, "test-config")
	if err != ErrBenchmarkConfigNotFound {
		t.Errorf("err = %v, want ErrBenchmarkConfigNotFound", err)
	}
}

func TestSQLiteModelLoadConfigCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	cfg := &ModelLoadConfig{
		NodeID:    "node-1",
		ModelID:   "model-1",
		ModelName: "测试模型",
		Name:      "",
		Config:    map[string]interface{}{"gpu_layers": float64(99)},
	}
	if err := s.SaveModelLoadConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveModelLoadConfig: %v", err)
	}

	got, err := s.GetModelLoadConfig(ctx, "node-1", "model-1")
	if err != nil {
		t.Fatalf("GetModelLoadConfig: %v", err)
	}
	if got.ModelName != "测试模型" {
		t.Errorf("ModelName = %q, want %q", got.ModelName, "测试模型")
	}

	configs, err := s.ListModelLoadConfigs(ctx, "node-1", "model-1")
	if err != nil {
		t.Fatalf("ListModelLoadConfigs: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("len = %d, want 1", len(configs))
	}

	if err := s.DeleteModelLoadConfig(ctx, "node-1", "model-1"); err != nil {
		t.Fatalf("DeleteModelLoadConfig: %v", err)
	}
	_, err = s.GetModelLoadConfig(ctx, "node-1", "model-1")
	if err != ErrModelLoadConfigNotFound {
		t.Errorf("err = %v, want ErrModelLoadConfigNotFound", err)
	}
}

func TestSQLiteNamedModelLoadConfig(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	cfg := &ModelLoadConfig{
		NodeID:    "node-1",
		ModelID:   "model-1",
		ModelName: "测试模型",
		Name:      "fast-preset",
		Config:    map[string]interface{}{"gpu_layers": float64(50)},
	}
	if err := s.SaveNamedModelLoadConfig(ctx, cfg); err != nil {
		t.Fatalf("SaveNamedModelLoadConfig: %v", err)
	}

	configs, err := s.ListModelLoadConfigs(ctx, "node-1", "model-1")
	if err != nil {
		t.Fatalf("ListModelLoadConfigs: %v", err)
	}
	if len(configs) != 1 || configs[0].Name != "fast-preset" {
		t.Fatalf("配置不匹配: %v", configs)
	}

	if err := s.DeleteNamedModelLoadConfig(ctx, "node-1", "model-1", "fast-preset"); err != nil {
		t.Fatalf("DeleteNamedModelLoadConfig: %v", err)
	}
	configs, _ = s.ListModelLoadConfigs(ctx, "node-1", "model-1")
	if len(configs) != 0 {
		t.Errorf("删除后配置数 = %d, want 0", len(configs))
	}
}

func TestSQLiteModelMetadataCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	metadata := &ModelMetadata{
		ModelID:      "model-1",
		Alias:        "我的模型",
		Favourite:    true,
		Tags:         []string{"chat", "english"},
		Description:  "测试模型描述",
		Capabilities: &Capabilities{TTS: true},
	}
	if err := s.SaveModelMetadata(ctx, metadata); err != nil {
		t.Fatalf("SaveModelMetadata: %v", err)
	}

	got, err := s.GetModelMetadata(ctx, "model-1")
	if err != nil {
		t.Fatalf("GetModelMetadata: %v", err)
	}
	if got.Alias != "我的模型" {
		t.Errorf("Alias = %q, want %q", got.Alias, "我的模型")
	}
	if !got.Favourite {
		t.Error("期望 Favourite = true")
	}
	if !got.Capabilities.TTS {
		t.Error("期望 Capabilities.TTS = true")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt 不应为零值")
	}

	metadatas, err := s.ListModelMetadata(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListModelMetadata: %v", err)
	}
	if len(metadatas) != 1 {
		t.Fatalf("len = %d, want 1", len(metadatas))
	}

	allMeta, err := s.GetAllModelMetadata(ctx)
	if err != nil {
		t.Fatalf("GetAllModelMetadata: %v", err)
	}
	if len(allMeta) != 1 {
		t.Fatalf("len = %d, want 1", len(allMeta))
	}

	if err := s.DeleteModelMetadata(ctx, "model-1"); err != nil {
		t.Fatalf("DeleteModelMetadata: %v", err)
	}
	_, err = s.GetModelMetadata(ctx, "model-1")
	if err != ErrModelMetadataNotFound {
		t.Errorf("err = %v, want ErrModelMetadataNotFound", err)
	}
}

func TestSQLiteTTSHistoryCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	item := &TTSHistoryItem{
		Model:     "tts-model",
		InputText: "你好世界",
		AudioPath: "/tmp/audio.wav",
		Format:    "wav",
		Duration:  2.5,
	}
	if err := s.CreateTTSHistory(ctx, item); err != nil {
		t.Fatalf("CreateTTSHistory: %v", err)
	}
	if item.ID == "" {
		t.Fatal("期望自动生成 ID")
	}

	got, err := s.GetTTSHistory(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetTTSHistory: %v", err)
	}
	if got.InputText != "你好世界" {
		t.Errorf("InputText = %q, want %q", got.InputText, "你好世界")
	}

	items, err := s.ListTTSHistory(ctx, 100, 0, nil)
	if err != nil {
		t.Fatalf("ListTTSHistory: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}

	if err := s.UpdateTTSHistoryFavourite(ctx, item.ID, true); err != nil {
		t.Fatalf("UpdateTTSHistoryFavourite: %v", err)
	}
	got, _ = s.GetTTSHistory(ctx, item.ID)
	if !got.Favourite {
		t.Error("期望 Favourite = true")
	}

	favOnly := true
	favItems, err := s.ListTTSHistory(ctx, 100, 0, &favOnly)
	if err != nil {
		t.Fatalf("ListTTSHistory(favOnly): %v", err)
	}
	if len(favItems) != 1 {
		t.Errorf("收藏项数 = %d, want 1", len(favItems))
	}

	if err := s.DeleteTTSHistory(ctx, item.ID); err != nil {
		t.Fatalf("DeleteTTSHistory: %v", err)
	}
	_, err = s.GetTTSHistory(ctx, item.ID)
	if err != ErrTTSHistoryNotFound {
		t.Errorf("err = %v, want ErrTTSHistoryNotFound", err)
	}
}

func TestSQLiteDownloadTaskCRUD(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	task := &DownloadTask{
		URL:             "https://example.com/model.gguf",
		Path:            "/tmp/models",
		FileName:        "model.gguf",
		State:           "idle",
		TotalBytes:      1024000,
		RangeSupported:  true,
		FileType:        "gguf",
		SourceType:      "huggingface",
		RepoID:          "org/model",
		MaxRetries:      5,
		CreatedAt:       now,
	}
	if err := s.CreateDownloadTask(ctx, task); err != nil {
		t.Fatalf("CreateDownloadTask: %v", err)
	}
	if task.ID == "" {
		t.Fatal("期望自动生成 ID")
	}

	got, err := s.GetDownloadTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetDownloadTask: %v", err)
	}
	if got.URL != "https://example.com/model.gguf" {
		t.Errorf("URL = %q", got.URL)
	}
	if !got.RangeSupported {
		t.Error("期望 RangeSupported = true")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt 不应为零值")
	}

	tasks, err := s.ListDownloadTasks(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListDownloadTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len = %d, want 1", len(tasks))
	}

	got.State = "completed"
	got.DownloadedBytes = 1024000
	if err := s.UpdateDownloadTask(ctx, got); err != nil {
		t.Fatalf("UpdateDownloadTask: %v", err)
	}

	active, err := s.ListActiveDownloadTasks(ctx)
	if err != nil {
		t.Fatalf("ListActiveDownloadTasks: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("活跃任务数 = %d, want 0", len(active))
	}

	if err := s.DeleteDownloadTask(ctx, task.ID); err != nil {
		t.Fatalf("DeleteDownloadTask: %v", err)
	}
	_, err = s.GetDownloadTask(ctx, task.ID)
	if err != ErrDownloadTaskNotFound {
		t.Errorf("err = %v, want ErrDownloadTaskNotFound", err)
	}
}

func TestSQLiteStats(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats["type"] != "sqlite" {
		t.Errorf("type = %v, want sqlite", stats["type"])
	}
	if stats["path"] == nil {
		t.Error("期望 path 字段存在")
	}

	// 创建数据后再检查
	s.CreateConversation(ctx, &Conversation{Model: "m", Title: "t"})
	stats, _ = s.Stats()
	if stats["conversations"].(int64) != 1 {
		t.Errorf("conversations = %v, want 1", stats["conversations"])
	}
}

func TestSQLiteEmptyList(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	convs, err := s.ListConversations(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convs) != 0 {
		t.Errorf("len = %d, want 0", len(convs))
	}

	benchmarks, err := s.ListBenchmarks(ctx, "", 100, 0)
	if err != nil {
		t.Fatalf("ListBenchmarks: %v", err)
	}
	if len(benchmarks) != 0 {
		t.Errorf("len = %d, want 0", len(benchmarks))
	}

	tasks, err := s.ListDownloadTasks(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListDownloadTasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("len = %d, want 0", len(tasks))
	}
}

func TestSQLitePagination(t *testing.T) {
	s := newTestSQLiteStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		s.CreateConversation(ctx, &Conversation{Model: "model", Title: "会话"})
	}

	page1, _ := s.ListConversations(ctx, 2, 0)
	if len(page1) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1))
	}

	page2, _ := s.ListConversations(ctx, 2, 2)
	if len(page2) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2))
	}

	page3, _ := s.ListConversations(ctx, 2, 4)
	if len(page3) != 1 {
		t.Errorf("page3 len = %d, want 1", len(page3))
	}

	page4, _ := s.ListConversations(ctx, 2, 10)
	if len(page4) != 0 {
		t.Errorf("page4 len = %d, want 0", len(page4))
	}
}

func TestSQLiteDBFileCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "test.db")
	store, err := NewSQLiteStore(&SQLiteConfig{Path: path})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	store.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("数据库文件未创建: %s", path)
	}
}
