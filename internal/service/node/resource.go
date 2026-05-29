package node

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/gpu"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// ResourceMonitor is responsible for collecting node resource information.
type ResourceMonitor struct {
	// Configuration
	interval      time.Duration        // Sampling interval
	llamacppPaths []string             // llama.cpp executable paths
	callback      func(*NodeResources) // Resource update callback

	// Runtime state
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	mu         sync.RWMutex
	lastUpdate time.Time
	startTime  time.Time

	// Resource data
	resources    *NodeResources
	gpuInfo      []gpu.Info
	llamacppInfo *LlamacppInfo

	// GPU detector (using the gpu package)
	gpuDetector *gpu.Detector

	// Logger
	log *logger.Logger
}

// ResourceMonitorConfig holds configuration for the resource monitor.
type ResourceMonitorConfig struct {
	Interval      time.Duration        // Sampling interval, default 5s
	LlamacppPaths []string             // llama.cpp executable paths
	Callback      func(*NodeResources) // Resource update callback
	Logger        *logger.Logger
}

// NewResourceMonitor creates a new resource monitor.
func NewResourceMonitor(config *ResourceMonitorConfig) *ResourceMonitor {
	if config == nil {
		config = &ResourceMonitorConfig{}
	}

	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize resource data
	resources := &NodeResources{
		CPUTotal:    int64(runtime.NumCPU()) * 1000, // convert to millicores
		MemoryTotal: 0,                              // will be set during initialization
		DiskTotal:   0,                              // will be set during initialization
		GPUInfo:     make([]gpu.Info, 0),
		LoadAverage: make([]float64, 3),
	}

	rm := &ResourceMonitor{
		interval:      config.Interval,
		llamacppPaths: config.LlamacppPaths,
		callback:      config.Callback,
		ctx:           ctx,
		cancel:        cancel,
		resources:     resources,
		gpuInfo:       make([]gpu.Info, 0),
		llamacppInfo:  nil,
		gpuDetector:   gpu.NewDetector(&gpu.Config{}),
		log:           config.Logger,
		startTime:     time.Now(),
	}

	return rm
}

// Start starts the resource monitor.
func (rm *ResourceMonitor) Start() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.running {
		return fmt.Errorf("资源监控器已在运行")
	}

	rm.running = true
	rm.lastUpdate = time.Time{}

	// Initialize resource information
	if err := rm.initializeResources(); err != nil {
		rm.log.Errorf("初始化资源信息失败: %v", err)
		// Don't return error, continue running
	}

	rm.wg.Add(1)
	go rm.monitorLoop()

	if rm.log != nil {
		rm.log.Infof("资源监控器已启动，采样间隔: %v", rm.interval)
	}
	return nil
}

// Stop stops the resource monitor.
func (rm *ResourceMonitor) Stop() error {
	rm.mu.Lock()
	if !rm.running {
		rm.mu.Unlock()
		return nil
	}

	rm.running = false
	rm.mu.Unlock()

	// Cancel context first so monitorLoop can exit
	rm.cancel()
	rm.wg.Wait()

	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.log != nil {
		rm.log.Infof("资源监控器已停止")
	}
	return nil
}

// GetSnapshot returns the current resource snapshot.
func (rm *ResourceMonitor) GetSnapshot() *NodeResources {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return a copy of the resource data
	if rm.resources == nil {
		return nil
	}

	snapshot := *rm.resources
	if len(rm.gpuInfo) > 0 {
		snapshot.GPUInfo = make([]gpu.Info, len(rm.gpuInfo))
		copy(snapshot.GPUInfo, rm.gpuInfo)
	}

	return &snapshot
}

// GetGPUInfo returns GPU information.
func (rm *ResourceMonitor) GetGPUInfo() []gpu.Info {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	gpuInfo := make([]gpu.Info, len(rm.gpuInfo))
	copy(gpuInfo, rm.gpuInfo)
	return gpuInfo
}

// GetLlamacppInfo returns llama.cpp installation information.
func (rm *ResourceMonitor) GetLlamacppInfo() *LlamacppInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rm.llamacppInfo == nil {
		return nil
	}

	info := *rm.llamacppInfo
	return &info
}

// initializeResources initializes resource information.
func (rm *ResourceMonitor) initializeResources() error {
	// Get total memory
	if vmStat, err := mem.VirtualMemory(); err == nil {
		rm.resources.MemoryTotal = int64(vmStat.Total)
	} else {
		if rm.log != nil {
			rm.log.Errorf("获取内存总量失败: %v", err)
		}
	}

	// Get total disk (root partition)
	if diskStat, err := disk.Usage("/"); err == nil {
		rm.resources.DiskTotal = int64(diskStat.Total)
	} else {
		if rm.log != nil {
			rm.log.Errorf("获取磁盘总量失败: %v", err)
		}
	}

	// Get CPU model
	if cpuInfos, err := cpu.Info(); err == nil && len(cpuInfos) > 0 {
		rm.resources.CPUModel = cpuInfos[0].ModelName
	}

	// Get platform information
	rm.resources.Platform = runtime.GOOS
	rm.resources.Arch = runtime.GOARCH
	if hostname, err := os.Hostname(); err == nil {
		rm.resources.Hostname = hostname
	}

	// Get host IP
	rm.resources.HostIP = rm.detectHostIP()

	// Detect GPUs
	rm.detectGPUs()

	// Detect ROCm version
	rm.resources.ROCmVersion = rm.detectROCmVersion()

	// Detect kernel version
	rm.resources.KernelVersion = rm.detectKernelVersion()

	// Detect llama.cpp
	rm.detectLlamacpp()

	return nil
}

// monitorLoop is the main monitoring loop.
func (rm *ResourceMonitor) monitorLoop() {
	defer rm.wg.Done()

	// Execute an update immediately
	rm.updateResources()

	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.updateResources()
		}
	}
}

// updateResources updates resource information.
func (rm *ResourceMonitor) updateResources() {
	// Check if context has been cancelled
	select {
	case <-rm.ctx.Done():
		return
	default:
	}

	var resourcesCopy *NodeResources

	func() {
		rm.mu.Lock()
		defer rm.mu.Unlock()

		rm.lastUpdate = time.Now()

		// Update CPU usage
		rm.updateCPUUsage()

		// Update memory usage
		rm.updateMemoryUsage()

		// Update disk usage
		rm.updateDiskUsage()

		// Update GPU information
		rm.updateGPUInfo()

		// Update system load average
		rm.updateLoadAverage()

		// Update uptime
		rm.resources.Uptime = int64(time.Since(rm.startTime).Seconds())

		// Create a copy to avoid concurrent access issues
		if rm.callback != nil {
			resourcesCopy = &NodeResources{}
			*resourcesCopy = *rm.resources
			if len(rm.gpuInfo) > 0 {
				resourcesCopy.GPUInfo = make([]gpu.Info, len(rm.gpuInfo))
				copy(resourcesCopy.GPUInfo, rm.gpuInfo)
			}
		}
	}()

	if rm.callback != nil && resourcesCopy != nil {
		rm.callback(resourcesCopy)
	}
}

// updateCPUUsage updates CPU usage metrics.
func (rm *ResourceMonitor) updateCPUUsage() {
	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		// 转换为 millicores
		rm.resources.CPUUsed = int64(cpuPercent[0] * float64(rm.resources.CPUTotal) / 100.0)
	} else {
		if rm.log != nil {
			rm.log.Errorf("获取CPU使用率失败: %v", err)
		}
		rm.resources.CPUUsed = 0
	}
}

// updateMemoryUsage updates memory usage metrics.
func (rm *ResourceMonitor) updateMemoryUsage() {
	if vmStat, err := mem.VirtualMemory(); err == nil {
		rm.resources.MemoryUsed = int64(vmStat.Used)
	} else {
		if rm.log != nil {
			rm.log.Errorf("获取内存使用情况失败: %v", err)
		}
		rm.resources.MemoryUsed = 0
	}
}

// updateDiskUsage updates disk usage metrics.
func (rm *ResourceMonitor) updateDiskUsage() {
	if diskStat, err := disk.Usage("/"); err == nil {
		rm.resources.DiskUsed = int64(diskStat.Used)
	} else {
		if rm.log != nil {
			rm.log.Errorf("获取磁盘使用情况失败: %v", err)
		}
		rm.resources.DiskUsed = 0
	}
}

// updateGPUInfo updates GPU information (using the gpu package).
func (rm *ResourceMonitor) updateGPUInfo() {
	if rm.gpuDetector == nil {
		return
	}

	// 定期重新检测GPU（每分钟）
	if len(rm.gpuInfo) == 0 || time.Since(rm.lastUpdate) > time.Minute {
		rm.detectGPUs()
	}

	// 更新现有GPU的使用情况
	for i := range rm.gpuInfo {
		if err := rm.gpuDetector.Update(rm.ctx, &rm.gpuInfo[i]); err != nil {
			if rm.log != nil {
				rm.log.Debugf("更新GPU[%d]信息失败: %v", rm.gpuInfo[i].Index, err)
			}
		}
	}
}

// updateLoadAverage updates system load average.
func (rm *ResourceMonitor) updateLoadAverage() {
	if loadStat, err := load.Avg(); err == nil {
		rm.resources.LoadAverage = []float64{loadStat.Load1, loadStat.Load5, loadStat.Load15}
	} else {
		if rm.log != nil {
			rm.log.Errorf("获取系统负载失败: %v", err)
		}
		rm.resources.LoadAverage = []float64{0, 0, 0}
	}
}

// detectGPUs detects GPUs (using the gpu package).
func (rm *ResourceMonitor) detectGPUs() {
	if rm.gpuDetector == nil {
		return
	}

	gpus, err := rm.gpuDetector.DetectAll(rm.ctx)
	if err != nil {
		if rm.log != nil {
			rm.log.Errorf("GPU检测失败: %v", err)
		}
		return
	}

	rm.gpuInfo = gpus
}

// detectLlamacpp detects llama.cpp installation.
func (rm *ResourceMonitor) detectLlamacpp() {
	rm.llamacppInfo = nil

	for _, path := range rm.llamacppPaths {
		if info := rm.testLlamacppPath(path); info != nil {
			rm.llamacppInfo = info
			if rm.log != nil {
				rm.log.Infof("检测到llama.cpp: %s (版本: %s)", path, info.Version)
			}
			return
		}
	}

	if rm.log != nil {
		rm.log.Debugf("未检测到有效的llama.cpp安装")
	}
}

// testLlamacppPath tests a llama.cpp binary path.
func (rm *ResourceMonitor) testLlamacppPath(path string) *LlamacppInfo {
	// 检查文件是否存在且可执行
	cmd := exec.Command("test", "-x", path)
	if err := cmd.Run(); err != nil {
		return nil
	}

	info := &LlamacppInfo{
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

// detectROCmVersion detects the ROCm version.
func (rm *ResourceMonitor) detectROCmVersion() string {
	// 检测优先级：version文件 > hipcc路径 > rocm-smi-lib版本 > rocm-smi工具版本

	// 方法1: 读取 /opt/rocm/.info/version (最可靠的ROCm平台版本)
	if data, err := os.ReadFile("/opt/rocm/.info/version"); err == nil {
		version := strings.TrimSpace(string(data))
		if version != "" {
			return version
		}
	}
	if data, err := os.ReadFile("/opt/rocm/bin/.info/version"); err == nil {
		version := strings.TrimSpace(string(data))
		if version != "" {
			return version
		}
	}

	// 方法2: 从 hipcc 路径提取版本 (例如 /opt/rocm-7.2.0/)
	cmd := exec.Command("hipcc", "--version")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// 查找 "InstalledDir: /opt/rocm-7.2.0/..."
			if idx := strings.Index(line, "InstalledDir:"); idx != -1 {
				pathPart := line[idx+13:]
				// 从路径中提取版本，如 /opt/rocm-7.2.0/
				if rocmIdx := strings.Index(pathPart, "rocm-"); rocmIdx != -1 {
					versionPart := pathPart[rocmIdx+5:]
					// 移除后续路径组件
					if slashIdx := strings.IndexAny(versionPart, "/\t\n"); slashIdx != -1 {
						versionPart = versionPart[:slashIdx]
					}
					versionPart = strings.TrimSpace(versionPart)
					if versionPart != "" && isValidROCmVersion(versionPart) {
						return versionPart
					}
				}
			}
		}
	}

	// 方法3: rocm-smi --showversion (查找 ROCM-SMI-LIB 版本，更接近平台版本)
	cmd = exec.Command("rocm-smi", "--showversion")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// 查找 "ROCM-SMI-LIB version:"，这个更接近ROCm平台版本
			if strings.Contains(line, "ROCM-SMI-LIB version:") {
				parts := strings.Split(line, "ROCM-SMI-LIB version:")
				if len(parts) >= 2 {
					version := strings.TrimSpace(parts[1])
					if version != "" {
						return version
					}
				}
			}
		}
	}

	// 方法4: rocm-smi --version (工具版本 - 最低优先级)
	cmd = exec.Command("rocm-smi", "--version")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "ROCM-SMI version:") {
				parts := strings.Split(line, "ROCM-SMI version:")
				if len(parts) >= 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}

	return ""
}

// isValidROCmVersion checks if a string is a valid version number.
func isValidROCmVersion(s string) bool {
	// 必须包含至少一个点 (例如 "7.2.0")
	if !strings.Contains(s, ".") {
		return false
	}
	// 必须以数字开头
	if len(s) == 0 || s[0] < '0' || s[0] > '9' {
		return false
	}
	return true
}

// detectKernelVersion detects the Linux kernel version.
func (rm *ResourceMonitor) detectKernelVersion() string {
	if hostStat, err := host.Info(); err == nil {
		return hostStat.KernelVersion
	}
	return ""
}

// detectHostIP returns the primary non-loopback IP address.
func (rm *ResourceMonitor) detectHostIP() string {
	cmd := exec.CommandContext(rm.ctx, "hostname", "-I")
	out, err := cmd.Output()
	if err == nil {
		parts := strings.Fields(string(out))
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}
