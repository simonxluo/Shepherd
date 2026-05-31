import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// ---------------------------------------------------------------------------
// Form defaults
// ---------------------------------------------------------------------------

const defaultGenericForm = {
  language: '',
  prompt: '',
  responseFormat: 'text',
  temperature: 0,
};

const defaultQwen3Form = {
  language: '',
  prompt: '',
  responseFormat: 'text',
  temperature: 0,
};

// ---------------------------------------------------------------------------
// Store types
// ---------------------------------------------------------------------------

export type GenericASRForm = typeof defaultGenericForm;
export type Qwen3ASRForm = typeof defaultQwen3Form;

interface ASRState {
  // Shared persisted state
  activePluginId: string;
  modelByPlugin: Record<string, string>;
  genericForm: GenericASRForm;
  qwen3Form: Qwen3ASRForm;

  // Actions — shared
  setActivePluginId: (id: string) => void;
  setModelForPlugin: (pluginId: string, modelName: string) => void;

  // Actions — generic form
  setGenericField: <K extends keyof GenericASRForm>(key: K, value: GenericASRForm[K]) => void;
  resetGenericForm: () => void;

  // Actions — qwen3 form
  setQwen3Field: <K extends keyof Qwen3ASRForm>(key: K, value: Qwen3ASRForm[K]) => void;
  resetQwen3Form: () => void;

  /** Reset a specific plugin's form to defaults */
  resetForm: (pluginId: string) => void;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useASRStore = create<ASRState>()(
  persist(
    (set) => ({
      activePluginId: 'generic',
      modelByPlugin: {},
      genericForm: { ...defaultGenericForm },
      qwen3Form: { ...defaultQwen3Form },

      // Shared actions
      setActivePluginId: (id) => set({ activePluginId: id }),
      setModelForPlugin: (pluginId, modelName) =>
        set((s) => ({
          modelByPlugin: { ...s.modelByPlugin, [pluginId]: modelName },
        })),

      // Generic form
      setGenericField: (key, value) =>
        set((s) => ({
          genericForm: { ...s.genericForm, [key]: value },
        })),
      resetGenericForm: () => set({ genericForm: { ...defaultGenericForm } }),

      // Qwen3 form
      setQwen3Field: (key, value) =>
        set((s) => ({
          qwen3Form: { ...s.qwen3Form, [key]: value },
        })),
      resetQwen3Form: () => set({ qwen3Form: { ...defaultQwen3Form } }),

      resetForm: (pluginId) => {
        switch (pluginId) {
          case 'qwen3asr':
            set({ qwen3Form: { ...defaultQwen3Form } });
            break;
          default:
            set({ genericForm: { ...defaultGenericForm } });
            break;
        }
      },
    }),
    {
      name: 'shepherd-asr-store',
      partialize: (state) => ({
        activePluginId: state.activePluginId,
        modelByPlugin: state.modelByPlugin,
        genericForm: state.genericForm,
        qwen3Form: state.qwen3Form,
      }),
    },
  ),
);
