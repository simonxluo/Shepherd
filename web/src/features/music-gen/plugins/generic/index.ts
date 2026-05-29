import { musicRegistry } from '../../registry';
import { GenericMusicPanel } from './GenericMusicPanel';
import type { MusicPlugin } from '../../types';

const genericMusicPlugin: MusicPlugin = {
  id: 'generic',
  labelKey: 'musicGen.tabs.generic',
  labelFallback: 'Generic',
  order: 0,
  match: () => true,
  component: GenericMusicPanel,
  features: {
    supportsLyrics: false,
    supportsBPM: false,
    supportsKeyScale: false,
    supportsTimeSignature: false,
    supportsVocalLanguage: false,
    supportsInferenceSteps: false,
    supportsGuidanceScale: false,
    supportsSeed: false,
    supportsTaskType: false,
    defaultDuration: 30,
    defaultFormat: 'wav',
  },
};

musicRegistry.register(genericMusicPlugin);

export { genericMusicPlugin };
