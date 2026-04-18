import { type ReactNode } from 'react';
import { Star, Loader2, Play, Square, Info } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ModelIcon } from '@/features/models/components/ModelIcon';
import { cn } from '@/lib/utils';
import type { Model, ModelStatus } from '@/types';

interface ModelCardProps {
  model: Model;
  onLoad?: () => void;
  onUnload?: () => void;
  onToggleFavourite?: () => void;
  onShowDetail?: () => void;
  onEditAlias?: () => void;
  actions?: ReactNode;
}

/**
 * Model status color mapping
 */
const STATUS_COLORS: Record<ModelStatus, string> = {
  stopped: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  loading: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  running: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900 dark:text-emerald-300',
  unloading: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
  error: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
};

/**
 * Model status labels
 */
const STATUS_LABELS: Record<ModelStatus, string> = {
  stopped: '已停止',
  loading: '加载中',
  running: '运行中',
  unloading: '卸载中',
  error: '错误',
};

/**
 * Format file size
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

export function ModelCard({ model, onLoad, onUnload, onToggleFavourite, onShowDetail, onEditAlias, actions }: ModelCardProps) {
  const statusColor = STATUS_COLORS[model.status];
  const statusLabel = STATUS_LABELS[model.status];
  const isLoading = model.status === 'loading' || model.isLoading;
  const isLoaded = model.status === 'running' || model.isLoaded;

  const quantizationLabel = model.metadata.fileTypeDescriptor || model.metadata.quantization || '未知';

  return (
    <div className="group flex items-center gap-4 px-4 py-4 bg-card hover:bg-accent/5 rounded-lg border border-border transition-all duration-200">
      {/* Left - model info */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
        {/* Favourite star */}
        <button
          onClick={onToggleFavourite}
          className={cn(
            'shrink-0 p-1 rounded-full hover:bg-accent transition-colors',
            model.favourite && 'text-yellow-500'
          )}
        >
          <Star className={cn('w-5 h-5', model.favourite && 'fill-current')} />
        </button>

        {/* Model icon - click to edit alias */}
        <button
          onClick={onEditAlias}
          className={cn(
            'shrink-0 w-10 h-10 rounded-full flex items-center justify-center',
            'bg-muted hover:bg-accent transition-all duration-200',
            'relative overflow-hidden'
          )}
          title="编辑别名"
        >
          <ModelIcon
            architecture={model.metadata.architecture}
            className="w-5 h-5"
          />
        </button>

        {/* Model name and metadata */}
        <div className="flex-1 min-w-0">
          <h3 className="font-bold text-base text-foreground truncate">
            {model.alias || model.displayName || model.name}
          </h3>
          <div className="flex items-center gap-2 text-sm text-muted-foreground mt-0.5">
            <span className="truncate">{model.metadata.architecture}</span>
            <span className="shrink-0">|</span>
            <span className="truncate">{quantizationLabel}</span>
            <span className="shrink-0">|</span>
            <span className="shrink-0">{formatSize(model.totalSize ?? model.size)}</span>
          </div>
        </div>
      </div>

      {/* Right - actions */}
      <div className="flex items-center gap-2 shrink-0">
        {/* Status indicator */}
        <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <div className={cn(
            'w-2 h-2 rounded-full',
            isLoaded ? 'bg-green-500' : 'bg-gray-400'
          )} />
          <span className="whitespace-nowrap">{statusLabel}</span>
        </div>

        {/* Primary action button */}
        {!isLoaded ? (
          <Button
            onClick={onLoad}
            disabled={isLoading}
            variant="default"
            className="min-w-[100px] h-9 px-4 justify-center rounded-lg"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span className="ml-1.5">加载中</span>
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                <span className="ml-1.5">启动</span>
              </>
            )}
          </Button>
        ) : (
          <Button
            onClick={onUnload}
            disabled={model.status === 'unloading'}
            variant="destructive"
            className="min-w-[100px] h-9 px-4 justify-center rounded-lg"
          >
            {model.status === 'unloading' ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span className="ml-1.5">卸载中</span>
              </>
            ) : (
              <>
                <Square className="w-4 h-4" />
                <span className="ml-1.5">停止</span>
              </>
            )}
          </Button>
        )}

        {/* Secondary actions */}
        <div className="flex items-center gap-1 ml-1">
          {/* Model detail button */}
          <Button
            onClick={onShowDetail}
            variant="ghost"
            size="icon"
            title="模型详情"
            className="h-8 w-8 sm:h-9 sm:w-9"
          >
            <Info className="w-3 h-3 sm:w-4 sm:h-4" />
          </Button>
          {actions}
        </div>
      </div>
    </div>
  );
}
