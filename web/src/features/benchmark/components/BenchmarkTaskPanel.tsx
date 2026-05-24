import { useTranslation } from 'react-i18next';
import { Loader2, CheckCircle2, XCircle, Clock, Ban, XCircle as CancelIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useCancelBenchmarkTask } from '../hooks/useBenchmarkState';
import type { BenchmarkTask, BenchmarkTaskStatus } from '@/types';

interface BenchmarkTaskPanelProps {
  tasks: BenchmarkTask[];
}

const statusConfig: Record<BenchmarkTaskStatus, { icon: typeof Loader2; color: string; labelKey: string }> = {
  pending:   { icon: Clock,        color: 'text-yellow-500', labelKey: 'benchmark.status.pending' },
  running:   { icon: Loader2,      color: 'text-blue-500',   labelKey: 'benchmark.status.running' },
  completed: { icon: CheckCircle2, color: 'text-green-500',  labelKey: 'benchmark.status.completed' },
  failed:    { icon: XCircle,      color: 'text-red-500',    labelKey: 'benchmark.status.failed' },
  cancelled: { icon: Ban,          color: 'text-gray-400',   labelKey: 'benchmark.status.cancelled' },
};

export function BenchmarkTaskPanel({ tasks }: BenchmarkTaskPanelProps) {
  const { t } = useTranslation();
  const cancelTask = useCancelBenchmarkTask();

  if (tasks.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm text-muted-foreground py-8">
        {t('benchmark.noTasks')}
      </div>
    );
  }

  return (
    <div className="space-y-1 p-2">
      {tasks.map((task) => {
        const config = statusConfig[task.status];
        const Icon = config.icon;
        const isRunning = task.status === 'running' || task.status === 'pending';

        return (
          <div
            key={task.id}
            className="flex items-center gap-2 px-2 py-1.5 rounded text-sm hover:bg-muted/50 group"
          >
            <Icon className={`w-3.5 h-3.5 flex-shrink-0 ${config.color} ${task.status === 'running' ? 'animate-spin' : ''}`} />
            <div className="flex-1 min-w-0">
              <div className="truncate text-foreground text-xs font-medium">
                {task.modelName || task.name}
              </div>
              <div className="text-[10px] text-muted-foreground">
                {t(config.labelKey)}
                {task.startedAt && (
                  <span className="ml-1">
                    {formatDuration(task.startedAt, task.finishedAt)}
                  </span>
                )}
              </div>
            </div>
            {isRunning && (
              <Button
                variant="ghost"
                size="icon"
                className="h-5 w-5 opacity-0 group-hover:opacity-100"
                onClick={() => cancelTask.mutate(task.id)}
              >
                <CancelIcon className="w-3 h-3" />
              </Button>
            )}
          </div>
        );
      })}
    </div>
  );
}

function formatDuration(start: string, end?: string): string {
  const startTime = new Date(start).getTime();
  const endTime = end ? new Date(end).getTime() : Date.now();
  const seconds = Math.floor((endTime - startTime) / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainSeconds = seconds % 60;
  return `${minutes}m ${remainSeconds}s`;
}
