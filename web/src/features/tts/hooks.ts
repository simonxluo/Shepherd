import { useCallback } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { v1ApiClient } from '@/lib/api/client';
import {
  useModelLoadConfig,
  useSaveModelLoadConfig,
  useDeleteModelLoadConfig,
} from '@/features/models/config';
import type { LoadedModel } from '@/types/model';
import type { LoadModelParams } from '@/types/model';
import type { TTSRequest, TTSConfig, TTSModelFeatures } from './types';
import type { VoicesResponse } from '@/lib/api/voices';
import { ttsRegistry } from './registry';

/** Default feature set returned when no plugin matches. */
const DEFAULT_FEATURES: TTSModelFeatures = {
  supportsVoiceSelection: true,
  supportsInstructions: false,
  supportsRefAudio: false,
  supportsStreamPcm: false,
  supportsVoiceDesign: false,
  defaultSampleRate: 24000,
  defaultFormat: 'mp3',
};

/**
 * Look up model features via the plugin registry.
 * Falls back to DEFAULT_FEATURES when no plugin claims the model.
 */
export function getTTSModelFeatures(model: LoadedModel): TTSModelFeatures {
  const plugin = ttsRegistry.getPluginForModel(model);
  if (plugin) return plugin.features;
  return DEFAULT_FEATURES;
}

export function useTTS() {
  return useMutation({
    mutationFn: async (params: TTSRequest & { signal?: AbortSignal }) => {
      const { signal, ...body } = params;
      const response = await fetch(`${v1ApiClient.getBaseUrl()}/audio/speech`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal,
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

export interface VoiceOption {
  id: string;
  name: string;
  description?: string;
  isUploaded?: boolean;
}

export function useVoices(model?: string) {
  return useQuery({
    queryKey: ['voices', model],
    queryFn: async () => {
      if (!model) return [];
      const res = await v1ApiClient.get<VoicesResponse>('/audio/voices', { model });
      // vLLM-Omni returns voices: string[] and uploaded_voices
      const presetVoices: VoiceOption[] = (res.voices ?? []).map(v => ({
        id: v,
        name: v,
      }));
      const uploadedVoices: VoiceOption[] = (res.uploaded_voices ?? []).map(v => ({
        id: v.name,
        name: v.name,
        description: v.speaker_description,
        isUploaded: true,
      }));

      return [...presetVoices, ...uploadedVoices];
    },
    enabled: !!model,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

function extractTTSConfig(raw?: Record<string, unknown>): TTSConfig | null {
  if (!raw) return null;
  return {
    voice: (raw.voice as string) || undefined,
    speed: raw.speed as number | undefined,
    responseFormat: (raw.responseFormat as string) || undefined,
    stream: raw.stream as boolean | undefined,
    instructions: (raw.instructions as string) || undefined,
    refAudio: (raw.refAudio as string) || undefined,
    refText: (raw.refText as string) || undefined,
    seed: raw.seed !== undefined && raw.seed !== '' ? String(raw.seed) : undefined,
    maxNewTokens: raw.maxNewTokens !== undefined && raw.maxNewTokens !== '' ? String(raw.maxNewTokens) : undefined,
    language: (raw.language as string) || undefined,
  };
}

export function useTTSConfig(modelId: string) {
  const { data, isLoading } = useModelLoadConfig(modelId);
  const rawSaveConfig = useSaveModelLoadConfig();
  const deleteConfig = useDeleteModelLoadConfig();

  const ttsConfig = (data?.exists && data.config) ? extractTTSConfig(data.config.config as Record<string, unknown>) : null;

  const saveTTSConfig = useCallback((config: TTSConfig) => {
    if (!modelId) return;
    rawSaveConfig.mutate({ modelId, config: config as unknown as LoadModelParams });
  }, [modelId, rawSaveConfig]);

  return {
    ttsConfig,
    isLoading,
    saveConfig: { ...rawSaveConfig, mutate: saveTTSConfig },
    deleteConfig,
  };
}

// ---------------------------------------------------------------------------
// Auto-transcribe hook (used by VoxCPM2 Ultimate Cloning mode)
// ---------------------------------------------------------------------------

/** Convert a data-URI or URL audio source to a File object. */
async function audioSourceToFile(audioSource: string): Promise<File> {
  if (audioSource.startsWith('data:')) {
    const [meta, base64] = audioSource.split(',');
    const mimeMatch = meta.match(/data:(.*?);/);
    const mime = mimeMatch ? mimeMatch[1] : 'audio/wav';
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    const ext = mime.includes('mp3') ? 'mp3' : mime.includes('ogg') ? 'ogg' : 'wav';
    return new File([bytes], `audio.${ext}`, { type: mime });
  }
  // URL-based: fetch then convert
  const resp = await fetch(audioSource);
  const blob = await resp.blob();
  const ext = blob.type.includes('mp3') ? 'mp3' : blob.type.includes('ogg') ? 'ogg' : 'wav';
  return new File([blob], `audio.${ext}`, { type: blob.type || 'audio/wav' });
}

/**
 * Mutation hook that calls the ASR endpoint to auto-transcribe audio.
 * Used in VoxCPM2 Ultimate Cloning mode for prompt text generation.
 */
export function useAutoTranscribe() {
  return useMutation({
    mutationFn: async ({ audioSource, asrModelName }: {
      audioSource: string;
      asrModelName: string;
    }) => {
      const file = await audioSourceToFile(audioSource);
      const formData = new FormData();
      formData.append('file', file);
      formData.append('model', asrModelName);

      const response = await fetch(`${v1ApiClient.getBaseUrl()}/audio/transcriptions`, {
        method: 'POST',
        body: formData,
      });
      if (!response.ok) throw new Error(`ASR failed: ${response.status}`);
      const result = await response.json();
      return result.text || '';
    },
  });
}
