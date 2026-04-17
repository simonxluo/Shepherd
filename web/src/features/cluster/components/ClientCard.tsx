import { useState } from 'react';
import { Server, Cpu, HardDrive, Wifi, WifiOff, AlertCircle, Clock, Info } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { ClientInfoDialog } from './ClientInfoDialog';
import type { Client, ClientStatus } from '@/types';

interface ClientCardProps {
  client: Client;
  onDisconnect?: () => void;
  actions?: React.ReactNode;
}

/**
 * 客户端状态颜色映射
 */
const STATUS_COLORS: Record<ClientStatus, string> = {
  online: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  offline: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  busy: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
  error: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
  degraded: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300',
  disabled: 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400',
};

/**
 * 客户端状态标签
 */
const STATUS_LABELS: Record<ClientStatus, string> = {
  online: '在线',
  offline: '离线',
  busy: '忙碌',
  error: '错误',
  degraded: '降级',
  disabled: '已禁用',
};

/**
 * 格式化字节大小
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
 * 格式化最后见到时间
 */
function formatLastSeen(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff < 60000) return '刚刚';
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`;
  return date.toLocaleDateString('zh-CN');
}

export function ClientCard({ client, onDisconnect, actions }: ClientCardProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const statusColor = STATUS_COLORS[client.status];
  const statusLabel = STATUS_LABELS[client.status];
  const isConnected = client.status === 'online' || client.status === 'busy';

  // 资源使用百分比
  const cpuPercent = client.resources?.cpuPercent ?? 0;
  const memoryPercent = client.resources?.memoryUsed
    ? (client.resources.memoryUsed / (client.resources.memoryTotal ?? 1)) * 100
    : 0;
  const gpuPercent = client.resources?.gpuPercent ?? 0;
  const gpuMemoryPercent = client.resources?.gpuMemoryUsed && client.resources?.gpuMemoryTotal
    ? (client.resources.gpuMemoryUsed / client.resources.gpuMemoryTotal) * 100
    : 0;

  return (
    <>
      <ClientInfoDialog
        client={client}
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
      />
    <div className="bg-card rounded-lg border border-border p-4 hover:shadow-lg transition-shadow">
      {/* 标题栏 */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              'p-2 rounded-lg',
              isConnected
                ? 'bg-green-100 dark:bg-green-900/30'
                : 'bg-muted'
            )}
          >
            {isConnected ? (
              <Server className="w-5 h-5 text-green-600 dark:text-green-400" />
            ) : (
              <Server className="w-5 h-5 text-muted-foreground" />
            )}
          </div>
          <div>
            <h3 className="font-semibold text-foreground">{client.name}</h3>
            <p className="text-sm text-muted-foreground">
              {client.address}:{client.port}
            </p>
          </div>
        </div>

        <span className={cn('px-2 py-1 rounded-md text-xs font-medium', statusColor)}>
          {statusLabel}
        </span>
      </div>

      {/* 能力信息 */}
      {client.capabilities && (
        <div className="grid grid-cols-2 gap-3 mb-4 p-3 bg-muted rounded-lg">
          <div className="flex items-center gap-2 text-sm">
            <Cpu className="w-4 h-4 text-muted-foreground" />
            <span className="text-muted-foreground">
              {client.capabilities.cpuCount} 核心
            </span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <HardDrive className="w-4 h-4 text-muted-foreground" />
            <span className="text-muted-foreground">
              {formatSize(client.capabilities.memory)}
            </span>
          </div>
          {(client.resources?.gpuInfo?.length ?? 0) > 0 && (
            <>
              <div className="flex items-center gap-2 text-sm">
                <Server className="w-4 h-4 text-purple-500" />
                <span className="text-muted-foreground">
                  {client.resources?.gpuInfo?.length} GPU
                </span>
              </div>
              <div className="flex items-center gap-2 text-sm">
                <HardDrive className="w-4 h-4 text-purple-500" />
                <span className="text-muted-foreground">
                  {client.capabilities.gpuMemory ? formatSize(client.capabilities.gpuMemory) : 'N/A'}
                </span>
              </div>
            </>
          )}
        </div>
      )}

      {/* 标签 */}
      {client.tags.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-4">
          {client.tags.map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded text-xs"
            >
              {tag}
            </span>
          ))}
        </div>
      )}

      {/* 资源使用 */}
      {client.resources && isConnected && (
        <div className="space-y-2 mb-4">
          {/* CPU */}
          <div>
            <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
              <span>CPU</span>
              <span>{cpuPercent.toFixed(1)}%</span>
            </div>
            <div className="w-full h-1.5 bg-muted rounded-full overflow-hidden">
              <div
                className={cn(
                  'h-full transition-all',
                  cpuPercent > 80
                    ? 'bg-red-500'
                    : cpuPercent > 50
                      ? 'bg-yellow-500'
                      : 'bg-green-500'
                )}
                style={{ width: `${cpuPercent}%` }}
              />
            </div>
          </div>

          {/* 内存 */}
          <div>
            <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
              <span>内存</span>
              <span>
                {formatSize(client.resources.memoryUsed)} / {formatSize(client.resources.memoryTotal)}
              </span>
            </div>
            <div className="w-full h-1.5 bg-muted rounded-full overflow-hidden">
              <div
                className={cn(
                  'h-full transition-all',
                  memoryPercent > 80
                    ? 'bg-red-500'
                    : memoryPercent > 50
                      ? 'bg-yellow-500'
                      : 'bg-blue-500'
                )}
                style={{ width: `${memoryPercent}%` }}
              />
            </div>
          </div>

          {/* GPU */}
          {(client.resources?.gpuInfo?.length ?? 0) > 0 && (
            <>
              <div>
                <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
                  <span>GPU</span>
                  <span>{gpuPercent.toFixed(1)}%</span>
                </div>
                <div className="w-full h-1.5 bg-muted rounded-full overflow-hidden">
                  <div
                    className={cn(
                      'h-full transition-all',
                      gpuPercent > 80 ? 'bg-red-500' : gpuPercent > 50 ? 'bg-yellow-500' : 'bg-purple-500'
                    )}
                    style={{ width: `${gpuPercent}%` }}
                  />
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
                  <span>GPU 内存</span>
                  <span>
                    {formatSize(client.resources.gpuMemoryUsed ?? 0)} / {formatSize(client.resources.gpuMemoryTotal ?? 0)}
                  </span>
                </div>
                <div className="w-full h-1.5 bg-muted rounded-full overflow-hidden">
                  <div
                    className={cn(
                      'h-full transition-all',
                      gpuMemoryPercent > 80
                        ? 'bg-red-500'
                        : gpuMemoryPercent > 50
                          ? 'bg-yellow-500'
                          : 'bg-purple-500'
                    )}
                    style={{ width: `${gpuMemoryPercent}%` }}
                  />
                </div>
              </div>
            </>
          )}
        </div>
      )}

      {/* 最后见到时间和操作 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <Clock className="w-3 h-3" />
          <span>{formatLastSeen(client.lastSeen)}</span>
        </div>

        <div className="flex items-center gap-2">
          {actions}
          <Button
            onClick={() => setDialogOpen(true)}
            variant="outline"
            size="xs"
          >
            <Info className="w-3 h-3 mr-1" />
            详情
          </Button>
          {onDisconnect && isConnected && (
            <Button
              onClick={onDisconnect}
              variant="destructive"
              size="xs"
            >
              断开
            </Button>
          )}
        </div>
      </div>

      {/* 元数据 */}
      {Object.keys(client.metadata).length > 0 && (
        <details className="mt-3">
          <summary className="cursor-pointer text-xs text-muted-foreground">
            元数据
          </summary>
          <div className="mt-2 p-2 bg-muted rounded text-xs">
            {Object.entries(client.metadata).map(([key, value]) => (
              <div key={key} className="flex justify-between gap-4">
                <span className="text-muted-foreground">{key}:</span>
                <span className="text-foreground font-mono">{String(value)}</span>
              </div>
            ))}
          </div>
        </details>
      )}
    </div>
    </>
  );
}
