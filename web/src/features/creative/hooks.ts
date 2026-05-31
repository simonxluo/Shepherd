import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type { ModelCapabilities } from '@/types/model';
import { useModels, useAllModelCapabilities } from '@/features/models';
import type { Model } from '@/types/model';

export interface LoadedModel {
  id: string;
  name: string;
  alias?: string;
  backendType?: string;
  capabilities?: ModelCapabilities;
}

interface LoadedModelsResponse {
  success: boolean;
  data: {
    models: LoadedModel[];
    total: number;
  };
  metadata?: {
    timestamp: string;
    requestId: string;
  };
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
      return res.data?.models ?? [];
    },
    refetchInterval: 5000,
  });
}

export function useAvailableModels(capability: keyof ModelCapabilities) {
  const { data: allModels = [] } = useModels();
  const modelIds = useMemo(() => allModels.map((m: Model) => m.id), [allModels]);
  const capsResults = useAllModelCapabilities(modelIds);

  return useMemo(() => {
    return allModels.filter((m: Model, i: number) => {
      if (m.isLoaded || m.status === 'loading') return false;
      const caps = capsResults[i]?.data;
      return caps?.[capability] === true;
    });
  }, [allModels, capsResults, capability]);
}
