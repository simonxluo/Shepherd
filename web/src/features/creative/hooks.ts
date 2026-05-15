import { useQuery } from '@tanstack/react-query';
import { apiClient, ApiClient } from '@/lib/api/client';
import type { ModelCapabilities } from '@/types/model';

/**
 * API client for OpenAI-compatible /v1 endpoints (audio, images, etc.)
 */
export const v1Client = new ApiClient('/v1');

export interface LoadedModel {
  id: string;
  name: string;
  alias?: string;
  backendType?: string;
  capabilities?: ModelCapabilities;
}

interface LoadedModelsResponse {
  success: boolean;
  models: LoadedModel[];
}

export const BACKEND_LABELS: Record<string, string> = {
  llamacpp: 'llama.cpp',
  vllm: 'vLLM',
  vllm_omni: 'vLLM-Omni',
};

export function useLoadedModels() {
  return useQuery({
    queryKey: ['models', 'loaded'],
    queryFn: async () => {
      const res = await apiClient.get<LoadedModelsResponse>('/models/loaded');
      return res.models ?? [];
    },
    refetchInterval: 5000,
  });
}
