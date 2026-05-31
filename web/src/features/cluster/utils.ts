import type { NodeResources, NodeStatus } from '@/types';
import type { TFunction } from 'react-i18next';

/**
 * Resource percentage calculation result
 */
export interface ResourcePercentages {
  cpu: number;
  memory: number;
  disk: number;
  gpu: number;
  gpuMemory: number;
}

/**
 * Calculate resource usage percentages from node resources
 */
export function getResourcePercentages(resources: NodeResources | null | undefined): ResourcePercentages {
  if (!resources) {
    return { cpu: 0, memory: 0, disk: 0, gpu: 0, gpuMemory: 0 };
  }

  return {
    cpu: resources.cpuPercent ?? 0,
    memory: resources.memoryTotal
      ? ((resources.memoryUsed || 0) / resources.memoryTotal) * 100
      : 0,
    disk: resources.diskTotal
      ? ((resources.diskUsed || 0) / resources.diskTotal) * 100
      : 0,
    gpu: resources.gpuPercent ?? 0,
    gpuMemory: resources.gpuMemoryUsed && resources.gpuMemoryTotal
      ? (resources.gpuMemoryUsed / resources.gpuMemoryTotal) * 100
      : 0,
  };
}

/**
 * Status color mapping for client cards
 */
export const STATUS_COLORS: Record<NodeStatus, string> = {
  online: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  offline: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  busy: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
  error: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  degraded: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300',
  disabled: 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400',
};

/**
 * Status label keys for i18n
 */
const STATUS_LABEL_KEYS: Record<NodeStatus, string> = {
  online: 'cluster.status.online',
  offline: 'cluster.status.offline',
  busy: 'cluster.status.busy',
  error: 'cluster.status.error',
  degraded: 'cluster.status.degraded',
  disabled: 'cluster.status.disabled',
};

/**
 * Get the i18n label for a given node status.
 * Replaces the former Proxy-based STATUS_LABELS to avoid calling
 * i18n.t() on every property access.
 */
export function getStatusLabel(status: string, t: TFunction): string {
  const key = STATUS_LABEL_KEYS[status as NodeStatus];
  return key ? t(key) : status;
}

/**
 * Detailed status configuration for the info dialog
 */
export function getStatusConfig(status: string, t: TFunction) {
  switch (status) {
    case 'online':
      return {
        color: 'text-emerald-600 dark:text-emerald-400',
        bg: 'bg-emerald-50 dark:bg-emerald-950/30',
        border: 'border-emerald-200 dark:border-emerald-800',
        label: t('cluster.status.online'),
        indicator: 'bg-emerald-500',
      };
    case 'busy':
      return {
        color: 'text-amber-600 dark:text-amber-400',
        bg: 'bg-amber-50 dark:bg-amber-950/30',
        border: 'border-amber-200 dark:border-amber-800',
        label: t('cluster.status.busy'),
        indicator: 'bg-amber-500',
      };
    case 'offline':
      return {
        color: 'text-slate-600 dark:text-slate-400',
        bg: 'bg-slate-50 dark:bg-slate-950/30',
        border: 'border-slate-200 dark:border-slate-800',
        label: t('cluster.status.offline'),
        indicator: 'bg-slate-500',
      };
    case 'error':
      return {
        color: 'text-red-600 dark:text-red-400',
        bg: 'bg-red-50 dark:bg-red-950/30',
        border: 'border-red-200 dark:border-red-800',
        label: t('cluster.status.error'),
        indicator: 'bg-red-500',
      };
    case 'degraded':
      return {
        color: 'text-orange-600 dark:text-orange-400',
        bg: 'bg-orange-50 dark:bg-orange-950/30',
        border: 'border-orange-200 dark:border-orange-800',
        label: t('cluster.status.degraded'),
        indicator: 'bg-orange-500',
      };
    case 'disabled':
      return {
        color: 'text-slate-600 dark:text-slate-400',
        bg: 'bg-slate-50 dark:bg-slate-950/30',
        border: 'border-slate-200 dark:border-slate-800',
        label: t('cluster.status.disabled'),
        indicator: 'bg-slate-500',
      };
    default:
      return {
        color: 'text-slate-600 dark:text-slate-400',
        bg: 'bg-slate-50 dark:bg-slate-950/30',
        border: 'border-slate-200 dark:border-slate-800',
        label: status,
        indicator: 'bg-slate-500',
      };
  }
}
