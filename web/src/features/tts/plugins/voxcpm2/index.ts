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
    // Match VoxCPM models, or vllm_omni models that aren't claimed by other specific plugins
    if (nameLower.includes('voxcpm')) return true;
    if (nameLower.includes('qwen3-tts') || nameLower.includes('qwen3tts') || nameLower.includes('qwen3_tts')) return false;
    if (nameLower.includes('cosyvoice') || nameLower.includes('omnivoice')) return false;
    return model.backendType === 'vllm_omni';
  },
  component: VoxCPM2Panel,
  features: {
    supportsVoiceSelection: false,   // VoxCPM2 无预设声音，voice 字段被忽略
    supportsInstructions: true,
    supportsRefAudio: true,
    supportsUltimateCloning: true,
    supportsStreamPcm: true,
    supportsCfgValue: true,
    supportsInferenceTimesteps: true,
    supportsCfgCutoffRatio: true,
    supportsSwaySampling: true,
    supportsEmotion: true,
    defaultSampleRate: 48000,         // VoxCPM2 输出 48kHz
    defaultFormat: 'pcm',
  },
};

ttsRegistry.register(voxcpm2Plugin);

export { voxcpm2Plugin };
