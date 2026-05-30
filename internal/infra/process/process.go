// Package process provides process management for llama.cpp instances.
// It handles starting, stopping, and monitoring external llama.cpp server processes.
package process

import (
	"bufio"
	"context"
	"fmt"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
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

	// Start options
	SkipLDLibraryPath bool     // Skip LD_LIBRARY_PATH setup (used by conda backends)
	EnvVars           []string // Additional environment variables (e.g., "KEY=VALUE")

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
	cmdDone     chan struct{}
	exitCode    int
}

// Handler is a callback function for process output
type Handler func(line string)

// NewProcess creates a new process wrapper
func NewProcess(id, name, cmd, binPath string, skipLDLibraryPath bool, envVars []string) *Process {
	ctx, cancel := context.WithCancel(context.Background())

	return &Process{
		ID:                id,
		Name:              name,
		Cmd:               cmd,
		BinPath:           binPath,
		SkipLDLibraryPath: skipLDLibraryPath,
		EnvVars:           envVars,
		ctx:               ctx,
		cancel:            cancel,
		outputChan:        make(chan string, 100),
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

	// Set platform-specific process attributes (Pdeathsig on Unix)
	setSysProcAttr(p.cmd)

	// Graceful shutdown: send SIGTERM (Unix) or Kill (Windows) on context cancel,
	// then SIGKILL after 10s if process hasn't exited
	setCmdCancel(p.cmd)
	p.cmd.WaitDelay = 10 * time.Second

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
	p.exitCode = -1

	// Start output processor and readers BEFORE cmd.Wait(),
	// per Go docs: "It is incorrect to call Wait before all reads from the pipe have completed."
	p.wg.Add(1)
	go p.processOutput()
	p.wg.Add(2)
	go p.readOutput(p.stdoutPipe, "stdout")
	go p.readOutput(p.stderrPipe, "stderr")

	// Background goroutine: wait for process exit and capture exit code
	p.cmdDone = make(chan struct{})
	go func() {
		defer close(p.cmdDone)
		if err := p.cmd.Wait(); err != nil {
			if status, ok := err.(*exec.ExitError); ok {
				if ws, ok := status.Sys().(syscall.WaitStatus); ok {
					p.mu.Lock()
					p.exitCode = ws.ExitStatus()
					p.mu.Unlock()
				}
			}
		} else {
			p.mu.Lock()
			p.exitCode = 0
			p.mu.Unlock()
		}
		p.mu.Lock()
		p.Running = false
		p.mu.Unlock()
	}()

	// Check if the process survived the first 500ms
	p.mu.Unlock()
	time.Sleep(500 * time.Millisecond)
	p.mu.Lock()

	select {
	case <-p.cmdDone:
		p.Running = false
		return fmt.Errorf("process exited immediately (PID: %d)", p.PID)
	default:
	}

	return nil
}

// setupEnvironment configures the process environment.
// Note: The env var merge logic here mirrors backend.buildEnvWithVars() in
// internal/service/model/backend/backend.go. If modifying this logic, keep
// both implementations in sync.
func (p *Process) setupEnvironment(cmd *exec.Cmd, binPath string) error {
	// Get current environment
	env := os.Environ()

	// Apply custom environment variables from config
	for _, ev := range p.EnvVars {
		if idx := strings.Index(ev, "="); idx > 0 {
			key := ev[:idx]
			// Replace existing variable with the same name
			prefix := key + "="
			found := false
			for i, e := range env {
				if strings.HasPrefix(e, prefix) {
					env[i] = ev
					found = true
					break
				}
			}
			if !found {
				env = append(env, ev)
			}
		}
	}

	// Conda backends (vLLM/vLLM-omni) manage their own environment via conda run; skip LD_LIBRARY_PATH
	if p.SkipLDLibraryPath {
		cmd.Env = env
		return nil
	}

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
		// Decrement reader count and close channel if we're the last reader.
		// Note: do NOT close the pipe here; cmd.Wait() manages pipe lifecycle.
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

	if len(line) > 0 && line[0] != '[' {
		logger.Debug(fmt.Sprintf("[%s] %s", p.Name, line))
	}
}

// Send sends input to the process stdin
func (p *Process) Send(input string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.Running || p.stdinPipe == nil {
		return fmt.Errorf("process not running")
	}

	_, err := p.stdinPipe.Write([]byte(input))
	return err
}

// Stop stops the process gracefully.
// Context cancel triggers cmd.Cancel (SIGTERM), then cmd.WaitDelay (10s) before SIGKILL.
func (p *Process) Stop() error {
	p.mu.Lock()

	if !p.Running {
		p.mu.Unlock()
		return nil
	}

	p.Running = false
	cmdDone := p.cmdDone
	p.mu.Unlock()

	// Cancel context → triggers cmd.Cancel (SIGTERM) → WaitDelay → SIGKILL
	p.cancel()

	// Close stdin to unblock any pending reads
	p.mu.Lock()
	if p.stdinPipe != nil {
		utils.CloseQuietly(p.stdinPipe)
		p.stdinPipe = nil
	}
	p.mu.Unlock()

	// Wait for cmd.Wait() goroutine to finish (process fully exited)
	if cmdDone != nil {
		select {
		case <-cmdDone:
		case <-time.After(12 * time.Second):
			// Safety timeout beyond WaitDelay (10s)
		}
	}

	// Wait for output readers to finish
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
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

	// Check if cmd.Wait() goroutine has returned
	if p.cmdDone != nil {
		select {
		case <-p.cmdDone:
			p.Running = false
			return false
		default:
		}
	}

	return true
}

// GetPID returns the process PID
func (p *Process) GetPID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.PID
}

// GetExitCode returns the process exit code if it has exited
func (p *Process) GetExitCode() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil {
		return 0, fmt.Errorf("process not started")
	}

	// Check if process is still running
	if p.Running {
		return 0, fmt.Errorf("process still running")
	}

	// Wait for cmd.Wait() goroutine to capture exit code
	if p.cmdDone != nil {
		select {
		case <-p.cmdDone:
		default:
			return 0, fmt.Errorf("process still exiting")
		}
	}

	return p.exitCode, nil
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
