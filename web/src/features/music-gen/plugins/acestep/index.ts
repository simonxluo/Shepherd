import { musicRegistry } from '../../registry';
import { AceStepPanel } from './AceStepPanel';
import type { MusicPlugin, LoadedModel } from '../../types';

const aceStepPlugin: MusicPlugin = {
  id: 'acestep',
  labelKey: 'musicGen.tabs.aceStep',
  labelFallback: 'ACE-Step',
  order: 10,
  match: (model: LoadedModel) => {
    const nameLower = model.name.toLowerCase();
    return nameLower.includes('acestep') || nameLower.includes('ace-step') || nameLower.includes('ace_step');
  },
  component: AceStepPanel,
};

musicRegistry.register(aceStepPlugin);

export { aceStepPlugin };
