export interface User {
  id: string;
  username: string;
  displayName: string;
  email?: string;
  avatar?: string;
  role: 'admin' | 'user';
  createdAt: string;
  lastLoginAt?: string;
}

export interface UserSettings {
  theme: 'light' | 'dark' | 'system';
  language: string;
  notifications: boolean;
  autoSave: boolean;
  defaultModel?: string;
}

export interface UserProfile {
  user: User;
  settings: UserSettings;
}

export interface UpdateProfileRequest {
  displayName?: string;
  email?: string;
  avatar?: string;
}

export interface UpdateSettingsRequest {
  theme?: 'light' | 'dark' | 'system';
  language?: string;
  notifications?: boolean;
  autoSave?: boolean;
  defaultModel?: string;
}

