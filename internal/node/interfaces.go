// Package node provides interface definitions for node components
// 这个包提供节点组件的接口定义，实现职责分离
package node

import (
	"context"
	"time"
)

// INode represents the core node interface - only manages its own state
// INode 接口 - 只管理自身状态（使用 INode 前缀避免与 Node 结构体冲突）
type INode interface {
	// Identity information - 身份信息
	ID() string
	Name() string
	Role() NodeRole
	Status() NodeStatus
	Address() string
	Port() int

	// Lifecycle management - 生命周期管理
	Start() error
	Stop() error
	IsRunning() bool

	// Health check - 健康检查
	Health() *HealthStatus

	// Configuration - 配置
	GetConfig() *NodeConfig
	UpdateConfig(*NodeConfig) error

	// Capabilities and resources - 能力和资源
	GetCapabilities() *NodeCapabilities
	GetResources() *NodeResources

	// Context - 上下文
	Context() context.Context
}

// ClientRegistry manages client registration and discovery
// ClientRegistry 接口 - 管理客户端注册和发现
type ClientRegistry interface {
	// Register a new client - 注册新客户端
	Register(info *NodeInfo) error

	// Unregister a client - 注销客户端
	Unregister(nodeID string) error

	// Get client by ID - 根据 ID 获取客户端
	Get(nodeID string) (*NodeInfo, error)

	// List all clients - 列出所有客户端
	List() []*NodeInfo

	// Get statistics - 获取统计信息
	GetStats() *RegistryStats

	// Find clients by criteria - 根据条件查找客户端
	Find(predicate func(*NodeInfo) bool) []*NodeInfo

	// Update client status - 更新客户端状态
	UpdateStatus(nodeID string, status NodeStatus) error

	// Update client resources - 更新客户端资源
	UpdateResources(nodeID string, resources *NodeResources) error

	// Get online clients - 获取在线客户端
	GetOnlineClients() []*NodeInfo

	// Cleanup offline clients - 清理离线客户端
	Cleanup(timeout time.Duration) int
}

// CommandQueue manages command queuing and distribution
// CommandQueue 接口 - 管理命令队列和分发
type CommandQueue interface {
	// Enqueue a command for a node - 将命令加入队列
	Enqueue(nodeID string, cmd *Command) error

	// Dequeue gets the next command for a node - 获取节点的下一个命令
	Dequeue(nodeID string) (*Command, error)

	// Peek returns the next command without removing it - 查看下一个命令
	Peek(nodeID string) (*Command, error)

	// Cancel a command - 取消命令
	Cancel(commandID string) error

	// GetQueueSize returns the queue size for a node - 获取队列大小
	GetQueueSize(nodeID string) int

	// ListQueuedCommands returns all queued commands for a node - 列出队列中的命令
	ListQueuedCommands(nodeID string) []*Command

	// ClearQueue removes all commands for a node - 清空队列
	ClearQueue(nodeID string) int

	// RetryCommand requeues a failed command - 重试失败的命令
	RetryCommand(commandID string) error
}

// IResourceMonitor monitors node resource usage
// IResourceMonitor 接口 - 监控节点资源使用（使用 I 前缀避免与结构体冲突）
type IResourceMonitor interface {
	// Start begins monitoring - 开始监控
	Start() error

	// Stop stops monitoring - 停止监控
	Stop() error

	// GetResources returns current resource usage - 获取当前资源使用情况
	GetResources() *NodeResources

	// GetSnapshot returns a snapshot of current resources (alias for GetResources)
	GetSnapshot() *NodeResources

	// Watch registers a callback for resource updates - 注册资源更新回调
	Watch(callback func(*NodeResources))

	// SetUpdateInterval sets the monitoring interval - 设置监控间隔
	SetUpdateInterval(interval time.Duration)

	// GetMetrics returns historical metrics - 获取历史指标
	GetMetrics() *NodeMetrics

	// GetGPUInfo returns GPU information
	GetGPUInfo() []GPUInfo

	// GetLlamacppInfo returns llama.cpp information
	GetLlamacppInfo() *LlamacppInfo
}

// RegistryStats contains statistics about the client registry
// 注册表统计信息
type RegistryStats struct {
	TotalClients   int `json:"totalClients"`
	OnlineClients  int `json:"onlineClients"`
	OfflineClients int `json:"offlineClients"`
	BusyClients    int `json:"busyClients"`
	ErrorClients   int `json:"errorClients"`
}

// CommandQueueStats contains statistics about the command queue
// 命令队列统计信息
type CommandQueueStats struct {
	TotalCommands     int   `json:"totalCommands"`
	QueuedCommands    int   `json:"queuedCommands"`
	RunningCommands   int   `json:"runningCommands"`
	CompletedCommands int64 `json:"completedCommands"`
	FailedCommands    int64 `json:"failedCommands"`
}
