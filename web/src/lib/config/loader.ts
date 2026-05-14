import { useState, useEffect } from 'react';
import yaml from 'js-yaml';
import type { AppConfig } from './types';

/**
 * Default configuration
 */
const DEFAULT_CONFIG: AppConfig = {
  api: {
    baseUrl: '',
    basePath: '/api',
    timeout: 30000,
    connectTimeout: 5000,
    retryCount: 3,
    retryDelay: 1000,
  },
  sse: {
    endpoint: '/events',
    reconnect: true,
    reconnectDelay: 3000,
    maxReconnectAttempts: 10,
    connectionTimeout: 30000,
    heartbeatEnabled: true,
    heartbeatInterval: 30000,
  },
  features: {
    models: true,
    downloads: true,
    cluster: true,
    logs: true,
    chat: true,
    multimodal: true,
    settings: true,
    dashboard: true,
  },
  ui: {
    theme: 'auto',
    language: 'zh-CN',
    pageSize: 10,
    pageSizeOptions: [10, 20, 50, 100],
    virtualScrollThreshold: 100,
    animations: true,
    skeleton: true,
    breadcrumb: true,
    sidebarExpanded: true,
    compactMode: false,
  },
  logging: {
    level: 'info',
    console: true,
    remote: false,
    remoteEndpoint: '',
    batchSize: 100,
    flushInterval: 5000,
  },
  cache: {
    modelsTTL: 300000,
    clientsTTL: 60000,
    downloadsTTL: 10000,
    configTTL: 300000,
    persistent: true,
    prefix: 'shepherd:',
    versioning: true,
  },
  openai: {},
  performance: {},
  server: {
    masterAddress: '',
    clientName: '',
  },
};

export class ConfigLoader {
  private cachedConfig: AppConfig | null = null;

  /**
   * Load configuration file. Caches the result so subsequent calls
   * return the same config without re-fetching.
   */
  async load(): Promise<AppConfig> {
    if (this.cachedConfig) {
      return this.cachedConfig;
    }

    try {
      const response = await fetch('/config.yaml')
      if (!response.ok) {
        throw new Error(`Failed to load config: ${response.status}`)
      }
      const yamlText = await response.text()
      const parsed = yaml.load(yamlText) as Record<string, unknown>

      // Merge default config with parsed config
      this.cachedConfig = this.mergeConfig(DEFAULT_CONFIG, parsed)
    } catch (error) {
      console.warn('Failed to load config.yaml, using default config:', error)
      this.cachedConfig = DEFAULT_CONFIG
    }

    return this.cachedConfig;
  }

  /**
   * Merge configuration
   */
  private mergeConfig(defaults: AppConfig, loaded: Record<string, any>): AppConfig {
    // Convert backend.urls array format to api.baseUrl single value
    let apiBaseUrl = defaults.api.baseUrl;
    if (loaded?.backend?.urls && Array.isArray(loaded.backend.urls)) {
      const index = loaded.backend.currentIndex ?? 0;
      apiBaseUrl = loaded.backend.urls[index] ?? loaded.backend.urls[0] ?? defaults.api.baseUrl;
    } else if (loaded?.api?.baseUrl) {
      apiBaseUrl = loaded.api.baseUrl;
    }

    return {
      api: {
        ...defaults.api,
        ...loaded?.api,
        baseUrl: apiBaseUrl,
      },
      sse: { ...defaults.sse, ...loaded?.sse },
      features: { ...defaults.features, ...loaded?.features },
      ui: { ...defaults.ui, ...loaded?.ui },
      logging: { ...defaults.logging, ...loaded?.logging },
      cache: { ...defaults.cache, ...loaded?.cache },
      openai: loaded?.openai ? { ...defaults.openai, ...loaded.openai } : defaults.openai,
      performance: loaded?.performance ? { ...defaults.performance, ...loaded.performance } : defaults.performance,
      server: { ...defaults.server, ...loaded?.server },
    }
  }
}

/**
 * Config loader singleton
 */
export const configLoader = new ConfigLoader();

/**
 * React hook for accessing app configuration.
 *
 * @example
 * const config = useConfig();
 * console.log(config.api.baseUrl);
 */
export function useConfig(): AppConfig {
  const [config, setConfig] = useState<AppConfig>(DEFAULT_CONFIG);

  useEffect(() => {
    let mounted = true;

    configLoader.load().then((loadedConfig) => {
      if (mounted) {
        setConfig(loadedConfig);
      }
    });

    return () => {
      mounted = false;
    };
  }, []);

  return config;
}
