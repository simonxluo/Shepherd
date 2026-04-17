// Package client provides client node management for the master node.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/service/cluster"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

// Manager manages connected client nodes
type Manager struct {
	clients    map[string]*cluster.Client
	configPath string
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	log        *logger.Logger
	httpClient HTTPClient
}

// HTTPClient interface for making HTTP requests (for testability)
type HTTPClient interface {
	Get(url string) (int, []byte, error)
	Post(url string, body []byte) (int, []byte, error)
}

// DefaultHTTPClient implements HTTPClient using standard HTTP
type DefaultHTTPClient struct{}

// Get performs a GET request
func (c *DefaultHTTPClient) Get(url string) (int, []byte, error) {
	return 0, nil, fmt.Errorf("not implemented")
}

// Post performs a POST request
func (c *DefaultHTTPClient) Post(url string, body []byte) (int, []byte, error) {
	return 0, nil, fmt.Errorf("not implemented")
}

// NewManager creates a new client manager
func NewManager(cfg *config.MasterConfig, log *logger.Logger) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Ensure client config directory exists
	if cfg.ClientConfigDir != "" {
		if err := os.MkdirAll(cfg.ClientConfigDir, 0755); err != nil {
			cancel()
			return nil, fmt.Errorf("创建客户端配置目录失败: %w", err)
		}
	}

	m := &Manager{
		clients:    make(map[string]*cluster.Client),
		configPath: filepath.Join(cfg.ClientConfigDir, "clients.yaml"),
		ctx:        ctx,
		cancel:     cancel,
		log:        log,
		httpClient: &DefaultHTTPClient{},
	}

	// Load existing client configurations
	if err := m.loadClients(); err != nil {
		log.Warn("加载客户端配置失败", nil)
	}

	return m, nil
}

// Register registers a new client
func (m *Manager) Register(clientInfo *cluster.Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if client already exists
	if existing, exists := m.clients[clientInfo.ID]; exists {
		// Update existing client
		existing.Name = clientInfo.Name
		existing.Address = clientInfo.Address
		existing.Port = clientInfo.Port
		existing.Capabilities = clientInfo.Capabilities
		existing.Tags = clientInfo.Tags
		existing.Status = cluster.ClientStatusOnline
		existing.LastSeen = time.Now()
		existing.Connected = true
		m.log.Info(fmt.Sprintf("客户端重新注册: %s (%s)", clientInfo.ID, clientInfo.Name))
	} else {
		// Add new client
		clientInfo.Status = cluster.ClientStatusOnline
		clientInfo.LastSeen = time.Now()
		clientInfo.Connected = true
		m.clients[clientInfo.ID] = clientInfo
		m.log.Info(fmt.Sprintf("新客户端注册: %s (%s) from %s", clientInfo.ID, clientInfo.Name, clientInfo.Address))
	}

	// Persist configuration
	if err := m.saveClients(); err != nil {
		m.log.Error("保存客户端配置失败")
	}

	return nil
}

// Unregister removes a client
func (m *Manager) Unregister(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.clients[clientID]; exists {
		client.Connected = false
		client.Status = cluster.ClientStatusOffline
		m.log.Info(fmt.Sprintf("客户端注销: %s", clientID))
	}

	return nil
}

// GetClient retrieves a client by ID
func (m *Manager) GetClient(clientID string) (*cluster.Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[clientID]
	return client, exists
}

// ListClients returns all registered clients
func (m *Manager) ListClients() []*cluster.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]*cluster.Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	return clients
}

// GetOnlineClients returns all online clients
func (m *Manager) GetOnlineClients() []*cluster.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]*cluster.Client, 0)
	for _, client := range m.clients {
		if client.Status == cluster.ClientStatusOnline {
			clients = append(clients, client)
		}
	}
	return clients
}

// SendCommand sends a command to a specific client
func (m *Manager) SendCommand(clientID string, command *cluster.Command) (map[string]interface{}, error) {
	client, exists := m.GetClient(clientID)
	if !exists {
		return nil, fmt.Errorf("客户端不存在: %s", clientID)
	}

	if !client.Connected || client.Status != cluster.ClientStatusOnline {
		return nil, fmt.Errorf("客户端离线: %s", clientID)
	}

	// Prepare command request
	url := fmt.Sprintf("http://%s:%d/api/client/commands", client.Address, client.Port)
	body, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("序列化命令失败: %w", err)
	}

	// Send command
	statusCode, respBody, err := m.httpClient.Post(url, body)
	if err != nil {
		return nil, fmt.Errorf("发送命令失败: %w", err)
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("命令执行失败: HTTP %d", statusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return result, nil
}

// UpdateHeartbeat updates client heartbeat status
func (m *Manager) UpdateHeartbeat(heartbeat *cluster.Heartbeat) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[heartbeat.ClientID]
	if !exists {
		return fmt.Errorf("客户端不存在: %s", heartbeat.ClientID)
	}

	client.Status = heartbeat.Status
	client.LastSeen = time.Now()

	return nil
}

// CheckHeartbeats checks for stale clients and marks them offline
func (m *Manager) CheckHeartbeats(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, client := range m.clients {
		if client.Connected && now.Sub(client.LastSeen) > timeout {
			client.Status = cluster.ClientStatusOffline
			client.Connected = false
			m.log.Warn(fmt.Sprintf("客户端超时离线: %s (%s)", client.ID, client.Name))
		}
	}
}

// GetClusterInfo returns cluster-wide information
func (m *Manager) GetClusterInfo() *cluster.ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := &cluster.ClusterInfo{
		TotalClients:   len(m.clients),
		OnlineClients:  0,
		BusyClients:    0,
		OfflineClients: 0,
	}

	for _, client := range m.clients {
		switch client.Status {
		case cluster.ClientStatusOnline:
			info.OnlineClients++
		case cluster.ClientStatusBusy:
			info.BusyClients++
		case cluster.ClientStatusOffline:
			info.OfflineClients++
		}
	}

	return info
}

// Start starts the client manager background tasks
func (m *Manager) Start() {
	// Start heartbeat checker
	m.wg.Add(1)
	go m.heartbeatChecker()
}

// Stop stops the client manager
func (m *Manager) Stop() {
	m.cancel()
	m.wg.Wait()
}

// heartbeatChecker periodically checks client heartbeats
func (m *Manager) heartbeatChecker() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.CheckHeartbeats(90 * time.Second)
		}
	}
}

// loadClients loads client configurations from file
func (m *Manager) loadClients() error {
	if m.configPath == "" {
		return nil
	}

	_, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config file yet
		}
		return err
	}

	// For now, just initialize empty map
	// YAML parsing would be implemented here
	m.clients = make(map[string]*cluster.Client)

	return nil
}

// saveClients saves client configurations to file
func (m *Manager) saveClients() error {
	if m.configPath == "" {
		return nil
	}

	// For now, just return success
	// YAML serialization would be implemented here
	return nil
}
