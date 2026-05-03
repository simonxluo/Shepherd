export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal';

export type LogSource = 'system' | 'model' | 'download' | 'cluster' | 'client';

export interface LogEntry {
  timestamp: number;
  level: LogLevel;
  source: LogSource;
  message: string;
  modelId?: string;
  clientId?: string;
  metadata?: Record<string, unknown>;
}

export interface LogFileInfo {
  name: string;
  path: string;
  size: number;
  modifiedAt: string;
  createdAt: string;
  role: string;
  isBackup?: boolean;
}

export interface ParsedLogEntry {
  timestamp: string;
  level: string;
  message: string;
  caller?: string;
  fields?: Record<string, unknown>;
}

export interface LogFileFilter {
  level?: string;
  search?: string;
  offset: number;
  limit: number;
}
