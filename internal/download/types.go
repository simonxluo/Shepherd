// Package download provides file download management with resume support.
// It handles concurrent downloads, progress tracking, and pause/resume functionality.
package download

import "time"

// DownloadState represents the current state of a download task
type DownloadState int

const (
	StateIdle       DownloadState = iota
	StatePreparing
	StateDownloading
	StateMerging
	StateVerifying
	StateCompleted
	StateFailed
	StatePaused
)

func (s DownloadState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StatePreparing:
		return "preparing"
	case StateDownloading:
		return "downloading"
	case StateMerging:
		return "merging"
	case StateVerifying:
		return "verifying"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StatePaused:
		return "paused"
	default:
		return "unknown"
	}
}

// Task represents a download task
type Task struct {
	ID              string
	URL             string
	Path            string
	FileName        string
	State           DownloadState

	// Progress tracking
	DownloadedBytes int64
	TotalBytes      int64
	Speed           int64    // bytes per second
	ETA             int64    // seconds remaining

	// Resume support
	ETag            string
	RangeSupported  bool
	FinalURL        string
	TempFileName    string

	// Part downloads (for parallel downloads)
	Parts           []PartDownload
	PartsTotal      int
	PartsCompleted  int

	// Timing
	CreatedAt       time.Time
	StartedAt       time.Time
	FinishedAt      time.Time

	// Error handling
	Error           error
	RetryCount      int
	MaxRetries      int

	// Control
	Paused          bool
	StopRequested   bool

	// Metadata
	FileType        string // "gguf", "json", etc.
	SourceType      string // "huggingface", "modelscope", etc.
	RepoID          string // "Qwen/Qwen2-7B-Instruct"
}

// PartDownload represents a part of a parallel download
type PartDownload struct {
	ID              int
	StartPos        int64
	EndPos          int64
	DownloadedBytes int64
	FileName        string
}

// Progress represents the current download progress
type Progress struct {
	TaskID          string
	State           DownloadState
	DownloadedBytes int64
	TotalBytes      int64
	Speed           int64
	ETA             int64
	PartsTotal      int
	PartsCompleted  int
}

// ProgressListener is a callback for download progress updates
type ProgressListener func(progress Progress)

// DownloadConfig contains configuration for downloads
type DownloadConfig struct {
	MaxConcurrent   int           // Maximum concurrent downloads
	ChunkSize       int64         // Size of each download chunk
	Timeout         time.Duration // Request timeout
	RetryCount      int           // Maximum retry attempts
	MinPartSize     int64         // Minimum size for parallel download
	MaxParallelism  int           // Maximum parallel parts
	UserAgent       string
}
