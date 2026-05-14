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

export interface UpdateProfileRequest {
  displayName?: string;
  email?: string;
  avatar?: string;
}

