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
  list: (params?: { backendType?: string; modelScope?: string }) =>
    apiClient.get<{ success: boolean; data: LaunchProfilesResponse }>('/launch-profiles', params as Record<string, unknown>),
  get: (id: string) =>
    apiClient.get<{ success: boolean; data: { profile: LaunchProfile } }>(`/launch-profiles/${encodeURIComponent(id)}`),
  create: (profile: Omit<LaunchProfile, 'id'> & { id?: string }) =>
    apiClient.post<{ success: boolean; data: { profile: LaunchProfile } }>('/launch-profiles', profile),
  update: (id: string, profile: Partial<LaunchProfile>) =>
    apiClient.put<{ success: boolean; data: { profile: LaunchProfile } }>(`/launch-profiles/${encodeURIComponent(id)}`, profile),
  remove: (id: string) =>
    apiClient.delete<{ success: boolean }>(`/launch-profiles/${encodeURIComponent(id)}`),
};
