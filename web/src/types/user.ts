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
  language: string;
  notifications: boolean;
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
  language?: string;
  notifications?: boolean;
  defaultModel?: string;
}

