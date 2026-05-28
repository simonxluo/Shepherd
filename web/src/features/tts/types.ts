import type { LoadedModel } from '@/features/creative/hooks';
import type { TTSRequest, TTSModelFeatures, TTSConfig } from './hooks';
import type { StreamState, TTSStreamMetrics } from './lib/StreamAudioPlayer';

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

export type { TTSRequest, TTSModelFeatures, TTSConfig, StreamState, TTSStreamMetrics, LoadedModel };
