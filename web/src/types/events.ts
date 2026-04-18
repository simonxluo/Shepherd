/**
 * SSE event types
 */
export type SSEEventType =
  | 'heartbeat'
  | 'systemStatus'
  | 'modelLoadStart'
  | 'modelLoad'
  | 'modelStop'
  | 'modelSlots'
  | 'console'
  | 'download_progress'
  | 'download_status'
  | 'scan_progress'
  | 'scan_complete'
  | 'clientRegistered'
  | 'clientDisconnected'
  | 'clientResourcesUpdated'
  | 'taskUpdate';

/**
 * Base SSE event interface
 */
export interface SSEEvent<T = Record<string, unknown>> {
  type: SSEEventType;
  timestamp: number;
  data: T;
}
