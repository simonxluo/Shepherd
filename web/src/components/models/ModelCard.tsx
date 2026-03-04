import { type ReactNode } from 'react';
import { Brain, Star, Loader2, Play, Square, Info, Share2, List, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ModelIcon } from '@/components/models/ModelIcon';
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
 * 模型状态颜色映射
 */
const STATUS_COLORS: Record<ModelStatus, string> = {
  stopped: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  loading: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  running: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900 dark:text-emerald-300',
  unloading: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
  error: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
};

/**
 * 模型状态标签
 */
const STATUS_LABELS: Record<ModelStatus, string> = {
  stopped: '已停止',
  loading: '加载中',
  running: '运行中',
  unloading: '卸载中',
  error: '错误',
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

export function ModelCard({ model, onLoad, onUnload, onToggleFavourite, onShowDetail, onEditAlias, actions }: ModelCardProps) {
  const statusColor = STATUS_COLORS[model.status];
  const statusLabel = STATUS_LABELS[model.status];
  const isLoading = model.status === 'loading' || model.isLoading;
  const isLoaded = model.status === 'running' || model.isLoaded;

  // 获取量化级别显示文本
  const quantizationLabel = model.metadata.fileTypeDescriptor || model.metadata.quantization || '未知';

  return (
    <div className="group flex items-center gap-4 px-4 py-3 bg-card hover:bg-accent/5 rounded-lg border border-border transition-all duration-200">
      {/* 左侧 - 模型信息区 */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
        {/* 收藏星标 */}
        <button
          onClick={onToggleFavourite}
          className={cn(
            'shrink-0 p-1 rounded-full hover:bg-accent transition-colors',
            model.favourite && 'text-yellow-500'
          )}
        >
          <Star className={cn('w-5 h-5', model.favourite && 'fill-current')} />
        </button>

        {/* 圆形模型图标 - 点击可编辑别名，悬停时发光 */}
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

        {/* 模型名称和元数据 */}
        <div className="flex-1 min-w-0">
          <h3 className="font-semibold text-base text-foreground truncate">
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

      {/* 右侧 - 操作区 */}
      <div className="flex items-center gap-2 shrink-0">
        {/* 状态指示器 */}
        <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <div className={cn(
            'w-2 h-2 rounded-full',
            isLoaded ? 'bg-green-500' : 'bg-gray-400'
          )} />
          <span className="whitespace-nowrap">{statusLabel}</span>
        </div>

        {/* 主操作按钮 */}
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

        {/* 次要操作按钮 */}
        <div className="flex items-center gap-1 ml-1">
          {/* 模型详情按钮 */}
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
