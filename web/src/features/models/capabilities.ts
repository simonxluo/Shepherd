import { useQuery, useMutation, useQueryClient, useQueries } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type { ModelCapabilities } from '@/types';

// ========== Model category ==========

export type ModelCategory = 'all' | 'chat' | 'audio' | 'image' | 'embedding';

export const MODEL_CATEGORIES: { key: ModelCategory; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'chat', label: 'Chat' },
  { key: 'audio', label: '音频' },
  { key: 'image', label: '图像' },
  { key: 'embedding', label: '嵌入' },
];

export function getModelCategory(capabilities?: ModelCapabilities | null): ModelCategory {
  if (!capabilities) return 'chat';
  if (capabilities.tts || capabilities.asr) return 'audio';
  if (capabilities.imageGeneration) return 'image';
  if (capabilities.embedding || capabilities.rerank) return 'embedding';
  return 'chat';
}

export function useAllModelCapabilities(modelIds: string[]) {
  return useQueries({
    queries: modelIds.map(id => ({
      queryKey: ['models', 'capabilities', id],
      queryFn: async () => {
        const response = await apiClient.get<{ success: boolean; data: { capabilities: ModelCapabilities } }>('/models/capabilities/get', { modelId: id });
        return response.data.capabilities;
      },
      staleTime: 10 * 60 * 1000,
      enabled: !!id,
    })),
  });
}

// ========== GPU & llama.cpp backend ==========

/**
 * System endpoint GPU info type (Shepherd extended format)
 * Used by /system/gpus endpoint response
 */
export interface SystemGPUInfo {
  id: string;
  name: string;
  totalMemory?: string;
  freeMemory?: string;
  architecture?: string;
  available: boolean;
}

/**
 * System GPU list response
 */
interface SystemGPUListResponse {
  gpus: SystemGPUInfo[];
  devices: string[];
  count: number;
}

/**
 * Fetch system GPU list hook
 * @param llamaCppPath - Optional llama.cpp path for GPU info
 */
export function useGPUs(llamaCppPath?: string) {
  return useQuery<SystemGPUListResponse>({
    queryKey: ['system', 'gpus', llamaCppPath],
    queryFn: async () => {
      const params = llamaCppPath ? `?llamacppPath=${encodeURIComponent(llamaCppPath)}` : '';
      const response = await apiClient.get<{ success: boolean; data: SystemGPUListResponse }>(`/system/gpus${params}`);
      return response.data;
    },
    staleTime: 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

/**
 * llama.cpp backend info type
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
 * Fetch available llama.cpp backends hook
 */
export function useLlamacppBackends() {
  return useQuery({
    queryKey: ['system', 'llamacpp-backends'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: LlamacppBackendListResponse }>('/system/llamacpp-backends');
      return response.data.backends;
    },
    staleTime: 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

// ========== Model capabilities ==========

/**
 * Model capabilities config hook
 */
export function useModelCapabilities(modelId: string) {
  return useQuery({
    queryKey: ['models', 'capabilities', modelId],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: { capabilities: ModelCapabilities } }>('/models/capabilities/get', { modelId });
      return response.data.capabilities;
    },
    enabled: !!modelId,
    staleTime: 10 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

/**
 * Set model capabilities hook
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
      queryClient.invalidateQueries({
        queryKey: ['models', 'capabilities', variables.modelId]
      });
    },
  });
}

/**
 * Auto-detect model capabilities hook
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
      queryClient.invalidateQueries({
        queryKey: ['models', 'capabilities', modelId]
      });
    },
  });
}

// ========== VRAM estimation ==========

/**
 * VRAM estimation request params
 */
interface EstimateVRAMParams {
  modelId: string;
  llamaBinPath?: string;
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
 * VRAM estimation response (unwrapped from API envelope)
 */
interface EstimateVRAMData {
  vramGB?: string;
  error?: string;
}

/**
 * Estimate VRAM hook
 */
export function useEstimateVRAM() {
  return useMutation({
    mutationFn: async (params: EstimateVRAMParams): Promise<EstimateVRAMData> => {
      const response = await apiClient.post<{
        success: boolean;
        data?: {
          estimatedVRAMGB?: string;
          modelSizeGB?: string;
          kvCacheGB?: string;
        };
        error?: { message?: string };
      }>('/models/vram/estimate', params);

      if (!response.success) {
        return { error: response.error?.message || '估算失败' };
      }

      return { vramGB: response.data?.estimatedVRAMGB };
    },
  });
}
