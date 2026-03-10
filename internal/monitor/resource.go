// Package monitor provides resource monitoring implementation
// 这个包提供资源监控的实现
package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/node"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// MemoryResourceMonitor implements IResourceMonitor using in-memory storage
// MemoryResourceMonitor 使用内存存储实现 IResourceMonitor
type MemoryResourceMonitor struct {
	// 配置参数
	interval      time.Duration               // 采样间隔
	llamacppPaths []string                    // llama.cpp 可执行文件路径
	callbacks     []func(*node.NodeResources) // 资源更新回调函数列表

	// 运行时状态
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	mu         sync.RWMutex
	lastUpdate time.Time
	startTime  time.Time

	// 资源数据
	resources    *node.NodeResources
	gpuInfo      []node.GPUInfo
	llamacppInfo *node.LlamacppInfo
	metrics      []node.NodeMetrics // 历史指标
	maxMetrics   int                // 最大保存指标数量

	// 日志
	log *logger.Logger
}

// ResourceMonitorConfig 资源监控器配置
type ResourceMonitorConfig struct {
	Interval      time.Duration             // 采样间隔，默认5秒
	LlamacppPaths []string                  // llama.cpp 可执行文件路径
	Callback      func(*node.NodeResources) // 资源更新回调（单个，向后兼容）
	Logger        *logger.Logger
	MaxMetrics    int // 最大保存历史指标数量
}

// NewMemoryResourceMonitor creates a new in-memory resource monitor
// NewMemoryResourceMonitor 创建新的内存资源监控器
func NewMemoryResourceMonitor(config *ResourceMonitorConfig) *MemoryResourceMonitor {
	if config == nil {
		config = &ResourceMonitorConfig{}
	}

	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}

	if config.MaxMetrics == 0 {
		config.MaxMetrics = 100 // 默认保存100个历史指标
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 初始化资源数据
	resources := &node.NodeResources{
		CPUTotal:    int64(runtime.NumCPU()) * 1000, // 转换为 millicores
		MemoryTotal: 0,                              // 将在初始化时设置
		DiskTotal:   0,                              // 将在初始化时设置
		GPUInfo:     make([]node.GPUInfo, 0),
		LoadAverage: make([]float64, 3),
	}

	m := &MemoryResourceMonitor{
		interval:      config.Interval,
		llamacppPaths: config.LlamacppPaths,
		resources:     resources,
		gpuInfo:       make([]node.GPUInfo, 0),
		llamacppInfo:  nil,
		metrics:       make([]node.NodeMetrics, 0, config.MaxMetrics),
		maxMetrics:    config.MaxMetrics,
		ctx:           ctx,
		cancel:        cancel,
		log:           config.Logger,
		startTime:     time.Now(),
	}

	// 向后兼容：支持单个回调
	if config.Callback != nil {
		m.callbacks = append(m.callbacks, config.Callback)
	}

	return m
}

// Start starts the resource monitor
func (m *MemoryResourceMonitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("资源监控器已在运行")
	}

	m.running = true
	m.lastUpdate = time.Time{}

	// 初始化资源信息
	if err := m.initializeResources(); err != nil {
		m.log.Errorf("初始化资源信息失败: %v", err)
		// 不返回错误，继续运行
	}

	m.wg.Add(1)
	go m.monitorLoop()

	m.log.Infof("资源监控器已启动，采样间隔: %v", m.interval)
	return nil
}

// Stop stops the resource monitor
func (m *MemoryResourceMonitor) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}

	m.running = false
	m.mu.Unlock()

	// 先取消context，让monitorLoop能够退出
	m.cancel()
	m.wg.Wait()

	m.log.Infof("资源监控器已停止")
	return nil
}

// GetResources returns the current resource snapshot
func (m *MemoryResourceMonitor) GetResources() *node.NodeResources {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回资源数据的副本
	if m.resources == nil {
		return nil
	}

	snapshot := *m.resources
	if len(m.gpuInfo) > 0 {
		snapshot.GPUInfo = make([]node.GPUInfo, len(m.gpuInfo))
		copy(snapshot.GPUInfo, m.gpuInfo)
	}

	return &snapshot
}

// Watch adds a callback function to be called when resources are updated
func (m *MemoryResourceMonitor) Watch(callback func(*node.NodeResources)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callbacks = append(m.callbacks, callback)
}

// SetUpdateInterval sets the update interval for resource monitoring
func (m *MemoryResourceMonitor) SetUpdateInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.interval = interval
	m.log.Infof("资源采样间隔已更新: %v", interval)
}

// GetMetrics returns the current metrics
func (m *MemoryResourceMonitor) GetMetrics() *node.NodeMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.metrics) == 0 {
		return &node.NodeMetrics{
			Timestamp: time.Now(),
		}
	}

	// 返回最新的指标副本
	latest := m.metrics[len(m.metrics)-1]
	metricsCopy := latest
	return &metricsCopy
}

// GetMetricsHistory returns historical metrics up to the specified count
func (m *MemoryResourceMonitor) GetMetricsHistory(count int) []node.NodeMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.metrics) == 0 {
		return []node.NodeMetrics{}
	}

	// 限制请求的数量
	if count <= 0 || count > len(m.metrics) {
		count = len(m.metrics)
	}

	// 返回最近的 count 个指标
	start := len(m.metrics) - count
	result := make([]node.NodeMetrics, count)
	copy(result, m.metrics[start:])

	return result
}

// GetGPUInfo returns GPU information
func (m *MemoryResourceMonitor) GetGPUInfo() []node.GPUInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gpuInfo := make([]node.GPUInfo, len(m.gpuInfo))
	copy(gpuInfo, m.gpuInfo)
	return gpuInfo
}

// GetLlamacppInfo returns llama.cpp information
func (m *MemoryResourceMonitor) GetLlamacppInfo() *node.LlamacppInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.llamacppInfo == nil {
		return nil
	}

	info := *m.llamacppInfo
	return &info
}

// IsRunning returns whether the monitor is running
func (m *MemoryResourceMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetLastUpdateTime returns the last update time
func (m *MemoryResourceMonitor) GetLastUpdateTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastUpdate
}

// initializeResources 初始化资源信息
func (m *MemoryResourceMonitor) initializeResources() error {
	// 获取内存总量
	if vmStat, err := mem.VirtualMemory(); err == nil {
		m.resources.MemoryTotal = int64(vmStat.Total)
	} else {
		m.log.Errorf("获取内存总量失败: %v", err)
	}

	// 获取磁盘总量（根目录）
	if diskStat, err := disk.Usage("/"); err == nil {
		m.resources.DiskTotal = int64(diskStat.Total)
	} else {
		m.log.Errorf("获取磁盘总量失败: %v", err)
	}

	// 检测GPU
	m.detectGPUs()

	// 检测llama.cpp
	m.detectLlamacpp()

	return nil
}

// monitorLoop 监控循环
func (m *MemoryResourceMonitor) monitorLoop() {
	defer m.wg.Done()

	// 立即执行一次更新
	m.updateResources()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateResources()
		}
	}
}

// updateResources 更新资源信息
func (m *MemoryResourceMonitor) updateResources() {
	// 检查context是否已取消
	select {
	case <-m.ctx.Done():
		return
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastUpdate = time.Now()

	// 更新CPU使用率
	m.updateCPUUsage()

	// 更新内存使用情况
	m.updateMemoryUsage()

	// 更新磁盘使用情况
	m.updateDiskUsage()

	// 更新GPU信息
	m.updateGPUInfo()

	// 更新系统负载
	m.updateLoadAverage()

	// 更新运行时间
	m.resources.Uptime = int64(time.Since(m.startTime).Seconds())

	// 创建指标记录
	m.recordMetrics()

	// 调用回调函数（在锁外执行以避免死锁）
	var resourcesCopy *node.NodeResources
	if len(m.callbacks) > 0 {
		// 创建副本避免并发访问问题
		resourcesCopy = &node.NodeResources{}
		*resourcesCopy = *m.resources
		if len(m.gpuInfo) > 0 {
			resourcesCopy.GPUInfo = make([]node.GPUInfo, len(m.gpuInfo))
			copy(resourcesCopy.GPUInfo, m.gpuInfo)
		}
	}
	m.mu.Unlock()

	for _, callback := range m.callbacks {
		if resourcesCopy != nil {
			callback(resourcesCopy)
		}
	}

	// 重新获取锁以保持函数语义
	m.mu.Lock()
}

// recordMetrics 记录当前指标到历史记录
func (m *MemoryResourceMonitor) recordMetrics() {
	metrics := node.NodeMetrics{
		Timestamp:   m.lastUpdate,
		CPUUsage:    float64(m.resources.CPUUsed) / float64(m.resources.CPUTotal) * 100,
		MemoryUsage: float64(m.resources.MemoryUsed) / float64(m.resources.MemoryTotal) * 100,
		DiskUsage:   float64(m.resources.DiskUsed) / float64(m.resources.DiskTotal) * 100,
		NetworkRx:   m.resources.NetworkRx,
		NetworkTx:   m.resources.NetworkTx,
		Uptime:      m.resources.Uptime,
		LoadAverage: m.resources.LoadAverage,
	}

	m.metrics = append(m.metrics, metrics)

	// 限制历史记录数量
	if len(m.metrics) > m.maxMetrics {
		// 移除最旧的记录
		m.metrics = m.metrics[1:]
	}
}

// updateCPUUsage 更新CPU使用率
func (m *MemoryResourceMonitor) updateCPUUsage() {
	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		// 转换为 millicores
		m.resources.CPUUsed = int64(cpuPercent[0] * float64(m.resources.CPUTotal) / 100.0)
	} else {
		m.log.Errorf("获取CPU使用率失败: %v", err)
		m.resources.CPUUsed = 0
	}
}

// updateMemoryUsage 更新内存使用情况
func (m *MemoryResourceMonitor) updateMemoryUsage() {
	if vmStat, err := mem.VirtualMemory(); err == nil {
		m.resources.MemoryUsed = int64(vmStat.Used)
	} else {
		m.log.Errorf("获取内存使用情况失败: %v", err)
		m.resources.MemoryUsed = 0
	}
}

// updateDiskUsage 更新磁盘使用情况
func (m *MemoryResourceMonitor) updateDiskUsage() {
	if diskStat, err := disk.Usage("/"); err == nil {
		m.resources.DiskUsed = int64(diskStat.Used)
	} else {
		m.log.Errorf("获取磁盘使用情况失败: %v", err)
		m.resources.DiskUsed = 0
	}
}

// updateGPUInfo 更新GPU信息
func (m *MemoryResourceMonitor) updateGPUInfo() {
	// 定期重新检测GPU（每分钟）
	if len(m.gpuInfo) == 0 || time.Since(m.lastUpdate) > time.Minute {
		m.detectGPUs()
	}

	// 更新现有GPU的使用情况
	for i := range m.gpuInfo {
		switch m.gpuInfo[i].Vendor {
		case "NVIDIA":
			m.updateNvidiaGPU(&m.gpuInfo[i])
		case "AMD":
			m.updateAMDGPU(&m.gpuInfo[i])
		case "Intel":
			m.updateIntelGPU(&m.gpuInfo[i])
		}
	}
}

// updateLoadAverage 更新系统负载
func (m *MemoryResourceMonitor) updateLoadAverage() {
	if loadStat, err := load.Avg(); err == nil {
		m.resources.LoadAverage = []float64{loadStat.Load1, loadStat.Load5, loadStat.Load15}
	} else {
		m.log.Errorf("获取系统负载失败: %v", err)
		m.resources.LoadAverage = []float64{0, 0, 0}
	}
}

// detectGPUs 检测GPU
func (m *MemoryResourceMonitor) detectGPUs() {
	m.gpuInfo = m.gpuInfo[:0] // 清空现有数据

	// 检测NVIDIA GPU
	if nvidiaGPUs := m.detectNvidiaGPUs(); len(nvidiaGPUs) > 0 {
		m.gpuInfo = append(m.gpuInfo, nvidiaGPUs...)
	}

	// 检测AMD GPU
	if amdGPUs := m.detectAMDGPUs(); len(amdGPUs) > 0 {
		m.gpuInfo = append(m.gpuInfo, amdGPUs...)
	}

	// 检测Intel GPU
	if intelGPUs := m.detectIntelGPUs(); len(intelGPUs) > 0 {
		m.gpuInfo = append(m.gpuInfo, intelGPUs...)
	}

	if len(m.gpuInfo) > 0 {
		m.log.Infof("检测到 %d 个GPU", len(m.gpuInfo))
	}
}

// detectNvidiaGPUs 检测NVIDIA GPU
func (m *MemoryResourceMonitor) detectNvidiaGPUs() []node.GPUInfo {
	var gpus []node.GPUInfo

	// 尝试使用 nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total,memory.used,temperature.gpu,utilization.gpu,power.draw,driver_version", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		m.log.Debugf("未检测到NVIDIA GPU或nvidia-smi不可用: %v", err)
		return gpus
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 8 {
			continue
		}

		// 解析字段
		index, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		name := strings.TrimSpace(fields[1])
		totalMemory, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
		usedMemory, _ := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64)
		temperature, _ := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
		utilization, _ := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)
		powerUsage, _ := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64)
		driverVersion := strings.TrimSpace(fields[7])

		gpus = append(gpus, node.GPUInfo{
			Index:         index,
			Name:          name,
			Vendor:        "NVIDIA",
			TotalMemory:   totalMemory * 1024 * 1024, // MB to bytes
			UsedMemory:    usedMemory * 1024 * 1024,  // MB to bytes
			Temperature:   temperature,
			Utilization:   utilization,
			PowerUsage:    powerUsage,
			DriverVersion: driverVersion,
		})

		m.log.Debugf("检测到NVIDIA GPU[%d]: %s", index, name)
	}

	return gpus
}

// detectAMDGPUs 检测AMD GPU
func (m *MemoryResourceMonitor) detectAMDGPUs() []node.GPUInfo {
	var gpus []node.GPUInfo

	// 尝试使用 rocm-smi
	cmd := exec.Command("rocm-smi", "--showproductname")
	output, err := cmd.Output()
	if err != nil {
		m.log.Debugf("未检测到AMD GPU或rocm-smi不可用: %v", err)
		return gpus
	}

	// 解析AMD GPU信息（简化实现）
	lines := strings.Split(string(output), "\n")
	index := 0
	for _, line := range lines {
		if strings.Contains(line, "Card series") || strings.Contains(line, "GPU") {
			name := strings.TrimSpace(line)
			if name != "" {
				gpus = append(gpus, node.GPUInfo{
					Index:       index,
					Name:        name,
					Vendor:      "AMD",
					TotalMemory: 0, // ROCM-SMI 不提供这些信息
					UsedMemory:  0,
					Temperature: 0,
					Utilization: 0,
				})
				m.log.Debugf("检测到AMD GPU[%d]: %s", index, name)
				index++
			}
		}
	}

	return gpus
}

// detectIntelGPUs 检测Intel GPU
func (m *MemoryResourceMonitor) detectIntelGPUs() []node.GPUInfo {
	var gpus []node.GPUInfo

	// Intel GPU 检测通常通过系统信息
	if hostStat, err := host.Info(); err == nil {
		// 检查是否包含Intel集成显卡
		if strings.Contains(strings.ToLower(hostStat.KernelVersion), "i915") ||
			strings.Contains(strings.ToLower(hostStat.KernelVersion), "intel") {
			gpus = append(gpus, node.GPUInfo{
				Index:       0,
				Name:        "Intel Integrated Graphics",
				Vendor:      "Intel",
				TotalMemory: 0, // 集成显卡使用系统内存
				UsedMemory:  0,
				Temperature: 0,
				Utilization: 0,
			})
			m.log.Debugf("检测到Intel集成显卡")
		}
	}

	return gpus
}

// updateNvidiaGPU 更新NVIDIA GPU信息
func (m *MemoryResourceMonitor) updateNvidiaGPU(gpu *node.GPUInfo) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=memory.used,temperature.gpu,utilization.gpu,power.draw", "--format=csv,noheader,nounits", fmt.Sprintf("--id=%d", gpu.Index))
	output, err := cmd.Output()
	if err != nil {
		return
	}

	fields := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(fields) < 4 {
		return
	}

	if usedMemory, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 64); err == nil {
		gpu.UsedMemory = usedMemory * 1024 * 1024 // MB to bytes
	}
	if temperature, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64); err == nil {
		gpu.Temperature = temperature
	}
	if utilization, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64); err == nil {
		gpu.Utilization = utilization
	}
	if powerUsage, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64); err == nil {
		gpu.PowerUsage = powerUsage
	}
}

// updateAMDGPU 更新AMD GPU信息
func (m *MemoryResourceMonitor) updateAMDGPU(gpu *node.GPUInfo) {
	// AMD GPU 监控的具体实现取决于ROCm版本
	// 这里提供一个占位符实现
}

// updateIntelGPU 更新Intel GPU信息
func (m *MemoryResourceMonitor) updateIntelGPU(gpu *node.GPUInfo) {
	// Intel GPU 监控的具体实现
	// 这里提供一个占位符实现
}

// detectLlamacpp 检测llama.cpp
func (m *MemoryResourceMonitor) detectLlamacpp() {
	m.llamacppInfo = nil

	for _, path := range m.llamacppPaths {
		if info := m.testLlamacppPath(path); info != nil {
			m.llamacppInfo = info
			m.log.Infof("检测到llama.cpp: %s (版本: %s)", path, info.Version)
			return
		}
	}

	m.log.Debugf("未检测到有效的llama.cpp安装")
}

// testLlamacppPath 测试llama.cpp路径
func (m *MemoryResourceMonitor) testLlamacppPath(path string) *node.LlamacppInfo {
	// 检查文件是否存在且可执行
	cmd := exec.Command("test", "-x", path)
	if err := cmd.Run(); err != nil {
		return nil
	}

	info := &node.LlamacppInfo{
		Path:     path,
		Binaries: make(map[string]string),
	}

	// 获取版本信息
	cmd = exec.Command(path, "--version")
	if output, err := cmd.Output(); err == nil {
		versionStr := strings.TrimSpace(string(output))
		// 尝试解析版本字符串
		if strings.Contains(versionStr, "version") {
			parts := strings.Fields(versionStr)
			for i, part := range parts {
				if strings.ToLower(part) == "version" && i+1 < len(parts) {
					info.Version = parts[i+1]
					break
				}
			}
		} else {
			info.Version = versionStr
		}
	}

	// 检测GPU后端支持
	cmd = exec.Command(path, "--help")
	if output, err := cmd.Output(); err == nil {
		helpStr := strings.ToLower(string(output))
		if strings.Contains(helpStr, "cuda") {
			info.GPUBackend = "cuda"
			info.SupportsGPU = true
		} else if strings.Contains(helpStr, "metal") {
			info.GPUBackend = "metal"
			info.SupportsGPU = true
		} else if strings.Contains(helpStr, "opencl") {
			info.GPUBackend = "opencl"
			info.SupportsGPU = true
		} else if strings.Contains(helpStr, "rocm") || strings.Contains(helpStr, "hip") {
			info.GPUBackend = "rocm"
			info.SupportsGPU = true
		}

		// 检测支持的格式
		if strings.Contains(helpStr, "gguf") {
			info.SupportedFormats = append(info.SupportedFormats, "gguf")
		}
	}

	info.Available = true
	info.Binaries["main"] = path

	return info
}

// ClearMetrics 清空历史指标
func (m *MemoryResourceMonitor) ClearMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics = make([]node.NodeMetrics, 0, m.maxMetrics)
	m.log.Debug("历史指标已清空")
}

// GetStats returns monitor statistics
func (m *MemoryResourceMonitor) GetStats() *MonitorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &MonitorStats{
		Running:        m.running,
		StartTime:      m.startTime,
		LastUpdate:     m.lastUpdate,
		UpdateCount:    len(m.metrics),
		UpdateInterval: m.interval,
		GPUCount:       len(m.gpuInfo),
		LlamacppFound:  m.llamacppInfo != nil,
	}
}

// MonitorStats represents monitor statistics
type MonitorStats struct {
	Running        bool          `json:"running"`
	StartTime      time.Time     `json:"startTime"`
	LastUpdate     time.Time     `json:"lastUpdate"`
	UpdateCount    int           `json:"updateCount"`
	UpdateInterval time.Duration `json:"updateInterval"`
	GPUCount       int           `json:"gpuCount"`
	LlamacppFound  bool          `json:"llamacppFound"`
}
