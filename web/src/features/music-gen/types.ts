import type { LoadedModel } from '@/features/creative/hooks';

/**
 * Extended music generation request with ACE-Step-specific fields.
 */
export interface MusicGenRequest {
  model: string;
  prompt: string;
  duration?: number;
  response_format?: string;
  temperature?: number;
  // ACE-Step extended fields
  lyrics?: string;
  bpm?: number;
  key_scale?: string;
  time_signature?: string;
  vocal_language?: string;
  inference_steps?: number;
  guidance_scale?: number;
  seed?: number;
  task_type?: string;
}

/**
 * Props passed to every Music plugin panel component.
 */
export interface MusicPluginPanelProps {
  /** Currently selected model within this plugin's scope */
  model: LoadedModel | null;
  /** All loaded models that match this plugin */
  matchedModels: LoadedModel[];
  /** Trigger music generation */
  onGenerate: (payload: MusicGenRequest) => void;
  /** Whether generation is currently in progress */
  isGenerating: boolean;
  /** Selected model change callback */
  onModelChange: (modelName: string) => void;
}

/**
 * A Music plugin definition that can be registered in the plugin registry.
 */
export interface MusicPlugin {
  /** Unique identifier, e.g., 'generic', 'acestep' */
  id: string;
  /** i18n key for the tab label */
  labelKey: string;
  /** Fallback label if i18n key is not found */
  labelFallback: string;
  /** Match function: returns true if this plugin can handle the given model */
  match: (model: LoadedModel) => boolean;
  /** The panel component to render for this plugin */
  component: React.ComponentType<MusicPluginPanelProps>;
  /** Sort order for tab display (lower = leftmost/topmost) */
  order?: number;
}

export type { LoadedModel };
