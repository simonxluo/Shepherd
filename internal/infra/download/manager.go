package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/simonxluo/Shepherd/internal/comm/event"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/infra/storage"
)

// ManagerDeps holds optional dependencies for the download manager
type ManagerDeps struct {
	Store    storage.Store
	EventMgr *event.Manager
}

// Manager manages download tasks
type Manager struct {
	config          DownloadConfig
	tasks           map[string]*Task
	activeDownloads int32
	deps            ManagerDeps

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Throttle persistence writes
	lastPersist     map[string]time.Time
	lastPersistMu   sync.Mutex
}

// NewManager creates a new download manager
func NewManager(config DownloadConfig, deps ManagerDeps) *Manager {
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
		config:      config,
		tasks:       make(map[string]*Task),
		deps:        deps,
		ctx:         ctx,
		cancel:      cancel,
		lastPersist: make(map[string]time.Time),
	}

	return m
}

// RestoreAndResume loads active/paused tasks from the database on startup,
// marks interrupted tasks (downloading/preparing) as paused, and auto-resumes
// up to MaxConcurrent tasks.
func (m *Manager) RestoreAndResume() {
	if m.deps.Store == nil {
		return
	}

	tasks, err := m.deps.Store.ListActiveDownloadTasks(context.Background())
	if err != nil {
		logger.Warnf("Failed to restore download tasks: %v", err)
		return
	}

	m.mu.Lock()
	for _, dt := range tasks {
		task := storageModelToTask(dt)
		// Mark interrupted tasks as paused
		if task.State == StateDownloading || task.State == StatePreparing || task.State == StateMerging || task.State == StateVerifying {
			task.State = StatePaused
			task.Paused = true
			task.StopRequested = false
		}
		if task.State == StateIdle {
			task.Paused = false
			task.StopRequested = false
		}
		m.tasks[task.ID] = task
	}
	m.mu.Unlock()

	// Persist the updated states
	m.mu.RLock()
	for _, task := range m.tasks {
		if task.State == StatePaused {
			m.persistTask(task)
		}
	}
	m.mu.RUnlock()

	// Auto-resume up to MaxConcurrent
	m.mu.RLock()
	var resumable []*Task
	for _, task := range m.tasks {
		if task.State == StateIdle || task.State == StatePaused {
			resumable = append(resumable, task)
		}
	}
	m.mu.RUnlock()

	resumed := 0
	for _, task := range resumable {
		if !m.canStartDownload() {
			break
		}
		task.Paused = false
		task.StopRequested = false
		task.State = StateIdle
		m.wg.Add(1)
		go m.executeDownload(task)
		resumed++
	}

	if len(tasks) > 0 {
		logger.Infof("Restored %d download tasks, auto-resumed %d", len(tasks), resumed)
	}
}

// CreateTask creates a new download task
func (m *Manager) CreateTask(url, path, fileName, source, repoId string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Ensure download directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("cannot create download directory %s: %w", path, err)
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

	// Persist new task
	m.persistNewTask(task)

	// Broadcast status
	m.broadcastStatus(task)

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

	if task.State != StateDownloading && task.State != StatePreparing {
		return fmt.Errorf("task is not downloading: %s", taskID)
	}

	task.Paused = true
	task.StopRequested = true
	task.State = StatePaused

	// Persist and broadcast
	m.persistTask(task)
	m.broadcastStatus(task)

	return nil
}

// Resume resumes a paused download
func (m *Manager) Resume(taskID string) error {
	m.mu.Lock()
	task, exists := m.tasks[taskID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.State != StatePaused && task.State != StateFailed {
		m.mu.Unlock()
		return fmt.Errorf("task is not paused or failed: %s", taskID)
	}

	task.Paused = false
	task.StopRequested = false
	task.Error = nil
	m.mu.Unlock()

	// Resume download
	if m.canStartDownload() {
		m.wg.Add(1)
		go m.executeDownload(task)
	} else {
		task.State = StateIdle
		m.persistTask(task)
		m.broadcastStatus(task)
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

	// Delete from persistence
	m.deletePersistedTask(taskID)

	// Delete temp files if any
	if task.TempFileName != "" {
		tempPath := filepath.Join(task.Path, task.TempFileName)
		utils.RemoveQuietly(tempPath)
	}

	// Delete part files
	partPattern := filepath.Join(task.Path, task.FileName+".downloading.part*")
	if partFiles, err := filepath.Glob(partPattern); err == nil {
		for _, partFile := range partFiles {
			utils.RemoveQuietly(partFile)
		}
	}

	// Delete downloaded file
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

// startNextTask dequeues the next idle task when a download slot opens
func (m *Manager) startNextTask() {
	if !m.canStartDownload() {
		return
	}

	m.mu.RLock()
	var next *Task
	for _, task := range m.tasks {
		if task.State == StateIdle && !task.Paused && !task.StopRequested {
			next = task
			break
		}
	}
	m.mu.RUnlock()

	if next != nil {
		m.wg.Add(1)
		go m.executeDownload(next)
	}
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
	defer func() {
		atomic.AddInt32(&m.activeDownloads, -1)
		// Try to start next queued task
		m.startNextTask()
	}()

	// Update state
	task.State = StatePreparing
	if task.StartedAt.IsZero() {
		task.StartedAt = time.Now()
	}

	// Broadcast & persist
	m.broadcastStatus(task)
	m.persistTask(task)

	// Build progress callback
	progressFn := func(t *Task) {
		m.broadcastProgress(t)
		m.throttledPersist(t)
	}

	// Execute download
	downloader := newDownloader(m.config, task, progressFn)
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

	// Broadcast & persist final state
	m.broadcastStatus(task)
	m.persistTask(task)
}

// Close closes the manager: persists active tasks as paused and waits for goroutines
func (m *Manager) Close() error {
	m.cancel()

	// Mark active tasks as paused for persistence
	m.mu.RLock()
	for _, task := range m.tasks {
		if task.State == StateDownloading || task.State == StatePreparing || task.State == StateMerging {
			task.Paused = true
			task.StopRequested = true
			task.State = StatePaused
			m.persistTask(task)
		}
	}
	m.mu.RUnlock()

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

// --- Persistence helpers ---

func (m *Manager) persistNewTask(task *Task) {
	if m.deps.Store == nil {
		return
	}
	dt := taskToStorageModel(task)
	if err := m.deps.Store.CreateDownloadTask(context.Background(), dt); err != nil {
		logger.Warnf("Failed to persist new download task %s: %v", task.ID, err)
	}
}

func (m *Manager) persistTask(task *Task) {
	if m.deps.Store == nil {
		return
	}
	dt := taskToStorageModel(task)
	if err := m.deps.Store.UpdateDownloadTask(context.Background(), dt); err != nil {
		logger.Warnf("Failed to persist download task %s: %v", task.ID, err)
	}
}

func (m *Manager) throttledPersist(task *Task) {
	if m.deps.Store == nil {
		return
	}
	m.lastPersistMu.Lock()
	last, ok := m.lastPersist[task.ID]
	if ok && time.Since(last) < 5*time.Second {
		m.lastPersistMu.Unlock()
		return
	}
	m.lastPersist[task.ID] = time.Now()
	m.lastPersistMu.Unlock()

	m.persistTask(task)
}

func (m *Manager) deletePersistedTask(taskID string) {
	if m.deps.Store == nil {
		return
	}
	if err := m.deps.Store.DeleteDownloadTask(context.Background(), taskID); err != nil {
		logger.Warnf("Failed to delete persisted download task %s: %v", taskID, err)
	}
}

// --- SSE broadcasting helpers ---

func (m *Manager) broadcastStatus(task *Task) {
	if m.deps.EventMgr == nil {
		return
	}
	var errMsg string
	if task.Error != nil {
		errMsg = task.Error.Error()
	}
	evt := event.NewDownloadStatusEvent(
		task.ID,
		task.State.String(),
		task.DownloadedBytes,
		task.TotalBytes,
		int(task.PartsCompleted),
		task.PartsTotal,
		task.FileName,
		errMsg,
	)
	m.deps.EventMgr.Broadcast(evt)
}

func (m *Manager) broadcastProgress(task *Task) {
	if m.deps.EventMgr == nil {
		return
	}
	var ratio float64
	if task.TotalBytes > 0 {
		ratio = float64(task.DownloadedBytes) / float64(task.TotalBytes)
	}
	evt := event.NewDownloadProgressEvent(
		task.ID,
		task.DownloadedBytes,
		task.TotalBytes,
		int(task.PartsCompleted),
		task.PartsTotal,
		ratio,
	)
	m.deps.EventMgr.Broadcast(evt)
}
