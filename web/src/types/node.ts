/**
 * 节点角色
 */
export type NodeRole = 'master' | 'client' | 'hybrid';

/**
 * 节点状态 - 统一状态定义，与后端 types.NodeState 保持一致
 */
export type NodeStatus = 'offline' | 'online' | 'busy' | 'error' | 'degraded' | 'disabled';

/**
 * GPU 信息
 */
export interface GPUInfo {
  index: number;
  name: string;
  vendor: string;
  totalMemory: number;
  usedMemory: number;
  temperature: number;
  utilization: number;
  powerUsage: number;
  driverVersion?: string;
}

/**
 * llama.cpp 信息
 */
export interface LlamacppInfo {
  path: string;
  version: string;
  buildType: string;
  gpuBackend?: string;
  supportsGPU: boolean;
  available: boolean;
}

/**
 * 模型信息
 */
export interface NodeModelInfo {
  path: string;
  name: string;
  size: number;
  format: string;
  loaded: boolean;
  loadedAt?: string;
  loadedBy?: string;
}

/**
 * 节点资源
 */
export interface NodeResources {
  cpuUsed: number;
  cpuTotal: number;
  memoryUsed: number;
  memoryTotal: number;
  diskUsed: number;
  diskTotal: number;
  gpuInfo: GPUInfo[];
  networkRx: number;
  networkTx: number;
  uptime: number;
  loadAverage: number[];
  // 兼容性字段（从 ResourceUsage 迁移）
  cpuPercent?: number;
  gpuPercent?: number;
  gpuMemoryUsed?: number;
  gpuMemoryTotal?: number;
  rocmVersion?: string;
  kernelVersion?: string;
}

/**
 * 节点能力
 */
export interface NodeCapabilities {
  gpu: boolean;
  gpuCount: number;
  gpuNames: string[];
  gpuName?: string;
  gpuMemory?: number;
  cpuCount: number;
  memory: number;
  supportsLlama: boolean;
  supportsPython: boolean;
  condaEnvs: string[];
  dockerEnabled: boolean;
}

/**
 * 心跳消息
 */
export interface HeartbeatMessage {
  nodeId: string;
  timestamp: string;
  status: NodeStatus;
  role: NodeRole;
  resources?: NodeResources;
  capabilities?: NodeCapabilities;
  metadata?: Record<string, unknown>;
  sequence: number;
}

/**
 * 命令类型
 */
export type CommandType = 
  | 'load_model' 
  | 'unload_model' 
  | 'run_llamacpp' 
  | 'stop_process' 
  | 'update_config' 
  | 'collect_logs' 
  | 'scan_models';

/**
 * 命令
 */
export interface NodeCommand {
  id: string;
  type: CommandType;
  fromNodeId: string;
  toNodeId?: string;
  payload: Record<string, unknown>;
  createdAt: string;
  timeout?: number;
  priority: number;
  retryCount: number;
  maxRetries: number;
}

/**
 * 命令结果
 */
export interface CommandResult {
  commandId: string;
  fromNodeId: string;
  toNodeId: string;
  success: boolean;
  result?: Record<string, unknown>;
  error?: string;
  completedAt: string;
  duration: number;
  metadata?: Record<string, string>;
}

/**
 * 节点统计信息
 */
export interface NodeStats {
  total: number;
  online: number;
  offline: number;
  busy: number;
}

/**
 * ==================== 统一节点类型（v0.2.0+）====================
 * ==================== Unified Node Types (v0.2.0+) ====================
 */

/**
 * 统一节点信息（匹配后端 types.NodeInfo）
 * 这是前端唯一应该使用的节点类型
 */
export interface UnifiedNode {
  id: string;
  name: string;
  address: string;
  port: number;
  role: NodeRole;
  status: NodeStatus;
  version: string;
  tags: string[];
  capabilities?: NodeCapabilities;
  resources?: NodeResources;
  metadata: Record<string, string>;
  createdAt: string;
  updatedAt: string;
  lastSeen: string;
  registeredAt?: string;
}

// ==================== 类型别名（向后兼容）====================
// ==================== Type Aliases (Backward Compatibility) ====================

/**
 * @deprecated 使用 UnifiedNode 代替
 */
export type DistributedNode = UnifiedNode;

/**
 * @deprecated 使用 UnifiedNode 代替
 */
export type Client = UnifiedNode;

/**
 * @deprecated 使用 NodeCapabilities 代替
 */
export type Capabilities = NodeCapabilities;

/**
 * @deprecated 使用 NodeResources 代替
 */
export interface ResourceUsage extends NodeResources {
  cpuPercent?: number; // 保留兼容性
  gpuPercent?: number;
}

/**
 * 节点列表响应 - 匹配后端 GET /api/master/nodes 响应格式
 */
export interface NodeListResponse {
  nodes: DistributedNode[];
  stats: NodeStats;
}

/**
 * 节点注册请求
 */
export interface NodeRegisterRequest {
  id: string;
  name: string;
  address: string;
  port: number;
  role: NodeRole;
  capabilities: NodeCapabilities;
  metadata?: Record<string, string>;
}

/**
 * 节点注册响应
 */
export interface NodeRegisterResponse {
  message: string;
  node: DistributedNode;
}

/**
 * 发送命令请求
 */
export interface SendCommandRequest {
  type: CommandType;
  payload: Record<string, unknown>;
  timeout?: number;
  priority?: number;
}

/**
 * 发送命令响应
 */
export interface SendCommandResponse {
  message: string;
  command: NodeCommand;
}


/**
 * llama.cpp 路径信息
 */
export interface LlamacppPathInfo {
  path: string;
  exists: boolean;
  version?: string;
  isDefault?: boolean;
}

/**
 * 模型路径信息
 */
export interface ModelPathInfo {
  path: string;
  exists: boolean;
  modelCount?: number;
}

/**
 * 环境信息
 */
export interface EnvironmentInfo {
  os: string;
  architecture: string;
  kernelVersion?: string;
  goVersion: string;
  pythonVersion?: string;
  rocmVersion?: string;
  cudaVersion?: string;
}

/**
 * Conda 配置信息
 */
export interface CondaConfigInfo {
  enabled: boolean;
  defaultEnv?: string;
  availableEnvs: string[];
  condaPath?: string;
}

/**
 * 执行器配置信息
 */
export interface ExecutorConfigInfo {
  pythonPath: string;
  timeout: number;
  maxRetries: number;
}

/**
 * 节点配置信息
 */
export interface NodeConfigInfo {
  llamaCppPaths: LlamacppPathInfo[];
  modelPaths: ModelPathInfo[];
  environment: EnvironmentInfo;
  conda: CondaConfigInfo;
  executor: ExecutorConfigInfo;
  collectedAt: string;
}

/**
 * llama.cpp 测试结果
 */
export interface LlamacppTestResult {
  success: boolean;
  path: string;
  version?: string;
  error?: string;
  output?: string;
  duration: number;
  testedAt: string;
}