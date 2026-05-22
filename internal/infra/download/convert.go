package download

import (
	"fmt"

	"github.com/simonxluo/Shepherd/internal/infra/storage"
)

// taskToStorageModel converts an in-memory Task to a storage DownloadTask
func taskToStorageModel(t *Task) *storage.DownloadTask {
	var errMsg string
	if t.Error != nil {
		errMsg = t.Error.Error()
	}
	return &storage.DownloadTask{
		ID:              t.ID,
		URL:             t.URL,
		Path:            t.Path,
		FileName:        t.FileName,
		State:           t.State.String(),
		DownloadedBytes: t.DownloadedBytes,
		TotalBytes:      t.TotalBytes,
		ETag:            t.ETag,
		RangeSupported:  t.RangeSupported,
		FinalURL:        t.FinalURL,
		TempFileName:    t.TempFileName,
		PartsTotal:      t.PartsTotal,
		PartsCompleted:  int(t.PartsCompleted),
		FileType:        t.FileType,
		SourceType:      t.SourceType,
		RepoID:         t.RepoID,
		ErrorMessage:    errMsg,
		RetryCount:      t.RetryCount,
		MaxRetries:      t.MaxRetries,
		CreatedAt:       t.CreatedAt,
		StartedAt:       t.StartedAt,
		FinishedAt:      t.FinishedAt,
	}
}

// storageModelToTask converts a storage DownloadTask back to an in-memory Task
func storageModelToTask(dt *storage.DownloadTask) *Task {
	var state DownloadState
	switch dt.State {
	case "idle":
		state = StateIdle
	case "preparing":
		state = StatePreparing
	case "downloading":
		state = StateDownloading
	case "merging":
		state = StateMerging
	case "verifying":
		state = StateVerifying
	case "completed":
		state = StateCompleted
	case "failed":
		state = StateFailed
	case "paused":
		state = StatePaused
	default:
		state = StateIdle
	}

	var taskErr error
	if dt.ErrorMessage != "" {
		taskErr = fmt.Errorf("%s", dt.ErrorMessage)
	}

	return &Task{
		ID:              dt.ID,
		URL:             dt.URL,
		Path:            dt.Path,
		FileName:        dt.FileName,
		State:           state,
		DownloadedBytes: dt.DownloadedBytes,
		TotalBytes:      dt.TotalBytes,
		ETag:            dt.ETag,
		RangeSupported:  dt.RangeSupported,
		FinalURL:        dt.FinalURL,
		TempFileName:    dt.TempFileName,
		PartsTotal:      dt.PartsTotal,
		PartsCompleted:  int32(dt.PartsCompleted),
		FileType:        dt.FileType,
		SourceType:      dt.SourceType,
		RepoID:         dt.RepoID,
		Error:           taskErr,
		RetryCount:      dt.RetryCount,
		MaxRetries:      dt.MaxRetries,
		CreatedAt:       dt.CreatedAt,
		StartedAt:       dt.StartedAt,
		FinishedAt:      dt.FinishedAt,
	}
}
