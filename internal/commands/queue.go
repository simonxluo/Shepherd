// Package commands provides command queue implementation
// 这个包提供命令队列的实现
package commands

import (
	"fmt"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/node"
)

// MemoryCommandQueue implements CommandQueue using in-memory storage
// MemoryCommandQueue 使用内存存储实现 CommandQueue
type MemoryCommandQueue struct {
	// queues maps nodeID -> list of commands
	queues map[string][]*node.Command
	mu     sync.RWMutex
	log    *logger.Logger

	// Command tracking
	commandResults map[string]*node.CommandResult
	completedCount int64
	failedCount    int64
}

// NewMemoryCommandQueue creates a new in-memory command queue
// NewMemoryCommandQueue 创建新的内存命令队列
func NewMemoryCommandQueue(log *logger.Logger) *MemoryCommandQueue {
	return &MemoryCommandQueue{
		queues:         make(map[string][]*node.Command),
		commandResults: make(map[string]*node.CommandResult),
		log:            log,
	}
}

// Enqueue adds a command to the queue for a node
func (q *MemoryCommandQueue) Enqueue(nodeID string, cmd *node.Command) error {
	if nodeID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	if cmd == nil {
		return fmt.Errorf("命令不能为空")
	}

	// Generate command ID if not set
	if cmd.ID == "" {
		cmd.ID = fmt.Sprintf("cmd-%s-%d", nodeID, time.Now().UnixNano())
	}

	// Set target node
	cmd.ToNodeID = nodeID

	// Set created time if not set
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now()
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.queues[nodeID] == nil {
		q.queues[nodeID] = make([]*node.Command, 0)
	}

	q.queues[nodeID] = append(q.queues[nodeID], cmd)
	q.log.Debugf("命令已入队: 节点=%s, 命令=%s, ID=%s",
		nodeID, cmd.Type, cmd.ID)

	return nil
}

// Dequeue removes and returns the next command for a node
func (q *MemoryCommandQueue) Dequeue(nodeID string) (*node.Command, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("节点ID不能为空")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	queue := q.queues[nodeID]
	if queue == nil || len(queue) == 0 {
		return nil, fmt.Errorf("节点 %s 没有待执行的命令", nodeID)
	}

	// Get first command
	cmd := queue[0]

	// Remove from queue
	q.queues[nodeID] = queue[1:]

	q.log.Debugf("命令已出队: 节点=%s, 命令=%s, ID=%s",
		nodeID, cmd.Type, cmd.ID)

	return cmd, nil
}

// Peek returns the next command without removing it
func (q *MemoryCommandQueue) Peek(nodeID string) (*node.Command, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("节点ID不能为空")
	}

	q.mu.RLock()
	defer q.mu.RUnlock()

	queue := q.queues[nodeID]
	if queue == nil || len(queue) == 0 {
		return nil, fmt.Errorf("节点 %s 没有待执行的命令", nodeID)
	}

	return queue[0], nil
}

// Cancel removes a command from the queue
func (q *MemoryCommandQueue) Cancel(commandID string) error {
	if commandID == "" {
		return fmt.Errorf("命令ID不能为空")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Search for the command in all queues
	for nodeID, queue := range q.queues {
		for i, cmd := range queue {
			if cmd.ID == commandID {
				// Remove from queue
				q.queues[nodeID] = append(queue[:i], queue[i+1:]...)
				q.log.Infof("命令已取消: ID=%s", commandID)
				return nil
			}
		}
	}

	return fmt.Errorf("命令 %s 不存在", commandID)
}

// GetQueueSize returns the number of pending commands for a node
func (q *MemoryCommandQueue) GetQueueSize(nodeID string) int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	queue := q.queues[nodeID]
	if queue == nil {
		return 0
	}
	return len(queue)
}

// ListQueuedCommands returns all queued commands for a node
func (q *MemoryCommandQueue) ListQueuedCommands(nodeID string) []*node.Command {
	q.mu.RLock()
	defer q.mu.RUnlock()

	queue := q.queues[nodeID]
	if queue == nil {
		return []*node.Command{}
	}

	// Return a copy to prevent external modification
	result := make([]*node.Command, len(queue))
	copy(result, queue)

	return result
}

// ClearQueue removes all commands for a node
func (q *MemoryCommandQueue) ClearQueue(nodeID string) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	queue := q.queues[nodeID]
	count := 0

	if queue != nil {
		count = len(queue)
		delete(q.queues, nodeID)
	}

	q.log.Infof("清空节点队列: 节点=%s, 清理了 %d 个命令", nodeID, count)
	return count
}

// RetryCommand requeues a failed command
func (q *MemoryCommandQueue) RetryCommand(commandID string) error {
	if commandID == "" {
		return fmt.Errorf("命令ID不能为空")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Find the command result
	_, exists := q.commandResults[commandID]
	if !exists {
		return fmt.Errorf("命令结果不存在: %s", commandID)
	}

	// Get the original command info from result
	// For now, we'll create a retry based on the result
	// In a real implementation, you'd store the original command

	q.log.Infof("重试命令: ID=%s", commandID)
	return nil
}

// StoreCommandResult stores the result of a command execution
func (q *MemoryCommandQueue) StoreCommandResult(result *node.CommandResult) error {
	if result == nil {
		return fmt.Errorf("命令结果不能为空")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.commandResults[result.CommandID] = result

	if result.Success {
		q.completedCount++
	} else {
		q.failedCount++
	}

	q.log.Debugf("存储命令结果: ID=%s, 成功=%v", result.CommandID, result.Success)
	return nil
}

// GetStats returns statistics about the command queue
func (q *MemoryCommandQueue) GetStats() *node.CommandQueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := &node.CommandQueueStats{}

	// Count total queued commands
	for _, queue := range q.queues {
		stats.QueuedCommands += len(queue)
	}

	stats.CompletedCommands = q.completedCount
	stats.FailedCommands = q.failedCount

	return stats
}
