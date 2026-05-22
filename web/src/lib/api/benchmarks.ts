import { apiClient } from './client';
import type {
  CreateBenchmarkRequest,
  CreateBenchmarkResponse,
  BenchmarkParamsResponse,
  BenchmarkHistoryFile,
  LlamaCppVersion,
} from '@/types';

/**
 * Benchmark API
 */
export const benchmarksApi = {
  async getParams(): Promise<BenchmarkParamsResponse> {
    return apiClient.get<BenchmarkParamsResponse>('/models/param/benchmark/list');
  },

  async getLlamaCppVersions(): Promise<{ success: boolean; data?: { items: LlamaCppVersion[] } }> {
    return apiClient.get('/llamacpp/list');
  },

  async create(params: CreateBenchmarkRequest): Promise<CreateBenchmarkResponse> {
    return apiClient.post<CreateBenchmarkResponse>('/models/benchmark', params);
  },

  async listHistory(modelId: string): Promise<{ success: boolean; data?: { files: BenchmarkHistoryFile[] } }> {
    return apiClient.get(`/models/benchmark/list?modelId=${encodeURIComponent(modelId)}`);
  },

  async getHistoryFile(fileName: string): Promise<{ success: boolean; data?: { rawOutput: string; fileName: string } }> {
    return apiClient.get(`/models/benchmark/get?fileName=${encodeURIComponent(fileName)}`);
  },

  async deleteHistoryFile(fileName: string): Promise<{ success: boolean }> {
    return apiClient.post(`/models/benchmark/delete?fileName=${encodeURIComponent(fileName)}`, {});
  },
};
