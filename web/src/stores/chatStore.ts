import { create } from 'zustand';

interface ChatState {
  activeConvId: string | null;
  selectedModel: string;
  showSidebar: boolean;

  setActiveConvId: (id: string | null) => void;
  setSelectedModel: (model: string) => void;
  setShowSidebar: (show: boolean) => void;
  toggleSidebar: () => void;
  resetConversation: () => void;
}

export const useChatStore = create<ChatState>()((set) => ({
  activeConvId: null,
  selectedModel: '',
  showSidebar: false,

  setActiveConvId: (id) => set({ activeConvId: id }),
  setSelectedModel: (model) => set({ selectedModel: model }),
  setShowSidebar: (show) => set({ showSidebar: show }),
  toggleSidebar: () => set((s) => ({ showSidebar: !s.showSidebar })),
  resetConversation: () => set({ activeConvId: null }),
}));
