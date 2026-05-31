import { asrRegistry } from '@/features/asr/registry';
import { GenericASRPanel } from './GenericASRPanel';
import type { ASRPlugin } from '@/features/asr/types';

const genericPlugin: ASRPlugin = {
  id: 'generic',
  labelKey: 'asr.tabs.generic',
  labelFallback: 'Generic',
  order: 0,
  match: () => true, // Fallback: matches anything not claimed by other plugins
  component: GenericASRPanel,
};

asrRegistry.register(genericPlugin);

export { genericPlugin };
