import { apiClient } from './client';
import type {
  BenchmarkConfig,
  CreateBenchmarkRequest,
  CreateBenchmarkResponse,
  BenchmarkParamsResponse,
  LlamaCppVersion,
} from '@/types';

/**
 * Benchmark API
 */
export const benchmarksApi = {
  async getParams(): Promise<BenchmarkParamsResponse> {
    return apiClient.get<BenchmarkParamsResponse>('/models/param/benchmark/list');
  },

  async getDevices(llamaBinPath: string): Promise<{ success: boolean; data?: { devices: string[] }; error?: string }> {
    return apiClient.get<{ success: boolean; data?: { devices: string[] }; error?: string }>(
      `/model/device/list?llamaBinPath=${encodeURIComponent(llamaBinPath)}`
    );
  },

  async getLlamaCppVersions(): Promise<{ success: boolean; data?: { items: LlamaCppVersion[] } }> {
    return apiClient.get('/llamacpp/list');
  },

  async create(params: CreateBenchmarkRequest): Promise<CreateBenchmarkResponse> {
    return apiClient.post<CreateBenchmarkResponse>('/models/benchmark', params);
  },
};
