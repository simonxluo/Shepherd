import { useState } from 'react';
import { Server, Cpu, HardDrive, Clock, Info } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { cn, formatBytes } from '@/lib/utils';
import { ClientInfoDialog } from './ClientInfoDialog';
import { STATUS_COLORS, STATUS_LABELS, getResourcePercentages } from '../utils';
import type { Client } from '@/types';

interface ClientCardProps {
  client: Client;
  onDisconnect?: () => void;
  actions?: React.ReactNode;
}

/**
 * Format last seen time
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

  const { cpu: cpuPercent, memory: memoryPercent, gpu: gpuPercent, gpuMemory: gpuMemoryPercent } = getResourcePercentages(client.resources);

  return (
    <>
      <ClientInfoDialog
        client={client}
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
      />
    <div className="bg-card rounded-lg border border-border p-4 hover:shadow-lg transition-shadow">
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
              {formatBytes(client.capabilities.memory)}
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
                  {client.capabilities.gpuMemory ? formatBytes(client.capabilities.gpuMemory) : 'N/A'}
                </span>
              </div>
            </>
          )}
        </div>
      )}

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

      {client.resources && isConnected && (
        <div className="space-y-2 mb-4">
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

          <div>
            <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
              <span>内存</span>
              <span>
                {formatBytes(client.resources.memoryUsed)} / {formatBytes(client.resources.memoryTotal)}
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
                    {formatBytes(client.resources.gpuMemoryUsed ?? 0)} / {formatBytes(client.resources.gpuMemoryTotal ?? 0)}
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

      {Object.keys(client.metadata).length > 0 && (
        <Collapsible className="mt-3">
          <CollapsibleTrigger className="cursor-pointer text-xs text-muted-foreground">
            元数据
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="mt-2 p-2 bg-muted rounded text-xs">
              {Object.entries(client.metadata).map(([key, value]) => (
                <div key={key} className="flex justify-between gap-4">
                  <span className="text-muted-foreground">{key}:</span>
                  <span className="text-foreground font-mono">{String(value)}</span>
                </div>
              ))}
            </div>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
    </>
  );
}
