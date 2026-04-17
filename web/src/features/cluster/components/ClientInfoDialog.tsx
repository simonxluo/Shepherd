import { useState } from 'react';
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
  Terminal as TerminalIcon,

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
import { useClient, useNodeConfig, useTestNodeLlamacpp } from '@/features/cluster/hooks';
import { cn } from '@/lib/utils';
import type { Client, GPUInfo, NodeConfigInfo, LlamacppTestResult } from '@/types';
import { useToast } from '@/hooks/useToast';

interface ClientInfoDialogProps {
  client: Client | null;
  open: boolean;
  onClose: () => void;
}

/**
 * 格式化字节大小
 */
function formatSize(bytes: number): string {
  if (bytes === 0 || bytes === undefined) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${size.toFixed(1)} ${units[unitIndex]}`;
}

/**
 * 获取状态颜色配置
 */
function getStatusConfig(status: string) {
  switch (status) {
    case 'online':
      return {
        color: 'text-emerald-600 dark:text-emerald-400',
        bg: 'bg-emerald-50 dark:bg-emerald-950/30',
        border: 'border-emerald-200 dark:border-emerald-800',
        label: '在线',
        indicator: 'bg-emerald-500',
      };
    case 'busy':
      return {
        color: 'text-amber-600 dark:text-amber-400',
        bg: 'bg-amber-50 dark:bg-amber-950/30',
        border: 'border-amber-200 dark:border-amber-800',
        label: '忙碌',
        indicator: 'bg-amber-500',
      };
    case 'offline':
      return {
        color: 'text-slate-600 dark:text-slate-400',
        bg: 'bg-slate-50 dark:bg-slate-950/30',
        border: 'border-slate-200 dark:border-slate-800',
        label: '离线',
        indicator: 'bg-slate-500',
      };
    case 'error':
      return {
        color: 'text-red-600 dark:text-red-400',
        bg: 'bg-red-50 dark:bg-red-950/30',
        border: 'border-red-200 dark:border-red-800',
        label: '错误',
        indicator: 'bg-red-500',
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

/**
 * 环形进度条组件
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
 * 资源指标卡片
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
 * 头部统计卡片 - 精致简约风格
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
 * GPU 详情卡片
 */
function GPUCard({ gpu, index }: { gpu: GPUInfo; index: number }) {
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
                利用率
              </span>
              <span className="font-medium">{gpu.utilization.toFixed(1)}%</span>
            </div>
            <div className="h-1.5 bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-purple-500 to-blue-500 rounded-full transition-all duration-500"
                style={{ width: `${gpu.utilization}%` }}
              />
            </div>
          </div>

          <div className="space-y-1">
            <div className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground flex items-center gap-1">
                <Layers className="w-3 h-3" />
                显存
              </span>
              <span className="font-medium">{vramPercent.toFixed(1)}%</span>
            </div>
            <div className="h-1.5 bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-blue-500 to-cyan-500 rounded-full transition-all duration-500"
                style={{ width: `${vramPercent}%` }}
              />
            </div>
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
            <span>显存使用</span>
            <span className="font-mono">
              {formatSize(gpu.usedMemory)} / {formatSize(gpu.totalMemory)}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * 客户端详细信息对话框
 * 现代化卡片式设计，实时显示客户端系统信息
 */
export function ClientInfoDialog({ client, open, onClose }: ClientInfoDialogProps) {
  const { data: liveClient } = useClient(client?.id || '', {
    enabled: open && !!client?.id,
  });

  const displayClient = liveClient || client;
  const resources = displayClient?.resources;
  const capabilities = displayClient?.capabilities;
  const status = getStatusConfig(displayClient?.status || 'offline');

  const cpuPercent = resources?.cpuPercent ?? 0;
  const memoryPercent = resources?.memoryTotal
    ? ((resources.memoryUsed || 0) / resources.memoryTotal) * 100
    : 0;
  const diskPercent = resources?.diskTotal
    ? ((resources.diskUsed || 0) / resources.diskTotal) * 100
    : 0;

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
              label="CPU 核心"
              value={capabilities?.cpuCount || 0}
              color="blue"
            />
            <HeaderStatCard
              icon={HardDrive}
              label="内存"
              value={formatSize(capabilities?.memory || 0)}
              color="green"
            />
            <HeaderStatCard
              icon={Microchip}
              label="GPU"
              value={resources?.gpuInfo?.length || 0}
              color="purple"
            />
            <HeaderStatCard
              icon={Globe}
              label="ROCm"
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
                资源监控
              </TabsTrigger>
              <TabsTrigger
                value="hardware"
                className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:text-primary py-2.5 text-sm font-medium text-muted-foreground transition-all duration-200 ease-out hover:text-foreground data-[state=active]:hover:text-primary"
              >
                <Monitor className="w-4 h-4 mr-2" />
                硬件信息
              </TabsTrigger>
              <TabsTrigger
                value="metadata"
                className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:text-primary py-2.5 text-sm font-medium text-muted-foreground transition-all duration-200 ease-out hover:text-foreground data-[state=active]:hover:text-primary"
              >
                <Terminal className="w-4 h-4 mr-2" />
                元数据
              </TabsTrigger>
              <TabsTrigger
                value="config"
                className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm data-[state=active]:text-primary py-2.5 text-sm font-medium text-muted-foreground transition-all duration-200 ease-out hover:text-foreground data-[state=active]:hover:text-primary"
              >
                <Settings className="w-4 h-4 mr-2" />
                配置
              </TabsTrigger>
            </TabsList>
          </div>

          <TabsContent value="resources" className="p-6 space-y-6 mt-0 flex-1 overflow-y-auto">
            <div className="flex justify-center gap-8 py-4">
              <CircularProgress
                value={cpuPercent}
                label="CPU"
                subLabel={cpuPercent > 80 ? '高负载' : '正常'}
                color={cpuPercent > 80 ? 'stroke-red-500' : cpuPercent > 50 ? 'stroke-amber-500' : 'stroke-blue-500'}
              />
              <CircularProgress
                value={memoryPercent}
                label="内存"
                subLabel={formatSize(resources?.memoryUsed || 0)}
                color={memoryPercent > 80 ? 'stroke-red-500' : memoryPercent > 50 ? 'stroke-amber-500' : 'stroke-emerald-500'}
              />
              <CircularProgress
                value={diskPercent}
                label="磁盘"
                subLabel={formatSize(resources?.diskUsed || 0)}
                color={diskPercent > 80 ? 'stroke-red-500' : diskPercent > 50 ? 'stroke-amber-500' : 'stroke-violet-500'}
              />
            </div>

            <div className="grid grid-cols-3 gap-4">
              <MetricCard
                icon={Cpu}
                title="CPU 使用率"
                value={`${cpuPercent.toFixed(1)}%`}
                subValue={`${capabilities?.cpuCount || 0} 核心`}
                colorClass={cpuPercent > 80 ? 'text-red-600' : cpuPercent > 50 ? 'text-amber-600' : 'text-blue-600'}
              />
              <MetricCard
                icon={HardDrive}
                title="内存使用"
                value={formatSize(resources?.memoryUsed || 0)}
                subValue={`/ ${formatSize(resources?.memoryTotal || 0)} (${memoryPercent.toFixed(1)}%)`}
                colorClass={memoryPercent > 80 ? 'text-red-600' : memoryPercent > 50 ? 'text-amber-600' : 'text-emerald-600'}
              />
              <MetricCard
                icon={Layers}
                title="磁盘使用"
                value={formatSize(resources?.diskUsed || 0)}
                subValue={`/ ${formatSize(resources?.diskTotal || 0)} (${diskPercent.toFixed(1)}%)`}
                colorClass={diskPercent > 80 ? 'text-red-600' : diskPercent > 50 ? 'text-amber-600' : 'text-violet-600'}
              />
            </div>

            {resources?.gpuInfo && resources.gpuInfo.length > 0 && (
              <div className="space-y-4">
                <div className="flex items-center gap-2">
                  <Zap className="w-5 h-5 text-purple-500" />
                  <h3 className="text-lg font-semibold">GPU 详情</h3>
                  <Badge variant="secondary">{resources.gpuInfo.length} 个设备</Badge>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {resources.gpuInfo.map((gpu, index) => (
                    <GPUCard key={index} gpu={gpu} index={index} />
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
                  系统信息
                </div>
                <div className="space-y-3">
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Hash className="w-4 h-4" />
                      内核版本
                    </span>
                    <code className="px-2 py-1 bg-slate-100 dark:bg-slate-800 rounded text-xs font-mono">
                      {resources?.kernelVersion || 'N/A'}
                    </code>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Globe className="w-4 h-4" />
                      ROCm 版本
                    </span>
                    <code className="px-2 py-1 bg-slate-100 dark:bg-slate-800 rounded text-xs font-mono">
                      {resources?.rocmVersion || 'N/A'}
                    </code>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Clock className="w-4 h-4" />
                      最后在线
                    </span>
                    <span className="text-sm">
                      {displayClient?.lastSeen
                        ? new Date(displayClient.lastSeen).toLocaleString('zh-CN')
                        : 'N/A'}
                    </span>
                  </div>
                </div>
              </div>

              <div className="rounded-xl border border-border bg-card p-5 space-y-4">
                <div className="flex items-center gap-2 text-lg font-semibold">
                  <Server className="w-5 h-5 text-emerald-500" />
                  硬件规格
                </div>
                <div className="space-y-3">
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Cpu className="w-4 h-4" />
                      CPU 核心数
                    </span>
                    <Badge variant="outline">{capabilities?.cpuCount || 0} 核</Badge>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <HardDrive className="w-4 h-4" />
                      总内存
                    </span>
                    <Badge variant="outline">{formatSize(capabilities?.memory || 0)}</Badge>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Microchip className="w-4 h-4" />
                      GPU 数量
                    </span>
                    <Badge variant="outline">{resources?.gpuInfo?.length || 0} 个</Badge>
                  </div>
                  <div className="flex items-center justify-between py-2 border-b border-border/50">
                    <span className="text-muted-foreground flex items-center gap-2">
                      <Layers className="w-4 h-4" />
                      GPU 显存
                    </span>
                    <Badge variant="outline">
                      {resources?.gpuInfo && resources.gpuInfo.length > 0
                        ? formatSize(resources.gpuInfo.reduce((sum, gpu) => sum + gpu.totalMemory, 0))
                        : 'N/A'}
                    </Badge>
                  </div>
                </div>
              </div>
            </div>

            <div className="rounded-xl border border-border bg-card p-5">
              <div className="flex items-center gap-2 text-lg font-semibold mb-4">
                <Globe className="w-5 h-5 text-amber-500" />
                网络配置
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="p-3 rounded-lg bg-slate-50 dark:bg-slate-900/50">
                  <div className="text-xs text-muted-foreground mb-1">节点 ID</div>
                  <code className="text-sm font-mono break-all">{displayClient?.id}</code>
                </div>
                <div className="p-3 rounded-lg bg-slate-50 dark:bg-slate-900/50">
                  <div className="text-xs text-muted-foreground mb-1">连接地址</div>
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
                  <span className="font-medium text-sm">自定义元数据</span>
                  <Badge variant="secondary" className="ml-auto">
                    {Object.keys(displayClient.metadata).length} 项
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
                <p>暂无元数据</p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="config" className="p-6 mt-0 flex-1 overflow-y-auto">
            <NodeConfigPanel clientId={client?.id || ''} />
          </TabsContent>
        </Tabs>

        <div className="flex justify-end gap-2 p-4 border-t border-border bg-slate-50/50 dark:bg-slate-900/20 flex-shrink-0">
          <Button onClick={onClose} variant="outline">
            关闭
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}


/**
 * 节点配置面板组件
 * 显示节点配置信息和 llama.cpp 测试功能
 */
function NodeConfigPanel({ clientId }: { clientId: string }) {
  const { data: config, isLoading } = useNodeConfig(clientId, { enabled: !!clientId });
  const testLlamacpp = useTestNodeLlamacpp();
  const toastHelpers = useToast();
  const [testResult, setTestResult] = useState<LlamacppTestResult | null>(null);

  const handleTest = async () => {
    try {
      const result = await testLlamacpp.mutateAsync(clientId);
      setTestResult(result);
      if (result.success) {
        toastHelpers.success('测试成功', `llama.cpp 版本: ${result.version || '未知'}`);
      } else {
        toastHelpers.error('测试失败', result.error || '未知错误');
      }
    } catch (error) {
      toastHelpers.error('测试失败', error instanceof Error ? error.message : '请求失败');
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
        <p>无法获取节点配置信息</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* llama.cpp 测试按钮 */}
      <div className="flex items-center justify-between p-4 bg-card rounded-xl border border-border">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-primary/10 dark:bg-primary/20 rounded-lg">
            <TerminalIcon className="w-5 h-5 text-primary" />
          </div>
          <div>
            <h3 className="font-semibold">llama.cpp 测试</h3>
            <p className="text-sm text-muted-foreground">测试节点上 llama.cpp 的可用性</p>
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
          开始测试
        </Button>
      </div>

      {/* 测试结果 */}
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
                {testResult.success ? '测试通过' : '测试失败'}
              </div>
              {testResult.version && (
                <div className="text-sm text-muted-foreground mt-1">
                  版本: {testResult.version}
                </div>
              )}
              {testResult.error && (
                <div className="text-sm text-destructive mt-1">
                  错误: {testResult.error}
                </div>
              )}
              <div className="text-xs text-muted-foreground mt-2">
                测试路径: {testResult.path}
              </div>
              <div className="text-xs text-muted-foreground">
                耗时: {testResult.duration.toFixed(2)}ms · {new Date(testResult.testedAt).toLocaleString('zh-CN')}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* llama.cpp 路径 */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-border flex items-center gap-2">
          <FileCode className="w-4 h-4 text-primary" />
          <span className="font-medium text-sm">llama.cpp 路径</span>
          <Badge variant="secondary" className="ml-auto">
            {config.llamaCppPaths.length} 个
          </Badge>
        </div>
        <div className="divide-y divide-border">
          {config.llamaCppPaths.length === 0 ? (
            <div className="px-4 py-3 text-sm text-muted-foreground">暂无配置</div>
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
                    <Badge variant="outline" className="text-xs flex-shrink-0">默认</Badge>
                  )}
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  {item.version && (
                    <span className="text-xs text-muted-foreground">{item.version}</span>
                  )}
                  <Badge
                    variant={item.exists ? 'success' : 'secondary'}
                    className="text-xs"
                  >
                    {item.exists ? '可用' : '不可用'}
                  </Badge>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* 模型路径 */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-border flex items-center gap-2">
          <FolderOpen className="w-4 h-4 text-primary" />
          <span className="font-medium text-sm">模型路径</span>
          <Badge variant="secondary" className="ml-auto">
            {config.modelPaths.length} 个
          </Badge>
        </div>
        <div className="divide-y divide-border">
          {config.modelPaths.length === 0 ? (
            <div className="px-4 py-3 text-sm text-muted-foreground">暂无配置</div>
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
                    <Badge variant="secondary" className="text-xs">{item.modelCount} 个模型</Badge>
                  )}
                  <Badge
                    variant={item.exists ? 'success' : 'secondary'}
                    className="text-xs"
                  >
                    {item.exists ? '可用' : '不可用'}
                  </Badge>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* 环境信息 */}
      <div className="grid grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 mb-3">
            <Globe className="w-4 h-4 text-primary" />
            <span className="font-medium text-sm">系统环境</span>
          </div>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">操作系统</span>
              <span className="font-mono">{config.environment.os}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">架构</span>
              <span className="font-mono">{config.environment.architecture}</span>
            </div>
            {config.environment.kernelVersion && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">内核版本</span>
                <span className="font-mono">{config.environment.kernelVersion}</span>
              </div>
            )}
            {config.environment.rocmVersion && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">ROCm 版本</span>
                <span className="font-mono">{config.environment.rocmVersion}</span>
              </div>
            )}
            {config.environment.cudaVersion && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">CUDA 版本</span>
                <span className="font-mono">{config.environment.cudaVersion}</span>
              </div>
            )}
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 mb-3">
            <FileCode className="w-4 h-4 text-primary" />
            <span className="font-medium text-sm">运行时</span>
          </div>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Go 版本</span>
              <span className="font-mono">{config.environment.goVersion}</span>
            </div>
            {config.environment.pythonVersion && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">Python 版本</span>
                <span className="font-mono">{config.environment.pythonVersion}</span>
              </div>
            )}
            <div className="flex justify-between">
              <span className="text-muted-foreground">Python 路径</span>
              <span className="font-mono truncate max-w-[150px]" title={config.executor.pythonPath}>
                {config.executor.pythonPath}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">超时时间</span>
              <span>{config.executor.timeout} 秒</span>
            </div>
          </div>
        </div>
      </div>

      {/* Conda 配置 */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="px-4 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-border flex items-center gap-2">
          <Layers className="w-4 h-4 text-primary" />
          <span className="font-medium text-sm">Conda 环境</span>
          <Badge
            variant={config.conda.enabled ? 'success' : 'secondary'}
            className="ml-auto"
          >
            {config.conda.enabled ? '已启用' : '未启用'}
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
                  {env === config.conda.defaultEnv && ' (默认)'}
                </Badge>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* 收集时间 */}
      <div className="text-xs text-muted-foreground text-right">
        配置收集时间: {new Date(config.collectedAt).toLocaleString('zh-CN')}
      </div>
    </div>
  );
}