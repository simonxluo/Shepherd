package event

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventTypeHeartbeat)

	assert.Equal(t, EventTypeHeartbeat, event.Type)
	assert.Greater(t, event.Timestamp, int64(0))
}

func TestEventToJSON(t *testing.T) {
	t.Run("Heartbeat event", func(t *testing.T) {
		event := NewHeartbeatEvent()
		data, err := event.ToJSON()
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type":"heartbeat"`)
		assert.Contains(t, string(data), `"timestamp":`)
	})

	t.Run("Model load event", func(t *testing.T) {
		event := NewModelLoadEvent("test-model", true, "Loaded successfully", 8080)
		data, err := event.ToJSON()
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type":"modelLoad"`)
		assert.Contains(t, string(data), `"modelId":"test-model"`)
		assert.Contains(t, string(data), `"success":true`)
		assert.Contains(t, string(data), `"port":8080`)
	})

	t.Run("Console line event", func(t *testing.T) {
		event := NewConsoleLineEvent("test-model", "Test log line")
		data, err := event.ToJSON()
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type":"console"`)
		assert.Contains(t, string(data), `"modelId":"test-model"`)
		assert.Contains(t, string(data), `"line64":`)
	})
}

func TestEventBuilders(t *testing.T) {
	t.Run("Heartbeat", func(t *testing.T) {
		event := NewHeartbeatEvent()
		assert.Equal(t, EventTypeHeartbeat, event.Type)
		assert.Greater(t, event.Timestamp, int64(0))
	})

	t.Run("System status", func(t *testing.T) {
		event := NewSystemStatusEvent(3, 5, 4)
		assert.Equal(t, EventTypeSystemStatus, event.Type)
		assert.Equal(t, 3, event.LoadedModels)
		assert.Equal(t, 5, event.Connections)
		assert.Equal(t, 4, event.ConfirmedConnections)
	})

	t.Run("Model load start", func(t *testing.T) {
		event := NewModelLoadStartEvent("model-123", 9090, "Starting load...")
		assert.Equal(t, EventTypeModelLoadStart, event.Type)
		assert.Equal(t, "model-123", event.ModelID)
		assert.Equal(t, 9090, event.Port)
		assert.Equal(t, "Starting load...", event.Message)
	})

	t.Run("Model load", func(t *testing.T) {
		event := NewModelLoadEvent("model-123", true, "Success", 9090)
		assert.Equal(t, EventTypeModelLoad, event.Type)
		assert.True(t, event.Success)
		assert.Equal(t, "Success", event.Message)
		assert.Equal(t, 9090, event.Port)
	})

	t.Run("Model stop", func(t *testing.T) {
		event := NewModelStopEvent("model-123", true, "Stopped")
		assert.Equal(t, EventTypeModelStop, event.Type)
		assert.Equal(t, "model-123", event.ModelID)
		assert.True(t, event.Success)
	})

	t.Run("Model slots", func(t *testing.T) {
		slots := map[string]interface{}{"slot1": "processing"}
		event := NewModelSlotsEvent("model-123", slots)
		assert.Equal(t, EventTypeModelSlots, event.Type)
		assert.Equal(t, "model-123", event.ModelID)
		assert.NotNil(t, event.Data)
	})

	t.Run("Console line", func(t *testing.T) {
		event := NewConsoleLineEvent("model-123", "Log output")
		assert.Equal(t, EventTypeConsole, event.Type)
		assert.Equal(t, "model-123", event.ModelID)
		assert.NotEmpty(t, event.Line64)
	})

	t.Run("Download status", func(t *testing.T) {
		event := NewDownloadStatusEvent("task-1", "downloading", 1024, 2048, 1, 2, "model.gguf", "")
		assert.Equal(t, EventTypeDownloadUpdate, event.Type)
		assert.Equal(t, "task-1", event.TaskID)
		assert.Equal(t, "downloading", event.State)
		assert.Equal(t, int64(1024), event.DownloadedBytes)
		assert.Equal(t, int64(2048), event.TotalBytes)
	})

	t.Run("Download progress", func(t *testing.T) {
		event := NewDownloadProgressEvent("task-1", 1024, 2048, 1, 2, 0.5)
		assert.Equal(t, EventTypeDownloadProgress, event.Type)
		assert.Equal(t, "task-1", event.TaskID)
		assert.Equal(t, 0.5, event.ProgressRatio)
	})

	t.Run("Scan progress", func(t *testing.T) {
		event := NewScanProgressEvent("/models", 10, 50)
		assert.Equal(t, EventTypeScanProgress, event.Type)
		assert.NotNil(t, event.Data)
	})

	t.Run("Scan complete", func(t *testing.T) {
		event := NewScanCompleteEvent(5, 5*time.Second)
		assert.Equal(t, EventTypeScanComplete, event.Type)
		assert.NotNil(t, event.Data)
	})
}

func TestManager(t *testing.T) {
	t.Run("Create manager", func(t *testing.T) {
		mgr := NewManager(nil)
		assert.NotNil(t, mgr)
		assert.NotNil(t, mgr.eventChan)
		assert.NotNil(t, mgr.connections)
		assert.NotNil(t, mgr.connectionStatus)
	})

	t.Run("Start and stop manager", func(t *testing.T) {
		mgr := NewManager(nil)

		// Start manager
		mgr.Start()
		assert.Equal(t, 0, mgr.GetConnectionCount())

		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		// Stop manager
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stopped := make(chan bool)
		go func() {
			mgr.Stop()
			stopped <- true
		}()

		select {
		case <-stopped:
			// Stopped successfully
		case <-ctx.Done():
			t.Fatal("Manager did not stop within timeout")
		}
	})

	t.Run("Broadcast with no connections", func(t *testing.T) {
		mgr := NewManager(nil)
		mgr.Start()
		defer mgr.Stop()

		// Should not panic
		mgr.Broadcast(NewHeartbeatEvent())

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("Connection count", func(t *testing.T) {
		mgr := NewManager(nil)

		assert.Equal(t, 0, mgr.GetConnectionCount())

		// Simulate adding connections
		mgr.mu.Lock()
		mgr.connections["conn-1"] = &Connection{ID: "conn-1", Send: make(chan *Event, 10)}
		mgr.connections["conn-2"] = &Connection{ID: "conn-2", Send: make(chan *Event, 10)}
		mgr.mu.Unlock()

		assert.Equal(t, 2, mgr.GetConnectionCount())
	})
}

func TestManagerBroadcastHelpers(t *testing.T) {
	mgr := NewManager(nil)
	mgr.Start()
	defer mgr.Stop()

	// These should not panic
	t.Run("Model load start", func(t *testing.T) {
		mgr.BroadcastModelLoadStart("model-1", 9090, "Loading...")
	})

	t.Run("Model load", func(t *testing.T) {
		mgr.BroadcastModelLoad("model-1", true, "Success", 9090)
	})

	t.Run("Model stop", func(t *testing.T) {
		mgr.BroadcastModelStop("model-1", true, "Stopped")
	})

	t.Run("Model slots", func(t *testing.T) {
		mgr.BroadcastModelSlots("model-1", map[string]interface{}{})
	})

	t.Run("Console line", func(t *testing.T) {
		mgr.BroadcastConsoleLine("model-1", "Log output")
	})

	t.Run("Download status", func(t *testing.T) {
		mgr.BroadcastDownloadStatus("task-1", "downloading", 1024, 2048, 1, 2, "model.gguf", "")
	})

	t.Run("Download progress", func(t *testing.T) {
		mgr.BroadcastDownloadProgress("task-1", 1024, 2048, 1, 2, 0.5)
	})

	t.Run("Scan progress", func(t *testing.T) {
		mgr.BroadcastScanProgress("/models", 10, 50)
	})

	t.Run("Scan complete", func(t *testing.T) {
		mgr.BroadcastScanComplete(5, 5*time.Second)
	})

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)
}

func TestConnection(t *testing.T) {
	t.Run("Create connection", func(t *testing.T) {
		conn := &Connection{
			ID:   "conn-1",
			Send: make(chan *Event, 10),
		}

		assert.Equal(t, "conn-1", conn.ID)
		assert.False(t, conn.closed)
	})

	t.Run("Close connection", func(t *testing.T) {
		conn := &Connection{
			ID:   "conn-1",
			Send: make(chan *Event, 10),
		}

		assert.False(t, conn.closed)
		conn.closed = true
		assert.True(t, conn.closed)
	})
}
