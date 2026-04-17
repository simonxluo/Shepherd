// Package types provides unified node type definitions
// 这个包提供统一的节点类型定义，消除不同模块间的类型重复
package types

import (
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/gpu"
)

// ==================== 节点能力 ====================

// NodeCapabilities describes what a node can do (unified)
// NodeCapabilities 描述节点可以做什么（统一类型）
type NodeCapabilities struct {
	GPU            bool     `json:"gpu"`
	GPUCount       int      `json:"gpuCount"`
	GPUName        string   `json:"gpuName,omitempty"`
	GPUNames       []string `json:"gpuNames,omitempty"`
	GPUMemory      int64    `json:"gpuMemory,omitempty"`
	CPUCount       int      `json:"cpuCount"`
	Memory         int64    `json:"memory"`
	SupportsLlama  bool     `json:"supportsLlama"`
	SupportsPython bool     `json:"supportsPython"`
	CondaEnvs      []string `json:"condaEnvs,omitempty"`
	DockerEnabled  bool     `json:"dockerEnabled,omitempty"`
}

// ==================== 节点资源 ====================

// NodeResources represents current resource usage and availability (unified)
// NodeResources 表示当前资源使用情况和可用性（统一类型）
type NodeResources struct {
	CPUUsed       int64      `json:"cpuUsed"`     // cores * 1000 (millicores)
	CPUTotal      int64      `json:"cpuTotal"`    // cores * 1000 (millicores)
	MemoryUsed    int64      `json:"memoryUsed"`  // bytes
	MemoryTotal   int64      `json:"memoryTotal"` // bytes
	DiskUsed      int64      `json:"diskUsed"`    // bytes
	DiskTotal     int64      `json:"diskTotal"`   // bytes
	GPUInfo       []gpu.Info `json:"gpuInfo,omitempty"`
	NetworkRx     int64      `json:"networkRx"`               // bytes per second
	NetworkTx     int64      `json:"networkTx"`               // bytes per second
	Uptime        int64      `json:"uptime"`                  // seconds
	LoadAverage   []float64  `json:"loadAverage,omitempty"`   // 1min, 5min, 15min
	ROCmVersion   string     `json:"rocmVersion,omitempty"`   // ROCm version
	KernelVersion string     `json:"kernelVersion,omitempty"` // Linux kernel version
}

// GPUInfo represents GPU information
// GPUInfo 表示 GPU 信息
type GPUInfo struct {
	Index         int     `json:"index"`
	Name          string  `json:"name"`
	Vendor        string  `json:"vendor"`
	TotalMemory   int64   `json:"totalMemory"` // bytes
	UsedMemory    int64   `json:"usedMemory"`  // bytes
	Temperature   float64 `json:"temperature"` // celsius
	Utilization   float64 `json:"utilization"` // percentage 0-100
	PowerUsage    float64 `json:"powerUsage"`  // watts
	DriverVersion string  `json:"driverVersion,omitempty"`
}

// ==================== 节点信息 ====================

// NodeInfo contains unified node/client information
// NodeInfo 包含统一的节点/客户端信息
type NodeInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Address      string            `json:"address"`
	Port         int               `json:"port"`
	Role         string            `json:"role"` // "standalone" | "master" | "client" | "hybrid"
	Status       NodeState         `json:"status"`
	Version      string            `json:"version"`
	Tags         []string          `json:"tags"`
	Capabilities *NodeCapabilities `json:"capabilities,omitempty"`
	Resources    *NodeResources    `json:"resources,omitempty"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	LastSeen     time.Time         `json:"lastSeen"`
	RegisteredAt time.Time         `json:"registeredAt,omitempty"` // Client registration time
}

// ==================== 心跳消息 ====================

// HeartbeatMessage represents a unified heartbeat message
// HeartbeatMessage 表示统一的心跳消息
type HeartbeatMessage struct {
	NodeID       string                 `json:"nodeId"`
	Timestamp    time.Time              `json:"timestamp"`
	Status       NodeState              `json:"status"`
	Role         string                 `json:"role"`
	Resources    *NodeResources         `json:"resources,omitempty"`
	Capabilities *NodeCapabilities      `json:"capabilities,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Sequence     int64                  `json:"sequence"`
}

// ==================== 命令类型 ====================

// CommandType represents the type of command
// CommandType 表示命令类型
type CommandType string

const (
	CommandTypeLoadModel    CommandType = "load_model"
	CommandTypeUnloadModel  CommandType = "unload_model"
	CommandTypeRunLlamacpp  CommandType = "run_llamacpp"
	CommandTypeStopProcess  CommandType = "stop_process"
	CommandTypeUpdateConfig CommandType = "update_config"
	CommandTypeCollectLogs  CommandType = "collect_logs"
	CommandTypeScanModels   CommandType = "scan_models"
	CommandTypeStartTask    CommandType = "start_task"
	CommandTypeStopTask     CommandType = "stop_task"
	CommandTypeRestart      CommandType = "restart"
	CommandTypeShutdown     CommandType = "shutdown"
	CommandTypeTestLlamacpp CommandType = "test_llamacpp"
	CommandTypeGetConfig    CommandType = "get_config"
)

// ==================== 命令 ====================

// Command represents a unified command structure
// Command 表示统一的命令结构
type Command struct {
	ID         string                 `json:"id"`
	Type       CommandType            `json:"type"`
	FromNodeID string                 `json:"fromNodeId"`
	ToNodeID   string                 `json:"toNodeId,omitempty"`
	Payload    map[string]interface{} `json:"payload"`
	CreatedAt  time.Time              `json:"createdAt"`
	Timeout    *time.Duration         `json:"timeout,omitempty"`
	Priority   int                    `json:"priority"` // 0=low, 5=normal, 10=high
	RetryCount int                    `json:"retryCount"`
	MaxRetries int                    `json:"maxRetries"`
}

// CommandResult represents the result of a command execution
// CommandResult 表示命令执行的结果
type CommandResult struct {
	CommandID   string                 `json:"commandId"`
	FromNodeID  string                 `json:"fromNodeId"`
	ToNodeID    string                 `json:"toNodeId"`
	Success     bool                   `json:"success"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CompletedAt time.Time              `json:"completedAt"`
	Duration    int64                  `json:"duration"` // milliseconds
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

// ==================== 节点配置信息 ====================

// NodeConfigInfo contains node configuration information reported by client
// NodeConfigInfo 包含 Client 上报的节点配置信息
type NodeConfigInfo struct {
	// LlamaCppPaths llama.cpp 可执行文件路径列表
	LlamaCppPaths []LlamaCppPathInfo `json:"llamaCppPaths,omitempty"`
	// ModelPaths 模型存储路径列表
	ModelPaths []ModelPathInfo `json:"modelPaths,omitempty"`
	// Environment 环境信息
	Environment *EnvironmentInfo `json:"environment,omitempty"`
	// Conda 配置
	Conda *CondaConfigInfo `json:"conda,omitempty"`
	// Executor 执行器配置
	Executor *ExecutorConfigInfo `json:"executor,omitempty"`
	// 报告时间
	ReportedAt time.Time `json:"reportedAt"`
}

// LlamaCppPathInfo represents llama.cpp binary path information
type LlamaCppPathInfo struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Exists      bool   `json:"exists"`
	Version     string `json:"version,omitempty"`
}

// ModelPathInfo represents model directory path information
type ModelPathInfo struct {
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Exists      bool   `json:"exists"`
	ModelCount  int    `json:"modelCount,omitempty"`
}

// EnvironmentInfo contains environment information
type EnvironmentInfo struct {
	OS              string `json:"os"`
	Architecture    string `json:"architecture"`
	KernelVersion   string `json:"kernelVersion,omitempty"`
	ROCmVersion     string `json:"rocmVersion,omitempty"`
	CUDAVersion     string `json:"cudaVersion,omitempty"`
	PythonVersion   string `json:"pythonVersion,omitempty"`
	GoVersion       string `json:"goVersion,omitempty"`
	LlamaCppVersion string `json:"llamaCppVersion,omitempty"`
}

// CondaConfigInfo contains conda environment configuration
type CondaConfigInfo struct {
	Enabled      bool              `json:"enabled"`
	CondaPath    string            `json:"condaPath,omitempty"`
	Environments map[string]string `json:"environments,omitempty"` // name -> path
}

// ExecutorConfigInfo contains executor configuration
type ExecutorConfigInfo struct {
	MaxConcurrent   int      `json:"maxConcurrent"`
	TaskTimeout     int      `json:"taskTimeout"`
	AllowRemoteStop bool     `json:"allowRemoteStop"`
	AllowedCommands []string `json:"allowedCommands,omitempty"`
}

// LlamacppTestResult represents llama.cpp availability test result
// LlamacppTestResult 表示 llama.cpp 可用性测试结果
type LlamacppTestResult struct {
	Success    bool      `json:"success"`
	BinaryPath string    `json:"binaryPath,omitempty"`
	Version    string    `json:"version,omitempty"`
	Output     string    `json:"output,omitempty"`
	Error      string    `json:"error,omitempty"`
	TestedAt   time.Time `json:"testedAt"`
	Duration   int64     `json:"duration"` // milliseconds
}
