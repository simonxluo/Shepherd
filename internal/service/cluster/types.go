// Package cluster provides distributed cluster management types and interfaces.
package cluster

import (
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
)

// ClientStatus is an alias for the unified NodeState type
// ClientStatus 是统一 NodeState 类型的别名，保持向后兼容
type ClientStatus = types.NodeState

// ClientStatus constants - 使用统一的 NodeState 常量
const (
	ClientStatusOffline  ClientStatus = types.StateOffline
	ClientStatusOnline   ClientStatus = types.StateOnline
	ClientStatusBusy     ClientStatus = types.StateBusy
	ClientStatusError    ClientStatus = types.StateError
	ClientStatusDegraded ClientStatus = types.StateDegraded
	ClientStatusDisabled ClientStatus = types.StateDisabled
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeLoadModel   TaskType = "load_model"
	TaskTypeUnloadModel TaskType = "unload_model"
	TaskTypeRunPython   TaskType = "run_python"
	TaskTypeRunLlamacpp TaskType = "run_llamacpp"
	TaskTypeCustom      TaskType = "custom"
)

// Client represents a connected client node
type Client struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Address      string            `json:"address"`
	Port         int               `json:"port"`
	Tags         []string          `json:"tags"`
	Capabilities *Capabilities     `json:"capabilities"`
	Status       ClientStatus      `json:"status"`
	LastSeen     time.Time         `json:"lastSeen"`
	Metadata     map[string]string `json:"metadata"`
	Connected    bool              `json:"connected"`
}

// Capabilities describes what a client can do
type Capabilities struct {
	GPU            bool     `json:"gpu"`
	GPUCount       int      `json:"gpuCount,omitempty"`
	GPUName        string   `json:"gpuName,omitempty"`
	GPUMemory      int64    `json:"gpuMemory,omitempty"` // bytes
	CPUCount       int      `json:"cpuCount"`
	Memory         int64    `json:"memory"` // bytes
	SupportsLlama  bool     `json:"supportsLlama"`
	SupportsPython bool     `json:"supportsPython"`
	CondaEnvs      []string `json:"condaEnvs,omitempty"`
}

// Task represents a distributed task
type Task struct {
	ID          string                 `json:"id"`
	Type        TaskType               `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	AssignedTo  string                 `json:"assignedTo,omitempty"` // client ID
	Status      TaskStatus             `json:"status"`
	CreatedAt   time.Time              `json:"createdAt"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retryCount"`
	MaxRetries  int                    `json:"maxRetries"`
}

// Command represents a command sent from master to client
type Command struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// Heartbeat represents a heartbeat message from client to master
type Heartbeat struct {
	ClientID  string                 `json:"clientId"`
	Timestamp time.Time              `json:"timestamp"`
	Status    ClientStatus           `json:"status"`
	Resources *ResourceUsage         `json:"resources,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceUsage represents current resource usage
type ResourceUsage struct {
	CPUPercent     float64 `json:"cpuPercent"`
	MemoryUsed     int64   `json:"memoryUsed"`  // bytes
	MemoryTotal    int64   `json:"memoryTotal"` // bytes
	GPUPercent     float64 `json:"gpuPercent"`
	GPUMemoryUsed  int64   `json:"gpuMemoryUsed"`  // bytes
	GPUMemoryTotal int64   `json:"gpuMemoryTotal"` // bytes
	DiskPercent    float64 `json:"diskPercent"`
	Uptime         int64   `json:"uptime"` // seconds
}

// LogEntry represents a log entry from a client
type LogEntry struct {
	ClientID  string            `json:"clientId"`
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// DiscoveredClient represents a client found during network scan
type DiscoveredClient struct {
	Address      string        `json:"address"`
	Port         int           `json:"port"`
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Version      string        `json:"version"`
	Capabilities *Capabilities `json:"capabilities"`
	Tags         []string      `json:"tags"`
}

// ScanStatus represents the status of a network scan
type ScanStatus struct {
	Running      bool               `json:"running"`
	StartTime    time.Time          `json:"startTime,omitempty"`
	Progress     float64            `json:"progress"` // 0-1
	Found        []DiscoveredClient `json:"found,omitempty"`
	TotalScanned int                `json:"totalScanned"`
	Errors       []string           `json:"errors,omitempty"`
}

// ClusterInfo represents cluster-wide information
type ClusterInfo struct {
	TotalClients   int               `json:"totalClients"`
	OnlineClients  int               `json:"onlineClients"`
	BusyClients    int               `json:"busyClients"`
	OfflineClients int               `json:"offlineClients"`
	TotalTasks     int               `json:"totalTasks"`
	RunningTasks   int               `json:"runningTasks"`
	PendingTasks   int               `json:"pendingTasks"`
	Resources      *ClusterResources `json:"resources"`
}

// ClusterResources represents aggregated cluster resources
type ClusterResources struct {
	TotalCPUCores  int   `json:"totalCPUCores"`
	TotalMemory    int64 `json:"totalMemory"`    // bytes
	TotalGPUMemory int64 `json:"totalGPUMemory"` // bytes
	UsedCPUCores   int   `json:"usedCPUCores"`
	UsedMemory     int64 `json:"usedMemory"`
	UsedGPUMemory  int64 `json:"usedGPUMemory"`
}
