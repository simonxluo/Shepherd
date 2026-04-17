// Package node provides distributed node management types and interfaces.
package node

import (
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/gpu"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
)

// NodeRole represents the role of a node in the distributed architecture
type NodeRole string

const (
	NodeRoleMaster NodeRole = "master"
	NodeRoleClient NodeRole = "client"
	NodeRoleHybrid NodeRole = "hybrid"
)

// NodeStatus is an alias for the unified NodeState type
// NodeStatus 是统一 NodeState 类型的别名，保持向后兼容
type NodeStatus = types.NodeState

// NodeStatus constants - 使用统一的 NodeState 常量
const (
	NodeStatusOffline  NodeStatus = types.StateOffline
	NodeStatusOnline   NodeStatus = types.StateOnline
	NodeStatusBusy     NodeStatus = types.StateBusy
	NodeStatusError    NodeStatus = types.StateError
	NodeStatusDegraded NodeStatus = types.StateDegraded
	NodeStatusDisabled NodeStatus = types.StateDisabled
)

// 向后兼容：旧的代码可以继续使用 NodeStatus，实际上使用的是统一的 NodeState

// NodeCapabilities describes what a node can do
type NodeCapabilities struct {
	GPU               bool              `json:"gpu"`
	GPUCount          int               `json:"gpuCount"`
	GPUName           string            `json:"gpuName,omitempty"`
	GPUNames          []string          `json:"gpuNames,omitempty"`
	GPUMemory         int64             `json:"gpuMemory,omitempty"`
	CPUCount          int               `json:"cpuCount"`
	Memory            int64             `json:"memory"` // bytes
	SupportsLlama     bool              `json:"supportsLlama"`
	SupportsPython    bool              `json:"supportsPython"`
	PythonVersion     string            `json:"pythonVersion,omitempty"`
	CondaPath         string            `json:"condaPath,omitempty"`
	CondaEnvironments map[string]string `json:"condaEnvironments,omitempty"`
	CondaEnvs         []string          `json:"condaEnvs,omitempty"` // 已废弃，使用 CondaEnvironments
	DockerEnabled     bool              `json:"dockerEnabled,omitempty"`
}

// NodeResources represents current resource usage and availability
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
	ROCmVersion   string     `json:"rocmVersion,omitempty"`   // ROCm version (if AMD GPU)
	KernelVersion string     `json:"kernelVersion,omitempty"` // Linux kernel version
}

// GPUInfo is an alias for gpu.Info for backward compatibility.
// Deprecated: Use gpu.Info directly instead.
type GPUInfo = gpu.Info

// LlamacppInfo contains information about llama.cpp installation
type LlamacppInfo struct {
	Path             string            `json:"path"`
	Version          string            `json:"version"`
	BuildType        string            `json:"buildType"`            // debug, release
	GPUBackend       string            `json:"gpuBackend,omitempty"` // cuda, metal, opencl, etc.
	SupportsGPU      bool              `json:"supportsGPU"`
	Available        bool              `json:"available"`
	SupportedFormats []string          `json:"supportedFormats,omitempty"`
	Binaries         map[string]string `json:"binaries,omitempty"` // binary name -> path
}

// ModelInfo contains information about a model
type ModelInfo struct {
	Path         string            `json:"path"`
	Name         string            `json:"name"`
	Alias        string            `json:"alias,omitempty"`
	Size         int64             `json:"size"`   // bytes
	Format       string            `json:"format"` // gguf, etc.
	Architecture string            `json:"architecture,omitempty"`
	Parameters   string            `json:"parameters,omitempty"`   // 7B, 13B, etc.
	Quantization string            `json:"quantization,omitempty"` // q4_0, q5_1, etc.
	ContextSize  int               `json:"contextSize,omitempty"`
	Embedding    bool              `json:"embedding"`
	Tags         []string          `json:"tags,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Loaded       bool              `json:"loaded"`
	LoadedAt     *time.Time        `json:"loadedAt,omitempty"`
	LoadedBy     string            `json:"loadedBy,omitempty"` // node ID
}

// CommandType represents the type of command
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

// Command represents a command sent between nodes
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

// HeartbeatMessage represents a heartbeat message between nodes
type HeartbeatMessage struct {
	NodeID       string                 `json:"nodeId"`
	Timestamp    time.Time              `json:"timestamp"`
	Status       NodeStatus             `json:"status"`
	Role         NodeRole               `json:"role"`
	Resources    *NodeResources         `json:"resources,omitempty"`
	Capabilities *NodeCapabilities      `json:"capabilities,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Sequence     int64                  `json:"sequence"`
}

// HealthStatus represents the health status of a node
type HealthStatus struct {
	NodeID       string          `json:"nodeId"`
	Status       NodeStatus      `json:"status"`
	Healthy      bool            `json:"healthy"`
	LastSeen     time.Time       `json:"lastSeen"`
	ResponseTime int64           `json:"responseTime"`     // milliseconds
	Checks       map[string]bool `json:"checks,omitempty"` // check name -> result
	Issues       []string        `json:"issues,omitempty"`
	Warnings     []string        `json:"warnings,omitempty"`
}

// NodeInfo contains basic information about a node
type NodeInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Address      string            `json:"address"`
	Port         int               `json:"port"`
	Role         NodeRole          `json:"role"`
	Status       NodeStatus        `json:"status"`
	Version      string            `json:"version"`
	Tags         []string          `json:"tags"`
	Capabilities *NodeCapabilities `json:"capabilities,omitempty"`
	Resources    *NodeResources    `json:"resources,omitempty"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	LastSeen     time.Time         `json:"lastSeen"`
	RegisteredAt time.Time         `json:"registeredAt"` // 注册时间（客户端）
}

// NodeConfig contains configuration for a node
type NodeConfig struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Role                NodeRole          `json:"role"`
	Address             string            `json:"address"`
	Port                int               `json:"port"`
	MasterAddress       string            `json:"masterAddress,omitempty"`
	AdvertiseAddress    string            `json:"advertiseAddress,omitempty"`
	HeartbeatInterval   time.Duration     `json:"heartbeatInterval"`
	HealthCheckInterval time.Duration     `json:"healthCheckInterval"`
	MaxRetries          int               `json:"maxRetries"`
	Timeout             time.Duration     `json:"timeout"`
	EnableMetrics       bool              `json:"enableMetrics"`
	LogLevel            string            `json:"logLevel"`
	DataDir             string            `json:"dataDir"`
	TempDir             string            `json:"tempDir"`
	MaxMemoryUsage      int64             `json:"maxMemoryUsage"` // bytes
	MaxCPUUsage         float64           `json:"maxCPUUsage"`    // percentage
	MaxGpuMemory        int64             `json:"maxGpuMemory"`   // bytes
	Tags                []string          `json:"tags"`           // 节点标签
	Metadata            map[string]string `json:"metadata"`       // 节点元数据
	Capabilities        *NodeCapabilities `json:"capabilities"`   // 节点能力
	Version             string            `json:"version"`
}

// NodeConnection represents a connection between nodes
type NodeConnection struct {
	FromNodeID    string    `json:"fromNodeId"`
	ToNodeID      string    `json:"toNodeId"`
	Status        string    `json:"status"` // connected, disconnected, error
	ConnectedAt   time.Time `json:"connectedAt"`
	LastActivity  time.Time `json:"lastActivity"`
	BytesSent     int64     `json:"bytesSent"`
	BytesReceived int64     `json:"bytesReceived"`
	Latency       int64     `json:"latency"` // milliseconds
}

// NodeMetrics contains metrics about a node
type NodeMetrics struct {
	NodeID      string    `json:"nodeId"`
	Timestamp   time.Time `json:"timestamp"`
	CPUUsage    float64   `json:"cpuUsage"`    // percentage
	MemoryUsage float64   `json:"memoryUsage"` // percentage
	DiskUsage   float64   `json:"diskUsage"`   // percentage
	NetworkRx   int64     `json:"networkRx"`   // bytes per second
	NetworkTx   int64     `json:"networkTx"`   // bytes per second
	ActiveTasks int       `json:"activeTasks"`
	TotalTasks  int64     `json:"totalTasks"`
	FailedTasks int64     `json:"failedTasks"`
	Uptime      int64     `json:"uptime"` // seconds
	LoadAverage []float64 `json:"loadAverage,omitempty"`
}

// TaskExecution represents a task being executed on a node
type TaskExecution struct {
	TaskID        string            `json:"taskId"`
	NodeID        string            `json:"nodeId"`
	Type          string            `json:"type"`
	Status        string            `json:"status"`
	Priority      int               `json:"priority"`
	Progress      float64           `json:"progress"` // 0-1
	StartedAt     time.Time         `json:"startedAt"`
	EstimatedEnd  *time.Time        `json:"estimatedEnd,omitempty"`
	ResourceUsage *NodeResources    `json:"resourceUsage,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// NodeEvent represents an event that occurred on a node
type NodeEvent struct {
	ID        string                 `json:"id"`
	NodeID    string                 `json:"nodeId"`
	Type      string                 `json:"type"`     // status_change, error, resource_alert, etc.
	Severity  string                 `json:"severity"` // info, warning, error, critical
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source,omitempty"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name       string `json:"name"`
	IPAddress  string `json:"ipAddress"`
	Netmask    string `json:"netmask"`
	Gateway    string `json:"gateway,omitempty"`
	MAC        string `json:"mac,omitempty"`
	IsUp       bool   `json:"isUp"`
	IsLoopback bool   `json:"isLoopback"`
	Speed      int64  `json:"speed,omitempty"` // bits per second
}

// SystemInfo contains system-level information
type SystemInfo struct {
	OS                string             `json:"os"`
	Arch              string             `json:"arch"`
	Kernel            string             `json:"kernel"`
	Hostname          string             `json:"hostname"`
	BootTime          time.Time          `json:"bootTime"`
	Timezone          string             `json:"timezone"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces,omitempty"`
	Processes         int                `json:"processes"`
	Users             []string           `json:"users,omitempty"`
}

// ProcessInfo contains information about a process
type ProcessInfo struct {
	PID         int               `json:"pid"`
	Name        string            `json:"name"`
	CmdLine     string            `json:"cmdLine"`
	Status      string            `json:"status"`
	CPUUsage    float64           `json:"cpuUsage"`    // percentage
	MemoryUsage int64             `json:"memoryUsage"` // bytes
	StartTime   time.Time         `json:"startTime"`
	User        string            `json:"user"`
	ParentPID   int               `json:"parentPid"`
	Children    []int             `json:"children,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
