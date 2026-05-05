// Package utils provides utility functions for common operations.
// This package avoids circular dependencies by not importing other internal packages.
package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

// closeQuietly closes an io.Closer and ignores the error.
// This is intended for use in defer statements where the error is not critical.
//
// Example:
//
//	file, err := os.Open("file.txt")
//	if err != nil {
//	    return err
//	}
//	defer utils.CloseQuietly(file)
func closeQuietly(c io.Closer) {
	_ = c.Close()
}

// removeQuietly removes a file and ignores errors if file doesn't exist.
// Other errors are printed to stderr for debugging.
//
// Example:
//
//	utils.RemoveQuietly("/tmp/temp-file.txt")
func removeQuietly(path string) {
	if err := os.Remove(path); err != nil {
		// Don't print if file doesn't exist - that's expected for cleanup
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "[WARN] 删除文件失败 %s: %v\n", path, err)
		}
	}
}

// CloseQuietly closes an io.Closer and ignores the error.
// This is the exported version for use in other packages.
func CloseQuietly(c io.Closer) {
	closeQuietly(c)
}

// RemoveQuietly removes a file and logs a warning if removal fails.
// This is the exported version for use in other packages.
func RemoveQuietly(path string) {
	removeQuietly(path)
}

// WriteQuietly writes data and logs a warning if it fails.
// This is useful for HTTP responses where write failures are not critical.
func WriteQuietly(writer io.Writer, data []byte) {
	if _, err := writer.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] 写入失败: %v\n", err)
	}
}

// KillQuietly kills a process and logs a warning if it fails.
// If the process has already exited, no warning is printed.
func KillQuietly(process *os.Process) {
	if process == nil {
		return
	}
	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		fmt.Fprintf(os.Stderr, "[WARN] 终止进程失败 %d: %v\n", process.Pid, err)
	}
}

// SignalQuietly sends a signal to a process and logs a warning if it fails.
// If the process has already exited, no warning is printed.
func SignalQuietly(process *os.Process, sig syscall.Signal) {
	if process == nil {
		return
	}
	if err := process.Signal(sig); err != nil && !errors.Is(err, os.ErrProcessDone) {
		fmt.Fprintf(os.Stderr, "[WARN] 发送信号失败 %d: %v\n", process.Pid, err)
	}
}

// SetReadDeadlineQuietly sets a read deadline and ignores errors.
// This is useful for network operations where deadline failures are acceptable.
func SetReadDeadlineQuietly(conn interface{ SetReadDeadline(time.Time) error }, timeout time.Duration) {
	if timeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
	}
}

// SetWriteDeadlineQuietly sets a write deadline and ignores errors.
// This is useful for network operations where deadline failures are acceptable.
func SetWriteDeadlineQuietly(conn interface{ SetWriteDeadline(time.Time) error }, timeout time.Duration) {
	if timeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	}
}

// UnmarshalQuietly unmarshals JSON data and logs a warning if it fails.
// Returns true if unmarshaling succeeded, false otherwise.
// This is useful for optional metadata fields where parsing failures are acceptable.
func UnmarshalQuietly(data []byte, v interface{}, fieldName string) bool {
	if err := json.Unmarshal(data, v); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] 解析%s失败: %v\n", fieldName, err)
		return false
	}
	return true
}

// WriteMessageQuietly writes a WebSocket message and ignores errors.
// This is useful for WebSocket close messages where failures are acceptable.
func WriteMessageQuietly(conn interface{ WriteMessage(int, []byte) error }, messageType int, data []byte) {
	if err := conn.WriteMessage(messageType, data); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] 写入WebSocket消息失败: %v\n", err)
	}
}


