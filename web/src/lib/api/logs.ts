import { apiClient } from '@/lib/api/client';
import type { LogFileInfo, LogFileContent, LogFileFilter } from '@/types/logs';

/**
 * API 响应包装类型
 */
interface ApiResponse<T> {
  success: boolean;
  data: T;
  metadata?: {
    timestamp: string;
    requestId: string;
  };
}

/**
 * 获取所有日志文件列表
 */
export async function listLogFiles(): Promise<LogFileInfo[]> {
  const response = await apiClient.get<ApiResponse<{ files: LogFileInfo[]; count: number }>>('/logs/files');
  return response.data?.files ?? [];
}

/**
 * 获取日志文件内容
 */
export async function getLogFileContent(
  filename: string,
  filter?: LogFileFilter
): Promise<LogFileContent> {
  const params: Record<string, string> = {};

  if (filter?.level) params.level = filter.level;
  if (filter?.search) params.search = filter.search;
  if (filter?.offset) params.offset = filter.offset.toString();
  if (filter?.limit) params.limit = filter.limit.toString();

  const response = await apiClient.get<ApiResponse<LogFileContent>>(
    `/logs/files/${encodeURIComponent(filename)}`,
    params
  );
  return response.data ?? { entries: [], count: 0 };
}

/**
 * 获取日志文件统计信息
 */
export async function getLogFileStats(filename: string): Promise<Record<string, number>> {
  const response = await apiClient.get<ApiResponse<Record<string, number>>>(
    `/logs/files/${encodeURIComponent(filename)}/stats`
  );
  return response.data ?? {};
}

/**
 * 删除日志文件
 */
export async function deleteLogFile(filename: string): Promise<{ message: string }> {
  const response = await apiClient.delete<ApiResponse<{ message: string }>>(
    `/logs/files/${encodeURIComponent(filename)}`
  );
  return response.data ?? { message: '删除成功' };
}

/**
 * 格式化文件大小
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';

  const units = ['B', 'KB', 'MB', 'GB'];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const size = bytes / Math.pow(k, i);

  return `${size.toFixed(2)} ${units[i]}`;
}

/**
 * 格式化日期
 */
export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}
