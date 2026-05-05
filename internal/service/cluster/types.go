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
	ClientStatusOnline ClientStatus = types.StateOnline
	ClientStatusError  ClientStatus = types.StateError
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
	TaskTypeRunLlamacpp TaskType = "run_llamacpp"
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

