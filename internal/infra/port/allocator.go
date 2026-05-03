// Package port provides port allocation utilities for managing dynamic port assignment
package port

import (
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"net"
	"sync"
	"time"
)

// PortAllocator manages dynamic port allocation
type PortAllocator struct {
	mu        sync.Mutex
	basePort  int          // Base port number (e.g., 8081)
	maxPort   int          // Maximum port number (e.g., 9000)
	allocated map[int]bool // Allocated ports
}

// NewPortAllocator creates a new port allocator
func NewPortAllocator(basePort, maxPort int) *PortAllocator {
	return &PortAllocator{
		basePort:  basePort,
		maxPort:   maxPort,
		allocated: make(map[int]bool),
	}
}

// NextPort allocates the next available port
func (a *PortAllocator) NextPort() (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for port := a.basePort; port <= a.maxPort; port++ {
		if !a.allocated[port] {
			// Double-check if port is actually available
			if !a.isPortInUse(port) {
				a.allocated[port] = true
				return port, nil
			}
		}
	}

	return 0, fmt.Errorf("no available ports between %d and %d", a.basePort, a.maxPort)
}

// Release releases an allocated port
func (a *PortAllocator) Release(port int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.allocated, port)
}

// isPortInUse checks if a port is actually in use by attempting to connect
func (a *PortAllocator) isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf(":%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	utils.CloseQuietly(conn)
	return true
}
