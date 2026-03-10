// Package master provides master connection functionality for client nodes.
package master

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/client"
	"github.com/shepherd-project/shepherd/Shepherd/internal/cluster"
	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/gpu"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/netutil"
	"github.com/shepherd-project/shepherd/Shepherd/internal/node"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// Connector manages the connection to the master node
type Connector struct {
	nodeConfig  *config.NodeConfig
	clientInfo  *client.ClientInfo
	httpClient  *http.Client
	connected   bool
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	log         *logger.Logger
	gpuDetector *gpu.Detector
}

// NewConnector creates a new master connector
func NewConnector(cfg *config.NodeConfig, log *logger.Logger) (*Connector, error) {
	// Generate client info
	clientInfo, err := generateClientInfo(cfg)
	if err != nil {
		return nil, fmt.Errorf("生成客户端信息失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Connector{
		nodeConfig: cfg,
		clientInfo: clientInfo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		connected:   false,
		ctx:         ctx,
		cancel:      cancel,
		log:         log,
		gpuDetector: gpu.NewDetector(&gpu.Config{Logger: log}),
	}, nil
}

// Start starts the connector and connects to the master
func (c *Connector) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nodeConfig.ClientRole.MasterAddress == "" {
		return fmt.Errorf("master 地址未配置")
	}

	// Register with master
	if err := c.register(); err != nil {
		return fmt.Errorf("注册到 master 失败: %w", err)
	}

	c.connected = true
	c.log.Info(fmt.Sprintf("已连接到 Master: %s", c.nodeConfig.ClientRole.MasterAddress))

	// Start heartbeat
	c.wg.Add(1)
	go c.heartbeatLoop()

	return nil
}

// Stop stops the connector
func (c *Connector) Stop() {
	c.cancel()
	c.wg.Wait()

	c.mu.Lock()
	if c.connected {
		if err := c.unregister(); err != nil {
			logger.Warn(fmt.Sprintf("取消注册失败: %v", err))
		}
		c.connected = false
	}
	c.mu.Unlock()
}

// IsConnected returns whether the client is connected to the master
func (c *Connector) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetClientInfo returns the client information
func (c *Connector) GetClientInfo() *client.ClientInfo {
	return c.clientInfo
}

// register registers this client with the master
func (c *Connector) register() error {
	url := fmt.Sprintf("%s/api/master/clients/register", c.nodeConfig.ClientRole.MasterAddress)

	body, err := json.Marshal(c.clientInfo)
	if err != nil {
		return fmt.Errorf("序列化客户端信息失败: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("注册请求失败: %w", err)
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("注册失败: HTTP %d - %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// unregister unregisters this client from the master
func (c *Connector) unregister() error {
	url := fmt.Sprintf("%s/api/master/clients/%s", c.nodeConfig.ClientRole.MasterAddress, c.clientInfo.ID)

	req, err := http.NewRequestWithContext(context.Background(), "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer utils.CloseQuietly(resp.Body)

	return nil
}

// heartbeatLoop sends periodic heartbeats to the master
func (c *Connector) heartbeatLoop() {
	defer c.wg.Done()

	interval := time.Duration(c.nodeConfig.ClientRole.HeartbeatInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Send initial heartbeat
	if err := c.sendHeartbeat(); err != nil {
		logger.Warn(fmt.Sprintf("发送心跳失败: %v", err))
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				logger.Warn(fmt.Sprintf("发送心跳失败: %v", err))
			}
		}
	}
}

// sendHeartbeat sends a heartbeat to the master
func (c *Connector) sendHeartbeat() error {
	resources := c.getNodeResources()

	heartbeat := &node.HeartbeatMessage{
		NodeID:    c.clientInfo.ID,
		Timestamp: time.Now(),
		Status:    node.NodeStatusOnline,
		Resources: resources,
	}

	url := fmt.Sprintf("%s/api/master/heartbeat", c.nodeConfig.ClientRole.MasterAddress)
	body, err := json.Marshal(heartbeat)
	if err != nil {
		return fmt.Errorf("序列化心跳失败: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		return fmt.Errorf("发送心跳失败: %w", err)
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode == http.StatusOK {
		c.mu.Lock()
		if !c.connected {
			c.connected = true
			c.log.Info("重新连接到 Master")
		}
		c.mu.Unlock()
	}

	return nil
}

// getNodeResources gets current resource usage in NodeResources format
func (c *Connector) getNodeResources() *node.NodeResources {
	vmStat, _ := mem.VirtualMemory()
	hostInfo, _ := host.Info()

	// 获取CPU使用率
	var cpuPercent float64
	if cpuPercents, err := cpu.Percent(0, false); err == nil && len(cpuPercents) > 0 {
		cpuPercent = cpuPercents[0]
	}

	// 获取CPU核心数并计算millicores
	cpuCores := runtime.NumCPU()
	cpuTotal := int64(cpuCores * 1000) // millicores
	cpuUsed := int64(cpuPercent * float64(cpuTotal) / 100.0)

	// 获取磁盘使用情况
	var diskUsed, diskTotal int64
	if diskStat, err := disk.Usage("/"); err == nil {
		diskUsed = int64(diskStat.Used)
		diskTotal = int64(diskStat.Total)
	}

	// 获取 GPU 信息 (使用新的 gpu 包)
	var gpuInfo []gpu.Info
	if c.gpuDetector != nil {
		if gpus, err := c.gpuDetector.DetectAll(c.ctx); err == nil {
			gpuInfo = gpus
		}
	}

	return &node.NodeResources{
		CPUUsed:       cpuUsed,
		CPUTotal:      cpuTotal,
		MemoryUsed:    int64(vmStat.Used),
		MemoryTotal:   int64(vmStat.Total),
		DiskUsed:      diskUsed,
		DiskTotal:     diskTotal,
		GPUInfo:       gpuInfo,
		Uptime:        int64(hostInfo.Uptime),
		KernelVersion: hostInfo.KernelVersion,
		ROCmVersion:   c.detectROCmVersion(),
	}
}

// detectROCmVersion detects ROCm version
func (c *Connector) detectROCmVersion() string {
	// Try multiple methods to detect ROCm version
	// Priority: version file > hipcc path > rocm-smi-lib version > rocm-smi tool version

	// Method 1: Check /opt/rocm/.info/version (most reliable for ROCm platform version)
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

	// Method 2: Extract ROCm version from hipcc path (e.g., /opt/rocm-7.2.0/)
	cmd := exec.Command("hipcc", "--version")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// Look for "InstalledDir: /opt/rocm-7.2.0/..."
			if idx := strings.Index(line, "InstalledDir:"); idx != -1 {
				pathPart := line[idx+13:]
				// Extract version from path like /opt/rocm-7.2.0/
				if rocmIdx := strings.Index(pathPart, "rocm-"); rocmIdx != -1 {
					versionPart := pathPart[rocmIdx+5:]
					// Remove trailing path components
					if slashIdx := strings.IndexAny(versionPart, "/\t\n"); slashIdx != -1 {
						versionPart = versionPart[:slashIdx]
					}
					versionPart = strings.TrimSpace(versionPart)
					if versionPart != "" && isValidVersion(versionPart) {
						return versionPart
					}
				}
			}
		}
	}

	// Method 3: rocm-smi --showversion (look for ROCM-SMI-LIB version which is more accurate)
	cmd = exec.Command("rocm-smi", "--showversion")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// Look for "ROCM-SMI-LIB version:" which is closer to ROCm platform version
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

	// Method 4: rocm-smi --version (tool version - least preferred)
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

// isValidVersion checks if a string looks like a valid version number
func isValidVersion(s string) bool {
	// Must contain at least one dot (e.g., "7.2.0")
	if !strings.Contains(s, ".") {
		return false
	}
	// Must start with a digit
	if len(s) == 0 || s[0] < '0' || s[0] > '9' {
		return false
	}
	return true
}

// parseROCmVersionOutput parses ROCm version from rocm-smi --showversion output
func parseROCmVersionOutput(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for version patterns like "6.1.0" or "5.7.0-1234"
		if matched := extractVersionNumber(line); matched != "" {
			return matched
		}
	}
	return ""
}

// extractVersionNumber extracts a version number from a string
func extractVersionNumber(line string) string {
	// Match patterns like "6.1.0", "5.7.0-1234", "5.4.3+build"
	parts := strings.Fields(line)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Check if it looks like a version (contains digits and dots)
		if strings.Contains(part, ".") {
			digitCount := 0
			dotCount := 0
			for _, ch := range part {
				if ch >= '0' && ch <= '9' {
					digitCount++
				} else if ch == '.' {
					dotCount++
				} else if ch == '-' || ch == '+' {
					// Allow these in version strings
					continue
				} else if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' {
					// If we hit a letter that's not part of the version pattern, stop
					if digitCount > 0 {
						break
					}
				}
			}
			// Valid version has at least 2 digits and 1 dot (e.g., "6.1")
			if digitCount >= 2 && dotCount >= 1 {
				return part
			}
		}
	}
	return ""
}

// generateClientInfo generates client information
func generateClientInfo(cfg *config.NodeConfig) (*client.ClientInfo, error) {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Generate client ID if not provided
	clientID := cfg.ID
	if clientID == "" || clientID == "auto" {
		interfaces, _ := net.Interfaces()
		for _, iface := range interfaces {
			if iface.HardwareAddr != nil && len(iface.HardwareAddr) > 0 {
				clientID = iface.HardwareAddr.String()
				break
			}
		}
		if clientID == "" {
			clientID = hostname
		}
	}

	// Get client name
	clientName := cfg.Name
	if clientName == "" {
		clientName = hostname
	}

	// Get local IP address - 优先使用对外可访问的IP
	localIP := netutil.GetBestLocalIP()

	// Get system info
	hostInfo, _ := host.Info()
	vmStat, _ := mem.VirtualMemory()

	// 检测 GPU 信息
	gpuDetector := gpu.NewDetector(&gpu.Config{})
	var gpuCount int
	var gpuMemory int64
	var hasGPU bool
	var gpuList []gpu.Info

	if gpus, err := gpuDetector.DetectAll(context.Background()); err == nil {
		gpuList = gpus
		gpuCount = len(gpus)
		hasGPU = gpuCount > 0
		for _, g := range gpus {
			gpuMemory += g.TotalMemory
		}
		fmt.Printf("[DEBUG] GPU detection succeeded: found %d GPUs\n", gpuCount)
		for i, g := range gpus {
			fmt.Printf("[DEBUG] GPU[%d]: %s (%s), Memory: %d MB\n", i, g.Name, g.Vendor, g.TotalMemory/1024/1024)
		}
	} else {
		fmt.Printf("[DEBUG] GPU detection failed: %v\n", err)
	}

	// Build capabilities
	capabilities := &cluster.Capabilities{
		GPU:            hasGPU,
		CPUCount:       runtime.NumCPU(),
		Memory:         int64(vmStat.Total),
		GPUMemory:      gpuMemory,
		SupportsLlama:  true,
		SupportsPython: cfg.Capabilities.PythonEnabled,
	}

	// 添加 GPU 名称（如果有）
	if hasGPU && len(gpuList) > 0 {
		capabilities.GPUName = gpuList[0].Name
	}

	if cfg.Capabilities.PythonEnabled {
		envs := make([]string, 0, len(cfg.Capabilities.CondaEnvironments))
		for name := range cfg.Capabilities.CondaEnvironments {
			envs = append(envs, name)
		}
		capabilities.CondaEnvs = envs
	}

	// Build metadata - 合并配置的 metadata 和系统信息
	metadata := make(map[string]string)
	for k, v := range cfg.Metadata {
		metadata[k] = v
	}
	metadata["os"] = hostInfo.OS
	metadata["platform"] = hostInfo.Platform
	metadata["platformFamily"] = hostInfo.PlatformFamily
	metadata["platformVersion"] = hostInfo.PlatformVersion
	metadata["kernelVersion"] = hostInfo.KernelVersion
	metadata["kernelArch"] = hostInfo.KernelArch

	return &client.ClientInfo{
		ID:           clientID,
		Name:         clientName,
		Address:      localIP,
		Port:         9191, // Default client port
		Tags:         cfg.Tags,
		Capabilities: capabilities,
		Version:      "0.1.0-alpha",
		Metadata:     metadata,
	}, nil
}
