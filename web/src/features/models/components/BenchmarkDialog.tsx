import { useState, useEffect, useRef, useMemo } from 'react';
import { X, Loader2, Gauge, RotateCcw, Play } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
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

  const initializeDefaults = () => {
    const defaults: Record<string, string> = {};
    benchmarkParams.forEach((param) => {
      if (param.defaultValue) {
        defaults[param.fullName] = param.defaultValue;
      }
    });
    setParamValues(defaults);
  };

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
  // Use ref to prevent duplicate requests
  const isLoadingDevicesRef = useRef(false);

  useEffect(() => {
    if (isOpen && llamaCppPath && !isLoadingDevicesRef.current) {
      const loadDevices = async () => {
        if (isLoadingDevicesRef.current) return;
        isLoadingDevicesRef.current = true;
        
        setDevicesLoading(true);
        try {
          const response = await fetch(`/api/model/device/list?llamaBinPath=${encodeURIComponent(llamaCppPath)}`);

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
          console.error('Failed to load devices:', error);
          const errorMsg = error instanceof Error ? error.message : '未知错误';
          toast.error('无法加载计算设备列表', errorMsg);
          setAvailableDevices([]);
          setSelectedDevices([]);
        } finally {
          setDevicesLoading(false);
          isLoadingDevicesRef.current = false;
        }
      };
      loadDevices();
    }
    // Note: toast is stable and doesn't need to be in deps
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
  }, [isOpen, benchmarkParamsKeys]);
  useEffect(() => {
    if (isOpen && !wasOpen.current) {
      initializeDefaults();
      setAvailableDevices([]);
      setSelectedDevices([]);
    }
    wasOpen.current = isOpen;
  }, [isOpen, benchmarkParamsKeys]);

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
          <select
            id={id}
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
          >
            {param.values.map((v) => (
              <option key={v} value={v}>
                {v}
              </option>
            ))}
          </select>
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
          <select
            id={id}
            value={value || 'false'}
            onChange={(e) => handleParamChange(param.fullName, e.target.value)}
            disabled={isLoading}
            className={cn(
              "w-full px-2 py-1.5 text-sm",
              "border border-border rounded-md",
              "bg-input text-foreground",
              "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}
          >
            <option value="true">true</option>
            <option value="false">false</option>
          </select>
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
          <input
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
          <input
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
        <input
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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-lg shadow-xl w-full max-w-5xl max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border flex-shrink-0">
          <div className="flex items-center gap-2">
            <Gauge className="w-5 h-5 text-blue-500" />
            <h2 className="text-lg font-semibold text-foreground">
              模型性能测试
            </h2>
          </div>
          <button
            onClick={onClose}
            disabled={isLoading}
            className="p-1 text-muted-foreground hover:text-foreground disabled:opacity-50 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

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
                    <select
                      value={llamaCppPath}
                      onChange={(e) => setLlamaCppPath(e.target.value)}
                      disabled={isLoading || versionsLoading}
                      className={cn(
                        "w-full px-3 py-2 text-sm",
                        "border border-border rounded-md",
                        "bg-input text-foreground",
                        "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
                        "disabled:opacity-50 disabled:cursor-not-allowed"
                      )}
                    >
                      {llamaCppVersions.length > 0 ? (
                        llamaCppVersions.map((version) => (
                          <option key={version.path} value={version.path}>
                            {version.name || version.path} {version.description && `(${version.description})`}
                          </option>
                        ))
                      ) : (
                        <option value="">未配置 llama.cpp 路径</option>
                      )}
                    </select>
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
                                <input
                                  type="checkbox"
                                  checked={selectedDevices.includes(device)}
                                  onChange={() => handleDeviceToggle(device)}
                                  disabled={isLoading}
                                  className="rounded border-border text-blue-600 focus:ring-blue-500 w-3.5 h-3.5 mt-0.5 flex-shrink-0"
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
        <div className="flex justify-end items-center gap-2 px-4 py-3 border-t border-border bg-card flex-shrink-0">
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
        </div>
      </div>
    </div>
  );
}
