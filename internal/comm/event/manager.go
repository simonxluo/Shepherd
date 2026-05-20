package event

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
)

// Manager manages WebSocket connections and broadcasts events
type Manager struct {
	// Connection management
	connections       map[string]*Connection
	connectionStatus  map[string]bool
	connectionCounter int

	// Channels
	eventChan chan *Event

	// Model manager reference (for status updates)
	modelMgr *model.Manager

	// Synchronization
	mu sync.RWMutex
	wg sync.WaitGroup

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// Connection represents a WebSocket connection
type Connection struct {
	ID     string
	Send   chan *Event
	mu     sync.Mutex
	closed bool
}

// NewManager creates a new WebSocket manager
func NewManager(modelMgr *model.Manager) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	mgr := &Manager{
		connections:      make(map[string]*Connection),
		connectionStatus: make(map[string]bool),
		eventChan:        make(chan *Event, 256),
		modelMgr:         modelMgr,
		ctx:              ctx,
		cancel:           cancel,
	}

	return mgr
}

// Start starts the WebSocket manager
func (m *Manager) Start() {
	m.wg.Add(1)
	go m.run()

	// Start heartbeat
	m.wg.Add(1)
	go m.heartbeatLoop()

	// Start system status broadcast
	m.wg.Add(1)
	go m.systemStatusLoop()

	logger.Info("WebSocket 管理器已启动")
}

// Stop stops the WebSocket manager
func (m *Manager) Stop() {
	logger.Info("正在停止 WebSocket 管理器...")

	m.cancel()

	// Close all connections
	m.mu.Lock()
	for _, conn := range m.connections {
		close(conn.Send)
		conn.closed = true
	}
	m.connections = make(map[string]*Connection)
	m.connectionStatus = make(map[string]bool)
	m.mu.Unlock()

	// Wait for goroutines
	m.wg.Wait()

	logger.Info("WebSocket 管理器已停止")
}

// run is the main event loop
func (m *Manager) run() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return

		case event := <-m.eventChan:
			m.broadcastEvent(event)
		}
	}
}

// heartbeatLoop sends periodic heartbeat messages
func (m *Manager) heartbeatLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			count := len(m.connections)
			m.mu.RUnlock()

			if count > 0 {
				m.Broadcast(NewHeartbeatEvent())
				logger.Debugf("发送心跳消息到 %d 个连接", count)
			}
		}
	}
}

// systemStatusLoop sends periodic system status updates
func (m *Manager) systemStatusLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			count := len(m.connections)
			confirmedCount := m.getConfirmedConnectionCount()
			m.mu.RUnlock()

			if count > 0 && m.modelMgr != nil {
				loadedModels := m.modelMgr.GetLoadedModelCount()
				m.Broadcast(NewSystemStatusEvent(loadedModels, count, confirmedCount))
				logger.Debugf("发送系统状态: 已加载模型=%d, 连接=%d, 已确认=%d",
					loadedModels, count, confirmedCount)
			}
		}
	}
}

// Broadcast sends an event to all connected clients
func (m *Manager) Broadcast(event *Event) {
	select {
	case m.eventChan <- event:
	case <-m.ctx.Done():
		logger.Warn("无法广播事件: 管理器已关闭")
	}
}

// broadcastEvent is the internal broadcast implementation
func (m *Manager) broadcastEvent(event *Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for connID, conn := range m.connections {
		if !conn.closed {
			select {
			case conn.Send <- event:
			default:
				// Channel full, close connection
				logger.Warnf("连接 %s 的发送通道已满，关闭连接", connID)
				go m.closeConnection(connID)
			}
		}
	}
}

// HandleWebSocket handles WebSocket upgrade and connection
func (m *Manager) HandleWebSocket(c *gin.Context) {
	// For SSE (Server-Sent Events) implementation
	// This is simpler than full WebSocket and works with Gin's built-in support

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if client supports streaming
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Create connection
	m.mu.Lock()
	m.connectionCounter++
	connID := fmt.Sprintf("conn-%d", m.connectionCounter)
	conn := &Connection{
		ID:   connID,
		Send: make(chan *Event, 64),
	}
	m.connections[connID] = conn
	m.connectionStatus[connID] = false
	m.mu.Unlock()

	logger.Infof("SSE 连接已建立: %s (总连接数: %d)", connID, len(m.connections))

	// Ensure connection is cleaned up
	defer func() {
		// Send a final close event before disconnecting
		c.SSEvent("close", fmt.Sprintf(`{"connectionId":"%s","timestamp":%d}`, connID, time.Now().UnixMilli()))
		flusher.Flush()

		m.mu.Lock()
		if _, exists := m.connections[connID]; exists {
			delete(m.connections, connID)
			delete(m.connectionStatus, connID)
		}
		m.mu.Unlock()

		conn.mu.Lock()
		if !conn.closed {
			conn.closed = true
			close(conn.Send)
		}
		conn.mu.Unlock()

		logger.Infof("SSE 连接已断开: %s (剩余连接数: %d)", connID, m.GetConnectionCount())
	}()

	// Send initial connection event
	c.SSEvent("connected", fmt.Sprintf(`{"connectionId":"%s","timestamp":%d}`, connID, time.Now().UnixMilli()))
	flusher.Flush()

	// Stream events to client
	notify := c.Request.Context().Done()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-notify:
			// Client disconnected
			return

		case <-m.ctx.Done():
			// Server shutting down
			return

		case event, ok := <-conn.Send:
			conn.mu.Lock()
			closed := conn.closed
			conn.mu.Unlock()

			if !ok || closed {
				// Connection closed
				return
			}

			// Send as SSE event
			c.SSEvent("message", event.String())
			flusher.Flush()

		case <-keepalive.C:
			// Send keepalive comment to prevent timeout
			if _, err := c.Writer.Write([]byte(": keepalive\n\n")); err != nil {
				// Write failed, client probably disconnected
				logger.Debugf("Keepalive 写入失败，断开连接 %s: %v", connID, err)
				return
			}
			flusher.Flush()
		}
	}
}

// GetConnectionCount returns the total number of connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// getConfirmedConnectionCount returns the number of confirmed connections
func (m *Manager) getConfirmedConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, status := range m.connectionStatus {
		if status {
			count++
		}
	}
	return count
}

// closeConnection closes a specific connection
func (m *Manager) closeConnection(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, ok := m.connections[connID]; ok {
		if !conn.closed {
			close(conn.Send)
			conn.closed = true
		}
		delete(m.connections, connID)
		delete(m.connectionStatus, connID)
	}
}
