import type { UnifiedNode } from './node';

/**
 * 扫描状态
 */
export interface ScanStatus {
  running: boolean;
  found: DiscoveredClient[];
}

/**
 * 发现的客户端
 */
export interface DiscoveredClient {
  address: string;
  port: number;
  respondedAt: string;
}

/**
 * 集群概览 - 匹配后端 GET /api/master/overview 响应格式
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
 * 客户端列表响应 - 匹配后端 GET /api/master/clients 响应格式
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
