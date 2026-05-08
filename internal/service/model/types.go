// Package model provides model scanning and management functionality.
// It handles discovering GGUF models, reading their metadata, and managing model loading.
package model

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/gguf"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model/backend"
)

// Model represents a discovered GGUF model with HuggingFace-style management
type Model struct {
	// Basic information
	ID          string   // Unique model ID (path hash)
	Name        string   // Model name
	DisplayName string   // Display name for UI (handles duplicates)
	Alias       string   // Display alias
	Description string   // Model description/card (HuggingFace-style)
	Path        string   // File path to the GGUF file
	PathPrefix  string   // Path prefix for duplicate identification (e.g., "models/A", "cache/B")
	Size        int64    // File size in bytes
	Favourite   bool     // User's favorite flag
	Tags        []string // Model tags for categorization (e.g., "chat", "code", "multilingual")
	License     string   // Model license
	Author      string   // Model author/organization
	Downloads   int      // Download count for downloaded models

	// GGUF metadata
	Metadata *gguf.Metadata

	// Additional files (e.g., mmproj)
	MmprojPath string
	MmprojMeta *gguf.Metadata

	// 分卷文件信息（Split GGUF files）
	ShardCount int      // 分卷数量，0 表示非分卷模型
	ShardFiles []string // 所有分卷文件路径（仅主模型使用）
	TotalSize  int64    // 包含所有分卷的总大小（仅主模型使用）

	// Scanning info
	ScannedAt  time.Time
	SourcePath string // Original scan path
	SourceType string // "local", "huggingface", "modelscope"

	// Usage statistics
	LoadCount   int       // Number of times loaded
	LastLoaded  time.Time // Last load time
	TotalTokens int64     // Total tokens generated (if tracked)
}

type ModelStatus struct {
	ID          string
	Name        string
	State       LoadState
	ProcessID   string
	Port        int
	CtxSize     int
	LoadedAt    time.Time
	BackendType string // 加载时使用的后端类型 (llamacpp/vllm/vllm_omni)
	Error       error

	mu sync.Mutex

	LoadWait         sync.WaitGroup
	InflightWg       sync.WaitGroup
	InflightCount    atomic.Int32
	LastRequestTime  time.Time
	ConcurrencySem   chan struct{}
	ConcurrencyLimit int
	UnloadAfter      time.Duration

	TotalPromptTokens     int64
	TotalCompletionTokens int64
	tokenMu               sync.Mutex
}

func (s *ModelStatus) transitionTo(newState LoadState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !isValidTransition(s.State, newState) {
		return fmt.Errorf("forbidden state transition: %s -> %s", s.State, newState)
	}
	s.State = newState
	return nil
}

type LoadState int

const (
	StateUnloaded LoadState = iota
	StateLoading
	StateLoaded
	StateUnloading
	StateError
)

func (s LoadState) String() string {
	switch s {
	case StateUnloaded:
		return "stopped"
	case StateLoading:
		return "loading"
	case StateLoaded:
		return "running"
	case StateUnloading:
		return "unloading"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

var validTransitions = map[LoadState][]LoadState{
	StateUnloaded:  {StateLoading},
	StateLoading:   {StateLoaded, StateError, StateUnloading},
	StateLoaded:    {StateUnloading, StateError},
	StateUnloading: {StateUnloaded, StateError},
	StateError:     {StateUnloaded, StateLoading},
}

func isValidTransition(from, to LoadState) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ScanResult represents the result of a scan operation
type ScanResult struct {
	Models       []*Model
	Errors       []ScanError
	ScannedAt    time.Time
	Duration     time.Duration
	TotalFiles   int
	MatchedFiles int
}

// ScanError represents an error during scanning
type ScanError struct {
	Path  string
	Error string
}

// SpecDecodingParams extends backend.SpecDecodingParams with API-level fields.
// The SpecDraftModelID is resolved to SpecDraftModelPath by the handler.
type SpecDecodingParams struct {
	backend.SpecDecodingParams
	SpecDraftModelID string `json:"specDraftModelId"` // Model ID for draft model, resolved to path by handler
}

// ToBackend converts API-level SpecDecodingParams to backend.SpecDecodingParams
func (p *SpecDecodingParams) ToBackend() *backend.SpecDecodingParams {
	if p == nil {
		return nil
	}
	cp := p.SpecDecodingParams
	return &cp
}

// LoadRequest contains parameters for loading a model
type LoadRequest struct {
	ModelID       string  `json:"modelId"`
	NodeID        string  `json:"nodeId"` // 指定运行节点 ID，为空表示自动调度
	CtxSize       int     `json:"ctxSize"`
	BatchSize     int     `json:"batchSize"`
	Threads       int     `json:"threads"`
	GPULayers     int     `json:"gpuLayers"`
	Temperature   float64 `json:"temperature"`
	TopP          float64 `json:"topP"`
	TopK          int     `json:"topK"`
	RepeatPenalty float64 `json:"repeatPenalty"`
	Seed          int     `json:"seed"`
	NPredict      int     `json:"nPredict"`
	// GPU device selection
	Devices []string `json:"devices"` // -dev flags (e.g., ["cuda:0", "cuda:1"])
	MainGPU int      `json:"mainGpu"` // -mg flag (main GPU index)

	// Backend selection
	BackendType string `json:"backendType,omitempty"` // Explicit backend: "llamacpp", "vllm", "vllm_omni"

	// Custom command configuration
	CustomCmd   string `json:"llamaCppPath"` // Custom llama.cpp binary path override (frontend uses llamaCppPath)
	ExtraParams string `json:"extraArgs"`    // Extra CLI arguments appended to command (frontend uses extraArgs)

	// Vision/Multimodal support
	MmprojPath   string `json:"mmprojPath"`   // Path to mmproj.gguf for vision models
	EnableVision bool   `json:"enableVision"` // Enable vision/multimodal capabilities

	// Performance feature flags
	FlashAttention bool `json:"flashAttention"` // -fa flag
	NoMmap         bool `json:"noMmap"`         // --no-mmap flag
	LockMemory     bool `json:"lockMemory"`     // --mlock flag

	// Server feature flags
	NoWebUI       bool   `json:"noWebUI"`       // --no-webui flag
	EnableMetrics bool   `json:"enableMetrics"` // --metrics flag
	SlotSavePath  string `json:"slotSavePath"`  // --slot-save-path
	CacheRAM      int    `json:"cacheRam"`      // --cache-ram size in MB

	// Chat template configuration
	ChatTemplateFile string `json:"chatTemplateFile"` // --chat-template-file path

	// Runtime configuration
	Timeout int    `json:"timeout"` // --timeout in seconds
	Alias   string `json:"alias"`   // --alias for model identification

	// Batch processing
	UBatchSize    int `json:"uBatchSize"`    // --ubatch-size
	ParallelSlots int `json:"parallelSlots"` // --parallel

	// KV cache configuration
	KVCacheTypeK   string `json:"kvCacheTypeK"`   // --kv-cache-type-k
	KVCacheTypeV   string `json:"kvCacheTypeV"`   // --kv-cache-type-v
	KVCacheUnified bool   `json:"kvCacheUnified"` // --kv-unified
	KVCacheSize    int    `json:"kvCacheSize"`    // --kv-cache-size

	// Additional sampling parameters
	LogitsAll        bool    `json:"logitsAll"`        // --logits-all
	Reranking        bool    `json:"reranking"`        // --reranking
	MinP             float64 `json:"minP"`             // --min-p
	PresencePenalty  float64 `json:"presencePenalty"`  // --presence-penalty
	FrequencyPenalty float64 `json:"frequencyPenalty"` // --frequency-penalty

	// Thread configuration
	ThreadsBatch int `json:"threadsBatch"` // --threads-batch

	// Template and processing
	DirectIo     string `json:"directIo"`     // --dio
	DisableJinja bool   `json:"disableJinja"` // --jinja (false to disable)
	ChatTemplate string `json:"chatTemplate"` // --chat-template
	ContextShift bool   `json:"contextShift"` // --context-shift

	// Extended sampling parameters
	RepeatLastN int     `json:"repeatLastN"` // --repeat-last-n
	TypicalP    float64 `json:"typicalP"`    // --typical-p
	IgnoreEOS   bool    `json:"ignoreEos"`   // --ignore-eos

	// Multi-GPU configuration
	SplitMode   string `json:"splitMode"`   // --split-mode (none, layer, row)
	TensorSplit string `json:"tensorSplit"` // --tensor-split (comma-separated values)

	// Server optimization
	ContBatching bool `json:"contBatching"` // --cont-batching
	CachePrompt  bool `json:"cachePrompt"`  // --cache-prompt

	// Structured generation
	Grammar     string `json:"grammar"`     // --grammar
	GrammarFile string `json:"grammarFile"` // --grammar-file

	// LoRA adapter support
	Lora       string `json:"lora"`       // --lora
	LoraScaled string `json:"loraScaled"` // --lora-scaled

	// Chat template kwargs
	ChatTemplateKwargs string `json:"chatTemplateKwargs"` // --chat-template-kwargs

	// RoPE scaling (for extended context)
	RopeScaling   string  `json:"ropeScaling"`   // --rope-scaling
	RopeScale     float64 `json:"ropeScale"`     // --rope-scale
	RopeFreqBase  float64 `json:"ropeFreqBase"`  // --rope-freq-base
	RopeFreqScale float64 `json:"ropeFreqScale"` // --rope-freq-scale

	// TODO: remove after migration period (v0.8.0)
	DraftModelID   string `json:"draftModelId"`   // Deprecated: use SpecDecoding with specType="draft"
	DraftMaxTokens int    `json:"draftMaxTokens"`  // Deprecated: use SpecDecoding with specDraftNMax

	// Speculative decoding (new unified system)
	SpecDecoding *SpecDecodingParams `json:"specDecoding,omitempty"`

	// Runtime management
	UnloadAfterMinutes int `json:"unloadAfterMinutes"` // TTL: idle minutes before auto-unload. 0 = never unload, >0 = custom minutes
	ConcurrencyLimit   int `json:"concurrencyLimit"`   // Max concurrent requests. 0 = unlimited, >0 = custom limit

	// vLLM 专用参数
	DataType             string  `json:"dtype,omitempty"`
	MaxModelLen          int     `json:"maxModelLen,omitempty"`
	GPUMemoryUtilization float64 `json:"gpuMemoryUtilization,omitempty"`
	TensorParallelSize   int     `json:"tensorParallelSize,omitempty"`
	PipelineParallelSize int     `json:"pipelineParallelSize,omitempty"`
	TrustRemoteCode      bool    `json:"trustRemoteCode"`
	ServedModelName      string  `json:"servedModelName,omitempty"`
	Quantization         string  `json:"quantization,omitempty"`
	MaxNumSeqs           int     `json:"maxNumSeqs,omitempty"`
	MaxNumBatchedTokens  int     `json:"maxNumBatchedTokens,omitempty"`
	EnablePrefixCaching  bool    `json:"enablePrefixCaching"`
	EnableChunkedPrefill bool    `json:"enableChunkedPrefill"`
	DisableLogRequests   bool    `json:"disableLogRequests"`
	EnforceEager         bool    `json:"enforceEager"`
	// vLLM-Omni 专用参数
	Omni             bool    `json:"omni"`
	VideoPruningRate float64 `json:"videoPruningRate,omitempty"`
	MMTensorIPC      bool    `json:"mmTensorIPC"`
}

// LoadResult represents the result of a load operation
type LoadResult struct {
	Success       bool
	ModelID       string
	Port          int
	CtxSize       int
	Error         error
	Duration      time.Duration
	Async         bool // 异步加载标志
	Loading       bool // 正在加载中（仅当 Async=true 时有效）
	AlreadyLoaded bool // 模型已加载（仅当 Async=true 时有效）
}

// ModelFilter represents filter criteria for model search
type ModelFilter struct {
	Tags         []string
	Architecture string
	MinContext   int
	MaxSize      int64
	LoadedOnly   bool
	Favourites   bool
	SearchQuery  string
	SourceType   string
	License      string
}

// ModelSort represents sort options for model listing
type ModelSort struct {
	Field     string // "name", "size", "scanned_at", "load_count"
	Direction string // "asc", "desc"
}

// ModelSearchResult represents the result of a model search
type ModelSearchResult struct {
	Models        []*Model
	Total         int
	Filtered      int
	Tags          map[string]int // Tag frequency
	Architectures map[string]int // Architecture frequency
}
