import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type { Model } from '@/types';

/**
 * Scan models hook
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
      queryClient.invalidateQueries({ queryKey: ['models'], refetchType: 'all' });
    },
  });
}
