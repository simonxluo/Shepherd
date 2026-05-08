/**
 * Base SSE event interface
 */
export interface SSEEvent<T = Record<string, unknown>> {
  type:
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
  timestamp: number;
  data: T;
}
