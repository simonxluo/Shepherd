import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { downloadsApi } from '@/lib/api/downloads';
import type {
  DownloadTask,
  DownloadListResponse,
  CreateDownloadParams,
  DownloadState,
} from '@/types';
import type { ModelFileInfo } from '@/lib/api/downloads';

/**
 * 下载任务列表 Hook
 */
export function useDownloads() {
  return useQuery({
    queryKey: ['downloads'],
    queryFn: async () => {
      const response = await downloadsApi.list();
      return response.data?.downloads || [];
    },
    staleTime: 5 * 1000, // 5 秒
    // ✅ 动态调整刷新频率: 只在有活跃任务时轮询
    refetchInterval: (query) => {
      const data = query.state.data as DownloadTask[] | undefined;
      if (!data || data.length === 0) return false; // 无任务时不刷新
      const activeStates: DownloadState[] = ['preparing', 'downloading', 'merging', 'verifying'];
      const hasActiveTasks = data.some(task => activeStates.includes(task.state));
      return hasActiveTasks ? 1000 : false; // 活跃任务 1 秒刷新,否则不刷新
    },
  });
}

/**
 * 单个下载任务 Hook
 */
export function useDownload(taskId: string) {
  return useQuery({
    queryKey: ['downloads', taskId],
    queryFn: async () => {
      const response = await downloadsApi.get(taskId);
      return response.data;
    },
    enabled: !!taskId,
    refetchInterval: (query) => {
      const data = query.state.data as DownloadTask | undefined;
      const activeStates: DownloadState[] = ['preparing', 'downloading', 'merging', 'verifying'];
      return data && activeStates.includes(data.state) ? 1000 : false;
    },
  });
}

/**
 * 创建下载任务 Hook
 */
export function useCreateDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: CreateDownloadParams) => {
      const response = await downloadsApi.create(params);
      if (!response.success) {
        throw new Error(response.error || '创建下载失败');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
    },
  });
}

/**
 * 暂停下载 Hook
 */
export function usePauseDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (taskId: string) => {
      return await downloadsApi.pause(taskId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
    },
  });
}

/**
 * 恢复下载 Hook
 */
export function useResumeDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (taskId: string) => {
      return await downloadsApi.resume(taskId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
    },
  });
}

/**
 * 取消下载 Hook
 */
export function useCancelDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (taskId: string) => {
      return await downloadsApi.cancel(taskId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
    },
  });
}

/**
 * 重试下载 Hook
 */
export function useRetryDownload() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (taskId: string) => {
      return await downloadsApi.retry(taskId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
    },
  });
}

/**
 * 清理已完成下载 Hook
 */
export function useClearCompletedDownloads() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      return await downloadsApi.clearCompleted();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
    },
  });
}

/**
 * 过滤下载任务 Hook
 */
export function useFilteredDownloads(
  downloads: DownloadTask[] | undefined,
  filters: {
    search?: string;
    state?: DownloadState;
    source?: 'huggingface' | 'modelscope';
  }
) {
  if (!downloads) return [];

  return downloads.filter((task) => {
    // 搜索过滤
    if (filters.search) {
      const search = filters.search.toLowerCase();
      const matchRepo = task.repoId.toLowerCase().includes(search);
      const matchFile = task.fileName.toLowerCase().includes(search);
      if (!matchRepo && !matchFile) return false;
    }

    // 状态过滤
    if (filters.state && task.state !== filters.state) return false;

    // 来源过滤
    if (filters.source && task.source !== filters.source) return false;

    return true;
  });
}

/**
 * 下载统计 Hook
 */
export function useDownloadStats(downloads: DownloadTask[] | undefined) {
  if (!downloads) {
    return {
      total: 0,
      active: 0,
      completed: 0,
      failed: 0,
      totalBytes: 0,
      downloadedBytes: 0,
    };
  }

  const total = downloads.length;
  const active = downloads.filter((d) => ['preparing', 'downloading', 'merging', 'verifying'].includes(d.state)).length;
  const completed = downloads.filter((d) => d.state === 'completed').length;
  const failed = downloads.filter((d) => d.state === 'failed').length;
  const totalBytes = downloads.reduce((sum, d) => sum + d.totalBytes, 0);
  const downloadedBytes = downloads.reduce((sum, d) => sum + d.downloadedBytes, 0);

  return {
    total,
    active,
    completed,
    failed,
    totalBytes,
    downloadedBytes,
  };
}

/**
 * 获取模型文件列表 Hook
 */
export function useModelFiles(source: 'huggingface' | 'modelscope', repoId: string) {
  return useQuery({
    queryKey: ['model-files', source, repoId],
    queryFn: async ({ signal }) => {
      const response = await downloadsApi.listModelFiles(source, repoId, signal);
      if (!response.success) {
        throw new Error(response.error || '获取文件列表失败');
      }
      return response.data;
    },
    enabled: !!source && !!repoId && repoId.length > 3,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

/**
 * 搜索 HuggingFace 模型 Hook
 * @param query 搜索关键词
 * @param limit 返回数量限制
 * @param format 格式过滤器 (gguf, safetensors, onnx, bin, all)
 */
export function useHuggingFaceSearch(query: string, limit?: number, format?: string) {
  return useQuery({
    queryKey: ['huggingface-search', query, limit, format],
    queryFn: async ({ signal }) => {
      const response = await downloadsApi.searchHuggingFace(query, limit, format, signal);
      if (!response.success) {
        throw new Error(response.error || '搜索模型失败');
      }
      return response.data;
    },
    enabled: !!query && query.length >= 2,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

interface ModelRepoConfig {
  endpoint: string;
  token: string;
  timeout: number;
}

/**
 * 获取模型仓库配置 Hook
 */
export function useModelRepoConfig() {
  return useQuery({
    queryKey: ['model-repo-config'],
    queryFn: async () => {
      const response = await downloadsApi.getModelRepoConfig();
      if (!response.success) {
        throw new Error(response.error || '获取配置失败');
      }
      return response.data;
    },
    staleTime: 60 * 1000,
  });
}

/**
 * 获取可用端点列表 Hook
 */
export function useAvailableEndpoints() {
  return useQuery({
    queryKey: ['model-repo-endpoints'],
    queryFn: async () => {
      const response = await downloadsApi.getAvailableEndpoints();
      if (!response.success) {
        throw new Error(response.error || '获取端点列表失败');
      }
      return response.data;
    },
    staleTime: 24 * 60 * 60 * 1000,
  });
}

/**
 * 更新模型仓库配置 Hook
 */
export function useUpdateModelRepoConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (config: Partial<ModelRepoConfig>) => {
      const response = await downloadsApi.updateModelRepoConfig(config);
      if (!response.success) {
        throw new Error(response.error || '更新配置失败');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['model-repo-config'] });
    },
  });
}
