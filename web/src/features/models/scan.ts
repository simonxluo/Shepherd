import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
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
