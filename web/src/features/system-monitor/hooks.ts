import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type { SystemResources, ModelStats } from './types';

export function useSystemResources() {
  return useQuery<SystemResources>({
    queryKey: ['system', 'resources'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: SystemResources }>(
        '/system/resources'
      );
      return response.data;
    },
    refetchInterval: 3000,
    staleTime: 2000,
  });
}

export function useModelStatistics() {
  return useQuery<ModelStats[]>({
    queryKey: ['system', 'model-stats'],
    queryFn: async () => {
      const response = await apiClient.get<{
        success: boolean;
        data: { models: ModelStats[]; count: number };
      }>('/system/model-stats');
      return response.data.models;
    },
    refetchInterval: 5000,
    staleTime: 3000,
  });
}
