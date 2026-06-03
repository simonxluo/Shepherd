// Package model provides model scanning and lifecycle management functionality.
// It handles discovering GGUF models, reading their metadata, and managing model loading/unloading.
package model

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/simonxluo/Shepherd/internal/infra/gguf"
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

// ModelStatus represents the runtime state of a model, including process info,
// concurrency control, and request statistics.
//
// State machine: StateUnloaded -> StateLoading -> StateLoaded -> StateUnloading -> StateUnloaded
//
//	StateLoading/StateLoaded/StateUnloading -> StateError (on failure)
//	StateError -> StateUnloaded (on recovery)
//
// Concurrency notes:
//   - mu: protects State, Port, ProcessID and other state transition fields
//   - tokenMu: independently protects token statistics to avoid contention with mu
//   - InflightCount/InflightWg: atomic ops + WaitGroup, lock-free
type ModelStatus struct {
	ID         string
	InstanceID string
	Name       string
	State      LoadState
	ProcessID  string
	Port       int
	CtxSize    int
	LoadedAt   time.Time
	PluginID   string // Backend plugin ID used for loading (llamacpp/vllm/vllmomni)
	Error      error

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
	RequestCount          int64
	ErrorCount            int64
	TotalLatencyMs        int64
	FirstRequestAt        time.Time
	LastRequestAt         time.Time
	tokenMu               sync.Mutex
}

// transitionTo transitions the model to newState.
// Returns an error if the transition is invalid (not in validTransitions table).
func (s *ModelStatus) transitionTo(newState LoadState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !isValidTransition(s.State, newState) {
		return fmt.Errorf("forbidden state transition: %s -> %s", s.State, newState)
	}
	s.State = newState
	return nil
}

// LoadState represents the loading state of a model.
// String() returns the user-facing state name (stopped/loading/running/unloading/error).
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
	StateLoaded:    {StateUnloading},
	StateUnloading: {StateUnloaded},
	StateError:     {StateUnloaded},
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

// RuntimeInstance represents a model runtime process instance.
type RuntimeInstance struct {
	InstanceID     string    `json:"instanceId"`
	ModelID        string    `json:"modelId"`
	ModelName      string    `json:"modelName"`
	ProfileID      string    `json:"profileId,omitempty"`
	InstallationID string    `json:"installationId,omitempty"`
	ProcessID      string    `json:"processId,omitempty"`
	Port           int       `json:"port,omitempty"`
	State          string    `json:"state"`
	PluginID       string    `json:"pluginId,omitempty"`
	CommandPreview string    `json:"commandPreview,omitempty"`
	LastError      string    `json:"lastError,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
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

// SpecDecodingParams holds speculative decoding configuration at the API level.
// SpecDraftModelID is resolved to a file path by the handler before loading.
type SpecDecodingParams struct {
	SpecType           string  `json:"specType"`
	SpecDraftModelID   string  `json:"specDraftModelId"`
	SpecDraftModelPath string  `json:"-"`
	SpecDraftNMax      int     `json:"specDraftNMax"`
	SpecDraftNMin      int     `json:"specDraftNMin"`
	SpecDraftPSplit    float64 `json:"specDraftPSplit"`
	SpecDraftPMin      float64 `json:"specDraftPMin"`
	SpecDraftCtxSize   int     `json:"specDraftCtxSize"`
	SpecDraftNGL       int     `json:"specDraftNgl"`
	SpecDraftDevice    string  `json:"specDraftDevice"`

	SpecNgramModNMin   int `json:"specNgramModNMin"`
	SpecNgramModNMax   int `json:"specNgramModNMax"`
	SpecNgramModNMatch int `json:"specNgramModNMatch"`

	SpecNgramSimpleSizeN   int `json:"specNgramSimpleSizeN"`
	SpecNgramSimpleSizeM   int `json:"specNgramSimpleSizeM"`
	SpecNgramSimpleMinHits int `json:"specNgramSimpleMinHits"`

	SpecNgramMapKSizeN   int `json:"specNgramMapKSizeN"`
	SpecNgramMapKSizeM   int `json:"specNgramMapKSizeM"`
	SpecNgramMapKMinHits int `json:"specNgramMapKMinHits"`

	SpecNgramMapK4VSizeN   int `json:"specNgramMapK4VSizeN"`
	SpecNgramMapK4VSizeM   int `json:"specNgramMapK4VSizeM"`
	SpecNgramMapK4VMinHits int `json:"specNgramMapK4VMinHits"`

	LookupCacheStatic  string `json:"lookupCacheStatic"`
	LookupCacheDynamic string `json:"lookupCacheDynamic"`
}

// LoadRequest contains parameters for loading a model
type LoadRequest struct {
	ModelID       string  `json:"modelId"`
	InstanceID    string  `json:"instanceId,omitempty"`
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
	PluginID  string `json:"pluginId,omitempty"`  // Explicit backend plugin: "llamacpp", "vllm", "vllmomni"
	ProfileID string `json:"profileId,omitempty"` // Reusable launch profile ID

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

	// Additional sampling parameters
	LogitsAll        bool    `json:"logitsAll"`        // --logits-all
	Reranking        bool    `json:"reranking"`        // --reranking
	MinP             float64 `json:"minP"`             // --min-p
	PresencePenalty  float64 `json:"presencePenalty"`  // --presence-penalty
	FrequencyPenalty float64 `json:"frequencyPenalty"` // --frequency-penalty

	// Thread configuration
	ThreadsBatch int `json:"threadsBatch"` // --threads-batch

	// Template and processing
	DirectIo     string `json:"directIo"`     // --direct-io (on/off)
	DisableJinja bool   `json:"disableJinja"` // --jinja (false to disable)
	ChatTemplate string `json:"chatTemplate"` // --chat-template
	ContextShift bool   `json:"contextShift"` // --context-shift

	// Extended sampling parameters
	RepeatLastN int     `json:"repeatLastN"` // --repeat-last-n
	TypicalP    float64 `json:"typicalP"`    // --typical-p
	IgnoreEOS   bool    `json:"ignoreEos"`   // --ignore-eos

	// Advanced sampling: DRY
	DryMultiplier       float64 `json:"dryMultiplier"`       // --dry-multiplier  (0=disabled)
	DryBase             float64 `json:"dryBase"`             // --dry-base  (default 1.75)
	DryAllowedLength    int     `json:"dryAllowedLength"`    // --dry-allowed-length
	DryPenaltyLastN     int     `json:"dryPenaltyLastN"`     // --dry-penalty-last-n
	DrySequenceBreakers string  `json:"drySequenceBreakers"` // --dry-sequence-breaker

	// Advanced sampling: Mirostat
	Mirostat    int     `json:"mirostat"`    // --mirostat  (0=disabled, 1, 2)
	MirostatLR  float64 `json:"mirostatLr"`  // --mirostat-lr  (eta)
	MirostatEnt float64 `json:"mirostatEnt"` // --mirostat-ent  (tau)

	// Advanced sampling: dynamic temperature
	DynaTempRange float64 `json:"dynaTempRange"` // --dynatemp-range  (0=disabled)
	DynaTempExp   float64 `json:"dynaTempExp"`   // --dynatemp-exp

	// Advanced sampling: XTC
	XTCProbability float64 `json:"xtcProbability"` // --xtc-probability  (0=disabled)
	XTCThreshold   float64 `json:"xtcThreshold"`   // --xtc-threshold  (1=disabled)

	// Sampling: misc
	TopNSigma float64 `json:"topNSigma"` // --top-n-sigma  (-1=disabled)
	Samplers  string  `json:"samplers"`  // --samplers  semicolon-separated

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
	RopeScaling   string  `json:"ropeScaling"`   // --rope-scaling  (none/linear/yarn)
	RopeScale     float64 `json:"ropeScale"`     // --rope-scale
	RopeFreqBase  float64 `json:"ropeFreqBase"`  // --rope-freq-base
	RopeFreqScale float64 `json:"ropeFreqScale"` // --rope-freq-scale

	// YaRN extended context
	YarnOrigCtx    int     `json:"yarnOrigCtx"`    // --yarn-orig-ctx
	YarnExtFactor  float64 `json:"yarnExtFactor"`  // --yarn-ext-factor
	YarnAttnFactor float64 `json:"yarnAttnFactor"` // --yarn-attn-factor
	YarnBetaSlow   float64 `json:"yarnBetaSlow"`   // --yarn-beta-slow
	YarnBetaFast   float64 `json:"yarnBetaFast"`   // --yarn-beta-fast

	// KV cache extended
	KVOffload            bool    `json:"kvOffload"`            // --kv-offload
	CacheIdleSlots       bool    `json:"cacheIdleSlots"`       // --cache-idle-slots
	CacheReuse           int     `json:"cacheReuse"`           // --cache-reuse
	CtxCheckpoints       int     `json:"ctxCheckpoints"`       // --ctx-checkpoints
	CheckpointMinStep    int     `json:"checkpointMinStep"`    // --checkpoint-min-step
	SlotPromptSimilarity float64 `json:"slotPromptSimilarity"` // --slot-prompt-similarity

	// GPU: MoE CPU override
	CpuMoe  bool `json:"cpuMoe"`  // --cpu-moe
	NCpuMoe int  `json:"nCpuMoe"` // --n-cpu-moe

	// CPU affinity & NUMA
	CpuMask      string `json:"cpuMask"`      // --cpu-mask  (hex)
	CpuRange     string `json:"cpuRange"`     // --cpu-range  (lo-hi)
	Priority     int    `json:"priority"`     // --prio  (-1~3)
	NumaStrategy string `json:"numaStrategy"` // --numa  (distribute/isolate/numactl)

	// Server operation extended
	ThreadsHTTP      int  `json:"threadsHttp"`      // --threads-http
	ReusePort        bool `json:"reusePort"`        // --reuse-port
	SleepIdleSeconds int  `json:"sleepIdleSeconds"` // --sleep-idle-seconds  (-1=disabled)

	// Reasoning / thinking
	Reasoning       string `json:"reasoning"`       // --reasoning  (auto/on/off)
	ReasoningFormat string `json:"reasoningFormat"` // --reasoning-format
	ReasoningBudget int    `json:"reasoningBudget"` // --reasoning-budget  (-1=unlimited)

	// Embedding / reranking extended
	Pooling       string `json:"pooling"`       // --pooling  (none/mean/cls/last/rank)
	EmbdNormalize int    `json:"embdNormalize"` // --embd-normalize  (-1~2)

	// Multimodal extended
	MmprojOffload bool `json:"mmprojOffload"` // --mmproj-offload

	// JSON schema structured generation
	JSONSchema     string `json:"jsonSchema"`     // -j / --json-schema
	JSONSchemaFile string `json:"jsonSchemaFile"` // -jf / --json-schema-file

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

	// 环境变量配置（适用于 vllm/vllm_omni 后端）
	EnvVars []string `json:"envVars,omitempty"` // 附加环境变量，格式: "KEY=VALUE"
}

// LoadResult represents the result of a load operation
type LoadResult struct {
	Success       bool
	ModelID       string
	InstanceID    string
	Port          int
	CtxSize       int
	Error         error
	Duration      time.Duration
	Async         bool // 异步加载标志
	Loading       bool // 正在加载中（仅当 Async=true 时有效）
	AlreadyLoaded bool // 模型已加载（仅当 Async=true 时有效）
}
