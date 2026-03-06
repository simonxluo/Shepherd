package download

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"github.com/google/uuid"
)

// Manager manages download tasks
type Manager struct {
	config          DownloadConfig
	tasks           map[string]*Task
	activeDownloads int32
	listeners       []ProgressListener
	progressChan    chan Progress

	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
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
		config:       config,
		tasks:        make(map[string]*Task),
		progressChan: make(chan Progress, 100),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start progress broadcaster
	m.wg.Add(1)
	go m.progressBroadcaster()

	return m
}

// AddProgressListener adds a listener for progress updates
func (m *Manager) AddProgressListener(listener ProgressListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, listener)
}

// CreateTask creates a new download task
func (m *Manager) CreateTask(url, path, fileName string) (string, error) {
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
		State:      StateIdle,
		CreatedAt:  time.Now(),
		MaxRetries: m.config.RetryCount,
	}

	// Add to tasks map
	m.mu.Lock()
	m.tasks[taskID] = task
	m.mu.Unlock()

	// Notify task created
	m.notifyProgress(Progress{
		TaskID: taskID,
		State:  StateIdle,
	})

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

	// Delete completed file if exists
	if task.State == StateCompleted {
		filePath := filepath.Join(task.Path, task.FileName)
		utils.RemoveQuietly(filePath)
	}

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

// notifyProgress sends progress notification
func (m *Manager) notifyProgress(progress Progress) {
	select {
	case m.progressChan <- progress:
	case <-time.After(100 * time.Millisecond):
		// Don't block if channel is full
	}
}

// progressBroadcaster broadcasts progress to all listeners
func (m *Manager) progressBroadcaster() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case progress := <-m.progressChan:
			m.mu.RLock()
			listeners := make([]ProgressListener, len(m.listeners))
			copy(listeners, m.listeners)
			m.mu.RUnlock()

			for _, listener := range listeners {
				listener(progress)
			}
		}
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

// SaveTasks saves all tasks to a JSON file for persistence
func (m *Manager) SaveTasks(filePath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(m.tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Write to temp file first
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		utils.RemoveQuietly(tempPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// LoadTasks loads tasks from a JSON file
func (m *Manager) LoadTasks(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No saved tasks
		}
		return fmt.Errorf("failed to read tasks: %w", err)
	}

	var tasks map[string]*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("failed to parse tasks: %w", err)
	}

	m.mu.Lock()
	m.tasks = tasks
	m.mu.Unlock()

	return nil
}

// ResumePendingTasks resumes all incomplete tasks
func (m *Manager) ResumePendingTasks() error {
	tasks := m.ListTasks()

	for _, task := range tasks {
		if task.State == StateIdle || task.State == StatePaused {
			if task.DownloadedBytes > 0 {
				// Can resume
				if m.canStartDownload() {
					m.wg.Add(1)
					go m.executeDownload(task)
				}
			}
		}
	}

	return nil
}

// closeQuietly closes a file and ignores the error (used in defer)
func closeQuietly(c io.Closer) {
	_ = c.Close()
}

// removeQuietly removes a file and ignores errors (file may not exist)
func removeQuietly(path string) {
	utils.RemoveQuietly(path)
}
