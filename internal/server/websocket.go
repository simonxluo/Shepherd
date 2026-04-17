// Package server provides WebSocket support for real-time event notifications
package server

import (
	"encoding/json"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Event types
const (
	EventModelLoadStart    = "model_load_start"
	EventModelLoadProgress = "model_load_progress"
	EventModelLoadComplete = "model_load_complete"
	EventModelLoadError    = "model_load_error"
	EventModelUnload       = "model_unload"
	EventModelStatusChange = "model_status_changed"
	EventProcessMetrics    = "process_metrics"
	EventProcessLog        = "process_log"
)

// Event represents a WebSocket event
type Event struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
	Hub  *WebSocketHub
	mu   sync.Mutex
}

// WebSocketHub manages WebSocket connections and broadcasts
type WebSocketHub struct {
	clients    map[string]*WebSocketClient
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan []byte
	mu         sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[string]*WebSocketClient),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan []byte, 256),
	}
}

// Run starts the hub's event loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// Client is blocked, close it
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Emit broadcasts an event to all connected clients
func (h *WebSocketHub) Emit(eventType string, data interface{}) {
	event := Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		return
	}

	select {
	case h.broadcast <- bytes:
	default:
		// Broadcast channel is full, drop the event
	}
}

// Register adds a new client
func (h *WebSocketHub) Register(client *WebSocketClient) {
	h.register <- client
}

// Unregister removes a client
func (h *WebSocketHub) Unregister(client *WebSocketClient) {
	h.unregister <- client
}

// ClientCount returns the number of connected clients
func (h *WebSocketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *WebSocketClient) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		utils.CloseQuietly(c.Conn)
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.mu.Lock()
			if !ok {
				// Hub closed the channel
				utils.WriteMessageQuietly(c.Conn, websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}

			utils.SetWriteDeadlineQuietly(c.Conn, 10*time.Second)
			err := c.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()

		case <-ticker.C:
			// Send keepalive ping
			c.mu.Lock()
			utils.SetWriteDeadlineQuietly(c.Conn, 10*time.Second)
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		}
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		utils.CloseQuietly(c.Conn)
	}()

	utils.SetReadDeadlineQuietly(c.Conn, 60*time.Second)
	c.Conn.SetPongHandler(func(string) error {
		utils.SetReadDeadlineQuietly(c.Conn, 60*time.Second)
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log unexpected close
			}
			break
		}
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}

// WebSocketUpgrader handles upgrading HTTP to WebSocket
var WebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// handleWebSocket handles WebSocket connection requests
func (s *Server) HandleWebSocket(c *gin.Context) {
	conn, err := WebSocketUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &WebSocketClient{
		ID:   generateClientID(),
		Conn: conn,
		Send: make(chan []byte, 256),
		Hub:  s.wsHub,
	}

	s.wsHub.Register(client)

	// Start pumps
	go client.WritePump()
	client.ReadPump()
}
