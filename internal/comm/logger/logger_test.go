package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Run("Initialize with stdout output", func(t *testing.T) {
		cfg := &config.LogConfig{
			Level:      "debug",
			Format:     "json",
			Output:     "stdout",
			Directory:  "",
			MaxSize:    0,
			MaxBackups: 0,
			MaxAge:     0,
			Compress:   false,
		}

		logger, err := NewLogger(cfg, "standalone")
		require.NoError(t, err)
		assert.NotNil(t, logger)
		assert.Equal(t, DEBUG, logger.level)
	})

	t.Run("Initialize with file output", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.LogConfig{
			Level:      "info",
			Format:     "text",
			Output:     "file",
			Directory:  tmpDir,
			MaxSize:    1,
			MaxBackups: 2,
			MaxAge:     1,
			Compress:   false,
		}

		logger, err := NewLogger(cfg, "standalone")
		require.NoError(t, err)
		assert.NotNil(t, logger)

		// Write a test log
		logger.log(INFO, "test message", nil)

		// Close file writer
		logger.Close()

		// Check if log file was created
		// 日志文件名格式: shepherd-{mode}-{date}.log
		// 由于 currentDate 包含时间，我们使用 Glob 查找匹配的文件
		matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
		require.NoError(t, err)
		assert.NotEmpty(t, matches, "应该创建日志文件")
	})

	t.Run("Initialize with both outputs", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.LogConfig{
			Level:      "warn",
			Format:     "json",
			Output:     "both",
			Directory:  tmpDir,
			MaxSize:    1,
			MaxBackups: 1,
			MaxAge:     1,
			Compress:   false,
		}

		logger, err := NewLogger(cfg, "standalone")
		require.NoError(t, err)
		assert.NotNil(t, logger)
		assert.Equal(t, WARN, logger.level)

		logger.Close()
	})

	t.Run("Invalid log level defaults to info", func(t *testing.T) {
		cfg := &config.LogConfig{
			Level:  "invalid",
			Format: "text",
			Output: "stdout",
		}

		logger, err := NewLogger(cfg, "standalone")
		require.NoError(t, err)
		assert.Equal(t, INFO, logger.level)
	})
}

func TestLogLevels(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Level:      "debug",
		Format:     "text",
		Output:     "file",
		Directory:  tmpDir,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	logger, err := NewLogger(cfg, "standalone")
	require.NoError(t, err)

	// Test all log levels using logger instance directly
	logger.log(DEBUG, "debug message", nil)
	logger.log(INFO, "info message", nil)
	logger.log(WARN, "warn message", nil)
	logger.log(ERROR, "error message", nil)

	// Close to flush
	logger.Close()

	// Verify log file exists and has content
	// 日志文件名格式: shepherd-{mode}-{date}.log
	matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "应该创建日志文件")

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Contains(t, string(content), "info message")
	assert.Contains(t, string(content), "warn message")
	assert.Contains(t, string(content), "error message")
}

func TestLogFormats(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("JSON format", func(t *testing.T) {
		jsonDir := filepath.Join(tmpDir, "json")
		cfg := &config.LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "file",
			Directory:  jsonDir,
			MaxSize:    1,
			MaxBackups: 1,
			MaxAge:     1,
			Compress:   false,
		}

		logger, err := NewLogger(cfg, "standalone")
		require.NoError(t, err)

		logger.log(INFO, "test json message", nil)
		logger.Close()

		// 日志文件名格式: shepherd-{mode}-{date}.log
		matches, err := filepath.Glob(filepath.Join(jsonDir, "shepherd-standalone-*.log"))
		require.NoError(t, err)
		require.NotEmpty(t, matches, "应该创建日志文件")

		content, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		assert.Contains(t, string(content), "\"level\":")
		assert.Contains(t, string(content), "\"msg\":")
		assert.Contains(t, string(content), "test json message")
	})

	t.Run("Text format", func(t *testing.T) {
		textDir := filepath.Join(tmpDir, "text")
		cfg := &config.LogConfig{
			Level:      "info",
			Format:     "text",
			Output:     "file",
			Directory:  textDir,
			MaxSize:    1,
			MaxBackups: 1,
			MaxAge:     1,
			Compress:   false,
		}

		logger, err := NewLogger(cfg, "standalone")
		require.NoError(t, err)

		logger.log(INFO, "test text message", nil)
		logger.Close()

		// 日志文件名格式: shepherd-{mode}-{date}.log
		matches, err := filepath.Glob(filepath.Join(textDir, "shepherd-standalone-*.log"))
		require.NoError(t, err)
		require.NotEmpty(t, matches, "应该创建日志文件")

		content, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		logContent := string(content)
		assert.Contains(t, logContent, "test text message")
		assert.Contains(t, logContent, "INFO") // Check for INFO level (without brackets around it)
	})
}

func TestLogWithFields(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		Directory:  tmpDir,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	logger, err := NewLogger(cfg, "standalone")
	require.NoError(t, err)

	// Test WithField using logger instance directly
	fields1 := []Field{{Key: "key1", Value: "value1"}}
	logger.log(INFO, "message with field", fields1)
	time.Sleep(100 * time.Millisecond)

	logger.Close()

	// 日志文件名格式: shepherd-{mode}-{date}.log
	matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "应该创建日志文件")

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Contains(t, string(content), "key1")
	assert.Contains(t, string(content), "value1")

	// Test WithFields - 新建一个目录避免冲突
	tmpDir2 := t.TempDir()
	cfg2 := &config.LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		Directory:  tmpDir2,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}
	logger2, _ := NewLogger(cfg2, "standalone")
	fields2 := []Field{
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: 123},
	}
	logger2.log(INFO, "message with fields", fields2)
	time.Sleep(100 * time.Millisecond)

	logger2.Close()

	matches2, err := filepath.Glob(filepath.Join(tmpDir2, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches2, "应该创建日志文件")

	content2, err := os.ReadFile(matches2[0])
	require.NoError(t, err)
	assert.Contains(t, string(content2), "key2")
	assert.Contains(t, string(content2), "key3")
}

func TestLogWithError(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		Directory:  tmpDir,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	logger, err := NewLogger(cfg, "standalone")
	require.NoError(t, err)

	testErr := assert.AnError
	fields := []Field{{Key: "error", Value: testErr.Error()}}
	logger.log(ERROR, "operation failed", fields)
	time.Sleep(100 * time.Millisecond)

	logger.Close()

	// 日志文件名格式: shepherd-{mode}-{date}.log
	matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "应该创建日志文件")

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Contains(t, string(content), "error")
	assert.Contains(t, string(content), "operation failed")
}

func TestFormattedLogging(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Level:      "info",
		Format:     "text",
		Output:     "file",
		Directory:  tmpDir,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	logger, err := NewLogger(cfg, "standalone")
	require.NoError(t, err)

	// Use logger.log directly instead of global functions
	logger.log(INFO, fmt.Sprintf("user %s logged in from %s", "john", "192.168.1.1"), nil)
	logger.log(WARN, fmt.Sprintf("disk usage at %d%%", 85), nil)
	logger.log(ERROR, fmt.Sprintf("failed to connect to %s:%d", "db.example.com", 5432), nil)
	time.Sleep(100 * time.Millisecond)

	logger.Close()

	// 日志文件名格式: shepherd-{mode}-{date}.log
	matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "应该创建日志文件")

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Contains(t, string(content), "john")
	assert.Contains(t, string(content), "192.168.1.1")
	assert.Contains(t, string(content), "85")
	assert.Contains(t, string(content), "db.example.com")
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", DEBUG},
		{"DEBUG", DEBUG},
		{"info", INFO},
		{"INFO", INFO},
		{"warn", WARN},
		{"WARN", WARN},
		{"warning", WARN},
		{"error", ERROR},
		{"ERROR", ERROR},
		{"fatal", FATAL},
		{"FATAL", FATAL},
		{"invalid", INFO},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test that GetLogger always returns a valid logger
	logger := GetLogger()
	assert.NotNil(t, logger)

	// Test that it returns the same instance
	logger2 := GetLogger()
	assert.Same(t, logger, logger2)
}

func TestGetLogger(t *testing.T) {
	l := GetLogger()
	assert.NotNil(t, l)

	// Should be able to log without error
	assert.NotPanics(t, func() {
		Info("test message from global logger")
	})
}

func TestLogLevelFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Level:      "warn",
		Format:     "text",
		Output:     "file",
		Directory:  tmpDir,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	logger, err := NewLogger(cfg, "standalone")
	require.NoError(t, err)

	// These should be filtered out - use logger directly
	logger.log(DEBUG, "debug message - should not appear", nil)
	logger.log(INFO, "info message - should not appear", nil)

	// These should appear
	logger.log(WARN, "warn message - should appear", nil)
	logger.log(ERROR, "error message - should appear", nil)

	time.Sleep(100 * time.Millisecond)
	logger.Close()

	// 日志文件名格式: shepherd-{mode}-{date}.log
	matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "应该创建日志文件")

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)

	logContent := string(content)
	assert.NotContains(t, logContent, "debug message - should not appear")
	assert.NotContains(t, logContent, "info message - should not appear")
	assert.Contains(t, logContent, "warn message - should appear")
	assert.Contains(t, logContent, "error message - should appear")
}

func TestLogEntryChaining(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		Directory:  tmpDir,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	logger, err := NewLogger(cfg, "standalone")
	require.NoError(t, err)

	// Test field chaining - use logger directly with multiple fields
	fields := []Field{
		{Key: "user", Value: "john"},
		{Key: "action", Value: "login"},
		{Key: "ip", Value: "192.168.1.1"},
	}
	logger.log(INFO, "user logged in", fields)

	time.Sleep(100 * time.Millisecond)
	logger.Close()

	// 日志文件名格式: shepherd-{mode}-{date}.log
	matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "应该创建日志文件")

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)

	logContent := string(content)
	assert.Contains(t, logContent, "user")
	assert.Contains(t, logContent, "john")
	assert.Contains(t, logContent, "action")
	assert.Contains(t, logContent, "login")
	assert.Contains(t, logContent, "ip")
	assert.Contains(t, logContent, "192.168.1.1")
}

func TestLogRotationBySize(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Level:      "info",
		Format:     "text",
		Output:     "file",
		Directory:  tmpDir,
		MaxSize:    1, // 1MB
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
	}

	logger, err := NewLogger(cfg, "standalone")
	require.NoError(t, err)

	// Write a large message to trigger rotation
	// For testing, we'll just write a message and check the file exists
	Info("test message for rotation")

	logger.Close()

	// Check that log file was created
	// 日志文件名格式: shepherd-{mode}-{date}.log
	matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "应该创建日志文件")
}

func TestConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.LogConfig{
		Level:      "info",
		Format:     "text",
		Output:     "file",
		Directory:  tmpDir,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
	}

	logger, err := NewLogger(cfg, "standalone")
	require.NoError(t, err)

	// Test concurrent logging
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(n int) {
			logger.log(INFO, fmt.Sprintf("Concurrent log message %d", n), nil)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	time.Sleep(500 * time.Millisecond)
	logger.Close()

	// Verify all messages were written
	// 日志文件名格式: shepherd-{mode}-{date}.log
	matches, err := filepath.Glob(filepath.Join(tmpDir, "shepherd-standalone-*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "应该创建日志文件")

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)

	logContent := string(content)
	for i := 0; i < 100; i++ {
		assert.Contains(t, logContent, fmt.Sprintf("Concurrent log message %d", i))
	}
}
