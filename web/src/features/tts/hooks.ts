import { useMutation, useQuery } from '@tanstack/react-query';
import { v1Client } from '@/features/creative/hooks';

export interface TTSRequest {
  model: string;
  input: string;
  voice?: string;
  response_format?: string;
  speed?: number;
  language?: string;
  stream?: boolean;
}

export function useTTS() {
  return useMutation({
    mutationFn: async (params: TTSRequest) => {
      const response = await fetch(`${v1Client.getBaseUrl()}/audio/speech`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: { message: response.statusText } }));
        throw new Error(err.error?.message || `TTS request failed (${response.status})`);
      }

      const blob = await response.blob();
      return { blob, contentType: response.headers.get('Content-Type') || 'audio/mpeg' };
    },
  });
}

interface VoicesResponse {
  voices?: Array<{ id: string; name?: string }>;
}

export function useVoices(model?: string) {
  return useQuery({
    queryKey: ['voices', model],
    queryFn: async () => {
      if (!model) return [];
      const res = await v1Client.get<VoicesResponse>('/audio/voices', { model });
      return res.voices ?? [];
    },
    enabled: !!model,
  });
}
