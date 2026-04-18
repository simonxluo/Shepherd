import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';

/**
 * 模型加载配置响应类型
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
