import { apiClient } from './client';
import type {
  Benchmark,
  BenchmarkConfig,
  BenchmarkListResponse,
  CreateBenchmarkRequest,
  CreateBenchmarkResponse,
  SaveBenchmarkConfigRequest,
  SaveBenchmarkConfigResponse,
  LoadBenchmarkConfigResponse,
  BenchmarkParamsResponse,
} from '@/types';

/**
 * Benchmark API
 */
export const benchmarksApi = {
  /**
   * Get benchmark parameter list
   */
  async getParams(): Promise<BenchmarkParamsResponse> {
    return apiClient.get<BenchmarkParamsResponse>('/models/param/benchmark/list');
  },

  /**
   * Get available compute devices
   */
  async getDevices(llamaBinPath: string): Promise<{ success: boolean; data?: { devices: string[] }; error?: string }> {
    return apiClient.get<{ success: boolean; data?: { devices: string[] }; error?: string }>(
      `/model/device/list?llamaBinPath=${encodeURIComponent(llamaBinPath)}`
    );
  },

  /**
   * Get llama.cpp version list
   */
  async getLlamaCppVersions(): Promise<{ success: boolean; data?: { items: Array<{ path: string; name?: string; description?: string }> } }> {
    return apiClient.get('/llamacpp/list');
  },

  /**
   * Create benchmark task
   */
  async create(params: CreateBenchmarkRequest): Promise<CreateBenchmarkResponse> {
    return apiClient.post<CreateBenchmarkResponse>('/models/benchmark', params);
  },

  // Legacy endpoints removed


  /**
   * List benchmark tasks
   */
  async list(modelId?: string): Promise<BenchmarkListResponse> {
    const url = modelId ? `/models/benchmark/tasks?modelId=${encodeURIComponent(modelId)}` : '/models/benchmark/tasks';
    return apiClient.get<BenchmarkListResponse>(url);
  },

  /**
   * Get a single benchmark task
   */
  async get(benchmarkId: string): Promise<{ success: boolean; data?: Benchmark; error?: string }> {
    return apiClient.get<{ success: boolean; data?: Benchmark; error?: string }>(`/models/benchmark/tasks/${benchmarkId}`);
  },

  /**
   * Cancel benchmark task
   */
  async cancel(benchmarkId: string): Promise<{ success: boolean; error?: string }> {
    return apiClient.post<{ success: boolean; error?: string }>(`/models/benchmark/tasks/${benchmarkId}/cancel`);
  },

  /**
   * Save benchmark config
   */
  async saveConfig(params: SaveBenchmarkConfigRequest): Promise<SaveBenchmarkConfigResponse> {
    return apiClient.post<SaveBenchmarkConfigResponse>('/models/benchmark/configs', params);
  },

  /**
   * List benchmark configs
   */
  async listConfigs(): Promise<LoadBenchmarkConfigResponse> {
    return apiClient.get<LoadBenchmarkConfigResponse>('/models/benchmark/configs');
  },

  /**
   * Get a single benchmark config
   */
  async getConfig(name: string): Promise<{ success: boolean; data?: BenchmarkConfig; error?: string }> {
    return apiClient.get<{ success: boolean; data?: BenchmarkConfig; error?: string }>(
      `/models/benchmark/configs/${encodeURIComponent(name)}`
    );
  },

  /**
   * Delete benchmark config
   */
  async deleteConfig(name: string): Promise<{ success: boolean; error?: string }> {
    return apiClient.delete<{ success: boolean; error?: string }>(
      `/models/benchmark/configs/${encodeURIComponent(name)}`
    );
  },
};
