import type { LoadedModel } from '@/types/model';
import type { ModelStatus } from '@/types';

/** Request payload sent to the /v1/audio/transcriptions endpoint. */
export interface ASRRequest {
  model: string;
  file: File;
  language?: string;
  prompt?: string;
  response_format?: string;
  temperature?: number;
}

/** Response from the ASR transcription endpoint. */
export interface ASRResponse {
  text: string;
  language?: string;
  duration?: number;
}

/**
 * Props passed to every ASR plugin panel component.
 */
export interface ASRPluginPanelProps {
  /** Currently selected model within this plugin's scope */
  model: LoadedModel | null;
  /** All loaded models that match this plugin */
  matchedModels: LoadedModel[];
  /** Trigger transcription */
  onTranscribe: (payload: ASRRequest) => void;
  /** Whether transcription is currently in progress */
  isTranscribing: boolean;
  /** Transcription result */
  result: ASRResponse | null;
  /** Selected model change callback */
  onModelChange: (modelName: string) => void;
  /** Model runtime status */
  modelStatus?: ModelStatus;
  /** Full model ID (for load operations) */
  fullModelId?: string;
}

/**
 * A ASR plugin definition that can be registered in the plugin registry.
 */
export interface ASRPlugin {
  /** Unique identifier, e.g., 'generic', 'qwen3asr' */
  id: string;
  /** i18n key for the tab label */
  labelKey: string;
  /** Fallback label if i18n key is not found */
  labelFallback: string;
  /** Match function: returns true if this plugin can handle the given model */
  match: (model: LoadedModel) => boolean;
  /** The panel component to render for this plugin */
  component: React.ComponentType<ASRPluginPanelProps>;
  /** Sort order for tab display (lower = leftmost/topmost) */
  order?: number;
}
