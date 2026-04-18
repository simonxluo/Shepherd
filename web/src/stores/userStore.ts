import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User, UserSettings, UpdateProfileRequest } from '@/types/user';

interface UserState {
  // 当前用户
  user: User | null;
  isAuthenticated: boolean;
  
  // 用户设置
  settings: UserSettings;
  
  // Actions
  setUser: (user: User | null) => void;
  updateProfile: (data: UpdateProfileRequest) => void;
  updateSettings: (settings: Partial<UserSettings>) => void;
  logout: () => void;
  
  // UI State
  showProfileDialog: boolean;
  setShowProfileDialog: (show: boolean) => void;
  showSettingsDialog: boolean;
  setShowSettingsDialog: (show: boolean) => void;
}

const defaultSettings: UserSettings = {
  theme: 'system',
  language: 'zh-CN',
  notifications: true,
  autoSave: true,
};

export const useUserStore = create<UserState>()(
  persist(
    (set, get) => ({
      // 初始状态
      user: null,
      isAuthenticated: false,
      settings: defaultSettings,
      
      // Actions
      setUser: (user) => set({ 
        user, 
        isAuthenticated: !!user 
      }),
      
      updateProfile: (data) => {
        const { user } = get();
        if (user) {
          set({
            user: { ...user, ...data }
          });
        }
      },
      
      updateSettings: (newSettings) => {
        set((state) => ({
          settings: { ...state.settings, ...newSettings }
        }));
      },
      
      logout: () => set({
        user: null,
        isAuthenticated: false,
      }),
      
      // UI State
      showProfileDialog: false,
      setShowProfileDialog: (show) => set({ showProfileDialog: show }),
      showSettingsDialog: false,
      setShowSettingsDialog: (show) => set({ showSettingsDialog: show }),
    }),
    {
      name: 'shepherd-user-storage',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        settings: state.settings,
      }),
    }
  )
);
