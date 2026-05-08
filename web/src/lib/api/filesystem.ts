/**
 * Filesystem API client
 */

import { apiClient } from './client';

/**
 * Directory item
 */
export interface DirectoryItem {
  name: string;
  path: string;
  size?: number; // File size in bytes; empty for directories
}

/**
 * Directory list response
 */
interface DirectoryListResponse {
  currentPath: string;
  parentPath: string;
  folders: DirectoryItem[];
  files: DirectoryItem[];
  roots?: DirectoryItem[];
  error?: string;
}

/**
 * Filesystem API
 */
export const filesystemApi = {
  /**
   * List directory contents
   * @param path Directory path; empty for root
   */
  listDirectory: (path?: string): Promise<{ success: boolean; data: DirectoryListResponse }> =>
    apiClient.get<{ success: boolean; data: DirectoryListResponse }>('/system/filesystem', path ? { path } : undefined),
};
