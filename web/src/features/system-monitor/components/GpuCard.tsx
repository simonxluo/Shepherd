import { useTranslation } from 'react-i18next';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ResourceBar } from './ResourceBar';
import { formatBytes } from '@/lib/utils';
import type { GpuInfo } from '../types';

interface GpuCardProps {
  gpu: GpuInfo;
}

export function GpuCard({ gpu }: GpuCardProps) {
  const { t } = useTranslation();
  const memPercent =
    gpu.memoryTotal > 0 ? (gpu.memoryUsed / gpu.memoryTotal) * 100 : 0;

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center justify-between">
          <span>
            GPU {gpu.index}: {gpu.name}
          </span>
          <span className="text-xs text-muted-foreground font-normal">{gpu.vendor}</span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <ResourceBar
          label={t('systemMonitor.vram')}
          value={gpu.memoryUsed}
          max={gpu.memoryTotal}
          percent={memPercent}
          formatValue={formatBytes}
          formatMax={formatBytes}
        />

        {gpu.utilization > 0 && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t('systemMonitor.utilization')}</span>
            <span className="font-medium">{gpu.utilization.toFixed(1)}%</span>
          </div>
        )}

        {gpu.temperature > 0 && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t('systemMonitor.temperature')}</span>
            <span className="font-medium">{gpu.temperature.toFixed(0)}°C</span>
          </div>
        )}

        {gpu.powerUsage > 0 && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t('systemMonitor.power')}</span>
            <span className="font-medium">{gpu.powerUsage.toFixed(1)} W</span>
          </div>
        )}

        {gpu.driverVersion && (
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">{t('systemMonitor.driver')}</span>
            <span className="font-mono text-xs">{gpu.driverVersion}</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
