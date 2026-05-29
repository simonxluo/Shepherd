package event

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of WebSocket event
type EventType string

const (
	EventTypeHeartbeat        EventType = "heartbeat"
	EventTypeSystemStatus     EventType = "systemStatus"
	EventTypeModelLoad        EventType = "modelLoad"
	EventTypeModelLoadStart   EventType = "modelLoadStart"
	EventTypeModelStop        EventType = "modelStop"
	EventTypeModelSlots       EventType = "model_slots"
	EventTypeConsole          EventType = "console"
	EventTypeDownloadUpdate   EventType = "download_update"
	EventTypeDownloadProgress EventType = "download_progress"
	EventTypeScanProgress     EventType = "scan_progress"
	EventTypeScanComplete     EventType = "scan_complete"
	EventTypeBenchmarkUpdate  EventType = "benchmarkUpdate"
)

// Event represents a WebSocket event
type Event struct {
	Type      EventType `json:"type"`
	Timestamp int64     `json:"timestamp"`

	// Model events
	ModelID string      `json:"modelId,omitempty"`
	Success bool        `json:"success,omitempty"`
	Message string      `json:"message,omitempty"`
	Port    int         `json:"port,omitempty"`
	Data    interface{} `json:"data,omitempty"`

	// Console events
	Line64 string `json:"line64,omitempty"` // Base64 encoded console line

	// Download events
	TaskID          string  `json:"taskId,omitempty"`
	State           string  `json:"state,omitempty"`
	DownloadedBytes int64   `json:"downloadedBytes,omitempty"`
	TotalBytes      int64   `json:"totalBytes,omitempty"`
	PartsCompleted  int     `json:"partsCompleted,omitempty"`
	PartsTotal      int     `json:"partsTotal,omitempty"`
	FileName        string  `json:"fileName,omitempty"`
	ErrorMessage    string  `json:"errorMessage,omitempty"`
	ProgressRatio   float64 `json:"progressRatio,omitempty"`

	// System status
	LoadedModels         int `json:"loadedModels,omitempty"`
	Connections          int `json:"connections,omitempty"`
	ConfirmedConnections int `json:"confirmedConnections,omitempty"`
}

// NewEvent creates a new event with current timestamp
func NewEvent(eventType EventType) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ToJSON converts the event to JSON bytes
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// String returns the JSON string representation
func (e *Event) String() string {
	data, err := e.ToJSON()
	if err != nil {
		return fmt.Sprintf(`{"type":"error","message":"%s"}`, err.Error())
	}
	return string(data)
}

// Event builders

// NewHeartbeatEvent creates a heartbeat event
func NewHeartbeatEvent() *Event {
	return NewEvent(EventTypeHeartbeat)
}

// NewSystemStatusEvent creates a system status event
func NewSystemStatusEvent(loadedModels, connections, confirmedConnections int) *Event {
	event := NewEvent(EventTypeSystemStatus)
	event.LoadedModels = loadedModels
	event.Connections = connections
	event.ConfirmedConnections = confirmedConnections
	return event
}

// NewModelLoadStartEvent creates a model load start event
func NewModelLoadStartEvent(modelID string, port int, message string) *Event {
	event := NewEvent(EventTypeModelLoadStart)
	event.ModelID = modelID
	event.Port = port
	event.Message = message
	return event
}

// NewModelLoadEvent creates a model load event
func NewModelLoadEvent(modelID string, success bool, message string, port int) *Event {
	event := NewEvent(EventTypeModelLoad)
	event.ModelID = modelID
	event.Success = success
	event.Message = message
	if port > 0 {
		event.Port = port
	}
	return event
}

// NewModelStopEvent creates a model stop event
func NewModelStopEvent(modelID string, success bool, message string) *Event {
	event := NewEvent(EventTypeModelStop)
	event.ModelID = modelID
	event.Success = success
	event.Message = message
	return event
}

// NewModelSlotsEvent creates a model slots update event
func NewModelSlotsEvent(modelID string, slots interface{}) *Event {
	event := NewEvent(EventTypeModelSlots)
	event.ModelID = modelID
	event.Data = slots
	return event
}

// NewConsoleLineEvent creates a console line event (line is base64 encoded)
func NewConsoleLineEvent(modelID string, line string) *Event {
	event := NewEvent(EventTypeConsole)
	event.ModelID = modelID
	if line != "" {
		event.Line64 = base64.StdEncoding.EncodeToString([]byte(line))
	}
	return event
}

// NewDownloadStatusEvent creates a download status event
func NewDownloadStatusEvent(taskID, state string, downloadedBytes, totalBytes int64,
	partsCompleted, partsTotal int, fileName, errorMessage string) *Event {

	event := NewEvent(EventTypeDownloadUpdate)
	event.TaskID = taskID
	event.State = state
	event.DownloadedBytes = downloadedBytes
	event.TotalBytes = totalBytes
	event.PartsCompleted = partsCompleted
	event.PartsTotal = partsTotal
	event.FileName = fileName
	event.ErrorMessage = errorMessage
	return event
}

// NewDownloadProgressEvent creates a download progress event
func NewDownloadProgressEvent(taskID string, downloadedBytes, totalBytes int64,
	partsCompleted, partsTotal int, progressRatio float64) *Event {

	event := NewEvent(EventTypeDownloadProgress)
	event.TaskID = taskID
	event.DownloadedBytes = downloadedBytes
	event.TotalBytes = totalBytes
	event.PartsCompleted = partsCompleted
	event.PartsTotal = partsTotal
	event.ProgressRatio = progressRatio
	return event
}

// NewScanProgressEvent creates a scan progress event
func NewScanProgressEvent(directory string, scanned, total int) *Event {
	event := NewEvent(EventTypeScanProgress)
	event.Message = directory
	event.Data = map[string]interface{}{
		"directory": directory,
		"scanned":   scanned,
		"total":     total,
	}
	return event
}

// NewScanCompleteEvent creates a scan complete event
func NewScanCompleteEvent(foundModels int, duration time.Duration) *Event {
	event := NewEvent(EventTypeScanComplete)
	event.Data = map[string]interface{}{
		"foundModels": foundModels,
		"duration":    duration.Milliseconds(),
	}
	return event
}

// NewBenchmarkUpdateEvent creates a benchmark task status update event.
func NewBenchmarkUpdateEvent(taskID, modelID, status string, data interface{}) *Event {
	event := NewEvent(EventTypeBenchmarkUpdate)
	event.TaskID = taskID
	event.ModelID = modelID
	event.State = status
	event.Data = data
	return event
}
