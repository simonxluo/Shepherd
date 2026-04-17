import { useState, useEffect, useRef, useCallback } from 'react';
import { Plug, CheckCircle2, XCircle, AlertTriangle, Loader2, Power } from 'lucide-react';

export interface ApiConfig {
  enabled: boolean;
  port: number;
}

interface ApiConfigCardProps {
  type: 'ollama' | 'lmstudio';
  config: ApiConfig;
  onConfigChange: (config: ApiConfig) => void;
  saveStatus?: 'idle' | 'saving' | 'success' | 'error';
  onTestConnection?: (port: number, type: 'ollama' | 'lmstudio') => Promise<boolean>;
  onConnectionFailed?: (type: 'ollama' | 'lmstudio', port: number) => void;
}

export function ApiConfigCard({
  type,
  config,
  onConfigChange,
  saveStatus = 'idle',
  onTestConnection,
  onConnectionFailed,
}: ApiConfigCardProps) {
  const [connectionStatus, setConnectionStatus] = useState<'unknown' | 'testing' | 'reachable' | 'unreachable'>('unknown');
  const [localPort, setLocalPort] = useState(config.port.toString());
  const [isHovered, setIsHovered] = useState(false);
  
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const testTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const failedNotifiedRef = useRef(false);

  const title = type === 'ollama' ? 'Ollama API' : 'LM Studio API';
  const defaultPort = type === 'ollama' ? 11434 : 1234;

  const clearTimers = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    if (testTimeoutRef.current) {
      clearTimeout(testTimeoutRef.current);
      testTimeoutRef.current = null;
    }
  }, []);

  const runTest = useCallback(async () => {
    if (!config.enabled || !onTestConnection) {
      setConnectionStatus('unknown');
      return;
    }

    setConnectionStatus('testing');
    try {
      const reachable = await onTestConnection(config.port, type);
      setConnectionStatus(reachable ? 'reachable' : 'unreachable');
      
      // If connection failed and not already notified, trigger callback
      if (!reachable && !failedNotifiedRef.current && onConnectionFailed) {
        failedNotifiedRef.current = true;
        onConnectionFailed(type, config.port);
      }
    } catch {
      setConnectionStatus('unreachable');
      // If connection failed and not already notified, trigger callback
      if (!failedNotifiedRef.current && onConnectionFailed) {
        failedNotifiedRef.current = true;
        onConnectionFailed(type, config.port);
      }
    }
  }, [config.enabled, config.port, type, onTestConnection, onConnectionFailed]);

  useEffect(() => {
    clearTimers();

    if (!config.enabled) {
      setConnectionStatus('unknown');
      failedNotifiedRef.current = false;
      return;
    }

    testTimeoutRef.current = setTimeout(() => {
      runTest();
      intervalRef.current = setInterval(runTest, 10000);
    }, 500);

    return clearTimers;
  }, [config.enabled, config.port, runTest, clearTimers]);

  useEffect(() => {
    setLocalPort(config.port.toString());
  }, [config.port]);

  const handleToggle = () => {
    const newEnabled = !config.enabled;
    onConfigChange({ 
      ...config, 
      enabled: newEnabled,
      port: config.port || defaultPort
    });
  };

  const handlePortChange = (value: string) => {
    setLocalPort(value);
    const port = parseInt(value) || 0;
    if (port >= 1 && port <= 65535) {
      onConfigChange({ ...config, port });
    }
  };

  const getToggleButton = () => {
    const baseClasses = "relative inline-flex items-center gap-2 px-4 py-2 rounded-lg font-medium text-sm transition-all duration-200";
    
    if (config.enabled) {
      return (
        <button
          onClick={handleToggle}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          className={`${baseClasses} bg-green-100 text-green-700 hover:bg-red-100 hover:text-red-600 dark:bg-green-900/30 dark:text-green-400 dark:hover:bg-red-900/30 dark:hover:text-red-400`}
        >
          <Power className="w-4 h-4" />
          <span>{isHovered ? '关闭' : '已启用'}</span>
          <CheckCircle2 className="w-4 h-4" />
        </button>
      );
    }

    return (
      <button
        onClick={handleToggle}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
        className={`${baseClasses} bg-muted text-muted-foreground hover:bg-blue-100 hover:text-blue-600 dark:hover:bg-blue-900/30 dark:hover:text-blue-400`}
      >
        <Power className="w-4 h-4" />
        <span>{isHovered ? '启用' : '待启用'}</span>
      </button>
    );
  };

  const getConnectionBadge = () => {
    if (!config.enabled) return null;

    switch (connectionStatus) {
      case 'testing':
        return (
          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400">
            <Loader2 className="w-3 h-3 animate-spin" />
            检测中
          </span>
        );
      case 'reachable':
        return (
          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400">
            <CheckCircle2 className="w-3 h-3" />
            运行中
          </span>
        );
      case 'unreachable':
        return (
          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400">
            <AlertTriangle className="w-3 h-3" />
            未响应
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400">
            <Loader2 className="w-3 h-3 animate-spin" />
            初始化
          </span>
        );
    }
  };

  const getSaveIndicator = () => {
    if (saveStatus === 'idle') return null;

    const indicators = {
      saving: { icon: Loader2, text: '保存中...', className: 'animate-spin text-blue-500' },
      success: { icon: CheckCircle2, text: '已保存', className: 'text-green-500' },
      error: { icon: XCircle, text: '保存失败', className: 'text-red-500' },
    };

    const { icon: Icon, text, className } = indicators[saveStatus];
    return (
      <div className="flex items-center gap-1.5 text-xs">
        <Icon className={`w-3 h-3 ${className}`} />
        <span className={className}>{text}</span>
      </div>
    );
  };

  return (
    <div className="rounded-xl border bg-card p-5 shadow-sm">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`flex h-10 w-10 items-center justify-center rounded-xl ${config.enabled ? 'bg-green-100 dark:bg-green-900/30' : 'bg-muted'}`}>
            <Plug size={20} className={config.enabled ? 'text-green-600 dark:text-green-400' : 'text-muted-foreground'} />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-semibold">{title}</h3>
              {getConnectionBadge()}
            </div>
            <p className="text-xs text-muted-foreground mt-0.5">
              {config.enabled ? `服务运行在端口 ${config.port}` : '点击启用以开始服务'}
            </p>
          </div>
        </div>

        {getToggleButton()}
      </div>

      <div className="space-y-3 pt-3 border-t">
        <div className="flex items-center gap-4">
          <div className="flex-1">
            <label className="block text-xs font-medium mb-1.5 text-muted-foreground">
              服务端口
            </label>
            <input
              type="number"
              min="1"
              max="65535"
              value={localPort}
              onChange={(e) => handlePortChange(e.target.value)}
              className="w-full max-w-[140px] rounded-lg border px-3 py-2 text-sm
                focus:ring-2 focus:ring-primary focus:border-transparent
                bg-background transition-all duration-200"
              placeholder={defaultPort.toString()}
            />
          </div>

          {config.enabled && connectionStatus === 'reachable' && (
            <div className="flex items-center gap-2 text-xs text-green-600 dark:text-green-400">
              <CheckCircle2 className="w-4 h-4" />
              <span>http://localhost:{config.port}</span>
            </div>
          )}
          {config.enabled && connectionStatus === 'unreachable' && (
            <div className="flex items-center gap-2 text-xs text-red-600 dark:text-red-400">
              <AlertTriangle className="w-4 h-4" />
              <span>端口 {config.port} 无响应</span>
            </div>
          )}
        </div>
      </div>

      <div className="mt-3 pt-2 border-t flex justify-end">
        {getSaveIndicator()}
      </div>
    </div>
  );
}
