package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StreamLogEntry represents a single log entry for streaming
type StreamLogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LogStream manages a stream of log entries
type LogStream struct {
	mu          sync.RWMutex
	entries     []StreamLogEntry
	maxSize     int
	subscribers map[chan StreamLogEntry]struct{}
	closed      bool
}

// Add adds a log entry to the stream
func (ls *LogStream) Add(entry StreamLogEntry) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if ls.closed {
		return
	}

	ls.entries = append(ls.entries, entry)

	if len(ls.entries) > ls.maxSize {
		ls.entries = ls.entries[1:]
	}

	for ch := range ls.subscribers {
		select {
		case ch <- entry:
		default:
		}
	}
}

// Subscribe subscribes to log entries
func (ls *LogStream) Subscribe() chan StreamLogEntry {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ch := make(chan StreamLogEntry, 100)
	ls.subscribers[ch] = struct{}{}
	return ch
}

// Unsubscribe unsubscribes from log entries
func (ls *LogStream) Unsubscribe(ch chan StreamLogEntry) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	close(ch)
	delete(ls.subscribers, ch)
}

// GetEntries returns recent log entries
func (ls *LogStream) GetEntries(limit int) []StreamLogEntry {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	if limit <= 0 || limit > len(ls.entries) {
		limit = len(ls.entries)
	}

	start := len(ls.entries) - limit
	if start < 0 {
		start = 0
	}

	result := make([]StreamLogEntry, limit)
	copy(result, ls.entries[start:])
	return result
}

// Close closes the log stream
func (ls *LogStream) Close() {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.closed = true

	for ch := range ls.subscribers {
		close(ch)
	}
	ls.subscribers = make(map[chan StreamLogEntry]struct{})
}

var (
	globalLogStream *LogStream
	onceStream      sync.Once
)

// InitLogStream initializes the global log stream
func InitLogStream(maxSize int) {
	onceStream.Do(func() {
		globalLogStream = &LogStream{
			entries:     make([]StreamLogEntry, 0, maxSize),
			maxSize:     maxSize,
			subscribers: make(map[chan StreamLogEntry]struct{}),
		}
	})
}

// GetLogStream returns the global log stream
func GetLogStream() *LogStream {
	if globalLogStream == nil {
		InitLogStream(1000)
	}
	return globalLogStream
}

// GetLatestLogFile returns the path to the latest log file
func GetLatestLogFile(logDir string, serverMode string) (string, error) {
	if logDir == "" {
		return "", fmt.Errorf("log directory not configured")
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	date := time.Now().Format("2006-01-02")
	logFileName := fmt.Sprintf("shepherd-%s-%s.log", serverMode, date)
	logPath := filepath.Join(logDir, logFileName)

	return logPath, nil
}
