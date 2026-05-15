import { Pause, Play, X, CloudDownload, CheckCircle2, XCircle, AlertCircle, AlertTriangle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { cn, formatBytes } from '@/lib/utils';
import type { DownloadTask, DownloadState } from '@/types';
import { ACTIVE_DOWNLOAD_STATES } from '@/features/downloads/hooks';

interface DownloadCardProps {
  task: DownloadTask;
  onPause?: () => void;
  onResume?: () => void;
  onCancel?: () => void;
}

/**
 * Download state color mapping
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
 * Download state labels (i18n keys)
 */
const STATE_LABEL_KEYS: Record<DownloadState, string> = {
  idle: 'downloads.stateLabels.idle',
  preparing: 'downloads.stateLabels.preparing',
  downloading: 'downloads.stateLabels.downloading',
  merging: 'downloads.stateLabels.merging',
  verifying: 'downloads.stateLabels.verifying',
  completed: 'downloads.stateLabels.completed',
  failed: 'downloads.stateLabels.failed',
  paused: 'downloads.stateLabels.paused',
};

/**
 * Format speed
 */
function formatSpeed(bytesPerSecond: number): string {
  return `${formatBytes(bytesPerSecond)}/s`;
}

/**
 * Format time
 */
function formatTime(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${Math.round(seconds % 60)}s`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
}

/**
 * Get state icon
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
 * Get source label
 */
function getSourceLabel(source: 'huggingface' | 'modelscope'): string {
  return source === 'huggingface' ? 'HuggingFace' : 'ModelScope';
}

export function DownloadCard({ task, onPause, onResume, onCancel }: DownloadCardProps) {
  const { t } = useTranslation();
  const progressPercent = Math.round(task.progress * 100);
  const isActive = ACTIVE_DOWNLOAD_STATES.includes(task.state);
  const canPause = task.state === 'downloading';
  const canResume = task.state === 'paused';
  const canCancel = !task.completedAt && task.state !== 'completed';

  return (
    <div className="bg-card rounded-lg border border-border p-4 hover:shadow-md transition-shadow">
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

        <span className={cn('px-2 py-1 rounded-md text-xs font-medium shrink-0', STATE_COLORS[task.state])}>
          {t(STATE_LABEL_KEYS[task.state])}
        </span>
      </div>

      <div className="flex items-center gap-2 mb-3">
        <span className="px-2 py-0.5 bg-muted text-muted-foreground rounded text-xs">
          {getSourceLabel(task.source)}
        </span>
        <span className="text-xs text-muted-foreground">
          → {task.path}
        </span>
      </div>

      <div className="mb-3">
        <div className="flex items-center justify-between text-sm mb-1">
          <span className="text-muted-foreground">
            {formatBytes(task.downloadedBytes)} / {formatBytes(task.totalBytes)}
          </span>
          <span className="font-medium text-foreground">{progressPercent}%</span>
        </div>

        <Progress
          value={progressPercent}
          className={cn(
            'h-2',
            !isActive && '[&>[data-slot=progress-indicator]]:bg-gray-400'
          )}
        />

        {task.partsTotal > 1 && (
          <div className="text-xs text-muted-foreground mt-1">
            {t('downloads.parts')}: {task.partsCompleted} / {task.partsTotal}
          </div>
        )}
      </div>

      {isActive && task.speed > 0 && (
        <div className="flex items-center gap-4 text-sm text-muted-foreground mb-3">
          <span>{t('downloads.speed')}: {formatSpeed(task.speed)}</span>
          {task.eta > 0 && <span>{t('downloads.eta')}: {formatTime(task.eta)}</span>}
        </div>
      )}

      {task.state === 'failed' && task.error && (
        <Alert variant="destructive" className="mb-3 py-2">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription className="text-sm">{task.error}</AlertDescription>
        </Alert>
      )}

      <div className="flex items-center gap-2">
        {canPause && (
          <Button
            onClick={onPause}
            variant="secondary"
            size="sm"
            className="border-yellow-500/50 text-yellow-700 hover:bg-yellow-200 hover:border-yellow-500 dark:bg-yellow-900/30 dark:text-yellow-400 dark:hover:bg-yellow-900/50"
          >
            <Pause className="w-4 h-4" />
            {t('downloads.pause')}
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
            {t('downloads.resume')}
          </Button>
        )}

        {canCancel && (
          <Button
            onClick={onCancel}
            variant="destructive"
            size="sm"
          >
            <X className="w-4 h-4" />
            {t('downloads.cancel')}
          </Button>
        )}

        <div className="ml-auto text-xs text-muted-foreground">
          {new Date(task.createdAt).toLocaleString()}
        </div>
      </div>
    </div>
  );
}
