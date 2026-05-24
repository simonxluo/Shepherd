package taskmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// OnTaskUpdate 任务状态变更回调类型
type OnTaskUpdate func(task *Task)

// Manager 一次性任务管理器
type Manager struct {
	tasks        map[string]*Task
	mu           sync.RWMutex
	ctx          context.Context
	cancelFunc   context.CancelFunc
	onTaskUpdate OnTaskUpdate
}

// NewManager 创建任务管理器
func NewManager(onTaskUpdate OnTaskUpdate) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		tasks:        make(map[string]*Task),
		ctx:          ctx,
		cancelFunc:   cancel,
		onTaskUpdate: onTaskUpdate,
	}
}

// Register 注册一个新任务到管理器
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

// Get 获取任务
func (m *Manager) Get(taskID string) (*Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[taskID]
	return t, ok
}

// List 按类型列出任务
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

// ListByModel 按模型列出任务
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

// Cancel 取消任务
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

// Remove 移除已完成的任务
func (m *Manager) Remove(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tasks, taskID)
}

// RunningCount 获取指定类型的运行中任务数
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

// Context 返回管理器的 context
func (m *Manager) Context() context.Context {
	return m.ctx
}

// Shutdown 关闭管理器，取消所有运行中的任务并等待完成
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
