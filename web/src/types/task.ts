/**
 * Task-related type definitions.
 * Migrated from cluster.ts to avoid circular dependencies.
 */

/**
 * Task type — matches backend cluster.TaskType
 */
export type TaskType = 'load_model' | 'unload_model' | 'run_python' | 'run_llamacpp' | 'custom';

/**
 * Task status — matches backend cluster.TaskStatus
 */
export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

/**
 * Schedule strategy — matches backend scheduler.SchedulingStrategy
 */
export type ScheduleStrategy = 'round_robin' | 'least_loaded' | 'resource_aware';

/**
 * Cluster task — matches backend cluster.Task struct
 */
export interface ClusterTask {
  id: string;
  type: TaskType;
  payload: Record<string, unknown>;
  assignedTo?: string;  // Assigned client ID
  status: TaskStatus;
  createdAt: string;    // ISO 8601
  startedAt?: string;   // ISO 8601
  completedAt?: string; // ISO 8601
  result?: Record<string, unknown>;
  error?: string;
  retryCount?: number;  // Optional; not in backend Task struct
  maxRetries?: number;  // Optional; not in backend Task struct
}

/**
 * Task list response
 */
export interface TaskListResponse {
  tasks: ClusterTask[];
  total: number;
}
