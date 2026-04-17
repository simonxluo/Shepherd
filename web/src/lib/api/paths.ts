/**
 * 路径配置 API 客户端
 */

import { apiClient } from './client';
import type {
  LlamaCppPathConfig,
  ModelPathConfig,
  PathListResponse,
} from '@/lib/config';

/**
 * 路径添加/更新响应
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
 * 路径测试响应
 */
interface PathTestResponse {
  success: boolean;
  data?: {
    valid: boolean;
    message?: string;
    error?: string;
  };
}

/**
 * Llama.cpp 路径管理 API
 */
export const llamacppPathsApi = {
  /**
   * 获取所有 llama.cpp 路径
   */
  list: () =>
    apiClient.get<PathListResponse<LlamaCppPathConfig>>('/config/llamacpp/paths'),

  /**
   * 添加 llama.cpp 路径
   */
  add: (data: LlamaCppPathConfig) =>
    apiClient.post<PathMutationResponse>('/config/llamacpp/paths', data),

  /**
   * 更新 llama.cpp 路径
   */
  update: (data: LlamaCppPathConfig) =>
    apiClient.put<PathMutationResponse>('/config/llamacpp/paths', data),

  /**
   * 删除 llama.cpp 路径
   */
  remove: (path: string) =>
    apiClient.delete<PathMutationResponse>(
      `/config/llamacpp/paths?path=${encodeURIComponent(path)}`
    ),

  /**
   * 测试 llama.cpp 路径
   */
  test: (path: string) =>
    apiClient.post<PathTestResponse>('/config/llamacpp/paths/test', { path }),
};

/**
 * 模型路径管理 API
 */
export const modelPathsApi = {
  /**
   * 获取所有模型路径
   */
  list: () =>
    apiClient.get<PathListResponse<ModelPathConfig>>('/config/models/paths'),

  /**
   * 添加模型路径
   */
  add: (data: ModelPathConfig) =>
    apiClient.post<PathMutationResponse>('/config/models/paths', data),

  /**
   * 更新模型路径
   */
  update: (data: ModelPathConfig) =>
    apiClient.put<PathMutationResponse>('/config/models/paths', data),

  /**
   * 删除模型路径
   */
  remove: (path: string) =>
    apiClient.delete<PathMutationResponse>(
      `/config/models/paths?path=${encodeURIComponent(path)}`
    ),
};
