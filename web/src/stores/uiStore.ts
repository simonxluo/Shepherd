import { create } from 'zustand';
import { persist } from 'zustand/middleware';

/**
 * Theme type
 */
export type Theme = 'light' | 'dark' | 'system';

/**
 * UI state interface
 */
interface UIState {
  // Sidebar
  sidebarOpen: boolean;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;

  // Theme
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;

  // Current view
  currentView: string;
  setCurrentView: (view: string) => void;

  // Modals
  activeModal: string | null;
  openModal: (modal: string) => void;
  closeModal: () => void;

  // Filters
  modelStatusFilter: string;
  setModelStatusFilter: (status: string) => void;
  showFavouritesOnly: boolean;
  setShowFavouritesOnly: (show: boolean) => void;

  // Model view mode
  modelViewMode: 'grid' | 'list';
  setModelViewMode: (mode: 'grid' | 'list') => void;
}

/**
 * Get system theme
 */
function getSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

/**
 * Apply theme to DOM
 */
function applyTheme(theme: Theme) {
  const root = document.documentElement;
  const effectiveTheme = theme === 'system' ? getSystemTheme() : theme;

  root.classList.remove('light', 'dark');
  root.classList.add(effectiveTheme);
}

/**
 * UI state store
 */
export const useUIStore = create<UIState>()(
  persist(
    (set, get) => ({
      // Sidebar
      sidebarOpen: true,
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),

      // Theme
      theme: 'system',
      setTheme: (theme) => {
        set({ theme });
        applyTheme(theme);
      },
      toggleTheme: () => {
        const currentTheme = get().theme;
        const themes: Theme[] = ['light', 'dark', 'system'];
        const currentIndex = themes.indexOf(currentTheme);
        const nextTheme = themes[(currentIndex + 1) % themes.length];
        get().setTheme(nextTheme);
      },

      // Current view
      currentView: 'dashboard',
      setCurrentView: (currentView) => set({ currentView }),

      // Modals
      activeModal: null,
      openModal: (activeModal) => set({ activeModal }),
      closeModal: () => set({ activeModal: null }),

      // Filters
      modelStatusFilter: 'all',
      setModelStatusFilter: (modelStatusFilter) => set({ modelStatusFilter }),
      showFavouritesOnly: false,
      setShowFavouritesOnly: (showFavouritesOnly) => set({ showFavouritesOnly }),

      // Model view mode
      modelViewMode: 'grid',
      setModelViewMode: (modelViewMode) => set({ modelViewMode }),
    }),
    {
      name: 'shepherd-ui-storage',
      partialize: (state) => ({
        theme: state.theme,
        sidebarOpen: state.sidebarOpen,
        modelViewMode: state.modelViewMode,
      }),
    }
  )
);

// Initialize theme
applyTheme(useUIStore.getState().theme);

// Listen for system theme changes
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  if (useUIStore.getState().theme === 'system') {
    applyTheme('system');
  }
});
