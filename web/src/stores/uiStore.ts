import { create } from 'zustand';
import { persist } from 'zustand/middleware';

/**
 * 主题类型
 */
export type Theme = 'light' | 'dark' | 'system';

/**
 * UI 状态接口
 */
interface UIState {
  // 侧边栏
  sidebarOpen: boolean;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;

  // 主题
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;

  // 当前视图
  currentView: string;
  setCurrentView: (view: string) => void;

  // 模态框
  activeModal: string | null;
  openModal: (modal: string) => void;
  closeModal: () => void;

  // 过滤器
  modelStatusFilter: string;
  setModelStatusFilter: (status: string) => void;
  showFavouritesOnly: boolean;
  setShowFavouritesOnly: (show: boolean) => void;

  // 模型视图模式
  modelViewMode: 'grid' | 'list';
  setModelViewMode: (mode: 'grid' | 'list') => void;
}

/**
 * 获取系统主题
 */
function getSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

/**
 * 应用主题到 DOM
 */
function applyTheme(theme: Theme) {
  const root = document.documentElement;
  const effectiveTheme = theme === 'system' ? getSystemTheme() : theme;

  root.classList.remove('light', 'dark');
  root.classList.add(effectiveTheme);
}

/**
 * UI 状态 Store
 */
export const useUIStore = create<UIState>()(
  persist(
    (set, get) => ({
      // 侧边栏
      sidebarOpen: true,
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),

      // 主题
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

      // 当前视图
      currentView: 'dashboard',
      setCurrentView: (currentView) => set({ currentView }),

      // 模态框
      activeModal: null,
      openModal: (activeModal) => set({ activeModal }),
      closeModal: () => set({ activeModal: null }),

      // 过滤器
      modelStatusFilter: 'all',
      setModelStatusFilter: (modelStatusFilter) => set({ modelStatusFilter }),
      showFavouritesOnly: false,
      setShowFavouritesOnly: (showFavouritesOnly) => set({ showFavouritesOnly }),

      // 模型视图模式
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

// 初始化主题
applyTheme(useUIStore.getState().theme);

// 监听系统主题变化
window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  if (useUIStore.getState().theme === 'system') {
    applyTheme('system');
  }
});
