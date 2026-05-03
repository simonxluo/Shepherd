import { useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type {
  Model,
  ModelListResponse,
  LoadModelParams,
  ModelStatus,
  ModelCapabilities,
} from '@/types';
import type { ModelCategory } from './capabilities';
import { getModelCategory } from './capabilities';

/**
 * Model list hook
 */
export function useModels() {
  return useQuery<Model[]>({
    queryKey: ['models'],
    queryFn: async (): Promise<Model[]> => {
      const response = await apiClient.get<{ success: boolean; data: ModelListResponse }>('/models');
      const data = response.data;
      return data.models;
    },
    staleTime: 2000,
    refetchOnWindowFocus: true,
    refetchInterval: (query) => {
      const data = query.state.data;
      const hasLoadingOrUnloading = data?.some(m => m.status === 'loading' || m.status === 'unloading');
      return hasLoadingOrUnloading ? 2000 : false;
    },
  });
}

/**
 * Single model hook
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
 * Load model hook
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
      await queryClient.invalidateQueries({ queryKey: ['models'], refetchType: 'all' });
    },
  });
}

/**
 * Unload model hook
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
      await queryClient.invalidateQueries({ queryKey: ['models'], refetchType: 'all' });
    },
  });
}

/**
 * Update model alias hook
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
 * Set model favourite hook
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
 * Filter models hook (with sorting)
 */
export function useFilteredModels(
  models: Model[] | undefined,
  filters: {
    search?: string;
    status?: ModelStatus;
    favourite?: boolean;
    category?: ModelCategory;
    capabilitiesMap?: Record<string, ModelCapabilities>;
  }
): Model[] {
  return useMemo(() => {
    if (!models) return [];

    const filtered = models.filter((model) => {
      if (filters.search) {
        const search = filters.search.toLowerCase();
        const matchName = model.name ? model.name.toLowerCase().includes(search) : false;
        const matchAlias = model.alias ? model.alias.toLowerCase().includes(search) : false;
        const matchArch = model.metadata.architecture ? model.metadata.architecture.toLowerCase().includes(search) : false;
        if (!matchName && !matchAlias && !matchArch) return false;
      }

      if (filters.status && model.status !== filters.status) return false;

      if (filters.favourite && !model.favourite) return false;

      if (filters.category && filters.category !== 'all') {
        const caps = filters.capabilitiesMap?.[model.id];
        const cat = getModelCategory(caps);
        if (cat !== filters.category) return false;
      }

      return true;
    });

    return [...filtered].sort((a: Model, b: Model) => {
      const aArch = (a.metadata?.architecture || '').toLowerCase();
      const bArch = (b.metadata?.architecture || '').toLowerCase();
      const archCompare = aArch.localeCompare(bArch);
      if (archCompare !== 0) return archCompare;

      const aName = (a.alias || a.displayName || a.name).toLowerCase();
      const bName = (b.alias || b.displayName || b.name).toLowerCase();

      const nameCompare = aName.localeCompare(bName, 'zh-CN');
      if (nameCompare !== 0) return nameCompare;

      const aTime = new Date(a.scannedAt).getTime();
      const bTime = new Date(b.scannedAt).getTime();
      if (aTime !== bTime) return bTime - aTime;

      return a.path.localeCompare(b.path);
    });
  }, [models, filters.search, filters.status, filters.favourite, filters.category, filters.capabilitiesMap]);
}
