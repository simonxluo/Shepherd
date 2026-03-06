// Package node provides distributed node management implementation.
package node

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
)

// HeartbeatManager 管理 Client 到 Master 的心跳连接
type HeartbeatManager struct {
	// 配置
	nodeID     string
	masterAddr string
	interval   time.Duration
	timeout    time.Duration
	maxRetries int

	// HTTP 客户端
	client *http.Client

	// 状态
	connected   bool
	lastSuccess time.Time
	retryCount  int
	lastError   error

	// 资源监控器（获取资源信息）
	resourceMonitor *ResourceMonitor

	// 回调函数
	onConnect    func()
	onDisconnect func(error)
	onHeartbeat  func(success bool)

	// 并发控制
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// 日志
	log *logger.Logger
}

// HeartbeatConfig 心跳管理器配置
type HeartbeatConfig struct {
	NodeID          string
	MasterAddr      string
	Interval        time.Duration    // 默认 5 秒
	Timeout         time.Duration    // 默认 15 秒（3 个周期）
	MaxRetries      int              // 默认 5
	ResourceMonitor *ResourceMonitor // 资源监控器
	OnConnect       func()
	OnDisconnect    func(error)
	OnHeartbeat     func(success bool)
	Logger          *logger.Logger
}

// NewHeartbeatManager 创建新的心跳管理器
func NewHeartbeatManager(config *HeartbeatConfig) *HeartbeatManager {
	if config == nil {
		config = &HeartbeatConfig{}
	}

	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 3 * config.Interval // 3 个周期
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 5
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HeartbeatManager{
		nodeID:          config.NodeID,
		masterAddr:      config.MasterAddr,
		interval:        config.Interval,
		timeout:         config.Timeout,
		maxRetries:      config.MaxRetries,
		client:          &http.Client{Timeout: 30 * time.Second},
		resourceMonitor: config.ResourceMonitor,
		onConnect:       config.OnConnect,
		onDisconnect:    config.OnDisconnect,
		onHeartbeat:     config.OnHeartbeat,
		ctx:             ctx,
		cancel:          cancel,
		log:             config.Logger,
	}
}

// Start 启动心跳管理器
func (hm *HeartbeatManager) Start() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.running {
		return fmt.Errorf("心跳管理器已在运行")
	}

	if hm.masterAddr == "" {
		return fmt.Errorf("Master 地址不能为空")
	}

	if hm.nodeID == "" {
		return fmt.Errorf("节点 ID 不能为空")
	}

	hm.running = true
	hm.retryCount = 0

	hm.wg.Add(1)
	go hm.heartbeatLoop()

	if hm.log != nil {
		hm.log.Infof("心跳管理器已启动，目标: %s，间隔: %v", hm.masterAddr, hm.interval)
	}
	return nil
}

// Stop 停止心跳管理器
func (hm *HeartbeatManager) Stop() error {
	hm.mu.Lock()
	if !hm.running {
		hm.mu.Unlock()
		return nil
	}
	hm.running = false
	hm.mu.Unlock()

	// 取消 context
	hm.cancel()

	// 等待 goroutine 退出
	hm.wg.Wait()

	if hm.log != nil {
		hm.log.Info("心跳管理器已停止")
	}
	return nil
}

// IsRunning 返回是否正在运行
func (hm *HeartbeatManager) IsRunning() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.running
}

// IsConnected 返回是否已连接
func (hm *HeartbeatManager) IsConnected() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.connected
}

// GetLastSuccess 返回最后一次成功时间
func (hm *HeartbeatManager) GetLastSuccess() time.Time {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.lastSuccess
}

// GetRetryCount 返回当前重试次数
func (hm *HeartbeatManager) GetRetryCount() int {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.retryCount
}

// 心跳循环
func (hm *HeartbeatManager) heartbeatLoop() {
	defer hm.wg.Done()

	// 立即发送一次心跳
	hm.sendHeartbeatWithRetry()

	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			hm.sendHeartbeatWithRetry()
		}
	}
}

// 发送心跳（带重试）
func (hm *HeartbeatManager) sendHeartbeatWithRetry() {
	success := false
	var lastErr error

	// 最多重试 maxRetries 次
	for attempt := 0; attempt <= hm.maxRetries; attempt++ {
		if attempt > 0 {
			// 计算退避延迟：指数退避 + 抖动
			delay := hm.calculateBackoff(attempt)
			if hm.log != nil {
				hm.log.Infof("心跳发送失败，%v 后重试 (第 %d/%d 次)", delay, attempt, hm.maxRetries)
			}

			select {
			case <-time.After(delay):
			case <-hm.ctx.Done():
				return
			}
		}

		err := hm.sendHeartbeat()
		if err == nil {
			success = true
			break
		}
		lastErr = err
	}

	hm.mu.Lock()
	wasConnected := hm.connected
	if success {
		hm.connected = true
		hm.lastSuccess = time.Now()
		hm.retryCount = 0
		hm.lastError = nil
	} else {
		hm.retryCount++
		hm.lastError = lastErr
		// 如果重试次数过多，标记为断开
		if hm.retryCount >= hm.maxRetries {
			hm.connected = false
		}
	}
	hm.mu.Unlock()

	// 回调
	if hm.onHeartbeat != nil {
		hm.onHeartbeat(success)
	}

	if success {
		if hm.onConnect != nil && !wasConnected {
			hm.onConnect()
		}
	} else {
		if hm.onDisconnect != nil && wasConnected {
			hm.onDisconnect(lastErr)
		}
	}
}

// 发送单个心跳
func (hm *HeartbeatManager) sendHeartbeat() error {
	// 构建心跳消息
	message := &HeartbeatMessage{
		NodeID:    hm.nodeID,
		Timestamp: time.Now(),
		Status:    NodeStatusOnline,
	}

	// 获取资源信息
	if hm.resourceMonitor != nil {
		message.Resources = hm.resourceMonitor.GetSnapshot()
	}

	// 序列化
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化心跳消息失败: %w", err)
	}

	// 发送 HTTP 请求
	url := fmt.Sprintf("%s/api/master/heartbeat", hm.masterAddr)
	resp, err := hm.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送心跳请求失败: %w", err)
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("心跳请求返回非 200 状态码: %d", resp.StatusCode)
	}

	return nil
}

// 计算退避延迟（指数退避 + 抖动）
func (hm *HeartbeatManager) calculateBackoff(attempt int) time.Duration {
	// 基础延迟：1s, 2s, 4s, 8s...
	delay := time.Duration(1<<uint(attempt-1)) * time.Second

	// 最大 60 秒
	if delay > 60*time.Second {
		delay = 60 * time.Second
	}

	// 添加抖动 (±25%)
	jitter := time.Duration(float64(delay) * 0.25 * (float64(time.Now().UnixNano()%100) / 100.0))
	delay = delay + jitter

	return delay
}

// 检查之前是否已连接（用于回调）
func (hm *HeartbeatManager) wasConnected() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.connected
}
