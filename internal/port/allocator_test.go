// Package port provides unit tests for port allocation
package port

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPortAllocator tests creating a new port allocator
func TestNewPortAllocator(t *testing.T) {
	allocator := NewPortAllocator(8081, 9000)

	assert.NotNil(t, allocator)
	assert.Equal(t, 8081, allocator.basePort)
	assert.Equal(t, 9000, allocator.maxPort)
	assert.NotNil(t, allocator.allocated)
}

// TestNextPort tests basic port allocation
func TestNextPort(t *testing.T) {
	allocator := NewPortAllocator(10000, 10010) // 使用高端口避免冲突

	port1, err := allocator.NextPort()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, port1, 10000)
	assert.LessOrEqual(t, port1, 10010)

	port2, err := allocator.NextPort()
	require.NoError(t, err)
	assert.NotEqual(t, port1, port2)

	// Verify ports are marked as allocated
	assert.True(t, allocator.IsAllocated(port1))
	assert.True(t, allocator.IsAllocated(port2))

	// Use the variables
	_ = port1
	_ = port2
}

// TestNextPortExhaustion tests port exhaustion
func TestNextPortExhaustion(t *testing.T) {
	allocator := NewPortAllocator(10000, 10002) // 仅 3 个端口

	// 分配所有端口
	_, err := allocator.NextPort()
	require.NoError(t, err)

	port2, err := allocator.NextPort()
	require.NoError(t, err)

	_, err = allocator.NextPort()
	require.NoError(t, err)

	// 下一个应该失败
	_, err = allocator.NextPort()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available ports")

	// 释放一个端口
	allocator.Release(port2)

	// 现在应该可以分配
	port4, err := allocator.NextPort()
	require.NoError(t, err)
	assert.Equal(t, port2, port4)
}

// TestRelease tests port release
func TestRelease(t *testing.T) {
	allocator := NewPortAllocator(10000, 10010)

	port, err := allocator.NextPort()
	require.NoError(t, err)

	assert.True(t, allocator.IsAllocated(port))

	allocator.Release(port)

	assert.False(t, allocator.IsAllocated(port))

	// 重新分配应该获得相同端口
	port2, err := allocator.NextPort()
	require.NoError(t, err)
	assert.Equal(t, port, port2)
}

// TestConcurrentAllocation tests concurrent port allocation
func TestConcurrentAllocation(t *testing.T) {
	allocator := NewPortAllocator(10000, 10100)
	numGoroutines := 50
	allocatedPorts := make(chan int, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			port, err := allocator.NextPort()
			if err == nil {
				allocatedPorts <- port
			}
		}()
	}

	wg.Wait()
	close(allocatedPorts)

	// 收集所有分配的端口
	ports := make(map[int]bool)
	portCount := 0
	for port := range allocatedPorts {
		if ports[port] {
			t.Errorf("端口 %d 被分配了两次", port)
		}
		ports[port] = true
		portCount++
	}

	// 验证所有端口都是唯一的
	assert.Equal(t, portCount, len(ports))
}

// TestIsPortInUse tests port availability checking
func TestIsPortInUse(t *testing.T) {
	allocator := NewPortAllocator(10000, 10010)

	// 启动一个简单的 TCP 服务器来占用端口
	listener, err := net.Listen("tcp", "127.0.0.1:10001")
	require.NoError(t, err)
	defer listener.Close()

	// 给服务器一点时间启动
	time.Sleep(100 * time.Millisecond)

	// 测试已占用端口
	port1, err := allocator.NextPort()
	// 应该跳过 10001，分配其他端口
	require.NoError(t, err)
	assert.NotEqual(t, 10001, port1)

	// 验证可以分配其他端口
	port2, err := allocator.NextPort()
	require.NoError(t, err)
	assert.NotEqual(t, 10001, port2)
	assert.NotEqual(t, port1, port2)
}

// TestPortAllocationOrder tests that ports are allocated sequentially
func TestPortAllocationOrder(t *testing.T) {
	allocator := NewPortAllocator(10000, 10010)

	expectedPorts := []int{10000, 10001, 10002, 10003, 10004}

	for _, expectedPort := range expectedPorts {
		port, err := allocator.NextPort()
		require.NoError(t, err)
		assert.Equal(t, expectedPort, port)
	}
}

// BenchmarkNextPort benchmarks port allocation
func BenchmarkNextPort(b *testing.B) {
	allocator := NewPortAllocator(10000, 20000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		port, err := allocator.NextPort()
		if err == nil {
			allocator.Release(port)
		}
	}
}

// BenchmarkConcurrentAllocation benchmarks concurrent port allocation
func BenchmarkConcurrentAllocation(b *testing.B) {
	allocator := NewPortAllocator(10000, 20000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			port, err := allocator.NextPort()
			if err == nil {
				// 立即释放以避免耗尽
				allocator.Release(port)
			}
		}
	})
}

// TestPortAllocatorRealWorldScenario tests a realistic scenario
func TestPortAllocatorRealWorldScenario(t *testing.T) {
	allocator := NewPortAllocator(8081, 9000)

	// 模拟多个服务请求端口
	services := []string{"model-a", "model-b", "model-c", "model-d"}
	servicePorts := make(map[string]int)

	for _, service := range services {
		port, err := allocator.NextPort()
		require.NoError(t, err, fmt.Sprintf("Failed to allocate port for %s", service))
		servicePorts[service] = port
		t.Logf("%s -> port %d", service, port)
	}

	// 验证所有端口都是唯一的
	ports := make(map[int]bool)
	for _, port := range servicePorts {
		assert.False(t, ports[port], "Port %d was allocated twice", port)
		ports[port] = true
	}

	// 模拟服务停止
	allocator.Release(servicePorts["model-b"])
	delete(servicePorts, "model-b")

	// 新服务应该能够使用释放的端口
	newPort, err := allocator.NextPort()
	require.NoError(t, err)
	assert.Equal(t, 8082, newPort) // 第二个端口
	t.Logf("model-e -> port %d (reused)", newPort)
}
