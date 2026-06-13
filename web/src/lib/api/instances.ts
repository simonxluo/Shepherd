import { apiClient } from './client';

export interface RuntimeInstance {
  instanceId: string;
  modelId: string;
  modelName: string;
  profileId?: string;
  installationId?: string;
  processId?: string;
  port?: number;
  state: string;
  pluginId?: string;
  commandPreview?: string;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RuntimeInstancesResponse {
  instances: RuntimeInstance[];
  count: number;
}

export const instancesApi = {
  list: () => apiClient.get<{ success: boolean; data: RuntimeInstancesResponse }>('/instances'),
  get: (id: string) =>
    apiClient.get<{ success: boolean; data: { instance: RuntimeInstance } }>(`/instances/${encodeURIComponent(id)}`),
  stop: (id: string) =>
    apiClient.post<{ success: boolean }>(`/instances/${encodeURIComponent(id)}/stop`),
};
