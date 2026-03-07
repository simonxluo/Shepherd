package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// downloader handles the actual download logic
type downloader struct {
	config    DownloadConfig
	task      *Task
	client    *http.Client
	manager   *Manager
}

// newDownloader creates a new downloader
func newDownloader(config DownloadConfig, task *Task) *downloader {
	return &downloader{
		config: config,
		task:   task,
		client: &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Track final URL after redirects
				if len(via) > 0 {
					task.FinalURL = req.URL.String()
				}
				return nil
			},
		},
	}
}

// Download executes the download
func (d *downloader) Download(ctx context.Context) error {
	// Check if paused
	if d.task.Paused {
		return nil
	}

	// Prepare download
	if err := d.prepare(ctx); err != nil {
		return fmt.Errorf("prepare failed: %w", err)
	}

	// Check if paused
	if d.task.Paused {
		return nil
	}

	// Execute download
	if d.task.RangeSupported && d.task.TotalBytes > d.config.MinPartSize {
		// Parallel download
		if err := d.downloadParallel(ctx); err != nil {
			return err
		}
	} else {
		// Simple download
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

	// Create HEAD request to check file
	req, err := http.NewRequestWithContext(ctx, "HEAD", d.task.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", d.config.UserAgent)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer closeQuietly(resp.Body)

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
		// Try to get from Content-Disposition
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

	// Determine start position
	startPos := d.task.DownloadedBytes

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
	defer closeQuietly(resp.Body)

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Update total bytes if not set
	if d.task.TotalBytes == 0 {
		d.task.TotalBytes = resp.ContentLength + startPos
	}

	// Create temp file
	tempPath := filepath.Join(d.task.Path, d.task.TempFileName)

	var file *os.File
	if startPos > 0 {
		// Open existing file for append
		file, err = os.OpenFile(tempPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		// Create new file
		file, err = os.Create(tempPath)
	}

	if err != nil {
		return err
	}
	defer closeQuietly(file)

	// Start progress updater
	stopProgress := make(chan struct{})
	go d.updateProgress()

	defer close(stopProgress)

	// Download with progress tracking
	buf := make([]byte, d.config.ChunkSize)
	lastUpdate := time.Now()

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

			d.task.DownloadedBytes += int64(n)

			// Update speed periodically
			if time.Since(lastUpdate) > time.Second {
				d.calculateSpeed()
				lastUpdate = time.Now()
			}
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
	var parts []PartDownload
	numParts := int(d.task.TotalBytes / partSize)
	if int(d.task.TotalBytes)%int(partSize) != 0 {
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

		part := PartDownload{
			ID:       i,
			StartPos: startPos,
			EndPos:   endPos,
		}
		parts = append(parts, part)
	}

	d.task.Parts = parts

	// Download parts concurrently
	errChan := make(chan error, len(parts))

	for i := range parts {
		go func(part *PartDownload) {
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

	if firstError != nil {
		return firstError
	}

	// Merge parts
	d.task.State = StateMerging
	if err := d.mergeParts(); err != nil {
		return err
	}

	return nil
}

// downloadPart downloads a single part
func (d *downloader) downloadPart(ctx context.Context, part *PartDownload) error {
	// Check if paused
	if d.task.Paused || d.task.StopRequested {
		return nil
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", d.task.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", d.config.UserAgent)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", part.StartPos, part.EndPos))

	// Do request
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer closeQuietly(resp.Body)

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create temp file for part
	partPath := filepath.Join(d.task.Path, fmt.Sprintf("%s.part%d", d.task.TempFileName, part.ID))
	file, err := os.Create(partPath)
	if err != nil {
		return err
	}
	defer closeQuietly(file)

	// Download part
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	part.FileName = partPath
	part.DownloadedBytes = part.EndPos - part.StartPos + 1

	d.task.PartsCompleted++
	d.task.DownloadedBytes += part.DownloadedBytes

	return nil
}

// mergeParts merges downloaded parts into final file
func (d *downloader) mergeParts() error {
	finalPath := filepath.Join(d.task.Path, d.task.FileName)

	file, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer closeQuietly(file)

	for _, part := range d.task.Parts {
		if part.FileName == "" {
			continue
		}

		partFile, err := os.Open(part.FileName)
		if err != nil {
			return err
		}
		defer closeQuietly(partFile)

		_, err = io.Copy(file, partFile)
		removeQuietly(part.FileName)

		if err != nil {
			return err
		}
	}

	return nil
}

// verify verifies the downloaded file
func (d *downloader) verify() error {
	// Basic check: file exists and has correct size
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

// updateProgress updates progress information
func (d *downloader) updateProgress() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	lastBytes := d.task.DownloadedBytes

	for {
		select {
		case <-ticker.C:
			currentBytes := d.task.DownloadedBytes
			if currentBytes > lastBytes {
				d.calculateSpeed()
				lastBytes = currentBytes
			}
		}
	}
}

// calculateSpeed calculates download speed and ETA
func (d *downloader) calculateSpeed() {
	elapsed := time.Since(d.task.StartedAt)
	if elapsed.Seconds() <= 0 {
		return
	}

	d.task.Speed = int64(float64(d.task.DownloadedBytes) / elapsed.Seconds())

	if d.task.TotalBytes > 0 && d.task.Speed > 0 {
		remaining := d.task.TotalBytes - d.task.DownloadedBytes
		d.task.ETA = remaining / d.task.Speed
	}

	// Notify progress
	d.manager.notifyProgress(Progress{
		TaskID:          d.task.ID,
		State:           d.task.State,
		DownloadedBytes: d.task.DownloadedBytes,
		TotalBytes:      d.task.TotalBytes,
		Speed:           d.task.Speed,
		ETA:             d.task.ETA,
		PartsTotal:      d.task.PartsTotal,
		PartsCompleted:  d.task.PartsCompleted,
	})
}

// extractFileNameFromURL extracts filename from URL
func extractFileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]

	// Remove query parameters
	if idx := strings.Index(filename, "?"); idx > 0 {
		filename = filename[:idx]
	}

	return filename
}

// parseFileName parses filename from Content-Disposition header
func parseFileName(cd string) string {
	// Try to extract filename from Content-Disposition header
	// Format: attachment; filename="file.txt"
	if idx := strings.Index(cd, "filename="); idx > 0 {
		filename := cd[idx+9:]
		filename = strings.Trim(filename, `"`)
		return filename
	}
	return ""
}
