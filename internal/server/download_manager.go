// Package server provides download management functionality for the Shepherd application.
package server

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DownloadStatus represents the status of a download task
type DownloadStatus string

const (
	DownloadStatusPending     DownloadStatus = "pending"
	DownloadStatusDownloading DownloadStatus = "downloading"
	DownloadStatusPaused      DownloadStatus = "paused"
	DownloadStatusCompleted   DownloadStatus = "completed"
	DownloadStatusFailed      DownloadStatus = "failed"
)

// DownloadTask represents a single download task
type DownloadTask struct {
	ID          string         `json:"id"`
	URL         string         `json:"url"`
	TargetPath  string         `json:"target_path"`
	Status      DownloadStatus `json:"status"`
	Progress    float64        `json:"progress"`
	TotalBytes  int64          `json:"total_bytes"`
	Downloaded  int64          `json:"downloaded"`
	Speed       int64          `json:"speed"` // bytes per second
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	PausedAt    *time.Time     `json:"paused_at,omitempty"`

	ctx    context.Context
	cancel context.CancelFunc
	client *http.Client
	file   *os.File
	mu     sync.Mutex
}

// DownloadManager manages multiple download tasks
type DownloadManager struct {
	downloads      map[string]*DownloadTask
	mu             sync.RWMutex
	maxConcurrent  int
	activeCount    int
	semaphore      chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewDownloadManager creates a new download manager
func NewDownloadManager(maxConcurrent int) *DownloadManager {
	if maxConcurrent <= 0 {
		maxConcurrent = 3 // 默认最多3个并发下载
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DownloadManager{
		downloads:     make(map[string]*DownloadTask),
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// CreateDownload creates a new download task
func (dm *DownloadManager) CreateDownload(url, targetPath string) (*DownloadTask, error) {
	if url == "" {
		return nil, fmt.Errorf("下载 URL 不能为空")
	}
	if targetPath == "" {
		return nil, fmt.Errorf("目标路径不能为空")
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	id := fmt.Sprintf("dl-%d", time.Now().UnixNano())

	task := &DownloadTask{
		ID:        id,
		URL:       url,
		TargetPath: targetPath,
		Status:    DownloadStatusPending,
		CreatedAt: time.Now(),
		ctx:       dm.ctx,
		client:    &http.Client{Timeout: 30 * time.Second},
	}

	dm.downloads[id] = task

	// 启动下载（如果未超过并发限制）
	go dm.startDownload(task)

	return task, nil
}

// GetDownload retrieves a download task by ID
func (dm *DownloadManager) GetDownload(id string) (*DownloadTask, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	task, exists := dm.downloads[id]
	return task, exists
}

// ListDownloads returns all download tasks
func (dm *DownloadManager) ListDownloads() []*DownloadTask {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	downloads := make([]*DownloadTask, 0, len(dm.downloads))
	for _, task := range dm.downloads {
		downloads = append(downloads, task)
	}
	return downloads
}

// PauseDownload pauses a download task
func (dm *DownloadManager) PauseDownload(id string) error {
	task, exists := dm.GetDownload(id)
	if !exists {
		return fmt.Errorf("下载任务不存在")
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	if task.Status == DownloadStatusDownloading {
		task.Status = DownloadStatusPaused
		now := time.Now()
		task.PausedAt = &now
		if task.cancel != nil {
			task.cancel()
		}

		// 释放信号量
		dm.semaphore <- struct{}{}
		dm.activeCount--
	}

	return nil
}

// ResumeDownload resumes a paused download task
func (dm *DownloadManager) ResumeDownload(id string) error {
	task, exists := dm.GetDownload(id)
	if !exists {
		return fmt.Errorf("下载任务不存在")
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	if task.Status == DownloadStatusPaused {
		task.Status = DownloadStatusPending
		task.PausedAt = nil

		// 重新启动下载
		go dm.startDownload(task)
	}

	return nil
}

// DeleteDownload deletes a download task and removes partial files
func (dm *DownloadManager) DeleteDownload(id string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	task, exists := dm.downloads[id]
	if !exists {
		return fmt.Errorf("下载任务不存在")
	}

	// 取消下载
	task.mu.Lock()
	if task.cancel != nil {
		task.cancel()
	}
	task.mu.Unlock()

	// 删除部分下载的文件
	if _, err := os.Stat(task.TargetPath); err == nil {
		utils.RemoveQuietly(task.TargetPath)
	}

	delete(dm.downloads, id)
	return nil
}

// startDownload starts the download process
func (dm *DownloadManager) startDownload(task *DownloadTask) {
	// 获取信号量（控制并发）
	dm.semaphore <- struct{}{}
	dm.activeCount++

	task.mu.Lock()
	task.Status = DownloadStatusDownloading
	now := time.Now()
	task.StartedAt = &now

	// 创建任务上下文
	ctx, cancel := context.WithCancel(dm.ctx)
	task.cancel = cancel
	task.ctx = ctx
	task.mu.Unlock()

	// 执行下载
	task.download()

	// 释放信号量
	<-dm.semaphore
	dm.activeCount--
}

// download performs the actual download
func (t *DownloadTask) download() {
	defer func() {
		if t.file != nil {
			utils.CloseQuietly(t.file)
		}
	}()

	// 发起 HTTP 请求
	req, err := http.NewRequestWithContext(t.ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		t.setError(fmt.Sprintf("创建请求失败: %v", err))
		return
	}

	resp, err := t.client.Do(req)
	if err != nil {
		t.setError(fmt.Sprintf("HTTP 请求失败: %v", err))
		return
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.setError(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status))
		return
	}

	// 获取文件大小
	t.TotalBytes = resp.ContentLength

	// 创建目标目录
	if err := os.MkdirAll(filepath.Dir(t.TargetPath), 0755); err != nil {
		t.setError(fmt.Sprintf("创建目录失败: %v", err))
		return
	}

	// 创建目标文件
	file, err := os.Create(t.TargetPath)
	if err != nil {
		t.setError(fmt.Sprintf("创建文件失败: %v", err))
		return
	}
	t.file = file

	// 开始下载
	t.downloadWithProgress(resp.Body)
}

// downloadWithProgress downloads with progress tracking
func (t *DownloadTask) downloadWithProgress(body io.ReadCloser) {
	defer utils.CloseQuietly(body)

	buffer := make([]byte, 32*1024)
	var lastUpdate time.Time

	for {
		n, err := body.Read(buffer)
		if err != nil {
			if err == io.EOF {
				// 下载完成
				t.complete()
			} else {
				t.setError(fmt.Sprintf("读取数据失败: %v", err))
			}
			return
		}

		// 写入文件
		if _, err := t.file.Write(buffer[:n]); err != nil {
			t.setError(fmt.Sprintf("写入文件失败: %v", err))
			return
		}

		t.Downloaded += int64(n)

		// 更新进度
		if t.TotalBytes > 0 {
			t.Progress = float64(t.Downloaded) / float64(t.TotalBytes) * 100
		}

		// 计算速度（每秒更新一次）
		now := time.Now()
		if now.Sub(lastUpdate) >= time.Second {
			if t.StartedAt != nil {
				elapsed := now.Sub(*t.StartedAt).Seconds()
				if elapsed > 0 {
					t.Speed = int64(float64(t.Downloaded) / elapsed)
				}
			}
			lastUpdate = now
		}
	}
}

// complete marks the download as completed
func (t *DownloadTask) complete() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Status = DownloadStatusCompleted
	t.Progress = 100
	now := time.Now()
	t.CompletedAt = &now
}

// setError sets the error status
func (t *DownloadTask) setError(err string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Status = DownloadStatusFailed
	t.Error = err
}

// Stop stops all downloads
func (dm *DownloadManager) Stop() {
	dm.cancel()
}
