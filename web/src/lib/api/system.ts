/**
 * System API client
 */

import { apiClient } from './client';

/**
 * Server info response
 */
export interface ServerInfoResponse {
  success: boolean;
  data?: {
    version: string;
    buildTime: string;
    gitCommit: string;
    name: string;
    status: string;
    mode: string;
    ports: {
      web: number;
      anthropic: number;
      ollama: number;
      lmstudio: number;
    };
  };
  error?: string;
}

/**
 * System API
 */
export const systemApi = {
  /**
   * Get server info (version, build time, etc.)
   */
  getInfo: () => apiClient.get<ServerInfoResponse>('/info'),
};
