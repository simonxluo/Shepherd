import type { LoadedModel } from '@/types/model';
import type { StreamState, TTSStreamMetrics } from './lib/StreamAudioPlayer';
import type { ModelStatus } from '@/types';

/** Request payload sent to the /v1/audio/speech endpoint. */
export interface TTSRequest {
  model: string;
  input: string;
  voice?: string;
  response_format?: string;
  speed?: number;
  language?: string;
  stream?: boolean;
  // Voice cloning / ultimate cloning extensions
  instructions?: string;
  ref_audio?: string;
  ref_text?: string;
  max_new_tokens?: number;
  seed?: number;
  // Sampling params (Qwen3-TTS)
  temperature?: number;
  top_p?: number;
  top_k?: number;
  repetition_penalty?: number;
  x_vector_only_mode?: boolean;
  /** vLLM-Omni escape hatch: merged directly into SamplingParams.extra_args */
  extra_params?: Record<string, unknown>;
}

/** Persisted configuration for a TTS model. */
export interface TTSConfig {
  voice?: string;
  speed?: number;
  responseFormat?: string;
  stream?: boolean;
  instructions?: string;
  refAudio?: string;
  refText?: string;
  seed?: string;
  maxNewTokens?: string;
  language?: string;
}

/** Feature flags describing what a TTS model / plugin supports. */
export interface TTSModelFeatures {
  supportsVoiceSelection: boolean;
  supportsInstructions: boolean;
  supportsRefAudio: boolean;
  supportsStreamPcm: boolean;
  /** Voice Design mode via bracket convention */
  supportsVoiceDesign: boolean;
  defaultSampleRate: number;
  defaultFormat: string;
}

// ---------------------------------------------------------------------------
// Plugin interfaces
// ---------------------------------------------------------------------------

/**
 * Props passed to every TTS plugin panel component.
 */
export interface TTSPluginPanelProps {
  /** Currently selected model within this plugin's scope */
  model: LoadedModel | null;
  /** All loaded models that match this plugin */
  matchedModels: LoadedModel[];
  /** Trigger speech generation */
  onGenerate: (payload: TTSRequest) => void;
  /** Cancel in-progress generation */
  onCancel?: () => void;
  /** Whether generation is currently in progress */
  isGenerating: boolean;
  /** Current stream playback state */
  streamState: StreamState;
  /** Stream playback metrics */
  streamMetrics: TTSStreamMetrics;
  /** Audio URL for non-stream playback */
  audioUrl: string | null;
  /** Selected model change callback */
  onModelChange: (modelName: string) => void;
  /** External ref audio override (e.g., from history "Use as reference") */
  refAudioOverride?: string;
  /** 模型实时状态（来自 useModels 全量列表） */
  modelStatus?: ModelStatus;
  /** 完整模型 ID（用于加载操作） */
  fullModelId?: string;
  /** 递增计数器，用于触发插件面板刷新语音列表 */
  voiceRefreshTrigger?: number;
}

/**
 * A TTS plugin definition that can be registered in the plugin registry.
 */
export interface TTSPlugin {
  /** Unique identifier, e.g., 'generic', 'voxcpm2' */
  id: string;
  /** i18n key for the tab label */
  labelKey: string;
  /** Fallback label if i18n key is not found */
  labelFallback: string;
  /** Match function: returns true if this plugin can handle the given model */
  match: (model: LoadedModel) => boolean;
  /** The panel component to render for this plugin */
  component: React.ComponentType<TTSPluginPanelProps>;
  /** Feature descriptor for this plugin's models */
  features: TTSModelFeatures;
  /** Default configuration values */
  defaultConfig?: Partial<TTSConfig>;
  /** Sort order for tab display (lower = leftmost/topmost) */
  order?: number;
}

export type { StreamState, TTSStreamMetrics };
