package logger

import (
	"bufio"
	"fmt"
	"io"
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

// NewLogStream creates a new log stream
func NewLogStream(maxSize int) *LogStream {
	return &LogStream{
		entries:     make([]StreamLogEntry, 0, maxSize),
		maxSize:     maxSize,
		subscribers: make(map[chan StreamLogEntry]struct{}),
		closed:      false,
	}
}

// Add adds a log entry to the stream
func (ls *LogStream) Add(entry StreamLogEntry) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if ls.closed {
		return
	}

	// Add to buffer
	ls.entries = append(ls.entries, entry)

	// Keep buffer size under limit
	if len(ls.entries) > ls.maxSize {
		// Remove oldest entry
		ls.entries = ls.entries[1:]
	}

	// Send to subscribers
	for ch := range ls.subscribers {
		select {
		case ch <- entry:
		default:
			// Channel is full, skip this subscriber
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

	// Return most recent entries
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

	// Close all subscriber channels
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
		globalLogStream = NewLogStream(maxSize)
	})
}

// GetLogStream returns the global log stream
func GetLogStream() *LogStream {
	if globalLogStream == nil {
		InitLogStream(1000)
	}
	return globalLogStream
}

// StreamLogFile streams log entries from a log file
func StreamLogFile(logPath string, fromBeginning bool) (<-chan StreamLogEntry, error) {
	// Open log file
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	ch := make(chan StreamLogEntry, 100)

	go func() {
		defer close(ch)
		defer closeQuietly(file)

		// If not from beginning, seek to end
		if !fromBeginning {
			_, err := file.Seek(0, io.SeekEnd)
			if err != nil {
				return
			}
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			// Parse log line (simple parsing)
			entry := parseLogLine(line)
			if entry != nil {
				ch <- *entry
			}
		}
	}()

	return ch, nil
}

// GetLatestLogFile returns the path to the latest log file
func GetLatestLogFile(logDir string, serverMode string) (string, error) {
	if logDir == "" {
		return "", fmt.Errorf("log directory not configured")
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Log file name format: shepherd-{mode}-{date}.log
	date := time.Now().Format("2006-01-02")
	logFileName := fmt.Sprintf("shepherd-%s-%s.log", serverMode, date)
	logPath := filepath.Join(logDir, logFileName)

	return logPath, nil
}

// parseLogLine parses a log line into a LogEntry
func parseLogLine(line string) *StreamLogEntry {
	if line == "" {
		return nil
	}

	// Simple parsing - handle both JSON and text formats
	// For now, just create a basic entry
	entry := &StreamLogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   line,
		Fields:    make(map[string]interface{}),
	}

	return entry
}
