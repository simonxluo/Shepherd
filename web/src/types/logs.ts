/**
 * 日志级别
 */
export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal';

/**
 * 日志来源
 */
export type LogSource = 'system' | 'model' | 'download' | 'cluster' | 'client';

/**
 * 日志条目
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
 * 日志过滤参数
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
 * 日志统计
 */
export interface LogStats {
  total: number;
  byLevel: Record<LogLevel, number>;
  bySource: Record<LogSource, number>;
}

/**
 * 日志文件信息
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
 * 日志文件内容响应
 */
export interface LogFileContent {
  entries: ParsedLogEntry[];
  count: number;
}

/**
 * 解析后的日志条目（从文件读取）
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
 * 日志文件过滤器
 */
export interface LogFileFilter {
  level?: string;
  search?: string;
  offset?: number;
  limit?: number;
}
