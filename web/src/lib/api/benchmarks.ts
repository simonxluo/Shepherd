import { apiClient } from './client';
import type {
  CreateBenchmarkRequest,
  CreateBenchmarkResponse,
  BenchmarkParamsResponse,
  BenchmarkHistoryFile,
  LlamaCppVersion,
  BenchmarkV2Request,
  BenchmarkV2Response,
  BenchmarkV2Record,
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

  // V2 Benchmark APIs
  async createV2(params: BenchmarkV2Request): Promise<BenchmarkV2Response> {
    return apiClient.post<BenchmarkV2Response>('/models/benchmark/v2', params);
  },

  async listV2(modelId: string): Promise<{ success: boolean; data?: { records: BenchmarkV2Record[] } }> {
    return apiClient.get(`/models/benchmark/v2/list?modelId=${encodeURIComponent(modelId)}`);
  },

  async deleteV2(modelId: string, lineNumber: number): Promise<{ success: boolean }> {
    return apiClient.post('/models/benchmark/v2/delete', { modelId, lineNumber });
  },
};
