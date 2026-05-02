// Package logger provides structured logging with file rotation support.
// It uses a simple custom logger implementation to avoid external dependencies.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// Logger is the main logger structure
type Logger struct {
	mu          sync.Mutex
	level       LogLevel
	formatJSON  bool
	outputs     []io.Writer
	fileWriter  io.WriteCloser
	monitor     *LogMonitor
	logDir      string
	maxSize     int64 // MB
	maxBackups  int
	maxAge      int // days
	currentSize int64
	currentDate string
	role        string // master, client, hybrid
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// InitLogger initializes the global logger with the given configuration
func InitLogger(cfg *config.LogConfig, role string) error {
	logger, err := NewLogger(cfg, role)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// NewLogger creates a new logger instance
func NewLogger(cfg *config.LogConfig, role string) (*Logger, error) {
	l := &Logger{
		level:       parseLevel(cfg.Level),
		formatJSON:  cfg.Format == "json",
		outputs:     []io.Writer{},
		monitor:     NewLogMonitor(),
		logDir:      cfg.Directory,
		maxSize:     int64(cfg.MaxSize),
		maxBackups:  cfg.MaxBackups,
		maxAge:      cfg.MaxAge,
		currentSize: 0,
		currentDate: time.Now().Format("2006-01-02 15-04-05"),
		role:        role,
	}

	switch strings.ToLower(cfg.Output) {
	case "stdout":
		l.outputs = append(l.outputs, l.monitor)
	case "file":
		if err := l.setupFileWriter(); err != nil {
			return nil, err
		}
		l.outputs = append(l.outputs, l.monitor)
	case "both":
		l.outputs = append(l.outputs, l.monitor)
		if err := l.setupFileWriter(); err != nil {
			return nil, err
		}
	default:
		l.outputs = append(l.outputs, l.monitor)
	}

	return l, nil
}

func (l *Logger) setupFileWriter() error {
	// Ensure log directory exists
	if err := os.MkdirAll(l.logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 使用新的日志文件命名格式: shepherd-{role}-{date}.log
	logFileName := fmt.Sprintf("shepherd-%s-%s.log", l.role, l.currentDate)
	logFile := filepath.Join(l.logDir, logFileName)

	// Check if file exists and get current size
	if info, err := os.Stat(logFile); err == nil {
		l.currentSize = info.Size()
	}

	// Open file in append mode
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	l.fileWriter = f
	l.outputs = append(l.outputs, f)

	// Start rotation checker
	go l.rotationChecker()

	return nil
}

// rotationChecker periodically checks if log rotation is needed
func (l *Logger) rotationChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.checkRotation()
	}
}

func (l *Logger) checkRotation() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	currentDate := now.Format("2006-01-02")

	// Rotate on date change
	if currentDate != l.currentDate {
		l.rotateLog("date")
		l.currentDate = currentDate
		return
	}

	// Rotate on size
	if l.currentSize >= l.maxSize*1024*1024 {
		l.rotateLog("size")
	}
}

func (l *Logger) rotateLog(reason string) {
	if l.fileWriter == nil {
		return
	}

	// 第一步：先从 outputs 中移除旧的 fileWriter，防止其他 goroutine 继续使用
	newOutputs := []io.Writer{}
	for _, w := range l.outputs {
		if w != l.fileWriter {
			newOutputs = append(newOutputs, w)
		}
	}
	l.outputs = newOutputs

	// 第二步：安全地关闭文件
	if err := l.fileWriter.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] 关闭日志文件失败: %v\n", err)
	}

	// 第三步：重命名当前日志文件为备份
	// 格式: shepherd-{role}-{date}-{timestamp}-{reason}.log
	logFileName := fmt.Sprintf("shepherd-%s-%s.log", l.role, l.currentDate)
	logFile := filepath.Join(l.logDir, logFileName)
	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(l.logDir, fmt.Sprintf("shepherd-%s-%s-%s-%s.log", l.role, l.currentDate, timestamp, reason))

	if err := os.Rename(logFile, backupFile); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] 重命名日志文件失败: %v\n", err)
	}

	// 第四步：清理旧备份
	l.cleanOldBackups()

	// 第五步：创建新日志文件
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] 创建新日志文件失败: %v\n", err)
		return
	}

	// 第六步：更新 fileWriter 并添加到 outputs
	l.fileWriter = f
	l.currentSize = 0
	l.outputs = append(l.outputs, f)
}

func (l *Logger) cleanOldBackups() {
	files, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	// Group backup files by date and mode
	// 新格式: shepherd-{mode}-{date}-{timestamp}-{reason}.log
	backupsByDate := make(map[string][]string)
	for _, file := range files {
		name := file.Name()
		// 匹配新的日志文件格式
		if strings.HasPrefix(name, "shepherd-") && strings.HasSuffix(name, ".log") {
			// 提取日期部分 (格式: shepherd-role-date-timestamp-reason.log)
			parts := strings.Split(name, "-")
			if len(parts) >= 3 {
				// parts[0] = "shepherd"
				// parts[1] = role
				// parts[2] = date (YYYYMMDD)
				datePart := parts[2]
				// 只清理当前角色的备份日志
				if parts[1] == l.role {
					backupsByDate[datePart] = append(backupsByDate[datePart], name)
				}
			}
		}
	}

	// Remove old backups based on maxAge
	cutoffDate := time.Now().AddDate(0, 0, -l.maxAge)
	for dateStr, backupFiles := range backupsByDate {
		fileDate, err := time.Parse("20060102", dateStr)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoffDate) {
			for _, fileName := range backupFiles {
				removeQuietly(filepath.Join(l.logDir, fileName))
			}
		}
	}
}

// removeQuietly removes a file and ignores errors (file may not exist)
func removeQuietly(path string) {
	_ = os.Remove(path)
}

// parseLevel converts string level to LogLevel
func parseLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

func GetLogger() *Logger {
	if defaultLogger == nil {
		once.Do(func() {
			defaultLogger, _ = NewLogger(&config.LogConfig{
				Level:  "info",
				Format: "text",
				Output: "stdout",
			}, "hybrid")
		})
	}
	return defaultLogger
}

func GetMonitor() *LogMonitor {
	return GetLogger().monitor
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, msg string, fields []Field) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取调用者信息 (文件名和行号)
	// skip=2 跳过当前函数和调用 log 的函数 (如 Info/Debug 等)
	_, file, line, ok := runtime.Caller(2)
	caller := ""
	if ok {
		// 只保留文件名，不包含完整路径
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var logLine string

	if l.formatJSON {
		// JSON format - 顺序: time, caller, level, msg, fields
		fieldStr := ""
		if len(fields) > 0 {
			fieldPairs := make([]string, 0, len(fields)*2)
			for _, f := range fields {
				fieldPairs = append(fieldPairs, fmt.Sprintf(`"%s"`, f.Key), fmt.Sprintf(`"%v"`, f.Value))
			}
			fieldStr = "," + strings.Join(fieldPairs, ":")
		}
		callerStr := ""
		if caller != "" {
			callerStr = fmt.Sprintf(`,"caller":"%s"`, caller)
		}
		logLine = fmt.Sprintf(`{"time":"%s"%s,"level":"%s","msg":"%s"%s}`+"\n", timestamp, callerStr, level, msg, fieldStr)
	} else {
		// Text format - 顺序: [时间] [文件:行号] 级别 消息 字段...
		fieldStr := ""
		if len(fields) > 0 {
			fieldPairs := make([]string, 0, len(fields))
			for _, f := range fields {
				fieldPairs = append(fieldPairs, fmt.Sprintf("%s=%v", f.Key, f.Value))
			}
			fieldStr = " " + strings.Join(fieldPairs, " ")
		}
		callerStr := ""
		if caller != "" {
			callerStr = fmt.Sprintf(" [%s]", caller)
		}
		logLine = fmt.Sprintf("[%s]%s %s %s%s\n", timestamp, callerStr, level, msg, fieldStr)
	}

	for _, w := range l.outputs {
		n, err := w.Write([]byte(logLine))
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] 写入日志失败: %v\n", err)
			continue
		}
		if n != len(logLine) {
			fmt.Fprintf(os.Stderr, "[WARN] 写入不完整: %d/%d\n", n, len(logLine))
		}
		if f, ok := w.(*os.File); ok {
			fileInfo, err := f.Stat()
			if err == nil && (fileInfo.Mode()&os.ModeType) == 0 {
				if err := f.Sync(); err != nil {
					fmt.Fprintf(os.Stderr, "[WARN] 同步文件失败: %v\n", err)
				}
				l.currentSize = fileInfo.Size()
			}
		}
	}
}

// LogEntry represents a log entry with fields
type LogEntry struct {
	logger  *Logger
	fields  []Field
	context string
}

// WithField creates a log entry with a single field
func (l *Logger) WithField(key string, value interface{}) *LogEntry {
	return &LogEntry{
		logger:  l,
		fields:  []Field{{Key: key, Value: value}},
		context: "",
	}
}

// WithFields creates a log entry with multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *LogEntry {
	fieldList := make([]Field, 0, len(fields))
	for k, v := range fields {
		fieldList = append(fieldList, Field{Key: k, Value: v})
	}
	return &LogEntry{
		logger:  l,
		fields:  fieldList,
		context: "",
	}
}

// WithError creates a log entry with an error field
func (l *Logger) WithError(err error) *LogEntry {
	return &LogEntry{
		logger:  l,
		fields:  []Field{{Key: "error", Value: err.Error()}},
		context: "",
	}
}

// WithField adds a field to the log entry
func (e *LogEntry) WithField(key string, value interface{}) *LogEntry {
	e.fields = append(e.fields, Field{Key: key, Value: value})
	return e
}

// WithFields adds multiple fields to the log entry
func (e *LogEntry) WithFields(fields map[string]interface{}) *LogEntry {
	for k, v := range fields {
		e.fields = append(e.fields, Field{Key: k, Value: v})
	}
	return e
}

// WithError adds an error field to the log entry
func (e *LogEntry) WithError(err error) *LogEntry {
	e.fields = append(e.fields, Field{Key: "error", Value: err.Error()})
	return e
}

// Debug logs at debug level
func (e *LogEntry) Debug(args ...interface{}) {
	msg := fmt.Sprint(args...)
	e.logger.log(DEBUG, msg, e.fields)
}

// Debugf logs a formatted message at debug level
func (e *LogEntry) Debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	e.logger.log(DEBUG, msg, e.fields)
}

// Info logs at info level
func (e *LogEntry) Info(args ...interface{}) {
	msg := fmt.Sprint(args...)
	e.logger.log(INFO, msg, e.fields)
}

// Infof logs a formatted message at info level
func (e *LogEntry) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	e.logger.log(INFO, msg, e.fields)
}

// Warn logs at warning level
func (e *LogEntry) Warn(args ...interface{}) {
	msg := fmt.Sprint(args...)
	e.logger.log(WARN, msg, e.fields)
}

// Warnf logs a formatted message at warning level
func (e *LogEntry) Warnf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	e.logger.log(WARN, msg, e.fields)
}

// Error logs at error level
func (e *LogEntry) Error(args ...interface{}) {
	msg := fmt.Sprint(args...)
	e.logger.log(ERROR, msg, e.fields)
}

// Errorf logs a formatted message at error level
func (e *LogEntry) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	e.logger.log(ERROR, msg, e.fields)
}

// Fatal logs at fatal level and exits
func (e *LogEntry) Fatal(args ...interface{}) {
	msg := fmt.Sprint(args...)
	e.logger.log(FATAL, msg, e.fields)
	os.Exit(1)
}

// Fatalf logs a formatted message at fatal level and exits
func (e *LogEntry) Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	e.logger.log(FATAL, msg, e.fields)
	os.Exit(1)
}

// Global convenience functions

// WithField creates a logger entry with a single field
func WithField(key string, value interface{}) *LogEntry {
	return GetLogger().WithField(key, value)
}

// WithFields creates a logger entry with multiple fields
func WithFields(fields map[string]interface{}) *LogEntry {
	return GetLogger().WithFields(fields)
}

// WithError creates a logger entry with an error field
func WithError(err error) *LogEntry {
	return GetLogger().WithError(err)
}

// Debug logs a message at debug level
func Debug(args ...interface{}) {
	GetLogger().log(DEBUG, fmt.Sprint(args...), nil)
}

// Debugf logs a formatted message at debug level
func Debugf(format string, args ...interface{}) {
	GetLogger().log(DEBUG, fmt.Sprintf(format, args...), nil)
}

// Info logs a message at info level
func Info(args ...interface{}) {
	GetLogger().log(INFO, fmt.Sprint(args...), nil)
}

// Infof logs a formatted message at info level
func Infof(format string, args ...interface{}) {
	GetLogger().log(INFO, fmt.Sprintf(format, args...), nil)
}

// Warn logs a message at warning level
func Warn(args ...interface{}) {
	GetLogger().log(WARN, fmt.Sprint(args...), nil)
}

// Warnf logs a formatted message at warning level
func Warnf(format string, args ...interface{}) {
	GetLogger().log(WARN, fmt.Sprintf(format, args...), nil)
}

// Error logs a message at error level
func Error(args ...interface{}) {
	GetLogger().log(ERROR, fmt.Sprint(args...), nil)
}

// Errorf logs a formatted message at error level
func Errorf(format string, args ...interface{}) {
	GetLogger().log(ERROR, fmt.Sprintf(format, args...), nil)
}

// Fatal logs a message at fatal level and exits
func Fatal(args ...interface{}) {
	GetLogger().log(FATAL, fmt.Sprint(args...), nil)
	os.Exit(1)
}

// Fatalf logs a formatted message at fatal level and exits
func Fatalf(format string, args ...interface{}) {
	GetLogger().log(FATAL, fmt.Sprintf(format, args...), nil)
	os.Exit(1)
}

// Close closes the logger and releases resources
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fileWriter != nil {
		return l.fileWriter.Close()
	}
	return nil
}

// Info logs a message at info level
func (l *Logger) Info(args ...interface{}) {
	l.log(INFO, fmt.Sprint(args...), nil)
}

// Infof logs a formatted message at info level
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...), nil)
}

// Warn logs a message at warning level
func (l *Logger) Warn(args ...interface{}) {
	l.log(WARN, fmt.Sprint(args...), nil)
}

// Warnf logs a formatted message at warning level
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...), nil)
}

// Error logs a message at error level
func (l *Logger) Error(args ...interface{}) {
	l.log(ERROR, fmt.Sprint(args...), nil)
}

// Errorf logs a formatted message at error level
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...), nil)
}

// Debug logs a message at debug level
func (l *Logger) Debug(args ...interface{}) {
	l.log(DEBUG, fmt.Sprint(args...), nil)
}

// Debugf logs a formatted message at debug level
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...), nil)
}
