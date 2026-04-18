import type { UnifiedNode } from './node';

/**
 * Scan status
 */
export interface ScanStatus {
  running: boolean;
  found: DiscoveredClient[];
}

/**
 * Discovered client
 */
export interface DiscoveredClient {
  address: string;
  port: number;
  respondedAt: string;
}

/**
 * Cluster overview — matches backend GET /api/master/overview
 */
export interface ClusterOverview {
  totalClients: number;
  onlineClients: number;
  offlineClients: number;
  busyClients: number;
  totalTasks: number;
  pendingTasks: number;
  runningTasks: number;
  completedTasks: number;
  failedTasks: number;
  nodes?: {
    stats: {
      total: number;
      online: number;
      offline: number;
      busy: number;
    };
  };
}

/**
 * Client list response — matches backend GET /api/master/clients
 */
export interface ClientListResponse {
  clients: UnifiedNode[];
  total: number;
  stats?: {
    total: number;
    online: number;
    offline: number;
    busy: number;
  };
}
