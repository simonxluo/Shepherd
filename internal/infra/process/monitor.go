// Package process provides process monitoring capabilities
package process

import (
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

// ProcessMetrics represents process performance metrics
type ProcessMetrics struct {
	PID        int     `json:"pid"`
	Port       int     `json:"port"`
	State      string  `json:"state"`
	MemoryMB   float64 `json:"memory_mb"`
	CPUPercent float64 `json:"cpu_percent"`
	Uptime     int64   `json:"uptime_seconds"`
	LastLog    string  `json:"last_log"`
	HealthOK   bool    `json:"health_ok"`
}

// ProcessMonitor monitors a managed process
type ProcessMonitor struct {
	process     *Process
	interval    time.Duration
	stopChan    chan bool
	metrics     ProcessMetrics
	mu          sync.RWMutex
	callbacks   []func(ProcessMetrics)
	log         *logger.Logger
	startTime   time.Time
	lastCPUTime float64
	logBuffer   []string // Circular buffer for log lines
	logMu       sync.Mutex
}

// NewProcessMonitor creates a new process monitor
func NewProcessMonitor(process *Process, interval time.Duration, log *logger.Logger) *ProcessMonitor {
	return &ProcessMonitor{
		process:   process,
		interval:  interval,
		stopChan:  make(chan bool),
		log:       log,
		startTime: time.Now(),
		metrics: ProcessMetrics{
			State: "unknown",
		},
		logBuffer: make([]string, 0, 100), // Keep last 100 log lines
	}
}

// Start begins monitoring the process
func (m *ProcessMonitor) Start() {
	m.log.Info(fmt.Sprintf("ProcessMonitor: 开始监控进程 (ID: %s)", m.process.ID))

	// Set output handler to capture logs
	m.process.SetOutputHandler(m.handleOutputLine)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectMetrics()
		case <-m.stopChan:
			m.log.Info(fmt.Sprintf("ProcessMonitor: 停止监控进程 (ID: %s)", m.process.ID))
			return
		}
	}
}

// Stop stops monitoring
func (m *ProcessMonitor) Stop() {
	close(m.stopChan)
}

// GetMetrics returns the current metrics
func (m *ProcessMonitor) GetMetrics() ProcessMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

// AddCallback adds a callback function to be called when metrics are updated
func (m *ProcessMonitor) AddCallback(cb func(ProcessMetrics)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, cb)
}

// handleOutputLine captures output lines for monitoring
func (m *ProcessMonitor) handleOutputLine(line string) {
	m.logMu.Lock()
	defer m.logMu.Unlock()

	// Add to circular buffer
	m.logBuffer = append(m.logBuffer, line)
	if len(m.logBuffer) > 100 {
		// Remove oldest entry
		m.logBuffer = m.logBuffer[1:]
	}

	// Check for loading completion
	if strings.Contains(line, "all slots are idle") {
		m.log.Info(fmt.Sprintf("ProcessMonitor: 模型加载完成 (ID: %s)", m.process.ID))
		// Trigger metrics collection immediately
		go m.collectMetrics()
	}
}

// collectMetrics collects process metrics
func (m *ProcessMonitor) collectMetrics() ProcessMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update uptime
	m.metrics.Uptime = int64(time.Since(m.startTime).Seconds())

	// Get process state
	if !m.process.IsRunning() {
		m.metrics.State = "stopped"
		m.metrics.HealthOK = false
	} else {
		m.metrics.State = "running"
	}

	// Get PID and port
	m.metrics.PID = m.process.GetPID()
	m.metrics.Port = m.process.GetPort()

	// Collect resource usage
	m.metrics.MemoryMB = m.getMemoryUsage()
	m.metrics.CPUPercent = m.getCPUUsage()

	// Health check
	if m.metrics.Port > 0 && m.metrics.State == "running" {
		m.metrics.HealthOK = m.healthCheck()
	}

	// Update last log line
	m.logMu.Lock()
	if len(m.logBuffer) > 0 {
		m.metrics.LastLog = m.logBuffer[len(m.logBuffer)-1]
	}
	m.logMu.Unlock()

	// Notify callbacks
	for _, cb := range m.callbacks {
		cb(m.metrics)
	}

	return m.metrics
}

// getMemoryUsage returns memory usage in MB
func (m *ProcessMonitor) getMemoryUsage() float64 {
	pid := m.process.GetPID()
	if pid == 0 {
		return 0
	}

	// Read /proc/{pid}/statm
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}

	rss, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}

	// Convert pages to MB (usually 4KB per page)
	return float64(rss) * 4096 / 1024 / 1024
}

// getCPUUsage returns CPU usage percentage
func (m *ProcessMonitor) getCPUUsage() float64 {
	pid := m.process.GetPID()
	if pid == 0 {
		return 0
	}

	// Read /proc/{pid}/stat
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}

	fields := strings.Fields(string(data))
	if len(fields) < 17 {
		return 0
	}

	// utime (field 13) and stime (field 14)
	utime, err := strconv.ParseFloat(fields[13], 64)
	if err != nil {
		return 0
	}

	stime, err := strconv.ParseFloat(fields[14], 64)
	if err != nil {
		return 0
	}

	// Calculate total CPU time in seconds
	totalCPUTime := utime + stime

	// Get system uptime
	uptime, err := m.getSystemUptime()
	if err != nil {
		return 0
	}

	// Calculate CPU usage percentage (simplified)
	delta := totalCPUTime - m.lastCPUTime
	m.lastCPUTime = totalCPUTime

	if uptime > 0 && m.metrics.Uptime > 0 {
		// CPU% = (delta CPU time / elapsed time) * 100
		return (delta / float64(m.metrics.Uptime)) * 100
	}

	return 0
}

// getSystemUptime returns system uptime in seconds
func (m *ProcessMonitor) getSystemUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid uptime format")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return uptime, nil
}

// healthCheck performs HTTP health check
func (m *ProcessMonitor) healthCheck() bool {
	port := m.process.GetPort()
	if port == 0 {
		return false
	}

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	url := fmt.Sprintf("http://localhost:%d/health", port)
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer utils.CloseQuietly(resp.Body)

	return resp.StatusCode == 200
}
