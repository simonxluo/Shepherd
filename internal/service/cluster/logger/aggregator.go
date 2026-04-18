// Package logger provides log aggregation functionality for the master node.
package logger

import (
	"context"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	shepherdLogger "github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/cluster"
)

// Aggregator aggregates logs from multiple clients
type Aggregator struct {
	log        *shepherdLogger.Logger
	clientLogs map[string][]*cluster.LogEntry // clientID -> logs
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	config     *config.LogAggregationConfig
}

// NewAggregator creates a new log aggregator
func NewAggregator(cfg *config.LogAggregationConfig, log *shepherdLogger.Logger) *Aggregator {
	ctx, cancel := context.WithCancel(context.Background())

	return &Aggregator{
		log:        log,
		clientLogs: make(map[string][]*cluster.LogEntry),
		ctx:        ctx,
		cancel:     cancel,
		config:     cfg,
	}
}

// Start starts the log aggregator
func (a *Aggregator) Start() {
	a.wg.Add(1)
	go a.cleanupLoop()
}

// Stop stops the log aggregator
func (a *Aggregator) Stop() {
	a.cancel()
	a.wg.Wait()
}

// AddLogs adds logs from a client
func (a *Aggregator) AddLogs(clientID string, logs []*cluster.LogEntry) error {
	if !a.config.Enabled {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check buffer size
	if len(a.clientLogs[clientID]) >= a.config.MaxBufferSize/1000 { // Approximate limit
		// Remove oldest logs
		a.clientLogs[clientID] = a.clientLogs[clientID][len(a.clientLogs[clientID])/2:]
	}

	a.clientLogs[clientID] = append(a.clientLogs[clientID], logs...)

	return nil
}

// GetLogs returns logs from a specific client
func (a *Aggregator) GetLogs(clientID string, limit int) []*cluster.LogEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	logs, exists := a.clientLogs[clientID]
	if !exists {
		return []*cluster.LogEntry{}
	}

	if limit > 0 && limit < len(logs) {
		return logs[len(logs)-limit:]
	}

	return logs
}

// GetAllLogs returns all logs from all clients
func (a *Aggregator) GetAllLogs() map[string][]*cluster.LogEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string][]*cluster.LogEntry)
	for clientID, logs := range a.clientLogs {
		result[clientID] = append([]*cluster.LogEntry{}, logs...)
	}

	return result
}

// ClearLogs clears logs from a specific client
func (a *Aggregator) ClearLogs(clientID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.clientLogs, clientID)
}

// GetClientCount returns the number of clients with logs
func (a *Aggregator) GetClientCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.clientLogs)
}

// cleanupLoop periodically cleans up old logs
func (a *Aggregator) cleanupLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(time.Duration(a.config.FlushInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.cleanup()
		}
	}
}

// cleanup removes old logs to prevent memory bloat
func (a *Aggregator) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for clientID, logs := range a.clientLogs {
		if len(logs) > a.config.MaxBufferSize/100 {
			// Keep only recent logs
			a.clientLogs[clientID] = logs[len(logs)/2:]
		}
	}
}
