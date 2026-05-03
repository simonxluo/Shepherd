// Package process provides process management for llama.cpp instances.
// It handles starting, stopping, and monitoring external llama.cpp server processes.
package process

import (
	"bufio"
	"context"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Process represents a running llama.cpp server process
type Process struct {
	ID      string
	Name    string
	Cmd     string
	BinPath string

	// Runtime state
	PID     int
	Running bool
	CtxSize int
	Port    int

	// Internal fields
	cmd        *exec.Cmd
	stdoutPipe io.ReadCloser
	stderrPipe io.ReadCloser
	stdinPipe  io.WriteCloser
	outputChan chan string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	// Logging
	outputHandler func(string)

	mu          sync.Mutex
	readersDone int32 // counter for finished output readers (0, 1, or 2)
}

// Handler is a callback function for process output
type Handler func(line string)

// NewProcess creates a new process wrapper
func NewProcess(id, name, cmd, binPath string) *Process {
	ctx, cancel := context.WithCancel(context.Background())

	return &Process{
		ID:         id,
		Name:       name,
		Cmd:        cmd,
		BinPath:    binPath,
		ctx:        ctx,
		cancel:     cancel,
		outputChan: make(chan string, 100),
	}
}

// SetOutputHandler sets the callback for process output
func (p *Process) SetOutputHandler(handler Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.outputHandler = handler
}

// SetCtxSize sets the context size (set after successful model loading)
func (p *Process) SetCtxSize(ctxSize int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.CtxSize = ctxSize
}

// GetCtxSize returns the context size
func (p *Process) GetCtxSize() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.CtxSize
}

// GetPort returns the port number
func (p *Process) GetPort() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Port
}

// Start starts the process
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Running {
		return fmt.Errorf("process already running")
	}

	// Reset state for fresh start
	atomic.StoreInt32(&p.readersDone, 0)
	p.outputChan = make(chan string, 100)

	// Parse command line arguments
	args, err := splitCommandLineArgs(p.Cmd)
	if err != nil {
		return fmt.Errorf("failed to parse command: %w", err)
	}

	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}

	// Create command
	binPath := args[0]
	cmdArgs := args[1:]
	p.cmd = exec.CommandContext(p.ctx, binPath, cmdArgs...)

	// Setup environment
	if err := p.setupEnvironment(p.cmd, binPath); err != nil {
		return fmt.Errorf("failed to setup environment: %w", err)
	}

	// Setup pipes
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	p.stdoutPipe = stdout

	stderr, err := p.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	p.stderrPipe = stderr

	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	p.stdinPipe = stdin

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	p.PID = p.cmd.Process.Pid
	p.Running = true

	// 验证进程是否真正运行（防止立即崩溃）
	// 延迟 500ms 让进程初始化
	// 注意：必须在释放锁后进行睡眠和检查，避免死锁
	// 因为 IsRunning() 也会获取 p.mu 锁
	p.mu.Unlock() // 临时释放锁，允许睡眠期间其他 goroutine 访问
	time.Sleep(500 * time.Millisecond)

	// 使用 Signal(0) 检查进程是否仍在运行
	// 重新获取锁进行状态检查
	p.mu.Lock()
	isRunning := p.checkRunningUnsafe() // 使用不获取锁的内部方法
	p.mu.Unlock()

	if !isRunning {
		// 进程已退出，清理状态
		p.mu.Lock()
		p.Running = false
		p.mu.Unlock()
		return fmt.Errorf("process exited immediately (PID: %d)", p.PID)
	}

	p.mu.Lock() // 重新获取锁，保持函数语义一致

	// Start output processor first (it will drain any remaining lines)
	p.wg.Add(1)
	go p.processOutput()

	// Start output readers (they close the channel when done)
	p.wg.Add(2)
	go p.readOutput(p.stdoutPipe, "stdout")
	go p.readOutput(p.stderrPipe, "stderr")

	return nil
}

// setupEnvironment configures the process environment
func (p *Process) setupEnvironment(cmd *exec.Cmd, binPath string) error {
	// Get current environment
	env := os.Environ()

	// Add bin directory to library path on Unix-like systems
	if strings.HasPrefix(binPath, "/") {
		binDir := filepath.Dir(binPath)

		// Find LD_LIBRARY_PATH and add our bin directory
		found := false
		for i, e := range env {
			if strings.HasPrefix(e, "LD_LIBRARY_PATH=") {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					env[i] = "LD_LIBRARY_PATH=" + binDir + ":" + parts[1]
				}
				found = true
				break
			}
		}

		if !found {
			env = append(env, "LD_LIBRARY_PATH="+binDir)
		}
	}

	cmd.Env = env
	return nil
}

// readOutput reads from a pipe and sends lines to the output channel
// Each readOutput goroutine is responsible for closing its pipe.
// The last readOutput to finish closes the output channel.
func (p *Process) readOutput(pipe io.ReadCloser, name string) {
	defer p.wg.Done()
	defer func() {
		utils.CloseQuietly(pipe)
		// Decrement reader count and close channel if we're the last reader
		if atomic.AddInt32(&p.readersDone, 1) == 2 {
			close(p.outputChan)
		}
	}()

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case p.outputChan <- line:
		case <-p.ctx.Done():
			return
		}
	}
}

// processOutput processes output lines from the channel
func (p *Process) processOutput() {
	defer p.wg.Done()

	for {
		select {
		case line, ok := <-p.outputChan:
			if !ok {
				return
			}
			p.handleOutputLine(line)
		case <-p.ctx.Done():
			// Drain remaining lines before exiting
			for line := range p.outputChan {
				p.handleOutputLine(line)
			}
			return
		}
	}
}

// handleOutputLine processes a single output line
func (p *Process) handleOutputLine(line string) {
	p.mu.Lock()
	handler := p.outputHandler
	p.mu.Unlock()

	// Call handler if set
	if handler != nil {
		handler(line)
	}

	// Filter out noisy logs (same as Java version)
	if strings.Contains(line, "update_slots") || strings.Contains(line, "log_server_r") {
		return
	}

	// 发送到日志系统
	// 使用 outputHandler 进行日志转发，由外部决定如何处理日志
	// 默认情况下，非日志行会打印到控制台
	if len(line) > 0 && line[0] != '[' {
		// 非日志行（可能是服务器输出）
		fmt.Printf("[%s] %s\n", p.Name, line)
	}

	// 日志行（以 [ 开头）已通过 outputHandler 处理，这里不再重复打印
}

// Stop stops the process
func (p *Process) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Running {
		return nil
	}

	p.Running = false

	// Send cancel signal
	p.cancel()

	// Close stdin
	if p.stdinPipe != nil {
		utils.CloseQuietly(p.stdinPipe)
		p.stdinPipe = nil
	}

	// Try graceful shutdown first
	if p.cmd != nil && p.cmd.Process != nil {
		// Send SIGTERM
		utils.SignalQuietly(p.cmd.Process, syscall.SIGTERM)

		// Wait for up to 5 seconds
		done := make(chan error, 1)
		go func() {
			_, err := p.cmd.Process.Wait()
			done <- err
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(5 * time.Second):
			// Timeout, force kill
			utils.KillQuietly(p.cmd.Process)
			<-done // Wait for the goroutine to finish
		}
	}

	// Wait for output readers to finish
	// Note: Don't wait forever in case something is stuck
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		// Give up waiting
	}

	return nil
}

// IsRunning returns whether the process is currently running
func (p *Process) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.checkRunningUnsafe()
}

// checkRunningUnsafe checks if process is running without acquiring lock
// IMPORTANT: Caller must hold p.mu lock
func (p *Process) checkRunningUnsafe() bool {
	if !p.Running || p.cmd == nil || p.cmd.Process == nil {
		return false
	}

	// Check if process is still alive
	if err := p.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		p.Running = false
		return false
	}

	return true
}

// GetPID returns the process PID
func (p *Process) GetPID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.PID
}

// splitCommandLineArgs splits a command line string into arguments
// Handles quoted strings and escape sequences (ported from Java)
func splitCommandLineArgs(commandLine string) ([]string, error) {
	if commandLine == "" {
		return []string{}, nil
	}

	s := strings.TrimSpace(commandLine)
	if s == "" {
		return []string{}, nil
	}

	var out []string
	var cur strings.Builder

	allowSingle := !isWindows()
	inSingle := false
	inDouble := false

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		c := runes[i]

		// Handle escape sequences
		if inDouble && c == '\\' {
			if i+1 < len(runes) {
				n := runes[i+1]
				if n == '"' {
					cur.WriteRune(n)
					i++
					continue
				}
			}
			cur.WriteRune(c)
			continue
		}

		if allowSingle && inSingle && c == '\\' {
			if i+1 < len(runes) {
				n := runes[i+1]
				if n == '\'' {
					cur.WriteRune(n)
					i++
					continue
				}
			}
			cur.WriteRune(c)
			continue
		}

		// Handle quotes
		if c == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if allowSingle && c == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}

		// Handle whitespace (argument separator)
		if !inSingle && !inDouble && isSpace(c) {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}

		cur.WriteRune(c)
	}

	// Add last argument
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}

	return out, nil
}

// isSpace returns true if the rune is a whitespace character
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// isWindows returns true if running on Windows
func isWindows() bool {
	return strings.HasPrefix(strings.ToLower(os.Getenv("OS")), "windows")
}
