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

  // Model architecture parameters (may be 0)
  contextLength?: number;
  embeddingLength?: number;
  layerCount?: number;
  headCount?: number;

  // Retained for backward compatibility (backend no longer returns these)
  blockSize?: number;
  feedForwardLength?: number;
  attentionHeadCount?: number;
  attentionHeadCountKeyValue?: number;
  ropeDimensionCount?: number;
  ggmlFileType?: string;
  tokenizer?: string;
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
  backendType?: string; // 推荐后端类型 (llamacpp/vllm/vllm_omni)
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
 * Load model parameters
 */
export interface LoadModelParams {
  // Basic parameters
  modelId: string;
  nodeId?: string;              // Target node ID; undefined = auto-schedule
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
  backendType?: string;       // Explicit backend type (llamacpp/vllm/vllm_omni)

  // Capability toggles
  capabilities?: ModelCapabilities & {
    thinking?: boolean;
    tools?: boolean;
    translation?: boolean;
    embedding?: boolean;
    tts?: boolean;
    asr?: boolean;
    imageGeneration?: boolean;
    music?: boolean;
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
  kvCacheSize?: number;           // KV cache memory limit
  kvCacheUnified?: boolean;       // Unified KV cache
  kvCacheTypeK?: string;          // KV cache type K (f16, f32, q8_0)
  kvCacheTypeV?: string;          // KV cache type V

  // Other parameters
  directIo?: string;              // DirectIO mode
  disableJinja?: boolean;         // Disable Jinja templates
  chatTemplate?: string;          // Built-in chat template
  contextShift?: boolean;         // Context shift
  extraArgs?: string;             // Extra CLI arguments

  // Thread config
  threadsBatch?: number;          // Batch thread count

  // Extended sampling parameters
  repeatLastN?: number;           // Repeat penalty range
  typicalP?: number;              // Typical sampling
  ignoreEos?: boolean;            // Ignore EOS token

  // Multi-GPU config
  splitMode?: string;             // GPU split mode (none, layer, row)
  tensorSplit?: string;           // Tensor split ratio

  // Server optimization
  contBatching?: boolean;         // Continuous batching
  cachePrompt?: boolean;          // Prompt caching

  // Structured generation
  grammar?: string;               // BNF grammar
  grammarFile?: string;           // Grammar file path

  // LoRA adapters
  lora?: string;                  // LoRA adapter path
  loraScaled?: string;            // Scaled LoRA

  // Chat template extra parameters
  chatTemplateKwargs?: string;    // Extra template JSON parameters

  // RoPE extension
  ropeScaling?: string;           // RoPE scaling method
  ropeScale?: number;             // RoPE scaling factor
  ropeFreqBase?: number;          // RoPE base frequency
  ropeFreqScale?: number;         // RoPE frequency scaling

  draftModelId?: string;
  draftMaxTokens?: number;

  // Embedding & retrieval
  embedding?: boolean;

  // UI options
  noWebUI?: boolean;

  // Reasoning
  reasoning?: string;
  reasoningFormat?: string;
  reasoningBudget?: number;

  // Multimedia
  mmprojOffload?: boolean;

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

  // Parameter enable flags: false = use llama-server defaults
  enabled?: {
    // Basic parameters
    ctxSize?: boolean;
    batchSize?: boolean;
    threads?: boolean;
    threadsBatch?: boolean;
    gpuLayers?: boolean;
    temperature?: boolean;
    topP?: boolean;
    topK?: boolean;
    repeatPenalty?: boolean;
    repeatLastN?: boolean;
    seed?: boolean;
    nPredict?: boolean;

    // Sampling parameters
    minP?: boolean;
    typicalP?: boolean;
    presencePenalty?: boolean;
    frequencyPenalty?: boolean;
    ignoreEos?: boolean;

    // Batch & cache
    uBatchSize?: boolean;
    parallelSlots?: boolean;
    contBatching?: boolean;
    cachePrompt?: boolean;

    // KV cache
    kvCacheSize?: boolean;
    kvCacheUnified?: boolean;
    kvCacheTypeK?: boolean;
    kvCacheTypeV?: boolean;

    // Performance options
    flashAttention?: boolean;
    noMmap?: boolean;
    lockMemory?: boolean;

    // GPU config
    splitMode?: boolean;
    tensorSplit?: boolean;

    // Structured generation
    grammar?: boolean;
    grammarFile?: boolean;

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

    // Other
    contextShift?: boolean;
    directIo?: boolean;
    extraArgs?: boolean;
    logitsAll?: boolean;
    reranking?: boolean;
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

    draftModel?: boolean;
    draftMaxTokens?: boolean;

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
 * Model capabilities response
 */
export interface ModelCapabilitiesResponse {
  modelId: string;
  capabilities: ModelCapabilities;
  success?: boolean;
  error?: string;
}

/**
 * Benchmark parameter type
 */
export type BenchmarkParamType = 'STRING' | 'INTEGER' | 'FLOAT' | 'LOGIC';

/**
 * Benchmark parameter definition
 */
export interface BenchmarkParam {
  fullName: string;        // Full parameter name, e.g. -t
  name: string;            // Display name
  abbreviation: string;    // Abbreviation
  description: string;     // Description
  type: BenchmarkParamType; // Parameter type
  defaultValue: string;    // Default value
  values?: string[];       // Allowed values (enum)
  sort?: number;           // Sort order
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
 * Compute device info
 */
export interface ComputeDevice {
  id: string;              // Device identifier
  name: string;            // Device name
  type: 'CPU' | 'GPU' | 'Accelerator'; // Device type
  selected?: boolean;      // Whether selected
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
 * Benchmark status
 */
export type BenchmarkStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

/**
 * Benchmark task
 */
export interface Benchmark {
  id: string;              // Benchmark ID
  modelId: string;         // Model ID
  modelName: string;       // Model name
  status: BenchmarkStatus; // Status
  command: string;         // Executed command
  config: Record<string, unknown>; // Benchmark config
  createdAt: string;       // Created at
  startedAt?: string;      // Started at
  finishedAt?: string;     // Finished at
  error?: string;          // Error message
  metrics?: {
    total_time_ms?: number;
    tokens_per_second?: number;
    raw_output?: string;
    [key: string]: unknown;
  };
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
 * Benchmark list response
 */
export interface BenchmarkListResponse {
  success: boolean;
  data?: {
    benchmarks: Benchmark[]; // Backend returns benchmarks array
  };
  error?: string;
}

/**
 * Benchmark result detail response
 */
export interface BenchmarkResultResponse {
  success: boolean;
  data?: BenchmarkResult;
  error?: string;
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
 * Save benchmark config request
 */
export interface SaveBenchmarkConfigRequest {
  name: string;            // Config name
  config: BenchmarkConfig;
}

/**
 * Save benchmark config response
 */
export interface SaveBenchmarkConfigResponse {
  success: boolean;
  error?: string;
}

/**
 * Load benchmark config response
 */
export interface LoadBenchmarkConfigResponse {
  success: boolean;
  data?: {
    configs: Array<{
      name: string;
      config: BenchmarkConfig;
      createdAt: string;
    }>;
  };
  error?: string;
}

/**
 * Benchmark list data response
 */
export interface BenchmarkListDataResponse {
  success: boolean;
  data?: {
    benchmarks: Benchmark[];
    total: number;
  };
  error?: string;
}
