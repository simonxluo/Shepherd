// Package shutdown provides graceful shutdown functionality for the Shepherd application.
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

// HookPriority defines the order in which hooks are executed
type HookPriority int

const (
	// PriorityCritical hooks run first (e.g., stop accepting new connections)
	PriorityCritical HookPriority = 0
	// PriorityHigh hooks run second (e.g., stop processing)
	PriorityHigh HookPriority = 1
	// PriorityNormal hooks run third (e.g., cleanup resources)
	PriorityNormal HookPriority = 2
	// PriorityLow hooks run last (e.g., flush logs)
	PriorityLow HookPriority = 3
)

// shutdownHook represents a registered shutdown hook with priority
type shutdownHook struct {
	name     string
	hook     func(ctx context.Context) error
	priority HookPriority
}

// Manager manages graceful shutdown
type Manager struct {
	mu          sync.RWMutex
	hooks       []shutdownHook
	timeout     time.Duration
	sigChan     chan os.Signal
	shutdownCtx context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	started     bool
	shutdown    bool
}

// NewManager creates a new shutdown manager
func NewManager(timeout time.Duration) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		hooks:       make([]shutdownHook, 0),
		timeout:     timeout,
		sigChan:     make(chan os.Signal, 1),
		shutdownCtx: ctx,
		cancel:      cancel,
	}
}

// Register registers a new shutdown hook with the given name and priority
func (m *Manager) Register(name string, hook func(ctx context.Context) error, priority HookPriority) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks = append(m.hooks, shutdownHook{
		name:     name,
		hook:     hook,
		priority: priority,
	})

	logger.Debugf("Registered shutdown hook: %s (priority: %d)", name, priority)
}

// Start begins listening for shutdown signals
func (m *Manager) Start() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	// Listen for interrupt signals
	signal.Notify(m.sigChan,
		os.Interrupt,    // Ctrl+C
		syscall.SIGTERM, // kill command
		syscall.SIGQUIT, // quit signal
	)

	m.wg.Add(1)
	go m.waitForShutdown()
}

// waitForShutdown waits for shutdown signals
func (m *Manager) waitForShutdown() {
	defer m.wg.Done()

	select {
	case sig := <-m.sigChan:
		logger.Infof("收到关闭信号: %v", sig)
		m.performShutdown()
	case <-m.shutdownCtx.Done():
		logger.Info("收到上下文取消请求")
		m.performShutdown()
	}
}

// performShutdown executes all shutdown hooks in order
func (m *Manager) performShutdown() {
	m.mu.Lock()
	if m.shutdown {
		m.mu.Unlock()
		return
	}
	m.shutdown = true
	m.mu.Unlock()

	logger.Info("开始优雅关闭...")

	// Sort hooks by priority
	sortedHooks := make([]shutdownHook, len(m.hooks))
	copy(sortedHooks, m.hooks)

	// Simple bubble sort by priority (low number = high priority = run first)
	for i := 0; i < len(sortedHooks)-1; i++ {
		for j := 0; j < len(sortedHooks)-i-1; j++ {
			if sortedHooks[j].priority > sortedHooks[j+1].priority {
				sortedHooks[j], sortedHooks[j+1] = sortedHooks[j+1], sortedHooks[j]
			}
		}
	}

	// Execute each hook with timeout
	for _, hook := range sortedHooks {
		ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
		defer cancel()

		logger.Infof("执行关闭钩子: %s", hook.name)

		done := make(chan error, 1)
		go func() {
			done <- hook.hook(ctx)
		}()

		select {
		case err := <-done:
			if err != nil {
				logger.Errorf("关闭钩子 %s 失败: %v", hook.name, err)
			} else {
				logger.Infof("关闭钩子 %s 完成", hook.name)
			}
		case <-ctx.Done():
			logger.Errorf("关闭钩子 %s 超时 (%v)", hook.name, m.timeout)
		}
	}

	logger.Info("优雅关闭完成")

	// Cancel the shutdown context to signal completion
	m.cancel()
}

// Context returns the shutdown context
func (m *Manager) Context() context.Context {
	return m.shutdownCtx
}

// Done returns a channel that's closed when shutdown is complete
func (m *Manager) Done() <-chan struct{} {
	return m.shutdownCtx.Done()
}

// Wait blocks until shutdown is complete
func (m *Manager) Wait() {
	m.wg.Wait()
}
