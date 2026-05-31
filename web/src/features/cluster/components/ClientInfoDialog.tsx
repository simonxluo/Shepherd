import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Cpu,
  HardDrive,
  Server,
  Thermometer,
  Zap,
  Activity,
  Terminal,
  Monitor,
  Clock,
  Globe,
  Microchip,
  Layers,
  Hash,
  Power,
  Gauge,
  Settings,
  FileCode,
  FolderOpen,
  Play,
  CheckCircle,
  XCircle,
  Loader2,
  AlertTriangle,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { useClient, useNodeConfig, useTestNodeLlamacpp } from '@/features/cluster/hooks';
import { cn, formatBytes } from '@/lib/utils';
import { getStatusConfig, getResourcePercentages } from '../utils';
import type { Client, GPUInfo, LlamacppTestResult } from '@/types';
import { toast } from '@/hooks/useToast';

interface ClientInfoDialogProps {
  client: Client | null;
  open: boolean;
  onClose: () => void;
}

/**
 * Circular progress bar component
 */
function CircularProgress({
  value,
  size = 80,
  strokeWidth = 8,
  color = 'stroke-blue-500',
  bgColor = 'stroke-slate-100 dark:stroke-slate-800',
  label,
  subLabel,
}: {
  value: number;
  size?: number;
  strokeWidth?: number;
  color?: string;
  bgColor?: string;
  label: string;
  subLabel?: string;
}) {
  const radius = (size - strokeWidth) / 2;
  const circumference = radius * 2 * Math.PI;
  const offset = circumference - (value / 100) * circumference;

  return (
    <div className="flex flex-col items-center">
      <div className="relative" style={{ width: size, height: size }}>
        <svg
          className="transform -rotate-90"
          width={size}
          height={size}
        >
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            strokeWidth={strokeWidth}
            className={bgColor}
          />
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            className={cn('transition-all duration-500 ease-out', color)}
            style={{
              strokeDasharray: circumference,
              strokeDashoffset: offset,
            }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-lg font-bold">{value.toFixed(0)}%</span>
          {subLabel && (
            <span className="text-[10px] text-muted-foreground">{subLabel}</span>
          )}
        </div>
      </div>
      <span className="mt-2 text-sm font-medium text-muted-foreground">{label}</span>
    </div>
  );
}

/**
 * Resource metric card
 */
function MetricCard({
  icon: Icon,
  title,
  value,
  subValue,
  colorClass,
}: {
  icon: React.ElementType;
  title: string;
  value: string;
  subValue?: string;
  colorClass?: string;
}) {
  return (
    <div className="relative overflow-hidden rounded-xl border border-border/50 bg-gradient-to-br from-card to-card/50 p-4 transition-all hover:shadow-md hover:border-border">
      <div className={cn('absolute top-0 right-0 w-20 h-20 opacity-5', colorClass)}>
        <Icon className="w-full h-full" />
      </div>
      <div className="relative z-10">
        <div className="flex items-center gap-2 mb-2">
          <div className={cn('p-1.5 rounded-lg', colorClass?.replace('text-', 'bg-').replace('600', '100').replace('400', '900/30'))}>
            <Icon className={cn('w-4 h-4', colorClass)} />
          </div>
          <span className="text-sm text-muted-foreground">{title}</span>
        </div>
        <div className="text-2xl font-bold tracking-tight">{value}</div>
        {subValue && (
          <div className="text-xs text-muted-foreground mt-1">{subValue}</div>
        )}
      </div>
    </div>
  );
}

/**
 * Header stat card
 */
interface HeaderStatCardProps {
  icon: React.ElementType;
  label: string;
  value: string | number;
  color: 'blue' | 'green' | 'purple' | 'amber' | 'red' | 'slate';
}

function HeaderStatCard({ icon: Icon, label, value, color }: HeaderStatCardProps) {
  const colorMap = {
    blue: 'text-blue-400',
    green: 'text-emerald-400',
    purple: 'text-purple-400',
    amber: 'text-amber-400',
    red: 'text-red-400',
    slate: 'text-slate-400',
  };

  return (
    <div className="flex items-center justify-center gap-3 px-4 py-3 rounded-xl bg-white/[0.05] border border-white/[0.1] backdrop-blur-sm">
      <Icon className={cn('w-5 h-5 flex-shrink-0', colorMap[color])} />
      <div className="flex flex-col justify-center min-w-0">
        <div className="text-xs text-slate-400 leading-tight">{label}</div>
        <div className="text-lg font-bold text-white leading-tight mt-0.5">{value}</div>
      </div>
    </div>
  );
}

/**
 * GPU detail card
 */
function GPUCard({ gpu, index, t }: { gpu: GPUInfo; index: number; t: (key: string, options?: Record<string, unknown>) => string }) {
  const vramPercent = gpu.totalMemory > 0 ? (gpu.usedMemory / gpu.totalMemory) * 100 : 0;
  const tempColor = gpu.temperature > 80 ? 'text-red-500' : gpu.temperature > 60 ? 'text-amber-500' : 'text-emerald-500';

  return (
    <div className="group relative overflow-hidden rounded-xl border border-border/50 bg-gradient-to-br from-card via-card to-purple-50/30 dark:to-purple-950/10 p-4 transition-all hover:shadow-lg hover:border-purple-200 dark:hover:border-purple-800">
      <div className="absolute -top-10 -right-10 w-32 h-32 bg-gradient-to-br from-purple-500/10 to-blue-500/10 rounded-full blur-2xl group-hover:scale-150 transition-transform duration-500" />

      <div className="relative z-10">
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-gradient-to-br from-purple-500 to-blue-500 text-white shadow-lg">
              <Microchip className="w-5 h-5" />
            </div>
            <div>
              <div className="font-semibold text-foreground">{gpu.name}</div>
              <div className="text-xs text-muted-foreground">{gpu.vendor}</div>
            </div>
          </div>
          <Badge variant="secondary" className="font-mono">GPU {index}</Badge>
        </div>

        <div className="grid grid-cols-2 gap-4 mb-4">
          <div className="space-y-1">
            <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground flex items-center gap-1">
                <Gauge className="w-3 h-3" />
                {t('cluster.info.utilization')}
              </span>
              <span className="font-medium">{gpu.utilization.toFixed(1)}%</span>
            </div>
            <Progress value={gpu.utilization} className="h-1.5" />
          </div>

          <div className="space-y-1">
            <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground flex items-center gap-1">
                <Layers className="w-3 h-3" />
                {t('cluster.info.vram')}
              </span>
              <span className="font-medium">{vramPercent.toFixed(1)}%</span>
            </div>
            <Progress value={vramPercent} className="h-1.5" />
          </div>
        </div>

        <div className="grid grid-cols-3 gap-2 text-xs">
          {gpu.temperature > 0 && (
            <div className="flex items-center gap-1.5 p-2 rounded-lg bg-slate-50 dark:bg-slate-900/50">
              <Thermometer className={cn('w-3.5 h-3.5', tempColor)} />
              <span className="text-muted-foreground">{gpu.temperature.toFixed(0)}°C</span>
            </div>
          )}
          {gpu.powerUsage > 0 && (
            <div className="flex items-center gap-1.5 p-2 rounded-lg bg-slate-50 dark:bg-slate-900/50">
              <Power className="w-3.5 h-3.5 text-amber-500" />
              <span className="text-muted-foreground">{gpu.powerUsage.toFixed(0)}W</span>
            </div>
          )}
          {gpu.driverVersion && (
            <div className="flex items-center gap-1.5 p-2 rounded-lg bg-slate-50 dark:bg-slate-900/50 col-span-2">
              <Hash className="w-3.5 h-3.5 text-slate-400" />
              <span className="text-muted-foreground truncate" title={gpu.driverVersion}>
                {gpu.driverVersion}
              </span>
            </div>
          )}
        </div>

        <div className="mt-3 pt-3 border-t border-border/50">
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>{t('cluster.info.vramUsage')}</span>
            <span className="font-mono">
              {formatBytes(gpu.usedMemory)} / {formatBytes(gpu.totalMemory)}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * Client detail dialog with real-time system information
 */
export function ClientInfoDialog({ client, open, onClose }: ClientInfoDialogProps) {
  const { t } = useTranslation();
  const { data: liveClient } = useClient(client?.id || '', {
    enabled: open && !!client?.id,
  });

  const displayClient = liveClient || client;
  const resources = displayClient?.resources;
  const capabilities = displayClient?.capabilities;
  const status = getStatusConfig(displayClient?.status || 'offline');

  const { cpu: cpuPercent, memory: memoryPercent, disk: diskPercent } = getResourcePercentages(resources);

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-5xl max-h-[90vh] p-0 gap-0 flex flex-col overflow-hidden">
        <div className="relative overflow-hidden bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 text-white flex-shrink-0">
          <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-blue-500/20 via-purple-500/10 to-transparent" />
          <div className="absolute -top-24 -right-24 w-48 h-48 bg-purple-500/20 rounded-full blur-3xl" />
          <div className="absolute -bottom-24 -left-24 w-48 h-48 bg-blue-500/20 rounded-full blur-3xl" />

          <DialogHeader className="relative z-10 p-6 pb-4">
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-4">
                <div className="p-3 rounded-2xl bg-white/10 backdrop-blur-sm border border-white/20">
                  <Server className="w-6 h-6" />
                </div>
                <div>
                  <DialogTitle className="text-2xl font-bold text-white">
                    {displayClient?.name}
                  </DialogTitle>
                  <div className="flex items-center gap-2 mt-1">
                    <span className={cn('w-2 h-2 rounded-full animate-pulse', status.indicator)} />
                    <span className="text-slate-300 text-sm">{status.label}</span>
                    <span className="text-slate-500">•</span>
                    <span className="text-slate-400 text-sm font-mono">
                      {displayClient?.id?.slice(0, 8)}...
                    </span>
                  </div>
                </div>
              </div>
              <Badge
                variant="outline"
                className="bg-white/5 border-white/20 text-white backdrop-blur-sm"
              >
                {displayClient?.address}:{displayClient?.port}
              </Badge>
            </div>
          </DialogHeader>

          <div className="relative z-10 grid grid-cols-4 gap-4 px-6 pb-6">
            <HeaderStatCard
              icon={Cpu}
              label={t('cluster.info.cpuCores')}
              value={capabilities?.cpuCount || 0}
              color="blue"
            />
            <HeaderStatCard
              icon={HardDrive}
              label={t('cluster.info.memory')}
              value={formatBytes(capabilities?.memory || 0)}
              color="green"
            />
            <HeaderStatCard
              icon={Microchip}
              label={t('cluster.info.gpu')}
              value={resources?.gpuInfo?.length || 0}
              color="purple"
            />
            <HeaderStatCard
              icon={Globe}
              label={t('cluster.info.rocm')}
              value={resources?.rocmVersion || 'N/A'}
              color="amber"
            />
          </div>
        </div>

        <Tabs defaultValue="resources" className="w-full flex flex-col flex-1 overflow-hidden">
          <div className="px-6 pt-4 flex-shrink-0">
            <TabsList className="w-full grid grid-cols-3 bg-muted/50 p-1 rounded-xl h-auto">
              <TabsTrigger
                value="resources"
                className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:text-primary py-2.5 text-sm font-medium text-muted-foreground transition-all duration-200 ease-out hover:text-foreground data-[state=active]:hover:text-primary"
              >
                <Activity className="w-4 h-4 mr-2" />
                {t('cluster.info.tabResources')}
              </TabsTrigger>
              <TabsTrigger
                value="hardware"
                className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:text-primary py-2.5 text-sm font-medium text-muted-foreground transition-all duration-200 ease-out hover:text-foreground data-[state=active]:hover:text-primary"
              >
                <Monitor className="w-4 h-4 mr-2" />
                {t('cluster.info.tabHardware')}
              </TabsTrigger>
              <TabsTrigger
                value="metadata"
                className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:text-primary py-2.5 text-sm font-medium text-muted-foreground transition-all duration-200 ease-out hover:text-foreground data-[state=active]:hover:text-primary"
              >
                <Terminal className="w-4 h-4 mr-2" />
                {t('cluster.info.tabMetadata')}
              </TabsTrigger>
              <TabsTrigger
                value="config"
                className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:text-primary py-2.5 text-sm font-medium text-muted-foreground transition-all duration-200 ease-out hover:text-foreground data-[state=active]:hover:text-primary"
              >
                <Settings className="w-4 h-4 mr-2" />
                {t('cluster.info.tabConfig')}
              </TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value="resources" className="p-6 space-y-6 mt-0 flex-1 overflow-y-auto">
            <div className="flex justify-center gap-8 py-4">
              <CircularProgress
                value={cpuPercent}
                label="CPU"
                subLabel={cpuPercent > 80 ? t('cluster.info.highLoad') : t('cluster.info.normal')}
                color={cpuPercent > 80 ? 'stroke-red-500' : cpuPercent > 50 ? 'stroke-amber-500' : 'stroke-blue-500'}
              />
              <CircularProgress
                value={memoryPercent}
                label={t('cluster.info.memory')}
                subLabel={formatBytes(resources?.memoryUsed || 0)}
                color={memoryPercent > 80 ? 'stroke-red-500' : memoryPercent > 50 ? 'stroke-amber-500' : 'stroke-emerald-500'}
              />
              <CircularProgress
                value={diskPercent}
                label={t('cluster.info.disk')}
                subLabel={formatBytes(resources?.diskUsed || 0)}
                color={diskPercent > 80 ? 'stroke-red-500' : diskPercent > 50 ? 'stroke-amber-500' : 'stroke-violet-500'}
              />
            </div>

            <div className="grid grid-cols-3 gap-4">
              <MetricCard
                icon={Cpu}
                title={t('cluster.info.cpuUsage')}
                value={`${cpuPercent.toFixed(1)}%`}
                subValue={t('cluster.info.coresCount', { count: capabilities?.cpuCount || 0 })}
                colorClass={cpuPercent > 80 ? 'text-red-600' : cpuPercent > 50 ? 'text-amber-600' : 'text-blue-600'}
              />
              <MetricCard
                icon={HardDrive}
                title={t('cluster.info.memoryUsage')}
                value={formatBytes(resources?.memoryUsed || 0)}
                subValue={`/ ${formatBytes(resources?.memoryTotal || 0)} (${memoryPercent.toFixed(1)}%)`}
                colorClass={memoryPercent > 80 ? 'text-red-600' : memoryPercent > 50 ? 'text-amber-600' : 'text-emerald-600'}
              />
              <MetricCard
                icon={Layers}
                title={t('cluster.info.diskUsage')}
                value={formatBytes(resources?.diskUsed || 0)}
                subValue={`/ ${formatBytes(resources?.diskTotal || 0)} (${diskPercent.toFixed(1)}%)`}
                colorClass={diskPercent > 80 ? 'text-red-600' : diskPercent > 50 ? 'text-amber-600' : 'text-violet-600'}
              />
            </div>

            {resources?.gpuInfo && resources.gpuInfo.length > 0 && (
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <Zap className="w-5 h-5 text-purple-500" />
                  <h3 className="text-lg font-semibold">{t('cluster.info.gpuDetails')}</h3>
                  <Badge variant="secondary">{t('cluster.info.devices', { count: resources.gpuInfo.length })}</Badge>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {resources.gpuInfo.map((gpu, index) => (
                    <GPUCard key={index} gpu={gpu} index={index} t={t} />
                  ))}
                </div>
              </div>
            )}
          </TabsContent>

          <TabsContent value="hardware" className="p-6 space-y-6 mt-0 flex-1 overflow-y-auto">
            <div className="grid grid-cols-2 gap-4">
              <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                <div className="flex items-center gap-2 text-lg font-semibold">
                  <Monitor className="w-5 h-5 text-blue-500" />
                  {t('cluster.info.systemInfo')}
                </div>
                <div className="space-y-3">
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Hash className="w-4 h-4" />
                      {t('cluster.info.kernelVersion')}
                    </span>
                    <code className="px-2 py-1 bg-slate-100 dark:bg-slate-800 rounded text-xs font-mono">
                      {resources?.kernelVersion || 'N/A'}
                    </code>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Globe className="w-4 h-4" />
                      {t('cluster.info.rocmVersion')}
                    </span>
                    <code className="px-2 py-1 bg-slate-100 dark:bg-slate-800 rounded text-xs font-mono">
                      {resources?.rocmVersion || 'N/A'}
                    </code>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Clock className="w-4 h-4" />
                      {t('cluster.info.lastOnline')}
                    </span>
                    <span className="text-sm">
                      {displayClient?.lastSeen
                        ? new Date(displayClient.lastSeen).toLocaleString()
                        : 'N/A'}
                    </span>
                  </div>
                </div>
              </div>

              <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                <div className="flex items-center gap-2 text-lg font-semibold">
                  <Server className="w-5 h-5 text-emerald-500" />
                  {t('cluster.info.hardwareSpecs')}
                </div>
                <div className="space-y-3">
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Cpu className="w-4 h-4" />
                      {t('cluster.info.cpuCoreCount')}
                    </span>
                    <Badge variant="outline">{t('cluster.info.coresCount', { count: capabilities?.cpuCount || 0 })}</Badge>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <HardDrive className="w-4 h-4" />
                      {t('cluster.info.totalMemory')}
                    </span>
                    <Badge variant="outline">{formatBytes(capabilities?.memory || 0)}</Badge>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Microchip className="w-4 h-4" />
                      {t('cluster.info.gpuCount')}
                    </span>
                    <Badge variant="outline">{t('cluster.info.countItems', { count: resources?.gpuInfo?.length || 0 })}</Badge>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Layers className="w-4 h-4" />
                      {t('cluster.info.gpuMemoryTotal')}
                    </span>
                    <Badge variant="outline">
                      {resources?.gpuInfo && resources.gpuInfo.length > 0
                        ? formatBytes(resources.gpuInfo.reduce((sum, gpu) => sum + gpu.totalMemory, 0))
                        : 'N/A'}
                    </Badge>
                  </div>
                </div>
              </div>
            </div>

            <div className="rounded-xl border border-border bg-card p-5">
              <div className="flex items-center gap-2 text-lg font-semibold mb-4">
                <Globe className="w-5 h-5 text-amber-500" />
                {t('cluster.info.networkConfig')}
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="p-3 rounded-lg bg-slate-50 dark:bg-slate-900/50">
                  <div className="text-xs text-muted-foreground mb-1">{t('cluster.info.nodeId')}</div>
                  <code className="text-sm font-mono break-all">{displayClient?.id}</code>
                </div>
                <div className="p-3 rounded-lg bg-slate-50 dark:bg-slate-900/50">
                  <div className="text-xs text-muted-foreground mb-1">{t('cluster.info.connectionAddress')}</div>
                  <div className="text-sm font-medium">
                    {displayClient?.address}:{displayClient?.port}
                  </div>
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="metadata" className="p-6 mt-0 flex-1 overflow-y-auto">
            {displayClient?.metadata && Object.keys(displayClient.metadata).length > 0 ? (
              <div className="rounded-xl border border-border bg-card overflow-hidden">
                <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-border flex items-center gap-2">
                  <Terminal className="w-4 h-4 text-slate-500" />
                  <span className="font-medium text-sm">{t('cluster.info.customMetadata')}</span>
                  <Badge variant="secondary" className="ml-auto">
                    {t('cluster.info.items', { count: Object.keys(displayClient.metadata).length })}
                  </Badge>
                </div>
                <div className="divide-y divide-border">
                  {Object.entries(displayClient.metadata).map(([key, value]) => (
                    <div
                      key={key}
                      className="flex items-center justify-between px-4 py-3 hover:bg-slate-50/50 dark:hover:bg-slate-900/30 transition-colors"
                    >
                      <span className="text-sm text-muted-foreground font-medium">{key}</span>
                      <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded max-w-[50%] truncate">
                        {String(value)}
                      </code>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Terminal className="w-12 h-12 mb-4 opacity-20" />
                <p>{t('cluster.info.noMetadata')}</p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="config" className="p-6 mt-0 flex-1 overflow-y-auto">
            <NodeConfigPanel clientId={client?.id || ''} />
          </TabsContent>
        </Tabs>

        <div className="flex justify-end gap-2 p-4 border-t border-border bg-slate-50/50 dark:bg-slate-900/20 flex-shrink-0">
          <Button onClick={onClose} variant="outline">
            {t('cluster.info.close')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}


/**
 * Node configuration panel with llama.cpp testing
 */
function NodeConfigPanel({ clientId }: { clientId: string }) {
  const { t } = useTranslation();
  const { data: config, isLoading } = useNodeConfig(clientId, { enabled: !!clientId });
  const testLlamacpp = useTestNodeLlamacpp();
  const [testResult, setTestResult] = useState<LlamacppTestResult | null>(null);

  const handleTest = async () => {
    try {
      const result = await testLlamacpp.mutateAsync(clientId);
      setTestResult(result);
      if (result.success) {
        toast.success(
          t('cluster.info.testSuccessTitle'),
          t('cluster.info.testSuccessDesc', { version: result.version || t('cluster.info.unknownError') })
        );
      } else {
        toast.error(t('cluster.info.testFailedTitle'), result.error || t('cluster.info.unknownError'));
      }
    } catch (error) {
      toast.error(t('cluster.info.testFailedTitle'), error instanceof Error ? error.message : t('cluster.info.requestFailed'));
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!config) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <AlertTriangle className="w-12 h-12 mb-4" />
        <p>{t('cluster.info.cannotFetchConfig')}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between p-4 bg-card rounded-xl border border-border">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-primary/10 dark:bg-primary/20 rounded-lg">
            <Terminal className="w-5 h-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold">{t('cluster.info.llamacppTest')}</h3>
            <p className="text-sm text-muted-foreground">{t('cluster.info.llamacppTestDesc')}</p>
          </div>
        </div>
        <Button
          onClick={handleTest}
          disabled={testLlamacpp.isPending}
          className="gap-2"
        >
          {testLlamacpp.isPending ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Play className="w-4 h-4" />
          )}
          {t('cluster.info.startTest')}
        </Button>
      </div>

      {testResult && (
        <div
          className={cn(
            'p-4 rounded-xl border',
            testResult.success
              ? 'bg-primary/10 dark:bg-primary/20 border-primary/20 dark:border-primary/30'
              : 'bg-destructive/10 dark:bg-destructive/20 border-destructive/20 dark:border-destructive/30'
          )}
        >
          <div className="flex items-start gap-3">
            {testResult.success ? (
              <CheckCircle className="w-5 h-5 text-primary mt-0.5" />
            ) : (
              <XCircle className="w-5 h-5 text-destructive mt-0.5" />
            )}
            <div className="flex-1 min-w-0">
              <div className="font-medium">
                {testResult.success ? t('cluster.info.testPassed') : t('cluster.info.testFailed')}
              </div>
              {testResult.version && (
                <div className="text-sm text-muted-foreground mt-1">
                  {t('cluster.info.version', { version: testResult.version })}
                </div>
              )}
              {testResult.error && (
                <div className="text-sm text-destructive mt-1">
                  {t('cluster.info.error', { error: testResult.error })}
                </div>
              )}
              <div className="text-xs text-muted-foreground mt-2">
                {t('cluster.info.testPath', { path: testResult.path })}
              </div>
              <div className="text-xs text-muted-foreground">
                {t('cluster.info.testDuration', { duration: testResult.duration.toFixed(2) })} · {new Date(testResult.testedAt).toLocaleString()}
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-border flex items-center gap-2">
          <FileCode className="w-4 h-4 text-primary" />
          <span className="font-medium text-sm">{t('cluster.info.llamacppPaths')}</span>
          <Badge variant="secondary" className="ml-auto">
            {t('cluster.info.countItems', { count: config.llamaCppPaths.length })}
          </Badge>
        </div>
        <div className="divide-y divide-border">
          {config.llamaCppPaths.length === 0 ? (
            <div className="px-4 py-3 text-sm text-muted-foreground">{t('cluster.info.noConfig')}</div>
          ) : (
            config.llamaCppPaths.map((item, index) => (
              <div
                key={index}
                className="flex items-center justify-between px-4 py-3 hover:bg-slate-50/50 dark:hover:bg-slate-900/30 transition-colors"
              >
                <div className="flex items-center gap-3 min-w-0">
                  {item.exists ? (
                    <CheckCircle className="w-4 h-4 text-primary flex-shrink-0" />
                  ) : (
                    <XCircle className="w-4 h-4 text-destructive flex-shrink-0" />
                  )}
                  <code className="text-xs font-mono truncate">{item.path}</code>
                  {item.isDefault && (
                    <Badge variant="outline" className="text-xs flex-shrink-0">{t('cluster.info.default')}</Badge>
                  )}
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  {item.version && (
                    <span className="text-xs text-muted-foreground">{item.version}</span>
                  )}
                  <Badge
                    variant={item.exists ? 'outline' : 'secondary'}
                    className="text-xs"
                  >
                    {item.exists ? t('cluster.info.available') : t('cluster.info.unavailable')}
                  </Badge>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-border flex items-center gap-2">
          <FolderOpen className="w-4 h-4 text-primary" />
          <span className="font-medium text-sm">{t('cluster.info.modelPaths')}</span>
          <Badge variant="secondary" className="ml-auto">
            {t('cluster.info.countItems', { count: config.modelPaths.length })}
          </Badge>
        </div>
        <div className="divide-y divide-border">
          {config.modelPaths.length === 0 ? (
            <div className="px-4 py-3 text-sm text-muted-foreground">{t('cluster.info.noConfig')}</div>
          ) : (
            config.modelPaths.map((item, index) => (
              <div
                key={index}
                className="flex items-center justify-between px-4 py-3 hover:bg-slate-50/50 dark:hover:bg-slate-900/30 transition-colors"
              >
                <div className="flex items-center gap-3 min-w-0">
                  {item.exists ? (
                    <CheckCircle className="w-4 h-4 text-primary flex-shrink-0" />
                  ) : (
                    <XCircle className="w-4 h-4 text-destructive flex-shrink-0" />
                  )}
                  <code className="text-xs font-mono truncate">{item.path}</code>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  {typeof item.modelCount === 'number' && (
                    <Badge variant="secondary" className="text-xs">{t('cluster.info.modelCount', { count: item.modelCount })}</Badge>
                  )}
                  <Badge
                    variant={item.exists ? 'outline' : 'secondary'}
                    className="text-xs"
                  >
                    {item.exists ? t('cluster.info.available') : t('cluster.info.unavailable')}
                  </Badge>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 mb-3">
            <Globe className="w-4 h-4 text-primary" />
            <span className="font-medium text-sm">{t('cluster.info.systemEnvironment')}</span>
          </div>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('cluster.info.os')}</span>
              <span className="font-mono">{config.environment.os}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('cluster.info.architecture')}</span>
              <span className="font-mono">{config.environment.architecture}</span>
            </div>
            {config.environment.kernelVersion && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{t('cluster.info.kernel')}</span>
                <span className="font-mono">{config.environment.kernelVersion}</span>
              </div>
            )}
            {config.environment.rocmVersion && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{t('cluster.info.rocmVersion')}</span>
                <span className="font-mono">{config.environment.rocmVersion}</span>
              </div>
            )}
            {config.environment.cudaVersion && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{t('cluster.info.cudaVersion')}</span>
                <span className="font-mono">{config.environment.cudaVersion}</span>
              </div>
            )}
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 mb-3">
            <FileCode className="w-4 h-4 text-primary" />
            <span className="font-medium text-sm">{t('cluster.info.runtime')}</span>
          </div>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('cluster.info.goVersion')}</span>
              <span className="font-mono">{config.environment.goVersion}</span>
            </div>
            {config.environment.pythonVersion && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{t('cluster.info.pythonVersion')}</span>
                <span className="font-mono">{config.environment.pythonVersion}</span>
              </div>
            )}
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('cluster.info.pythonPath')}</span>
              <span className="font-mono truncate max-w-[150px]" title={config.executor.pythonPath}>
                {config.executor.pythonPath}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t('cluster.info.timeout')}</span>
              <span>{t('cluster.info.timeoutSeconds', { seconds: config.executor.timeout })}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-border flex items-center gap-2">
          <Layers className="w-4 h-4 text-primary" />
          <span className="font-medium text-sm">{t('cluster.info.condaEnv')}</span>
          <Badge
            variant={config.conda.enabled ? 'outline' : 'secondary'}
            className="ml-auto"
          >
            {config.conda.enabled ? t('cluster.info.condaEnabled') : t('cluster.info.condaDisabled')}
          </Badge>
        </div>
        {config.conda.enabled && config.conda.availableEnvs.length > 0 && (
          <div className="p-4">
            <div className="flex flex-wrap gap-2">
              {config.conda.availableEnvs.map((env) => (
                <Badge
                  key={env}
                  variant={env === config.conda.defaultEnv ? 'default' : 'outline'}
                  className="text-xs"
                >
                  {env}
                  {env === config.conda.defaultEnv ? t('cluster.info.condaDefault') : ''}
                </Badge>
              ))}
            </div>
          </div>
        )}
      </div>

      <div className="text-xs text-muted-foreground text-right">
        {t('cluster.info.configCollectedAt', { time: new Date(config.collectedAt).toLocaleString() })}
      </div>
    </div>
  );
}
