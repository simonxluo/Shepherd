import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { benchmarksApi } from '@/lib/api/benchmarks';
import type {
  Benchmark,
  BenchmarkParam,
  BenchmarkConfig,
  CreateBenchmarkRequest,
  SaveBenchmarkConfigRequest,
} from '@/types';

/**
 * 获取压测参数列表 Hook
 */
export function useBenchmarkParams() {
  return useQuery<BenchmarkParam[]>({
    queryKey: ['benchmark', 'params'],
    queryFn: async () => {
      const response = await benchmarksApi.getParams();
      return response.params || [];
    },
    staleTime: 30 * 60 * 1000, // 30 分钟缓存
  });
}

/**
 * 获取计算设备列表 Hook
 */
export function useBenchmarkDevices(llamaBinPath: string) {
  return useQuery<string[]>({
    queryKey: ['benchmark', 'devices', llamaBinPath],
    queryFn: async () => {
      const response = await benchmarksApi.getDevices(llamaBinPath);
      return response.data?.devices || [];
    },
    enabled: !!llamaBinPath,
    staleTime: 5 * 60 * 1000, // 5 分钟缓存
  });
}

/**
 * 获取 Llama.cpp 版本列表 Hook
 */
export function useLlamaCppVersions() {
  return useQuery<Array<{ path: string; name?: string; description?: string }>>({
    queryKey: ['llamacpp', 'versions'],
    queryFn: async () => {
      const response = await benchmarksApi.getLlamaCppVersions();
      return response.data?.items || [];
    },
    staleTime: 10 * 60 * 1000, // 10 分钟缓存
  });
}

/**
 * 获取压测任务列表 Hook
 */
export function useBenchmarks(modelId?: string) {
  return useQuery<Benchmark[]>({
    queryKey: ['benchmarks', modelId],
    queryFn: async () => {
      const response = await benchmarksApi.list();
      return response.data?.benchmarks || [];
    },
    enabled: !modelId, // 如果指定了 modelId，使用 useBenchmarkResults
    refetchInterval: (query) => {
      // 有运行中的任务时，每 2 秒刷新
      const data = query.state.data;
      const hasRunning = data?.some(b => b.status === 'running');
      return hasRunning ? 2000 : false;
    },
  });
}

/**
 * 单个压测任务 Hook
 */
export function useBenchmark(benchmarkId: string) {
  return useQuery<Benchmark | undefined>({
    queryKey: ['benchmarks', benchmarkId],
    queryFn: async () => {
      const response = await benchmarksApi.get(benchmarkId);
      return response.data;
    },
    enabled: !!benchmarkId,
    refetchInterval: (query) => {
      // 运行中的任务每 2 秒刷新
      const data = query.state.data;
      return data?.status === 'running' ? 2000 : false;
    },
  });
}

/**
 * 创建压测任务 Hook
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

/**
 * 取消压测任务 Hook
 */
export function useCancelBenchmark() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (benchmarkId: string) => {
      const response = await benchmarksApi.cancel(benchmarkId);
      if (!response.success) {
        throw new Error(response.error || '取消压测任务失败');
      }
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmarks'] });
    },
  });
}

/**
 * 获取压测结果列表 Hook
 */
export function useBenchmarkResults(modelId: string) {
  return useQuery<Benchmark[]>({
    queryKey: ['benchmark', 'results', modelId],
    queryFn: async () => {
      const response = await benchmarksApi.list(modelId);
      return response.data?.benchmarks || [];
    },
    enabled: !!modelId,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      // Refresh if any task is running
      const data = query.state.data;
      const hasRunning = data?.some(b => b.status === 'running');
      return hasRunning ? 2000 : false;
    },
  });
}


/**
 * 保存压测配置 Hook
 */
export function useSaveBenchmarkConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: SaveBenchmarkConfigRequest) => {
      const response = await benchmarksApi.saveConfig(params);
      if (!response.success) {
        throw new Error(response.error || '保存压测配置失败');
      }
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'configs'] });
    },
  });
}

/**
 * 获取压测配置列表 Hook
 */
export function useBenchmarkConfigs() {
  return useQuery<Array<{ name: string; config: BenchmarkConfig; createdAt: string }>>({
    queryKey: ['benchmark', 'configs'],
    queryFn: async () => {
      const response = await benchmarksApi.listConfigs();
      return response.data?.configs || [];
    },
    staleTime: 10 * 60 * 1000, // 10 分钟缓存
  });
}

/**
 * 获取单个压测配置 Hook
 */
export function useBenchmarkConfig(name: string) {
  return useQuery<BenchmarkConfig | undefined>({
    queryKey: ['benchmark', 'configs', name],
    queryFn: async () => {
      const response = await benchmarksApi.getConfig(name);
      return response.data;
    },
    enabled: !!name,
  });
}

/**
 * 删除压测配置 Hook
 */
export function useDeleteBenchmarkConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (name: string) => {
      const response = await benchmarksApi.deleteConfig(name);
      if (!response.success) {
        throw new Error(response.error || '删除压测配置失败');
      }
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'configs'] });
    },
  });
}
