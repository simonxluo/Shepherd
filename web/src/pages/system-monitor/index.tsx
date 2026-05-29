import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useSystemResources, useModelStatistics } from '@/features/system-monitor';
import { ResourceBar } from '@/features/system-monitor/components/ResourceBar';
import { GpuCard } from '@/features/system-monitor/components/GpuCard';
import { ModelStatsTable } from '@/features/system-monitor/components/ModelStatsTable';
import { formatBytes } from '@/lib/utils';

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h ${minutes}m`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

export function SystemMonitorPage() {
  const { t } = useTranslation();
  const { data: resources, isLoading: resourcesLoading } = useSystemResources();
  const { data: modelStats = [], isLoading: statsLoading } = useModelStatistics();

  if (resourcesLoading || statsLoading) {
    return <div className="flex items-center justify-center h-full">{t('common.loading')}</div>;
  }

  const cpuCores = resources ? Math.round(resources.cpu.total / 1000) : 0;

  return (
    <div className="space-y-6">
      {/* Page title */}
      <div>
        <h1 className="text-3xl font-bold text-foreground">{t('systemMonitor.title')}</h1>
        <p className="text-muted-foreground font-medium">{t('systemMonitor.subtitle')}</p>
      </div>

      {/* System info bar */}
      {resources && (
        <div className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted-foreground">
          {resources.hostname && <span>{resources.hostname}</span>}
          {resources.hostIp && <span>{resources.hostIp}</span>}
          {resources.platform && (
            <span>
              {resources.platform}/{resources.arch}
            </span>
          )}
          {resources.kernelVersion && <span>{resources.kernelVersion}</span>}
          {resources.rocmVersion && <span>ROCm {resources.rocmVersion}</span>}
          <span>
            {t('systemMonitor.uptime')}: {formatUptime(resources.uptime)}
          </span>
        </div>
      )}

      {/* System Resources */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('systemMonitor.systemResources')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {resources && (
            <>
              <ResourceBar
                label={`CPU (${cpuCores} ${t('systemMonitor.cores')}${resources.cpu.model ? ' - ' + resources.cpu.model : ''})`}
                value={resources.cpu.used}
                max={resources.cpu.total}
                percent={resources.cpu.percent}
                formatValue={(v) => `${(v / 1000).toFixed(1)} ${t('systemMonitor.cores')}`}
                formatMax={(v) => `${(v / 1000).toFixed(0)} ${t('systemMonitor.cores')}`}
              />
              <ResourceBar
                label={t('systemMonitor.memory')}
                value={resources.memory.used}
                max={resources.memory.total}
                percent={resources.memory.percent}
                formatValue={formatBytes}
                formatMax={formatBytes}
              />
              <ResourceBar
                label={t('systemMonitor.disk')}
                value={resources.disk.used}
                max={resources.disk.total}
                percent={resources.disk.percent}
                formatValue={formatBytes}
                formatMax={formatBytes}
              />
              {resources.loadAverage && resources.loadAverage.length >= 3 && (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">{t('systemMonitor.loadAverage')}</span>
                  <span className="font-mono">
                    {resources.loadAverage[0].toFixed(2)} / {resources.loadAverage[1].toFixed(2)} /{' '}
                    {resources.loadAverage[2].toFixed(2)}
                  </span>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* GPU Cards */}
      {resources && resources.gpu && resources.gpu.length > 0 && (
        <div>
          <h2 className="text-lg font-semibold mb-3">{t('systemMonitor.gpuDevices')}</h2>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {resources.gpu.map((gpu) => (
              <GpuCard key={gpu.index} gpu={gpu} />
            ))}
          </div>
        </div>
      )}

      {/* Model Statistics Table */}
      <ModelStatsTable models={modelStats} />
    </div>
  );
}
