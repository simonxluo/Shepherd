package download

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
)

// Manager manages download tasks
type Manager struct {
	config          DownloadConfig
	tasks           map[string]*Task
	activeDownloads int32

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager creates a new download manager
func NewManager(config DownloadConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 4
	}
	if config.ChunkSize == 0 {
		config.ChunkSize = 1024 * 1024 // 1MB
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.RetryCount == 0 {
		config.RetryCount = 5
	}
	if config.MinPartSize == 0 {
		config.MinPartSize = 8 * 1024 * 1024 // 8MB
	}
	if config.MaxParallelism == 0 {
		config.MaxParallelism = 8
	}
	if config.UserAgent == "" {
		config.UserAgent = "Shepherd Download Manager"
	}

	m := &Manager{
		config: config,
		tasks:  make(map[string]*Task),
		ctx:    ctx,
		cancel: cancel,
	}

	return m
}

// CreateTask creates a new download task
func (m *Manager) CreateTask(url, path, fileName, source, repoId string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Generate task ID
	taskID := uuid.New().String()

	// Create task
	task := &Task{
		ID:         taskID,
		URL:        url,
		Path:       path,
		FileName:   fileName,
		SourceType: source,
		RepoID:     repoId,
		State:      StateIdle,
		CreatedAt:  time.Now(),
		MaxRetries: m.config.RetryCount,
	}
	// Add to tasks map
	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()

	// Try to start download immediately
	if m.canStartDownload() {
		m.wg.Add(1)
		go m.executeDownload(task)
	} else {
		task.State = StateIdle
	}

	return taskID, nil
}

// Pause pauses a download task
func (m *Manager) Pause(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.State != StateDownloading {
		return fmt.Errorf("task is not downloading: %s", taskID)
	}

	task.Paused = true
	task.StopRequested = true
	task.State = StatePaused

	return nil
}

// Resume resumes a paused download
func (m *Manager) Resume(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.State != StatePaused {
		return fmt.Errorf("task is not paused: %s", taskID)
	}

	task.Paused = false
	task.StopRequested = false

	// Resume download
	if m.canStartDownload() {
		m.wg.Add(1)
		go m.executeDownload(task)
	}

	return nil
}

// Delete deletes a download task
func (m *Manager) Delete(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Stop if downloading
	if task.State == StateDownloading || task.State == StatePreparing {
		task.StopRequested = true
	}
	// Delete from map
	delete(m.tasks, taskID)

	// Delete temp files if any
	if task.TempFileName != "" {
		tempPath := filepath.Join(task.Path, task.TempFileName)
		utils.RemoveQuietly(tempPath)
	}

	// Delete part files (分片文件)
	partPattern := filepath.Join(task.Path, task.FileName+".downloading.part*")
	if partFiles, err := filepath.Glob(partPattern); err == nil {
		for _, partFile := range partFiles {
			utils.RemoveQuietly(partFile)
		}
	}

	// Delete downloaded file (不管状态)
	filePath := filepath.Join(task.Path, task.FileName)
	utils.RemoveQuietly(filePath)

	return nil
}

// GetTask returns a task by ID
func (m *Manager) GetTask(taskID string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, false
	}

	// Return a copy
	taskCopy := *task
	return &taskCopy, true
}

// ListTasks returns all tasks
func (m *Manager) ListTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}

	return tasks
}

// canStartDownload checks if we can start a new download
func (m *Manager) canStartDownload() bool {
	return atomic.LoadInt32(&m.activeDownloads) < int32(m.config.MaxConcurrent)
}

// executeDownload executes a download task
func (m *Manager) executeDownload(task *Task) {
	defer m.wg.Done()

	// Check context
	select {
	case <-m.ctx.Done():
		return
	default:
	}

	atomic.AddInt32(&m.activeDownloads, 1)
	defer atomic.AddInt32(&m.activeDownloads, -1)

	// Update state
	task.State = StatePreparing
	task.StartedAt = time.Now()

	// Execute download
	downloader := newDownloader(m.config, task)
	err := downloader.Download(m.ctx)

	if err != nil {
		if task.StopRequested || task.Paused {
			task.State = StatePaused
		} else {
			task.State = StateFailed
			task.Error = err
		}
	} else {
		task.State = StateCompleted
		task.FinishedAt = time.Now()
	}
}

// Close closes the manager and waits for all downloads to finish
func (m *Manager) Close() error {
	m.cancel()

	// Wait for all downloads to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for downloads to finish")
	}
}

// closeQuietly closes a file and ignores the error (used in defer)
func closeQuietly(c io.Closer) {
	_ = c.Close()
}

// removeQuietly removes a file and ignores errors (file may not exist)
func removeQuietly(path string) {
	utils.RemoveQuietly(path)
}
