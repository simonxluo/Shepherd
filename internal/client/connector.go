// Package client provides client-side functionality for connecting to master nodes.
//
// Deprecated: This package is part of the old distributed architecture.
// New code should use github.com/shepherd-project/shepherd/Shepherd/internal/node.Node instead.
// This package will be removed in a future release.
package client

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/node"
)

// MasterConnector 管理 Client 到 Master 的连接，包括注册、心跳和命令处理
//
// Deprecated: Use node.Node with Client role instead. This type will be removed in a future release.
type MasterConnector struct {
	// 节点标识
	nodeID     string
	masterAddr string
	nodeInfo   *node.NodeInfo

	// 组件
	heartbeatMgr *node.HeartbeatManager
	executor     *node.CommandExecutor

	// HTTP 客户端
	client *http.Client

	// 状态
	registered bool
	connected  bool
	mu         sync.RWMutex

	// 重连配置
	maxReconnectAttempts int
	reconnectBackoff     time.Duration

	// 命令轮询
	commandPollInterval time.Duration
	commandHandler      func(*node.Command) (*node.CommandResult, error)

	// 资源监控
	resourceMonitor *node.ResourceMonitor

	// 上下文控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 日志
	log *logger.Logger
}

// MasterConnectorConfig MasterConnector 配置
type MasterConnectorConfig struct {
	NodeID               string
	MasterAddr           string
	NodeInfo             *node.NodeInfo
	HeartbeatMgr         *node.HeartbeatManager
	Executor             *node.CommandExecutor
	ResourceMonitor      *node.ResourceMonitor
	MaxReconnectAttempts int           // 默认 10
	ReconnectBackoff     time.Duration // 默认 5 秒
	CommandPollInterval  time.Duration // 默认 2 秒
	CommandHandler       func(*node.Command) (*node.CommandResult, error)
	Logger               *logger.Logger
}

// NewMasterConnector 创建新的 MasterConnector
func NewMasterConnector(config *MasterConnectorConfig) (*MasterConnector, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	if config.NodeID == "" {
		return nil, fmt.Errorf("节点 ID 不能为空")
	}

	if config.MasterAddr == "" {
		return nil, fmt.Errorf("Master 地址不能为空")
	}

	if config.HeartbeatMgr == nil {
		return nil, fmt.Errorf("HeartbeatManager 不能为空")
	}

	if config.Executor == nil {
		return nil, fmt.Errorf("CommandExecutor 不能为空")
	}

	// 设置默认值
	if config.MaxReconnectAttempts == 0 {
		config.MaxReconnectAttempts = 10
	}
	if config.ReconnectBackoff == 0 {
		config.ReconnectBackoff = 5 * time.Second
	}
	if config.CommandPollInterval == 0 {
		config.CommandPollInterval = 2 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &MasterConnector{
		nodeID:               config.NodeID,
		masterAddr:           config.MasterAddr,
		nodeInfo:             config.NodeInfo,
		heartbeatMgr:         config.HeartbeatMgr,
		executor:             config.Executor,
		client:               &http.Client{Timeout: 30 * time.Second},
		maxReconnectAttempts: config.MaxReconnectAttempts,
		reconnectBackoff:     config.ReconnectBackoff,
		commandPollInterval:  config.CommandPollInterval,
		commandHandler:       config.CommandHandler,
		resourceMonitor:      config.ResourceMonitor,
		ctx:                  ctx,
		cancel:               cancel,
		log:                  config.Logger,
	}, nil
}

// Connect 向 Master 注册节点并启动连接
func (mc *MasterConnector) Connect() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.registered {
		return fmt.Errorf("节点已注册")
	}

	// 执行注册（带重试）
	var lastErr error
	for attempt := 0; attempt <= mc.maxReconnectAttempts; attempt++ {
		if attempt > 0 {
			delay := mc.calculateBackoff(attempt)
			if mc.log != nil {
				mc.log.Infof("注册失败，%v 后重试 (第 %d/%d 次)", delay, attempt, mc.maxReconnectAttempts)
			}

			select {
			case <-time.After(delay):
			case <-mc.ctx.Done():
				return fmt.Errorf("连接已取消")
			}
		}

		err := mc.register()
		if err == nil {
			mc.registered = true
			mc.connected = true
			if mc.log != nil {
				mc.log.Infof("成功注册到 Master: %s", mc.masterAddr)
			}

			// 启动心跳管理器
			if err := mc.heartbeatMgr.Start(); err != nil {
				mc.registered = false
				mc.connected = false
				return fmt.Errorf("启动心跳管理器失败: %w", err)
			}

			// 启动命令轮询
			mc.wg.Add(1)
			go mc.commandPollLoop()

			// 启动连接监控
			mc.wg.Add(1)
			go mc.connectionMonitor()

			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("注册到 Master 失败，已重试 %d 次: %w", mc.maxReconnectAttempts, lastErr)
}

// Disconnect 优雅断开与 Master 的连接
func (mc *MasterConnector) Disconnect() error {
	mc.mu.Lock()
	if !mc.registered {
		mc.mu.Unlock()
		return nil
	}
	mc.mu.Unlock()

	// 取消上下文，停止所有 goroutine
	mc.cancel()

	// 等待 goroutine 退出
	mc.wg.Wait()

	// 停止心跳管理器
	if err := mc.heartbeatMgr.Stop(); err != nil {
		if mc.log != nil {
			mc.log.Warnf("停止心跳管理器失败: %v", err)
		}
	}

	// 注销节点
	if err := mc.unregister(); err != nil {
		if mc.log != nil {
			mc.log.Warnf("从 Master 注销失败: %v", err)
		}
	}

	mc.mu.Lock()
	mc.registered = false
	mc.connected = false
	mc.mu.Unlock()

	if mc.log != nil {
		mc.log.Info("已从 Master 断开连接")
	}

	return nil
}

// IsConnected 返回是否已连接
func (mc *MasterConnector) IsConnected() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.connected && mc.heartbeatMgr.IsConnected()
}

// IsRegistered 返回是否已注册
func (mc *MasterConnector) IsRegistered() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.registered
}

// GetNodeID 返回节点 ID
func (mc *MasterConnector) GetNodeID() string {
	return mc.nodeID
}

// GetMasterAddr 返回 Master 地址
func (mc *MasterConnector) GetMasterAddr() string {
	return mc.masterAddr
}

// 向 Master 注册节点
func (mc *MasterConnector) register() error {
	url := fmt.Sprintf("%s/api/master/nodes/register", mc.masterAddr)

	// 构建注册请求
	registerReq := map[string]interface{}{
		"nodeId":       mc.nodeID,
		"nodeInfo":     mc.nodeInfo,
		"timestamp":    time.Now(),
		"capabilities": mc.getCapabilities(),
	}

	body, err := json.Marshal(registerReq)
	if err != nil {
		return fmt.Errorf("序列化注册请求失败: %w", err)
	}

	resp, err := mc.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送注册请求失败: %w", err)
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("注册失败，HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// 从 Master 注销节点
func (mc *MasterConnector) unregister() error {
	url := fmt.Sprintf("%s/api/master/nodes/%s/unregister", mc.masterAddr, mc.nodeID)

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, nil)
	if err != nil {
		return fmt.Errorf("创建注销请求失败: %w", err)
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送注销请求失败: %w", err)
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("注销失败，HTTP %d", resp.StatusCode)
	}

	return nil
}

// 获取节点能力信息
func (mc *MasterConnector) getCapabilities() map[string]interface{} {
	caps := map[string]interface{}{
		"nodeId":   mc.nodeID,
		"version":  "0.1.0",
		"hostname": mc.nodeID,
	}

	if mc.nodeInfo != nil && mc.nodeInfo.Capabilities != nil {
		caps["gpu"] = mc.nodeInfo.Capabilities.GPU
		caps["gpuCount"] = mc.nodeInfo.Capabilities.GPUCount
		caps["cpuCount"] = mc.nodeInfo.Capabilities.CPUCount
		caps["memory"] = mc.nodeInfo.Capabilities.Memory
	}

	return caps
}

// 命令轮询循环
func (mc *MasterConnector) commandPollLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.commandPollInterval)
	defer ticker.Stop()

	// 立即执行一次
	mc.pollAndExecuteCommands()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.pollAndExecuteCommands()
		}
	}
}

// 轮询并执行命令
func (mc *MasterConnector) pollAndExecuteCommands() {
	commands, err := mc.fetchCommands()
	if err != nil {
		if mc.log != nil {
			mc.log.Debugf("获取命令失败: %v", err)
		}
		return
	}

	for _, cmd := range commands {
		mc.executeAndReport(cmd)
	}
}

// 从 Master 获取待执行命令
func (mc *MasterConnector) fetchCommands() ([]*node.Command, error) {
	url := fmt.Sprintf("%s/api/master/nodes/%s/commands", mc.masterAddr, mc.nodeID)

	req, err := http.NewRequestWithContext(mc.ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取命令失败，HTTP %d", resp.StatusCode)
	}

	var commands []*node.Command
	if err := json.NewDecoder(resp.Body).Decode(&commands); err != nil {
		return nil, fmt.Errorf("解析命令响应失败: %w", err)
	}

	return commands, nil
}

// 执行命令并上报结果
func (mc *MasterConnector) executeAndReport(cmd *node.Command) {
	if mc.log != nil {
		mc.log.Infof("执行命令: %s (类型: %s)", cmd.ID, cmd.Type)
	}

	var result *node.CommandResult
	var err error

	// 如果有自定义处理器，使用它；否则使用默认执行器
	if mc.commandHandler != nil {
		result, err = mc.commandHandler(cmd)
	} else {
		result, err = mc.executor.Execute(cmd)
	}

	if err != nil {
		result = &node.CommandResult{
			CommandID:   cmd.ID,
			FromNodeID:  mc.nodeID,
			ToNodeID:    cmd.FromNodeID,
			Success:     false,
			Error:       err.Error(),
			CompletedAt: time.Now(),
		}
	}

	if result == nil {
		result = &node.CommandResult{
			CommandID:   cmd.ID,
			FromNodeID:  mc.nodeID,
			ToNodeID:    cmd.FromNodeID,
			Success:     false,
			Error:       "执行结果为空",
			CompletedAt: time.Now(),
		}
	}

	// 上报结果
	if err := mc.reportResult(result); err != nil {
		if mc.log != nil {
			mc.log.Errorf("上报命令结果失败: %v", err)
		}
	}
}

// 上报命令执行结果
func (mc *MasterConnector) reportResult(result *node.CommandResult) error {
	url := fmt.Sprintf("%s/api/master/command/result", mc.masterAddr)

	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("序列化结果失败: %w", err)
	}

	resp, err := mc.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送结果失败: %w", err)
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("上报结果失败，HTTP %d", resp.StatusCode)
	}

	return nil
}

// 连接监控循环
func (mc *MasterConnector) connectionMonitor() {
	defer mc.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.checkConnection()
		}
	}
}

// 检查连接状态
func (mc *MasterConnector) checkConnection() {
	isConnected := mc.heartbeatMgr.IsConnected()

	mc.mu.Lock()
	wasConnected := mc.connected
	mc.connected = isConnected
	mc.mu.Unlock()

	// 连接断开，尝试重连
	if wasConnected && !isConnected {
		if mc.log != nil {
			mc.log.Warn("检测到连接断开，准备重连")
		}
		mc.attemptReconnect()
	}
}

// 尝试重新连接
func (mc *MasterConnector) attemptReconnect() {
	mc.mu.Lock()
	if mc.registered {
		mc.mu.Unlock()
		return
	}
	mc.mu.Unlock()

	for attempt := 1; attempt <= mc.maxReconnectAttempts; attempt++ {
		delay := mc.calculateBackoff(attempt)
		if mc.log != nil {
			mc.log.Infof("等待 %v 后尝试重连 (第 %d/%d 次)", delay, attempt, mc.maxReconnectAttempts)
		}

		select {
		case <-time.After(delay):
		case <-mc.ctx.Done():
			return
		}

		if err := mc.register(); err != nil {
			if mc.log != nil {
				mc.log.Warnf("重连失败: %v", err)
			}
			continue
		}

		mc.mu.Lock()
		mc.registered = true
		mc.connected = true
		mc.mu.Unlock()

		if mc.log != nil {
			mc.log.Info("重连成功")
		}
		return
	}

	if mc.log != nil {
		mc.log.Errorf("重连失败，已达最大重试次数 %d", mc.maxReconnectAttempts)
	}
}

// 计算退避延迟（指数退避 + 抖动）
func (mc *MasterConnector) calculateBackoff(attempt int) time.Duration {
	// 基础延迟：1s, 2s, 4s, 8s... 最大 60s
	baseDelay := mc.reconnectBackoff
	if attempt > 1 {
		multiplier := 1 << uint(attempt-1)
		baseDelay = time.Duration(multiplier) * mc.reconnectBackoff
	}

	if baseDelay > 60*time.Second {
		baseDelay = 60 * time.Second
	}

	// 添加抖动 (±25%)
	jitter := time.Duration(float64(baseDelay) * 0.25 * (float64(time.Now().UnixNano()%100) / 100.0))
	return baseDelay + jitter
}

// UpdateNodeInfo 更新节点信息
func (mc *MasterConnector) UpdateNodeInfo(info *node.NodeInfo) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.nodeInfo = info
}

// SetCommandHandler 设置自定义命令处理器
func (mc *MasterConnector) SetCommandHandler(handler func(*node.Command) (*node.CommandResult, error)) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.commandHandler = handler
}
