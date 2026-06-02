/**
 * Model metadata — matches backend server.go fields
 */
export interface ModelMetadata {
  // Basic info
  name?: string;
  architecture: string;
  quantization?: string;
  type?: string; // "model", "adapter", "projector", "imatrix"
  author?: string | null;
  url?: string | null;
  description?: string | null;
  license?: string | null;

  // File type info
  fileType?: number;
  fileTypeDescriptor?: string; // Detailed file type (e.g. Q4_K_M, Q5_0_L)
  quantizationVersion?: number;

  // Model parameters
  parameters?: number;
  bitsPerWeight?: number;

  // File info
  alignment?: number;
  fileSize?: number;
  modelSize?: number;
  littleEndian?: boolean;

  // Model structure
  contextLength?: number;
  embeddingLength?: number;
  feedForwardLength?: number;
  blockCount?: number;       // Layer/block count
  headCount?: number;
  headCountKV?: number;
  layerNormRmsEps?: number;

  // Tokenizer
  tokenCount?: number;
  tokenizerModel?: string;
  bosTokenId?: number;
  eosTokenId?: number;
  padTokenId?: number;
  uncTokenId?: number;

  // RoPE configuration
  ropeDim?: number;
  ropeFreqBase?: number;
  ropeFreqScale?: number;

  // Other
  poolingType?: number;
  chatTemplate?: string;
}

/**
 * Model info
 */
export interface Model {
  id: string;
  name: string;
  displayName: string;
  alias?: string;
  path: string;
  pathPrefix: string;
  size: number;
  // Shard-related fields
  totalSize?: number;    // Total size across all shards
  shardCount?: number;   // Number of shards
  shardFiles?: string[]; // All shard file paths
  favourite: boolean;
  isLoaded: boolean;
  isLoading: boolean;
  isMultimodal: boolean;
  status: ModelStatus;
  port?: number;
  slots?: Slot[];
  metadata: ModelMetadata;
  tags?: string[];
  mmprojPath?: string;
  scannedAt: string;
  sourceType?: string;
  backendType?: string; // Recommended backend plugin ID (llamacpp/vllm/vllmomni)
}

/**
 * Model status
 */
export type ModelStatus = 'stopped' | 'loading' | 'running' | 'unloading' | 'error';

/**
 * Processing slot
 */
export interface Slot {
  id: number;
  isProcessing: boolean;
  isSpeculative?: boolean;
  taskId?: string;
}

/**
 * Speculative decoding configuration
 */
export interface SpecDecodingConfig {
  specType?: string; // none, draft, eagle3, ngram-simple, ngram-map-k, ngram-map-k4v, ngram-mod, ngram-cache

  // draft type parameters
  specDraftModelId?: string;
  specDraftNMax?: number;
  specDraftNMin?: number;
  specDraftPSplit?: number;
  specDraftPMin?: number;
  specDraftCtxSize?: number;
  specDraftNgl?: number;
  specDraftDevice?: string;

  // ngram-mod parameters
  specNgramModNMin?: number;
  specNgramModNMax?: number;
  specNgramModNMatch?: number;

  // ngram-simple parameters
  specNgramSimpleSizeN?: number;
  specNgramSimpleSizeM?: number;
  specNgramSimpleMinHits?: number;

  // ngram-map-k parameters
  specNgramMapKSizeN?: number;
  specNgramMapKSizeM?: number;
  specNgramMapKMinHits?: number;

  // ngram-map-k4v parameters
  specNgramMapK4VSizeN?: number;
  specNgramMapK4VSizeM?: number;
  specNgramMapK4VMinHits?: number;

  // ngram-cache parameters
  lookupCacheStatic?: string;
  lookupCacheDynamic?: string;
}

/**
 * Load model parameters
 */
export interface LoadModelParams {
  // Basic parameters
  modelId: string;
  nodeId?: string;              // Target node ID; undefined = auto-schedule
  profileId?: string;           // Reusable launch profile ID
  ctxSize?: number;
  batchSize?: number;
  threads?: number;
  gpuLayers?: number;
  temperature?: number;
  topP?: number;
  topK?: number;
  repeatPenalty?: number;
  seed?: number;
  nPredict?: number;

  // Backend config
  llamaCppPath?: string;      // llama.cpp executable path
  mainGpu?: number | string;  // Primary GPU selection
  backendType?: string;       // Plugin ID (llamacpp/vllm/vllmomni)

  // Capability toggles
  capabilities?: ModelCapabilities & {
    translation?: boolean;
  };

  // Context & acceleration
  flashAttention?: boolean;       // Flash Attention
  noMmap?: boolean;               // Disable memory mapping
  lockMemory?: boolean;           // Lock physical memory

  // Sampling parameters
  logitsAll?: boolean;            // Logits all mode
  reranking?: boolean;            // Reranking mode
  minP?: number;                  // Min-P sampling

  // Penalty parameters
  presencePenalty?: number;       // Presence penalty
  frequencyPenalty?: number;      // Frequency penalty

  // Batch parameters
  uBatchSize?: number;            // Micro-batch size
  parallelSlots?: number;         // Parallel slots

  // KV cache
  kvCacheUnified?: boolean;       // Unified KV cache
  kvCacheTypeK?: string;          // KV cache type K (f16, f32, q8_0)
  kvCacheTypeV?: string;          // KV cache type V

  // Other parameters
  directIo?: string;              // --direct-io  (on/off)
  disableJinja?: boolean;         // --no-jinja
  chatTemplate?: string;          // --chat-template
  contextShift?: boolean;         // --context-shift
  extraArgs?: string;             // Extra CLI arguments (appended verbatim)

  // Thread config
  threadsBatch?: number;          // -tb / --threads-batch
  threadsHttp?: number;           // --threads-http  (HTTP handler threads)

  // Extended sampling: penalties
  repeatLastN?: number;           // --repeat-last-n
  typicalP?: number;              // --typical-p
  ignoreEos?: boolean;            // --ignore-eos

  // Advanced sampling: DRY
  dryMultiplier?: number;         // --dry-multiplier  (0=disabled)
  dryBase?: number;               // --dry-base  (default 1.75)
  dryAllowedLength?: number;      // --dry-allowed-length
  dryPenaltyLastN?: number;       // --dry-penalty-last-n
  drySequenceBreakers?: string;   // --dry-sequence-breaker  (comma-separated)

  // Advanced sampling: Mirostat
  mirostat?: number;              // --mirostat  (0=disabled, 1, 2)
  mirostatLr?: number;            // --mirostat-lr  eta
  mirostatEnt?: number;           // --mirostat-ent  tau

  // Advanced sampling: dynamic temperature
  dynaTempRange?: number;         // --dynatemp-range  (0=disabled)
  dynaTempExp?: number;           // --dynatemp-exp  (default 1.0)

  // Advanced sampling: XTC
  xtcProbability?: number;        // --xtc-probability  (0=disabled)
  xtcThreshold?: number;          // --xtc-threshold  (1=disabled)

  // Sampling: misc
  topNSigma?: number;             // --top-n-sigma  (-1=disabled)
  samplers?: string;              // --samplers  semicolon-separated

  // Multi-GPU config
  splitMode?: string;             // -sm  (none / layer / row / tensor)
  tensorSplit?: string;           // -ts  (comma-separated GPU fractions)

  // GPU: MoE CPU override
  cpuMoe?: boolean;               // --cpu-moe
  nCpuMoe?: number;               // --n-cpu-moe

  // CPU affinity & NUMA
  cpuMask?: string;               // --cpu-mask  (hex)
  cpuRange?: string;              // --cpu-range  (lo-hi)
  priority?: number;              // --prio  (-1=low … 3=realtime)
  numaStrategy?: string;          // --numa  (distribute/isolate/numactl)

  // Server optimization
  contBatching?: boolean;         // --cont-batching
  cachePrompt?: boolean;          // --cache-prompt
  reusePort?: boolean;            // --reuse-port
  sleepIdleSeconds?: number;      // --sleep-idle-seconds  (-1=disabled)
  slotPromptSimilarity?: number;  // --slot-prompt-similarity  (0=disabled)

  // Structured generation
  grammar?: string;               // --grammar
  grammarFile?: string;           // --grammar-file
  jsonSchema?: string;            // -j / --json-schema
  jsonSchemaFile?: string;        // -jf / --json-schema-file

  // LoRA adapters
  lora?: string;                  // --lora
  loraScaled?: string;            // --lora-scaled

  // Chat template extra parameters
  chatTemplateKwargs?: string;    // --chat-template-kwargs

  // RoPE extension
  ropeScaling?: string;           // --rope-scaling  (none/linear/yarn)
  ropeScale?: number;             // --rope-scale
  ropeFreqBase?: number;          // --rope-freq-base
  ropeFreqScale?: number;         // --rope-freq-scale

  // YaRN extended context
  yarnOrigCtx?: number;           // --yarn-orig-ctx
  yarnExtFactor?: number;         // --yarn-ext-factor
  yarnAttnFactor?: number;        // --yarn-attn-factor
  yarnBetaSlow?: number;          // --yarn-beta-slow
  yarnBetaFast?: number;          // --yarn-beta-fast

  // KV cache extended
  kvOffload?: boolean;            // --kv-offload
  cacheIdleSlots?: boolean;       // --cache-idle-slots
  cacheReuse?: number;            // --cache-reuse
  ctxCheckpoints?: number;        // --ctx-checkpoints
  checkpointMinStep?: number;     // --checkpoint-min-step

  draftModelId?: string;   // Deprecated: use specDecoding
  draftMaxTokens?: number;  // Deprecated: use specDecoding

  // Speculative decoding (new unified system)
  specDecoding?: SpecDecodingConfig;

  // Embedding & retrieval
  embedding?: boolean;
  pooling?: string;               // --pooling  (none/mean/cls/last/rank)
  embdNormalize?: number;         // --embd-normalize  (-1=none, 0=max, 1=taxicab, 2=euclidean)

  // UI options
  noWebUI?: boolean;

  // Reasoning / thinking (DeepSeek-R1, QwQ, etc.)
  reasoning?: string;             // --reasoning  (auto/on/off)
  reasoningFormat?: string;       // --reasoning-format  (none/deepseek/deepseek-legacy)
  reasoningBudget?: number;       // --reasoning-budget  (-1=unlimited)

  // Multimedia
  mmprojOffload?: boolean;        // --mmproj-offload

  // Runtime management
  unloadAfterMinutes?: number;   // Auto-unload idle time (minutes). 0=never, >0=custom
  concurrencyLimit?: number;     // Max concurrent requests. 0=unlimited, >0=custom limit

  // vLLM 专用参数
  dtype?: string;                      // Data type: auto/float16/bfloat16/float32
  maxModelLen?: number;                // Maximum context length
  gpuMemoryUtilization?: number;       // GPU memory utilization ratio (0-1), default 0.92
  tensorParallelSize?: number;         // Tensor parallelism GPU count
  pipelineParallelSize?: number;       // Pipeline parallelism group count
  trustRemoteCode?: boolean;           // Trust remote code
  servedModelName?: string;            // API model name alias
  quantization?: string;               // Quantization method
  maxNumSeqs?: number;                 // Maximum concurrent sequences
  maxNumBatchedTokens?: number;        // Maximum tokens per iteration
  enablePrefixCaching?: boolean;       // Prefix caching
  enableChunkedPrefill?: boolean;      // Chunked prefill
  disableLogRequests?: boolean;        // Disable request logging
  enforceEager?: boolean;              // Enforce eager execution mode
  // vLLM-Omni 专用参数
  omni?: boolean;                      // 启用 omni 多模态模式
  videoPruningRate?: number;           // Video token pruning rate (0-1)
  mmTensorIPC?: boolean;               // Multimodal tensor IPC

  // Environment variables (for vllm/vllmomni backends)
  envVars?: string[];                  // 附加环境变量，格式: "KEY=VALUE"

  // Parameter enable flags: false = use llama-server defaults
  enabled?: {
    // Basic parameters
    ctxSize?: boolean;
    batchSize?: boolean;
    threads?: boolean;
    threadsBatch?: boolean;
    threadsHttp?: boolean;
    gpuLayers?: boolean;
    temperature?: boolean;
    topP?: boolean;
    topK?: boolean;
    minP?: boolean;
    topNSigma?: boolean;
    repeatPenalty?: boolean;
    repeatLastN?: boolean;
    typicalP?: boolean;
    presencePenalty?: boolean;
    frequencyPenalty?: boolean;
    ignoreEos?: boolean;
    seed?: boolean;
    nPredict?: boolean;
    samplers?: boolean;

    // DRY sampling
    dryMultiplier?: boolean;
    dryBase?: boolean;
    dryAllowedLength?: boolean;
    dryPenaltyLastN?: boolean;
    drySequenceBreakers?: boolean;

    // Mirostat
    mirostat?: boolean;
    mirostatLr?: boolean;
    mirostatEnt?: boolean;

    // Dynamic temperature
    dynaTempRange?: boolean;
    dynaTempExp?: boolean;

    // XTC
    xtcProbability?: boolean;
    xtcThreshold?: boolean;

    // Batch & cache
    uBatchSize?: boolean;
    parallelSlots?: boolean;
    contBatching?: boolean;
    cachePrompt?: boolean;
    reusePort?: boolean;
    sleepIdleSeconds?: boolean;
    slotPromptSimilarity?: boolean;

    // KV cache
    kvCacheUnified?: boolean;
    kvCacheTypeK?: boolean;
    kvCacheTypeV?: boolean;
    kvOffload?: boolean;
    cacheIdleSlots?: boolean;
    cacheReuse?: boolean;
    ctxCheckpoints?: boolean;
    checkpointMinStep?: boolean;

    // Performance options
    flashAttention?: boolean;
    noMmap?: boolean;
    lockMemory?: boolean;
    directIo?: boolean;

    // GPU config
    splitMode?: boolean;
    tensorSplit?: boolean;
    cpuMoe?: boolean;
    nCpuMoe?: boolean;

    // CPU affinity & NUMA
    cpuMask?: boolean;
    cpuRange?: boolean;
    priority?: boolean;
    numaStrategy?: boolean;

    // Structured generation
    grammar?: boolean;
    grammarFile?: boolean;
    jsonSchema?: boolean;
    jsonSchemaFile?: boolean;

    // LoRA
    lora?: boolean;
    loraScaled?: boolean;

    // Template
    chatTemplate?: boolean;
    chatTemplateKwargs?: boolean;
    disableJinja?: boolean;

    // RoPE extension
    ropeScaling?: boolean;
    ropeScale?: boolean;
    ropeFreqBase?: boolean;
    ropeFreqScale?: boolean;

    // YaRN
    yarnOrigCtx?: boolean;
    yarnExtFactor?: boolean;
    yarnAttnFactor?: boolean;
    yarnBetaSlow?: boolean;
    yarnBetaFast?: boolean;

    // Other
    contextShift?: boolean;
    extraArgs?: boolean;
    logitsAll?: boolean;
    reranking?: boolean;
    pooling?: boolean;
    embdNormalize?: boolean;
    embedding?: boolean;
    timeout?: boolean;
    alias?: boolean;

    // UI options
    noWebUI?: boolean;

    // Reasoning
    reasoning?: boolean;
    reasoningFormat?: boolean;
    reasoningBudget?: boolean;

    // Multimedia
    mmprojOffload?: boolean;

    specDecoding?: boolean;

    // Runtime management
    unloadAfterMinutes?: boolean;
    concurrencyLimit?: boolean;

    // vLLM parameters
    dtype?: boolean;
    maxModelLen?: boolean;
    gpuMemoryUtilization?: boolean;
    tensorParallelSize?: boolean;
    pipelineParallelSize?: boolean;
    trustRemoteCode?: boolean;
    servedModelName?: boolean;
    quantization?: boolean;
    maxNumSeqs?: boolean;
    maxNumBatchedTokens?: boolean;
    enablePrefixCaching?: boolean;
    enableChunkedPrefill?: boolean;
    disableLogRequests?: boolean;
    enforceEager?: boolean;
    // vLLM-Omni parameters
    omni?: boolean;
    videoPruningRate?: boolean;
    mmTensorIPC?: boolean;
    // 环境变量
    envVars?: boolean;
  };
}

/**
 * Model list response
 */
export interface ModelListResponse {
  models: Model[];
  total: number;
  loaded: number;
}

/**
 * Model capabilities
 */
export interface ModelCapabilities {
  thinking?: boolean;         // Thinking ability (e.g. DeepSeek-R1)
  tools?: boolean;            // Tool use / function calling
  rerank?: boolean;           // Reranking
  embedding?: boolean;        // Embedding generation
  tts?: boolean;              // Text-to-speech
  asr?: boolean;              // Automatic speech recognition
  imageGeneration?: boolean;  // Image generation (text-to-image)
  music?: boolean;            // Music generation (text-to-music)
}

/**
 * Benchmark parameter type
 */
export type BenchmarkParamType = 'STRING' | 'INTEGER' | 'FLOAT' | 'LOGIC';

/**
 * Benchmark parameter definition
 */
export interface BenchmarkParam {
  fullName: string;        // Full parameter name, e.g. --threads
  name: string;            // Display name
  abbreviation: string;    // Abbreviation
  description: string;     // Description
  type: BenchmarkParamType; // Parameter type
  defaultValue: string;    // Default value
  defaultEnabled?: boolean; // Whether enabled by default (undefined = true)
  values?: (string | { value: string; label: string })[]; // Allowed values (enum)
  sort?: number;           // Sort order
  group?: string;          // Group identifier
  groupOrder?: number;     // Group sort order
  groupCollapsed?: boolean; // Whether group starts collapsed
}

/**
 * Benchmark params response
 */
export interface BenchmarkParamsResponse {
  success: boolean;
  params?: BenchmarkParam[];
  error?: string;
}

/**
 * llama.cpp version info
 */
export interface LlamaCppVersion {
  path: string;            // Executable path
  name?: string;           // Display name
  description?: string;    // Description
}

/**
 * Benchmark configuration
 */
export interface BenchmarkConfig {
  modelId: string;         // Model ID
  modelName: string;       // Model name
  llamaCppPath: string;    // llama.cpp path
  devices?: string[];      // Selected devices (empty = auto)
  params: Record<string, string | number | boolean>; // Benchmark parameter key-value pairs
  configName?: string;     // Config name (for saving)
}

/**
 * Benchmark result
 */
export interface BenchmarkResult {
  id: string;              // Result ID
  benchmarkId: string;     // Associated benchmark ID
  modelId: string;         // Model ID
  modelName: string;       // Model name
  command: string[];       // Executed command
  commandStr: string;      // Command string
  exitCode: number;        // Exit code
  rawOutput: string;       // Raw output
  fileName: string;        // Saved file name
  savedPath: string;       // Saved path
  timestamp: string;       // Timestamp
  // Parsed performance metrics
  metrics?: {
    tps?: number;          // Tokens per second
    promptTps?: number;    // Prompt processing speed
    totalTokens?: number;  // Total tokens processed
    loadTime?: number;     // Model load time (ms)
    memoryUsage?: number;  // Memory usage (MB)
  };
}

/**
 * Benchmark result file entry
 */
export interface BenchmarkResultFile {
  name: string;            // File name
  size: number;            // File size
  modified: string;        // Modified time
}

/**
 * Benchmark history file (from file system)
 */
export interface BenchmarkHistoryFile {
  name: string;            // File name
  size: number;            // File size
  modified: string;        // Modified time
}

/**
 * Create benchmark request
 */
export interface CreateBenchmarkRequest {
  modelId: string;
  llamaBinPath: string;
  cmd?: string;            // Benchmark command string
  args?: string[];         // Benchmark command arguments
  configName?: string;     // Optional config name
  nodeId?: string;         // Optional node ID for distributed benchmarking
}

/**
 * Create benchmark response
 */
export interface CreateBenchmarkResponse {
  success: boolean;
  data?: BenchmarkResult;
  error?: string;
}

/**
 * V2 Benchmark: test loaded model throughput via chat completions
 */
export interface BenchmarkV2Request {
  modelId: string;
  promptTokens: number;
  maxTokens: number;
  nodeId?: string;
}

export interface BenchmarkV2Timings {
  prompt_n?: number;
  prompt_ms?: number;
  prompt_per_second?: number;
  predicted_n?: number;
  predicted_ms?: number;
  predicted_per_second?: number;
}

export interface BenchmarkV2Record {
  timestamp: string;
  modelId: string;
  promptTokens: number;
  maxTokens: number;
  timings: BenchmarkV2Timings;
  devices?: string[];
  cmd?: string;
  lineNumber?: number;
}

export interface BenchmarkV2Response {
  success: boolean;
  data?: BenchmarkV2Record;
  error?: string;
}

/**
 * Benchmark task status from TaskManager
 */
export type BenchmarkTaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

/**
 * Benchmark task from backend TaskManager
 */
export interface BenchmarkTask {
  id: string;
  type: 'benchmark';
  status: BenchmarkTaskStatus;
  name: string;
  modelId?: string;
  modelName?: string;
  command?: string;
  error?: string;
  progress?: number;
  metrics?: Record<string, unknown>;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}
