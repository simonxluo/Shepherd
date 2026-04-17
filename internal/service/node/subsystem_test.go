package node

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSubsystem 是一个用于测试的子系统实现
type mockSubsystem struct {
	name       string
	running    bool
	startErr   error
	stopErr    error
	startDelay time.Duration
	stopDelay  time.Duration
}

func (m *mockSubsystem) Name() string {
	return m.name
}

func (m *mockSubsystem) Start(ctx context.Context) error {
	if m.startDelay > 0 {
		select {
		case <-time.After(m.startDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if m.startErr != nil {
		return m.startErr
	}
	m.running = true
	return nil
}

func (m *mockSubsystem) Stop() error {
	if m.stopDelay > 0 {
		time.Sleep(m.stopDelay)
	}
	if m.stopErr != nil {
		return m.stopErr
	}
	m.running = false
	return nil
}

func (m *mockSubsystem) IsRunning() bool {
	return m.running
}

// TestSubsystemManager_New 测试 SubsystemManager 创建
func TestSubsystemManager_New(t *testing.T) {
	sm := NewSubsystemManager()
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.subsystems)
	assert.NotNil(t, sm.ctx)
	assert.NotNil(t, sm.cancel)
	assert.False(t, sm.IsRunning())
}

// TestSubsystemManager_Register 测试子系统注册
func TestSubsystemManager_Register(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *SubsystemManager
		subsystem Subsystem
		wantErr   bool
		errMsg    string
	}{
		{
			name: "register new subsystem",
			setup: func() *SubsystemManager {
				return NewSubsystemManager()
			},
			subsystem: &mockSubsystem{name: "test-subsystem"},
			wantErr:   false,
		},
		{
			name: "register duplicate subsystem",
			setup: func() *SubsystemManager {
				sm := NewSubsystemManager()
				sm.Register(&mockSubsystem{name: "test-subsystem"})
				return sm
			},
			subsystem: &mockSubsystem{name: "test-subsystem"},
			wantErr:   true,
			errMsg:    "子系统已存在",
		},
		{
			name: "register while running",
			setup: func() *SubsystemManager {
				sm := NewSubsystemManager()
				sm.Register(&mockSubsystem{name: "existing"})
				sm.Start()
				return sm
			},
			subsystem: &mockSubsystem{name: "new-subsystem"},
			wantErr:   true,
			errMsg:    "无法在运行时注册",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.setup()
			defer sm.Stop()

			err := sm.Register(tt.subsystem)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				// 验证注册成功
				_, exists := sm.Get(tt.subsystem.Name())
				assert.True(t, exists)
			}
		})
	}
}

// TestSubsystemManager_Unregister 测试子系统注销
func TestSubsystemManager_Unregister(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *SubsystemManager
		name2   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "unregister existing subsystem",
			setup: func() *SubsystemManager {
				sm := NewSubsystemManager()
				sm.Register(&mockSubsystem{name: "test-subsystem"})
				return sm
			},
			name2:   "test-subsystem",
			wantErr: false,
		},
		{
			name: "unregister non-existent subsystem",
			setup: func() *SubsystemManager {
				return NewSubsystemManager()
			},
			name2:   "non-existent",
			wantErr: true,
			errMsg:  "子系统不存在",
		},
		{
			name: "unregister while running",
			setup: func() *SubsystemManager {
				sm := NewSubsystemManager()
				sm.Register(&mockSubsystem{name: "test-subsystem"})
				sm.Start()
				return sm
			},
			name2:   "test-subsystem",
			wantErr: true,
			errMsg:  "无法在运行时注销",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.setup()
			defer sm.Stop()

			err := sm.Unregister(tt.name2)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				// 验证注销成功
				_, exists := sm.Get(tt.name2)
				assert.False(t, exists)
			}
		})
	}
}

// TestSubsystemManager_StartStop 测试启动和停止
func TestSubsystemManager_StartStop(t *testing.T) {
	t.Run("start and stop empty manager", func(t *testing.T) {
		sm := NewSubsystemManager()

		err := sm.Start()
		assert.NoError(t, err)
		assert.True(t, sm.IsRunning())

		err = sm.Stop()
		assert.NoError(t, err)
		assert.False(t, sm.IsRunning())
	})

	t.Run("start with subsystems", func(t *testing.T) {
		sm := NewSubsystemManager()
		sub1 := &mockSubsystem{name: "sub1"}
		sub2 := &mockSubsystem{name: "sub2"}

		sm.Register(sub1)
		sm.Register(sub2)

		err := sm.Start()
		require.NoError(t, err)

		// 验证子系统已启动
		assert.True(t, sub1.IsRunning())
		assert.True(t, sub2.IsRunning())

		err = sm.Stop()
		require.NoError(t, err)

		// 验证子系统已停止
		assert.False(t, sub1.IsRunning())
		assert.False(t, sub2.IsRunning())
	})

	t.Run("double start", func(t *testing.T) {
		sm := NewSubsystemManager()
		sm.Start()
		defer sm.Stop()

		err := sm.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "已在运行")
	})

	t.Run("stop when not running", func(t *testing.T) {
		sm := NewSubsystemManager()

		err := sm.Stop()
		assert.NoError(t, err)
	})
}

// TestSubsystemManager_StartFailure 测试启动失败
func TestSubsystemManager_StartFailure(t *testing.T) {
	sm := NewSubsystemManager()
	sub1 := &mockSubsystem{name: "sub1"}
	sub2 := &mockSubsystem{name: "sub2", startErr: fmt.Errorf("start failed")}
	sub3 := &mockSubsystem{name: "sub3"}

	// 注册子系统，sub2 会启动失败
	sm.Register(sub1)
	sm.Register(sub2)
	sm.Register(sub3)

	// 启动应该失败
	err := sm.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "启动子系统 sub2 失败")

	// 验证所有子系统都已停止（回滚）
	assert.False(t, sm.IsRunning())
	assert.False(t, sub1.IsRunning())
	assert.False(t, sub2.IsRunning())
	assert.False(t, sub3.IsRunning())
}

// TestSubsystemManager_Get 测试获取子系统
func TestSubsystemManager_Get(t *testing.T) {
	sm := NewSubsystemManager()
	sub := &mockSubsystem{name: "test-subsystem"}

	// 注册前获取
	_, exists := sm.Get("test-subsystem")
	assert.False(t, exists)

	// 注册后获取
	sm.Register(sub)
	got, exists := sm.Get("test-subsystem")
	assert.True(t, exists)
	assert.Equal(t, sub, got)
}

// TestSubsystemManager_List 测试列出子系统
func TestSubsystemManager_List(t *testing.T) {
	sm := NewSubsystemManager()

	// 空列表
	names := sm.List()
	assert.Empty(t, names)

	// 注册子系统
	sm.Register(&mockSubsystem{name: "sub1"})
	sm.Register(&mockSubsystem{name: "sub2"})
	sm.Register(&mockSubsystem{name: "sub3"})

	names = sm.List()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "sub1")
	assert.Contains(t, names, "sub2")
	assert.Contains(t, names, "sub3")
}

// TestSubsystemManager_ConcurrentAccess 测试并发访问
func TestSubsystemManager_ConcurrentAccess(t *testing.T) {
	sm := NewSubsystemManager()

	// 并发注册
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			sub := &mockSubsystem{name: fmt.Sprintf("sub-%d", idx)}
			sm.Register(sub)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有子系统都已注册
	names := sm.List()
	assert.Len(t, names, 10)
}

// ==================== HeartbeatSubsystem 测试 ====================

// TestHeartbeatSubsystem_New 测试创建
func TestHeartbeatSubsystem_New(t *testing.T) {
	node := &Node{id: "test-node"}

	tests := []struct {
		name     string
		node     *Node
		interval time.Duration
		validate func(t *testing.T, hs *HeartbeatSubsystem)
	}{
		{
			name:     "with default interval",
			node:     node,
			interval: 0,
			validate: func(t *testing.T, hs *HeartbeatSubsystem) {
				assert.Equal(t, "heartbeat", hs.Name())
				assert.Equal(t, 30*time.Second, hs.interval)
				assert.Equal(t, node, hs.node)
				assert.False(t, hs.IsRunning())
			},
		},
		{
			name:     "with custom interval",
			node:     node,
			interval: 10 * time.Second,
			validate: func(t *testing.T, hs *HeartbeatSubsystem) {
				assert.Equal(t, 10*time.Second, hs.interval)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hs := NewHeartbeatSubsystem(tt.node, tt.interval)
			tt.validate(t, hs)
		})
	}
}

// TestHeartbeatSubsystem_StartStop 测试启动停止
func TestHeartbeatSubsystem_StartStop(t *testing.T) {
	node := &Node{id: "test-node"}
	hs := NewHeartbeatSubsystem(node, 100*time.Millisecond)

	// 启动
	err := hs.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, hs.IsRunning())

	// 重复启动
	err = hs.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已在运行")

	// 停止
	err = hs.Stop()
	require.NoError(t, err)
	assert.False(t, hs.IsRunning())
}

// ==================== CommandSubsystem 测试 ====================

// TestCommandSubsystem_New 测试创建
func TestCommandSubsystem_New(t *testing.T) {
	node := &Node{id: "test-node"}
	cs := NewCommandSubsystem(node)

	assert.Equal(t, "commands", cs.Name())
	assert.Equal(t, node, cs.node)
	assert.False(t, cs.IsRunning())
}

// TestCommandSubsystem_StartStop 测试启动停止
func TestCommandSubsystem_StartStop(t *testing.T) {
	node := &Node{id: "test-node"}
	cs := NewCommandSubsystem(node)

	// 启动
	err := cs.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, cs.IsRunning())

	// 重复启动
	err = cs.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已在运行")

	// 停止
	err = cs.Stop()
	require.NoError(t, err)
	assert.False(t, cs.IsRunning())
}
