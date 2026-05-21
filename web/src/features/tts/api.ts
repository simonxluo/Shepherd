import { apiClient } from '@/lib/api/client';

// ============ Types ============

export interface TTSHistoryItem {
  id: string;
  model: string;
  inputText: string;
  audioPath: string;
  format: string;
  duration: number;
  favourite: boolean;
  params?: Record<string, unknown>;
  createdAt: string;
}

export interface TTSHistoryListResponse {
  items: TTSHistoryItem[];
  total: number;
}

export interface TTSHistoryListParams {
  limit?: number;
  offset?: number;
  favourite?: boolean;
}

// ============ API Functions ============

export async function fetchTTSHistory(params?: TTSHistoryListParams): Promise<TTSHistoryListResponse> {
  const query = new URLSearchParams();
  if (params?.limit) query.set('limit', String(params.limit));
  if (params?.offset) query.set('offset', String(params.offset));
  if (params?.favourite !== undefined) query.set('favourite', String(params.favourite));

  const qs = query.toString();
  const url = `/tts/history${qs ? `?${qs}` : ''}`;
  return apiClient.get<TTSHistoryListResponse>(url);
}

export async function fetchTTSHistoryItem(id: string): Promise<TTSHistoryItem> {
  return apiClient.get<TTSHistoryItem>(`/tts/history/${id}`);
}

export async function createTTSHistory(formData: FormData): Promise<TTSHistoryItem> {
  const res = await fetch('/api/tts/history', {
    method: 'POST',
    body: formData,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `Failed to create TTS history (${res.status})`);
  }
  return res.json();
}

export async function toggleTTSFavourite(id: string, favourite: boolean): Promise<void> {
  await apiClient.put(`/tts/history/${id}/favourite`, { favourite });
}

export async function deleteTTSHistory(id: string): Promise<void> {
  await apiClient.delete(`/tts/history/${id}`);
}

export function getTTSAudioUrl(id: string): string {
  return `/api/tts/audio/${id}`;
}
