package download

import "time"

type DownloadState int

const (
	StateIdle DownloadState = iota
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

type Task struct {
	ID       string
	URL      string
	Path     string
	FileName string
	State    DownloadState

	DownloadedBytes int64
	TotalBytes      int64
	Speed           int64
	ETA             int64

	ETag           string
	RangeSupported bool
	FinalURL       string
	TempFileName   string

	Parts          []partDownload
	PartsTotal     int
	PartsCompleted int

	CreatedAt  time.Time
	StartedAt  time.Time
	FinishedAt time.Time

	Error      error
	RetryCount int
	MaxRetries int

	Paused        bool
	StopRequested bool

	FileType   string
	SourceType string
	RepoID     string
}

type DownloadConfig struct {
	MaxConcurrent  int
	ChunkSize      int64
	Timeout        time.Duration
	RetryCount     int
	MinPartSize    int64
	MaxParallelism int
	UserAgent      string
}
