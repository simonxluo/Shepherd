import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type { Model } from '@/types';

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
