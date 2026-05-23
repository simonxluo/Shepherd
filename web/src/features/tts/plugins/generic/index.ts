import { ttsRegistry } from '../../registry';
import { GenericTTSPanel } from '../../components/GenericTTSPanel';
import type { TTSPlugin } from '../../types';

const genericPlugin: TTSPlugin = {
  id: 'generic',
  labelKey: 'tts.tabs.generic',
  labelFallback: 'Generic',
  order: 0,
  match: () => true, // Fallback: matches anything not claimed by other plugins
  component: GenericTTSPanel,
  features: {
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
  },
};

ttsRegistry.register(genericPlugin);

export { genericPlugin };
