package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
)

// partDownload represents a single part of a parallel download
type partDownload struct {
	ID              int
	StartPos        int64
	EndPos          int64
	DownloadedBytes int64
	FileName        string
}

type downloader struct {
	config     DownloadConfig
	task       *Task
	client     *http.Client
	progressFn func(*Task)
}

// newDownloader creates a new downloader
func newDownloader(config DownloadConfig, task *Task, progressFn func(*Task)) *downloader {
	return &downloader{
		config: config,
		task:   task,
		client: &http.Client{
			// No client-level Timeout — it kills body streaming for large files.
			// We use per-request context timeouts for HEAD/metadata requests only.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 {
					task.FinalURL = req.URL.String()
				}
				return nil
			},
		},
		progressFn: progressFn,
	}
}

// Download executes the download
func (d *downloader) Download(ctx context.Context) error {
	// Check if paused
	if d.task.Paused {
		return nil
	}

	// Skip prepare if we already have metadata (resumed task)
	if d.task.TotalBytes == 0 || d.task.TempFileName == "" {
		if err := d.prepare(ctx); err != nil {
			return fmt.Errorf("prepare failed: %w", err)
		}
	} else {
		d.task.State = StateDownloading
	}

	// Check if paused
	if d.task.Paused {
		return nil
	}

	// Execute download
	if d.task.RangeSupported && d.task.TotalBytes > d.config.MinPartSize {
		if err := d.downloadParallel(ctx); err != nil {
			return err
		}
	} else {
		if err := d.downloadSimple(ctx); err != nil {
			return err
		}
	}

	// Check if paused
	if d.task.Paused {
		return nil
	}

	// Verify download
	d.task.State = StateVerifying
	if err := d.verify(); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	return nil
}

// prepare prepares the download by checking server capabilities
func (d *downloader) prepare(ctx context.Context) error {
	d.task.State = StatePreparing

	// Use a short timeout for HEAD request only
	headCtx, cancel := context.WithTimeout(ctx, d.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(headCtx, "HEAD", d.task.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", d.config.UserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer utils.CloseQuietly(resp.Body)

	// Get content length
	contentLength := resp.ContentLength
	if contentLength > 0 {
		d.task.TotalBytes = contentLength
	}

	// Check for range support
	acceptRange := resp.Header.Get("Accept-Ranges")
	d.task.RangeSupported = strings.ToLower(acceptRange) == "bytes"

	// Get ETag for validation
	d.task.ETag = resp.Header.Get("ETag")

	// Set file name if not provided
	if d.task.FileName == "" {
		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			d.task.FileName = parseFileName(cd)
		}
		if d.task.FileName == "" {
			d.task.FileName = extractFileNameFromURL(d.task.URL)
		}
	}

	// Create temp file name
	d.task.TempFileName = d.task.FileName + ".downloading"

	return nil
}

// downloadSimple performs a simple single-thread download
func (d *downloader) downloadSimple(ctx context.Context) error {
	d.task.State = StateDownloading

	// Check actual file size on disk to determine real start position
	tempPath := filepath.Join(d.task.Path, d.task.TempFileName)
	startPos := int64(0)
	if info, err := os.Stat(tempPath); err == nil {
		startPos = info.Size()
		// Sync task's DownloadedBytes with reality
		atomic.StoreInt64(&d.task.DownloadedBytes, startPos)
	} else {
		atomic.StoreInt64(&d.task.DownloadedBytes, 0)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", d.task.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", d.config.UserAgent)

	// Set range header for resume
	if startPos > 0 && d.task.RangeSupported {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startPos))
	}

	// Do request
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer utils.CloseQuietly(resp.Body)

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// If server returned 200 (not 206), it's sending the full file — reset position
	if resp.StatusCode == http.StatusOK && startPos > 0 {
		startPos = 0
		atomic.StoreInt64(&d.task.DownloadedBytes, 0)
	}

	// Update total bytes if not set
	if d.task.TotalBytes == 0 {
		d.task.TotalBytes = resp.ContentLength + startPos
	}

	// Open file
	var file *os.File
	if startPos > 0 {
		file, err = os.OpenFile(tempPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(tempPath)
	}

	if err != nil {
		return err
	}
	defer utils.CloseQuietly(file)

	// Start progress updater
	stopProgress := make(chan struct{})
	go d.updateProgress(stopProgress)
	defer close(stopProgress)

	// Download with progress tracking
	buf := make([]byte, d.config.ChunkSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if paused or stopped
		if d.task.Paused || d.task.StopRequested {
			return nil
		}

		// Read chunk
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return writeErr
			}

			atomic.AddInt64(&d.task.DownloadedBytes, int64(n))
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	// Rename to final file
	if err := os.Rename(tempPath, filepath.Join(d.task.Path, d.task.FileName)); err != nil {
		return err
	}

	d.task.TempFileName = ""
	return nil
}

// downloadParallel performs a parallel multi-thread download
func (d *downloader) downloadParallel(ctx context.Context) error {
	d.task.State = StateDownloading

	// Calculate part size
	partSize := d.task.TotalBytes / int64(d.config.MaxParallelism)
	if partSize < d.config.MinPartSize {
		partSize = d.config.MinPartSize
	}

	// Create parts
	var parts []partDownload
	numParts := int(d.task.TotalBytes / partSize)
	if d.task.TotalBytes%partSize != 0 {
		numParts++
	}

	if numParts > d.config.MaxParallelism {
		numParts = d.config.MaxParallelism
	}

	d.task.PartsTotal = numParts

	for i := 0; i < numParts; i++ {
		startPos := int64(i) * partSize
		endPos := startPos + partSize - 1
		if endPos >= d.task.TotalBytes || i == numParts-1 {
			endPos = d.task.TotalBytes - 1
		}

		part := partDownload{
			ID:       i,
			StartPos: startPos,
			EndPos:   endPos,
		}
		parts = append(parts, part)
	}

	d.task.Parts = parts

	// Start progress updater
	stopProgress := make(chan struct{})
	go d.updateProgress(stopProgress)

	// Download parts concurrently
	errChan := make(chan error, len(parts))

	for i := range parts {
		go func(part *partDownload) {
			errChan <- d.downloadPart(ctx, part)
		}(&parts[i])
	}

	// Wait for all parts
	var firstError error
	for i := 0; i < len(parts); i++ {
		if err := <-errChan; err != nil && firstError == nil {
			firstError = err
		}
	}

	close(stopProgress)

	if firstError != nil {
		return firstError
	}

	// Check if paused
	if d.task.Paused || d.task.StopRequested {
		return nil
	}

	// Merge parts
	d.task.State = StateMerging
	if err := d.mergeParts(); err != nil {
		return err
	}

	return nil
}

// downloadPart downloads a single part with resume support
func (d *downloader) downloadPart(ctx context.Context, part *partDownload) error {
	// Check if paused
	if d.task.Paused || d.task.StopRequested {
		return nil
	}

	partPath := filepath.Join(d.task.Path, fmt.Sprintf("%s.part%d", d.task.TempFileName, part.ID))
	part.FileName = partPath

	// Check existing part file for resume
	actualStart := part.StartPos
	var file *os.File
	var err error

	if info, statErr := os.Stat(partPath); statErr == nil && info.Size() > 0 {
		existingSize := info.Size()
		expectedSize := part.EndPos - part.StartPos + 1

		if existingSize >= expectedSize {
			// Part already fully downloaded
			part.DownloadedBytes = expectedSize
			atomic.AddInt64(&d.task.DownloadedBytes, expectedSize)
			atomic.AddInt32(&d.task.PartsCompleted, 1)
			return nil
		}

		// Resume from where we left off
		actualStart = part.StartPos + existingSize
		part.DownloadedBytes = existingSize
		atomic.AddInt64(&d.task.DownloadedBytes, existingSize)

		file, err = os.OpenFile(partPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(partPath)
	}

	if err != nil {
		return err
	}
	defer utils.CloseQuietly(file)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", d.task.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", d.config.UserAgent)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", actualStart, part.EndPos))

	// Do request
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d for part %d", resp.StatusCode, part.ID)
	}

	// Download part with pause/stop checks
	buf := make([]byte, 32*1024) // 32KB buffer for parts
	for {
		// Check pause/stop
		if d.task.Paused || d.task.StopRequested {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			part.DownloadedBytes += int64(n)
			atomic.AddInt64(&d.task.DownloadedBytes, int64(n))
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	atomic.AddInt32(&d.task.PartsCompleted, 1)
	return nil
}

// mergeParts merges downloaded parts into final file
func (d *downloader) mergeParts() error {
	finalPath := filepath.Join(d.task.Path, d.task.FileName)

	file, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer utils.CloseQuietly(file)

	for _, part := range d.task.Parts {
		if part.FileName == "" {
			continue
		}

		partFile, err := os.Open(part.FileName)
		if err != nil {
			return err
		}

		_, err = io.Copy(file, partFile)
		utils.CloseQuietly(partFile)
		utils.RemoveQuietly(part.FileName)

		if err != nil {
			return err
		}
	}

	return nil
}

// verify verifies the downloaded file
func (d *downloader) verify() error {
	finalPath := filepath.Join(d.task.Path, d.task.FileName)

	info, err := os.Stat(finalPath)
	if err != nil {
		return err
	}

	if d.task.TotalBytes > 0 && info.Size() != d.task.TotalBytes {
		return fmt.Errorf("size mismatch: expected %d, got %d", d.task.TotalBytes, info.Size())
	}

	return nil
}

// updateProgress updates progress information periodically
func (d *downloader) updateProgress(stop <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.calculateSpeed()
			if d.progressFn != nil {
				d.progressFn(d.task)
			}
		case <-stop:
			return
		}
	}
}

// calculateSpeed calculates download speed and ETA
func (d *downloader) calculateSpeed() {
	elapsed := time.Since(d.task.StartedAt)
	if elapsed.Seconds() <= 0 {
		return
	}

	downloaded := atomic.LoadInt64(&d.task.DownloadedBytes)
	d.task.Speed = int64(float64(downloaded) / elapsed.Seconds())

	if d.task.TotalBytes > 0 && d.task.Speed > 0 {
		remaining := d.task.TotalBytes - downloaded
		d.task.ETA = remaining / d.task.Speed
	}
}

// extractFileNameFromURL extracts filename from URL
func extractFileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]

	if idx := strings.Index(filename, "?"); idx > 0 {
		filename = filename[:idx]
	}

	return filename
}

// parseFileName parses filename from Content-Disposition header
func parseFileName(cd string) string {
	if idx := strings.Index(cd, "filename="); idx > 0 {
		filename := cd[idx+9:]
		filename = strings.Trim(filename, `"`)
		return filename
	}
	return ""
}
