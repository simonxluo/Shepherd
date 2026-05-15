/**
 * Download management API client
 */

import { apiClient } from './client';
import type { CreateDownloadParams, DownloadTask } from '@/types';

/**
 * Model file info
 */
export interface ModelFileInfo {
  name: string;
  size: number;
  download_url: string;
}

/**
 * Model file list response
 */
interface ModelFilesResponse {
  success: boolean;
  data: ModelFileInfo[];
  error?: string;
}

/**
 * HuggingFace model info
 */
export interface HuggingFaceModel {
  id: string;
  modelId: string;
  author: string;
  sha: string;
  private: boolean;
  createdAt: string;
  lastModified: string;
  tags: string[];
  downloads: number;
  likes: number;
  library_name: string;
}

/**
 * HuggingFace search response
 */
interface HuggingFaceSearchResponse {
  success: boolean;
  data: {
    models: HuggingFaceModel[];
    count: number;
    total: number;
  };
  error?: string;
}

/**
 * Download list API response
 */
interface DownloadListApiResponse {
  success: boolean;
  data: {
    downloads: DownloadTask[];
    total: number;
  };
  error?: string;
}

/**
 * Single download task API response
 */
interface DownloadTaskApiResponse {
  success: boolean;
  data: DownloadTask;
  error?: string;
}

/**
 * Download management API
 */
export const downloadsApi = {
  /**
   * List download tasks
   */
  list: (): Promise<DownloadListApiResponse> =>
    apiClient.get<DownloadListApiResponse>('/downloads'),

  /**
   * Create download task
   */
  create: (params: CreateDownloadParams): Promise<{ success: boolean; message?: string; data?: DownloadTask; error?: string }> =>
    apiClient.post('/downloads', params),

  /**
   * Get a single download task
   */
  get: (id: string): Promise<DownloadTaskApiResponse> =>
    apiClient.get<DownloadTaskApiResponse>(`/downloads/${id}`),

  /**
   * Pause download
   */
  pause: (id: string): Promise<{ success: boolean; message?: string; error?: string }> =>
    apiClient.post(`/downloads/${id}/pause`),

  /**
   * Resume download
   */
  resume: (id: string): Promise<{ success: boolean; message?: string; error?: string }> =>
    apiClient.post(`/downloads/${id}/resume`),

  /**
   * Cancel download
   */
  cancel: (id: string): Promise<{ success: boolean; message?: string; error?: string }> =>
    apiClient.delete<{ success: boolean; message?: string; error?: string }>(`/downloads/${id}`),

  /**
   * List model files.
   * Uses query params to support repoId with slashes (e.g. Qwen/Qwen2-7B-Instruct).
   * Supports AbortSignal for request cancellation.
   */
  listModelFiles: (source: 'huggingface' | 'modelscope', repoId: string, signal?: AbortSignal): Promise<ModelFilesResponse> =>
    apiClient.get<ModelFilesResponse>('/repo/files', { source, repoId }, signal),

  /**
   * Search HuggingFace models
   * @param query Search keyword
   * @param limit Result count limit
   * @param format Format filter (gguf, safetensors, onnx, bin, all)
   * @param signal Abort signal
   */
  searchHuggingFace: (query: string, limit?: number, format?: string, signal?: AbortSignal): Promise<HuggingFaceSearchResponse> =>
    apiClient.get<HuggingFaceSearchResponse>('/repo/search', { q: query, limit: limit || 20, format: format || 'all' }, signal),

  /**
   * Get model repo config
   */
  getModelRepoConfig: (): Promise<{ success: boolean; data: { endpoint: string; token: string; timeout: number }; error?: string }> =>
    apiClient.get('/repo/config'),

  /**
   * Update model repo config
   */
  updateModelRepoConfig: (config: { endpoint?: string; token?: string; timeout?: number }): Promise<{ success: boolean; data: { endpoint: string; token: string; timeout: number }; error?: string }> =>
    apiClient.put('/repo/config', config),

  /**
   * Get available endpoints
   */
  getAvailableEndpoints: (): Promise<{ success: boolean; data: Record<string, string>; error?: string }> =>
    apiClient.get('/repo/endpoints'),
};
