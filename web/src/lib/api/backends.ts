import { useQuery } from '@tanstack/react-query';
import { apiClient } from './client';

export interface ParamDef {
  name: string;
  jsonName: string;
  flag: string;
  type: 'int' | 'float' | 'string' | 'bool' | 'strings';
  group: string;
  description: string;
  default?: unknown;
  min?: number;
  max?: number;
  options?: unknown[];
  advanced?: boolean;
  sinceVersion?: string;
}

export interface ParamSchema {
  pluginID: string;
  params: ParamDef[];
}

export interface BackendInfo {
  id: string;
  displayName: string;
  binPath: string;
  version: string;
  available: boolean;
}

export function useBackendParamSchema(pluginId: string) {
  return useQuery<ParamSchema>({
    queryKey: ['backends', pluginId, 'param-schema'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: ParamSchema }>(
        `/backends/${pluginId}/param-schema`
      );
      return response.data;
    },
    enabled: !!pluginId,
    staleTime: 30 * 60 * 1000,
  });
}

export function useInferenceBackends() {
  return useQuery<Record<string, BackendInfo>>({
    queryKey: ['backends', 'inference'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: Record<string, BackendInfo> }>(
        '/system/inference-backends'
      );
      return response.data;
    },
    staleTime: 60 * 1000,
  });
}
