import { ttsRegistry } from '../../registry';
import { Qwen3TTSPanel } from './Qwen3TTSPanel';
import type { LoadedModel } from '@/features/creative/hooks';
import type { TTSPlugin } from '../../types';

const qwen3ttsPlugin: TTSPlugin = {
  id: 'qwen3tts',
  labelKey: 'tts.tabs.qwen3tts',
  labelFallback: 'Qwen3-TTS',
  order: 5,
  match: (model: LoadedModel) => {
    const nameLower = model.name.toLowerCase();
    return nameLower.includes('qwen3-tts') || nameLower.includes('qwen3tts') || nameLower.includes('qwen3_tts');
  },
  component: Qwen3TTSPanel,
  features: {
    supportsVoiceSelection: true,
    supportsInstructions: true,
    supportsRefAudio: true,
    supportsStreamPcm: true,
    supportsVoiceDesign: false,
    defaultSampleRate: 24000,
    defaultFormat: 'pcm',
  },
};

ttsRegistry.register(qwen3ttsPlugin);

export { qwen3ttsPlugin };
