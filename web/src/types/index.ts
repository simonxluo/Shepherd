/**
 * 导出所有类型定义
 * Export all type definitions
 */

export * from './model';
export * from './download';
export * from './events';
export * from './logs';
export * from './websocket';

// 统一导出节点类型（推荐）
// Export unified node types (recommended)
export type * from './node';

// 任务相关类型（从 cluster.ts 迁移至此）
// Task-related types (migrated from cluster.ts)
export type {
  TaskType,
  TaskStatus,
  ScheduleStrategy,
  ClusterTask,
  TaskListResponse,
} from './task';

// Cluster 特有类型（ScanStatus、ClusterOverview 等）
// Cluster-specific types (ScanStatus, ClusterOverview, etc.)
export type {
  ScanStatus,
  ClusterOverview,
  ClientListResponse,
} from './cluster';

// 向后兼容：导出 ClientStatus 别名（使用 NodeStatus）
// Backward compatibility: export ClientStatus alias (using NodeStatus)
export type { NodeStatus as ClientStatus } from './node';

// 默认导出
export type { GPUInfo } from './node';
