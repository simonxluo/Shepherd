// Package llamacpp implements the backend.Plugin contract for llama.cpp.
package llamacpp

import "github.com/simonxluo/Shepherd/internal/backend"

// Params holds all llama.cpp-specific load parameters.
// Each field maps to a llama-server CLI flag.
type Params struct {
	backend.ParamsBase
	// Context & batch
	CtxSize    int `json:"ctxSize"`
	BatchSize  int `json:"batchSize"`
	UBatchSize int `json:"ubatchSize"`

	// Threading
	Threads      int `json:"threads"`
	ThreadsBatch int `json:"threadsBatch"`
	ThreadsHTTP  int `json:"threadsHttp"`

	// GPU offloading
	GPULayers   int    `json:"gpuLayers"`
	MainGPU     int    `json:"mainGpu"`
	SplitMode   string `json:"splitMode"`
	TensorSplit string `json:"tensorSplit"`
	CpuMoe      bool   `json:"cpuMoe"`
	NCpuMoe     int    `json:"nCpuMoe"`

	// Sampling: basic
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"topP"`
	TopK        int     `json:"topK"`
	MinP        float64 `json:"minP"`
	TopNSigma   float64 `json:"topNSigma"`
	TypicalP    float64 `json:"typicalP"`
	Seed        int     `json:"seed"`
	NPredict    int     `json:"nPredict"`
	Samplers    string  `json:"samplers"`

	// Sampling: penalties
	RepeatPenalty    float64 `json:"repeatPenalty"`
	RepeatLastN      int     `json:"repeatLastN"`
	PresencePenalty  float64 `json:"presencePenalty"`
	FrequencyPenalty float64 `json:"frequencyPenalty"`
	IgnoreEOS        bool    `json:"ignoreEos"`

	// Sampling: DRY
	DryMultiplier       float64 `json:"dryMultiplier"`
	DryBase             float64 `json:"dryBase"`
	DryAllowedLength    int     `json:"dryAllowedLength"`
	DryPenaltyLastN     int     `json:"dryPenaltyLastN"`
	DrySequenceBreakers string  `json:"drySequenceBreakers"`

	// Sampling: Mirostat
	Mirostat    int     `json:"mirostat"`
	MirostatLR  float64 `json:"mirostatLr"`
	MirostatEnt float64 `json:"mirostatEnt"`

	// Sampling: dynamic temperature
	DynaTempRange float64 `json:"dynaTempRange"`
	DynaTempExp   float64 `json:"dynaTempExp"`

	// Sampling: XTC
	XTCProbability float64 `json:"xtcProbability"`
	XTCThreshold   float64 `json:"xtcThreshold"`

	// CPU affinity & NUMA
	CpuMask      string `json:"cpuMask"`
	CpuRange     string `json:"cpuRange"`
	Priority     int    `json:"priority"`
	NumaStrategy string `json:"numaStrategy"`

	// Memory
	NoMmap     bool   `json:"noMmap"`
	LockMemory bool   `json:"lockMemory"`
	DirectIO   string `json:"directIo"`

	// Flash attention
	FlashAttention bool `json:"flashAttention"`

	// KV cache
	KVCacheTypeK         string  `json:"kvCacheTypeK"`
	KVCacheTypeV         string  `json:"kvCacheTypeV"`
	KVCacheUnified       bool    `json:"kvCacheUnified"`
	KVOffload            bool    `json:"kvOffload"`
	CacheIdleSlots       bool    `json:"cacheIdleSlots"`
	CacheReuse           int     `json:"cacheReuse"`
	CtxCheckpoints       int     `json:"ctxCheckpoints"`
	CheckpointMinStep    int     `json:"checkpointMinStep"`
	SlotPromptSimilarity float64 `json:"slotPromptSimilarity"`

	// Server operation
	ParallelSlots    int    `json:"parallelSlots"`
	ContBatching     bool   `json:"contBatching"`
	CachePrompt      bool   `json:"cachePrompt"`
	CacheRAM         int    `json:"cacheRam"`
	SlotSavePath     string `json:"slotSavePath"`
	ReusePort        bool   `json:"reusePort"`
	SleepIdleSeconds int    `json:"sleepIdleSeconds"`
	Timeout          int    `json:"timeout"`
	Alias            string `json:"alias"`
	NoWebUI          bool   `json:"noWebUI"`
	EnableMetrics    bool   `json:"enableMetrics"`

	// Reasoning / thinking
	Reasoning       string `json:"reasoning"`
	ReasoningFormat string `json:"reasoningFormat"`
	ReasoningBudget int    `json:"reasoningBudget"`

	// Embedding / reranking
	LogitsAll     bool   `json:"logitsAll"`
	Reranking     bool   `json:"reranking"`
	Pooling       string `json:"pooling"`
	EmbdNormalize int    `json:"embdNormalize"`

	// Multimodal
	MmprojPath    string `json:"mmprojPath"`
	EnableVision  bool   `json:"enableVision"`
	MmprojOffload bool   `json:"mmprojOffload"`

	// Chat template
	ChatTemplateFile   string `json:"chatTemplateFile"`
	ChatTemplate       string `json:"chatTemplate"`
	ChatTemplateKwargs string `json:"chatTemplateKwargs"`
	DisableJinja       bool   `json:"disableJinja"`
	ContextShift       bool   `json:"contextShift"`

	// RoPE scaling
	RopeScaling   string  `json:"ropeScaling"`
	RopeScale     float64 `json:"ropeScale"`
	RopeFreqBase  float64 `json:"ropeFreqBase"`
	RopeFreqScale float64 `json:"ropeFreqScale"`

	// YaRN extended context
	YarnOrigCtx    int     `json:"yarnOrigCtx"`
	YarnExtFactor  float64 `json:"yarnExtFactor"`
	YarnAttnFactor float64 `json:"yarnAttnFactor"`
	YarnBetaSlow   float64 `json:"yarnBetaSlow"`
	YarnBetaFast   float64 `json:"yarnBetaFast"`

	// Structured generation
	Grammar        string `json:"grammar"`
	GrammarFile    string `json:"grammarFile"`
	JSONSchema     string `json:"jsonSchema"`
	JSONSchemaFile string `json:"jsonSchemaFile"`

	// LoRA adapters
	Lora       string `json:"lora"`
	LoraScaled string `json:"loraScaled"`

	// Escape hatch
	CustomCmd   string `json:"customCmd"`
	ExtraParams string `json:"extraParams"`

	// Speculative decoding (llamacpp-specific)
	SpecDecoding *backend.SpecDecodingParams `json:"specDecoding,omitempty"`
}

var _ backend.Params = (*Params)(nil)
