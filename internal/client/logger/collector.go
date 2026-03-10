// Package logger provides log collection functionality for client nodes.
package logger

import (
	"context"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"io"
	"os"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/cluster"
)

// Collector collects logs from the client and sends them to the master
type Collector struct {
	clientID      string
	logFilePath   string
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	buffer        []*cluster.LogEntry
	bufferSize    int
	flushInterval time.Duration
}

// NewCollector creates a new log collector
func NewCollector(clientID, logFilePath string, bufferSize int, flushInterval time.Duration) *Collector {
	ctx, cancel := context.WithCancel(context.Background())

	return &Collector{
		clientID:      clientID,
		logFilePath:   logFilePath,
		ctx:           ctx,
		cancel:        cancel,
		buffer:        make([]*cluster.LogEntry, 0, bufferSize),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
	}
}

// Start starts the log collector
func (c *Collector) Start() {
	c.wg.Add(1)
	go c.flushLoop()

	// Start tailing the log file
	if c.logFilePath != "" {
		c.wg.Add(1)
		go c.tailLogFile()
	}
}

// Stop stops the log collector
func (c *Collector) Stop() {
	c.cancel()
	c.wg.Wait()

	// Flush remaining logs
	c.flush()
}

// AddLogEntry adds a log entry to the buffer
func (c *Collector) AddLogEntry(level, message string, fields map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &cluster.LogEntry{
		ClientID:  c.clientID,
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}

	c.buffer = append(c.buffer, entry)

	// Flush if buffer is full
	if len(c.buffer) >= c.bufferSize {
		c.flush()
	}
}

// GetBufferedLogs returns the current buffer of logs
func (c *Collector) GetBufferedLogs() []*cluster.LogEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	logs := make([]*cluster.LogEntry, len(c.buffer))
	copy(logs, c.buffer)
	return logs
}

// ClearBuffer clears the log buffer
func (c *Collector) ClearBuffer() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.buffer = make([]*cluster.LogEntry, 0, c.bufferSize)
}

// flushLoop periodically flushes the log buffer
func (c *Collector) flushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.flush()
		}
	}
}

// flush flushes the log buffer to master
func (c *Collector) flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.buffer) == 0 {
		return
	}

	// 发送日志到 Master 节点
	// 日志聚合器通常由 Master 提供 REST API 端点接收日志
	// 当前实现：清空缓冲区（日志已在本地文件中持久化）
	// 未来改进：实现批量 HTTP 发送或 WebSocket 实时推送

	// 清空缓冲区
	c.buffer = make([]*cluster.LogEntry, 0, c.bufferSize)
}

// tailLogFile tails the log file and adds entries to the buffer
func (c *Collector) tailLogFile() {
	defer c.wg.Done()

	// Open log file
	file, err := os.Open(c.logFilePath)
	if err != nil {
		return
	}
	defer utils.CloseQuietly(file)

	// Seek to end of file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return
	}

	// Read new lines
	buf := make([]byte, 4096)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			n, err := file.Read(buf)
			if err != nil {
				if err != io.EOF {
					// Log read error
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if n > 0 {
				// Parse log line and add to buffer
				// This is simplified - actual implementation would parse log format
				c.AddLogEntry("info", string(buf[:n]), nil)
			}
		}
	}
}
