import { ttsRegistry } from '../../registry';
import { VoxCPM2Panel } from './VoxCPM2Panel';
import type { TTSPlugin, LoadedModel } from '../../types';

const voxcpm2Plugin: TTSPlugin = {
  id: 'voxcpm2',
  labelKey: 'tts.tabs.voxcpm2',
  labelFallback: 'VoxCPM2',
  order: 10,
  match: (model: LoadedModel) => {
    const nameLower = model.name.toLowerCase();
    return nameLower.includes('voxcpm') || model.backendType === 'vllm_omni';
  },
  component: VoxCPM2Panel,
  features: {
    supportsVoiceSelection: false,
    supportsInstructions: true,
    supportsRefAudio: true,
    supportsUltimateCloning: true,
    supportsStreamPcm: true,
    supportsCfgValue: true,
    supportsInferenceTimesteps: true,
    supportsEmotion: true,
    defaultSampleRate: 24000,
    defaultFormat: 'pcm',
  },
};

ttsRegistry.register(voxcpm2Plugin);

export { voxcpm2Plugin };
