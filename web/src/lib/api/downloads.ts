/**
 * 下载管理 API 客户端
 */

import { apiClient } from './client';
import type { CreateDownloadParams, DownloadTask, DownloadListResponse } from '@/types';

/**
 * 模型文件信息
 */
export interface ModelFileInfo {
  name: string;
  size: number;
  download_url: string;
}

/**
 * 模型文件列表响应
 */
export interface ModelFilesResponse {
  success: boolean;
  data: ModelFileInfo[];
  error?: string;
}

/**
 * HuggingFace 模型信息
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
 * HuggingFace 搜索结果响应
 */
export interface HuggingFaceSearchResponse {
  success: boolean;
  data: {
    models: HuggingFaceModel[];
    count: number;
    total: number;
  };
  error?: string;
}

/**
 * 下载列表 API 响应（统一格式）
 */
export interface DownloadListApiResponse {
  success: boolean;
  data: {
    downloads: DownloadTask[];
    total: number;
  };
  error?: string;
}

/**
 * 单个下载任务 API 响应（统一格式）
 */
export interface DownloadTaskApiResponse {
  success: boolean;
  data: DownloadTask;
  error?: string;
}

/**
 * 下载管理 API
 */
export const downloadsApi = {
  /**
   * 获取下载任务列表
   */
  list: (): Promise<DownloadListApiResponse> =>
    apiClient.get<DownloadListApiResponse>('/downloads'),

  /**
   * 创建下载任务
   */
  create: (params: CreateDownloadParams): Promise<{ success: boolean; message?: string; data?: DownloadTask; error?: string }> =>
    apiClient.post('/downloads', params),

  /**
   * 获取单个下载任务
   */
  get: (id: string): Promise<DownloadTaskApiResponse> =>
    apiClient.get<DownloadTaskApiResponse>(`/downloads/${id}`),

  /**
   * 暂停下载
   */
  pause: (id: string): Promise<{ success: boolean; message?: string; error?: string }> =>
    apiClient.post(`/downloads/${id}/pause`),

  /**
   * 恢复下载
   */
  resume: (id: string): Promise<{ success: boolean; message?: string; error?: string }> =>
    apiClient.post(`/downloads/${id}/resume`),

  /**
   * 取消下载
   */
  cancel: (id: string): Promise<{ success: boolean; message?: string; error?: string }> =>
    apiClient.delete<{ success: boolean; message?: string; error?: string }>(`/downloads/${id}`),

  /**
   * 重试下载
   */
  retry: (id: string): Promise<{ success: boolean; message?: string; error?: string }> =>
    apiClient.post(`/downloads/${id}/retry`),

  /**
   * 清理已完成的下载
   */
  clearCompleted: (): Promise<{ success: boolean; message?: string; error?: string }> =>
    apiClient.delete<{ success: boolean; message?: string; error?: string }>('/downloads/completed'),

  /**
   * 获取模型文件列表
   * 使用查询参数以支持 repoId 中包含斜杠 (如 Qwen/Qwen2-7B-Instruct)
   * 支持 AbortSignal 用于取消请求
   */
  listModelFiles: (source: 'huggingface' | 'modelscope', repoId: string, signal?: AbortSignal): Promise<ModelFilesResponse> =>
    apiClient.get<ModelFilesResponse>('/repo/files', { source, repoId }, signal),

  /**
   * 搜索 HuggingFace 模型
   */
  searchHuggingFace: (query: string, limit?: number, signal?: AbortSignal): Promise<HuggingFaceSearchResponse> =>
    apiClient.get<HuggingFaceSearchResponse>('/repo/search', { q: query, limit: limit || 20 }, signal),

  /**
   * 获取模型仓库配置
   */
  getModelRepoConfig: (): Promise<{ success: boolean; data: { endpoint: string; token: string; timeout: number }; error?: string }> =>
    apiClient.get('/repo/config'),

  /**
   * 更新模型仓库配置
   */
  updateModelRepoConfig: (config: { endpoint?: string; token?: string; timeout?: number }): Promise<{ success: boolean; data: { endpoint: string; token: string; timeout: number }; error?: string }> =>
    apiClient.put('/repo/config', config),

  /**
   * 获取可用端点列表
   */
  getAvailableEndpoints: (): Promise<{ success: boolean; data: Record<string, string>; error?: string }> =>
    apiClient.get('/repo/endpoints'),
};
