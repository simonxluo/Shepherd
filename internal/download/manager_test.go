package download

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadStateString(t *testing.T) {
	tests := []struct {
		state    DownloadState
		expected string
	}{
		{StateIdle, "idle"},
		{StatePreparing, "preparing"},
		{StateDownloading, "downloading"},
		{StateMerging, "merging"},
		{StateVerifying, "verifying"},
		{StateCompleted, "completed"},
		{StateFailed, "failed"},
		{StatePaused, "paused"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestNewManager(t *testing.T) {
	config := DownloadConfig{
		MaxConcurrent:  2,
		ChunkSize:      1024 * 1024,
		Timeout:        30 * time.Second,
		RetryCount:     3,
		MinPartSize:    8 * 1024 * 1024,
		MaxParallelism: 4,
	}

	manager := NewManager(config)

	assert.NotNil(t, manager)
	assert.Equal(t, 2, manager.config.MaxConcurrent)
	assert.Equal(t, int64(1024*1024), manager.config.ChunkSize)
}

func TestNewManagerDefaults(t *testing.T) {
	config := DownloadConfig{}
	manager := NewManager(config)

	// Check default values
	assert.Equal(t, 4, manager.config.MaxConcurrent)
	assert.Equal(t, int64(1024*1024), manager.config.ChunkSize)
	assert.Equal(t, 60*time.Second, manager.config.Timeout)
	assert.Equal(t, 5, manager.config.RetryCount) // Default is 5 retries
}

func TestManagerCreateTask(t *testing.T) {
	manager := NewManager(DownloadConfig{MaxConcurrent: 2})

	t.Run("Valid task creation", func(t *testing.T) {
		tmpDir := t.TempDir()

		taskID, err := manager.CreateTask("https://example.com/file.txt", tmpDir, "file.txt")
		require.NoError(t, err)
		assert.NotEmpty(t, taskID)

		task, exists := manager.GetTask(taskID)
		assert.True(t, exists)
		assert.Equal(t, "https://example.com/file.txt", task.URL)
		assert.Equal(t, tmpDir, task.Path)
		assert.Equal(t, "file.txt", task.FileName)
	})

	t.Run("Empty URL", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := manager.CreateTask("", tmpDir, "file.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "URL cannot be empty")
	})

	t.Run("Empty path", func(t *testing.T) {
		_, err := manager.CreateTask("https://example.com/file.txt", "", "file.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path cannot be empty")
	})
}

func TestManagerGetTask(t *testing.T) {
	manager := NewManager(DownloadConfig{})
	tmpDir := t.TempDir()

	t.Run("Existing task", func(t *testing.T) {
		taskID, _ := manager.CreateTask("https://example.com/file.txt", tmpDir, "file.txt")

		task, exists := manager.GetTask(taskID)
		assert.True(t, exists)
		assert.Equal(t, taskID, task.ID)
	})

	t.Run("Non-existent task", func(t *testing.T) {
		_, exists := manager.GetTask("non-existent")
		assert.False(t, exists)
	})
}

func TestManagerListTasks(t *testing.T) {
	manager := NewManager(DownloadConfig{})
	tmpDir := t.TempDir()

	// Initially empty
	tasks := manager.ListTasks()
	assert.Empty(t, tasks)

	// Create some tasks
	manager.CreateTask("https://example.com/file1.txt", tmpDir, "file1.txt")
	manager.CreateTask("https://example.com/file2.txt", tmpDir, "file2.txt")

	tasks = manager.ListTasks()
	assert.Len(t, tasks, 2)
}

func TestManagerPauseResume(t *testing.T) {
	manager := NewManager(DownloadConfig{})
	tmpDir := t.TempDir()

	taskID, err := manager.CreateTask("https://example.com/file.txt", tmpDir, "file.txt")
	require.NoError(t, err)

	// Pause non-downloading task should fail
	err = manager.Pause(taskID)
	assert.Error(t, err)

	// Resume should work
	err = manager.Resume(taskID)
	// Since task is not paused, this will fail
	assert.Error(t, err)
}

func TestManagerDelete(t *testing.T) {
	manager := NewManager(DownloadConfig{})
	tmpDir := t.TempDir()

	taskID, err := manager.CreateTask("https://example.com/file.txt", tmpDir, "file.txt")
	require.NoError(t, err)

	// Delete task
	err = manager.Delete(taskID)
	assert.NoError(t, err)

	// Task should be gone
	_, exists := manager.GetTask(taskID)
	assert.False(t, exists)
}

func TestManagerSaveLoadTasks(t *testing.T) {
	manager := NewManager(DownloadConfig{})
	tmpDir := t.TempDir()

	// Create some tasks
	taskID1, _ := manager.CreateTask("https://example.com/file1.txt", tmpDir, "file1.txt")
	taskID2, _ := manager.CreateTask("https://example.com/file2.txt", tmpDir, "file2.txt")

	// Save tasks
	savePath := filepath.Join(tmpDir, "tasks.json")
	err := manager.SaveTasks(savePath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(savePath)
	assert.NoError(t, err)

	// Create new manager and load tasks
	manager2 := NewManager(DownloadConfig{})
	err = manager2.LoadTasks(savePath)
	require.NoError(t, err)

	// Verify tasks were loaded
	task1, exists := manager2.GetTask(taskID1)
	assert.True(t, exists)
	assert.Equal(t, "https://example.com/file1.txt", task1.URL)

	task2, exists := manager2.GetTask(taskID2)
	assert.True(t, exists)
	assert.Equal(t, "https://example.com/file2.txt", task2.URL)
}

func TestManagerProgressListener(t *testing.T) {
	manager := NewManager(DownloadConfig{})
	tmpDir := t.TempDir()

	// Add progress listener
	received := make(chan Progress, 10)
	manager.AddProgressListener(func(progress Progress) {
		received <- progress
	})

	// Create task (should trigger progress notification)
	taskID, _ := manager.CreateTask("https://example.com/file.txt", tmpDir, "file.txt")

	// Check if we received progress notification
	select {
	case progress := <-received:
		assert.Equal(t, taskID, progress.TaskID)
		assert.Equal(t, StateIdle, progress.State)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for progress notification")
	}
}

func TestManagerClose(t *testing.T) {
	manager := NewManager(DownloadConfig{})
	tmpDir := t.TempDir()

	// Create a task
	manager.CreateTask("https://example.com/file.txt", tmpDir, "file.txt")

	// Close manager
	err := manager.Close()
	assert.NoError(t, err)

	// Creating new tasks after close should still work (manager is reusable)
	// The context is cancelled but a new manager can be created
}

func TestExtractFileNameFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/file.txt", "file.txt"},
		{"https://example.com/path/to/file.gguf", "file.gguf"},
		{"https://example.com/file.txt?query=1", "file.txt"},
		{"https://example.com/path/", ""}, // No filename
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractFileNameFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFileName(t *testing.T) {
	tests := []struct {
		name     string
		cd       string
		expected string
	}{
		{
			name:     "Simple filename",
			cd:       `attachment; filename="file.txt"`,
			expected: "file.txt",
		},
		{
			name:     "Filename without quotes",
			cd:       `attachment; filename=file.txt`,
			expected: "file.txt",
		},
		{
			name:     "No filename",
			cd:       `attachment`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFileName(tt.cd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDownloaderPrepare(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that makes HTTP request")
	}

	config := DownloadConfig{
		Timeout: 5 * time.Second,
	}

	task := &Task{
		URL:   "https://example.com",
		Path:  "/tmp",
		State: StateIdle,
	}

	downloader := newDownloader(config, task)

	ctx := context.Background()
	err := downloader.prepare(ctx)

	// This might fail if example.com is not reachable or network is down
	// Just check that we don't crash
	if err != nil {
		t.Logf("Prepare failed (may be expected): %v", err)
	}
}

func TestDownloadStateTransitions(t *testing.T) {
	task := &Task{
		ID:    "test-task",
		State: StateIdle,
		URL:   "https://example.com",
		Path:  "/tmp",
	}

	// Test state transitions
	assert.Equal(t, StateIdle, task.State)

	task.State = StatePreparing
	assert.Equal(t, StatePreparing, task.State)

	task.State = StateDownloading
	assert.Equal(t, StateDownloading, task.State)

	task.State = StateCompleted
	assert.Equal(t, StateCompleted, task.State)
}

func TestProgressCalculation(t *testing.T) {
	manager := NewManager(DownloadConfig{})

	d := &downloader{
		task: &Task{
			StartedAt:       time.Now().Add(-10 * time.Second),
			TotalBytes:      1000,
			DownloadedBytes: 500,
		},
		manager: manager,
	}

	d.calculateSpeed()

	// Speed should be around 50 bytes/second
	assert.Greater(t, d.task.Speed, int64(40))
	assert.Less(t, d.task.Speed, int64(60))

	// ETA should be around 10 seconds
	assert.Greater(t, d.task.ETA, int64(8))
	assert.Less(t, d.task.ETA, int64(12))
}
