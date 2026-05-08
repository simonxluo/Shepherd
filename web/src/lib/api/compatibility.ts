/**
 * API compatibility client
 */

import { apiClient } from './client';

/**
 * API compatibility configuration
 */
interface CompatibilityConfig {
  ollama: {
    enabled: boolean;
    port: number;
  };
  lmstudio: {
    enabled: boolean;
    port: number;
  };
}

/**
 * Compatibility config response
 */
interface CompatibilityResponse {
  success: boolean;
  data: CompatibilityConfig;
  error?: string;
}

/**
 * Update config response
 */
interface UpdateResponse {
  success: boolean;
  message?: string;
  error?: string;
  errorType?: 'in_use' | 'permission' | 'invalid' | 'unknown';
  service?: 'ollama' | 'lmstudio';
  autoDisabled?: boolean;
  data?: CompatibilityConfig;
}

/**
 * Connection test response
 */
interface TestConnectionResponse {
  success: boolean;
  valid: boolean;
  message?: string;
  error?: string;
}

/**
 * API compatibility management
 */
export const compatibilityApi = {
  /**
   * Get compatibility config
   */
  get: (): Promise<CompatibilityResponse> =>
    apiClient.get<CompatibilityResponse>('/config/compatibility'),

  /**
   * Update compatibility config
   */
  update: (config: CompatibilityConfig): Promise<UpdateResponse> =>
    apiClient.put<UpdateResponse>('/config/compatibility', config),

  /**
   * Test port connection
   */
  testConnection: (port: number, type: 'ollama' | 'lmstudio'): Promise<TestConnectionResponse> =>
    apiClient.post<TestConnectionResponse>('/config/compatibility/test', { port, type }),
};
