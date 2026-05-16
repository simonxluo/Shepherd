package backend

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

// Registry manages available backends and provides lookup functionality
type Registry struct {
	mu       sync.RWMutex
	backends map[BackendType]Backend
	configs  map[BackendType]*BackendConfig
}

// NewRegistry creates a new backend registry with default backends registered
func NewRegistry() *Registry {
	r := &Registry{
		backends: make(map[BackendType]Backend),
		configs:  make(map[BackendType]*BackendConfig),
	}

	// Register built-in backends
	r.Register(NewLlamaCppBackend())
	r.Register(NewVLLMBackend())
	r.Register(NewVLLMOmniBackend())

	return r
}

// Register adds a backend to the registry
func (r *Registry) Register(b Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends[b.Type()] = b
}

// Get returns a backend by type
func (r *Registry) Get(bt BackendType) (Backend, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.backends[bt]
	return b, ok
}

// Configure sets the configuration for a specific backend type
func (r *Registry) Configure(bt BackendType, cfg *BackendConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[bt] = cfg
}

// CapabilityHint provides capability information to influence backend selection.
// When a model has multimodal capabilities (TTS/ASR/ImageGeneration), the registry
// can route it to a backend that supports those endpoints (e.g., vLLM-Omni).
type CapabilityHint struct {
	TTS             bool
	ASR             bool
	ImageGeneration bool
}

// NeedsMultimodal returns true if the hint indicates multimodal capability.
func (h *CapabilityHint) NeedsMultimodal() bool {
	return h != nil && (h.TTS || h.ASR || h.ImageGeneration)
}

// Resolve determines which backend to use for a given model path.
// It uses explicit type first, then capability-aware routing for multimodal models,
// then auto-detects from model format, then defaults to llama.cpp.
func (r *Registry) Resolve(modelPath string, explicitType BackendType, hints ...*CapabilityHint) (Backend, *BackendConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If explicit type is specified and configured, use it
	if explicitType != "" && explicitType != BackendLlamaCpp {
		b, ok := r.backends[explicitType]
		if !ok {
			return nil, nil, ErrBackendNotFound(explicitType)
		}
		cfg := r.configs[explicitType]
		if cfg != nil {
			return b, cfg, nil
		}
		// Explicit type not configured — fall through to auto-detection
		// rather than returning nil config and failing at Discover.
		// This handles the case where the frontend recommends vllm_omni
		// for TTS models but the user hasn't configured it.
	}

	// Capability-aware routing: multimodal models prefer vLLM-Omni
	var hint *CapabilityHint
	if len(hints) > 0 {
		hint = hints[0]
	}
	if hint.NeedsMultimodal() {
		if b, ok := r.backends[BackendVLLMOmni]; ok && b.SupportsModel(modelPath) {
			if cfg := r.configs[BackendVLLMOmni]; cfg != nil {
				return b, cfg, nil
			}
		}
	}

	// Auto-detect from model file extension using deterministic ordering
	// vLLM takes priority over vLLM-omni for non-multimodal safetensors models
	autoDetectOrder := []BackendType{BackendVLLM, BackendVLLMOmni}
	for _, bt := range autoDetectOrder {
		b, ok := r.backends[bt]
		if !ok || !b.SupportsModel(modelPath) {
			continue
		}
		cfg := r.configs[bt]
		if cfg != nil {
			return b, cfg, nil
		}
	}

	// If model is not GGUF and no vLLM backend is configured, return an error
	// rather than silently falling through to llama.cpp (which can't serve it)
	if !IsGGUFModel(modelPath) {
		return nil, nil, fmt.Errorf("no suitable backend found for model %q (not a GGUF file, requires vLLM or similar backend but none is configured)", modelPath)
	}

	// Default to llama.cpp for GGUF models
	b, ok := r.backends[BackendLlamaCpp]
	if !ok {
		return nil, nil, ErrBackendNotFound(BackendLlamaCpp)
	}
	cfg := r.configs[BackendLlamaCpp]
	return b, cfg, nil
}

// SyncFromConfig initializes backend configurations from the app config
func (r *Registry) SyncFromConfig(cfg *config.Config) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Sync llama.cpp from legacy config
	llamacppCfg := &BackendConfig{
		Type: BackendLlamaCpp,
		Name: "LlamaCpp",
	}
	if len(cfg.Llamacpp.Paths) > 0 {
		llamacppCfg.BinPath = cfg.Llamacpp.Paths[0].Path
		llamacppCfg.BinPaths = make([]string, 0, len(cfg.Llamacpp.Paths))
		for _, p := range cfg.Llamacpp.Paths {
			llamacppCfg.BinPaths = append(llamacppCfg.BinPaths, p.Path)
		}
	}
	r.configs[BackendLlamaCpp] = llamacppCfg

	// Sync vLLM config
	if cfg.Backends.VLLM != nil && cfg.Backends.VLLM.Enabled {
		vllmCfg := &BackendConfig{
			Type:        BackendVLLM,
			Name:        "vLLM",
			CondaEnv:    cfg.Backends.VLLM.CondaEnv,
			CondaPath:   cfg.Backends.VLLM.CondaPath,
			ServeBin:    cfg.Backends.VLLM.ServeBin,
			ExtraArgs:   cfg.Backends.VLLM.ExtraArgs,
			DefaultPort: cfg.Backends.VLLM.DefaultPort,
			EnvVars:     cfg.Backends.VLLM.Env,
		}
		if len(cfg.Backends.VLLM.Paths) > 0 {
			vllmCfg.BinPath = cfg.Backends.VLLM.Paths[0].Path
			vllmCfg.BinPaths = make([]string, 0, len(cfg.Backends.VLLM.Paths))
			for _, p := range cfg.Backends.VLLM.Paths {
				vllmCfg.BinPaths = append(vllmCfg.BinPaths, p.Path)
			}
		}
		r.configs[BackendVLLM] = vllmCfg
	}

	// Sync vLLM-omni config
	if cfg.Backends.VLLMOmni != nil && cfg.Backends.VLLMOmni.Enabled {
		omniCfg := &BackendConfig{
			Type:        BackendVLLMOmni,
			Name:        "vLLM-Omni",
			CondaEnv:    cfg.Backends.VLLMOmni.CondaEnv,
			CondaPath:   cfg.Backends.VLLMOmni.CondaPath,
			ServeBin:    cfg.Backends.VLLMOmni.ServeBin,
			ExtraArgs:   cfg.Backends.VLLMOmni.ExtraArgs,
			DefaultPort: cfg.Backends.VLLMOmni.DefaultPort,
			EnvVars:     cfg.Backends.VLLMOmni.Env,
		}
		if len(cfg.Backends.VLLMOmni.Paths) > 0 {
			omniCfg.BinPath = cfg.Backends.VLLMOmni.Paths[0].Path
			omniCfg.BinPaths = make([]string, 0, len(cfg.Backends.VLLMOmni.Paths))
			for _, p := range cfg.Backends.VLLMOmni.Paths {
				omniCfg.BinPaths = append(omniCfg.BinPaths, p.Path)
			}
		}
		r.configs[BackendVLLMOmni] = omniCfg
	}
}

// DiscoverAll discovers all configured backends and returns info about each.
// The lock is released during potentially slow I/O operations.
func (r *Registry) DiscoverAll() map[BackendType]*BackendInfo {
	type snapshot struct {
		backend Backend
		config  *BackendConfig
	}

	// Snapshot backends and configs under lock
	r.mu.RLock()
	items := make([]snapshot, 0, len(r.backends))
	for bt, b := range r.backends {
		cfg := r.configs[bt]
		if cfg == nil {
			cfg = &BackendConfig{Type: bt}
		}
		items = append(items, snapshot{backend: b, config: cfg})
	}
	r.mu.RUnlock()

	// Perform discovery without holding the lock
	results := make(map[BackendType]*BackendInfo)
	for _, item := range items {
		info, err := item.backend.Discover(item.config)
		if err != nil {
			logger.Warnf("backend discovery failed: backend=%s, error=%v", item.backend.Type(), err)
			results[item.backend.Type()] = &BackendInfo{
				Type:      item.backend.Type(),
				Available: false,
			}
			continue
		}
		results[item.backend.Type()] = info
	}
	return results
}

// ErrBackendNotFound creates a "backend not found" error
func ErrBackendNotFound(bt BackendType) error {
	return &BackendError{Msg: "backend not found: " + string(bt)}
}

// BackendError represents a backend-related error
type BackendError struct {
	Msg string
}

func (e *BackendError) Error() string { return e.Msg }

// IsSafeTensorsModel checks if a path looks like a HuggingFace safetensors model directory
func IsSafeTensorsModel(path string) bool {
	base := filepath.Base(path)

	// Individual safetensors file
	if strings.HasSuffix(base, ".safetensors") {
		return true
	}

	// HuggingFace model directory pattern
	if strings.Contains(path, "snapshots") {
		return true
	}

	return false
}

// IsGGUFModel checks if a path is a GGUF model file
func IsGGUFModel(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(base, ".gguf")
}
