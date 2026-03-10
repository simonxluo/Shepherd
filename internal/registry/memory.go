// Package registry provides client registry implementation
// 这个包提供客户端注册表的实现
package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/node"
)

// MemoryClientRegistry implements ClientRegistry using in-memory storage
// MemoryClientRegistry 使用内存存储实现 ClientRegistry
type MemoryClientRegistry struct {
	clients         map[string]*node.NodeInfo
	mu              sync.RWMutex
	log             *logger.Logger
	cleanupInterval time.Duration
	clientTimeout   time.Duration
}

// NewMemoryClientRegistry creates a new in-memory client registry
// NewMemoryClientRegistry 创建新的内存客户端注册表
func NewMemoryClientRegistry(log *logger.Logger, cleanupInterval, clientTimeout time.Duration) *MemoryClientRegistry {
	return &MemoryClientRegistry{
		clients:         make(map[string]*node.NodeInfo),
		log:             log,
		cleanupInterval: cleanupInterval,
		clientTimeout:   clientTimeout,
	}
}

// Register registers a new client node
// Register 注册新的客户端节点
func (r *MemoryClientRegistry) Register(info *node.NodeInfo) error {
	if info == nil {
		return fmt.Errorf("客户端信息不能为空")
	}

	if info.ID == "" {
		return fmt.Errorf("客户端节点ID不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[info.ID]; exists {
		return fmt.Errorf("客户端 %s 已存在", info.ID)
	}

	now := time.Now()
	info.RegisteredAt = now
	info.CreatedAt = now
	info.UpdatedAt = now
	info.LastSeen = now
	info.Status = node.NodeStatusOnline

	r.clients[info.ID] = info
	r.log.Infof("客户端已注册: ID=%s, 名称=%s", info.ID, info.Name)

	return nil
}

// Unregister removes a client node
// Unregister 注销客户端节点
func (r *MemoryClientRegistry) Unregister(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[nodeID]; !exists {
		return fmt.Errorf("客户端 %s 不存在", nodeID)
	}

	delete(r.clients, nodeID)
	r.log.Infof("客户端已注销: ID=%s", nodeID)

	return nil
}

// Get retrieves a client by ID
// Get 根据 ID 获取客户端
func (r *MemoryClientRegistry) Get(nodeID string) (*node.NodeInfo, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("节点ID不能为空")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.clients[nodeID]
	if !exists {
		return nil, fmt.Errorf("客户端 %s 不存在", nodeID)
	}

	// Return a copy to prevent external modification
	infoCopy := *info
	return &infoCopy, nil
}

// List returns all registered clients
// List 返回所有已注册的客户端
func (r *MemoryClientRegistry) List() []*node.NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*node.NodeInfo, 0, len(r.clients))
	for _, info := range r.clients {
		infoCopy := *info
		result = append(result, &infoCopy)
	}

	return result
}

// GetStats returns registry statistics
// GetStats 返回注册表统计信息
func (r *MemoryClientRegistry) GetStats() *node.RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &node.RegistryStats{
		TotalClients: len(r.clients),
	}

	for _, info := range r.clients {
		switch info.Status {
		case node.NodeStatusOnline:
			stats.OnlineClients++
		case node.NodeStatusOffline:
			stats.OfflineClients++
		case node.NodeStatusBusy:
			stats.BusyClients++
		case node.NodeStatusError:
			stats.ErrorClients++
		}
	}

	return stats
}

// Find clients matching the given predicate
// Find 根据条件查找客户端
func (r *MemoryClientRegistry) Find(predicate func(*node.NodeInfo) bool) []*node.NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*node.NodeInfo, 0)
	for _, info := range r.clients {
		infoCopy := *info
		if predicate(&infoCopy) {
			result = append(result, &infoCopy)
		}
	}

	return result
}

// UpdateStatus updates the status of a client
// UpdateStatus 更新客户端状态
func (r *MemoryClientRegistry) UpdateStatus(nodeID string, status node.NodeStatus) error {
	if nodeID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.clients[nodeID]
	if !exists {
		return fmt.Errorf("客户端 %s 不存在", nodeID)
	}

	info.Status = status
	info.UpdatedAt = time.Now()

	return nil
}

// UpdateResources updates the resources of a client
// UpdateResources 更新客户端资源
func (r *MemoryClientRegistry) UpdateResources(nodeID string, resources *node.NodeResources) error {
	if nodeID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.clients[nodeID]
	if !exists {
		return fmt.Errorf("客户端 %s 不存在", nodeID)
	}

	info.Resources = resources
	info.UpdatedAt = time.Now()

	return nil
}

// GetOnlineClients returns all online clients
// GetOnlineClients 返回所有在线客户端
func (r *MemoryClientRegistry) GetOnlineClients() []*node.NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*node.NodeInfo, 0)
	for _, info := range r.clients {
		if info.Status == node.NodeStatusOnline {
			infoCopy := *info
			result = append(result, &infoCopy)
		}
	}

	return result
}

// Cleanup removes clients that haven't been seen within the timeout
// Cleanup 清理超时的客户端
func (r *MemoryClientRegistry) Cleanup(timeout time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)

	for nodeID, info := range r.clients {
		if now.Sub(info.LastSeen) > timeout {
			toRemove = append(toRemove, nodeID)
		}
	}

	for _, nodeID := range toRemove {
		delete(r.clients, nodeID)
	}

	if len(toRemove) > 0 {
		r.log.Infof("清理超时客户端: 清理了 %d 个", len(toRemove))
	}

	return len(toRemove)
}

// Count returns the total number of registered clients
// Count 返回已注册客户端总数
func (r *MemoryClientRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Exists checks if a client exists
// Exists 检查客户端是否存在
func (r *MemoryClientRegistry) Exists(nodeID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.clients[nodeID]
	return exists
}
