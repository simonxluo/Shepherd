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

