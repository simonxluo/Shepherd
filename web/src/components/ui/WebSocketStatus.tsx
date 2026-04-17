import { forwardRef } from 'react';
import { cn } from '@/lib/utils';
import type { WebSocketConnectionStatus } from '@/types/websocket';
import { Wifi, WifiOff, RefreshCw, AlertTriangle, type LucideIcon } from 'lucide-react';

interface WebSocketStatusProps {
  status: WebSocketConnectionStatus;
  reconnectAttempts?: number;
  maxReconnectAttempts?: number;
  compact?: boolean;
  showLabel?: boolean;
  className?: string;
}

interface StatusConfig {
  icon: LucideIcon;
  label: string;
  colorClass: string;
  bgClass: string;
  pulseClass: string;
  spinIcon: boolean;
}

const STATUS_CONFIG: Record<WebSocketConnectionStatus, StatusConfig> = {
  connected: {
    icon: Wifi,
    label: 'Connected',
    colorClass: 'text-emerald-400',
    bgClass: 'bg-emerald-500/10 border-emerald-500/30',
    pulseClass: 'animate-pulse',
    spinIcon: false,
  },
  connecting: {
    icon: Wifi,
    label: 'Connecting',
    colorClass: 'text-sky-400',
    bgClass: 'bg-sky-500/10 border-sky-500/30',
    pulseClass: 'animate-pulse',
    spinIcon: false,
  },
  disconnected: {
    icon: WifiOff,
    label: 'Disconnected',
    colorClass: 'text-zinc-400',
    bgClass: 'bg-zinc-500/10 border-zinc-500/30',
    pulseClass: '',
    spinIcon: false,
  },
  reconnecting: {
    icon: RefreshCw,
    label: 'Reconnecting',
    colorClass: 'text-amber-400',
    bgClass: 'bg-amber-500/10 border-amber-500/30',
    pulseClass: '',
    spinIcon: true,
  },
  error: {
    icon: AlertTriangle,
    label: 'Connection Error',
    colorClass: 'text-rose-400',
    bgClass: 'bg-rose-500/10 border-rose-500/30',
    pulseClass: '',
    spinIcon: false,
  },
};

export const WebSocketStatus = forwardRef<HTMLDivElement, WebSocketStatusProps>(
  (
    {
      status,
      reconnectAttempts = 0,
      maxReconnectAttempts = 5,
      compact = false,
      showLabel = true,
      className,
    },
    ref
  ) => {
    const config = STATUS_CONFIG[status];
    const Icon = config.icon;

    const getReconnectLabel = () => {
      if (status === 'reconnecting' && reconnectAttempts > 0) {
        return `${config.label} (${reconnectAttempts}/${maxReconnectAttempts})`;
      }
      return config.label;
    };

    if (compact) {
      return (
        <div
          ref={ref}
          className={cn(
            'inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium border backdrop-blur-sm transition-all duration-300',
            config.bgClass,
            config.colorClass,
            className
          )}
          title={getReconnectLabel()}
        >
          <span className="relative flex h-2 w-2">
            <span
              className={cn(
                'absolute inline-flex h-full w-full rounded-full opacity-75',
                config.colorClass.replace('text-', 'bg-'),
                status === 'connected' && 'animate-ping'
              )}
            />
            <span
              className={cn(
                'relative inline-flex rounded-full h-2 w-2',
                config.colorClass.replace('text-', 'bg-')
              )}
            />
          </span>
          {showLabel && <span>{getReconnectLabel()}</span>}
        </div>
      );
    }

    return (
      <div
        ref={ref}
        className={cn(
          'inline-flex items-center gap-3 px-4 py-2.5 rounded-lg border backdrop-blur-sm transition-all duration-300',
          config.bgClass,
          className
        )}
      >
        <div className="relative">
          <div
            className={cn(
              'absolute inset-0 rounded-full blur-md opacity-50',
              config.colorClass.replace('text-', 'bg-')
            )}
          />
          <Icon
            className={cn(
              'relative h-5 w-5 transition-transform duration-300',
              config.colorClass,
              config.spinIcon && 'animate-spin',
              config.pulseClass
            )}
          />
        </div>

        {showLabel && (
          <div className="flex flex-col">
            <span
              className={cn(
                'text-sm font-semibold tracking-wide',
                config.colorClass
              )}
            >
              {getReconnectLabel()}
            </span>
            {status === 'error' && reconnectAttempts >= maxReconnectAttempts && (
              <span className="text-xs text-zinc-500 mt-0.5">
                Max retries reached
              </span>
            )}
          </div>
        )}

        {status === 'connected' && (
          <div className="flex items-center gap-1 ml-2">
            {[0, 1, 2].map((i) => (
              <span
                key={i}
                className={cn(
                  'w-1 rounded-full bg-emerald-400 transition-all duration-300',
                  i === 0 && 'h-1',
                  i === 1 && 'h-2',
                  i === 2 && 'h-3'
                )}
              />
            ))}
          </div>
        )}
      </div>
    );
  }
);

WebSocketStatus.displayName = 'WebSocketStatus';
