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

  async listHistory(modelId: string, nodeId?: string): Promise<{ success: boolean; data?: { files: BenchmarkHistoryFile[] } }> {
    let url = `/models/benchmark/list?modelId=${encodeURIComponent(modelId)}`;
    if (nodeId) url += `&nodeId=${encodeURIComponent(nodeId)}`;
    return apiClient.get(url);
  },

  async getHistoryFile(fileName: string, nodeId?: string): Promise<{ success: boolean; data?: { rawOutput: string; fileName: string } }> {
    let url = `/models/benchmark/get?fileName=${encodeURIComponent(fileName)}`;
    if (nodeId) url += `&nodeId=${encodeURIComponent(nodeId)}`;
    return apiClient.get(url);
  },

  async deleteHistoryFile(fileName: string, nodeId?: string): Promise<{ success: boolean }> {
    let url = `/models/benchmark/delete?fileName=${encodeURIComponent(fileName)}`;
    if (nodeId) url += `&nodeId=${encodeURIComponent(nodeId)}`;
    return apiClient.post(url, {});
  },

  // V2 Benchmark APIs
  async createV2(params: BenchmarkV2Request): Promise<BenchmarkV2Response> {
    return apiClient.post<BenchmarkV2Response>('/models/benchmark/v2', params);
  },

  async listV2(modelId: string, nodeId?: string): Promise<{ success: boolean; data?: { records: BenchmarkV2Record[] } }> {
    let url = `/models/benchmark/v2/list?modelId=${encodeURIComponent(modelId)}`;
    if (nodeId) url += `&nodeId=${encodeURIComponent(nodeId)}`;
    return apiClient.get(url);
  },

  async deleteV2(modelId: string, lineNumber: number, nodeId?: string): Promise<{ success: boolean }> {
    return apiClient.post('/models/benchmark/v2/delete', { modelId, lineNumber, nodeId });
  },
};
