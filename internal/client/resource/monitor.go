// Package resource provides resource monitoring for client nodes.
package resource

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/cluster"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// Monitor monitors system resources
type Monitor struct {
	mu             sync.RWMutex
	currentUsage   *cluster.ResourceUsage
	startTime      time.Time
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	updateInterval time.Duration
	log            *logger.Logger
}

// NewMonitor creates a new resource monitor
func NewMonitor(updateInterval time.Duration, log *logger.Logger) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		currentUsage:   &cluster.ResourceUsage{},
		startTime:      time.Now(),
		ctx:            ctx,
		cancel:         cancel,
		updateInterval: updateInterval,
		log:            log,
	}
}

// Start starts the resource monitor
func (m *Monitor) Start() {
	m.wg.Add(1)
	go m.updateLoop()
}

// Stop stops the resource monitor
func (m *Monitor) Stop() {
	m.cancel()
	m.wg.Wait()
}

// GetCurrentUsage returns the current resource usage
func (m *Monitor) GetCurrentUsage() *cluster.ResourceUsage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid race conditions
	usage := *m.currentUsage
	return &usage
}

// updateLoop periodically updates resource usage
func (m *Monitor) updateLoop() {
	defer m.wg.Done()

	// Initial update
	m.updateUsage()

	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateUsage()
		}
	}
}

// updateUsage updates the current resource usage
func (m *Monitor) updateUsage() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		m.log.Error(fmt.Sprintf("获取CPU使用率失败: %v", err), nil)
		cpuPercent = []float64{0}
	}

	// Get memory usage
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		m.log.Error(fmt.Sprintf("获取内存使用率失败: %v", err), nil)
		vmStat = &mem.VirtualMemoryStat{
			Used:  0,
			Total: 1,
		}
	}

	// 获取磁盘使用率
	diskPercent := m.getDiskUsage()

	// 获取 GPU 使用率
	gpuPercent, gpuMemUsed, gpuMemTotal := m.getGPUUsage()

	// 计算运行时间（秒）
	uptime := int64(time.Since(m.startTime).Seconds())

	m.currentUsage = &cluster.ResourceUsage{
		CPUPercent:     cpuPercent[0],
		MemoryUsed:     int64(vmStat.Used),
		MemoryTotal:    int64(vmStat.Total),
		GPUPercent:     gpuPercent,
		GPUMemoryUsed:  int64(gpuMemUsed),
		GPUMemoryTotal: int64(gpuMemTotal),
		DiskPercent:    diskPercent,
		Uptime:         uptime,
	}
}

// getDiskUsage 获取磁盘使用率
func (m *Monitor) getDiskUsage() float64 {
	// 获取根分区使用率
	if diskStat, err := disk.Usage("/"); err == nil {
		return diskStat.UsedPercent
	}
	return 0
}

// getGPUUsage 获取 GPU 使用率和内存使用情况
func (m *Monitor) getGPUUsage() (percent, memUsed, memTotal float64) {
	// 尝试检测 NVIDIA GPU
	if m.hasCommand("nvidia-smi") {
		return m.getNvidiaGPUUsage()
	}

	// 尝试检测 AMD GPU (ROCm)
	if m.hasCommand("rocm-smi") {
		return m.getAmdGPUUsage()
	}

	return 0, 0, 0
}

// hasCommand 检查命令是否存在
func (m *Monitor) hasCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// getNvidiaGPUUsage 获取 NVIDIA GPU 使用率
func (m *Monitor) getNvidiaGPUUsage() (percent, memUsed, memTotal float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=utilization.gpu,memory.used,memory.total",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0
	}

	// 解析输出: "85, 1024, 8192"
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) >= 3 {
		if p, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err == nil {
			percent = p
		}
		if u, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
			memUsed = u
		}
		if t, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64); err == nil {
			memTotal = t
		}
	}

	return percent, memUsed, memTotal
}

// getAmdGPUUsage 获取 AMD GPU 使用率 (ROCm)
func (m *Monitor) getAmdGPUUsage() (percent, memUsed, memTotal float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "rocm-smi",
		"--showuse",
		"--showmem",
		"--csv")

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0
	}

	// 解析 ROCm SMI 输出
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "GPU use") {
			// 提取使用率百分比
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "%" && i > 0 {
					if p, err := strconv.ParseFloat(strings.TrimSuffix(parts[i-1], "%"), 64); err == nil {
						percent = p
					}
					break
				}
			}
		}
	}

	// VRAM 使用情况需要更复杂的解析，这里简化处理
	return percent, 0, 0
}
