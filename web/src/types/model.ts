/**
 * 模型元数据 - 完全匹配后端 server.go 返回的字段
 */
export interface ModelMetadata {
  // 基本信息
  name?: string;
  architecture: string;
  quantization?: string;
  type?: string; // "model", "adapter", "projector", "imatrix"
  author?: string | null;
  url?: string | null;
  description?: string | null;
  license?: string | null;

  // 文件类型信息
  fileType?: number;
  fileTypeDescriptor?: string; // 更详细的文件类型描述（如 Q4_K_M, Q5_0_L 等）
  quantizationVersion?: number;

  // 模型参数
  parameters?: number;
  bitsPerWeight?: number;

  // 文件信息
  alignment?: number;
  fileSize?: number;
  modelSize?: number;

  // 模型架构参数（可能为 0）
  contextLength?: number;
  embeddingLength?: number;
  layerCount?: number;
  headCount?: number;

  // 以下字段保留以兼容旧代码（后端不再返回）
  blockSize?: number;
  feedForwardLength?: number;
  attentionHeadCount?: number;
  attentionHeadCountKeyValue?: number;
  ropeDimensionCount?: number;
  ggmlFileType?: string;
  tokenizer?: string;
}

/**
 * 模型信息
 */
export interface Model {
  id: string;
  name: string;
  displayName: string;
  alias?: string;
  path: string;
  pathPrefix: string;
  size: number;
  // 分卷模型相关字段
  totalSize?: number;    // 所有分卷的总大小
  shardCount?: number;   // 分卷数量
  shardFiles?: string[]; // 所有分卷文件路径
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
}

/**
 * 模型状态
 */
export type ModelStatus = 'stopped' | 'loading' | 'running' | 'unloading' | 'error';

/**
 * 处理槽位
 */
export interface Slot {
  id: number;
  isProcessing: boolean;
  isSpeculative?: boolean;
  taskId?: string;
}

/**
 * 加载模型参数
 */
export interface LoadModelParams {
  // 基础参数
  modelId: string;
  nodeId?: string;              // 指定运行节点 ID，undefined 表示自动调度
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

  // 后端配置
  llamaCppPath?: string;      // llama.cpp 可执行文件路径
  mainGpu?: number | string;  // 主GPU选择

  // 能力开关
  capabilities?: {
    thinking?: boolean;    // 思考能力
    tools?: boolean;       // 工具使用
    translation?: boolean; // 直译
    embedding?: boolean;   // 嵌入
  };

  // 上下文与加速
  flashAttention?: boolean;       // Flash Attention 加速
  noMmap?: boolean;               // 禁用内存映射
  lockMemory?: boolean;           // 锁定物理内存

  // 采样参数
  logitsAll?: boolean;            // 输入向量模式
  reranking?: boolean;            // 重排序模式
  minP?: number;                  // Min-P 采样

  // 惩罚参数
  presencePenalty?: number;       // 存在惩罚
  frequencyPenalty?: number;      // 频率惩罚

  // 批处理参数
  uBatchSize?: number;            // 微批大小
  parallelSlots?: number;         // 并发槽位数

  // KV缓存
  kvCacheSize?: number;           // KV缓存内存上限
  kvCacheUnified?: boolean;       // 统一KV缓存区
  kvCacheTypeK?: string;          // KV缓存类型K (f16, f32, q8_0)
  kvCacheTypeV?: string;          // KV缓存类型V

  // 其他参数
  directIo?: string;              // DirectIO 模式
  disableJinja?: boolean;         // 禁用 Jinja 模板
  chatTemplate?: string;          // 内置聊天模板
  contextShift?: boolean;         // 上下文移位
  extraArgs?: string;             // 额外命令行参数

  // 线程配置
  threadsBatch?: number;          // 批处理线程数

  // 扩展采样参数
  repeatLastN?: number;           // 重复惩罚范围
  typicalP?: number;              // 典型采样
  ignoreEos?: boolean;            // 忽略结束token

  // 多GPU配置
  splitMode?: string;             // GPU分割模式 (none, layer, row)
  tensorSplit?: string;           // 张量分割比例

  // 服务器优化
  contBatching?: boolean;         // 连续批处理
  cachePrompt?: boolean;          // 提示缓存

  // 结构化生成
  grammar?: string;               // BNF语法
  grammarFile?: string;           // 语法文件路径

  // LoRA适配器
  lora?: string;                  // LoRA适配器路径
  loraScaled?: string;            // 带缩放的LoRA

  // 聊天模板额外参数
  chatTemplateKwargs?: string;    // 模板额外JSON参数

  // RoPE扩展
  ropeScaling?: string;           // RoPE缩放方法
  ropeScale?: number;             // RoPE缩放因子
  ropeFreqBase?: number;          // RoPE基础频率
  ropeFreqScale?: number;         // RoPE频率缩放

  // 运行时管理
  unloadAfterMinutes?: number;   // 空闲自动卸载时间（分钟）。0=永不卸载，>0=自定义分钟数
  concurrencyLimit?: number;     // 最大并发请求数。0=不限，>0=自定义限制

  // 参数启用状态：标记哪些参数需要手动配置（false表示使用llama-server默认值）
  enabled?: {
    // 基础参数
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

    // 采样参数
    minP?: boolean;
    typicalP?: boolean;
    presencePenalty?: boolean;
    frequencyPenalty?: boolean;
    ignoreEos?: boolean;

    // 批处理和缓存
    uBatchSize?: boolean;
    parallelSlots?: boolean;
    contBatching?: boolean;
    cachePrompt?: boolean;

    // KV缓存
    kvCacheSize?: boolean;
    kvCacheUnified?: boolean;
    kvCacheTypeK?: boolean;
    kvCacheTypeV?: boolean;

    // 性能选项
    flashAttention?: boolean;
    noMmap?: boolean;
    lockMemory?: boolean;

    // GPU配置
    splitMode?: boolean;
    tensorSplit?: boolean;

    // 结构化生成
    grammar?: boolean;
    grammarFile?: boolean;

    // LoRA
    lora?: boolean;
    loraScaled?: boolean;

    // 模板
    chatTemplate?: boolean;
    chatTemplateKwargs?: boolean;
    disableJinja?: boolean;

    // RoPE扩展
    ropeScaling?: boolean;
    ropeScale?: boolean;
    ropeFreqBase?: boolean;
    ropeFreqScale?: boolean;

    // 其他
    contextShift?: boolean;
    directIo?: boolean;
    logitsAll?: boolean;
    reranking?: boolean;
    timeout?: boolean;
    alias?: boolean;

    // 运行时管理
    unloadAfterMinutes?: boolean;
    concurrencyLimit?: boolean;
  };
}

/**
 * 模型列表响应
 */
export interface ModelListResponse {
  models: Model[];
  total: number;
  loaded: number;
}

/**
 * 模型能力配置
 */
export interface ModelCapabilities {
  thinking?: boolean;    // 思考能力（如 DeepSeek-R1 等）
  tools?: boolean;       // 工具使用/函数调用
  rerank?: boolean;      // 重排序能力
  embedding?: boolean;   // 嵌入向量生成
}

/**
 * 模型能力响应
 */
export interface ModelCapabilitiesResponse {
  modelId: string;
  capabilities: ModelCapabilities;
  success?: boolean;
  error?: string;
}

/**
 * 压测参数类型
 */
export type BenchmarkParamType = 'STRING' | 'INTEGER' | 'FLOAT' | 'LOGIC';

/**
 * 压测参数定义
 */
export interface BenchmarkParam {
  fullName: string;        // 完整参数名，如 -t
  name: string;            // 显示名称
  abbreviation: string;    // 缩写
  description: string;     // 描述
  type: BenchmarkParamType; // 参数类型
  defaultValue: string;    // 默认值
  values?: string[];       // 可选值列表（枚举类型）
  sort?: number;           // 排序序号
}

/**
 * 压测参数配置响应
 */
export interface BenchmarkParamsResponse {
  success: boolean;
  params?: BenchmarkParam[];
  error?: string;
}

/**
 * 计算设备信息
 */
export interface ComputeDevice {
  id: string;              // 设备标识
  name: string;            // 设备名称
  type: 'CPU' | 'GPU' | 'Accelerator'; // 设备类型
  selected?: boolean;      // 是否已选择
}

/**
 * Llama.cpp 版本信息
 */
export interface LlamaCppVersion {
  path: string;            // 可执行文件路径
  name?: string;           // 显示名称
  description?: string;    // 描述
}

/**
 * 压测配置
 */
export interface BenchmarkConfig {
  modelId: string;         // 模型 ID
  modelName: string;       // 模型名称
  llamaCppPath: string;    // llama.cpp 路径
  devices?: string[];      // 选择的设备列表（为空表示使用 auto）
  params: Record<string, string | number | boolean>; // 压测参数键值对
  configName?: string;     // 配置名称（用于保存配置）
}

/**
 * 压测状态
 */
export type BenchmarkStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

/**
 * 压测任务
 */
export interface Benchmark {
  id: string;              // 压测 ID
  modelId: string;         // 模型 ID
  modelName: string;       // 模型名称
  status: BenchmarkStatus; // 状态
  command: string;         // 执行命令
  config: Record<string, any>; // 压测配置
  createdAt: string;       // 创建时间
  startedAt?: string;      // 开始时间
  finishedAt?: string;     // 完成时间
  error?: string;          // 错误信息
  metrics?: {
    total_time_ms?: number;
    tokens_per_second?: number;
    raw_output?: string;
    [key: string]: any;
  };
}

/**
 * 压测结果
 */
export interface BenchmarkResult {
  id: string;              // 结果 ID
  benchmarkId: string;     // 关联的压测 ID
  modelId: string;         // 模型 ID
  modelName: string;       // 模型名称
  command: string[];       // 执行的命令
  commandStr: string;      // 命令字符串
  exitCode: number;        // 退出码
  rawOutput: string;       // 原始输出
  fileName: string;        // 保存的文件名
  savedPath: string;       // 保存路径
  timestamp: string;       // 时间戳
  // 解析后的性能指标
  metrics?: {
    tps?: number;          // Tokens per second
    promptTps?: number;    // Prompt processing speed
    totalTokens?: number;  // Total tokens processed
    loadTime?: number;     // Model load time (ms)
    memoryUsage?: number;  // Memory usage (MB)
  };
}

/**
 * 压测结果列表项
 */
export interface BenchmarkResultFile {
  name: string;            // 文件名
  size: number;            // 文件大小
  modified: string;        // 修改时间
}

/**
 * 压测结果列表响应
 */
export interface BenchmarkListResponse {
  success: boolean;
  data?: {
    benchmarks: Benchmark[]; // 后端返回的是 benchmarks 数组
  };
  error?: string;
}

/**
 * 压测结果详情响应
 */
export interface BenchmarkResultResponse {
  success: boolean;
  data?: BenchmarkResult;
  error?: string;
}

/**
 * 创建压测请求
 */
export interface CreateBenchmarkRequest {
  modelId: string;
  llamaBinPath: string;
  cmd?: string;            // 压测命令字符串
  args?: string[];         // 压测命令参数数组
  configName?: string;     // 可选的配置名称
}

/**
 * 创建压测响应
 */
export interface CreateBenchmarkResponse {
  success: boolean;
  data?: BenchmarkResult;
  error?: string;
}

/**
 * 保存压测配置请求
 */
export interface SaveBenchmarkConfigRequest {
  name: string;            // 配置名称
  config: BenchmarkConfig;
}

/**
 * 保存压测配置响应
 */
export interface SaveBenchmarkConfigResponse {
  success: boolean;
  error?: string;
}

/**
 * 加载压测配置响应
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
 * 压测列表响应
 */
export interface BenchmarkListDataResponse {
  success: boolean;
  data?: {
    benchmarks: Benchmark[];
    total: number;
  };
  error?: string;
}
