package taskmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// OnTaskUpdate is a callback type invoked when a task's status changes.
type OnTaskUpdate func(task *Task)

// Manager is a one-shot task manager for tracking concurrent background tasks.
type Manager struct {
	tasks        map[string]*Task
	mu           sync.RWMutex
	ctx          context.Context
	cancelFunc   context.CancelFunc
	onTaskUpdate OnTaskUpdate
}

// NewManager creates a new task manager with an optional status-change callback.
func NewManager(onTaskUpdate OnTaskUpdate) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		tasks:        make(map[string]*Task),
		ctx:          ctx,
		cancelFunc:   cancel,
		onTaskUpdate: onTaskUpdate,
	}
}

// Register registers a new task with the manager, injecting context for cancellation.
func (m *Manager) Register(task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	taskCtx, taskCancel := context.WithCancel(m.ctx)
	task.ctx = taskCtx
	task.cancel = taskCancel

	task.SetOnUpdate(func(t *Task) {
		if m.onTaskUpdate != nil {
			m.onTaskUpdate(t)
		}
	})

	m.tasks[task.ID] = task
	return nil
}

// Get retrieves a task by ID.
func (m *Manager) Get(taskID string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[taskID]
	return t, ok
}

// List returns all tasks of the specified type.
func (m *Manager) List(taskType TaskType) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Task
	for _, t := range m.tasks {
		if t.Type == taskType {
			result = append(result, t)
		}
	}
	return result
}

// ListByModel returns tasks of the specified type for a given model.
func (m *Manager) ListByModel(taskType TaskType, modelID string) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Task
	for _, t := range m.tasks {
		if t.Type == taskType && t.ModelID == modelID {
			result = append(result, t)
		}
	}
	return result
}

// Cancel cancels a task by ID.
func (m *Manager) Cancel(taskID string) error {
	m.mu.RLock()
	t, ok := m.tasks[taskID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}

	t.Cancel()
	return nil
}

// Remove removes a completed task from the manager.
func (m *Manager) Remove(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, taskID)
}

// RunningCount returns the number of running tasks of the specified type.
func (m *Manager) RunningCount(taskType TaskType) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, t := range m.tasks {
		if t.Type == taskType && t.GetStatus() == TaskStatusRunning {
			count++
		}
	}
	return count
}

// Context returns the manager's context.
func (m *Manager) Context() context.Context {
	return m.ctx
}

// Shutdown shuts down the manager, cancels all running tasks and waits for completion.
func (m *Manager) Shutdown() {
	logger.Info("正在关闭任务管理器...")

	m.cancelFunc()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.mu.RLock()
			running := 0
			for _, t := range m.tasks {
				if t.GetStatus() == TaskStatusRunning {
					running++
				}
			}
			m.mu.RUnlock()
			logger.Warnf("任务管理器关闭超时，仍有 %d 个任务运行中", running)
			return
		case <-ticker.C:
			m.mu.RLock()
			running := 0
			for _, t := range m.tasks {
				if t.GetStatus() == TaskStatusRunning {
					running++
				}
			}
			m.mu.RUnlock()
			if running == 0 {
				logger.Info("任务管理器已关闭，所有任务已完成")
				return
			}
		}
	}
}
