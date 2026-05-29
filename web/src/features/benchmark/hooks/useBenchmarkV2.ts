import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { benchmarksApi } from '@/lib/api/benchmarks';
import type { BenchmarkV2Record } from '@/types';

/**
 * Create V2 benchmark task mutation
 */
export function useCreateBenchmarkV2() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { modelId: string; promptTokens: number; maxTokens: number }) => {
      const response = await benchmarksApi.createV2(params);
      if (!response.success) {
        throw new Error(response.error || 'Failed to run V2 benchmark');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'v2'] });
    },
  });
}

/**
 * Fetch V2 benchmark history records for a model
 */
export function useBenchmarkV2History(modelId: string | undefined) {
  return useQuery<BenchmarkV2Record[]>({
    queryKey: ['benchmark', 'v2', modelId],
    queryFn: async () => {
      if (!modelId) return [];
      const response = await benchmarksApi.listV2(modelId);
      return response.data?.records || [];
    },
    enabled: !!modelId,
    staleTime: 10 * 1000,
  });
}

/**
 * Delete V2 benchmark record mutation
 */
export function useDeleteBenchmarkV2() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { modelId: string; lineNumber: number }) => {
      const response = await benchmarksApi.deleteV2(params.modelId, params.lineNumber);
      if (!response.success) {
        throw new Error('Failed to delete record');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'v2'] });
    },
  });
}
