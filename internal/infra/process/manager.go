package process

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
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
func (m *Manager) Start(modelID, name, cmd, binPath string) (*Process, error) {
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
	process := NewProcess(modelID, name, cmd, binPath)

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

// GetRunningCount returns the number of running processes
func (m *Manager) GetRunningCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.processes)
}

// GetLoadingCount returns the number of loading processes
func (m *Manager) GetLoadingCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.loading)
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

// quoteAndJoin joins arguments into a command string with proper quoting
func quoteAndJoin(args []string) string {
	var result string
	for i, arg := range args {
		if i > 0 {
			result += " "
		}

		// Quote arguments that contain spaces or special characters
		if needsQuoting(arg) {
			result += `"` + escapeQuotes(arg) + `"`
		} else {
			result += arg
		}
	}
	return result
}

// needsQuoting returns true if an argument needs to be quoted
func needsQuoting(arg string) bool {
	for _, c := range arg {
		if c == ' ' || c == '\t' || c == '"' || c == '\'' || c == '\\' {
			return true
		}
	}
	return false
}

// escapeQuotes escapes quotes in a string
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

// LoadRequest contains parameters for building a llama-server command
// This is a local definition to avoid import cycle; the canonical version is in internal/model
type LoadRequest struct {
	ModelPath        string
	Port             int
	CtxSize          int
	BatchSize        int
	Threads          int
	GPULayers        int
	Temperature      float64
	TopP             float64
	TopK             int
	RepeatPenalty    float64
	Seed             int
	NPredict         int
	Devices          []string // GPU device selection (e.g., ["cuda:0", "cuda:1"])
	MainGPU          int      // Main GPU index for single-GPU mode
	CustomCmd        string   // Custom command string appended verbatim
	ExtraParams      string   // Extra parameters appended verbatim
	MmprojPath       string   // Path to mmproj.gguf for vision models
	EnableVision     bool     // Enable vision/multimodal capabilities
	FlashAttention   bool     // Enable Flash Attention (-fa)
	NoMmap           bool     // Disable memory mapping (--no-mmap)
	LockMemory       bool     // Lock model in RAM (--mlock)
	NoWebUI          bool     // Disable web UI (--no-webui)
	EnableMetrics    bool     // Enable /metrics endpoint (--metrics)
	SlotSavePath     string   // Slot cache directory (--slot-save-path)
	CacheRAM         int      // RAM cache limit in MB (--cache-ram, -1 = unlimited)
	ChatTemplateFile string   // Custom chat template file (--chat-template-file)
	Timeout          int      // Read/write timeout in seconds (--timeout)
	Alias            string   // Model alias for REST API (--alias)
	UBatchSize       int      // Micro-batch size (--ubatch-size)
	ParallelSlots    int      // Number of parallel slots (--parallel)
	KVCacheTypeK     string   // KV cache type for K (--kv-cache-type-k)
	KVCacheTypeV     string   // KV cache type for V (--kv-cache-type-v)
	KVCacheUnified   bool     // Use unified KV cache (--kv-unified)
	KVCacheSize      int      // KV cache size (--kv-cache-size)

	// Additional sampling parameters
	LogitsAll        bool    // --logits-all (input vector mode)
	Reranking        bool    // --reranking (reranking mode)
	MinP             float64 // --min-p (Min-P sampling)
	PresencePenalty  float64 // --presence-penalty
	FrequencyPenalty float64 // --frequency-penalty

	// Template and processing
	DirectIo     string // --dio (direct I/O mode)
	DisableJinja bool   // --no-jinja (disable Jinja template)
	ChatTemplate string // --chat-template (built-in chat template)
	ContextShift bool   // --context-shift (enable context shift)

	// Thread configuration
	ThreadsBatch int // --threads-batch (batch processing threads)

	// Extended sampling parameters
	RepeatLastN int     // --repeat-last-n
	TypicalP    float64 // --typical-p
	IgnoreEOS   bool    // --ignore-eos

	// Multi-GPU configuration
	SplitMode   string // --split-mode (none, layer, row)
	TensorSplit string // --tensor-split (comma-separated values)

	// Server optimization
	ContBatching bool // --cont-batching
	CachePrompt  bool // --cache-prompt

	// Structured generation
	Grammar     string // --grammar
	GrammarFile string // --grammar-file

	// LoRA adapter support
	Lora       string // --lora
	LoraScaled string // --lora-scaled

	// Chat template kwargs
	ChatTemplateKwargs string // --chat-template-kwargs

	// RoPE scaling (for extended context)
	RopeScaling   string  // --rope-scaling
	RopeScale     float64 // --rope-scale
	RopeFreqBase  float64 // --rope-freq-base
	RopeFreqScale float64 // --rope-freq-scale
}

// BuildCommandFromRequest builds the llama-server command line from a LoadRequest struct
// This is the new, comprehensive command builder that supports all llama.cpp flags
func BuildCommandFromRequest(req *LoadRequest, binPath string) (string, error) {
	// Validate required fields
	if req == nil {
		return "", fmt.Errorf("request cannot be nil")
	}
	if binPath == "" {
		return "", fmt.Errorf("binary path cannot be empty")
	}
	if req.ModelPath == "" {
		return "", fmt.Errorf("model path cannot be empty")
	}
	if req.Port <= 0 {
		return "", fmt.Errorf("port must be positive")
	}

	// Find the llama-server executable using unified utility function
	serverBin := utils.FindLlamacppBinary(binPath, "server")
	if serverBin == "" {
		return "", fmt.Errorf("llama-server not found in path: %s", binPath)
	}

	// Build command arguments
	args := []string{
		serverBin,
		"-m", req.ModelPath,
		"--port", strconv.Itoa(req.Port),
		"--host", "0.0.0.0",
	}

	// Context and batch size
	if req.CtxSize > 0 {
		args = append(args, "-c", strconv.Itoa(req.CtxSize))
	}
	if req.BatchSize > 0 {
		args = append(args, "-b", strconv.Itoa(req.BatchSize))
	}
	if req.Threads > 0 {
		args = append(args, "-t", strconv.Itoa(req.Threads))
	}
	if req.ThreadsBatch > 0 {
		args = append(args, "-tb", strconv.Itoa(req.ThreadsBatch))
	}

	// GPU configuration
	if req.GPULayers > 0 {
		args = append(args, "-ngl", strconv.Itoa(req.GPULayers))
	}

	// GPU device selection
	// Single GPU: -sm none -dev cuda:0 -mg 0
	// Multi-GPU: -dev cuda:0,cuda:1
	if len(req.Devices) > 0 {
		// Use explicit split mode if specified
		if req.SplitMode != "" {
			args = append(args, "-sm", req.SplitMode)
		} else if len(req.Devices) == 1 {
			// Single GPU mode: disable split mode by default
			args = append(args, "-sm", "none")
		}
		args = append(args, "-dev", strings.Join(req.Devices, ","))
		if len(req.Devices) == 1 {
			args = append(args, "-mg", strconv.Itoa(req.MainGPU))
		}
	}
	// Tensor split for multi-GPU
	if req.TensorSplit != "" {
		args = append(args, "-ts", req.TensorSplit)
	}

	// Vision/Multimodal support
	if req.MmprojPath != "" {
		args = append(args, "--mmproj", req.MmprojPath)
	}

	// Performance feature flags
	// Flash Attention requires a value: on, off, or auto
	if req.FlashAttention {
		args = append(args, "-fa", "on")
	}
	if req.NoMmap {
		args = append(args, "--no-mmap")
	}
	if req.LockMemory {
		args = append(args, "--mlock")
	}

	// Server feature flags
	if req.NoWebUI {
		args = append(args, "--no-webui")
	}
	if req.EnableMetrics {
		args = append(args, "--metrics")
	}
	if req.SlotSavePath != "" {
		args = append(args, "--slot-save-path", req.SlotSavePath)
	}
	if req.CacheRAM != 0 {
		args = append(args, "--cache-ram", strconv.Itoa(req.CacheRAM))
	}

	// Chat template
	if req.ChatTemplateFile != "" {
		args = append(args, "--chat-template-file", req.ChatTemplateFile)
	}

	// Batch processing
	if req.UBatchSize > 0 {
		args = append(args, "--ubatch-size", strconv.Itoa(req.UBatchSize))
	}
	if req.ParallelSlots > 0 {
		args = append(args, "--parallel", strconv.Itoa(req.ParallelSlots))
	}

	// KV cache configuration
	if req.KVCacheTypeK != "" {
		args = append(args, "-ctk", req.KVCacheTypeK)
	}
	if req.KVCacheTypeV != "" {
		args = append(args, "-ctv", req.KVCacheTypeV)
	}
	if req.KVCacheUnified {
		args = append(args, "-kvu")
	}
	if req.KVCacheSize > 0 {
		args = append(args, "--cache-ram", strconv.Itoa(req.KVCacheSize))
	}

	// Runtime configuration
	if req.Timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(req.Timeout))
	}
	if req.Alias != "" {
		args = append(args, "--alias", req.Alias)
	}

	// Sampling parameters (for server defaults)
	if req.Temperature != 0 {
		args = append(args, "--temp", fmt.Sprintf("%.2f", req.Temperature))
	}
	if req.TopP != 0 {
		args = append(args, "--top-p", fmt.Sprintf("%.2f", req.TopP))
	}
	if req.TopK > 0 {
		args = append(args, "--top-k", strconv.Itoa(req.TopK))
	}
	if req.RepeatPenalty != 0 {
		args = append(args, "--repeat-penalty", fmt.Sprintf("%.2f", req.RepeatPenalty))
	}
	if req.Seed != 0 {
		args = append(args, "--seed", strconv.Itoa(req.Seed))
	}
	if req.NPredict > 0 {
		args = append(args, "-n", strconv.Itoa(req.NPredict))
	}

	// Additional sampling parameters
	// Note: --logits-all is not supported by llama-server, only by llama-cli
	// if req.LogitsAll {
	// 	args = append(args, "--logits-all")
	// }
	if req.Reranking {
		args = append(args, "--reranking")
	}
	if req.MinP > 0 {
		args = append(args, "--min-p", fmt.Sprintf("%.2f", req.MinP))
	}
	if req.PresencePenalty != 0 {
		args = append(args, "--presence-penalty", fmt.Sprintf("%.2f", req.PresencePenalty))
	}
	if req.FrequencyPenalty != 0 {
		args = append(args, "--frequency-penalty", fmt.Sprintf("%.2f", req.FrequencyPenalty))
	}

	// Template and processing
	// DirectIo is a boolean flag (--dio)
	// NOTE: --dio requires specific filesystem support and may not work in all environments
	// Temporarily disabled for testing - uncomment if your environment supports DirectIO
	// if req.DirectIo != "" {
	// 	args = append(args, "--dio")
	// }
	if req.DisableJinja {
		args = append(args, "--no-jinja")
	}
	if req.ChatTemplate != "" {
		args = append(args, "--chat-template", req.ChatTemplate)
	}
	if req.ContextShift {
		args = append(args, "--context-shift")
	}

	// Extended sampling parameters
	if req.RepeatLastN != 0 {
		args = append(args, "--repeat-last-n", strconv.Itoa(req.RepeatLastN))
	}
	if req.TypicalP > 0 {
		args = append(args, "--typical-p", fmt.Sprintf("%.2f", req.TypicalP))
	}
	if req.IgnoreEOS {
		args = append(args, "--ignore-eos")
	}

	// Server optimization
	if req.ContBatching {
		args = append(args, "--cont-batching")
	} else if !req.ContBatching && req.ExtraParams == "" {
		// Only add --no-cont-batching if explicitly disabled and no extra params
		args = append(args, "--no-cont-batching")
	}
	if req.CachePrompt {
		args = append(args, "--cache-prompt")
	}

	// Structured generation
	if req.Grammar != "" {
		args = append(args, "--grammar", req.Grammar)
	}
	if req.GrammarFile != "" {
		args = append(args, "--grammar-file", req.GrammarFile)
	}

	// LoRA adapter support
	if req.Lora != "" {
		args = append(args, "--lora", req.Lora)
	}
	if req.LoraScaled != "" {
		args = append(args, "--lora-scaled", req.LoraScaled)
	}

	// Chat template kwargs
	if req.ChatTemplateKwargs != "" {
		args = append(args, "--chat-template-kwargs", req.ChatTemplateKwargs)
	}

	// RoPE scaling for extended context
	if req.RopeScaling != "" {
		args = append(args, "--rope-scaling", req.RopeScaling)
	}
	if req.RopeScale > 0 {
		args = append(args, "--rope-scale", fmt.Sprintf("%.2f", req.RopeScale))
	}
	if req.RopeFreqBase > 0 {
		args = append(args, "--rope-freq-base", fmt.Sprintf("%.2f", req.RopeFreqBase))
	}
	if req.RopeFreqScale > 0 {
		args = append(args, "--rope-freq-scale", fmt.Sprintf("%.2f", req.RopeFreqScale))
	}

	// Build the base command string
	cmd := quoteAndJoin(args)

	// Append custom command if provided
	if req.CustomCmd != "" {
		cmd += " " + strings.TrimSpace(req.CustomCmd)
	}

	// Append extra params if provided
	if req.ExtraParams != "" {
		cmd += " " + strings.TrimSpace(req.ExtraParams)
	}

	return cmd, nil
}
