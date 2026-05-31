import { useMutation } from '@tanstack/react-query';
import { v1ApiClient } from '@/lib/api/client';
import type { MusicGenRequest } from './types';

export function useMusicGeneration() {
  return useMutation({
    mutationFn: async (params: MusicGenRequest) => {
      const response = await fetch(`${v1ApiClient.getBaseUrl()}/audio/music`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(params),
      });

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: { message: response.statusText } }));
        throw new Error(err.error?.message || `Music generation request failed (${response.status})`);
      }

      const blob = await response.blob();
      return { blob, contentType: response.headers.get('Content-Type') || 'audio/wav' };
    },
  });
}
