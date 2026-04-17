// Package model provides model scanning and management functionality.
// It handles discovering GGUF models, reading their metadata, and managing model loading.
package model

import (
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/gguf"
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

// ModelStatus represents the loading status of a model
type ModelStatus struct {
	ID        string
	Name      string
	State     LoadState
	ProcessID string
	Port      int
	CtxSize   int
	LoadedAt  time.Time
	Error     error
}

// LoadState represents the loading state
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
		return "unloaded"
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

// ScanConfig contains configuration for model scanning
type ScanConfig struct {
	Paths          []string
	Recursive      bool
	FollowSymlinks bool
	MaxDepth       int
	IncludePattern string // Regex pattern for files to include
	ExcludePattern string // Regex pattern for files to exclude
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
