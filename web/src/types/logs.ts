/**
 * Log levels
 */
export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal';

/**
 * Log sources
 */
export type LogSource = 'system' | 'model' | 'download' | 'cluster' | 'client';

/**
 * Log entry
 */
export interface LogEntry {
  timestamp: number;
  level: LogLevel;
  source: LogSource;
  message: string;
  modelId?: string;
  clientId?: string;
  metadata?: Record<string, unknown>;
}

/**
 * Log filter parameters
 */
export interface LogFilters {
  level?: LogLevel;
  source?: LogSource;
  search?: string;
  modelId?: string;
  clientId?: string;
  startTime?: number;
  endTime?: number;
}

/**
 * Log statistics
 */
export interface LogStats {
  total: number;
  byLevel: Record<LogLevel, number>;
  bySource: Record<LogSource, number>;
}

/**
 * Log file info
 */
export interface LogFileInfo {
  name: string;
  path: string;
  size: number;
  role: string; // Node role: master, client, hybrid
  date: string;
  createdAt: string; // ISO 8601
  isBackup: boolean;
}

/**
 * Log file content response
 */
export interface LogFileContent {
  entries: ParsedLogEntry[];
  count: number;
}

/**
 * Parsed log entry (from file)
 */
export interface ParsedLogEntry {
  timestamp: string;
  level: string;
  message: string;
  caller?: string;
  fields?: Record<string, unknown>;
  raw: string;
}

/**
 * Log file filter
 */
export interface LogFileFilter {
  level?: string;
  search?: string;
  offset?: number;
  limit?: number;
}
