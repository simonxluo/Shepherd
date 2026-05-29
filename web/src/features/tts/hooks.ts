import { useMutation, useQuery } from '@tanstack/react-query';
import { v1Client } from '@/features/creative/hooks';
import {
  useModelLoadConfig,
  useSaveModelLoadConfig,
  useDeleteModelLoadConfig,
} from '@/features/models/config';
import type { LoadedModel } from '@/features/creative/hooks';

export interface TTSRequest {
  model: string;
  input: string;
  voice?: string;
  response_format?: string;
  speed?: number;
  language?: string;
  stream?: boolean;
  // VoxCPM2 / 声音克隆扩展字段
  instructions?: string;
  ref_audio?: string;
  ref_text?: string;
  prompt_audio?: string;
  prompt_text?: string;
  max_new_tokens?: number;
  seed?: number;
  cfg_value?: number;
  inference_timesteps?: number;
  emotion?: string;
  extra_params?: Record<string, unknown>;
}

export interface TTSModelFeatures {
  supportsVoiceSelection: boolean;
  supportsInstructions: boolean;
  supportsRefAudio: boolean;
  supportsUltimateCloning: boolean;
  supportsStreamPcm: boolean;
  supportsCfgValue: boolean;
  supportsInferenceTimesteps: boolean;
  supportsCfgCutoffRatio: boolean;
  supportsSwaySampling: boolean;
  supportsEmotion: boolean;
  defaultSampleRate: number;
  defaultFormat: string;
}

export function getTTSModelFeatures(model: LoadedModel): TTSModelFeatures {
  const nameLower = model.name.toLowerCase();
  const isVoxCPM = nameLower.includes('voxcpm');
  const isOmniBackend = model.backendType === 'vllm_omni';

  if (isVoxCPM || isOmniBackend) {
    return {
      supportsVoiceSelection: true,
      supportsInstructions: true,
      supportsRefAudio: true,
      supportsUltimateCloning: isVoxCPM,
      supportsStreamPcm: true,
      supportsCfgValue: isVoxCPM,
      supportsInferenceTimesteps: isVoxCPM,
      supportsCfgCutoffRatio: isVoxCPM,
      supportsSwaySampling: isVoxCPM,
      supportsEmotion: isVoxCPM,
      defaultSampleRate: 24000,
      defaultFormat: 'pcm',
    };
  }

  return {
    supportsVoiceSelection: true,
    supportsInstructions: false,
    supportsRefAudio: false,
    supportsUltimateCloning: false,
    supportsStreamPcm: false,
    supportsCfgValue: false,
    supportsInferenceTimesteps: false,
    supportsCfgCutoffRatio: false,
    supportsSwaySampling: false,
    supportsEmotion: false,
    defaultSampleRate: 24000,
    defaultFormat: 'mp3',
  };
}

export function useTTS() {
  return useMutation({
    mutationFn: async (params: TTSRequest & { signal?: AbortSignal }) => {
      const { signal, ...body } = params;
      const response = await fetch(`${v1Client.getBaseUrl()}/audio/speech`, {
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

interface VoicesResponse {
  voices?: string[];
  uploaded_voices?: Array<{
    name: string;
    speaker_description?: string;
    ref_text?: string;
  }>;
}

const VOICES_CACHE_KEY = 'shepherd-tts-voices-cache';

function getVoicesCache(): Record<string, VoiceOption[]> {
  try {
    const saved = localStorage.getItem(VOICES_CACHE_KEY);
    return saved ? JSON.parse(saved) : {};
  } catch {
    return {};
  }
}

function saveVoicesCache(cache: Record<string, VoiceOption[]>) {
  try {
    localStorage.setItem(VOICES_CACHE_KEY, JSON.stringify(cache));
  } catch { /* silent */ }
}

export function useVoices(model?: string) {
  return useQuery({
    queryKey: ['voices', model],
    queryFn: async () => {
      if (!model) return [];
      const res = await v1Client.get<VoicesResponse>('/audio/voices', { model });
      // vLLM-Omni 返回 voices: string[] 和 uploaded_voices: Array<{name, ...}>
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
    placeholderData: (): VoiceOption[] => {
      if (!model) return [];
      const cache = getVoicesCache();
      return cache[model] ?? [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export interface TTSConfig {
  voice?: string;
  speed?: number;
  responseFormat?: string;
  stream?: boolean;
  instructions?: string;
  refAudio?: string;
  refText?: string;
  promptAudio?: string;
  promptText?: string;
  ultimateCloning?: boolean;
  seed?: string;
  maxNewTokens?: string;
  language?: string;
  emotion?: string;
  cfgValue?: string;
  inferenceTimesteps?: string;
  cfgCutoffRatio?: string;
  swaySamplingCoef?: string;
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
    promptAudio: (raw.promptAudio as string) || undefined,
    promptText: (raw.promptText as string) || undefined,
    ultimateCloning: raw.ultimateCloning as boolean | undefined,
    seed: (raw.seed as string) || undefined,
    maxNewTokens: (raw.maxNewTokens as string) || undefined,
    language: (raw.language as string) || undefined,
    emotion: (raw.emotion as string) || undefined,
    cfgValue: (raw.cfgValue as string) || undefined,
    inferenceTimesteps: (raw.inferenceTimesteps as string) || undefined,
    cfgCutoffRatio: (raw.cfgCutoffRatio as string) || undefined,
    swaySamplingCoef: (raw.swaySamplingCoef as string) || undefined,
  };
}

export function useTTSConfig(modelId: string) {
  const { data, isLoading } = useModelLoadConfig(modelId);
  const saveConfig = useSaveModelLoadConfig();
  const deleteConfig = useDeleteModelLoadConfig();

  const ttsConfig = (data?.exists && data.config) ? extractTTSConfig(data.config.config as Record<string, unknown>) : null;

  return { ttsConfig, isLoading, saveConfig, deleteConfig };
}
