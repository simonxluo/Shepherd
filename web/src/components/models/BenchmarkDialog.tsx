import { useState, useEffect, useRef, useMemo } from 'react';
import { X, Loader2, Gauge, Save, FolderOpen, ChevronDown, Check, X as XIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import {
  useBenchmarkParams,
  useLlamaCppVersions,
  useSaveBenchmarkConfig,
  useBenchmarkConfigs,
} from '@/features/models/hooks';
import type {
  BenchmarkConfig,
  BenchmarkParam,
  LlamaCppVersion,
} from '@/types';
import { useToast } from '@/hooks/useToast';
import { LoadConfigDialog } from './LoadConfigDialog';

interface BenchmarkDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (config: BenchmarkConfig) => void;
  modelId: string;
  modelName: string;
  isLoading?: boolean;
}

/**
 * 压测对话框组件
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

  // 获取压测参数列表
  const { data: benchmarkParams = [], isLoading: paramsLoading } = useBenchmarkParams();

  // 获取 Llama.cpp 版本列表
  const { data: llamaCppVersions = [], isLoading: versionsLoading } = useLlamaCppVersions();

  // 获取已保存的压测配置
  const { data: savedConfigs = [] } = useBenchmarkConfigs();

  // 保存配置的 mutation
  const saveConfig = useSaveBenchmarkConfig();

  // 使用 useMemo 稳定依赖，避免无限循环
  const llamaCppVersionsPath = useMemo(() => llamaCppVersions.map(v => v.path).join(','), [llamaCppVersions]);
  const benchmarkParamsKeys = useMemo(() => benchmarkParams.map(p => p.fullName).join(','), [benchmarkParams]);

  // 压测配置状态
  const [llamaCppPath, setLlamaCppPath] = useState<string>('');
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);
  const [availableDevices, setAvailableDevices] = useState<string[]>([]);
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [configName, setConfigName] = useState<string>('');
  const [showSaveConfig, setShowSaveConfig] = useState(false);
  const [showLoadConfig, setShowLoadConfig] = useState(false);

  // 使用 ref 跟踪上次打开的对话框状态
  const wasOpen = useRef(false);

  // 当选择 llama.cpp 路径时，加载可用设备
  useEffect(() => {
    if (isOpen && llamaCppPath) {
      const loadDevices = async () => {
        try {
          // 显示加载状态
          setAvailableDevices([]);
          setSelectedDevices([]);

          const response = await fetch(`/api/model/device/list?llamaBinPath=${encodeURIComponent(llamaCppPath)}`);

          if (!response.ok) {
            const errorData = await response.json().catch(() => ({}));
            throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
          }

          const data = await response.json();
          if (data.success && data.data?.devices) {
            const devices = data.data.devices;
            setAvailableDevices(devices);
            // 默认全选所有设备
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
        }
      };
      loadDevices();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, llamaCppPath]);

  // 当对话框打开时，初始化默认值（仅一次）
  useEffect(() => {
    // 检测从关闭到打开的状态变化
    if (isOpen && !wasOpen.current) {
      // 设置默认的 llama.cpp 路径
      if (llamaCppVersions.length > 0) {
        const defaultPath = llamaCppVersions[0].path;
        setLlamaCppPath(defaultPath);
      }
      // 初始化参数默认值
      if (benchmarkParams.length > 0) {
        const defaults: Record<string, string> = {};
        benchmarkParams.forEach((param) => {
          if (param.defaultValue) {
            defaults[param.fullName] = param.defaultValue;
          }
        });
        setParamValues(defaults);
      }
      // 重置其他状态
      setAvailableDevices([]);
      setSelectedDevices([]);
      setConfigName('');
      setShowSaveConfig(false);
    }
    // 更新 ref
    wasOpen.current = isOpen;
  }, [isOpen, llamaCppVersionsPath, benchmarkParamsKeys]);

  // 处理设备选择
  const handleDeviceToggle = (device: string) => {
    setSelectedDevices((prev) =>
      prev.includes(device)
        ? prev.filter((d) => d !== device)
        : [...prev, device]
    );
  };

  // 处理参数值变化
  const handleParamChange = (fullName: string, value: string) => {
    setParamValues((prev) => ({ ...prev, [fullName]: value }));
  };

  // 加载保存的配置
  const handleLoadConfig = async (configName: string) => {
    try {
      const response = await fetch(`/api/models/benchmark/configs/${encodeURIComponent(configName)}`);
      const data = await response.json();
      if (data.success && data.data) {
        const config = data.data as BenchmarkConfig;
        setLlamaCppPath(config.llamaCppPath);
        setSelectedDevices(config.devices || []);
        setParamValues(
          Object.entries(config.params).reduce(
            (acc, [key, value]) => ({ ...acc, [key]: String(value) }),
            {}
          )
        );
      }
    } catch (error) {
      console.error('Failed to load config:', error);
      toast.error('加载配置失败', error instanceof Error ? error.message : '未知错误');
    }
  };

  // 保存配置
  const handleSaveConfig = async () => {
    if (!configName.trim()) {
      toast.warning('请输入配置名称');
      return;
    }
    const config: BenchmarkConfig = {
      modelId,
      modelName,
      llamaCppPath,
      devices: selectedDevices,
      params: paramValues,
    };
    saveConfig.mutate(
      { name: configName, config },
      {
        onSuccess: () => {
          toast.success('配置保存成功');
          setShowSaveConfig(false);
          setConfigName('');
        },
        onError: (error) => {
          toast.error('保存配置失败', error.message);
        },
      }
    );
  };

  // 构建命令字符串
  const buildCommand = (): string => {
    const parts: string[] = [];

    // 添加参数
    Object.entries(paramValues).forEach(([key, value]) => {
      if (value === 'true') {
        parts.push(key);
      } else if (value !== 'false' && value !== '') {
        parts.push(key, value);
      }
    });

    // 添加设备参数
    if (selectedDevices.length > 0 && selectedDevices.length < availableDevices.length) {
      parts.push('-dev', selectedDevices.join('/'));
    }

    return parts.join(' ');
  };

  // 提交压测
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

    // 通过 onConfirm 传递配置，由父组件调用 API
    onConfirm(config);
  };

  // 渲染参数输入字段
  const renderParamField = (param: BenchmarkParam) => {
    const value = paramValues[param.fullName] || param.defaultValue || '';
    const id = `param-${param.fullName.replace(/[^a-zA-Z0-9]/g, '_')}`;

    if (param.values && param.values.length > 0) {
      // 枚举类型 - 下拉选择
      return (
        <div key={param.fullName} className="col-span-1">
          <label className="flex items-center text-xs font-medium text-foreground mb-1">
            {param.name}
            {param.abbreviation && ` (${param.abbreviation})`}
          </label>
          <div className="relative">
            <select
              id={id}
              value={value}
              onChange={(e) => handleParamChange(param.fullName, e.target.value)}
              disabled={isLoading}
              className={cn(
                "w-full px-2 py-1.5 pr-8 text-sm",
                "border-2 border-border",
                "rounded-md bg-input",
                "text-foreground",
                "appearance-none cursor-pointer",
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
            <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
          </div>
          {param.description && (
            <p className="text-xs text-muted-foreground mt-1">{param.description}</p>
          )}
        </div>
      );
    }

    if (param.type === 'LOGIC') {
      // 布尔类型 - 下拉选择 true/false
      return (
        <div key={param.fullName} className="col-span-1">
          <label className="flex items-center text-xs font-medium text-foreground mb-1">
            {param.name}
            {param.abbreviation && ` (${param.abbreviation})`}
          </label>
          <div className="relative">
            <select
              id={id}
              value={value || 'false'}
              onChange={(e) => handleParamChange(param.fullName, e.target.value)}
              disabled={isLoading}
              className={cn(
                "w-full px-2 py-1.5 pr-8 text-sm",
                "border-2 border-border",
                "rounded-md bg-input",
                "text-foreground",
                "appearance-none cursor-pointer",
                "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
                "disabled:opacity-50 disabled:cursor-not-allowed"
              )}
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </select>
            <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
          </div>
          {param.description && (
            <p className="text-xs text-muted-foreground mt-1">{param.description}</p>
          )}
        </div>
      );
    }

    if (param.type === 'INTEGER') {
      // 整数类型
      return (
        <div key={param.fullName} className="col-span-1">
          <label className="flex items-center text-xs font-medium text-foreground mb-1">
            {param.name}
            {param.abbreviation && ` (${param.abbreviation})`}
          </label>
          <input
            id={id}
            type="number"
            value={value}
            onChange={(e) => handleParamChange(param.fullName, e.target.value)}
            disabled={isLoading}
            className={cn(
              "w-full px-2 py-1.5 text-sm",
              "border-2 border-border",
              "rounded-md bg-input",
              "text-foreground",
              "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}
          />
          {param.description && (
            <p className="text-xs text-muted-foreground mt-1">{param.description}</p>
          )}
        </div>
      );
    }

    if (param.type === 'FLOAT') {
      // 浮点类型
      return (
        <div key={param.fullName} className="col-span-1">
          <label className="flex items-center text-xs font-medium text-foreground mb-1">
            {param.name}
            {param.abbreviation && ` (${param.abbreviation})`}
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
              "border-2 border-border",
              "rounded-md bg-input",
              "text-foreground",
              "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}
          />
          {param.description && (
            <p className="text-xs text-muted-foreground mt-1">{param.description}</p>
          )}
        </div>
      );
    }

    // 默认字符串类型
    return (
      <div key={param.fullName} className="col-span-1">
        <label className="flex items-center text-xs font-medium text-foreground mb-1">
          {param.name}
          {param.abbreviation && ` (${param.abbreviation})`}
        </label>
        <input
          id={id}
          type="text"
          value={value}
          onChange={(e) => handleParamChange(param.fullName, e.target.value)}
          disabled={isLoading}
          className={cn(
            "w-full px-2 py-1.5 text-sm",
            "border-2 border-border",
            "rounded-md bg-input",
            "text-foreground",
            "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
            "disabled:opacity-50 disabled:cursor-not-allowed"
          )}
        />
        {param.description && (
          <p className="text-xs text-muted-foreground mt-1">{param.description}</p>
        )}
      </div>
    );
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] flex flex-col">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b border-border flex-shrink-0">
          <div className="flex items-center gap-2">
            <Gauge className="w-5 h-5 text-blue-500" />
            <h2 className="text-lg font-semibold text-foreground">
              模型性能测试
            </h2>
          </div>
          <button
            onClick={onClose}
            disabled={isLoading}
            className="p-1 text-muted-foreground hover:text-foreground disabled:opacity-50"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* 表单内容 */}
        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto p-4 min-h-0">
          <div className="space-y-6">
            {/* 模型信息 */}
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">
                模型
              </label>
              <div className="px-3 py-2 bg-muted rounded-md text-foreground text-sm">
                {modelName}
              </div>
            </div>

            {/* Llama.cpp 版本选择 */}
            <div>
              <label className="block text-sm font-medium text-foreground mb-1">
                Llama.cpp 版本
              </label>
              <div className="relative">
                <select
                  value={llamaCppPath}
                  onChange={(e) => setLlamaCppPath(e.target.value)}
                  disabled={isLoading || versionsLoading}
                  className={cn(
                    "w-full px-3 py-2 pr-8 text-sm",
                    "border-2 border-border",
                    "rounded-md bg-input",
                    "text-foreground",
                    "appearance-none cursor-pointer",
                    "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
                    "disabled:opacity-50 disabled:cursor-not-allowed"
                  )}
                >
                  {llamaCppVersions.length > 0 ? (
                    llamaCppVersions.map((version) => (
                      <option key={version.path} value={version.path}>
                        {version.name} {version.description && `(${version.description})`}
                      </option>
                    ))
                  ) : (
                    <option value="">未配置 llama.cpp 路径</option>
                  )}
                </select>
                <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
              </div>
            </div>

            {/* 设备选择 */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-sm font-medium text-foreground">
                  计算设备 (-dev)
                </label>
                {/* Removed redundant buttons as requested */}
              </div>
              <p className="text-xs text-muted-foreground mb-2">
                默认已勾选全部设备；取消勾选可排除设备；未选择设备时，使用 auto
              </p>
              <div className="border border-border rounded-lg p-3 bg-card max-h-48 overflow-y-auto">
                {availableDevices.length > 0 ? (
                  <div className="space-y-1">
                    {availableDevices.map((device, index) => {
                      // 解析设备信息: "ROCm0: AMD Radeon Graphics (122880 MiB, 114915 MiB free)"
                      const deviceLine = device.trim();
                      const parts = deviceLine.split(':');
                      const deviceId = parts[0]?.trim() || deviceLine;
                      const deviceDesc = parts.slice(1).join(':').trim() || deviceLine;

                      return (
                        <label
                          key={index}
                          className="flex items-start gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1.5 rounded"
                        >
                          <input
                            type="checkbox"
                            checked={selectedDevices.includes(device)}
                            onChange={() => handleDeviceToggle(device)}
                            disabled={isLoading}
                            className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4 mt-0.5"
                          />
                          <div className="flex-1 min-w-0">
                            <div className="font-medium text-foreground text-xs">{deviceId}</div>
                            <div className="text-xs text-muted-foreground break-all">{deviceDesc}</div>
                          </div>
                        </label>
                      );
                    })}
                  </div>
                ) : (
                  <div className="text-sm text-muted-foreground text-center py-4">
                    {llamaCppPath ? '未发现可用设备' : '请先选择 Llama.cpp 版本'}
                  </div>
                )}
              </div>
              {availableDevices.length > 0 && (
                <div className="mt-2 text-xs text-muted-foreground">
                  已选择 {selectedDevices.length} / {availableDevices.length} 个设备
                </div>
              )}
            </div>

            {/* 压测参数 */}
            <div>
              <h3 className="text-sm font-semibold text-foreground pb-2 border-b border-border mb-3">
                压测参数
              </h3>
              {paramsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
                  <span className="ml-2 text-sm text-muted-foreground">加载参数中...</span>
                </div>
              ) : benchmarkParams.length > 0 ? (
                <div className="grid grid-cols-2 gap-3">
                  {benchmarkParams
                    .sort((a, b) => (a.sort || 0) - (b.sort || 0))
                    .map((param) => renderParamField(param))}
                </div>
              ) : (
                <div className="text-sm text-muted-foreground text-center py-4">
                  无可用参数
                </div>
              )}
            </div>

          </div>
        </form>

        {/* 按钮区域 */}
        <div className="flex justify-between items-center gap-3 px-4 py-3 border-t border-border bg-card flex-shrink-0">
          {/* 左侧：配置管理 */}
          <div className="flex items-center gap-2">
            <div className="relative flex items-center">
              {showSaveConfig ? (
                <div className="flex items-center gap-2 bg-muted p-1 rounded-md">
                  <input
                    type="text"
                    value={configName}
                    onChange={(e) => setConfigName(e.target.value)}
                    placeholder="配置名称"
                    className={cn(
                      "w-32 px-2 py-1 text-xs",
                      "border border-border",
                      "rounded bg-input",
                      "text-foreground",
                      "focus:outline-none focus:ring-1 focus:ring-blue-500"
                    )}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        handleSaveConfig();
                      } else if (e.key === 'Escape') {
                        setShowSaveConfig(false);
                      }
                    }}
                  />
                  <Button
                    type="button"
                    size="sm"
                    className="h-6 px-2 text-xs"
                    onClick={handleSaveConfig}
                    disabled={saveConfig.isPending || !configName.trim()}
                  >
                    {saveConfig.isPending ? <Loader2 className="w-3 h-3 animate-spin" /> : '确定'}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-6 px-2 text-xs"
                    onClick={() => {
                      setShowSaveConfig(false);
                      setConfigName('');
                    }}
                  >
                    取消
                  </Button>
                </div>
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setShowSaveConfig(true)}
                  disabled={isLoading}
                >
                  <Save className="w-4 h-4 mr-1" />
                  保存配置
                </Button>
              )}
            </div>

            {!showSaveConfig && savedConfigs.length > 0 && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowLoadConfig(true)}
                disabled={isLoading}
              >
                <FolderOpen className="w-4 h-4 mr-1" />
                加载配置
              </Button>
            )}
          </div>

          {/* 右侧：操作按钮 */}
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={onClose}
              disabled={isLoading}
            >
              取消
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
                <Gauge className="w-4 h-4 mr-2" />
                开始测试
              </>
            )}
          </Button>
          </div>
        </div>
      </div>

      {/* 加载配置对话框 */}
      {showLoadConfig && (
        <LoadConfigDialog
          isOpen={showLoadConfig}
          onClose={() => setShowLoadConfig(false)}
          onLoad={handleLoadConfig}
          configs={savedConfigs}
          isLoading={isLoading}
        />
      )}
    </div>
  );
}
