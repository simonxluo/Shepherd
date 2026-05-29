import { asrRegistry } from '../../registry';
import { Qwen3ASRPanel } from './Qwen3ASRPanel';
import type { ASRPlugin, LoadedModel } from '../../types';

const qwen3asrPlugin: ASRPlugin = {
  id: 'qwen3asr',
  labelKey: 'asr.tabs.qwen3asr',
  labelFallback: 'Qwen3-ASR',
  order: 5,
  match: (model: LoadedModel) => {
    const nameLower = model.name.toLowerCase();
    return nameLower.includes('qwen3-asr') || nameLower.includes('qwen3asr') || nameLower.includes('qwen3_asr');
  },
  component: Qwen3ASRPanel,
};

asrRegistry.register(qwen3asrPlugin);

export { qwen3asrPlugin };
