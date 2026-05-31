import { v1ApiClient } from './client';

export interface VoiceInfo {
  name: string;
  consent?: string;
  created_at?: number;
  file_size?: number;
  mime_type?: string;
  embedding_source?: string;
  ref_text?: string;
  speaker_description?: string;
}

export interface VoicesResponse {
  voices: string[];
  uploaded_voices: VoiceInfo[];
}

export async function listVoices(model: string): Promise<VoicesResponse> {
  return v1ApiClient.get<VoicesResponse>('/audio/voices', { model });
}

export async function uploadVoice(
  model: string,
  file: File,
  name: string,
  options?: { consent?: string; ref_text?: string; speaker_description?: string }
): Promise<{ success: boolean; voice: VoiceInfo }> {
  const form = new FormData();
  form.append('model', model);
  form.append('name', name);
  form.append('consent', options?.consent || 'user-consent');
  form.append('audio_sample', file);
  if (options?.ref_text) form.append('ref_text', options.ref_text);
  if (options?.speaker_description) form.append('speaker_description', options.speaker_description);

  const resp = await fetch(`${v1ApiClient.getBaseUrl()}/audio/voices`, { method: 'POST', body: form });
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ error: { message: resp.statusText } }));
    throw new Error(err?.error?.message || `Upload failed: ${resp.status}`);
  }
  return resp.json();
}

export async function deleteVoice(model: string, name: string): Promise<void> {
  return v1ApiClient.delete(`/audio/voices/${encodeURIComponent(name)}?model=${encodeURIComponent(model)}`);
}
