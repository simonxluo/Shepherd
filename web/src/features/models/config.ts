import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';

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
    config: Record<string, any>;
    createdAt: string;
    updatedAt: string;
  } | null;
}

/**
 * Fetch model load config hook
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
 * Save model load config hook
 */
export function useSaveModelLoadConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, config }: { modelId: string; config: Record<string, any> }) => {
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
 * Delete model load config hook
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
