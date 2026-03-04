package model

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/gguf"
	"github.com/shepherd-project/shepherd/Shepherd/internal/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestStorageMgr 创建测试用的存储管理器
func createTestStorageMgr(tb testing.TB) *storage.Manager {
	storageCfg := storage.StorageConfig{
		Type: storage.StorageTypeMemory,
	}
	storageMgr, err := storage.NewManager(&storageCfg)
	require.NoError(tb, err, "无法创建存储管理器")
	return storageMgr
}

func TestLoadStateString(t *testing.T) {
	tests := []struct {
		state    LoadState
		expected string
	}{
		{StateUnloaded, "unloaded"},
		{StateLoading, "loading"},
		{StateLoaded, "running"},
		{StateUnloading, "unloading"},
		{StateError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestNewManager(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.models)
	assert.NotNil(t, manager.statuses)
	assert.NotNil(t, manager.scanStatus)
}

func TestManagerIsGGUFFile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"GGUF file", "/path/to/model.gguf", true},
		{"Uppercase GGUF", "/path/to/model.GGUF", true},
		{"gguf- prefix", "/path/to/gguf-model.bin", true},
		{"No GGUF", "/path/to/model.bin", false},
		{"HuggingFace cache", "/cache/models--org--model/snapshots/abc123/model.gguf", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isGGUFFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManagerGetSetModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	t.Run("Get non-existent model", func(t *testing.T) {
		_, exists := manager.GetModel("non-existent")
		assert.False(t, exists)
	})

	t.Run("List models returns model list", func(t *testing.T) {
		models := manager.ListModels()
		// 不再期望为空，因为管理器会从配置文件加载模型
		// 只验证返回值不为 nil
		assert.NotNil(t, models)
	})
}

func TestManagerSetAlias(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// Create a temp directory with a mock model
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.gguf")

	// Create a minimal GGUF file for testing
	err := createMinimalGGUF(modelPath)
	require.NoError(t, err)

	// Load model
	model, err := manager.loadModel(modelPath)
	require.NoError(t, err)

	manager.mu.Lock()
	manager.models[model.ID] = model
	manager.mu.Unlock()

	// Set alias
	err = manager.SetAlias(model.ID, "Test Model")
	assert.NoError(t, err)

	// Verify
	retrieved, _ := manager.GetModel(model.ID)
	assert.Equal(t, "Test Model", retrieved.Alias)
}

func TestManagerSetFavourite(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// Create a temp directory with a mock model
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.gguf")

	// Create a minimal GGUF file for testing
	err := createMinimalGGUF(modelPath)
	require.NoError(t, err)

	// Load model
	model, err := manager.loadModel(modelPath)
	require.NoError(t, err)

	manager.mu.Lock()
	manager.models[model.ID] = model
	manager.mu.Unlock()

	// Set favourite
	err = manager.SetFavourite(model.ID, true)
	assert.NoError(t, err)

	// Verify
	retrieved, _ := manager.GetModel(model.ID)
	assert.True(t, retrieved.Favourite)
}

func TestManagerGetStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	t.Run("Get non-existent status", func(t *testing.T) {
		_, exists := manager.GetStatus("non-existent")
		assert.False(t, exists)
	})

	t.Run("List status initially empty", func(t *testing.T) {
		statuses := manager.ListStatus()
		assert.Empty(t, statuses)
	})
}

func TestManagerGetScanStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	status := manager.GetScanStatus()
	assert.NotNil(t, status)
	assert.False(t, status.Scanning)
}

func TestFindAvailablePort(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	port := manager.findAvailablePort()
	assert.GreaterOrEqual(t, port, 8081)
	assert.Less(t, port, 8181)
}

func TestFindMmproj(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	t.Run("No mmproj found", func(t *testing.T) {
		mmproj := manager.findMmproj("/path/to/model.gguf")
		assert.Empty(t, mmproj)
	})
}

func TestGenerateModelID(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	metadata := &gguf.Metadata{
		Name: "test-model",
	}

	id := manager.generateModelID("/path/to/model.gguf", metadata)
	// ID 现在是 hash-based 格式: model-{hash}
	assert.Contains(t, id, "model-")
	// 验证格式: base-hash (两部分)
	parts := strings.Split(id, "-")
	assert.Len(t, parts, 2)
	// hash 部分应该是 16 字符 (SHA256 的前 8 字节，hex 编码)
	assert.Len(t, parts[1], 16)
}

func TestScanStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// Initial status
	status := manager.GetScanStatus()
	assert.False(t, status.Scanning)
	assert.Equal(t, 0.0, status.Progress)
}

// createMinimalGGUF creates a minimal valid GGUF file for testing
// File size is at least 2048 bytes to pass model validation (requires >= 1024 bytes)
func createMinimalGGUF(path string) error {
	// GGUF header (24 bytes)
	header := []byte{
		// Magic: "GGUF" (0x47475546)
		0x47, 0x47, 0x55, 0x46,
		// Version: 3 (little endian uint32)
		0x03, 0x00, 0x00, 0x00,
		// Tensor count: 0 (little endian uint64)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Metadata KV count: 0 (little endian uint64)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	// Pad to 2048 bytes to meet minimum file size requirement
	padding := make([]byte, 2048-24)
	data := append(header, padding...)

	return os.WriteFile(path, data, 0644)
}

func TestLoadModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	t.Run("Load minimal GGUF", func(t *testing.T) {
		tmpDir := t.TempDir()
		modelPath := filepath.Join(tmpDir, "test-model.gguf")

		err := createMinimalGGUF(modelPath)
		require.NoError(t, err)

		model, err := manager.loadModel(modelPath)
		require.NoError(t, err)

		assert.NotNil(t, model)
		assert.NotEmpty(t, model.ID)
		assert.NotEmpty(t, model.Name) // Should default to file name
		assert.Equal(t, modelPath, model.Path)
	})

	t.Run("Non-existent file", func(t *testing.T) {
		_, err := manager.loadModel("/nonexistent/file.gguf")
		assert.Error(t, err)
	})
}

func TestScanPath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	t.Run("Scan directory with GGUF files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create some test files
		modelPath1 := filepath.Join(tmpDir, "model1.gguf")
		modelPath2 := filepath.Join(tmpDir, "model2.GGUF")
		nonModelPath := filepath.Join(tmpDir, "readme.txt")

		createMinimalGGUF(modelPath1)
		createMinimalGGUF(modelPath2)
		os.WriteFile(nonModelPath, []byte("test"), 0644)

		// Scan
		models, errors := manager.scanPath(context.Background(), tmpDir)

		assert.Len(t, models, 2) // Should find 2 GGUF files
		assert.Len(t, errors, 0)
	})

	t.Run("Scan non-existent directory", func(t *testing.T) {
		_, errors := manager.scanPath(context.Background(), "/nonexistent/path")

		assert.NotEmpty(t, errors)
	})
}

func TestLoadUnload(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	t.Run("Load non-existent model", func(t *testing.T) {
		req := &LoadRequest{
			ModelID: "non-existent",
			CtxSize: 4096,
		}

		result, err := manager.Load(req)
		assert.Error(t, err)
		// Result may be nil if load fails early, just check error
		_ = result
	})

	t.Run("Unload non-existent model", func(t *testing.T) {
		err := manager.Unload("non-existent")
		assert.Error(t, err)
	})
}

func TestScanConfigDefaults(t *testing.T) {
	config := ScanConfig{
		Paths: []string{"/models"},
	}

	assert.NotEmpty(t, config.Paths)
	assert.False(t, config.Recursive) // Default is false
}

func TestLoadRequestDefaults(t *testing.T) {
	req := &LoadRequest{
		ModelID: "test-model",
		CtxSize: 4096,
	}

	assert.Equal(t, "test-model", req.ModelID)
	assert.Equal(t, 4096, req.CtxSize)
}

func TestLoadResultDefaults(t *testing.T) {
	result := &LoadResult{
		Success: true,
		ModelID: "test-model",
		Port:    8081,
	}

	assert.True(t, result.Success)
	assert.Equal(t, "test-model", result.ModelID)
	assert.Equal(t, 8081, result.Port)
}

func TestModelDefaults(t *testing.T) {
	model := &Model{
		ID:   "test-id",
		Name: "test-model",
		Path: "/path/to/model.gguf",
	}

	assert.Equal(t, "test-id", model.ID)
	assert.Equal(t, "test-model", model.Name)
	assert.Equal(t, "/path/to/model.gguf", model.Path)
	assert.False(t, model.Favourite)
}

func TestScanResultDefaults(t *testing.T) {
	result := &ScanResult{
		Models:    []*Model{},
		Errors:    []ScanError{},
		ScannedAt: time.Now(),
	}

	assert.NotNil(t, result.Models)
	assert.NotNil(t, result.Errors)
}

func TestScanErrorDefaults(t *testing.T) {
	err := ScanError{
		Path:  "/path/to/file",
		Error: "test error",
	}

	assert.Equal(t, "/path/to/file", err.Path)
	assert.Equal(t, "test error", err.Error)
}

func BenchmarkListModels(b *testing.B) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(b)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// Add some models
	for i := 0; i < 100; i++ {
		model := &Model{
			ID:   fmt.Sprintf("model-%d", i),
			Name: fmt.Sprintf("Model %d", i),
			Path: fmt.Sprintf("/path/to/model-%d.gguf", i),
		}
		manager.models[model.ID] = model
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ListModels()
	}
}

// TestLoadAsync tests asynchronous model loading
func TestLoadAsync(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// 创建测试模型
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	err := createMinimalGGUF(modelPath)
	require.NoError(t, err)

	// 加载模型
	model, err := manager.loadModel(modelPath)
	require.NoError(t, err)

	manager.mu.Lock()
	manager.models[model.ID] = model
	manager.mu.Unlock()

	// 测试异步加载
	t.Run("异步加载返回立即响应", func(t *testing.T) {
		req := &LoadRequest{
			ModelID: model.ID,
			CtxSize: 4096,
		}

		result, err := manager.LoadAsync(req)

		// 应该立即返回而不阻塞
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Async)
		assert.True(t, result.Loading)
	})

	// 等待异步加载完成或失败
	time.Sleep(2 * time.Second)

	// 检查状态
	status, exists := manager.GetStatus(model.ID)
	if exists {
		// 状态应该被设置（可能是 loading, loaded 或 error）
		assert.NotEqual(t, StateUnloaded, status.State)
	}
}

// TestLoadAsyncAlreadyLoaded tests loading an already loaded model
func TestLoadAsyncAlreadyLoaded(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// 创建测试模型
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	err := createMinimalGGUF(modelPath)
	require.NoError(t, err)

	model, err := manager.loadModel(modelPath)
	require.NoError(t, err)

	manager.mu.Lock()
	manager.models[model.ID] = model
	manager.mu.Unlock()

	// 手动设置为已加载状态
	manager.mu.Lock()
	manager.statuses[model.ID] = &ModelStatus{
		ID:       model.ID,
		Name:     model.Name,
		State:    StateLoaded,
		Port:     8081,
		LoadedAt: time.Now(),
	}
	manager.mu.Unlock()

	// 尝试再次加载
	req := &LoadRequest{
		ModelID: model.ID,
		CtxSize: 4096,
	}

	result, err := manager.LoadAsync(req)

	assert.NoError(t, err)
	assert.True(t, result.Async)
	assert.True(t, result.AlreadyLoaded)
	assert.False(t, result.Loading)
}

// TestLoadAsyncAlreadyLoading tests loading while already loading
func TestLoadAsyncAlreadyLoading(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// 创建测试模型
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	err := createMinimalGGUF(modelPath)
	require.NoError(t, err)

	model, err := manager.loadModel(modelPath)
	require.NoError(t, err)

	manager.mu.Lock()
	manager.models[model.ID] = model
	manager.mu.Unlock()

	// 手动设置为加载中状态
	manager.mu.Lock()
	manager.statuses[model.ID] = &ModelStatus{
		ID:    model.ID,
		Name:  model.Name,
		State: StateLoading,
	}
	manager.mu.Unlock()

	// 尝试再次加载
	req := &LoadRequest{
		ModelID: model.ID,
		CtxSize: 4096,
	}

	result, err := manager.LoadAsync(req)

	assert.NoError(t, err)
	assert.True(t, result.Async)
	assert.True(t, result.Loading)
	assert.False(t, result.AlreadyLoaded)
}

// TestIsLoading tests the isLoading helper method
func TestIsLoading(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	modelID := "test-model"

	// 初始状态：未加载
	assert.False(t, manager.isLoading(modelID))

	// 设置为加载中
	manager.mu.Lock()
	manager.statuses[modelID] = &ModelStatus{
		ID:    modelID,
		State: StateLoading,
	}
	manager.mu.Unlock()

	assert.True(t, manager.isLoading(modelID))

	// 设置为已加载
	manager.mu.Lock()
	manager.statuses[modelID].State = StateLoaded
	manager.mu.Unlock()

	assert.False(t, manager.isLoading(modelID))
}

// TestLoadAsyncNonExistentModel tests loading a non-existent model
func TestLoadAsyncNonExistentModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	req := &LoadRequest{
		ModelID: "non-existent-model",
		CtxSize: 4096,
	}

	result, err := manager.LoadAsync(req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "model not found")
}

// TestLoadResultAsyncFields tests LoadResult async fields
func TestLoadResultAsyncFields(t *testing.T) {
	result := &LoadResult{
		Success:       true,
		ModelID:       "test-model",
		Port:          8081,
		CtxSize:       4096,
		Async:         true,
		Loading:       true,
		AlreadyLoaded: false,
	}

	assert.True(t, result.Async)
	assert.True(t, result.Loading)
	assert.False(t, result.AlreadyLoaded)
}

// BenchmarkLoadAsync benchmarks asynchronous loading
func BenchmarkLoadAsync(b *testing.B) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(b)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// 创建测试模型
	modelPath := "/tmp/bench-model.gguf"
	_ = createMinimalGGUF(modelPath)
	defer os.Remove(modelPath)

	model, _ := manager.loadModel(modelPath)
	manager.mu.Lock()
	manager.models[model.ID] = model
	manager.mu.Unlock()

	req := &LoadRequest{
		ModelID: model.ID,
		CtxSize: 4096,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.LoadAsync(req)
		// 清理状态以便下次测试
		manager.mu.Lock()
		delete(manager.statuses, model.ID)
		manager.mu.Unlock()
	}
}

// TestModelStatusTransitions tests model status state transitions
func TestModelStatusTransitions(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	modelID := "test-model"

	// 初始状态：unloaded
	status, exists := manager.GetStatus(modelID)
	assert.False(t, exists)

	// 设置为 loading
	manager.mu.Lock()
	manager.statuses[modelID] = &ModelStatus{
		ID:    modelID,
		State: StateLoading,
	}
	manager.mu.Unlock()

	status, exists = manager.GetStatus(modelID)
	assert.True(t, exists)
	assert.Equal(t, StateLoading, status.State)

	// 转换到 loaded
	manager.mu.Lock()
	manager.statuses[modelID].State = StateLoaded
	manager.statuses[modelID].Port = 8081
	manager.statuses[modelID].LoadedAt = time.Now()
	manager.mu.Unlock()

	status, _ = manager.GetStatus(modelID)
	assert.Equal(t, StateLoaded, status.State)
	assert.Equal(t, 8081, status.Port)

	// 转换到 error
	manager.mu.Lock()
	manager.statuses[modelID].State = StateError
	manager.statuses[modelID].Error = fmt.Errorf("test error")
	manager.mu.Unlock()

	status, _ = manager.GetStatus(modelID)
	assert.Equal(t, StateError, status.State)
	assert.NotNil(t, status.Error)
}

// TestListStatus tests listing all model statuses
func TestListStatus(t *testing.T) {
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// 添加多个状态
	modelIDs := []string{"model-1", "model-2", "model-3"}

	states := []LoadState{StateLoaded, StateLoading, StateError}

	for i, modelID := range modelIDs {
		manager.mu.Lock()
		manager.statuses[modelID] = &ModelStatus{
			ID:    modelID,
			State: states[i],
		}
		manager.mu.Unlock()
	}

	// 列出所有状态
	allStatuses := manager.ListStatus()

	assert.Len(t, allStatuses, 3)

	for _, modelID := range modelIDs {
		status, exists := allStatuses[modelID]
		assert.True(t, exists)
		assert.NotEmpty(t, status.ID)
	}
}

// TestMergeSplitModels 测试分卷模型合并功能
func TestMergeSplitModels(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过分卷模型测试（需要真实模型文件）")
	}

	// Qwen3.5-397B-A17B 分卷模型目录
	shardDir := "/home/user/workspace/LlamacppServer/build/models/unsloth/Qwen3.5-397B-A17B-GGUF/Qwen3.5-397B-A17B-MXFP4_MOE"

	shards := []string{
		"Qwen3.5-397B-A17B-MXFP4_MOE-00001-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00002-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00003-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00004-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00005-of-00006.gguf",
		"Qwen3.5-397B-A17B-MXFP4_MOE-00006-of-00006.gguf",
	}

	// 检查所有分卷文件是否存在
	for _, shard := range shards {
		path := filepath.Join(shardDir, shard)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("分卷文件不存在: %s", path)
			return
		}
	}

	// 创建配置
	cfg := config.DefaultConfig()
	cfgMgr := config.NewManager()
	procMgr := process.NewManager()
	portAllocator := port.NewPortAllocator(8000, 9000)

	storageMgr := createTestStorageMgr(t)
	manager := NewManager(cfg, cfgMgr, procMgr, portAllocator, storageMgr)

	// 手动加载所有分卷到 manager
	var totalSize int64 = 0
	shardSizes := make([]int64, len(shards))

	for i, shard := range shards {
		path := filepath.Join(shardDir, shard)

		// 读取元数据
		metadata, err := gguf.ReadMetadata(path)
		if err != nil {
			t.Fatalf("读取分卷 %d 元数据失败: %v", i+1, err)
		}

		// 获取文件大小
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("读取分卷 %d 文件信息失败: %v", i+1, err)
		}

		shardSizes[i] = info.Size()
		totalSize += info.Size()

		// 创建模型
		model := &Model{
			ID:          fmt.Sprintf("test-shard-%d", i+1),
			Name:        fmt.Sprintf("Qwen3.5-397B-A17B-%05d-of-00006", i+1),
			DisplayName: fmt.Sprintf("Qwen3.5-397B-A17B [shard %d]", i+1),
			Path:        path,
			Size:        info.Size(),
			Metadata:    metadata,
			ScannedAt:   time.Now(),
		}

		manager.mu.Lock()
		manager.models[model.ID] = model
		manager.mu.Unlock()
	}

	// 查找 mmproj 文件并添加到总大小（与 mergeSplitModels() 的逻辑一致）
	mmprojPath := filepath.Join(shardDir, "mmproj-F32.gguf")
	if info, err := os.Stat(mmprojPath); err == nil {
		totalSize += info.Size()
		fmt.Printf("[INFO] 找到 mmproj 文件: mmproj-F32.gguf (%.2f GB)\n", float64(info.Size())/(1024*1024*1024))
	}

	fmt.Printf("\n========== 测试前状态 ==========\n")
	fmt.Printf("分卷数量: %d\n", len(shards))
	fmt.Printf("总大小: %.2f GB (包含 mmproj)\n", float64(totalSize)/(1024*1024*1024))

	// 调用合并函数
	mergedCount := manager.mergeSplitModels()

	fmt.Printf("\n========== 合并结果 ==========\n")
	fmt.Printf("合并的组数: %d\n", mergedCount)

	// 验证合并结果
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	// 应该只剩下 1 个模型（主模型）
	fmt.Printf("合并后模型数量: %d\n", len(manager.models))

	// 查找主模型
	var primaryModel *Model
	for _, model := range manager.models {
		if strings.Contains(model.Name, "Qwen3.5-397B-A17B") {
			primaryModel = model
			break
		}
	}

	require.NotNil(t, primaryModel, "应该找到合并后的主模型")

	fmt.Printf("\n========== 主模型信息 ==========\n")
	fmt.Printf("ID: %s\n", primaryModel.ID)
	fmt.Printf("Name: %s\n", primaryModel.Name)
	fmt.Printf("DisplayName: %s\n", primaryModel.DisplayName)
	fmt.Printf("Size: %.2f GB\n", float64(primaryModel.Size)/(1024*1024*1024))
	fmt.Printf("ShardCount: %d\n", primaryModel.ShardCount)
	fmt.Printf("TotalSize: %.2f GB\n", float64(primaryModel.TotalSize)/(1024*1024*1024))
	fmt.Printf("len(ShardFiles): %d\n", len(primaryModel.ShardFiles))

	// 验证关键字段
	assert.Equal(t, len(shards), primaryModel.ShardCount, "分卷数量应该匹配")
	assert.Equal(t, totalSize, primaryModel.TotalSize, "总大小应该匹配")
	assert.Len(t, primaryModel.ShardFiles, len(shards), "分卷文件列表长度应该匹配")

	// 验证第一个分卷文件路径
	assert.Contains(t, primaryModel.ShardFiles[0], "00001-of-00006.gguf", "第一个分卷应该是 00001")

	// 验证所有分卷文件路径都存在
	for _, shardPath := range primaryModel.ShardFiles {
		_, err := os.Stat(shardPath)
		assert.NoError(t, err, "分卷文件应该存在: %s", shardPath)
	}

	// 验证 ShardCount > 0（确保合并成功）
	assert.Greater(t, primaryModel.ShardCount, 0, "ShardCount 应该大于 0")
	assert.Greater(t, int64(primaryModel.TotalSize), int64(0), "TotalSize 应该大于 0")

	fmt.Printf("\n✅ 所有验证通过！\n")
}

// TestIsSplitGGUF 测试分卷文件识别
func TestIsSplitGGUF(t *testing.T) {
	tests := []struct {
		filename     string
		expectedBool bool
		baseName     string
		partNum      int
		totalParts   int
	}{
		{
			"Qwen3.5-397B-A17B-MXFP4_MOE-00001-of-00006.gguf",
			true,
			"Qwen3.5-397B-A17B-MXFP4_MOE",
			1,
			6,
		},
		{
			"model-00003-of-00010.gguf",
			true,
			"model",
			3,
			10,
		},
		{
			"model.gguf",
			false,
			"",
			0,
			0,
		},
		{
			"model-Q4_K_M.gguf",
			false,
			"",
			0,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			isSplit, baseName, partNum, totalParts := isSplitGGUF(tt.filename)
			assert.Equal(t, tt.expectedBool, isSplit)
			if isSplit {
				assert.Equal(t, tt.baseName, baseName)
				assert.Equal(t, tt.partNum, partNum)
				assert.Equal(t, tt.totalParts, totalParts)
			}
		})
	}
}
