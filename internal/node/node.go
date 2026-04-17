// Package node provides distributed node management implementation.
package node

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
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
		GPUInfo:     make([]GPUInfo, 0),
		LoadAverage: make([]float64, 3),
	}

	return node, nil
}

// Start 启动节点
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
	n.updatedAt = now
	n.lastSeen = now

	// Initialize subsystems
	if err := n.initSubsystems(); err != nil {
		n.status = NodeStatusError
		n.running = false
		return fmt.Errorf("初始化子系统失败: %w", err)
	}

	// Start subsystems
	if err := n.startSubsystems(); err != nil {
		n.status = NodeStatusError
		n.running = false
		return fmt.Errorf("启动子系统失败: %w", err)
	}

	return nil
}

// Stop 停止节点
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
	n.updatedAt = now

	// Stop subsystems
	n.stopSubsystems()

	// Cancel context
	n.cancel()

	return nil
}

// ID 获取节点ID（实现 INode 接口）
func (n *Node) ID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.id
}

// GetID 获取节点ID（向后兼容）
func (n *Node) GetID() string {
	return n.ID()
}

// Name 获取节点名称（实现 INode 接口）
func (n *Node) Name() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.name
}

// Role 获取节点角色（实现 INode 接口）
func (n *Node) Role() NodeRole {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.role
}

// Status 获取节点状态（实现 INode 接口）
func (n *Node) Status() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}

// Address 获取节点地址（实现 INode 接口）
func (n *Node) Address() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.address
}

// Port 获取节点端口（实现 INode 接口）
func (n *Node) Port() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.port
}

// Health 获取健康状态（实现 INode 接口）
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

// UpdateConfig 更新节点配置（实现 INode 接口）
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

// GetName 获取节点名称
func (n *Node) GetName() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.name
}

// SetName 设置节点名称
func (n *Node) SetName(name string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.name = name
	n.updatedAt = time.Now()
}

// GetRole 获取节点角色
func (n *Node) GetRole() NodeRole {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.role
}

// SetRole 设置节点角色
func (n *Node) SetRole(role NodeRole) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return fmt.Errorf("节点运行时不能更改角色")
	}

	n.role = role
	n.updatedAt = time.Now()
	return nil
}

// GetStatus 获取节点状态
func (n *Node) GetStatus() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}

// SetStatus 设置节点状态
func (n *Node) SetStatus(status NodeStatus) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.status = status
	n.updatedAt = time.Now()
}

// GetAddress 获取节点地址
func (n *Node) GetAddress() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.address
}

// GetPort 获取节点端口
func (n *Node) GetPort() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.port
}

// GetVersion 获取节点版本
func (n *Node) GetVersion() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.version
}

// GetTags 获取节点标签
func (n *Node) GetTags() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]string{}, n.tags...)
}

// SetTags 设置节点标签
func (n *Node) SetTags(tags []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.tags = append([]string{}, tags...)
	n.updatedAt = time.Now()
}

// AddTag 添加节点标签
func (n *Node) AddTag(tag string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, existing := range n.tags {
		if existing == tag {
			return // 已存在
		}
	}

	n.tags = append(n.tags, tag)
	n.updatedAt = time.Now()
}

// RemoveTag 移除节点标签
func (n *Node) RemoveTag(tag string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for i, existing := range n.tags {
		if existing == tag {
			n.tags = append(n.tags[:i], n.tags[i+1:]...)
			break
		}
	}

	n.updatedAt = time.Now()
}

// GetMetadata 获取节点元数据
func (n *Node) GetMetadata() map[string]string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	metadata := make(map[string]string)
	for k, v := range n.metadata {
		metadata[k] = v
	}
	return metadata
}

// SetMetadata 设置节点元数据
func (n *Node) SetMetadata(metadata map[string]string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.metadata = make(map[string]string)
	for k, v := range metadata {
		n.metadata[k] = v
	}
	n.updatedAt = time.Now()
}

// GetCapabilities 获取节点能力
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

// SetCapabilities 设置节点能力
func (n *Node) SetCapabilities(capabilities *NodeCapabilities) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if capabilities == nil {
		n.capabilities = nil
	} else {
		// Create a copy
		n.capabilities = &NodeCapabilities{}
		*n.capabilities = *capabilities
	}
	n.updatedAt = time.Now()
}

// GetResources 获取节点资源信息
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

// SetResources 设置节点资源信息
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

// GetConfig 获取节点配置
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

// IsRunning 检查节点是否正在运行
func (n *Node) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.running
}

// GetCreatedAt 获取节点创建时间
func (n *Node) GetCreatedAt() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.createdAt
}

// GetUpdatedAt 获取节点更新时间
func (n *Node) GetUpdatedAt() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.updatedAt
}

// GetLastSeen 获取节点最后活跃时间
func (n *Node) GetLastSeen() time.Time {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.lastSeen
}

// UpdateLastSeen 更新节点最后活跃时间
func (n *Node) UpdateLastSeen() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastSeen = time.Now()
}

// GetUptime 获取节点运行时长
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

// ToInfo 转换为NodeInfo
func (n *Node) ToInfo() *NodeInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	info := &NodeInfo{
		ID:           n.id,
		Name:         n.name,
		Address:      n.address,
		Port:         n.port,
		Role:         n.role,
		Status:       n.status,
		Version:      n.version,
		Tags:         append([]string{}, n.tags...),
		Capabilities: n.GetCapabilities(),
		Resources:    n.GetResources(),
		CreatedAt:    n.createdAt,
		UpdatedAt:    n.updatedAt,
		LastSeen:     n.lastSeen,
	}

	// Create metadata copy
	info.Metadata = make(map[string]string)
	for k, v := range n.metadata {
		info.Metadata[k] = v
	}

	return info
}

// String 返回节点的字符串表示
func (n *Node) String() string {
	return fmt.Sprintf("Node{id:%s, name:%s, role:%s, status:%s, address:%s:%d}",
		n.GetID(), n.GetName(), n.GetRole(), n.GetStatus(), n.GetAddress(), n.GetPort())
}

// initSubsystems 初始化子系统
func (n *Node) initSubsystems() error {
	// 创建子系统管理器
	n.subsystemManager = NewSubsystemManager()

	// 初始化资源监控器
	resourceConfig := &ResourceMonitorConfig{
		Interval: 5 * time.Second,
		Callback: func(resources *NodeResources) {
			// 当资源信息更新时，自动更新节点的资源信息
			n.SetResources(resources)
		},
	}
	n.resource = NewResourceMonitor(resourceConfig)

	// 根据节点角色初始化子系统
	switch n.role {
	case NodeRoleClient:
		// 客户端节点需要注册子系统和心跳子系统
		registrationSubsystem := NewRegistrationSubsystem(n)
		if err := n.subsystemManager.Register(registrationSubsystem); err != nil {
			return fmt.Errorf("注册注册子系统失败: %w", err)
		}

		heartbeatSubsystem := NewHeartbeatSubsystem(n, 30*time.Second)
		if err := n.subsystemManager.Register(heartbeatSubsystem); err != nil {
			return fmt.Errorf("注册心跳子系统失败: %w", err)
		}

	case NodeRoleHybrid:
		// Hybrid 节点需要注册、心跳和命令管理子系统
		registrationSubsystem := NewRegistrationSubsystem(n)
		if err := n.subsystemManager.Register(registrationSubsystem); err != nil {
			return fmt.Errorf("注册注册子系统失败: %w", err)
		}

		heartbeatSubsystem := NewHeartbeatSubsystem(n, 30*time.Second)
		if err := n.subsystemManager.Register(heartbeatSubsystem); err != nil {
			return fmt.Errorf("注册心跳子系统失败: %w", err)
		}

		commandSubsystem := NewCommandSubsystem(n)
		if err := n.subsystemManager.Register(commandSubsystem); err != nil {
			return fmt.Errorf("注册命令子系统失败: %w", err)
		}

	case NodeRoleMaster:
		// Master 节点需要命令管理子系统
		commandSubsystem := NewCommandSubsystem(n)
		if err := n.subsystemManager.Register(commandSubsystem); err != nil {
			return fmt.Errorf("注册命令子系统失败: %w", err)
		}
	}

	return nil
}

// startSubsystems 启动子系统
func (n *Node) startSubsystems() error {
	// 启动资源监控器
	if n.resource != nil {
		if err := n.resource.Start(); err != nil {
			return fmt.Errorf("启动资源监控器失败: %w", err)
		}
	}

	// 启动其他子系统
	if n.subsystemManager != nil {
		if err := n.subsystemManager.Start(); err != nil {
			// 停止已启动的资源监控器
			if n.resource != nil {
				n.resource.Stop() //errcheck:ignore
			}
			return fmt.Errorf("启动子系统失败: %w", err)
		}
	}

	return nil
}

// stopSubsystems 停止子系统
func (n *Node) stopSubsystems() {
	// 停止子系统管理器
	if n.subsystemManager != nil {
		if err := n.subsystemManager.Stop(); err != nil {
			// 记录错误但继续清理
			// 日志通过 Logger 记录，这里避免循环依赖
		}
	}

	// 停止资源监控器
	if n.resource != nil {
		//errcheck:ignore
		if err := n.resource.Stop(); err != nil {
			// 停止失败只记录日志，不影响其他清理
		}
	}
}

// Context 获取节点上下文
func (n *Node) Context() context.Context {
	return n.ctx
}

// GetResourceMonitor 获取资源监控器
func (n *Node) GetResourceMonitor() *ResourceMonitor {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.resource
}

// GetResourceSnapshot 获取当前资源快照（便捷方法）
func (n *Node) GetResourceSnapshot() *NodeResources {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.resource == nil {
		return n.resources
	}

	return n.resource.GetSnapshot()
}

// GetGPUInfo 获取GPU信息（便捷方法）
func (n *Node) GetGPUInfo() []GPUInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.resource == nil {
		return make([]GPUInfo, 0)
	}

	return n.resource.GetGPUInfo()
}

// GetLlamacppInfo 获取llama.cpp信息（便捷方法）
func (n *Node) GetLlamacppInfo() *LlamacppInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.resource == nil {
		return nil
	}

	return n.resource.GetLlamacppInfo()
}

// ==================== Master 功能：客户端管理 ====================
// 以下方法供 Master 角色使用

// clients 存储已注册的客户端节点
type clientRegistry struct {
	clients map[string]*NodeInfo // nodeID -> NodeInfo
	mu      sync.RWMutex
}

// RegisterClient 注册一个新的客户端节点
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

// UnregisterClient 注销客户端节点
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

// GetClient 获取指定客户端信息
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

// ListClients 列出所有已注册的客户端
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

// HandleHeartbeat 处理客户端心跳
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
		client.Status = NodeStatus(heartbeat.Status)
	}

	n.clientRegistry.clients[nodeID] = client
	n.updatedAt = time.Now()

	return nil
}

// GetClientCount 获取客户端数量统计
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

// ==================== Client 功能：命令管理 ====================
// 以下方法供 Client 角色使用

// pendingCommands 存储待执行的命令
type commandQueue struct {
	commands map[string][]*Command // nodeID -> commands
	mu       sync.RWMutex
}

// commandResults 存储命令执行结果
type commandResultStore struct {
	results map[string]*CommandResult // commandID -> result
	mu      sync.RWMutex
}

// QueueCommand 为客户端节点添加待执行命令
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

// GetPendingCommands 获取指定节点的待执行命令
func (n *Node) GetPendingCommands(nodeID string) []*Command {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.commandQueue == nil {
		return make([]*Command, 0)
	}

	n.commandQueue.mu.RLock()
	defer n.commandQueue.mu.RUnlock()

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

// ==================== 命令结果存储 ====================

// StoreCommandResult 存储命令执行结果
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

// GetCommandResult 获取命令执行结果
func (n *Node) GetCommandResult(commandID string) (*CommandResult, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.commandResults == nil {
		return nil, fmt.Errorf("命令结果不存在: %s", commandID)
	}

	n.commandResults.mu.RLock()
	defer n.commandResults.mu.RUnlock()

	result, exists := n.commandResults.results[commandID]
	if !exists {
		return nil, fmt.Errorf("命令结果不存在: %s", commandID)
	}

	// 返回副本
	resultCopy := *result
	return &resultCopy, nil
}

// GetCommandResultsByNode 获取指定节点的所有命令结果
func (n *Node) GetCommandResultsByNode(nodeID string, limit int) []*CommandResult {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.commandResults == nil {
		return make([]*CommandResult, 0)
	}

	n.commandResults.mu.RLock()
	defer n.commandResults.mu.RUnlock()

	results := make([]*CommandResult, 0)
	for _, result := range n.commandResults.results {
		if result.FromNodeID == nodeID || result.ToNodeID == nodeID {
			resultCopy := *result
			results = append(results, &resultCopy)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results
}

// CleanOldCommandResults 清理旧的命令结果（保留最近 N 条）
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
