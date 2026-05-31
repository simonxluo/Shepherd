import { musicRegistry } from '@/features/music-gen/registry';
import { GenericMusicPanel } from './GenericMusicPanel';
import type { MusicPlugin } from '@/features/music-gen/types';

const genericMusicPlugin: MusicPlugin = {
  id: 'generic',
  labelKey: 'musicGen.tabs.generic',
  labelFallback: 'Generic',
  order: 0,
  match: () => true,
  component: GenericMusicPanel,
};

musicRegistry.register(genericMusicPlugin);

export { genericMusicPlugin };
