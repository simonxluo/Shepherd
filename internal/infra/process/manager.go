package process

import (
	"fmt"
	"strings"
	"sync"
)

// Manager manages multiple llama.cpp processes
type Manager struct {
	processes map[string]*Process
	loading   map[string]*Process
	mu        sync.RWMutex
}

// NewManager creates a new process manager
func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*Process),
		loading:   make(map[string]*Process),
	}
}

// Start starts a new llama.cpp process for a model
func (m *Manager) Start(modelID, name, cmd, binPath string, skipLDLibraryPath bool, envVars []string) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already loaded
	if _, exists := m.processes[modelID]; exists {
		return nil, fmt.Errorf("model %s already loaded", modelID)
	}

	// Check if currently loading
	if _, exists := m.loading[modelID]; exists {
		return nil, fmt.Errorf("model %s is currently loading", modelID)
	}

	// Create process
	process := NewProcess(modelID, name, cmd, binPath, skipLDLibraryPath, envVars)

	// Add to loading map
	m.loading[modelID] = process

	// Start the process (outside the lock)
	m.mu.Unlock()
	err := process.Start()
	m.mu.Lock()

	if err != nil {
		// Remove from loading map on error
		delete(m.loading, modelID)
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Move from loading to loaded
	delete(m.loading, modelID)
	m.processes[modelID] = process

	return process, nil
}

// Stop stops a running process
func (m *Manager) Stop(modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check loaded processes
	process, exists := m.processes[modelID]
	if !exists {
		// Also check loading processes
		if process, exists = m.loading[modelID]; exists {
			delete(m.loading, modelID)
			return process.Stop()
		}
		return fmt.Errorf("model %s not found", modelID)
	}

	// Stop the process
	if err := process.Stop(); err != nil {
		return err
	}

	// Remove from map
	delete(m.processes, modelID)

	return nil
}

// Get returns a process by model ID
func (m *Manager) Get(modelID string) (*Process, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check loaded first
	if process, exists := m.processes[modelID]; exists {
		return process, true
	}

	// Check loading
	process, exists := m.loading[modelID]
	return process, exists
}

// List returns all running processes
func (m *Manager) List() map[string]*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	result := make(map[string]*Process, len(m.processes))
	for k, v := range m.processes {
		result[k] = v
	}

	return result
}

// ListAll returns both running and loading processes
func (m *Manager) ListAll() (running, loading map[string]*Process) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	running = make(map[string]*Process, len(m.processes))
	loading = make(map[string]*Process, len(m.loading))

	for k, v := range m.processes {
		running[k] = v
	}
	for k, v := range m.loading {
		loading[k] = v
	}

	return running, loading
}

// IsRunning returns true if a model is currently running
func (m *Manager) IsRunning(modelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if process, exists := m.processes[modelID]; exists {
		return process.IsRunning()
	}
	return false
}

// StopAll stops all running processes
func (m *Manager) StopAll() []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	// Stop all loaded processes
	for modelID, process := range m.processes {
		if err := process.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop %s: %w", modelID, err))
		}
		delete(m.processes, modelID)
	}

	// Stop all loading processes
	for modelID, process := range m.loading {
		if err := process.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop loading %s: %w", modelID, err))
		}
		delete(m.loading, modelID)
	}

	return errs
}


func quoteAndJoin(args []string) string {
	var result string
	for i, arg := range args {
		if i > 0 {
			result += " "
		}

		if needsQuoting(arg) {
			result += `"` + escapeQuotes(arg) + `"`
		} else {
			result += arg
		}
	}
	return result
}

func needsQuoting(arg string) bool {
	for _, c := range arg {
		if c == ' ' || c == '\t' || c == '"' || c == '\'' || c == '\\' {
			return true
		}
	}
	return false
}

func escapeQuotes(s string) string {
	result := strings.Builder{}
	for _, c := range s {
		if c == '"' || c == '\\' {
			result.WriteRune('\\')
		}
		result.WriteRune(c)
	}
	return result.String()
}
