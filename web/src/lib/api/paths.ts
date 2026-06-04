/**
 * Path configuration API client
 */

import { apiClient } from './client';
import type {
  LlamaCppPathConfig,
  ModelPathConfig,
  BackendPathConfig,
  MultimodalPathConfig,
  PathListResponse,
} from '@/lib/config';

/**
 * Path add/update response
 */
interface PathMutationResponse {
  success: boolean;
  data?: {
    message: string;
    added?: LlamaCppPathConfig | ModelPathConfig;
    updated?: ModelPathConfig;
    removed?: string;
    count: number;
  };
  error?: string;
}

/**
 * Path test response
 */
interface PathTestResponse {
  success: boolean;
  data?: {
    valid: boolean;
    message?: string;
    error?: string;
    binary?: string;
    version?: string;
    warnings?: string[];
    path?: string;
  };
}

/**
 * Llama.cpp path management API
 */
export const llamacppPathsApi = {
  /**
   * List all llama.cpp paths
   */
  list: () =>
    apiClient.get<PathListResponse<LlamaCppPathConfig>>('/config/llamacpp/paths'),

  /**
   * Add llama.cpp path
   */
  add: (data: LlamaCppPathConfig) =>
    apiClient.post<PathMutationResponse>('/config/llamacpp/paths', data),

  /**
   * Update llama.cpp path
   */
  update: (data: LlamaCppPathConfig) =>
    apiClient.put<PathMutationResponse>('/config/llamacpp/paths', data),

  /**
   * Remove llama.cpp path
   */
  remove: (path: string) =>
    apiClient.delete<PathMutationResponse>(
      `/config/llamacpp/paths?path=${encodeURIComponent(path)}`
    ),

  /**
   * Test llama.cpp path
   */
  test: (path: string) =>
    apiClient.post<PathTestResponse>('/config/llamacpp/paths/test', { path }),
};

/**
 * Model path management API
 */
export const modelPathsApi = {
  /**
   * List all model paths
   */
  list: () =>
    apiClient.get<PathListResponse<ModelPathConfig>>('/config/models/paths'),

  /**
   * Add model path
   */
  add: (data: ModelPathConfig) =>
    apiClient.post<PathMutationResponse>('/config/models/paths', data),

  /**
   * Update model path
   */
  update: (data: ModelPathConfig) =>
    apiClient.put<PathMutationResponse>('/config/models/paths', data),

  /**
   * Remove model path
   */
  remove: (path: string) =>
    apiClient.delete<PathMutationResponse>(
      `/config/models/paths?path=${encodeURIComponent(path)}`
    ),
};

export const vllmPathsApi = {
  list: () =>
    apiClient.get<PathListResponse<BackendPathConfig>>('/config/vllm/paths'),
  add: (data: BackendPathConfig) =>
    apiClient.post<PathMutationResponse>('/config/vllm/paths', data),
  update: (data: BackendPathConfig) =>
    apiClient.put<PathMutationResponse>('/config/vllm/paths', data),
  remove: (path: string) =>
    apiClient.delete<PathMutationResponse>(
      `/config/vllm/paths?path=${encodeURIComponent(path)}`
    ),
  test: (path: string) =>
    apiClient.post<PathTestResponse>('/config/vllm/paths/test', { path }),
};

export const vllmOmniPathsApi = {
  list: () =>
    apiClient.get<PathListResponse<BackendPathConfig>>('/config/vllmomni/paths'),
  add: (data: BackendPathConfig) =>
    apiClient.post<PathMutationResponse>('/config/vllmomni/paths', data),
  update: (data: BackendPathConfig) =>
    apiClient.put<PathMutationResponse>('/config/vllmomni/paths', data),
  remove: (path: string) =>
    apiClient.delete<PathMutationResponse>(
      `/config/vllmomni/paths?path=${encodeURIComponent(path)}`
    ),
  test: (path: string) =>
    apiClient.post<PathTestResponse>('/config/vllmomni/paths/test', { path }),
};

export const multimodalPathsApi = {
  list: () =>
    apiClient.get<PathListResponse<MultimodalPathConfig>>('/config/multimodal/paths'),
  add: (data: MultimodalPathConfig) =>
    apiClient.post<PathMutationResponse>('/config/multimodal/paths', data),
  update: (data: MultimodalPathConfig) =>
    apiClient.put<PathMutationResponse>('/config/multimodal/paths', data),
  remove: (path: string) =>
    apiClient.delete<PathMutationResponse>(
      `/config/multimodal/paths?path=${encodeURIComponent(path)}`
    ),
};
