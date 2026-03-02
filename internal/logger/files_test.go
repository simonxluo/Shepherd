package logger

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestListLogFiles(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test log files
	files := []string{
		"shepherd-standalone-2025-02-26.log",
		"shepherd-master-2025-02-26.log",
		"shepherd-client-2025-02-25.log",
		"shepherd-standalone-2025-02-25-20260226-120000-size.log",
		"not-a-log-file.txt",
	}

	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		// Set modification time
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Test listing all files
	logFiles, err := ListLogFiles(tempDir, "")
	if err != nil {
		t.Fatalf("ListLogFiles failed: %v", err)
	}

	// Should return 4 shepherd log files (not the txt file)
	if len(logFiles) != 4 {
		t.Errorf("Expected 4 log files, got %d", len(logFiles))
	}

	// Test filtering by mode
	standaloneFiles, err := ListLogFiles(tempDir, "standalone")
	if err != nil {
		t.Fatalf("ListLogFiles with mode failed: %v", err)
	}

	if len(standaloneFiles) != 2 {
		t.Errorf("Expected 2 standalone log files, got %d", len(standaloneFiles))
	}

	// Verify file info
	for _, file := range logFiles {
		if file.Name == "" {
			t.Error("File name is empty")
		}
		if file.Path == "" {
			t.Error("File path is empty")
		}
		if file.Role == "" {
			t.Error("File role is empty")
		}
	}
}

func TestReadLogFile(t *testing.T) {
	// Create temp log file
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Write test log content
	content := `[2025-02-26 20:17:51] [server.go:72] INFO 模型加载完成 modelCount=1
[2025-02-26 20:17:52] [server.go:100] ERROR 模型加载失败 error=file not found
[2025-02-26 20:17:53] [server.go:80] DEBUG 调试信息
[2025-02-26 20:17:54] [server.go:90] WARN 警告信息`

	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Test reading all entries
	entries, err := ReadLogFile(logFile, LogFileFilter{})
	if err != nil {
		t.Fatalf("ReadLogFile failed: %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("Expected 4 log entries, got %d", len(entries))
	}

	// Test level filter
	errorEntries, err := ReadLogFile(logFile, LogFileFilter{Level: "ERROR"})
	if err != nil {
		t.Fatalf("ReadLogFile with level filter failed: %v", err)
	}

	if len(errorEntries) != 1 {
		t.Errorf("Expected 1 ERROR entry, got %d", len(errorEntries))
		// Debug: print all entries and their levels
		allEntries, _ := ReadLogFile(logFile, LogFileFilter{})
		for i, e := range allEntries {
			t.Logf("Entry %d: Level='%s' Message='%s'", i, e.Level, e.Message)
		}
	}

	// Test search filter
	searchEntries, err := ReadLogFile(logFile, LogFileFilter{Search: "加载"})
	if err != nil {
		t.Fatalf("ReadLogFile with search filter failed: %v", err)
	}

	if len(searchEntries) != 2 {
		t.Errorf("Expected 2 entries with '加载', got %d", len(searchEntries))
	}

	// Test pagination
	pagedEntries, err := ReadLogFile(logFile, LogFileFilter{Offset: 1, Limit: 2})
	if err != nil {
		t.Fatalf("ReadLogFile with pagination failed: %v", err)
	}

	if len(pagedEntries) != 2 {
		t.Errorf("Expected 2 paged entries, got %d", len(pagedEntries))
	}

	// Verify entry content
	for _, entry := range entries {
		if entry.Timestamp.IsZero() {
			t.Error("Entry timestamp is zero")
		}
		if entry.Level == "" {
			t.Error("Entry level is empty")
		}
		if entry.Message == "" {
			t.Error("Entry message is empty")
		}
		if entry.Raw == "" {
			t.Error("Entry raw is empty")
		}
	}
}

func TestGetLogFileStats(t *testing.T) {
	// Create temp log file
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Write test log content
	content := `[2025-02-26 20:17:51] INFO 模型加载完成
[2025-02-26 20:17:52] ERROR 模型加载失败
[2025-02-26 20:17:53] DEBUG 调试信息
[2025-02-26 20:17:54] WARN 警告信息
[2025-02-26 20:17:55] INFO 另一条信息`

	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Get stats
	stats, err := GetLogFileStats(logFile)
	if err != nil {
		t.Fatalf("GetLogFileStats failed: %v", err)
	}

	// Verify stats
	total := stats["total"]
	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	infoCount := stats["INFO"]
	if infoCount != 2 {
		t.Errorf("Expected INFO count 2, got %d", infoCount)
	}

	errorCount := stats["ERROR"]
	if errorCount != 1 {
		t.Errorf("Expected ERROR count 1, got %d", errorCount)
	}
}

func TestIsSafeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "valid standalone log",
			filename: "shepherd-standalone-2025-02-26.log",
			want:     true,
		},
		{
			name:     "valid backup log",
			filename: "shepherd-master-2025-02-25-20260226-120000-size.log",
			want:     true,
		},
		{
			name:     "path traversal attack",
			filename: "../../../etc/passwd",
			want:     false,
		},
		{
			name:     "wrong extension",
			filename: "shepherd-standalone-2025-02-26.txt",
			want:     false,
		},
		{
			name:     "invalid format",
			filename: "random-file.log",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSafeFilename(tt.filename); got != tt.want {
				t.Errorf("isSafeFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function (copied from server.go for testing)
func isSafeFilename(filename string) bool {
	// Check for path traversal attempts
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return false
	}

	// Check file extension
	if !strings.HasSuffix(filename, ".log") {
		return false
	}

	// Check filename format (basic pattern match)
	// Format: shepherd-{mode}-{date}.log or shepherd-{mode}-{date}-{timestamp}-{reason}.log
	pattern := regexp.MustCompile(`^shepherd-[a-z]+-\d{4}-\d{2}-\d{2}(?:-\d{8}-\d{6}-[a-z]+)?\.log$`)
	return pattern.MatchString(filename)
}
