import { apiClient } from './client';

export interface LaunchProfile {
  id: string;
  name: string;
  backendType: string;
  installationId?: string;
  modelScope?: string;
  params: Record<string, unknown>;
  env?: string[];
  extraArgs?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface LaunchProfilesResponse {
  profiles: LaunchProfile[];
  count: number;
}

export const launchProfilesApi = {
  list: (params?: { backendType?: string; modelScope?: string }) => {
    const query = new URLSearchParams();
    if (params?.backendType) query.set('backendType', params.backendType);
    if (params?.modelScope) query.set('modelScope', params.modelScope);
    const suffix = query.toString() ? `?${query.toString()}` : '';
    return apiClient.get<{ success: boolean; data: LaunchProfilesResponse }>(`/launch-profiles${suffix}`);
  },
  get: (id: string) =>
    apiClient.get<{ success: boolean; data: { profile: LaunchProfile } }>(`/launch-profiles/${encodeURIComponent(id)}`),
  create: (profile: Omit<LaunchProfile, 'id'> & { id?: string }) =>
    apiClient.post<{ success: boolean; data: { profile: LaunchProfile } }>('/launch-profiles', profile),
  update: (id: string, profile: Partial<LaunchProfile>) =>
    apiClient.put<{ success: boolean; data: { profile: LaunchProfile } }>(`/launch-profiles/${encodeURIComponent(id)}`, profile),
  remove: (id: string) =>
    apiClient.delete<{ success: boolean }>(`/launch-profiles/${encodeURIComponent(id)}`),
};
