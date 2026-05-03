import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { Loader2, Gauge, RotateCcw, Play } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import {
  useBenchmarkParams,
  useLlamaCppVersions,
} from '@/features/models';
import type {
  BenchmarkConfig,
  BenchmarkParam,
} from '@/types';
import { useToast } from '@/hooks/useToast';

interface BenchmarkDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (config: BenchmarkConfig) => void;
  modelId: string;
  modelName: string;
  isLoading?: boolean;
}

/**
 * Benchmark dialog component
 * Modeled after LlamacppServer's model-benchmark.js
 */
export function BenchmarkDialog({
  isOpen,
  onClose,
  onConfirm,
  modelId,
  modelName,
  isLoading = false,
}: BenchmarkDialogProps) {
  const toast = useToast();

  const { data: benchmarkParams = [], isLoading: paramsLoading } = useBenchmarkParams();

  const { data: llamaCppVersions = [], isLoading: versionsLoading } = useLlamaCppVersions();

  const benchmarkParamsKeys = useMemo(() => benchmarkParams.map(p => p.fullName).join(','), [benchmarkParams]);

  const [llamaCppPath, setLlamaCppPath] = useState<string>('');
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);
  const [availableDevices, setAvailableDevices] = useState<string[]>([]);
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [devicesLoading, setDevicesLoading] = useState(false);

  const wasOpen = useRef(false);
  const prevParamsKeysRef = useRef<string>('');

  const initializeDefaults = useCallback(() => {
    const defaults: Record<string, string> = {};
    benchmarkParams.forEach((param) => {
      if (param.defaultValue) {
        defaults[param.fullName] = param.defaultValue;
      }
    });
    setParamValues(defaults);
  }, [benchmarkParams]);

  const handleReset = () => {
    initializeDefaults();
    setSelectedDevices(availableDevices);
  };

  useEffect(() => {
    if (isOpen && !llamaCppPath && llamaCppVersions.length > 0) {
      setLlamaCppPath(llamaCppVersions[0].path);
    }
  }, [isOpen, llamaCppPath, llamaCppVersions]);

  // Load available devices when llama.cpp path is selected
  useEffect(() => {
    if (!isOpen || !llamaCppPath) return;

    const abortController = new AbortController();

    const loadDevices = async () => {
      setDevicesLoading(true);
      try {
        const response = await fetch(
          `/api/model/device/list?llamaBinPath=${encodeURIComponent(llamaCppPath)}`,
          { signal: abortController.signal }
        );

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();
        if (data.success && data.data?.devices) {
          const devices = data.data.devices;
          setAvailableDevices(devices);
          setSelectedDevices(devices);
        } else {
          throw new Error(data.error || '无法解析设备列表响应');
        }
      } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') return;
        console.error('Failed to load devices:', error);
        const errorMsg = error instanceof Error ? error.message : '未知错误';
        toast.error('无法加载计算设备列表', errorMsg);
        setAvailableDevices([]);
        setSelectedDevices([]);
      } finally {
        if (!abortController.signal.aborted) {
          setDevicesLoading(false);
        }
      }
    };
    loadDevices();

    return () => abortController.abort();
  }, [isOpen, llamaCppPath]);

  useEffect(() => {
    const paramsJustLoaded = prevParamsKeysRef.current === '' && benchmarkParamsKeys !== '';

    if (isOpen && (!wasOpen.current || paramsJustLoaded)) {
      initializeDefaults();
      setAvailableDevices([]);
      setSelectedDevices([]);
    }
    wasOpen.current = isOpen;
    prevParamsKeysRef.current = benchmarkParamsKeys;
  }, [isOpen, benchmarkParamsKeys, initializeDefaults]);

  const handleDeviceToggle = (device: string) => {
    setSelectedDevices((prev) =>
      prev.includes(device)
        ? prev.filter((d) => d !== device)
        : [...prev, device]
    );
  };

  const handleSelectAllDevices = () => {
    if (selectedDevices.length === availableDevices.length) {
      setSelectedDevices([]);
    } else {
      setSelectedDevices([...availableDevices]);
    }
  };

  const handleParamChange = (fullName: string, value: string) => {
    setParamValues((prev) => ({ ...prev, [fullName]: value }));
  };

  const buildCommand = (): string => {
    const parts: string[] = [];

    Object.entries(paramValues).forEach(([key, value]) => {
      if (value === 'true') {
        parts.push(key);
      } else if (value !== 'false' && value !== '') {
        parts.push(key, value);
      }
    });

    if (selectedDevices.length > 0 && selectedDevices.length < availableDevices.length) {
      parts.push('-dev', selectedDevices.join('/'));
    }

    return parts.join(' ');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!llamaCppPath) {
      toast.warning('请选择 Llama.cpp 版本');
      return;
    }

    const cmd = buildCommand();
    if (!cmd) {
      toast.warning('请配置压测参数');
      return;
    }

    const config: BenchmarkConfig = {
      modelId,
      modelName,
      llamaCppPath,
      devices: selectedDevices,
      params: paramValues,
    };

    onConfirm(config);
  };

  const renderParamField = (param: BenchmarkParam) => {
    const value = paramValues[param.fullName] || param.defaultValue || '';
    const id = `param-${param.fullName.replace(/[^a-zA-Z0-9]/g, '_')}`;

    if (param.values && param.values.length > 0) {
      return (
        <div key={param.fullName} className="space-y-1">
          <label className="flex items-center text-xs font-medium text-foreground">
            {param.name}
            {param.abbreviation && (
              <span className="ml-1 text-muted-foreground">({param.abbreviation})</span>
            )}
            {param.description && (
              <span
                className="ml-1 text-muted-foreground cursor-help"
                title={param.description}
              >
                ⓘ
              </span>
            )}
          </label>
          <Select
            value={value}
            onValueChange={(v) => handleParamChange(param.fullName, v)}
            disabled={isLoading}
          >
            <SelectTrigger className={cn(
              "w-full px-2 py-1.5 text-sm",
              "border border-border rounded-md",
              "bg-input text-foreground",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {param.values.map((v) => (
                <SelectItem key={v} value={v}>
                  {v}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      );
    }

    if (param.type === 'LOGIC') {
      return (
        <div key={param.fullName} className="space-y-1">
          <label className="flex items-center text-xs font-medium text-foreground">
            {param.name}
            {param.abbreviation && (
              <span className="ml-1 text-muted-foreground">({param.abbreviation})</span>
            )}
            {param.description && (
              <span
                className="ml-1 text-muted-foreground cursor-help"
                title={param.description}
              >
                ⓘ
              </span>
            )}
          </label>
          <Select
            value={value || 'false'}
            onValueChange={(v) => handleParamChange(param.fullName, v)}
            disabled={isLoading}
          >
            <SelectTrigger className={cn(
              "w-full px-2 py-1.5 text-sm",
              "border border-border rounded-md",
              "bg-input text-foreground",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="true">true</SelectItem>
              <SelectItem value="false">false</SelectItem>
            </SelectContent>
          </Select>
        </div>
      );
    }

    if (param.type === 'INTEGER') {
      return (
        <div key={param.fullName} className="space-y-1">
          <label className="flex items-center text-xs font-medium text-foreground">
            {param.name}
            {param.abbreviation && (
              <span className="ml-1 text-muted-foreground">({param.abbreviation})</span>
            )}
            {param.description && (
              <span
                className="ml-1 text-muted-foreground cursor-help"
                title={param.description}
              >
                ⓘ
              </span>
            )}
          </label>
          <Input
            id={id}
            type="number"
            value={value}
            onChange={(e) => handleParamChange(param.fullName, e.target.value)}
            disabled={isLoading}
            className={cn(
              "w-full px-2 py-1.5 text-sm",
              "border border-border rounded-md",
              "bg-input text-foreground",
              "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}
          />
        </div>
      );
    }

    if (param.type === 'FLOAT') {
      return (
        <div key={param.fullName} className="space-y-1">
          <label className="flex items-center text-xs font-medium text-foreground">
            {param.name}
            {param.abbreviation && (
              <span className="ml-1 text-muted-foreground">({param.abbreviation})</span>
            )}
            {param.description && (
              <span
                className="ml-1 text-muted-foreground cursor-help"
                title={param.description}
              >
                ⓘ
              </span>
            )}
          </label>
          <Input
            id={id}
            type="number"
            step="0.01"
            value={value}
            onChange={(e) => handleParamChange(param.fullName, e.target.value)}
            disabled={isLoading}
            className={cn(
              "w-full px-2 py-1.5 text-sm",
              "border border-border rounded-md",
              "bg-input text-foreground",
              "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}
          />
        </div>
      );
    }

    return (
      <div key={param.fullName} className="space-y-1">
        <label className="flex items-center text-xs font-medium text-foreground">
          {param.name}
          {param.abbreviation && (
            <span className="ml-1 text-muted-foreground">({param.abbreviation})</span>
          )}
          {param.description && (
            <span
              className="ml-1 text-muted-foreground cursor-help"
              title={param.description}
            >
              ⓘ
            </span>
          )}
        </label>
        <Input
          id={id}
          type="text"
          value={value}
          onChange={(e) => handleParamChange(param.fullName, e.target.value)}
          disabled={isLoading}
          className={cn(
            "w-full px-2 py-1.5 text-sm",
            "border border-border rounded-md",
            "bg-input text-foreground",
            "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
            "disabled:opacity-50 disabled:cursor-not-allowed"
          )}
        />
      </div>
    );
  };

  if (!isOpen) return null;

  const sortedParams = [...benchmarkParams].sort((a, b) => (a.sort || 0) - (b.sort || 0));

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="sm:max-w-5xl max-h-[90vh] flex flex-col p-0">
        <DialogHeader className="px-4 py-3 border-b border-border flex-shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <Gauge className="w-5 h-5 text-blue-500" />
            模型性能测试
          </DialogTitle>
        </DialogHeader>

        {/* Form content - two-column layout */}
        <form onSubmit={handleSubmit} className="flex-1 overflow-hidden flex flex-col min-h-0">
          <div className="flex-1 overflow-y-auto p-4">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Left: basic params */}
              <div className="space-y-4">
                <div className="bg-muted/30 rounded-lg p-4 space-y-4">
                  <h3 className="text-sm font-semibold text-foreground flex items-center gap-2 pb-2 border-b border-border">
                    <span className="w-1.5 h-1.5 rounded-full bg-blue-500"></span>
                    基础参数
                  </h3>

                  {/* Model info */}
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">
                      模型
                    </label>
                    <div className="px-3 py-2 bg-muted rounded-md text-foreground text-sm truncate">
                      {modelName}
                    </div>
                  </div>

                  {/* Llama.cpp version */}
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">
                      Llama.cpp 版本
                    </label>
                    <Select
                      value={llamaCppPath}
                      onValueChange={setLlamaCppPath}
                      disabled={isLoading || versionsLoading}
                    >
                      <SelectTrigger className={cn(
                        "w-full px-3 py-2 text-sm",
                        "border border-border rounded-md",
                        "bg-input text-foreground",
                        "disabled:opacity-50 disabled:cursor-not-allowed"
                      )}>
                        <SelectValue placeholder="未配置 llama.cpp 路径" />
                      </SelectTrigger>
                      <SelectContent>
                        {llamaCppVersions.length > 0 ? (
                          llamaCppVersions.map((version) => (
                            <SelectItem key={version.path} value={version.path}>
                              {version.name || version.path} {version.description && `(${version.description})`}
                            </SelectItem>
                          ))
                        ) : (
                          <SelectItem value="" disabled>未配置 llama.cpp 路径</SelectItem>
                        )}
                      </SelectContent>
                    </Select>
                  </div>

                  {/* Device selection */}
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <label className="text-xs font-medium text-foreground">
                        计算设备 (-dev)
                      </label>
                      {availableDevices.length > 0 && (
                        <button
                          type="button"
                          onClick={handleSelectAllDevices}
                          className="text-xs text-blue-500 hover:text-blue-600"
                        >
                          {selectedDevices.length === availableDevices.length ? '取消全选' : '全选'}
                        </button>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mb-2">
                      默认已勾选全部设备；取消勾选可排除设备；未选择设备时，使用 auto
                    </p>
                    <div className="border border-border rounded-lg p-2 bg-card max-h-40 overflow-y-auto">
                      {devicesLoading ? (
                        <div className="flex items-center justify-center py-4">
                          <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                          <span className="ml-2 text-xs text-muted-foreground">加载设备中...</span>
                        </div>
                      ) : availableDevices.length > 0 ? (
                        <div className="space-y-1">
                          {availableDevices.map((device, index) => {
                            const deviceLine = device.trim();
                            const parts = deviceLine.split(':');
                            const deviceId = parts[0]?.trim() || deviceLine;
                            const deviceDesc = parts.slice(1).join(':').trim() || deviceLine;

                            return (
                              <label
                                key={index}
                                className="flex items-start gap-2 text-xs text-foreground cursor-pointer hover:bg-accent p-1.5 rounded transition-colors"
                              >
                                <Checkbox
                                  checked={selectedDevices.includes(device)}
                                  onCheckedChange={() => handleDeviceToggle(device)}
                                  disabled={isLoading}
                                  className="rounded border-border w-3.5 h-3.5 mt-0.5 flex-shrink-0"
                                />
                                <div className="flex-1 min-w-0">
                                  <div className="font-medium text-foreground">{deviceId}</div>
                                  {deviceDesc !== deviceId && (
                                    <div className="text-muted-foreground truncate">{deviceDesc}</div>
                                  )}
                                </div>
                              </label>
                            );
                          })}
                        </div>
                      ) : (
                        <div className="text-xs text-muted-foreground text-center py-4">
                          {llamaCppPath ? '未发现可用设备' : '请先选择 Llama.cpp 版本'}
                        </div>
                      )}
                    </div>
                    {availableDevices.length > 0 && (
                      <div className="mt-1 text-xs text-muted-foreground">
                        已选择 {selectedDevices.length} / {availableDevices.length} 个设备
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {/* Right: benchmark params */}
              <div className="space-y-4">
                <div className="bg-muted/30 rounded-lg p-4 h-full">
                  <h3 className="text-sm font-semibold text-foreground flex items-center gap-2 pb-2 border-b border-border mb-4">
                    <span className="w-1.5 h-1.5 rounded-full bg-green-500"></span>
                    压测参数
                  </h3>

                  {paramsLoading ? (
                    <div className="flex items-center justify-center py-8">
                      <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
                      <span className="ml-2 text-sm text-muted-foreground">加载参数中...</span>
                    </div>
                  ) : sortedParams.length > 0 ? (
                    <div className="grid grid-cols-2 gap-x-4 gap-y-3 max-h-[50vh] overflow-y-auto pr-1">
                      {sortedParams.map((param) => renderParamField(param))}
                    </div>
                  ) : (
                    <div className="text-sm text-muted-foreground text-center py-4">
                      无可用参数
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </form>

        {/* Footer */}
        <DialogFooter className="px-4 py-3 border-t border-border bg-card flex-shrink-0">
          <Button
            type="button"
            variant="ghost"
            onClick={onClose}
            disabled={isLoading}
          >
            取消
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={handleReset}
            disabled={isLoading}
          >
            <RotateCcw className="w-4 h-4 mr-1" />
            重置
          </Button>
          <Button
            type="submit"
            onClick={(e) => {
              e.preventDefault();
              const form = document.querySelector('form') as HTMLFormElement;
              if (form) form.requestSubmit();
            }}
            disabled={isLoading || !llamaCppPath}
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                测试中...
              </>
            ) : (
              <>
                <Play className="w-4 h-4 mr-2" />
                开始测试
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
