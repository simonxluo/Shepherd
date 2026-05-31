import { ttsRegistry } from '@/features/tts/registry';
import { GenericTTSPanel } from '@/features/tts/components/GenericTTSPanel';
import type { TTSPlugin } from '@/features/tts/types';

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
    supportsStreamPcm: false,
    supportsVoiceDesign: false,
    defaultSampleRate: 24000,
    defaultFormat: 'mp3',
  },
};

ttsRegistry.register(genericPlugin);

export { genericPlugin };
