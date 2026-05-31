import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// ---------------------------------------------------------------------------
// Form defaults
// ---------------------------------------------------------------------------

const defaultAceStepForm = {
  prompt: '',
  lyrics: '',
  duration: 30,
  responseFormat: 'wav',
  vocalLanguage: 'en',
  bpm: '',
  keyScale: '__auto',
  timeSignature: '__auto',
  inferenceSteps: 8,
  guidanceScale: 7.0,
  seed: '',
};

const defaultGenericForm = {
  prompt: '',
  duration: 30,
  responseFormat: 'wav',
  temperature: 0.7,
};

// ---------------------------------------------------------------------------
// Store types
// ---------------------------------------------------------------------------

export type AceStepForm = typeof defaultAceStepForm;
export type GenericMusicForm = typeof defaultGenericForm;

interface MusicGenState {
  // Shared persisted state
  activePluginId: string;
  modelByPlugin: Record<string, string>;
  aceStepForm: AceStepForm;
  genericForm: GenericMusicForm;

  // Actions — shared
  setActivePluginId: (id: string) => void;
  setModelForPlugin: (pluginId: string, modelName: string) => void;

  // Actions — AceStep form
  setAceStepField: <K extends keyof AceStepForm>(key: K, value: AceStepForm[K]) => void;
  resetAceStepForm: () => void;

  // Actions — generic form
  setGenericField: <K extends keyof GenericMusicForm>(key: K, value: GenericMusicForm[K]) => void;
  resetGenericForm: () => void;

  /** Reset a specific plugin's form to defaults */
  resetForm: (pluginId: string) => void;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useMusicGenStore = create<MusicGenState>()(
  persist(
    (set) => ({
      activePluginId: 'acestep',
      modelByPlugin: {},
      aceStepForm: { ...defaultAceStepForm },
      genericForm: { ...defaultGenericForm },

      // Shared actions
      setActivePluginId: (id) => set({ activePluginId: id }),
      setModelForPlugin: (pluginId, modelName) =>
        set((s) => ({
          modelByPlugin: { ...s.modelByPlugin, [pluginId]: modelName },
        })),

      // AceStep form
      setAceStepField: (key, value) =>
        set((s) => ({
          aceStepForm: { ...s.aceStepForm, [key]: value },
        })),
      resetAceStepForm: () => set({ aceStepForm: { ...defaultAceStepForm } }),

      // Generic form
      setGenericField: (key, value) =>
        set((s) => ({
          genericForm: { ...s.genericForm, [key]: value },
        })),
      resetGenericForm: () => set({ genericForm: { ...defaultGenericForm } }),

      resetForm: (pluginId) => {
        switch (pluginId) {
          case 'acestep':
            set({ aceStepForm: { ...defaultAceStepForm } });
            break;
          default:
            set({ genericForm: { ...defaultGenericForm } });
            break;
        }
      },
    }),
    {
      name: 'shepherd-musicgen-store',
      partialize: (state) => ({
        activePluginId: state.activePluginId,
        modelByPlugin: state.modelByPlugin,
        aceStepForm: state.aceStepForm,
        genericForm: state.genericForm,
      }),
    },
  ),
);
