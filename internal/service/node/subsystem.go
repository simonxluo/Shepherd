// Package node provides subsystem management for distributed nodes
package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

// Subsystem represents a pluggable component of a Node
type Subsystem interface {
	// Name returns the subsystem name
	Name() string

	// Start starts the subsystem
	Start(ctx context.Context) error

	// Stop stops the subsystem
	Stop() error

	// IsRunning returns true if the subsystem is running
	IsRunning() bool
}

// SubsystemManager manages all node subsystems
type SubsystemManager struct {
	subsystems map[string]Subsystem
	mu         sync.RWMutex
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewSubsystemManager creates a new subsystem manager
func NewSubsystemManager() *SubsystemManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &SubsystemManager{
		subsystems: make(map[string]Subsystem),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Register registers a subsystem
func (sm *SubsystemManager) Register(subsystem Subsystem) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("无法在运行时注册子系统: %s", subsystem.Name())
	}

	if _, exists := sm.subsystems[subsystem.Name()]; exists {
		return fmt.Errorf("子系统已存在: %s", subsystem.Name())
	}

	sm.subsystems[subsystem.Name()] = subsystem
	return nil
}

// Unregister removes a subsystem
func (sm *SubsystemManager) Unregister(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("无法在运行时注销子系统: %s", name)
	}

	if _, exists := sm.subsystems[name]; !exists {
		return fmt.Errorf("子系统不存在: %s", name)
	}

	delete(sm.subsystems, name)
	return nil
}

// Start starts all registered subsystems
func (sm *SubsystemManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("子系统管理器已在运行")
	}

	// 按依赖顺序启动子系统
	// registration 必须最先启动，因为客户端需要先注册到 Master
	startOrder := []string{"registration", "heartbeat", "commands", "resource"}

	for _, name := range startOrder {
		if subsystem, exists := sm.subsystems[name]; exists {
			if err := subsystem.Start(sm.ctx); err != nil {
				// 启动失败，停止已启动的子系统
				sm.stopAll()
				return fmt.Errorf("启动子系统 %s 失败: %w", name, err)
			}
		}
	}

	// 启动其他没有特定顺序的子系统
	for name, subsystem := range sm.subsystems {
		alreadyStarted := false
		for _, ordered := range startOrder {
			if name == ordered {
				alreadyStarted = true
				break
			}
		}

		if !alreadyStarted {
			if err := subsystem.Start(sm.ctx); err != nil {
				sm.stopAll()
				return fmt.Errorf("启动子系统 %s 失败: %w", name, err)
			}
		}
	}

	sm.running = true
	return nil
}

// Stop stops all running subsystems
func (sm *SubsystemManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	sm.cancel()
	sm.stopAll()
	sm.running = false
	return nil
}

// stopAll stops all subsystems (must be called with lock held)
func (sm *SubsystemManager) stopAll() {
	// 按启动相反顺序停止
	for _, subsystem := range sm.subsystems {
		if subsystem.IsRunning() {
			if err := subsystem.Stop(); err != nil {
				// 记录错误但继续停止其他子系统
				logger.Errorf("停止子系统 %s 失败: %v", subsystem.Name(), err)
			} else {
				logger.Infof("子系统 %s 已停止", subsystem.Name())
			}
		}
	}
}

// Get returns a subsystem by name
func (sm *SubsystemManager) Get(name string) (Subsystem, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subsystem, exists := sm.subsystems[name]
	return subsystem, exists
}

// List returns all registered subsystems
func (sm *SubsystemManager) List() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	names := make([]string, 0, len(sm.subsystems))
	for name := range sm.subsystems {
		names = append(names, name)
	}
	return names
}

// IsRunning returns true if the subsystem manager is running
func (sm *SubsystemManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.running
}

// 内置子系统实现

// HeartbeatSubsystem manages heartbeats for client nodes
type HeartbeatSubsystem struct {
	node     *Node
	running  bool
	mu       sync.RWMutex
	interval time.Duration
}

// NewHeartbeatSubsystem creates a new heartbeat subsystem
func NewHeartbeatSubsystem(node *Node, interval time.Duration) *HeartbeatSubsystem {
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &HeartbeatSubsystem{
		node:     node,
		interval: interval,
	}
}

func (hs *HeartbeatSubsystem) Name() string {
	return "heartbeat"
}

func (hs *HeartbeatSubsystem) Start(ctx context.Context) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.running {
		return fmt.Errorf("心跳子系统已在运行")
	}

	hs.running = true

	// 启动心跳协程
	go hs.heartbeatLoop(ctx)

	return nil
}

func (hs *HeartbeatSubsystem) Stop() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hs.running = false
	return nil
}

func (hs *HeartbeatSubsystem) IsRunning() bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.running
}

func (hs *HeartbeatSubsystem) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(hs.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 发送心跳到 Master（如果配置了 Master 地址）
			if hs.node.config != nil && hs.node.config.MasterAddress != "" {
				hs.sendHeartbeatToMaster()
			}
			// 更新最后活跃时间
			hs.node.UpdateLastSeen()
		}
	}
}

// sendHeartbeatToMaster 发送心跳到 Master 节点
func (hs *HeartbeatSubsystem) sendHeartbeatToMaster() {
	// 构建心跳消息
	heartbeat := &HeartbeatMessage{
		NodeID:    hs.node.GetID(),
		Timestamp: time.Now(),
		Status:    hs.node.GetStatus(),
		Resources: hs.node.GetResources(),
	}

	// 构建 Master URL
	masterURL := fmt.Sprintf("%s/api/master/nodes/%s/heartbeat",
		hs.node.config.MasterAddress, heartbeat.NodeID)

	// 序列化心跳消息
	body, err := json.Marshal(heartbeat)
	if err != nil {
		logger.Errorf("序列化心跳消息失败: %v", err)
		return
	}

	// 创建 HTTP 客户端（带超时）
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", masterURL, bytes.NewBuffer(body))
	if err != nil {
		logger.Errorf("创建心跳请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("发送心跳到 Master 失败: %v", err)
		return
	}
	defer utils.CloseQuietly(resp.Body)

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		logger.Errorf("心跳请求失败: HTTP %d", resp.StatusCode)
		return
	}

	logger.Debugf("心跳发送成功: 节点=%s, Master=%s", heartbeat.NodeID, hs.node.config.MasterAddress)
}

// CommandSubsystem manages command execution
type CommandSubsystem struct {
	node    *Node
	running bool
	mu      sync.RWMutex
}

// NewCommandSubsystem creates a new command subsystem
func NewCommandSubsystem(node *Node) *CommandSubsystem {
	return &CommandSubsystem{
		node: node,
	}
}

func (cs *CommandSubsystem) Name() string {
	return "commands"
}

func (cs *CommandSubsystem) Start(ctx context.Context) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.running {
		return fmt.Errorf("命令子系统已在运行")
	}

	cs.running = true

	// 启动命令处理协程
	go cs.commandLoop(ctx)

	return nil
}

func (cs *CommandSubsystem) Stop() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.running = false
	return nil
}

func (cs *CommandSubsystem) IsRunning() bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.running
}

func (cs *CommandSubsystem) commandLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 定期检查并清理旧命令结果
			if cs.node != nil {
				cs.node.CleanOldCommandResults(1000) // 保留最近 1000 条
			}
		}
	}
}

// RegistrationSubsystem handles initial registration with master
type RegistrationSubsystem struct {
	node    *Node
	running bool
	mu      sync.RWMutex
}

// NewRegistrationSubsystem creates a new registration subsystem
func NewRegistrationSubsystem(node *Node) *RegistrationSubsystem {
	return &RegistrationSubsystem{
		node: node,
	}
}

func (rs *RegistrationSubsystem) Name() string {
	return "registration"
}

func (rs *RegistrationSubsystem) Start(ctx context.Context) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.running {
		return fmt.Errorf("注册子系统已在运行")
	}

	rs.running = true

	// 立即执行注册
	go rs.registerWithMaster(ctx)

	return nil
}

func (rs *RegistrationSubsystem) Stop() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.running = false
	return nil
}

func (rs *RegistrationSubsystem) IsRunning() bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.running
}

// registerWithMaster 向 Master 注册当前节点
func (rs *RegistrationSubsystem) registerWithMaster(ctx context.Context) {
	// 检查是否配置了 Master 地址
	if rs.node.config == nil || rs.node.config.MasterAddress == "" {
		logger.Infof("未配置 Master 地址，跳过注册")
		return
	}

	// 构建节点信息
	nodeInfo := &NodeInfo{
		ID:       rs.node.GetID(),
		Name:     rs.node.GetName(),
		Address:  rs.node.GetAddress(),
		Port:     rs.node.GetPort(),
		Role:     rs.node.GetRole(),
		Status:   rs.node.GetStatus(),
		Version:  rs.node.GetVersion(),
		Tags:     rs.node.GetTags(),
		Metadata: rs.node.GetMetadata(),
		Capabilities: &NodeCapabilities{
			SupportsLlama:  rs.node.capabilities.SupportsLlama,
			SupportsPython: rs.node.capabilities.SupportsPython,
			GPU:            rs.node.capabilities.GPU,
			GPUCount:       rs.node.capabilities.GPUCount,
			CPUCount:       rs.node.capabilities.CPUCount,
			Memory:         rs.node.capabilities.Memory,
		},
		Resources:    rs.node.GetResources(),
		LastSeen:     time.Now(),
		RegisteredAt: time.Now(),
	}

	// 构建 Master URL
	masterURL := fmt.Sprintf("%s/api/master/nodes/register", rs.node.config.MasterAddress)

	// 序列化节点信息
	body, err := json.Marshal(nodeInfo)
	if err != nil {
		logger.Errorf("序列化节点信息失败: %v", err)
		return
	}

	// 创建 HTTP 客户端（带超时）
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 重试注册逻辑
	maxRetries := rs.node.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}

	retryDelay := 5 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			logger.Infof("注册失败，%v 后重试 (第 %d/%d 次)", retryDelay, attempt, maxRetries)
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				logger.Warnf("注册被取消")
				return
			}
		}

		// 创建 HTTP 请求
		req, err := http.NewRequestWithContext(ctx, "POST", masterURL, bytes.NewBuffer(body))
		if err != nil {
			logger.Errorf("创建注册请求失败: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		// 发送请求
		resp, err := client.Do(req)
		if err != nil {
			logger.Errorf("发送注册请求失败: %v", err)
			continue
		}

		// 读取响应
		respBody, _ := io.ReadAll(resp.Body)
		utils.CloseQuietly(resp.Body)

		// 检查响应状态
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			logger.Infof("成功注册到 Master: %s (节点ID: %s)", rs.node.config.MasterAddress, rs.node.GetID())
			return
		}

		logger.Errorf("注册失败: HTTP %d - %s", resp.StatusCode, string(respBody))
	}

	logger.Errorf("注册到 Master 失败，已达最大重试次数: %d", maxRetries)
}
