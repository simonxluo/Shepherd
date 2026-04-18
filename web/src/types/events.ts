/**
 * SSE 事件类型
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
 * SSE 事件基础接口
 */
export interface SSEEvent<T = Record<string, unknown>> {
  type: SSEEventType;
  timestamp: number;
  data: T;
}


