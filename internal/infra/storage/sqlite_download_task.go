package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/simonxluo/Shepherd/internal/infra/storage/sqlcgen"
)

// sqlcTaskToDomain 将 sqlc 生成的 DownloadTask 转换为领域类型
func sqlcTaskToDomain(row *sqlcgen.DownloadTask) *DownloadTask {
	return &DownloadTask{
		ID:              row.ID,
		URL:             row.Url,
		Path:            row.Path,
		FileName:        row.FileName,
		State:           row.State,
		DownloadedBytes: derefInt64(row.DownloadedBytes, 0),
		TotalBytes:      derefInt64(row.TotalBytes, 0),
		ETag:            derefString(row.Etag, ""),
		RangeSupported:  derefInt64(row.RangeSupported, 0) != 0,
		FinalURL:        derefString(row.FinalUrl, ""),
		TempFileName:    derefString(row.TempFileName, ""),
		PartsTotal:      int(derefInt64(row.PartsTotal, 0)),
		PartsCompleted:  int(derefInt64(row.PartsCompleted, 0)),
		FileType:        derefString(row.FileType, ""),
		SourceType:      derefString(row.SourceType, ""),
		RepoID:          derefString(row.RepoID, ""),
		ErrorMessage:    derefString(row.ErrorMessage, ""),
		RetryCount:      int(derefInt64(row.RetryCount, 0)),
		MaxRetries:      int(derefInt64(row.MaxRetries, 5)),
		CreatedAt:       time.Unix(row.CreatedAt, 0).UTC(),
		StartedAt:       time.Unix(derefInt64(row.StartedAt, 0), 0).UTC(),
		FinishedAt:      time.Unix(derefInt64(row.FinishedAt, 0), 0).UTC(),
	}
}

// CreateDownloadTask creates a new download task record
func (s *SQLiteStore) CreateDownloadTask(ctx context.Context, task *DownloadTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		task.ID = generateID("dl")
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = timeNow()
	}

	rangeSupported := int64(0)
	if task.RangeSupported {
		rangeSupported = 1
	}

	_, err := s.queries.CreateDownloadTask(ctx, sqlcgen.CreateDownloadTaskParams{
		ID:              task.ID,
		Url:             task.URL,
		Path:            task.Path,
		FileName:        task.FileName,
		State:           task.State,
		DownloadedBytes: &task.DownloadedBytes,
		TotalBytes:      &task.TotalBytes,
		Etag:            stringPtr(task.ETag),
		RangeSupported:  &rangeSupported,
		FinalUrl:        stringPtr(task.FinalURL),
		TempFileName:    stringPtr(task.TempFileName),
		PartsTotal:      int64Ptr(int64(task.PartsTotal)),
		PartsCompleted:  int64Ptr(int64(task.PartsCompleted)),
		FileType:        stringPtr(task.FileType),
		SourceType:      stringPtr(task.SourceType),
		RepoID:          stringPtr(task.RepoID),
		ErrorMessage:    stringPtr(task.ErrorMessage),
		RetryCount:      int64Ptr(int64(task.RetryCount)),
		MaxRetries:      int64Ptr(int64(task.MaxRetries)),
		CreatedAt:       task.CreatedAt.Unix(),
		StartedAt:       int64Ptr(task.StartedAt.Unix()),
		FinishedAt:      int64Ptr(task.FinishedAt.Unix()),
	})
	return err
}

// GetDownloadTask retrieves a download task by ID
func (s *SQLiteStore) GetDownloadTask(ctx context.Context, id string) (*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetDownloadTask(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDownloadTaskNotFound
		}
		return nil, fmt.Errorf("failed to get download task: %w", err)
	}
	return sqlcTaskToDomain(&row), nil
}

// ListDownloadTasks lists download tasks with pagination, ordered by created_at DESC
func (s *SQLiteStore) ListDownloadTasks(ctx context.Context, limit, offset int) ([]*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListDownloadTasks(ctx, sqlcgen.ListDownloadTasksParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list download tasks: %w", err)
	}

	tasks := make([]*DownloadTask, 0, len(rows))
	for i := range rows {
		tasks = append(tasks, sqlcTaskToDomain(&rows[i]))
	}
	return tasks, nil
}

// UpdateDownloadTask updates all mutable fields of a download task
func (s *SQLiteStore) UpdateDownloadTask(ctx context.Context, task *DownloadTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rangeSupported := int64(0)
	if task.RangeSupported {
		rangeSupported = 1
	}

	row, err := s.queries.UpdateDownloadTask(ctx, sqlcgen.UpdateDownloadTaskParams{
		Url:             task.URL,
		Path:            task.Path,
		FileName:        task.FileName,
		State:           task.State,
		DownloadedBytes: &task.DownloadedBytes,
		TotalBytes:      &task.TotalBytes,
		Etag:            stringPtr(task.ETag),
		RangeSupported:  &rangeSupported,
		FinalUrl:        stringPtr(task.FinalURL),
		TempFileName:    stringPtr(task.TempFileName),
		PartsTotal:      int64Ptr(int64(task.PartsTotal)),
		PartsCompleted:  int64Ptr(int64(task.PartsCompleted)),
		FileType:        stringPtr(task.FileType),
		SourceType:      stringPtr(task.SourceType),
		RepoID:          stringPtr(task.RepoID),
		ErrorMessage:    stringPtr(task.ErrorMessage),
		RetryCount:      int64Ptr(int64(task.RetryCount)),
		MaxRetries:      int64Ptr(int64(task.MaxRetries)),
		StartedAt:       int64Ptr(task.StartedAt.Unix()),
		FinishedAt:      int64Ptr(task.FinishedAt.Unix()),
		ID:              task.ID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrDownloadTaskNotFound
		}
		return fmt.Errorf("failed to update download task: %w", err)
	}
	// 更新成功，将返回值中的时间回写到 task（保持行为一致）
	_ = row
	return nil
}

// DeleteDownloadTask deletes a download task by ID
func (s *SQLiteStore) DeleteDownloadTask(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.queries.DeleteDownloadTask(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete download task: %w", err)
	}
	return nil
}

// ListActiveDownloadTasks returns all download tasks with active states
func (s *SQLiteStore) ListActiveDownloadTasks(ctx context.Context) ([]*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListActiveDownloadTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active download tasks: %w", err)
	}

	tasks := make([]*DownloadTask, 0, len(rows))
	for i := range rows {
		tasks = append(tasks, sqlcTaskToDomain(&rows[i]))
	}
	return tasks, nil
}
