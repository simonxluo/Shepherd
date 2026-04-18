import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type { ModelCapabilities } from '@/types';

// ========== GPU & llama.cpp 后端相关 ==========

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
interface SystemGPUListResponse {
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

interface LlamacppBackendListResponse {
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

// ========== 模型能力相关 ==========

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

// ========== 显存估算相关 ==========

/**
 * 显存估算请求参数
 */
interface EstimateVRAMParams {
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
interface EstimateVRAMData {
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
    mutationFn: async (params: EstimateVRAMParams): Promise<EstimateVRAMData> => {
      const response = await apiClient.post<EstimateVRAMData>(
        '/models/vram/estimate',
        params
      );
      return response;
    },
  });
}
