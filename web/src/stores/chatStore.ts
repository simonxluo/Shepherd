import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface SamplingParams {
  temperature: number;
  topP: number;
  topK: number;
  maxTokens: number;
  repeatPenalty: number;
}

const defaultSamplingParams: SamplingParams = {
  temperature: 0.7,
  topP: 0.95,
  topK: 40,
  maxTokens: 2048,
  repeatPenalty: 1.1,
};

interface ChatState {
  activeConvId: string | null;
  selectedModel: string;
  showSidebar: boolean;
  systemPrompt: string;
  samplingParams: SamplingParams;

  setActiveConvId: (id: string | null) => void;
  setSelectedModel: (model: string) => void;
  setShowSidebar: (show: boolean) => void;
  toggleSidebar: () => void;
  resetConversation: () => void;
  setSystemPrompt: (prompt: string) => void;
  setSamplingParams: (params: Partial<SamplingParams>) => void;
  resetSamplingParams: () => void;
}

export const useChatStore = create<ChatState>()(
  persist(
    (set) => ({
      activeConvId: null,
      selectedModel: '',
      showSidebar: false,
      systemPrompt: '',
      samplingParams: { ...defaultSamplingParams },

      setActiveConvId: (id) => set({ activeConvId: id }),
      setSelectedModel: (model) => set({ selectedModel: model }),
      setShowSidebar: (show) => set({ showSidebar: show }),
      toggleSidebar: () => set((s) => ({ showSidebar: !s.showSidebar })),
      resetConversation: () => set({ activeConvId: null }),
      setSystemPrompt: (prompt) => set({ systemPrompt: prompt }),
      setSamplingParams: (params) =>
        set((s) => ({ samplingParams: { ...s.samplingParams, ...params } })),
      resetSamplingParams: () => set({ samplingParams: { ...defaultSamplingParams } }),
    }),
    {
      name: 'shepherd-chat-store',
      partialize: (state) => ({
        systemPrompt: state.systemPrompt,
        samplingParams: state.samplingParams,
        selectedModel: state.selectedModel,
      }),
    },
  ),
);
