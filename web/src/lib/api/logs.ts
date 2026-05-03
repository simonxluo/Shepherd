import { apiClient } from './client';
import type { LogFileInfo, ParsedLogEntry, LogFileFilter } from '@/types/logs';

interface LogFileContentResponse {
  entries: ParsedLogEntry[];
  total: number;
}

export async function listLogFiles(): Promise<LogFileInfo[]> {
  const res = await apiClient.get<{ success: boolean; data: LogFileInfo[] }>('/system/logs/files');
  return res.data ?? [];
}

export async function getLogFileContent(fileName: string, filter: LogFileFilter): Promise<LogFileContentResponse> {
  return apiClient.post<LogFileContentResponse>('/system/logs/content', { fileName, ...filter });
}

export async function getLogFileStats(fileName: string): Promise<Record<string, number>> {
  const res = await apiClient.get<{ success: boolean; data: Record<string, number> }>('/system/logs/stats', { fileName });
  return res.data ?? {};
}

export async function deleteLogFile(fileName: string): Promise<void> {
  await apiClient.post('/system/logs/delete', { fileName });
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

export function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}
