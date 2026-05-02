/**
 * Frontend configuration type definitions.
 * Standalone file to avoid circular dependencies.
 */

/**
 * API configuration
 */
export interface ApiConfig {
  baseUrl: string;
  basePath: string;
  timeout: number;
  connectTimeout: number;
  retryCount: number;
  retryDelay: number;
}

/**
 * SSE configuration
 */
export interface SseConfig {
  endpoint: string;
  reconnect: boolean;
  reconnectDelay: number;
  maxReconnectAttempts: number;
  connectionTimeout: number;
  heartbeatEnabled: boolean;
  heartbeatInterval: number;
}

/**
 * Feature flags
 */
export interface FeaturesConfig {
  models: boolean;
  downloads: boolean;
  cluster: boolean;
  logs: boolean;
  chat: boolean;
  multimodal: boolean;
  settings: boolean;
  dashboard: boolean;
}

/**
 * UI configuration
 */
export interface UiConfig {
  theme: 'light' | 'dark' | 'auto';
  language: string;
  pageSize: number;
  pageSizeOptions: number[];
  virtualScrollThreshold: number;
  animations: boolean;
  skeleton: boolean;
  breadcrumb: boolean;
  sidebarExpanded: boolean;
  compactMode: boolean;
}

/**
 * Logging configuration
 */
export interface LoggingConfig {
  level: 'debug' | 'info' | 'warn' | 'error';
  console: boolean;
  remote: boolean;
  remoteEndpoint: string;
  batchSize: number;
  flushInterval: number;
}

/**
 * Cache configuration
 */
export interface CacheConfig {
  modelsTTL: number;
  clientsTTL: number;
  downloadsTTL: number;
  configTTL: number;
  persistent: boolean;
  prefix: string;
  versioning: boolean;
}

/**
 * OpenAI configuration
 */
export interface OpenAIConfig {
  endpoint: string;
  defaultModel: string;
  temperature: number;
  maxTokens: number;
  topP: number;
  frequencyPenalty: number;
  presencePenalty: number;
  streamTimeout: number;
}

/**
 * Performance configuration
 */
export interface PerformanceConfig {
  monitoring: boolean;
  sampleRate: number;
  preloading: boolean;
  virtualScroll: boolean;
  codeSplitting: boolean;
  lazyImageThreshold?: number;
  preloadResources?: string[];
}

/**
 * Server mode configuration
 */
export interface ServerModeConfig {
  // mode field removed; use backend node.role instead
  masterAddress?: string;
  clientName?: string;
}

/**
 * Application configuration
 */
export interface AppConfig {
  api: ApiConfig;
  sse: SseConfig;
  features: FeaturesConfig;
  ui: UiConfig;
  logging: LoggingConfig;
  cache: CacheConfig;
  openai?: OpenAIConfig;
  performance?: PerformanceConfig;
  server: ServerModeConfig;
}

/**
 * Server configuration (backward compatibility)
 */
export interface ServerConfig {
  host: string;
  port: number;
  https: boolean;
  cors: {
    enabled: boolean;
    origin: string;
    methods: string;
    headers: string;
    credentials: boolean;
  };
}

/**
 * llama.cpp path configuration
 */
export interface LlamaCppPathConfig {
  name: string;
  path: string;
  description: string;
}

/**
 * Model path configuration
 */
export interface ModelPathConfig {
  path: string;
  name?: string;
  description?: string;
}

/**
 * Path list response
 */
export interface PathListResponse<T> {
  success: boolean;
  data: {
    items: T[];
    count: number;
  };
}
