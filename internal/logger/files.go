// Package logger provides log file management functionality.
package logger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// LogFileInfo represents information about a log file
type LogFileInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Role      string    `json:"role"` // Node role: master, client, hybrid
	Date      string    `json:"date"`
	CreatedAt time.Time `json:"createdAt"`
	IsBackup  bool      `json:"isBackup"`
}

// ParsedLogEntry represents a parsed log entry from file
type ParsedLogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Raw       string                 `json:"raw"` // Original raw line
}

// LogFileFilter contains filter options for reading log files
type LogFileFilter struct {
	Level     string    `json:"level"`     // Filter by log level
	Search    string    `json:"search"`    // Search in message
	Offset    int       `json:"offset"`    // Skip N entries
	Limit     int       `json:"limit"`     // Max N entries (0 = unlimited)
	StartTime time.Time `json:"startTime"` // Filter by start time
	EndTime   time.Time `json:"endTime"`   // Filter by end time
}

// ListLogFiles lists all available log files in the log directory
func ListLogFiles(logDir string, role string) ([]LogFileInfo, error) {
	if logDir == "" {
		return nil, fmt.Errorf("日志目录未配置")
	}

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("无法访问日志目录: %w", err)
	}

	files, err := os.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("读取日志目录失败: %w", err)
	}

	var logFiles []LogFileInfo

	// Pattern: shepherd-{mode}-{date} {time}.log or shepherd-{mode}-{date} {time}-{timestamp}-{reason}.log
	// Supports log files with timestamps: shepherd-hybrid-2026-02-26 21-43-20.log
	pattern := regexp.MustCompile(`^shepherd-([a-z]+)-(\d{4}-\d{2}-\d{2})(?:\s(\d{2}-\d{2}-\d{2}))?(?:-(\d{8}-\d{6})-(.+))?\.log$`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		matches := pattern.FindStringSubmatch(name)
		if matches == nil {
			continue
		}

		// Extract file info
		info, err := file.Info()
		if err != nil {
			continue
		}

		fileMode := matches[1]
		fileDate := matches[2]

		// Only show files for current role or all if role is empty
		if role != "" && role != fileMode {
			continue
		}

		logFileInfo := LogFileInfo{
			Name:      name,
			Path:      filepath.Join(logDir, name),
			Size:      info.Size(),
			Role:      fileMode,
			Date:      fileDate,
			CreatedAt: info.ModTime(),
			IsBackup:  matches[4] != "", // Has rotation timestamp and reason
		}

		logFiles = append(logFiles, logFileInfo)
	}

	// Sort by creation time (newest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].CreatedAt.After(logFiles[j].CreatedAt)
	})

	return logFiles, nil
}

// ReadLogFile reads a log file and returns filtered entries
func ReadLogFile(logPath string, filter LogFileFilter) ([]ParsedLogEntry, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer closeQuietly(file)

	var entries []ParsedLogEntry
	scanner := bufio.NewScanner(file)

	// Read and parse lines
	for scanner.Scan() {
		line := scanner.Text()
		entry := parseLogLineToEntry(line)
		if entry == nil {
			continue
		}

		// Apply filters
		if !matchesFilter(entry, filter) {
			continue
		}

		entries = append(entries, *entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %w", err)
	}

	// Apply pagination
	if filter.Offset > 0 {
		if filter.Offset >= len(entries) {
			return []ParsedLogEntry{}, nil
		}
		entries = entries[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(entries) {
		entries = entries[:filter.Limit]
	}

	return entries, nil
}

// GetLogFileStats returns statistics about a log file
func GetLogFileStats(logPath string) (map[string]int, error) {
	entries, err := ReadLogFile(logPath, LogFileFilter{})
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int)
	total := 0

	for _, entry := range entries {
		level := strings.ToUpper(entry.Level)
		stats[level]++
		total++
	}

	stats["total"] = total
	return stats, nil
}

// parseLogLineToEntry parses a log line into a ParsedLogEntry
func parseLogLineToEntry(line string) *ParsedLogEntry {
	if line == "" {
		return nil
	}

	entry := &ParsedLogEntry{
		Fields: make(map[string]interface{}),
		Raw:    line,
	}

	// Try JSON format first
	if strings.HasPrefix(line, "{") {
		if err := parseJSONLogLine(line, entry); err == nil {
			return entry
		}
	}

	// Try text format: [2025-02-26 20:17:51] [file.go:72] INFO 消息内容
	textPattern := regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\](?: \[([^\]]+)\])? (\w+) (.+)`)
	matches := textPattern.FindStringSubmatch(line)
	if matches != nil {
		// Extract timestamp from the original line (more reliable)
		timestampStr := line[1:20] // Skip '[' and take exactly "YYYY-MM-DD HH:MM:SS"
		if timestamp, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
			entry.Timestamp = timestamp
		}
		entry.Caller = matches[1]
		entry.Level = strings.ToUpper(matches[2])
		entry.Message = matches[3]

		// Parse key=value fields from message
		parseTextFields(entry.Message, entry.Fields)

		return entry
	}

	// Fallback: treat as simple message
	entry.Level = "INFO"
	entry.Message = line
	entry.Timestamp = time.Now()

	return entry
}

// parseJSONLogLine parses a JSON format log line
func parseJSONLogLine(line string, entry *ParsedLogEntry) error {
	// Simple JSON parsing (avoiding encoding/json for performance)
	// Format: {"time":"...","level":"...","msg":"...",...}

	fields := make(map[string]string)

	// Extract key-value pairs using regex
	pattern := regexp.MustCompile(`"(\w+)"\s*:\s*"([^"]*)"`)
	matches := pattern.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		if len(match) == 3 {
			fields[match[1]] = match[2]
		}
	}

	// Map fields
	if t, ok := fields["time"]; ok {
		// Try multiple time formats
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",     // JSON log format
			"2006-01-02T15:04:05Z",    // ISO format
			"2006-01-02T15:04:05+08:00", // ISO with timezone
		}
		for _, format := range formats {
			if timestamp, err := time.Parse(format, t); err == nil {
				entry.Timestamp = timestamp
				break
			}
		}
	}
	if l, ok := fields["level"]; ok {
		entry.Level = strings.ToUpper(l)
	}
	if m, ok := fields["msg"]; ok {
		entry.Message = m
	}
	if c, ok := fields["caller"]; ok {
		entry.Caller = c
	}

	// Store other fields
	for k, v := range fields {
		if k != "time" && k != "level" && k != "msg" && k != "caller" {
			entry.Fields[k] = v
		}
	}

	return nil
}

// parseTextFields extracts key=value pairs from message text
func parseTextFields(message string, fields map[string]interface{}) {
	// Pattern: key=value or key="value"
	pattern := regexp.MustCompile(`(\w+)=("[^"]*"|[\w\-./]+)`)
	matches := pattern.FindAllStringSubmatch(message, -1)

	for _, match := range matches {
		if len(match) == 3 {
			key := match[1]
			value := strings.Trim(match[2], `"`)
			fields[key] = value
		}
	}
}

// matchesFilter checks if a log entry matches the given filter
func matchesFilter(entry *ParsedLogEntry, filter LogFileFilter) bool {
	// Level filter
	if filter.Level != "" && !strings.EqualFold(entry.Level, filter.Level) {
		return false
	}

	// Search filter
	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		messageLower := strings.ToLower(entry.Message)
		callerLower := strings.ToLower(entry.Caller)

		if !strings.Contains(messageLower, searchLower) &&
			!strings.Contains(callerLower, searchLower) {
			return false
		}
	}

	// Time range filter
	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}

	return true
}

// closeQuietly closes a file and ignores the error (used in defer)
func closeQuietly(c io.Closer) {
	_ = c.Close()
}
