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

// ResourceMonitor 负责收集节点的资源信息
type ResourceMonitor struct {
	// 配置参数
	interval      time.Duration        // 采样间隔
	llamacppPaths []string             // llama.cpp 可执行文件路径
	callback      func(*NodeResources) // 资源更新回调函数

	// 运行时状态
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	mu         sync.RWMutex
	lastUpdate time.Time
	startTime  time.Time

	// 资源数据
	resources    *NodeResources
	gpuInfo      []gpu.Info
	llamacppInfo *LlamacppInfo

	// GPU检测器 (使用新的gpu包)
	gpuDetector *gpu.Detector

	// 日志
	log *logger.Logger
}

// ResourceMonitorConfig 资源监控器配置
type ResourceMonitorConfig struct {
	Interval      time.Duration        // 采样间隔，默认5秒
	LlamacppPaths []string             // llama.cpp 可执行文件路径
	Callback      func(*NodeResources) // 资源更新回调
	Logger        *logger.Logger
}

// NewResourceMonitor 创建新的资源监控器
func NewResourceMonitor(config *ResourceMonitorConfig) *ResourceMonitor {
	if config == nil {
		config = &ResourceMonitorConfig{}
	}

	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 初始化资源数据
	resources := &NodeResources{
		CPUTotal:    int64(runtime.NumCPU()) * 1000, // 转换为 millicores
		MemoryTotal: 0,                              // 将在初始化时设置
		DiskTotal:   0,                              // 将在初始化时设置
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

// Start 启动资源监控
func (rm *ResourceMonitor) Start() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.running {
		return fmt.Errorf("资源监控器已在运行")
	}

	rm.running = true
	rm.lastUpdate = time.Time{}

	// 初始化资源信息
	if err := rm.initializeResources(); err != nil {
		rm.log.Errorf("初始化资源信息失败: %v", err)
		// 不返回错误，继续运行
	}

	rm.wg.Add(1)
	go rm.monitorLoop()

	if rm.log != nil {
		rm.log.Infof("资源监控器已启动，采样间隔: %v", rm.interval)
	}
	return nil
}

// Stop 停止资源监控
func (rm *ResourceMonitor) Stop() error {
	rm.mu.Lock()
	if !rm.running {
		rm.mu.Unlock()
		return nil
	}

	rm.running = false
	rm.mu.Unlock()

	// 先取消context，让monitorLoop能够退出
	rm.cancel()
	rm.wg.Wait()

	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.log != nil {
		rm.log.Infof("资源监控器已停止")
	}
	return nil
}

// GetSnapshot 获取当前资源快照
func (rm *ResourceMonitor) GetSnapshot() *NodeResources {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// 返回资源数据的副本
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

// GetGPUInfo 获取GPU信息
func (rm *ResourceMonitor) GetGPUInfo() []gpu.Info {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	gpuInfo := make([]gpu.Info, len(rm.gpuInfo))
	copy(gpuInfo, rm.gpuInfo)
	return gpuInfo
}

// GetLlamacppInfo 获取llama.cpp信息
func (rm *ResourceMonitor) GetLlamacppInfo() *LlamacppInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rm.llamacppInfo == nil {
		return nil
	}

	info := *rm.llamacppInfo
	return &info
}

// initializeResources 初始化资源信息
func (rm *ResourceMonitor) initializeResources() error {
	// 获取内存总量
	if vmStat, err := mem.VirtualMemory(); err == nil {
		rm.resources.MemoryTotal = int64(vmStat.Total)
	} else {
		if rm.log != nil {
			rm.log.Errorf("获取内存总量失败: %v", err)
		}
	}

	// 获取磁盘总量（根目录）
	if diskStat, err := disk.Usage("/"); err == nil {
		rm.resources.DiskTotal = int64(diskStat.Total)
	} else {
		if rm.log != nil {
			rm.log.Errorf("获取磁盘总量失败: %v", err)
		}
	}

	// 检测GPU
	rm.detectGPUs()

	// 检测ROCm版本
	rm.resources.ROCmVersion = rm.detectROCmVersion()

	// 检测内核版本
	rm.resources.KernelVersion = rm.detectKernelVersion()

	// 检测llama.cpp
	rm.detectLlamacpp()

	return nil
}

// monitorLoop 监控循环
func (rm *ResourceMonitor) monitorLoop() {
	defer rm.wg.Done()

	// 立即执行一次更新
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

// updateResources 更新资源信息
func (rm *ResourceMonitor) updateResources() {
	// 检查context是否已取消
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

		// 更新CPU使用率
		rm.updateCPUUsage()

		// 更新内存使用情况
		rm.updateMemoryUsage()

		// 更新磁盘使用情况
		rm.updateDiskUsage()

		// 更新GPU信息
		rm.updateGPUInfo()

		// 更新系统负载
		rm.updateLoadAverage()

		// 更新运行时间
		rm.resources.Uptime = int64(time.Since(rm.startTime).Seconds())

		// 创建副本避免并发访问问题
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

// updateCPUUsage 更新CPU使用率
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

// updateMemoryUsage 更新内存使用情况
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

// updateDiskUsage 更新磁盘使用情况
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

// updateGPUInfo 更新GPU信息 (使用新的gpu包)
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

// updateLoadAverage 更新系统负载
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

// detectGPUs 检测GPU (使用新的gpu包)
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

// detectLlamacpp 检测llama.cpp
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

// testLlamacppPath 测试llama.cpp路径
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

// detectROCmVersion 检测ROCm版本
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

// isValidROCmVersion 检查字符串是否是有效的版本号
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

// detectKernelVersion 检测Linux内核版本
func (rm *ResourceMonitor) detectKernelVersion() string {
	if hostStat, err := host.Info(); err == nil {
		return hostStat.KernelVersion
	}
	return ""
}
