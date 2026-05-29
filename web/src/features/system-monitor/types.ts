export interface CpuInfo {
  used: number;
  total: number;
  percent: number;
  model: string;
}

export interface MemoryInfo {
  used: number;
  total: number;
  percent: number;
}

export interface DiskInfo {
  used: number;
  total: number;
  percent: number;
}

export interface GpuInfo {
  index: number;
  name: string;
  vendor: string;
  memoryUsed: number;
  memoryTotal: number;
  temperature: number;
  utilization: number;
  powerUsage: number;
  driverVersion: string;
}

export interface SystemResources {
  cpu: CpuInfo;
  memory: MemoryInfo;
  disk: DiskInfo;
  gpu: GpuInfo[];
  loadAverage: number[];
  uptime: number;
  kernelVersion: string;
  rocmVersion: string;
  platform: string;
  arch: string;
  hostname: string;
  hostIp: string;
}

export interface ModelStats {
  modelId: string;
  modelName: string;
  instanceId: string;
  state: string;
  backendType: string;
  port: number;
  loadedAt: number;
  uptimeSeconds: number;
  requestCount: number;
  errorCount: number;
  totalPromptTokens: number;
  totalCompletionTokens: number;
  avgLatencyMs: number;
  inflightCount: number;
  lastRequestAt: number;
}
