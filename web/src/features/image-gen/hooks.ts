import { useMutation } from '@tanstack/react-query';
import { v1ApiClient } from '@/lib/api/client';

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
      return v1ApiClient.post<ImageGenerationResponse>('/images/generations', params);
    },
  });
}
