// Package client provides client node types and interfaces.
package client

import (
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/cluster"
)

// ClientInfo contains information about this client
type ClientInfo struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Address      string                `json:"address"`
	Port         int                   `json:"port"`
	Tags         []string              `json:"tags"`
	Capabilities *cluster.Capabilities `json:"capabilities"`
	Version      string                `json:"version"`
	Metadata     map[string]string     `json:"metadata"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	CommandID   string                 `json:"commandId"`
	Success     bool                   `json:"success"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CompletedAt time.Time              `json:"completedAt"`
	Duration    int64                  `json:"duration"` // milliseconds
}

// TaskExecutionContext provides context for task execution
type TaskExecutionContext struct {
	TaskID       string
	TaskType     cluster.TaskType
	Payload      map[string]interface{}
	StartTime    time.Time
	Timeout      time.Duration
	CondaEnv     string
	LlamacppPath string
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID      string                 `json:"taskId"`
	Success     bool                   `json:"success"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CompletedAt time.Time              `json:"completedAt"`
	Duration    int64                  `json:"duration"` // milliseconds
}

// ExecutorConfig contains configuration for task executor
type ExecutorConfig struct {
	CondaPath     string
	CondaEnvs     map[string]string
	LlamacppPaths []string
	TaskTimeout   time.Duration
	MaxConcurrent int
}
