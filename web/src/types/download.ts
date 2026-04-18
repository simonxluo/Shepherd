/**
 * Download state
 */
export type DownloadState =
  | 'idle'
  | 'preparing'
  | 'downloading'
  | 'merging'
  | 'verifying'
  | 'completed'
  | 'failed'
  | 'paused';

/**
 * Download source
 */
export type DownloadSource = 'huggingface' | 'modelscope';

/**
 * Download task
 */
export interface DownloadTask {
  id: string;
  source: DownloadSource;
  repoId: string;
  fileName: string;
  path: string;
  state: DownloadState;
  downloadedBytes: number;
  totalBytes: number;
  partsCompleted: number;
  partsTotal: number;
  progress: number; // 0-1
  speed: number; // bytes per second
  eta: number; // seconds remaining
  error?: string;
  createdAt: string;
  completedAt?: string;
}

/**
 * Create download task parameters
 */
export interface CreateDownloadParams {
  source: DownloadSource;
  repoId: string;
  fileName?: string;
  path?: string;
  maxRetries?: number;
  chunkSize?: number;
}

/**
 * Download list response
 */
export interface DownloadListResponse {
  downloads: DownloadTask[];
  total: number;
}
