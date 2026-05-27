import { apiClient } from './client';

export interface LlamaCppParamDef {
  name: string;
  jsonName: string;
  flag: string;
  type: 'int' | 'float' | 'string' | 'bool' | 'strings';
  group: string;
  description: string;
  default?: unknown;
  min?: number;
  max?: number;
  options?: string[];
  advanced?: boolean;
  sinceVersion?: string;
}

export interface LlamaCppParamRegistry {
  backend: string;
  params: LlamaCppParamDef[];
}

export interface LlamaCppCommandPreviewRequest {
  binPath?: string;
  modelPath: string;
  port: number;
  ctxSize?: number;
  gpuLayers?: number;
  threads?: number;
  devices?: string[];
  llamacppParams?: Record<string, unknown>;
}

export const llamacppApi = {
  schema: () => apiClient.get<{ success: boolean; data: LlamaCppParamRegistry }>('/backends/llamacpp/schema'),
  preview: (request: LlamaCppCommandPreviewRequest) =>
    apiClient.post<{ success: boolean; data: { command: string; spec: unknown } }>('/backends/llamacpp/preview', request),
};
