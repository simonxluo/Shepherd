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

  // Theme
  theme: Theme;
  setTheme: (theme: Theme) => void;

  // Model view mode
  modelViewMode: 'grid' | 'list';
  setModelViewMode: (mode: 'grid' | 'list') => void;

  // Mobile menu overlay
  mobileMenuOpen: boolean;
  toggleMobileMenu: () => void;
  setMobileMenuOpen: (open: boolean) => void;

  // Mobile More panel
  morePanelOpen: boolean;
  setMorePanelOpen: (open: boolean) => void;
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
    (set) => ({
      // Sidebar
      sidebarOpen: true,
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),

      // Theme
      theme: 'system',
      setTheme: (theme) => {
        set({ theme });
        applyTheme(theme);
      },

      // Model view mode
      modelViewMode: 'grid',
      setModelViewMode: (modelViewMode) => set({ modelViewMode }),

      // Mobile menu overlay
      mobileMenuOpen: false,
      toggleMobileMenu: () => set((s) => ({ mobileMenuOpen: !s.mobileMenuOpen })),
      setMobileMenuOpen: (mobileMenuOpen) => set({ mobileMenuOpen }),

      // Mobile More panel
      morePanelOpen: false,
      setMorePanelOpen: (morePanelOpen) => set({ morePanelOpen }),
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
