// Package node provides distributed node management implementation.
package node

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/mem"

	"github.com/simonxluo/Shepherd/internal/comm/gpu"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// Node represents a distributed node in the Shepherd system
type Node struct {
	// Basic information
	id       string
	name     string
	role     NodeRole
	status   NodeStatus
	address  string
	port     int
	version  string
	tags     []string
	metadata map[string]string

	// Capabilities and resources
	capabilities *NodeCapabilities
	resources    *NodeResources
	config       *NodeConfig

	// Runtime state
	createdAt time.Time
	updatedAt time.Time
	lastSeen  time.Time
	startedAt *time.Time
	stoppedAt *time.Time

	// Concurrency control
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc

	// Subsystems
	resource         *ResourceMonitor
	subsystemManager *SubsystemManager

	// Client registry (for Master role)
	clientRegistry *clientRegistry
	commandQueue   *commandQueue
	commandResults *commandResultStore
}

// NewNode creates a new Node instance
func NewNode(config *NodeConfig) (*Node, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	if config.ID == "" {
		return nil, fmt.Errorf("节点ID不能为空")
	}

	if config.Role == "" {
		config.Role = NodeRoleHybrid
	}

	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		id:        config.ID,
		name:      config.Name,
		role:      config.Role,
		status:    NodeStatusOffline,
		address:   config.Address,
		port:      config.Port,
		version:   config.Version,
		tags:      config.Tags,
		metadata:  config.Metadata,
		config:    config,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 确保 tags 和 metadata 不为 nil
	if node.tags == nil {
		node.tags = make([]string, 0)
	}
	if node.metadata == nil {
		node.metadata = make(map[string]string)
	}

	// 检测系统资源
	cpuCount := runtime.NumCPU()
	var memoryTotal int64 = 1024 * 1024 * 1024 // 1GB default
	if vmStat, err := mem.VirtualMemory(); err == nil {
		memoryTotal = int64(vmStat.Total)
	}

	// Initialize capabilities with detected system resources and config
	capabilities := &NodeCapabilities{
		SupportsLlama:  true,
		SupportsPython: config.Capabilities != nil && config.Capabilities.SupportsPython,
		GPU:            false,
		CPUCount:       cpuCount,
		Memory:         memoryTotal,
	}

	// 如果配置中有 Capabilities，复制其字段
	if config.Capabilities != nil {
		capabilities.PythonVersion = config.Capabilities.PythonVersion
		capabilities.CondaPath = config.Capabilities.CondaPath
		capabilities.CondaEnvironments = config.Capabilities.CondaEnvironments
	}

	node.capabilities = capabilities

	// Initialize resources with detected system resources
	node.resources = &NodeResources{
		CPUTotal:    int64(cpuCount) * 1000, // Convert to millicores
		MemoryTotal: memoryTotal,
		DiskTotal:   10 * 1024 * 1024 * 1024, // 10GB (will be updated by resource monitor)
		GPUInfo:     make([]gpu.Info, 0),
		LoadAverage: make([]float64, 3),
	}

	return node, nil
}

// transitionTo transitions the node to a new status with validation
func (n *Node) transitionTo(newStatus NodeStatus) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !isValidNodeTransition(n.status, newStatus) {
		return fmt.Errorf("invalid node state transition: %s -> %s", n.status, newStatus)
	}
	n.status = newStatus
	n.updatedAt = time.Now()
	return nil
}

// setStatusLocked transitions node status without acquiring the lock.
// Caller must hold n.mu write lock.
func (n *Node) setStatusLocked(newStatus NodeStatus) error {
	if !isValidNodeTransition(n.status, newStatus) {
		return fmt.Errorf("invalid node state transition: %s -> %s", n.status, newStatus)
	}
	n.status = newStatus
	n.updatedAt = time.Now()
	return nil
}

// Start starts the node and initializes all subsystems.
func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return fmt.Errorf("节点已经在运行")
	}

	n.status = NodeStatusOnline
	n.running = true
	now := time.Now()
	n.startedAt = &now
	n.lastSeen = now

	// Initialize subsystems
	if err := n.initSubsystems(); err != nil {
		n.setStatusLocked(NodeStatusError)
		n.running = false
		return fmt.Errorf("初始化子系统失败: %w", err)
	}

	// Start subsystems
	if err := n.startSubsystems(); err != nil {
		n.setStatusLocked(NodeStatusError)
		n.running = false
		return fmt.Errorf("启动子系统失败: %w", err)
	}

	return nil
}

// Stop stops the node and shuts down all subsystems.
func (n *Node) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running {
		return nil
	}

	n.status = NodeStatusOffline
	n.running = false
	now := time.Now()
	n.stoppedAt = &now

	// Stop subsystems
	n.stopSubsystems()

	// Cancel context
	n.cancel()

	return nil
}

// ID returns the node identifier
func (n *Node) ID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.id
}

// GetID returns the node ID (backward-compatible wrapper).
func (n *Node) GetID() string {
	return n.ID()
}

// Name returns the node name
func (n *Node) Name() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.name
}

// Role returns the node role
func (n *Node) Role() NodeRole {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.role
}

// Status returns the node status
func (n *Node) Status() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}

// Address returns the node address
func (n *Node) Address() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.address
}

// Port returns the node port
func (n *Node) Port() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.port
}

// Health returns the health status
func (n *Node) Health() *HealthStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()

	status := &HealthStatus{
		NodeID:   n.id,
		Status:   n.status,
		Healthy:  n.status == NodeStatusOnline,
		LastSeen: n.lastSeen,
		Checks:   make(map[string]bool),
		Issues:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// 基本检查
	status.Checks["running"] = n.running
	status.Checks["context_valid"] = n.ctx.Err() == nil

	// 资源检查
	if n.resources != nil {
		// 检查内存使用率
		if n.resources.MemoryTotal > 0 {
			memPercent := float64(n.resources.MemoryUsed) / float64(n.resources.MemoryTotal) * 100
			if memPercent > 95 {
				status.Issues = append(status.Issues, "内存使用率过高")
				status.Healthy = false
			} else if memPercent > 80 {
				status.Warnings = append(status.Warnings, "内存使用率较高")
			}
		}

		// 检查磁盘使用率
		if n.resources.DiskTotal > 0 {
			diskPercent := float64(n.resources.DiskUsed) / float64(n.resources.DiskTotal) * 100
			if diskPercent > 95 {
				status.Issues = append(status.Issues, "磁盘使用率过高")
				status.Healthy = false
			} else if diskPercent > 80 {
				status.Warnings = append(status.Warnings, "磁盘使用率较高")
			}
		}
	}

	return status
}

// UpdateConfig updates the node configuration
func (n *Node) UpdateConfig(config *NodeConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return fmt.Errorf("节点运行时不能更新配置")
	}

	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	// 更新配置
	n.config = config
	n.name = config.Name
	n.address = config.Address
	n.port = config.Port
	n.updatedAt = time.Now()

	return nil
}

// GetName returns the node name.
func (n *Node) GetName() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.name
}

// GetRole returns the node role.
func (n *Node) GetRole() NodeRole {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.role
}

// GetStatus returns the node status.
func (n *Node) GetStatus() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}

// GetAddress returns the node address.
func (n *Node) GetAddress() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.address
}

// GetPort returns the node port.
func (n *Node) GetPort() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.port
}

// GetVersion returns the node version.
func (n *Node) GetVersion() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.version
}

// GetTags returns the node tags.
func (n *Node) GetTags() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]string{}, n.tags...)
}

// GetMetadata returns the node metadata.
func (n *Node) GetMetadata() map[string]string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	metadata := make(map[string]string)
	for k, v := range n.metadata {
		metadata[k] = v
	}
	return metadata
}

// GetCapabilities returns the node capabilities.
func (n *Node) GetCapabilities() *NodeCapabilities {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.capabilities == nil {
		return nil
	}

	// Return a copy
	cap := *n.capabilities
	return &cap
}

// GetResources returns the node resource information.
func (n *Node) GetResources() *NodeResources {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.resources == nil {
		return nil
	}

	// Return a copy
	res := *n.resources
	return &res
}

// SetResources sets the node resource information.
func (n *Node) SetResources(resources *NodeResources) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if resources == nil {
		n.resources = nil
	} else {
		// Create a copy
		n.resources = &NodeResources{}
		*n.resources = *resources
	}
	n.updatedAt = time.Now()
}

// GetConfig returns the node configuration.
func (n *Node) GetConfig() *NodeConfig {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.config == nil {
		return nil
	}

	// Return a copy
	cfg := *n.config
	return &cfg
}

// IsRunning checks whether the node is currently running.
func (n *Node) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.running
}

// GetCreatedAt returns the node creation time.
func (n *Node) GetCreatedAt() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.createdAt
}

// GetUpdatedAt returns the last update time of the node.
func (n *Node) GetUpdatedAt() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.updatedAt
}

// GetLastSeen returns the last time the node was seen active.
func (n *Node) GetLastSeen() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.lastSeen
}

// UpdateLastSeen updates the node's last seen timestamp to now.
func (n *Node) UpdateLastSeen() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastSeen = time.Now()
}

// GetUptime returns how long the node has been running.
func (n *Node) GetUptime() time.Duration {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.startedAt == nil {
		return 0
	}

	if n.stoppedAt != nil {
		return n.stoppedAt.Sub(*n.startedAt)
	}

	return time.Since(*n.startedAt)
}

// String returns a string representation of the node.
func (n *Node) String() string {
	return fmt.Sprintf("Node{id:%s, name:%s, role:%s, status:%s, address:%s:%d}",
		n.GetID(), n.GetName(), n.GetRole(), n.GetStatus(), n.GetAddress(), n.GetPort())
}

// initSubsystems initializes node subsystems based on the node's role.
func (n *Node) initSubsystems() error {
	// Create subsystem manager
	n.subsystemManager = NewSubsystemManager()

	// Initialize resource monitor
	resourceConfig := &ResourceMonitorConfig{
		Interval: 5 * time.Second,
		Callback: func(resources *NodeResources) {
			// Automatically update node resources when monitor reports new data
			n.SetResources(resources)
		},
	}
	n.resource = NewResourceMonitor(resourceConfig)

	// Initialize subsystems based on node role
	switch n.role {
	case NodeRoleClient:
		// Client nodes need registration and heartbeat subsystems
		registrationSubsystem := NewRegistrationSubsystem(n)
		if err := n.subsystemManager.Register(registrationSubsystem); err != nil {
			return fmt.Errorf("failed to register registration subsystem: %w", err)
		}

		heartbeatSubsystem := NewHeartbeatSubsystem(n, 30*time.Second)
		if err := n.subsystemManager.Register(heartbeatSubsystem); err != nil {
			return fmt.Errorf("failed to register heartbeat subsystem: %w", err)
		}

	case NodeRoleHybrid:
		// Hybrid nodes need registration, heartbeat, and command management subsystems
		registrationSubsystem := NewRegistrationSubsystem(n)
		if err := n.subsystemManager.Register(registrationSubsystem); err != nil {
			return fmt.Errorf("failed to register registration subsystem: %w", err)
		}

		heartbeatSubsystem := NewHeartbeatSubsystem(n, 30*time.Second)
		if err := n.subsystemManager.Register(heartbeatSubsystem); err != nil {
			return fmt.Errorf("failed to register heartbeat subsystem: %w", err)
		}

		commandSubsystem := NewCommandSubsystem(n)
		if err := n.subsystemManager.Register(commandSubsystem); err != nil {
			return fmt.Errorf("failed to register command subsystem: %w", err)
		}

	case NodeRoleMaster:
		// Master nodes need command management subsystem
		commandSubsystem := NewCommandSubsystem(n)
		if err := n.subsystemManager.Register(commandSubsystem); err != nil {
			return fmt.Errorf("failed to register command subsystem: %w", err)
		}
	}

	return nil
}

// startSubsystems starts the resource monitor and all registered subsystems.
func (n *Node) startSubsystems() error {
	// Start resource monitor
	if n.resource != nil {
		if err := n.resource.Start(); err != nil {
			return fmt.Errorf("failed to start resource monitor: %w", err)
		}
	}

	// Start other subsystems
	if n.subsystemManager != nil {
		if err := n.subsystemManager.Start(); err != nil {
			// Stop already-started resource monitor
			if n.resource != nil {
				n.resource.Stop() //errcheck:ignore
			}
			return fmt.Errorf("failed to start subsystems: %w", err)
		}
	}

	return nil
}

// stopSubsystems stops all subsystems and the resource monitor.
func (n *Node) stopSubsystems() {
	// Stop subsystem manager
	if n.subsystemManager != nil {
		if err := n.subsystemManager.Stop(); err != nil {
			logger.Warnf("停止子系统管理器失败: %v", err)
		}
	}

	// Stop resource monitor
	if n.resource != nil {
		if err := n.resource.Stop(); err != nil {
			logger.Warnf("停止资源监控失败: %v", err)
		}
	}
}

// Context returns the node's context.
func (n *Node) Context() context.Context {
	return n.ctx
}

// GetResourceMonitor returns the resource monitor instance.
func (n *Node) GetResourceMonitor() *ResourceMonitor {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.resource
}

// GetResourceSnapshot returns the current resource snapshot (convenience method).
func (n *Node) GetResourceSnapshot() *NodeResources {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.resource == nil {
		return n.resources
	}

	return n.resource.GetSnapshot()
}

// GetGPUInfo returns GPU information (convenience method).
func (n *Node) GetGPUInfo() []gpu.Info {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.resource == nil {
		return make([]gpu.Info, 0)
	}

	return n.resource.GetGPUInfo()
}

// GetLlamacppInfo returns llama.cpp information (convenience method).
func (n *Node) GetLlamacppInfo() *LlamacppInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.resource == nil {
		return nil
	}

	return n.resource.GetLlamacppInfo()
}

// Master functionality: client management
// The following methods are used by nodes in the Master role.

// clientRegistry stores registered client nodes.
type clientRegistry struct {
	clients map[string]*NodeInfo // nodeID -> NodeInfo
	mu      sync.RWMutex
}

// RegisterClient registers a new client node.
func (n *Node) RegisterClient(info *NodeInfo) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if info.ID == "" {
		return fmt.Errorf("客户端节点ID不能为空")
	}

	// 检查是否已存在
	if n.clientRegistry == nil {
		n.clientRegistry = &clientRegistry{
			clients: make(map[string]*NodeInfo),
		}
	}

	n.clientRegistry.mu.Lock()
	defer n.clientRegistry.mu.Unlock()

	// 创建副本
	infoCopy := *info
	infoCopy.RegisteredAt = time.Now()
	infoCopy.LastSeen = time.Now()

	n.clientRegistry.clients[info.ID] = &infoCopy
	n.updatedAt = time.Now()

	return nil
}

// UnregisterClient unregisters a client node.
func (n *Node) UnregisterClient(nodeID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.clientRegistry == nil {
		return fmt.Errorf("客户端注册表未初始化")
	}

	n.clientRegistry.mu.Lock()
	defer n.clientRegistry.mu.Unlock()

	if _, exists := n.clientRegistry.clients[nodeID]; !exists {
		return fmt.Errorf("客户端节点不存在: %s", nodeID)
	}

	delete(n.clientRegistry.clients, nodeID)
	n.updatedAt = time.Now()

	return nil
}

// GetClient returns the specified client's information.
func (n *Node) GetClient(nodeID string) (*NodeInfo, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.clientRegistry == nil {
		return nil, fmt.Errorf("客户端注册表未初始化")
	}

	n.clientRegistry.mu.RLock()
	defer n.clientRegistry.mu.RUnlock()

	client, exists := n.clientRegistry.clients[nodeID]
	if !exists {
		return nil, fmt.Errorf("客户端节点不存在: %s", nodeID)
	}

	// 返回副本
	clientCopy := *client
	return &clientCopy, nil
}

// ListClients returns all registered clients.
func (n *Node) ListClients() []*NodeInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.clientRegistry == nil {
		return make([]*NodeInfo, 0)
	}

	n.clientRegistry.mu.RLock()
	defer n.clientRegistry.mu.RUnlock()

	clients := make([]*NodeInfo, 0, len(n.clientRegistry.clients))
	for _, client := range n.clientRegistry.clients {
		clientCopy := *client
		clients = append(clients, &clientCopy)
	}

	return clients
}

// HandleHeartbeat processes a client heartbeat message.
func (n *Node) HandleHeartbeat(nodeID string, heartbeat *HeartbeatMessage) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.clientRegistry == nil {
		return fmt.Errorf("客户端注册表未初始化")
	}

	n.clientRegistry.mu.Lock()
	defer n.clientRegistry.mu.Unlock()

	client, exists := n.clientRegistry.clients[nodeID]
	if !exists {
		return fmt.Errorf("客户端节点不存在: %s", nodeID)
	}

	// 更新客户端信息
	client.LastSeen = time.Now()
	if heartbeat.Resources != nil {
		client.Resources = heartbeat.Resources
	}
	if heartbeat.Status != "" {
		newStatus := NodeStatus(heartbeat.Status)
		if !isValidNodeTransition(client.Status, newStatus) {
			logger.Warnf("invalid status from heartbeat: %s -> %s for client %s", client.Status, newStatus, nodeID)
		} else {
			client.Status = newStatus
		}
	}

	n.clientRegistry.clients[nodeID] = client
	n.updatedAt = time.Now()

	return nil
}

// GetClientCount returns client count statistics (total, online, offline, busy).
func (n *Node) GetClientCount() (total, online, offline, busy int) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.clientRegistry == nil {
		return 0, 0, 0, 0
	}

	n.clientRegistry.mu.RLock()
	defer n.clientRegistry.mu.RUnlock()

	total = len(n.clientRegistry.clients)
	for _, client := range n.clientRegistry.clients {
		switch client.Status {
		case NodeStatusOnline:
			online++
		case NodeStatusOffline:
			offline++
		case NodeStatusBusy:
			busy++
		}
	}

	return total, online, offline, busy
}

// Client functionality: command management
// The following methods are used by nodes in the Client role.

// commandQueue stores pending commands for client nodes.
type commandQueue struct {
	commands map[string][]*Command // nodeID -> commands
	mu       sync.RWMutex
}

// commandResultStore stores command execution results.
type commandResultStore struct {
	results map[string]*CommandResult // commandID -> result
	mu      sync.RWMutex
}

// QueueCommand queues a command for a client node to execute.
func (n *Node) QueueCommand(nodeID string, cmd *Command) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.commandQueue == nil {
		n.commandQueue = &commandQueue{
			commands: make(map[string][]*Command),
		}
	}

	n.commandQueue.mu.Lock()
	defer n.commandQueue.mu.Unlock()

	if n.commandQueue.commands[nodeID] == nil {
		n.commandQueue.commands[nodeID] = make([]*Command, 0)
	}

	n.commandQueue.commands[nodeID] = append(n.commandQueue.commands[nodeID], cmd)
	n.updatedAt = time.Now()

	return nil
}

// GetPendingCommands retrieves and clears pending commands for the specified node.
func (n *Node) GetPendingCommands(nodeID string) []*Command {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.commandQueue == nil {
		return make([]*Command, 0)
	}

	n.commandQueue.mu.Lock()
	defer n.commandQueue.mu.Unlock()

	commands := n.commandQueue.commands[nodeID]
	if commands == nil {
		return make([]*Command, 0)
	}

	// 返回副本并清空队列
	result := make([]*Command, len(commands))
	copy(result, commands)
	n.commandQueue.commands[nodeID] = make([]*Command, 0)

	return result
}

// Command result storage

// StoreCommandResult stores a command execution result.
func (n *Node) StoreCommandResult(result *CommandResult) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.commandResults == nil {
		n.commandResults = &commandResultStore{
			results: make(map[string]*CommandResult),
		}
	}

	n.commandResults.mu.Lock()
	defer n.commandResults.mu.Unlock()

	// 创建副本
	resultCopy := *result
	n.commandResults.results[result.CommandID] = &resultCopy
	n.updatedAt = time.Now()

	return nil
}

// CleanOldCommandResults removes old command results, keeping only the most recent N.
func (n *Node) CleanOldCommandResults(keepCount int) int {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.commandResults == nil {
		return 0
	}

	n.commandResults.mu.Lock()
	defer n.commandResults.mu.Unlock()

	if len(n.commandResults.results) <= keepCount {
		return 0
	}

	// 按完成时间排序
	type resultWithTime struct {
		result    *CommandResult
		completed time.Time
	}

	sorted := make([]resultWithTime, 0, len(n.commandResults.results))
	for _, result := range n.commandResults.results {
		sorted = append(sorted, resultWithTime{
			result:    result,
			completed: result.CompletedAt,
		})
	}

	// 按完成时间降序排序
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].completed.After(sorted[i].completed) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// 删除旧的结果
	removed := 0
	for i := keepCount; i < len(sorted); i++ {
		delete(n.commandResults.results, sorted[i].result.CommandID)
		removed++
	}

	n.updatedAt = time.Now()
	return removed
}
