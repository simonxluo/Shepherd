import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User, UserSettings, UpdateProfileRequest } from '@/types/user';

interface UserState {
  // Current user
  user: User | null;

  // User settings
  settings: UserSettings;

  // Actions
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
  language: 'zh-CN',
  notifications: true,
};

export const useUserStore = create<UserState>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      settings: defaultSettings,

      // Actions
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
        settings: state.settings,
      }),
    }
  )
);
