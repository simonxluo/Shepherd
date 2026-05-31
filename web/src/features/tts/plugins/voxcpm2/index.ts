import { ttsRegistry } from '@/features/tts/registry';
import { VoxCPM2Panel } from './VoxCPM2Panel';
import type { LoadedModel } from '@/types/model';
import type { TTSPlugin } from '@/features/tts/types';

const voxcpm2Plugin: TTSPlugin = {
  id: 'voxcpm2',
  labelKey: 'tts.tabs.voxcpm2',
  labelFallback: 'VoxCPM2',
  order: 10,
  match: (model: LoadedModel) => {
    const nameLower = model.name.toLowerCase();
    // Match VoxCPM models, or vllm_omni models that aren't claimed by other specific plugins
    if (nameLower.includes('voxcpm')) return true;
    if (nameLower.includes('qwen3-tts') || nameLower.includes('qwen3tts') || nameLower.includes('qwen3_tts')) return false;
    if (nameLower.includes('cosyvoice') || nameLower.includes('omnivoice')) return false;
    return model.backendType === 'vllm_omni';
  },
  component: VoxCPM2Panel,
  features: {
    supportsVoiceSelection: false,   // VoxCPM2 has no preset voices; voice field ignored
    supportsInstructions: true,      // Instructions for voice design / style description
    supportsRefAudio: true,
    supportsStreamPcm: true,
    supportsVoiceDesign: true,       // Voice Design via bracket convention
    defaultSampleRate: 48000,        // VoxCPM2 outputs 48kHz
    defaultFormat: 'pcm',
  },
};

ttsRegistry.register(voxcpm2Plugin);

export { voxcpm2Plugin };
