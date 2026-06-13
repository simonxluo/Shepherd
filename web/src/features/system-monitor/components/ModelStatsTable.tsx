import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { cn } from '@/lib/utils';
import type { ModelStats } from '../types';

interface ModelStatsTableProps {
  models: ModelStats[];
}

function StatusBadge({ state }: { state: string }) {
  const colorMap: Record<string, string> = {
    running: 'bg-green-500/15 text-green-700 dark:text-green-400',
    loading: 'bg-yellow-500/15 text-yellow-700 dark:text-yellow-400',
    error: 'bg-red-500/15 text-red-700 dark:text-red-400',
    unloading: 'bg-orange-500/15 text-orange-700 dark:text-orange-400',
    stopped: 'bg-gray-500/15 text-gray-700 dark:text-gray-400',
  };

  return (
    <span className={cn('inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium', colorMap[state] || colorMap.stopped)}>
      {state}
    </span>
  );
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  return `${Math.floor(seconds / 86400)}d ${Math.floor((seconds % 86400) / 3600)}h`;
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toString();
}

export function ModelStatsTable({ models }: ModelStatsTableProps) {
  const { t } = useTranslation();

  const sortedModels = useMemo(
    () => [...models].sort((a, b) => b.requestCount - a.requestCount),
    [models]
  );

  if (sortedModels.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('systemMonitor.modelStats')}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-center text-muted-foreground py-8">
            {t('systemMonitor.noModelsLoaded')}
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t('systemMonitor.modelStats')}</CardTitle>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.model')}</th>
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.status')}</th>
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.backend')}</th>
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.port')}</th>
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.uptime')}</th>
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.requests')}</th>
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.promptTokens')}</th>
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.completionTokens')}</th>
              <th className="pb-2 pr-4 font-medium text-muted-foreground">{t('systemMonitor.table.avgLatency')}</th>
              <th className="pb-2 font-medium text-muted-foreground">{t('systemMonitor.table.inflight')}</th>
            </tr>
          </thead>
          <tbody>
            {sortedModels.map((model) => (
              <tr key={model.instanceId || model.modelId} className="border-b last:border-0">
                <td className="py-2 pr-4 font-medium truncate max-w-[200px]" title={model.modelName}>
                  {model.modelName}
                </td>
                <td className="py-2 pr-4">
                  <StatusBadge state={model.state} />
                </td>
                <td className="py-2 pr-4 text-muted-foreground">{model.pluginId || '-'}</td>
                <td className="py-2 pr-4 font-mono text-muted-foreground">{model.port || '-'}</td>
                <td className="py-2 pr-4 text-muted-foreground">{formatDuration(model.uptimeSeconds)}</td>
                <td className="py-2 pr-4">
                  {formatNumber(model.requestCount)}
                  {model.errorCount > 0 && (
                    <span className="ml-1 text-red-500 text-xs">({model.errorCount} err)</span>
                  )}
                </td>
                <td className="py-2 pr-4 text-muted-foreground">{formatNumber(model.totalPromptTokens)}</td>
                <td className="py-2 pr-4 text-muted-foreground">{formatNumber(model.totalCompletionTokens)}</td>
                <td className="py-2 pr-4 text-muted-foreground">
                  {model.avgLatencyMs > 0 ? `${model.avgLatencyMs.toFixed(0)}ms` : '-'}
                </td>
                <td className="py-2 font-medium">
                  {model.inflightCount > 0 ? (
                    <span className="text-blue-600 dark:text-blue-400">{model.inflightCount}</span>
                  ) : (
                    <span className="text-muted-foreground">0</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  );
}
