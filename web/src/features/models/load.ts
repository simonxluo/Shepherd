import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { apiClient } from '@/lib/api/client';
import type {
  Model,
  ModelListResponse,
  LoadModelParams,
  ModelStatus,
} from '@/types';

/**
 * 模型列表 Hook
 */
export function useModels() {
  return useQuery<Model[]>({
    queryKey: ['models'],
    queryFn: async (): Promise<Model[]> => {
      const response = await apiClient.get<{ success: boolean; data: ModelListResponse }>('/models');
      const data = response.data;
      return data.models;
    },
    staleTime: 2000, // 2秒后数据视为过期
    refetchOnWindowFocus: true, // 窗口获得焦点时刷新
    refetchInterval: (query) => {
      // 如果有模型正在加载或卸载，每2秒刷新一次
      const data = query.state.data;
      const hasLoadingOrUnloading = data?.some(m => m.status === 'loading' || m.status === 'unloading');
      return hasLoadingOrUnloading ? 2000 : false;
    },
  });
}

/**
 * 单个模型 Hook
 */
export function useModel(modelId: string) {
  return useQuery({
    queryKey: ['models', modelId],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: { model: Model } }>(`/models/${modelId}`);
      return response.data.model;
    },
    enabled: !!modelId,
  });
}

/**
 * 加载模型 Hook
 */
export function useLoadModel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: Partial<LoadModelParams>) => {
      const response = await apiClient.post<{ success: boolean }>(
        `/models/${params.modelId}/load`,
        params
      );
      return response;
    },
    onSuccess: async () => {
      // 使模型列表查询失效并强制重新获取
      await queryClient.invalidateQueries({ queryKey: ['models'], refetchType: 'all' });
    },
  });
}

/**
 * 卸载模型 Hook
 */
export function useUnloadModel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (modelId: string) => {
      const response = await apiClient.post<{ success: boolean }>(
        `/models/${modelId}/unload`
      );
      return response;
    },
    onSuccess: async () => {
      // 使模型列表查询失效并强制重新获取
      await queryClient.invalidateQueries({ queryKey: ['models'], refetchType: 'all' });
    },
  });
}

/**
 * 更新模型别名 Hook
 */
export function useUpdateModelAlias() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, alias }: { modelId: string; alias: string }) => {
      const response = await apiClient.put<{ success: boolean }>(
        `/models/${modelId}/alias`,
        { alias }
      );
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['models'] });
    },
  });
}

/**
 * 设置模型收藏 Hook
 */
export function useSetModelFavourite() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ modelId, favourite }: { modelId: string; favourite: boolean }) => {
      const response = await apiClient.put<{ success: boolean }>(
        `/models/${modelId}/favourite`,
        { favourite }
      );
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['models'] });
    },
  });
}

/**
 * 过滤模型 Hook（包含排序）
 * 使用 useMemo 确保排序稳定且只依赖项变化时重新计算
 */
export function useFilteredModels(
  models: Model[] | undefined,
  filters: {
    search?: string;
    status?: ModelStatus;
    favourite?: boolean;
  }
): Model[] {
  return useMemo(() => {
    if (!models) return [];

    // 过滤模型
    const filtered = models.filter((model) => {
      // 搜索过滤
      if (filters.search) {
        const search = filters.search.toLowerCase();
        const matchName = model.name ? model.name.toLowerCase().includes(search) : false;
        const matchAlias = model.alias ? model.alias.toLowerCase().includes(search) : false;
        const matchArch = model.metadata.architecture ? model.metadata.architecture.toLowerCase().includes(search) : false;
        if (!matchName && !matchAlias && !matchArch) return false;
      }

      // 状态过滤
      if (filters.status && model.status !== filters.status) return false;

      // 收藏过滤
      if (filters.favourite && !model.favourite) return false;

      return true;
    });

    // 排序模型：稳定的排序，确保每次刷新后顺序一致
    // 排序优先级：架构 > 名称（字母）> 扫描时间 > 路径
    return [...filtered].sort((a: Model, b: Model) => {
      // 优先按架构排序
      const aArch = (a.metadata?.architecture || '').toLowerCase();
      const bArch = (b.metadata?.architecture || '').toLowerCase();
      const archCompare = aArch.localeCompare(bArch);
      if (archCompare !== 0) return archCompare;

      // 架构相同时，按显示名称（别名或模型名）排序
      const aName = (a.alias || a.displayName || a.name).toLowerCase();
      const bName = (b.alias || b.displayName || b.name).toLowerCase();

      const nameCompare = aName.localeCompare(bName, 'zh-CN');
      if (nameCompare !== 0) return nameCompare;

      // 名称相同时，按扫描时间降序排序（最新的在前）
      const aTime = new Date(a.scannedAt).getTime();
      const bTime = new Date(b.scannedAt).getTime();
      if (aTime !== bTime) return bTime - aTime;

      // 扫描时间也相同时，按路径排序
      return a.path.localeCompare(b.path);
    });
  }, [models, filters.search, filters.status, filters.favourite]);
}
