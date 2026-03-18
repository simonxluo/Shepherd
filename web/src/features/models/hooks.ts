import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { apiClient } from '@/lib/api/client';
import type {
  Model,
  ModelListResponse,
  LoadModelParams,
  ModelStatus,
  ModelCapabilities,
} from '@/types';

/**
 * 模型列表 Hook
 */
export function useModels() {
  return useQuery<Model[]>({
    queryKey: ['models'],
    queryFn: async (): Promise<Model[]> => {
      const response = await apiClient.get<{ success: boolean; data: ModelListResponse }>('/models');
      const data = response.data;
      return data.models;
    },
    staleTime: 2000, // 2秒后数据视为过期
    refetchOnWindowFocus: true, // 窗口获得焦点时刷新
    refetchInterval: (query) => {
      // 如果有模型正在加载或卸载，每2秒刷新一次
      const data = query.state.data;
      const hasLoadingOrUnloading = data?.some(m => m.status === 'loading' || m.status === 'unloading');
      return hasLoadingOrUnloading ? 2000 : false;
    },
  });
}

/**
 * 单个模型 Hook
 */
export function useModel(modelId: string) {
  return useQuery({
    queryKey: ['models', modelId],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: { model: Model } }>(`/models/${modelId}`);
      return response.data.model;
    },
    enabled: !!modelId,
  });
}

/**
 * 加载模型 Hook
 */
export function useLoadModel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: Partial<LoadModelParams>) => {
      const response = await apiClient.post<{ success: boolean }>(
        `/models/${params.modelId}/load`,
        params
      );
      return response;
    },
    onSuccess: async () => {
      // 使模型列表查询失效并强制重新获取
      await queryClient.invalidateQueries({ queryKey: ['models'], refetchType: 'all' });
    },
  });
}

/**
 * 卸载模型 Hook
 */
export function useUnloadModel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (modelId: string) => {
      const response = await apiClient.post<{ success: boolean }>(
        `/models/${modelId}/unload`
      );
      return response;
    },
    onSuccess: async () => {
      // 使模型列表查询失效并强制重新获取
      await queryClient.invalidateQueries({ queryKey: ['models'], refetchType: 'all' });
    },
  });
}

/**
 * 更新模型别名 Hook
 */
export function useUpdateModelAlias() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, alias }: { modelId: string; alias: string }) => {
      const response = await apiClient.put<{ success: boolean }>(
        `/models/${modelId}/alias`,
        { alias }
      );
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['models'] });
    },
  });
}

/**
 * 设置模型收藏 Hook
 */
export function useSetModelFavourite() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, favourite }: { modelId: string; favourite: boolean }) => {
      const response = await apiClient.put<{ success: boolean }>(
        `/models/${modelId}/favourite`,
        { favourite }
      );
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['models'] });
    },
  });
}

/**
 * 扫描模型 Hook
 */
export function useScanModels() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const response = await apiClient.post<{
        success: boolean;
        data: {
          message: string;
          models_found: number;
          errors: number;
          duration_ms: number;
          models: Model[];
          scan_errors: Array<{ path: string; error: string }>;
        };
      }>('/model/scan');
      return response.data;
    },
    onSuccess: () => {
      // 扫描完成后强制刷新模型列表，清除缓存
      queryClient.invalidateQueries({ queryKey: ['models'], refetchType: 'all' });
    },
  });
}

/**
 * 扫描状态 Hook
 */
export function useScanStatus() {
  return useQuery({
    queryKey: ['scan', 'status'],
    queryFn: async () => {
      const response = await apiClient.get<{
        success: boolean;
        data: {
          scanning: boolean;
          progress?: number;
          currentPath?: string;
        };
      }>('/model/scan/status');
      return response.data;
    },
    refetchInterval: (query) => {
      // 如果正在扫描，每秒刷新一次；否则不刷新
      const data = query.state.data;
      return data?.scanning ? 1000 : false;
    },
  });
}

/**
 * 过滤模型 Hook（包含排序）
 * 使用 useMemo 确保排序稳定且只依赖项变化时重新计算
 */
export function useFilteredModels(
  models: Model[] | undefined,
  filters: {
    search?: string;
    status?: ModelStatus;
    favourite?: boolean;
  }
): Model[] {
  return useMemo(() => {
    if (!models) return [];

    // 过滤模型
    const filtered = models.filter((model) => {
      // 搜索过滤
      if (filters.search) {
        const search = filters.search.toLowerCase();
        const matchName = model.name ? model.name.toLowerCase().includes(search) : false;
        const matchAlias = model.alias ? model.alias.toLowerCase().includes(search) : false;
        const matchArch = model.metadata.architecture ? model.metadata.architecture.toLowerCase().includes(search) : false;
        if (!matchName && !matchAlias && !matchArch) return false;
      }

      // 状态过滤
      if (filters.status && model.status !== filters.status) return false;

      // 收藏过滤
      if (filters.favourite && !model.favourite) return false;

      return true;
    });

    // 排序模型：稳定的排序，确保每次刷新后顺序一致
    // 排序优先级：架构 > 名称（字母）> 扫描时间 > 路径
    return [...filtered].sort((a: Model, b: Model) => {
      // 优先按架构排序
      const aArch = (a.metadata?.architecture || '').toLowerCase();
      const bArch = (b.metadata?.architecture || '').toLowerCase();
      const archCompare = aArch.localeCompare(bArch);
      if (archCompare !== 0) return archCompare;

      // 架构相同时，按显示名称（别名或模型名）排序
      const aName = (a.alias || a.displayName || a.name).toLowerCase();
      const bName = (b.alias || b.displayName || b.name).toLowerCase();

      const nameCompare = aName.localeCompare(bName, 'zh-CN');
      if (nameCompare !== 0) return nameCompare;

      // 名称相同时，按扫描时间降序排序（最新的在前）
      const aTime = new Date(a.scannedAt).getTime();
      const bTime = new Date(b.scannedAt).getTime();
      if (aTime !== bTime) return bTime - aTime;

      // 扫描时间也相同时，按路径排序
      return a.path.localeCompare(b.path);
    });
  }, [models, filters.search, filters.status, filters.favourite]);
}

/**
 * 系统端点 GPU 信息类型（Shepherd 扩展格式）
 * 用于 /system/gpus 端点返回的 GPU 信息
 */
export interface SystemGPUInfo {
  id: string;          // 设备 ID，如 "ROCm0"
  name: string;        // GPU 名称
  totalMemory?: string; // 总内存，如 "122880 MiB"
  freeMemory?: string;  // 可用内存，如 "115050 MiB"
  architecture?: string; // 架构信息
  available: boolean;  // 是否可用
}

/**
 * 系统 GPU 列表响应
 */
export interface SystemGPUListResponse {
  gpus: SystemGPUInfo[];      // 详细 GPU 信息（Shepherd 扩展）
  devices: string[];    // 简单设备字符串列表（兼容 LlamacppServer 格式）
  count: number;
}

/**
 * 获取系统 GPU 列表 Hook
 * @param llamaCppPath - 可选的 llama.cpp 路径，用于获取该路径下的 GPU 信息
 */
export function useGPUs(llamaCppPath?: string) {
  return useQuery<SystemGPUListResponse>({
    queryKey: ['system', 'gpus', llamaCppPath],
    queryFn: async () => {
      const params = llamaCppPath ? `?llamacppPath=${encodeURIComponent(llamaCppPath)}` : '';
      const response = await apiClient.get<{ success: boolean; data: SystemGPUListResponse }>(`/system/gpus${params}`);
      return response.data;
    },
    staleTime: 60 * 1000, // GPU 信息缓存 1 分钟
    refetchOnWindowFocus: false,
  });
}

/**
 * llama.cpp 后端信息类型
 */
export interface LlamacppBackend {
  path: string;
  name: string;
  description: string;
  available: boolean;
}

export interface LlamacppBackendListResponse {
  backends: LlamacppBackend[];
  count: number;
}

/**
 * 获取可用的 llama.cpp 后端列表 Hook
 */
export function useLlamacppBackends() {
  return useQuery({
    queryKey: ['system', 'llamacpp-backends'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: LlamacppBackendListResponse }>('/system/llamacpp-backends');
      return response.data.backends;
    },
    staleTime: 60 * 1000, // 后端列表缓存 1 分钟
    refetchOnWindowFocus: false,
  });
}

/**
 * 模型能力配置 Hook
 */
export function useModelCapabilities(modelId: string) {
  return useQuery({
    queryKey: ['models', 'capabilities', modelId],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: { capabilities: ModelCapabilities } }>('/models/capabilities/get', { modelId });
      return response.data.capabilities;
    },
    enabled: !!modelId,
    staleTime: 10 * 60 * 1000, // 能力配置缓存 10 分钟
    refetchOnWindowFocus: false,
  });
}

/**
 * 设置模型能力 Hook
 */
export function useSetModelCapabilities() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, capabilities }: {
      modelId: string;
      capabilities: ModelCapabilities;
    }) => {
      const response = await apiClient.post<{ success: boolean; message?: string }>(
        '/models/capabilities/set',
        { modelId, capabilities }
      );
      return response;
    },
    onSuccess: (data, variables) => {
      // 使能力查询失效
      queryClient.invalidateQueries({
        queryKey: ['models', 'capabilities', variables.modelId]
      });
    },
  });
}

/**
 * 自动检测模型能力 Hook
 */
export function useAutoDetectCapabilities() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (modelId: string) => {
      const response = await apiClient.get<{
        success: boolean;
        data: { modelId: string; capabilities: ModelCapabilities };
      }>(`/models/capabilities/auto-detect?modelId=${modelId}`);
      return response.data;
    },
    onSuccess: (data, modelId) => {
      // 使能力查询失效，触发 UI 刷新
      queryClient.invalidateQueries({
        queryKey: ['models', 'capabilities', modelId]
      });
    },
  });
}

/**
 * 显存估算请求参数
 */
export interface EstimateVRAMParams {
  modelId: string;
  llamaBinPath: string;
  ctxSize?: number;
  batchSize?: number;
  uBatchSize?: number;
  parallel?: number;
  flashAttention?: boolean;
  kvUnified?: boolean;
  cacheTypeK?: string;
  cacheTypeV?: string;
  extraParams?: string;
}

/**
 * 显存估算响应数据
 */
export interface EstimateVRAMData {
  success: boolean;
  vram?: string;      // "60565"
  vramMB?: number;    // 60565
  vramGB?: string;    // "59.15"
  error?: string;
  details?: string;
}

/**
 * 估算显存 Hook
 */
export function useEstimateVRAM() {
  return useMutation({
    mutationFn: async (params: EstimateVRAMParams) => {
      const response = await apiClient.post<{ success: boolean; data: EstimateVRAMData }>(
        '/models/vram/estimate',
        params
      );
      return response.data;
    },
  });
}

// ========== 压测相关 Hooks ==========

import { benchmarksApi } from '@/lib/api/benchmarks';
import type {
  Benchmark,
  BenchmarkConfig,
  BenchmarkResult,
  BenchmarkResultFile,
  BenchmarkParam,
  CreateBenchmarkRequest,
  SaveBenchmarkConfigRequest,
} from '@/types';

/**
 * 获取压测参数列表 Hook
 */
export function useBenchmarkParams() {
  return useQuery<BenchmarkParam[]>({
    queryKey: ['benchmark', 'params'],
    queryFn: async () => {
      const response = await benchmarksApi.getParams();
      return response.params || [];
    },
    staleTime: 30 * 60 * 1000, // 30 分钟缓存
  });
}

/**
 * 获取计算设备列表 Hook
 */
export function useBenchmarkDevices(llamaBinPath: string) {
  return useQuery<string[]>({
    queryKey: ['benchmark', 'devices', llamaBinPath],
    queryFn: async () => {
      const response = await benchmarksApi.getDevices(llamaBinPath);
      return response.data?.devices || [];
    },
    enabled: !!llamaBinPath,
    staleTime: 5 * 60 * 1000, // 5 分钟缓存
  });
}

/**
 * 获取 Llama.cpp 版本列表 Hook
 */
export function useLlamaCppVersions() {
  return useQuery<Array<{ path: string; name?: string; description?: string }>>({
    queryKey: ['llamacpp', 'versions'],
    queryFn: async () => {
      const response = await benchmarksApi.getLlamaCppVersions();
      return response.data?.items || [];
    },
    staleTime: 10 * 60 * 1000, // 10 分钟缓存
  });
}

/**
 * 获取压测任务列表 Hook
 */
export function useBenchmarks(modelId?: string) {
  return useQuery<Benchmark[]>({
    queryKey: ['benchmarks', modelId],
    queryFn: async () => {
      const response = await benchmarksApi.list();
      return response.data?.benchmarks || [];
    },
    enabled: !modelId, // 如果指定了 modelId，使用 useBenchmarkResults
    refetchInterval: (query) => {
      // 有运行中的任务时，每 2 秒刷新
      const data = query.state.data;
      const hasRunning = data?.some(b => b.status === 'running');
      return hasRunning ? 2000 : false;
    },
  });
}

/**
 * 单个压测任务 Hook
 */
export function useBenchmark(benchmarkId: string) {
  return useQuery<Benchmark | undefined>({
    queryKey: ['benchmarks', benchmarkId],
    queryFn: async () => {
      const response = await benchmarksApi.get(benchmarkId);
      return response.data;
    },
    enabled: !!benchmarkId,
    refetchInterval: (query) => {
      // 运行中的任务每 2 秒刷新
      const data = query.state.data;
      return data?.status === 'running' ? 2000 : false;
    },
  });
}

/**
 * 创建压测任务 Hook
 */
export function useCreateBenchmark() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: CreateBenchmarkRequest) => {
      const response = await benchmarksApi.create(params);
      if (!response.success) {
        throw new Error(response.error || '创建压测任务失败');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmarks'] });
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'results'] });
    },
  });
}

/**
 * 取消压测任务 Hook
 */
export function useCancelBenchmark() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (benchmarkId: string) => {
      const response = await benchmarksApi.cancel(benchmarkId);
      if (!response.success) {
        throw new Error(response.error || '取消压测任务失败');
      }
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmarks'] });
    },
  });
}

/**
 * 获取压测结果列表 Hook
 */
export function useBenchmarkResults(modelId: string) {
  return useQuery<Benchmark[]>({
    queryKey: ['benchmark', 'results', modelId],
    queryFn: async () => {
      const response = await benchmarksApi.list(modelId);
      return response.data?.benchmarks || [];
    },
    enabled: !!modelId,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      // Refresh if any task is running
      const data = query.state.data;
      const hasRunning = data?.some(b => b.status === 'running');
      return hasRunning ? 2000 : false;
    },
  });
}

// Deleted useBenchmarkResult and useDeleteBenchmarkResult


/**
 * 保存压测配置 Hook
 */
export function useSaveBenchmarkConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: SaveBenchmarkConfigRequest) => {
      const response = await benchmarksApi.saveConfig(params);
      if (!response.success) {
        throw new Error(response.error || '保存压测配置失败');
      }
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'configs'] });
    },
  });
}

/**
 * 获取压测配置列表 Hook
 */
export function useBenchmarkConfigs() {
  return useQuery<Array<{ name: string; config: BenchmarkConfig; createdAt: string }>>({
    queryKey: ['benchmark', 'configs'],
    queryFn: async () => {
      const response = await benchmarksApi.listConfigs();
      return response.data?.configs || [];
    },
    staleTime: 10 * 60 * 1000, // 10 分钟缓存
  });
}

/**
 * 获取单个压测配置 Hook
 */
export function useBenchmarkConfig(name: string) {
  return useQuery<BenchmarkConfig | undefined>({
    queryKey: ['benchmark', 'configs', name],
    queryFn: async () => {
      const response = await benchmarksApi.getConfig(name);
      return response.data;
    },
    enabled: !!name,
  });
}

/**
 * 删除压测配置 Hook
 */
export function useDeleteBenchmarkConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (name: string) => {
      const response = await benchmarksApi.deleteConfig(name);
      if (!response.success) {
        throw new Error(response.error || '删除压测配置失败');
      }
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'configs'] });
    },
  });
}

// ========== 模型加载配置 Hooks ==========

/**
 * 模型加载配置响应类型
 */
export interface ModelLoadConfigResponse {
  exists: boolean;
  config?: {
    id: string;
    nodeId: string;
    modelId: string;
    modelName: string;
    config: Record<string, any>;
    createdAt: string;
    updatedAt: string;
  } | null;
}

/**
 * 获取模型加载配置 Hook
 */
export function useModelLoadConfig(modelId: string) {
  return useQuery<ModelLoadConfigResponse>({
    queryKey: ['models', modelId, 'load-config'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: ModelLoadConfigResponse }>(
        `/models/${modelId}/load-config`
      );
      return response.data;
    },
    enabled: !!modelId,
    staleTime: 10 * 60 * 1000, // 10 分钟缓存
    refetchOnWindowFocus: false,
  });
}

/**
 * 保存模型加载配置 Hook
 */
export function useSaveModelLoadConfig() {
  return useMutation({
    mutationFn: async ({ modelId, config }: { modelId: string; config: Record<string, any> }) => {
      const response = await apiClient.put<{ success: boolean }>(
        `/models/${modelId}/load-config`,
        { config }
      );
      return response;
    },
  });
}

/**
 * 删除模型加载配置 Hook
 */
export function useDeleteModelLoadConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (modelId: string) => {
      const response = await apiClient.delete<{ success: boolean }>(
        `/models/${modelId}/load-config`
      );
      return response;
    },
    onSuccess: (_, modelId) => {
      // 使配置查询失效
      queryClient.invalidateQueries({ queryKey: ['models', modelId, 'load-config'] });
    },
  });
}
