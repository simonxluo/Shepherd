import { Pause, Play, X, RotateCcw, CloudDownload, CheckCircle2, XCircle, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import type { DownloadTask, DownloadState } from '@/types';

interface DownloadCardProps {
  task: DownloadTask;
  onPause?: () => void;
  onResume?: () => void;
  onCancel?: () => void;
  onRetry?: () => void;
}

/**
 * 下载状态颜色映射
 */
const STATE_COLORS: Record<DownloadState, string> = {
  idle: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  preparing: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  downloading: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  merging: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  verifying: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-300',
  completed: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  failed: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  paused: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
};

/**
 * 下载状态标签
 */
const STATE_LABELS: Record<DownloadState, string> = {
  idle: '等待中',
  preparing: '准备中',
  downloading: '下载中',
  merging: '合并中',
  verifying: '验证中',
  completed: '已完成',
  failed: '失败',
  paused: '已暂停',
};

/**
 * 格式化文件大小
 */
function formatSize(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${size.toFixed(2)} ${units[unitIndex]}`;
}

/**
 * 格式化速度
 */
function formatSpeed(bytesPerSecond: number): string {
  return `${formatSize(bytesPerSecond)}/s`;
}

/**
 * 格式化时间
 */
function formatTime(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}秒`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${Math.round(seconds % 60)}秒`;
  return `${Math.floor(seconds / 3600)}小时${Math.floor((seconds % 3600) / 60)}分`;
}

/**
 * 获取状态图标
 */
function getStateIcon(state: DownloadState) {
  switch (state) {
    case 'completed':
      return <CheckCircle2 className="w-5 h-5 text-green-500" />;
    case 'failed':
      return <XCircle className="w-5 h-5 text-red-500" />;
    case 'paused':
      return <Pause className="w-5 h-5 text-yellow-500" />;
    case 'downloading':
    case 'merging':
    case 'verifying':
    case 'preparing':
      return <CloudDownload className="w-5 h-5 text-blue-500 animate-pulse" />;
    default:
      return <AlertCircle className="w-5 h-5 text-muted-foreground" />;
  }
}

/**
 * 获取来源标签
 */
function getSourceLabel(source: 'huggingface' | 'modelscope'): string {
  return source === 'huggingface' ? 'HuggingFace' : 'ModelScope';
}

export function DownloadCard({ task, onPause, onResume, onCancel, onRetry }: DownloadCardProps) {
  const progressPercent = Math.round(task.progress * 100);
  const isActive = ['preparing', 'downloading', 'merging', 'verifying'].includes(task.state);
  const canPause = task.state === 'downloading';
  const canResume = task.state === 'paused';
  const canCancel = !task.completedAt && task.state !== 'completed';
  const canRetry = task.state === 'failed';

  return (
    <div className="bg-card rounded-lg border border-border p-4 hover:shadow-md transition-shadow">
      {/* 标题栏 */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          {getStateIcon(task.state)}
          <div className="flex-1 min-w-0">
            <h3 className="font-medium text-foreground truncate">
              {task.fileName || task.repoId}
            </h3>
            <p className="text-sm text-muted-foreground truncate">
              {task.repoId}
            </p>
          </div>
        </div>

        {/* 状态标签 */}
        <span className={cn('px-2 py-1 rounded-md text-xs font-medium shrink-0', STATE_COLORS[task.state])}>
          {STATE_LABELS[task.state]}
        </span>
      </div>

      {/* 来源标签 */}
      <div className="flex items-center gap-2 mb-3">
        <span className="px-2 py-0.5 bg-muted text-muted-foreground rounded text-xs">
          {getSourceLabel(task.source)}
        </span>
        <span className="text-xs text-muted-foreground">
          → {task.path}
        </span>
      </div>

      {/* 进度条 */}
      <div className="mb-3">
        <div className="flex items-center justify-between text-sm mb-1">
          <span className="text-muted-foreground">
            {formatSize(task.downloadedBytes)} / {formatSize(task.totalBytes)}
          </span>
          <span className="font-medium text-foreground">{progressPercent}%</span>
        </div>

        {/* 总体进度条 */}
        <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
          <div
            className={cn(
              'h-full transition-all duration-300',
              isActive ? 'bg-blue-500' : 'bg-gray-400'
            )}
            style={{ width: `${progressPercent}%` }}
          />
        </div>

        {/* 分块进度 */}
        {task.partsTotal > 1 && (
          <div className="text-xs text-muted-foreground mt-1">
            分块: {task.partsCompleted} / {task.partsTotal}
          </div>
        )}
      </div>

      {/* 速度和预计时间 */}
      {isActive && task.speed > 0 && (
        <div className="flex items-center gap-4 text-sm text-muted-foreground mb-3">
          <span>速度: {formatSpeed(task.speed)}</span>
          {task.eta > 0 && <span>预计: {formatTime(task.eta)}</span>}
        </div>
      )}

      {/* 错误信息 */}
      {task.state === 'failed' && task.error && (
        <div className="p-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-sm text-red-700 dark:text-red-300 mb-3">
          {task.error}
        </div>
      )}

      {/* 操作按钮 */}
      <div className="flex items-center gap-2">
        {canPause && (
          <Button
            onClick={onPause}
            variant="secondary"
            size="sm"
            className="border-yellow-500/50 text-yellow-700 hover:bg-yellow-200 hover:border-yellow-500 dark:bg-yellow-900/30 dark:text-yellow-400 dark:hover:bg-yellow-900/50"
          >
            <Pause className="w-4 h-4" />
            暂停
          </Button>
        )}

        {canResume && (
          <Button
            onClick={onResume}
            variant="default"
            size="sm"
            className="bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600"
          >
            <Play className="w-4 h-4" />
            继续
          </Button>
        )}

        {canCancel && (
          <Button
            onClick={onCancel}
            variant="destructive"
            size="sm"
          >
            <X className="w-4 h-4" />
            取消
          </Button>
        )}

        {canRetry && (
          <Button
            onClick={onRetry}
            variant="outline"
            size="sm"
          >
            <RotateCcw className="w-4 h-4" />
            重试
          </Button>
        )}

        {/* 创建时间 */}
        <div className="ml-auto text-xs text-muted-foreground">
          {new Date(task.createdAt).toLocaleString('zh-CN')}
        </div>
      </div>
    </div>
  );
}
