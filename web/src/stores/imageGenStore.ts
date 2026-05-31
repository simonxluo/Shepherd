import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// ---------------------------------------------------------------------------
// Form defaults
// ---------------------------------------------------------------------------

const defaultForm = {
  model: '',
  prompt: '',
  size: '1024x1024',
  n: 1,
  quality: 'standard',
  style: 'vivid',
};

// ---------------------------------------------------------------------------
// Store types
// ---------------------------------------------------------------------------

export type ImageGenForm = typeof defaultForm;

interface ImageGenState {
  form: ImageGenForm;

  setField: <K extends keyof ImageGenForm>(key: K, value: ImageGenForm[K]) => void;
  resetForm: () => void;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useImageGenStore = create<ImageGenState>()(
  persist(
    (set) => ({
      form: { ...defaultForm },

      setField: (key, value) =>
        set((s) => ({
          form: { ...s.form, [key]: value },
        })),
      resetForm: () => set({ form: { ...defaultForm } }),
    }),
    {
      name: 'shepherd-imagegen-store',
      partialize: (state) => ({
        form: state.form,
      }),
    },
  ),
);
