import { useMutation } from '@tanstack/react-query';
import { v1ApiClient } from '@/lib/api/client';

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

      const response = await fetch(`${v1ApiClient.getBaseUrl()}/audio/transcriptions`, {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: { message: response.statusText } }));
        throw new Error(err.error?.message || `ASR request failed (${response.status})`);
      }

      return response.json() as Promise<ASRResponse>;
    },
  });
}
