import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { TTSConfig } from '@/features/tts/types';

// ---------------------------------------------------------------------------
// Form defaults
// ---------------------------------------------------------------------------

const defaultGenericForm = {
  input: '',
  voice: '',
  speed: 1,
  responseFormat: 'mp3',
  language: '',
  refAudio: '',
  refText: '',
  seed: '',
  maxNewTokens: '',
};

const defaultVoxcpm2Form = {
  input: '',
  refAudio: '',
  refText: '',
  instructions: '',
  seed: '',
  maxNewTokens: '',
  language: 'auto',
  selectedVoice: '',
};

export type GenerationMode = 'custom_voice' | 'voice_design' | 'voice_clone';

const defaultQwen3Form = {
  input: '',
  language: 'auto',
  mode: 'custom_voice' as GenerationMode,
  speaker: 'Vivian',
  instructions: '',
  voiceDesignPrompt: '',
  refAudio: '',
  refText: '',
  fastCloneMode: false,
  temperature: '0.9',
  topP: '1.0',
  topK: '50',
  repetitionPenalty: '1.05',
  maxNewTokens: '',
  seed: '',
};

// ---------------------------------------------------------------------------
// Store types
// ---------------------------------------------------------------------------

export type GenericForm = typeof defaultGenericForm;
export type Voxcpm2Form = typeof defaultVoxcpm2Form;
export type Qwen3Form = typeof defaultQwen3Form;

interface TTSState {
  // Shared persisted state
  activePluginId: string;
  modelByPlugin: Record<string, string>;
  autoPlay: boolean;
  genericForm: GenericForm;
  voxcpm2Form: Voxcpm2Form;
  qwen3Form: Qwen3Form;

  // Actions — shared
  setActivePluginId: (id: string) => void;
  setModelForPlugin: (pluginId: string, modelName: string) => void;
  setAutoPlay: (v: boolean) => void;
  toggleAutoPlay: () => void;

  // Actions — generic form
  setGenericField: <K extends keyof GenericForm>(key: K, value: GenericForm[K]) => void;
  resetGenericForm: () => void;

  // Actions — voxcpm2 form
  setVoxcpm2Field: <K extends keyof Voxcpm2Form>(key: K, value: Voxcpm2Form[K]) => void;
  resetVoxcpm2Form: () => void;

  // Actions — qwen3 form
  setQwen3Field: <K extends keyof Qwen3Form>(key: K, value: Qwen3Form[K]) => void;
  resetQwen3Form: () => void;

  /** Reset a specific plugin's form to defaults */
  resetForm: (pluginId: string) => void;

  /** Hydrate form fields from server-side TTSConfig */
  hydrateFromServerConfig: (pluginId: string, config: TTSConfig) => void;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useTTSStore = create<TTSState>()(
  persist(
    (set) => ({
      activePluginId: 'generic',
      modelByPlugin: {},
      autoPlay: false,
      genericForm: { ...defaultGenericForm },
      voxcpm2Form: { ...defaultVoxcpm2Form },
      qwen3Form: { ...defaultQwen3Form },

      // Shared actions
      setActivePluginId: (id) => set({ activePluginId: id }),
      setModelForPlugin: (pluginId, modelName) =>
        set((s) => ({
          modelByPlugin: { ...s.modelByPlugin, [pluginId]: modelName },
        })),
      setAutoPlay: (v) => set({ autoPlay: v }),
      toggleAutoPlay: () => set((s) => ({ autoPlay: !s.autoPlay })),

      // Generic form
      setGenericField: (key, value) =>
        set((s) => ({
          genericForm: { ...s.genericForm, [key]: value },
        })),
      resetGenericForm: () => set({ genericForm: { ...defaultGenericForm } }),

      // VoxCPM2 form
      setVoxcpm2Field: (key, value) =>
        set((s) => ({
          voxcpm2Form: { ...s.voxcpm2Form, [key]: value },
        })),
      resetVoxcpm2Form: () => set({ voxcpm2Form: { ...defaultVoxcpm2Form } }),

      // Qwen3 form
      setQwen3Field: (key, value) =>
        set((s) => ({
          qwen3Form: { ...s.qwen3Form, [key]: value },
        })),
      resetQwen3Form: () => set({ qwen3Form: { ...defaultQwen3Form } }),

      resetForm: (pluginId) => {
        switch (pluginId) {
          case 'voxcpm2':
            set({ voxcpm2Form: { ...defaultVoxcpm2Form } });
            break;
          case 'qwen3tts':
            set({ qwen3Form: { ...defaultQwen3Form } });
            break;
          default:
            set({ genericForm: { ...defaultGenericForm } });
            break;
        }
      },

      hydrateFromServerConfig: (pluginId, config) => {
        switch (pluginId) {
          case 'voxcpm2':
            set((s) => ({
              voxcpm2Form: {
                ...s.voxcpm2Form,
                ...(config.instructions !== undefined && { instructions: config.instructions }),
                ...(config.refAudio !== undefined && { refAudio: config.refAudio }),
                ...(config.refText !== undefined && { refText: config.refText }),
                ...(config.seed !== undefined && { seed: config.seed }),
                ...(config.maxNewTokens !== undefined && { maxNewTokens: config.maxNewTokens }),
                ...(config.language !== undefined && { language: config.language || 'auto' }),
              },
            }));
            break;
          case 'qwen3tts':
            set((s) => ({
              qwen3Form: {
                ...s.qwen3Form,
                ...(config.voice !== undefined && { speaker: config.voice }),
                ...(config.instructions !== undefined && { instructions: config.instructions }),
                ...(config.refAudio !== undefined && { refAudio: config.refAudio }),
                ...(config.refText !== undefined && { refText: config.refText }),
                ...(config.language !== undefined && { language: config.language || 'auto' }),
              },
            }));
            break;
          default:
            set((s) => ({
              genericForm: {
                ...s.genericForm,
                ...(config.voice !== undefined && { voice: config.voice }),
                ...(config.speed !== undefined && { speed: config.speed }),
                ...(config.responseFormat !== undefined && { responseFormat: config.responseFormat }),
                ...(config.refAudio !== undefined && { refAudio: config.refAudio }),
                ...(config.refText !== undefined && { refText: config.refText }),
                ...(config.seed !== undefined && { seed: config.seed }),
                ...(config.maxNewTokens !== undefined && { maxNewTokens: config.maxNewTokens }),
                ...(config.language !== undefined && { language: config.language }),
              },
            }));
            break;
        }
      },
    }),
    {
      name: 'shepherd-tts-store',
      partialize: (state) => ({
        activePluginId: state.activePluginId,
        modelByPlugin: state.modelByPlugin,
        autoPlay: state.autoPlay,
        genericForm: state.genericForm,
        voxcpm2Form: state.voxcpm2Form,
        qwen3Form: state.qwen3Form,
      }),
    },
  ),
);
