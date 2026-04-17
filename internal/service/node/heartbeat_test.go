package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHeartbeatManager_New 测试 HeartbeatManager 创建
func TestHeartbeatManager_New(t *testing.T) {
	tests := []struct {
		name     string
		config   *HeartbeatConfig
		wantNil  bool
		validate func(t *testing.T, hm *HeartbeatManager)
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantNil: false,
			validate: func(t *testing.T, hm *HeartbeatManager) {
				assert.Equal(t, 5*time.Second, hm.interval)
				assert.Equal(t, 15*time.Second, hm.timeout)
				assert.Equal(t, 5, hm.maxRetries)
				assert.NotNil(t, hm.client)
				assert.NotNil(t, hm.ctx)
				assert.NotNil(t, hm.cancel)
			},
		},
		{
			name: "custom config",
			config: &HeartbeatConfig{
				NodeID:     "test-node",
				MasterAddr: "http://localhost:9190",
				Interval:   10 * time.Second,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			wantNil: false,
			validate: func(t *testing.T, hm *HeartbeatManager) {
				assert.Equal(t, "test-node", hm.nodeID)
				assert.Equal(t, "http://localhost:9190", hm.masterAddr)
				assert.Equal(t, 10*time.Second, hm.interval)
				assert.Equal(t, 30*time.Second, hm.timeout)
				assert.Equal(t, 3, hm.maxRetries)
			},
		},
		{
			name: "zero values use defaults",
			config: &HeartbeatConfig{
				NodeID:     "test-node",
				MasterAddr: "http://localhost:9190",
				Interval:   0, // 应该使用默认值
				Timeout:    0, // 应该使用默认值
				MaxRetries: 0, // 应该使用默认值
			},
			wantNil: false,
			validate: func(t *testing.T, hm *HeartbeatManager) {
				assert.Equal(t, 5*time.Second, hm.interval)
				assert.Equal(t, 15*time.Second, hm.timeout)
				assert.Equal(t, 5, hm.maxRetries)
			},
		},
		{
			name: "with resource monitor",
			config: &HeartbeatConfig{
				NodeID:          "test-node",
				MasterAddr:      "http://localhost:9190",
				ResourceMonitor: NewResourceMonitor(nil),
			},
			wantNil: false,
			validate: func(t *testing.T, hm *HeartbeatManager) {
				assert.NotNil(t, hm.resourceMonitor)
			},
		},
		{
			name: "with callbacks",
			config: &HeartbeatConfig{
				NodeID:       "test-node",
				MasterAddr:   "http://localhost:9190",
				OnConnect:    func() {},
				OnDisconnect: func(err error) {},
				OnHeartbeat:  func(success bool) {},
			},
			wantNil: false,
			validate: func(t *testing.T, hm *HeartbeatManager) {
				assert.NotNil(t, hm.onConnect)
				assert.NotNil(t, hm.onDisconnect)
				assert.NotNil(t, hm.onHeartbeat)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hm := NewHeartbeatManager(tt.config)
			if tt.wantNil {
				assert.Nil(t, hm)
			} else {
				assert.NotNil(t, hm)
				if tt.validate != nil {
					tt.validate(t, hm)
				}
			}
		})
	}
}

// TestHeartbeatManager_StartStop 测试启动和停止
func TestHeartbeatManager_StartStop(t *testing.T) {
	tests := []struct {
		name       string
		nodeID     string
		masterAddr string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid start",
			nodeID:     "test-node",
			masterAddr: "http://localhost:9190",
			wantErr:    false,
		},
		{
			name:       "empty master address",
			nodeID:     "test-node",
			masterAddr: "",
			wantErr:    true,
			errMsg:     "Master 地址不能为空",
		},
		{
			name:       "empty node ID",
			nodeID:     "",
			masterAddr: "http://localhost:9190",
			wantErr:    true,
			errMsg:     "节点 ID 不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hm := NewHeartbeatManager(&HeartbeatConfig{
				NodeID:     tt.nodeID,
				MasterAddr: tt.masterAddr,
				Interval:   100 * time.Millisecond,
			})

			err := hm.Start()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
			assert.True(t, hm.IsRunning())

			// 测试重复启动
			err = hm.Start()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "已在运行")

			// 停止
			err = hm.Stop()
			require.NoError(t, err)
			assert.False(t, hm.IsRunning())

			// 测试重复停止（不应报错）
			err = hm.Stop()
			assert.NoError(t, err)
		})
	}
}

// TestHeartbeatManager_Callbacks 测试回调函数
func TestHeartbeatManager_Callbacks(t *testing.T) {
	var connectCalled, disconnectCalled, heartbeatCalled bool
	var lastHeartbeatSuccess bool

	// 创建一个模拟服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/master/heartbeat", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var msg HeartbeatMessage
		err := json.NewDecoder(r.Body).Decode(&msg)
		require.NoError(t, err)
		assert.Equal(t, "test-node", msg.NodeID)
		assert.NotZero(t, msg.Timestamp)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hm := NewHeartbeatManager(&HeartbeatConfig{
		NodeID:     "test-node",
		MasterAddr: server.URL,
		Interval:   100 * time.Millisecond,
		OnConnect: func() {
			connectCalled = true
		},
		OnDisconnect: func(err error) {
			disconnectCalled = true
			_ = err // 保留错误信息，用于调试
		},
		OnHeartbeat: func(success bool) {
			heartbeatCalled = true
			lastHeartbeatSuccess = success
		},
	})

	err := hm.Start()
	require.NoError(t, err)
	defer hm.Stop()

	for i := 0; i < 50; i++ {
		if heartbeatCalled && connectCalled {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	assert.True(t, heartbeatCalled, "heartbeat callback should be called")
	assert.True(t, lastHeartbeatSuccess, "heartbeat should be successful")
	assert.True(t, connectCalled, "connect callback should be called")
	assert.False(t, disconnectCalled, "disconnect should not be called yet")
}

// TestHeartbeatManager_SendHeartbeatSuccess 测试成功发送心跳
func TestHeartbeatManager_SendHeartbeatSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg HeartbeatMessage
		err := json.NewDecoder(r.Body).Decode(&msg)
		require.NoError(t, err)

		// 验证消息内容
		assert.Equal(t, "test-node", msg.NodeID)
		assert.Equal(t, NodeStatusOnline, msg.Status)
		assert.NotZero(t, msg.Timestamp)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	hm := NewHeartbeatManager(&HeartbeatConfig{
		NodeID:     "test-node",
		MasterAddr: server.URL,
		Interval:   100 * time.Millisecond,
	})

	err := hm.Start()
	require.NoError(t, err)
	defer hm.Stop()

	// 等待心跳发送
	time.Sleep(150 * time.Millisecond)

	// 验证状态
	assert.True(t, hm.IsConnected())
	assert.NotZero(t, hm.GetLastSuccess())
	assert.Equal(t, 0, hm.GetRetryCount())
}

// TestHeartbeatManager_SendHeartbeatFailure 测试心跳发送失败和重试
func TestHeartbeatManager_SendHeartbeatFailure(t *testing.T) {
	failCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	hm := NewHeartbeatManager(&HeartbeatConfig{
		NodeID:     "test-node",
		MasterAddr: server.URL,
		Interval:   100 * time.Millisecond,
		MaxRetries: 2,
	})

	err := hm.Start()
	require.NoError(t, err)
	defer hm.Stop()

	// 等待重试
	time.Sleep(500 * time.Millisecond)

	// 验证重试次数
	assert.GreaterOrEqual(t, failCount, 1)
}

// TestHeartbeatManager_ConnectionState 测试连接状态变化
func TestHeartbeatManager_ConnectionState(t *testing.T) {
	shouldFail := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldFail {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	hm := NewHeartbeatManager(&HeartbeatConfig{
		NodeID:     "test-node",
		MasterAddr: server.URL,
		Interval:   100 * time.Millisecond,
		MaxRetries: 1,
	})

	err := hm.Start()
	require.NoError(t, err)
	defer hm.Stop()

	for i := 0; i < 50; i++ {
		if !hm.IsConnected() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.False(t, hm.IsConnected())

	shouldFail = false

	for i := 0; i < 50; i++ {
		if hm.IsConnected() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.True(t, hm.IsConnected())
}

// TestHeartbeatManager_WithResourceMonitor 测试带资源监控器的心跳
func TestHeartbeatManager_WithResourceMonitor(t *testing.T) {
	var receivedResources *NodeResources

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg HeartbeatMessage
		err := json.NewDecoder(r.Body).Decode(&msg)
		require.NoError(t, err)

		receivedResources = msg.Resources
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rm := NewResourceMonitor(&ResourceMonitorConfig{
		Interval: 50 * time.Millisecond,
	})
	err := rm.Start()
	require.NoError(t, err)
	defer rm.Stop()

	hm := NewHeartbeatManager(&HeartbeatConfig{
		NodeID:          "test-node",
		MasterAddr:      server.URL,
		Interval:        100 * time.Millisecond,
		ResourceMonitor: rm,
	})

	err = hm.Start()
	require.NoError(t, err)
	defer hm.Stop()

	// 等待心跳发送
	time.Sleep(150 * time.Millisecond)

	// 验证资源信息被包含
	assert.NotNil(t, receivedResources)
	assert.True(t, receivedResources.CPUTotal > 0)
}

// TestHeartbeatManager_ContextCancellation 测试 Context 取消
func TestHeartbeatManager_ContextCancellation(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hm := NewHeartbeatManager(&HeartbeatConfig{
		NodeID:     "test-node",
		MasterAddr: server.URL,
		Interval:   100 * time.Millisecond,
	})

	err := hm.Start()
	require.NoError(t, err)

	// 等待几次心跳
	time.Sleep(250 * time.Millisecond)
	initialCount := requestCount

	// 停止
	err = hm.Stop()
	require.NoError(t, err)

	// 等待一段时间
	time.Sleep(200 * time.Millisecond)

	// 验证没有新的请求
	assert.Equal(t, initialCount, requestCount)
}

// TestHeartbeatManager_CalculateBackoff 测试退避计算
func TestHeartbeatManager_CalculateBackoff(t *testing.T) {
	hm := NewHeartbeatManager(&HeartbeatConfig{
		NodeID:     "test-node",
		MasterAddr: "http://localhost:9190",
	})

	tests := []struct {
		attempt int
		min     time.Duration
		max     time.Duration
	}{
		{attempt: 1, min: 1 * time.Second, max: 1300 * time.Millisecond},   // 1s + 25%
		{attempt: 2, min: 2 * time.Second, max: 2600 * time.Millisecond},   // 2s + 25%
		{attempt: 3, min: 4 * time.Second, max: 5200 * time.Millisecond},   // 4s + 25%
		{attempt: 4, min: 8 * time.Second, max: 10400 * time.Millisecond},  // 8s + 25%
		{attempt: 7, min: 60 * time.Second, max: 75000 * time.Millisecond}, // 最大 60s
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := hm.calculateBackoff(tt.attempt)
			assert.GreaterOrEqual(t, delay, tt.min)
			assert.LessOrEqual(t, delay, tt.max)
		})
	}
}

// TestHeartbeatManager_ConcurrentAccess 测试并发访问安全
func TestHeartbeatManager_ConcurrentAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hm := NewHeartbeatManager(&HeartbeatConfig{
		NodeID:     "test-node",
		MasterAddr: server.URL,
		Interval:   50 * time.Millisecond,
	})

	err := hm.Start()
	require.NoError(t, err)
	defer hm.Stop()

	// 并发访问测试
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				_ = hm.IsRunning()
				_ = hm.IsConnected()
				_ = hm.GetLastSuccess()
				_ = hm.GetRetryCount()
				time.Sleep(1 * time.Millisecond)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
