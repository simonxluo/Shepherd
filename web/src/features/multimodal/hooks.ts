import { useMutation, useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type { ModelCapabilities } from '@/types/model';

export interface LoadedModel {
  id: string;
  name: string;
  alias?: string;
  capabilities?: ModelCapabilities;
}

interface LoadedModelsResponse {
  success: boolean;
  models: LoadedModel[];
}

export function useLoadedModels() {
  return useQuery({
    queryKey: ['models', 'loaded'],
    queryFn: async () => {
      const res = await apiClient.get<LoadedModelsResponse>('/models/loaded');
      return res.models ?? [];
    },
    refetchInterval: 5000,
  });
}

export interface TTSRequest {
  model: string;
  input: string;
  voice?: string;
  response_format?: string;
  speed?: number;
  language?: string;
}

export function useTTS() {
  return useMutation({
    mutationFn: async (params: TTSRequest) => {
      const response = await fetch('/v1/audio/speech', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: { message: response.statusText } }));
        throw new Error(err.error?.message || `TTS 请求失败 (${response.status})`);
      }

      const blob = await response.blob();
      return { blob, contentType: response.headers.get('Content-Type') || 'audio/mpeg' };
    },
  });
}

export interface ASRRequest {
  model: string;
  file: File;
  language?: string;
  prompt?: string;
  response_format?: string;
  temperature?: number;
}

export interface ASRResponse {
  text: string;
  language?: string;
  duration?: number;
}

export function useASR() {
  return useMutation({
    mutationFn: async (params: ASRRequest) => {
      const formData = new FormData();
      formData.append('file', params.file);
      formData.append('model', params.model);
      if (params.language) formData.append('language', params.language);
      if (params.prompt) formData.append('prompt', params.prompt);
      if (params.response_format) formData.append('response_format', params.response_format);
      if (params.temperature !== undefined) formData.append('temperature', String(params.temperature));

      const response = await fetch('/v1/audio/transcriptions', {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: { message: response.statusText } }));
        throw new Error(err.error?.message || `ASR 请求失败 (${response.status})`);
      }

      return response.json() as Promise<ASRResponse>;
    },
  });
}

export interface ImageGenerationRequest {
  model: string;
  prompt: string;
  n?: number;
  size?: string;
  response_format?: string;
  quality?: string;
  style?: string;
}

export interface ImageGenerationResponse {
  created: number;
  data: Array<{
    url?: string;
    b64_json?: string;
  }>;
}

export function useImageGeneration() {
  return useMutation({
    mutationFn: async (params: ImageGenerationRequest) => {
      const response = await fetch('/v1/images/generations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: { message: response.statusText } }));
        throw new Error(err.error?.message || `图像生成请求失败 (${response.status})`);
      }

      return response.json() as Promise<ImageGenerationResponse>;
    },
  });
}
