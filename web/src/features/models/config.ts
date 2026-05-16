import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type { LoadModelParams } from '@/types/model';

/**
 * Model load config response type
 */
interface ModelLoadConfigResponse {
  exists: boolean;
  config?: {
    id: string;
    nodeId: string;
    modelId: string;
    modelName: string;
    config: Record<string, unknown>;
    createdAt: string;
    updatedAt: string;
  } | null;
}

/**
 * Named model load config entry
 */
export interface ModelLoadConfigEntry {
  name: string;
  config: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

/**
 * Named model load configs response
 */
export interface ModelLoadConfigsResponse {
  modelId: string;
  configs: ModelLoadConfigEntry[];
}

/**
 * Fetch model load config hook (default config)
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
    staleTime: 10 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

/**
 * Save model load config hook (default config)
 */
export function useSaveModelLoadConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, config }: { modelId: string; config: LoadModelParams }) => {
      const response = await apiClient.put<{ success: boolean }>(
        `/models/${modelId}/load-config`,
        { config }
      );
      return response;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['models', variables.modelId, 'load-config']
      });
    },
  });
}

/**
 * Delete model load config hook (default config)
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
      queryClient.invalidateQueries({ queryKey: ['models', modelId, 'load-config'] });
    },
  });
}

/**
 * List all load configs (default + named) for a model
 */
export function useModelLoadConfigs(modelId: string) {
  return useQuery<ModelLoadConfigsResponse>({
    queryKey: ['models', modelId, 'load-configs'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: ModelLoadConfigsResponse }>(
        `/models/${modelId}/load-configs`
      );
      return response.data;
    },
    enabled: !!modelId,
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

/**
 * Save a named load config preset
 */
export function useSaveNamedModelLoadConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, name, config }: { modelId: string; name: string; config: Record<string, unknown> }) => {
      const response = await apiClient.put<{ success: boolean }>(
        `/models/${modelId}/load-configs/${encodeURIComponent(name)}`,
        { config }
      );
      return response;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['models', variables.modelId, 'load-configs']
      });
    },
  });
}

/**
 * Delete a named load config preset
 */
export function useDeleteNamedModelLoadConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, name }: { modelId: string; name: string }) => {
      const response = await apiClient.delete<{ success: boolean }>(
        `/models/${modelId}/load-configs/${encodeURIComponent(name)}`
      );
      return response;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ['models', variables.modelId, 'load-configs']
      });
    },
  });
}
