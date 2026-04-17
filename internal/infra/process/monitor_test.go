// Package process provides unit tests for process monitoring
package process

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLogger 创建测试用 logger
func newTestLogger() *logger.Logger {
	logCfg := &config.LogConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
	log, _ := logger.NewLogger(logCfg, "test")
	return log
}

// TestNewProcessMonitor tests creating a new process monitor
func TestNewProcessMonitor(t *testing.T) {
	log := newTestLogger()
	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 1*time.Second, log)

	assert.NotNil(t, monitor)
	assert.Equal(t, proc, monitor.process)
	assert.Equal(t, 1*time.Second, monitor.interval)
	assert.NotNil(t, monitor.stopChan)
	assert.Equal(t, "unknown", monitor.metrics.State)
}

// TestProcessMonitorGetMetrics tests retrieving metrics
func TestProcessMonitorGetMetrics(t *testing.T) {
	log := newTestLogger()
	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)

	// 手动收集指标
	metrics := monitor.collectMetrics()

	assert.NotNil(t, metrics)
	// 进程未启动，PID 应该是 0
	assert.Equal(t, 0, metrics.PID)
	assert.Equal(t, "stopped", metrics.State)
}

// TestProcessMonitorOutputHandling tests output line handling
func TestProcessMonitorOutputHandling(t *testing.T) {
	log := newTestLogger()
	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)

	// 设置输出处理器
	proc.SetOutputHandler(monitor.handleOutputLine)

	// 模拟输出行
	testLines := []string{
		"Loading model...",
		"Processing shards...",
		"all slots are idle and system prompt is empty",
		"Model loaded successfully",
	}

	for _, line := range testLines {
		monitor.handleOutputLine(line)
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	monitor.logMu.Lock()
	logCount := len(monitor.logBuffer)
	monitor.logMu.Unlock()

	assert.Equal(t, len(testLines), logCount)
}

// TestProcessMonitorCallbacks tests metric update callbacks
func TestProcessMonitorCallbacks(t *testing.T) {
	log := newTestLogger()
	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)

	// 添加回调
	callbackCalled := false
	var callbackMetrics ProcessMetrics

	monitor.AddCallback(func(metrics ProcessMetrics) {
		callbackCalled = true
		callbackMetrics = metrics
	})

	// 手动收集指标
	monitor.collectMetrics()

	// 验证回调被调用
	assert.True(t, callbackCalled)
	assert.Equal(t, 0, callbackMetrics.PID)
}

// TestProcessMonitorLogBuffer tests circular log buffer
func TestProcessMonitorLogBuffer(t *testing.T) {
	log := newTestLogger()
	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)

	// 设置输出处理器
	proc.SetOutputHandler(monitor.handleOutputLine)

	// 添加超过缓冲区大小的日志行（缓冲区大小为 100）
	for i := 0; i < 150; i++ {
		monitor.handleOutputLine(fmt.Sprintf("Log line %d", i))
	}

	monitor.logMu.Lock()
	bufferSize := len(monitor.logBuffer)
	lastLog := monitor.logBuffer[len(monitor.logBuffer)-1]
	monitor.logMu.Unlock()

	// 验证缓冲区大小不超过 100
	assert.LessOrEqual(t, bufferSize, 100)

	// 验证最后一条日志是最新的
	assert.Equal(t, "Log line 149", lastLog)
}

// TestProcessMetricsStruct tests ProcessMetrics struct
func TestProcessMetricsStruct(t *testing.T) {
	metrics := ProcessMetrics{
		PID:        12345,
		Port:       8081,
		State:      "running",
		MemoryMB:   1024.5,
		CPUPercent: 45.2,
		Uptime:     3600,
		LastLog:    "Processing request",
		HealthOK:   true,
	}

	assert.Equal(t, 12345, metrics.PID)
	assert.Equal(t, 8081, metrics.Port)
	assert.Equal(t, "running", metrics.State)
	assert.Equal(t, 1024.5, metrics.MemoryMB)
	assert.Equal(t, 45.2, metrics.CPUPercent)
	assert.Equal(t, int64(3600), metrics.Uptime)
	assert.Equal(t, "Processing request", metrics.LastLog)
	assert.True(t, metrics.HealthOK)
}

// TestGetSystemUptime tests system uptime retrieval
func TestGetSystemUptime(t *testing.T) {
	log := newTestLogger()
	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)

	uptime, err := monitor.getSystemUptime()

	require.NoError(t, err)
	assert.Greater(t, uptime, 0.0)

	// 系统运行时间应该合理（大于 0，小于 10 年）
	assert.Less(t, uptime, 315360000.0) // 10 年（秒）
}

// TestProcessMonitorWithNonExistentProcess tests monitoring a non-existent process
func TestProcessMonitorWithNonExistentProcess(t *testing.T) {
	log := newTestLogger()

	proc := &Process{
		ID: "non-existent",
	}
	// 不启动进程

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)

	// 收集指标
	metrics := monitor.collectMetrics()

	// 状态应该是 stopped
	assert.Equal(t, "stopped", metrics.State)
	assert.False(t, metrics.HealthOK)
}

// TestProcessMonitorStoppedProcess tests monitoring a stopped process
func TestProcessMonitorStoppedProcess(t *testing.T) {
	log := newTestLogger()

	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)

	// 收集指标
	metrics := monitor.collectMetrics()

	// 验证状态
	assert.Equal(t, "stopped", metrics.State)
	assert.False(t, metrics.HealthOK)
}

// TestProcessMonitorLoadCompletionDetection tests load completion detection
func TestProcessMonitorLoadCompletionDetection(t *testing.T) {
	log := newTestLogger()
	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)
	proc.SetOutputHandler(monitor.handleOutputLine)

	// 模拟加载完成输出
	monitor.handleOutputLine("INFO: all slots are idle and system prompt is empty")

	// 验证日志被捕获
	monitor.logMu.Lock()
	lastLog := monitor.logBuffer[len(monitor.logBuffer)-1]
	monitor.logMu.Unlock()

	assert.Contains(t, lastLog, "all slots are idle")
}

// BenchmarkCollectMetrics benchmarks metrics collection
func BenchmarkCollectMetrics(b *testing.B) {
	log := newTestLogger()
	proc := &Process{
		ID: "test-id",
	}

	monitor := NewProcessMonitor(proc, 100*time.Millisecond, log)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.collectMetrics()
	}
}

// TestParseProcStat tests /proc/{pid}/stat parsing
func TestParseProcStat(t *testing.T) {
	if _, err := os.Stat("/proc/self/stat"); os.IsNotExist(err) {
		t.Skip("Skipping /proc test on non-Linux system")
	}

	pid := os.Getpid()
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	require.NoError(t, err)

	fields := strings.Fields(string(data))
	require.GreaterOrEqual(t, len(fields), 17)

	// 解析 utime (field 14) 和 stime (field 15)
	utime, err := strconv.ParseFloat(fields[13], 64)
	require.NoError(t, err)
	stime, err := strconv.ParseFloat(fields[14], 64)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, utime, 0.0)
	assert.GreaterOrEqual(t, stime, 0.0)
}

// TestParseProcStatm tests /proc/{pid}/statm parsing
func TestParseProcStatm(t *testing.T) {
	if _, err := os.Stat("/proc/self/statm"); os.IsNotExist(err) {
		t.Skip("Skipping /proc test on non-Linux system")
	}

	pid := os.Getpid()
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	require.NoError(t, err)

	// 确保转换为 string
	fields := strings.Fields(string(data))
	require.GreaterOrEqual(t, len(fields), 2)

	// 解析 RSS (field 2)
	rss, err := strconv.ParseInt(fields[1], 10, 64)
	require.NoError(t, err)

	assert.Greater(t, rss, int64(0))
}
