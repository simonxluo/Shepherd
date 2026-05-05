// Package backend defines the interface for inference backend engines
// (llama.cpp, vLLM, vLLM-omni) and provides a registry for managing them.
package backend

import "fmt"

// BackendType identifies the type of inference backend
type BackendType string

const (
	BackendLlamaCpp BackendType = "llamacpp"
	BackendVLLM     BackendType = "vllm"
	BackendVLLMOmni BackendType = "vllm_omni"
)

// String returns the string representation of the backend type
func (t BackendType) String() string {
	return string(t)
}

// ParseBackendType parses a string into a BackendType.
// Returns an error for unknown types instead of silently defaulting.
func ParseBackendType(s string) (BackendType, error) {
	switch BackendType(s) {
	case "", BackendLlamaCpp:
		return BackendLlamaCpp, nil
	case BackendVLLM:
		return BackendVLLM, nil
	case BackendVLLMOmni:
		return BackendVLLMOmni, nil
	default:
		return "", fmt.Errorf("unknown backend type: %q (valid: llamacpp, vllm, vllm_omni)", s)
	}
}

// BackendConfig is the configuration for discovering a backend
type BackendConfig struct {
	Type BackendType
	Name string // Human-readable name
	// LlamaCpp-specific
	BinPath  string   // Directory containing llama-server binary
	BinPaths []string // All configured binary paths (tried in order)
	// vLLM / vLLM-omni specific
	CondaEnv    string   // Conda environment name
	CondaPath   string   // Path to conda executable
	ServeBin    string   // Custom path to the serve binary (e.g., vllm)
	ExtraArgs   string   // Additional CLI arguments from config
	DefaultPort int      // Default port from config
	EnvVars     []string // Additional environment variables (e.g., "KEY=VALUE")
}

// BackendInfo contains discovered information about a backend
type BackendInfo struct {
	Type      BackendType
	Name      string
	BinPath   string // Resolved binary path or conda lib dir
	Version   string // Detected version
	Available bool   // Whether the backend is usable
	CondaEnv  string // Conda env name (if applicable)
	CondaPath string // Conda 可执行文件路径 (if applicable)
}

// StartConfig contains the command and metadata needed to start a backend process
type StartConfig struct {
	Command           string // Full command string to execute
	BinPath           string // Binary/library path for the process manager
	BackendType       BackendType
	SkipLDLibraryPath bool     // If true, skip setting LD_LIBRARY_PATH (for conda-based backends)
	CondaPath         string   // Conda 可执行文件路径 (传递给 process 层使用)
	EnvVars           []string // Additional environment variables (e.g., "KEY=VALUE")
}

// HealthResult contains the result of a health check
type HealthResult struct {
	Healthy bool
	Body    string
}

// LoadRequest contains parameters for building a start command
type LoadRequest struct {
	ModelPath string
	Port      int

	// Common parameters
	CtxSize   int
	GPULayers int
	Threads   int
	Devices   []string // GPU devices (e.g., ["cuda:0", "cuda:1"])

	DraftModelPath string
	DraftMaxTokens int

	// LlamaCpp-specific
	LlamacppParams *LlamacppLoadParams

	// vLLM-specific
	VLLMParams *VLLMLoadParams

	// vLLM-omni-specific
	VLLOmniParams *VLLOmniLoadParams
}

// Backend is the interface that all inference backends must implement
type Backend interface {
	// Type returns the backend type identifier
	Type() BackendType

	// Discover validates that the backend is installed and returns info about it
	Discover(cfg *BackendConfig) (*BackendInfo, error)

	// BuildStartConfig constructs the command to start serving a model
	BuildStartConfig(info *BackendInfo, req *LoadRequest) (*StartConfig, error)

	// IsLoadComplete checks if a line of stdout indicates the model is ready
	IsLoadComplete(outputLine string) bool

	// CheckHealth performs a health check on the running backend
	CheckHealth(port int) (*HealthResult, error)

	// SupportsModel returns true if this backend can serve the given model path
	SupportsModel(modelPath string) bool

	// SupportedEndpoints returns the set of API endpoints this backend supports
	SupportedEndpoints() map[string]bool
}

// LlamacppLoadParams contains llama.cpp-specific load parameters
type LlamacppLoadParams struct {
	BatchSize          int
	Temperature        float64
	TopP               float64
	TopK               int
	RepeatPenalty      float64
	Seed               int
	NPredict           int
	MainGPU            int
	CustomCmd          string
	ExtraParams        string
	MmprojPath         string
	EnableVision       bool
	FlashAttention     bool
	NoMmap             bool
	LockMemory         bool
	NoWebUI            bool
	EnableMetrics      bool
	SlotSavePath       string
	CacheRAM           int
	ChatTemplateFile   string
	Timeout            int
	Alias              string
	UBatchSize         int
	ParallelSlots      int
	KVCacheTypeK       string
	KVCacheTypeV       string
	KVCacheUnified     bool
	KVCacheSize        int
	LogitsAll          bool
	Reranking          bool
	MinP               float64
	PresencePenalty    float64
	FrequencyPenalty   float64
	DirectIo           string
	DisableJinja       bool
	ChatTemplate       string
	ContextShift       bool
	ThreadsBatch       int
	RepeatLastN        int
	TypicalP           float64
	IgnoreEOS          bool
	SplitMode          string
	TensorSplit        string
	ContBatching       bool
	CachePrompt        bool
	Grammar            string
	GrammarFile        string
	Lora               string
	LoraScaled         string
	ChatTemplateKwargs string
	RopeScaling        string
	RopeScale          float64
	RopeFreqBase       float64
	RopeFreqScale      float64
	DraftModelPath     string
	DraftMaxTokens     int
}

// VLLMLoadParams contains vLLM-specific load parameters
type VLLMLoadParams struct {
	DataType             string  // --dtype (e.g., "auto", "float16", "bfloat16")
	MaxModelLen          int     // --max-model-len
	GPUMemoryUtilization float64 // --gpu-memory-utilization (0.0-1.0)
	TensorParallelSize   int     // --tensor-parallel-size
	PipelineParallelSize int     // --pipeline-parallel-size
	TrustRemoteCode      bool    // --trust-remote-code
	ServedModelName      string  // --served-model-name
	Quantization         string  // --quantization (e.g., "awq", "gptq")
	MaxNumSeqs           int     // --max-num-seqs
	MaxNumBatchedTokens  int     // --max-num-batched-tokens
	EnablePrefixCaching  bool    // --enable-prefix-caching
	EnableChunkedPrefill bool    // --enable-chunked-prefill
	DisableLogRequests   bool    // --disable-log-requests
	EnforceEager         bool    // --enforce-eager
	ExtraArgs            string  // Additional CLI arguments
}

// VLLOmniLoadParams contains vLLM-omni-specific load parameters
type VLLOmniLoadParams struct {
	VLLMLoadParams           // Embed base vLLM params
	Omni             bool    // --omni (启用 omni 多模态模式)
	VideoPruningRate float64 // --video-pruning-rate
	MMTensorIPC      bool    // --mm-tensor-ipc
}

// Validate checks that the LoadRequest has required fields
func (r *LoadRequest) Validate() error {
	if r.ModelPath == "" {
		return fmt.Errorf("model path cannot be empty")
	}
	if r.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	return nil
}

// baseEndpoints 返回所有后端共有的基础端点集合
func baseEndpoints() map[string]bool {
	return map[string]bool{
		"/v1/chat/completions": true,
		"/v1/completions":      true,
		"/v1/models":           true,
		"/v1/embeddings":       true,
	}
}

// endpointsWithoutAudio 返回不支持音频端点的基础端点集合
func endpointsWithoutAudio() map[string]bool {
	ep := baseEndpoints()
	ep["/v1/audio/speech"] = false
	ep["/v1/audio/voices"] = false
	ep["/v1/audio/transcriptions"] = false
	ep["/v1/audio/translations"] = false
	return ep
}

// endpointsWithAudio 返回支持全部音频端点的端点集合
func endpointsWithAudio() map[string]bool {
	ep := baseEndpoints()
	ep["/v1/audio/speech"] = true
	ep["/v1/audio/voices"] = true
	ep["/v1/audio/transcriptions"] = true
	ep["/v1/audio/translations"] = true
	return ep
}
