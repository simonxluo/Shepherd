/**
 * Node role
 */
export type NodeRole = 'master' | 'client' | 'hybrid';

/**
 * Node status — unified definition matching backend types.NodeState
 */
export type NodeStatus = 'offline' | 'online' | 'busy' | 'error' | 'degraded' | 'disabled';

/**
 * GPU info
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
 * Node resources
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
  // Compatibility fields (migrated from ResourceUsage)
  cpuPercent?: number;
  gpuPercent?: number;
  gpuMemoryUsed?: number;
  gpuMemoryTotal?: number;
  rocmVersion?: string;
  kernelVersion?: string;
}

/**
 * Node capabilities
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
 * Unified Node Types (v0.2.0+)
 */

/**
 * Unified node info (matches backend types.NodeInfo).
 * This is the only node type the frontend should use.
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

/**
 * Type Aliases (Backward Compatibility)
 */

/**
 * @deprecated Use UnifiedNode instead
 */
export type Client = UnifiedNode;

/**
 * llama.cpp path info
 */
export interface LlamacppPathInfo {
  path: string;
  exists: boolean;
  version?: string;
  isDefault?: boolean;
}

/**
 * Model path info
 */
export interface ModelPathInfo {
  path: string;
  exists: boolean;
  modelCount?: number;
}

/**
 * Environment info
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
 * Conda configuration info
 */
export interface CondaConfigInfo {
  enabled: boolean;
  defaultEnv?: string;
  availableEnvs: string[];
  condaPath?: string;
}

/**
 * Executor configuration info
 */
export interface ExecutorConfigInfo {
  pythonPath: string;
  timeout: number;
  maxRetries: number;
}

/**
 * Node configuration info
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
 * llama.cpp test result
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
