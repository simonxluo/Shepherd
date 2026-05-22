import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { downloadsApi } from '@/lib/api/downloads';
import type {
  DownloadTask,
  CreateDownloadParams,
  DownloadState,
} from '@/types';

/**
 * Active download states that indicate ongoing work
 */
export const ACTIVE_DOWNLOAD_STATES: DownloadState[] = ['preparing', 'downloading', 'merging', 'verifying'];

/**
 * Download task list hook
 */
export function useDownloads() {
  return useQuery({
    queryKey: ['downloads'],
    queryFn: async () => {
      const response = await downloadsApi.list();
      return response.data?.downloads || [];
    },
    staleTime: 5 * 1000,
    refetchInterval: (query) => {
      const data = query.state.data as DownloadTask[] | undefined;
      if (!data || data.length === 0) return false;
      const hasActiveTasks = data.some(task => ACTIVE_DOWNLOAD_STATES.includes(task.state));
      // SSE now drives real-time updates; polling is a fallback only
      return hasActiveTasks ? 10000 : false;
    },
  });
}

/**
 * Create download task hook
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
 * Pause download hook
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
 * Resume download hook
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
 * Cancel download hook
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
 * Filter download tasks
 */
export function filterDownloads(
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

    if (filters.state && task.state !== filters.state) return false;

    if (filters.source && task.source !== filters.source) return false;

    return true;
  });
}

/**
 * Download statistics
 */
export function computeDownloadStats(downloads: DownloadTask[] | undefined) {
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
  const active = downloads.filter((d) => ACTIVE_DOWNLOAD_STATES.includes(d.state)).length;
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
 * Model file list hook
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
 * Search HuggingFace models hook
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
 * Model repo config hook
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
 * Available endpoints hook
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
 * Update model repo config hook
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
