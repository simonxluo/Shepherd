import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { benchmarksApi } from '@/lib/api/benchmarks';
import type {
  BenchmarkParam,
  CreateBenchmarkRequest,
} from '@/types';

/**
 * Fetch benchmark params list hook
 */
export function useBenchmarkParams() {
  return useQuery<BenchmarkParam[]>({
    queryKey: ['benchmark', 'params'],
    queryFn: async () => {
      const response = await benchmarksApi.getParams();
      return response.params || [];
    },
    staleTime: 30 * 60 * 1000,
  });
}

/**
 * Fetch Llama.cpp versions list hook
 */
export function useLlamaCppVersions() {
  return useQuery<Array<{ path: string; name?: string; description?: string }>>({
    queryKey: ['llamacpp', 'versions'],
    queryFn: async () => {
      const response = await benchmarksApi.getLlamaCppVersions();
      return response.data?.items || [];
    },
    staleTime: 10 * 60 * 1000,
  });
}

/**
 * Create benchmark task hook
 */
export function useCreateBenchmark() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: CreateBenchmarkRequest) => {
      const response = await benchmarksApi.create(params);
      if (!response.success) {
        throw new Error(response.error || '创建压测任务失败');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmarks'] });
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'results'] });
    },
  });
}


