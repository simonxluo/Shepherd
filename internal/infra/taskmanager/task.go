package taskmanager

import (
	"context"
	"sync"
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskType 任务类型
type TaskType string

const (
	TaskTypeBenchmark TaskType = "benchmark"
)

// Task 一次性任务信息
type Task struct {
	ID         string                 `json:"id"`
	Type       TaskType               `json:"type"`
	Status     TaskStatus             `json:"status"`
	Name       string                 `json:"name"`
	ModelID    string                 `json:"modelId,omitempty"`
	ModelName  string                 `json:"modelName,omitempty"`
	Command    string                 `json:"command,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Progress   float64                `json:"progress,omitempty"`
	Metrics    map[string]interface{} `json:"metrics,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	StartedAt  *time.Time             `json:"startedAt,omitempty"`
	FinishedAt *time.Time             `json:"finishedAt,omitempty"`

	// 内部字段（不序列化）
	ctx      context.Context
	cancel   context.CancelFunc
	onUpdate func(*Task)
	mu       sync.RWMutex
}

func (t *Task) Context() context.Context {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.ctx
}

func (t *Task) GetStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

func (t *Task) SetStatus(status TaskStatus) {
	t.mu.Lock()
	t.Status = status
	t.mu.Unlock()
	if t.onUpdate != nil {
		t.onUpdate(t)
	}
}

func (t *Task) Cancel() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *Task) SetOnUpdate(fn func(*Task)) {
	t.mu.Lock()
	t.onUpdate = fn
	t.mu.Unlock()
}

// ToMap 转为 map 用于 JSON 序列化（排除内部字段）
func (t *Task) ToMap() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	m := map[string]interface{}{
		"id":        t.ID,
		"type":      t.Type,
		"status":    t.Status,
		"name":      t.Name,
		"createdAt": t.CreatedAt,
	}
	if t.ModelID != "" {
		m["modelId"] = t.ModelID
	}
	if t.ModelName != "" {
		m["modelName"] = t.ModelName
	}
	if t.Command != "" {
		m["command"] = t.Command
	}
	if t.Error != "" {
		m["error"] = t.Error
	}
	if t.Progress > 0 {
		m["progress"] = t.Progress
	}
	if t.Metrics != nil {
		m["metrics"] = t.Metrics
	}
	if t.StartedAt != nil {
		m["startedAt"] = *t.StartedAt
	}
	if t.FinishedAt != nil {
		m["finishedAt"] = *t.FinishedAt
	}
	return m
}
