import { useState, useRef, useEffect } from 'react';
import { X, HelpCircle, Loader2, ChevronDown, Info, RotateCcw, ToggleLeft, ToggleRight, Save, Wand2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import type { LoadModelParams, ModelCapabilities } from '@/types';
import { useGPUs, useModelCapabilities, useSetModelCapabilities, useLlamacppBackends, useEstimateVRAM, useModelLoadConfig, useSaveModelLoadConfig, useDeleteModelLoadConfig, useAutoDetectCapabilities, type SystemGPUInfo, type LlamacppBackend } from '@/features/models/hooks';
import { useOnlineNodes } from '@/features/cluster/hooks';
import type { UnifiedNode } from '@/types';
import { useToast } from '@/hooks/useToast';

// NumberInput 组件 - 数字输入框
interface NumberInputProps {
  value: number | undefined;
  onChange: (value: number) => void;
  disabled?: boolean;
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
  className?: string;
  allowNegative?: boolean;
  allowMinusOne?: boolean;
}

const NumberInput = ({
  value,
  onChange,
  disabled,
  min,
  max,
  step = 1,
  placeholder,
  className = '',
  allowNegative = false,
  allowMinusOne = false,
}: NumberInputProps) => {
  const [inputValue, setInputValue] = useState(String(value ?? ''));
  const [error, setError] = useState('');

  // 同步外部 value 变化
  useEffect(() => {
    if (value !== undefined && String(value) !== inputValue) {
      setInputValue(String(value));
      setError('');
    }
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setInputValue(newValue);

    // 空值处理
    if (newValue === '') {
      setError('');
      return;
    }

    // 验证数字
    const num = Number(newValue);
    if (isNaN(num)) {
      setError('请输入有效数字');
      return;
    }

    // 验证范围
    if (min !== undefined && num < min && !(allowMinusOne && num === -1) && !(allowNegative && num < 0)) {
      setError(`最小值为 ${min}`);
      return;
    }
    if (max !== undefined && num > max) {
      setError(`最大值为 ${max}`);
      return;
    }

    // 特殊值验证
    if (allowMinusOne && num === -1) {
      setError('');
      onChange(-1);
      return;
    }

    if (allowNegative && num < 0) {
      setError('');
      onChange(num);
      return;
    }

    if (!allowNegative && !allowMinusOne && num < 0) {
      setError('不允许负值');
      return;
    }

    setError('');
    onChange(num);
  };

  const handleBlur = () => {
    if (inputValue === '' && value !== undefined) {
      setInputValue(String(value));
      setError('');
    }
  };

  return (
    <div>
      <input
        type="number"
        value={inputValue}
        onChange={handleChange}
        onBlur={handleBlur}
        disabled={disabled}
        min={allowMinusOne ? -1 : min}
        max={max}
        step={step}
        placeholder={placeholder}
        className={cn(
          "w-full px-2 py-1.5 text-sm",
          "border-2 rounded-md",
          error ? "border-red-500 dark:border-red-500" : "border-border",
          "bg-input",
          "text-foreground",
          "focus:outline-none focus:ring-2",
          error ? "focus:ring-red-500 focus:border-red-500" : "focus:ring-blue-500 focus:border-blue-500",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "transition-colors",
          className
        )}
      />
      {error && (
        <p className="mt-1 text-xs text-red-600 dark:text-red-400">{error}</p>
      )}
    </div>
  );
};

interface LoadModelDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (params: Partial<LoadModelParams>) => void;
  modelId: string;
  modelName: string;
  modelPath?: string;
  isLoading?: boolean;
}

// 预设配置
const PRESETS = {
  fast: {
    name: '快速加载',
    description: '最小化内存占用，快速启动',
    params: {
      ctxSize: 4096,
      batchSize: 512,
      gpuLayers: 20,
      flashAttention: true,
      kvCacheUnified: false,
    } as Partial<LoadModelParams>
  },
  balanced: {
    name: '均衡模式',
    description: '性能与内存的平衡',
    params: {
      ctxSize: 8192,
      batchSize: 1024,
      gpuLayers: 35,
      flashAttention: true,
      kvCacheUnified: true,
    } as Partial<LoadModelParams>
  },
  performance: {
    name: '性能优先',
    description: '最大化性能，需要更多内存',
    params: {
      ctxSize: 16384,
      batchSize: 2048,
      gpuLayers: 99,
      flashAttention: true,
      kvCacheUnified: true,
      uBatchSize: 512,
    } as Partial<LoadModelParams>
  },
  max: {
    name: '最大配置',
    description: '最高性能，适合高端硬件',
    params: {
      ctxSize: 131072,
      batchSize: 4096,
      gpuLayers: 999,
      flashAttention: true,
      noMmap: true,
      lockMemory: true,
    } as Partial<LoadModelParams>
  }
};

// 参数帮助说明
const PARAM_HELP = {
  ctxSize: '模型一次能处理的文本最大长度，单位token，值越大内存占用越高',
  batchSize: '每次推理处理的样本数量，影响吞吐量',
  threads: '模型运行的线程数，-1表示自动选择',
  threadsBatch: '批处理线程数，-1表示与threads相同',
  gpuLayers: '卸载到GPU的模型层数，-1表示全部，0表示仅CPU',
  temperature: '控制生成文本的随机性，值越高越多样',
  topP: '核采样阈值，值越低越保守',
  topK: '保留前K个最高概率的token',
  minP: '过滤概率低于此值的token',
  repeatPenalty: '惩罚重复token，值越高越抑制重复',
  repeatLastN: '重复惩罚考虑的最后N个token，0表示使用完整上下文',
  presencePenalty: '鼓励模型使用新token，避免重复',
  frequencyPenalty: '惩罚高频token，值越高越抑制常见词',
  typicalP: '典型采样，值越低输出越典型，1.0表示禁用',
  ignoreEos: '忽略结束符EOS，使模型持续生成',
  flashAttention: '启用Flash Attention加速，提升推理速度',
  noMmap: '禁止使用内存映射加载模型',
  lockMemory: '锁定模型到物理内存，防止被系统回收',
  uBatchSize: '微批大小，用于优化内存使用',
  parallelSlots: '并发处理的槽位数',
  kvCacheSize: 'KV缓存最大占用空间（MB），限制KV缓存使用的内存大小',
  kvCacheUnified: '启用共享KV缓存，提升多任务效率',
  kvCacheType: 'KV缓存的数据类型，f16精度较高，f32精度最高',
  splitMode: '多GPU分割模式：none=单GPU，layer=按层分割，row=按行分割',
  tensorSplit: '多GPU张量分割比例，逗号分隔如"3,1"表示GPU0占75%',
  contBatching: '启用连续批处理（动态批处理），提升多请求吞吐量',
  cachePrompt: '启用提示缓存，加快重复请求的响应速度',
  grammar: 'BNF语法约束，用于结构化输出如JSON',
  grammarFile: 'BNF语法文件路径，用于复杂结构化输出',
  lora: 'LoRA适配器路径，用于模型微调',
  loraScaled: '带缩放因子的LoRA，格式如"path:scale,path2:scale2"',
  chatTemplateKwargs: '聊天模板额外参数，JSON格式如\'{"key":"value"}\'',
  ropeScaling: 'RoPE扩展方法：linear、yarn等，用于扩展上下文',
  ropeScale: 'RoPE扩展因子，大于1.0扩展上下文长度',
  ropeFreqBase: 'RoPE基础频率，调整NTK感知缩放',
  ropeFreqScale: 'RoPE频率缩放因子，小于1.0扩展上下文',
};

export function LoadModelDialog({
  isOpen,
  onClose,
  onConfirm,
  modelId,
  modelName,
  modelPath,
  isLoading = false,
}: LoadModelDialogProps) {
  // 获取在线节点列表（用于节点选择）
  const { data: onlineNodes = [] } = useOnlineNodes();

  // 获取 llama.cpp 后端列表
  const { data: llamacppBackends = [] } = useLlamacppBackends();

  // 模型加载配置相关 hooks
  const { data: loadConfigData, isLoading: isLoadingConfig } = useModelLoadConfig(isOpen ? modelId : '');
  const saveModelLoadConfig = useSaveModelLoadConfig();
  const deleteModelLoadConfig = useDeleteModelLoadConfig();

  const autoDetectCapabilities = useAutoDetectCapabilities();

  const toast = useToast();

  // 初始化参数状态
  const [params, setParams] = useState<LoadModelParams>({
    modelId,
    ctxSize: 8192,
    batchSize: 4096,
    threads: 4,
    gpuLayers: 99,
    temperature: 0.7,
    topP: 0.95,
    topK: 40,
    repeatPenalty: 1.1,
    seed: -1,
    nPredict: -1,
    llamaCppPath: '/usr/local/bin',
    mainGpu: 'default',
    capabilities: {
      thinking: false,
      tools: false,
      translation: false,
      embedding: false,
    },
    flashAttention: true,
    noMmap: false,
    lockMemory: false,
    logitsAll: false,
    reranking: false,
    minP: 0.05,
    presencePenalty: 0.0,
    frequencyPenalty: 0.0,
    uBatchSize: 512,
    parallelSlots: 4,
    kvCacheSize: 8192,
    kvCacheUnified: true,
    kvCacheTypeK: 'f16',
    kvCacheTypeV: 'f16',
    directIo: 'default',
    disableJinja: false,
    chatTemplate: '',
    contextShift: false,
    extraArgs: '',
    // 新增参数默认值
    threadsBatch: 0,
    repeatLastN: 0,
    typicalP: 1.0,
    ignoreEos: false,
    splitMode: '',
    tensorSplit: '',
    contBatching: true,
    cachePrompt: true,
    grammar: '',
    grammarFile: '',
    lora: '',
    loraScaled: '',
    chatTemplateKwargs: '',
    ropeScaling: '',
    ropeScale: 0,
    ropeFreqBase: 0,
    ropeFreqScale: 0,
    // 参数启用状态（默认全部启用，用户可以选择禁用以使用默认值）
    enabled: {
      ctxSize: true,
      batchSize: true,
      threads: true,
      threadsBatch: false,
      gpuLayers: true,
      temperature: true,
      topP: true,
      topK: true,
      repeatPenalty: true,
      repeatLastN: false,
      seed: false,
      nPredict: false,
      minP: true,
      typicalP: false,
      presencePenalty: false,
      frequencyPenalty: false,
      ignoreEos: false,
      uBatchSize: true,
      parallelSlots: true,
      contBatching: false,
      cachePrompt: false,
      kvCacheSize: true,
      kvCacheUnified: true,
      kvCacheTypeK: true,
      kvCacheTypeV: true,
      flashAttention: true,
      noMmap: true,
      lockMemory: false,
      splitMode: false,
      tensorSplit: false,
      grammar: false,
      grammarFile: false,
      lora: false,
      loraScaled: false,
      chatTemplate: false,
      chatTemplateKwargs: false,
      disableJinja: false,
      ropeScaling: false,
      ropeScale: false,
      ropeFreqBase: false,
      ropeFreqScale: false,
      contextShift: false,
      directIo: false,
      logitsAll: false,
      reranking: false,
      timeout: false,
      alias: false,
    },
  });

  const [estimateResult, setEstimateResult] = useState<string | null>(null);
  const [loadingStatus, setLoadingStatus] = useState<string>('就绪');

  // 保存配置状态
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');

  // Tooltip 状态
  const [activeTooltip, setActiveTooltip] = useState<string | null>(null);

  // 调试：监控 params.enabled 的变化
  useEffect(() => {
    console.log('[LoadModelDialog] params.enabled changed:', params.enabled);
  }, [params.enabled]);

  // 当对话框打开时，重置保存状态和估算结果
  useEffect(() => {
    if (isOpen) {
      setSaveStatus('idle');
      setEstimateResult(null);
    }
  }, [isOpen]);

  // 获取 GPU 列表（依赖 params.llamaCppPath）
  const { data: gpuData } = useGPUs(params.llamaCppPath);
  const gpus = gpuData?.gpus || [];
  const gpuDevices = gpuData?.devices || [];

  // 获取模型能力配置
  const { data: savedCapabilities } = useModelCapabilities(isOpen ? modelId : '');

  // 设置模型能力的 mutation
  const setModelCapabilities = useSetModelCapabilities();

  // 显存估算 mutation
  const estimateVRAM = useEstimateVRAM();

  // 当对话框打开时，加载已保存的能力配置
  useEffect(() => {
    if (isOpen && savedCapabilities) {
      setParams(prev => ({
        ...prev,
        capabilities: {
          thinking: savedCapabilities.thinking || false,
          tools: savedCapabilities.tools || false,
          translation: prev.capabilities?.translation || false,
          embedding: savedCapabilities.embedding || false,
          reranking: savedCapabilities.rerank || false,
        },
      }));
    }
  }, [isOpen, savedCapabilities]);

  // 当对话框打开时，加载已保存的模型加载配置
  useEffect(() => {
    if (isOpen && loadConfigData && !isLoadingConfig) {
      if (loadConfigData.exists && loadConfigData.config) {
        // 从保存的配置中恢复参数
        const savedConfig = loadConfigData.config.config;
        setParams(prev => {
          // 提取 enabled，如果没有则使用默认值
          const savedEnabled = (savedConfig as any).enabled;
          return {
            ...prev,
            ...(savedConfig as Partial<LoadModelParams>),
            // 确保 enabled 字段总是存在
            enabled: savedEnabled || prev.enabled,
          };
        });
      }
      // 如果不存在保存的配置，使用默认值（不改变当前状态）
    }
  }, [isOpen, loadConfigData, isLoadingConfig]);

  // 处理能力变化，应用约束规则并保存
  const handleCapabilityChange = (key: string, value: boolean) => {
    setParams(prev => {
      const currentCaps = prev.capabilities || {};

      // 应用约束规则
      let newCaps = { ...currentCaps, [key]: value };
      let newReranking = prev.reranking || false;

      // 规则 1: embedding 和 reranking 互斥（embedding 是非聊天能力）
      if (key === 'embedding' && value) {
        newReranking = false;
        newCaps.thinking = false;
        newCaps.tools = false;
      } else if (key === 'reranking' && value) {
        newCaps.embedding = false;
        newCaps.thinking = false;
        newCaps.tools = false;
      }

      // 规则 2: thinking 或 tools 启用时，禁用 embedding 和 reranking
      if ((key === 'thinking' || key === 'tools') && value) {
        newCaps.embedding = false;
        newReranking = false;
      }

      // 保存到服务器（只保存聊天相关的能力）
      const capabilitiesToSave = {
        thinking: newCaps.thinking || false,
        tools: newCaps.tools || false,
        rerank: newReranking,
        embedding: newCaps.embedding || false,
      };

      setModelCapabilities.mutate({
        modelId,
        capabilities: capabilitiesToSave,
      });

      return {
        ...prev,
        capabilities: newCaps,
        reranking: newReranking,
      };
    });
  };

  // 处理保存配置
  const handleSaveConfig = async () => {
    setSaveStatus('saving');
    try {
      // 保存当前参数到服务器
      await saveModelLoadConfig.mutateAsync({
        modelId,
        config: params,
      });
      setSaveStatus('saved');
      // 3秒后重置状态
      setTimeout(() => setSaveStatus('idle'), 3000);
    } catch (error) {
      console.error('保存配置失败:', error);
      setSaveStatus('error');
      // 3秒后重置状态
      setTimeout(() => setSaveStatus('idle'), 3000);
    }
  };

  if (!isOpen) return null;

  // 参数过滤函数：根据 enabled 状态过滤掉未启用的参数
  const filterEnabledParams = (allParams: LoadModelParams): Partial<LoadModelParams> => {
    const enabled = allParams.enabled;
    if (!enabled) return allParams; // 如果没有 enabled 字段，返回全部参数

    const filtered: Partial<LoadModelParams> = {
      modelId: allParams.modelId, // 始终包含 modelId
      nodeId: allParams.nodeId,   // 始终包含 nodeId
    };

    // 遍历所有可能的参数，如果启用则添加到 filtered 中
    const paramKeys: (keyof LoadModelParams)[] = [
      'ctxSize', 'batchSize', 'threads', 'threadsBatch', 'gpuLayers',
      'temperature', 'topP', 'topK', 'repeatPenalty', 'repeatLastN',
      'seed', 'nPredict',
      'llamaCppPath', 'mainGpu',
      'flashAttention', 'noMmap', 'lockMemory',
      'logitsAll', 'reranking', 'minP',
      'presencePenalty', 'frequencyPenalty',
      'uBatchSize', 'parallelSlots',
      'kvCacheSize', 'kvCacheUnified', 'kvCacheTypeK', 'kvCacheTypeV',
      'directIo', 'disableJinja', 'chatTemplate', 'contextShift',
      'typicalP', 'ignoreEos',
      'splitMode', 'tensorSplit',
      'contBatching', 'cachePrompt',
      'grammar', 'grammarFile',
      'lora', 'loraScaled',
      'chatTemplateKwargs',
      'ropeScaling', 'ropeScale', 'ropeFreqBase', 'ropeFreqScale',
    ];

    for (const key of paramKeys) {
      const enabledKey = key as keyof NonNullable<LoadModelParams['enabled']>;
      if (enabled[enabledKey] === true && allParams[key] !== undefined) {
        (filtered as any)[key] = allParams[key];
      }
    }

    // 始终包含 extraArgs（用户自定义参数）
    if (allParams.extraArgs) {
      filtered.extraArgs = allParams.extraArgs;
    }

    return filtered;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoadingStatus('加载中...');

    // 保存当前配置
    try {
      await saveModelLoadConfig.mutateAsync({
        modelId,
        config: params,
      });
    } catch (error) {
      console.error('保存模型加载配置失败:', error);
      // 保存失败不影响加载操作
    }

    // 过滤参数，只发送启用的参数
    const filteredParams = filterEnabledParams(params);
    onConfirm(filteredParams);
  };

  const applyPreset = (presetParams: Partial<LoadModelParams>) => {
    setParams(prev => ({ ...prev, ...presetParams }));
  };

  // 重置为默认配置
  const handleResetConfig = async () => {
    // 删除保存的配置
    try {
      await deleteModelLoadConfig.mutateAsync(modelId);
    } catch (error) {
      console.error('删除模型加载配置失败:', error);
    }

    // 重置为默认参数
    setParams({
      modelId,
      ctxSize: 8192,
      batchSize: 4096,
      threads: 4,
      gpuLayers: 99,
      temperature: 0.7,
      topP: 0.95,
      topK: 40,
      repeatPenalty: 1.1,
      seed: -1,
      nPredict: -1,
      llamaCppPath: '/usr/local/bin',
      mainGpu: 'default',
      capabilities: {
        thinking: false,
        tools: false,
        translation: false,
        embedding: false,
      },
      flashAttention: true,
      noMmap: false,
      lockMemory: false,
      logitsAll: false,
      reranking: false,
      minP: 0.05,
      presencePenalty: 0.0,
      frequencyPenalty: 0.0,
      uBatchSize: 512,
      parallelSlots: 4,
      kvCacheSize: 8192,
      kvCacheUnified: true,
      kvCacheTypeK: 'f16',
      kvCacheTypeV: 'f16',
      directIo: 'default',
      disableJinja: false,
      chatTemplate: '',
      contextShift: false,
      extraArgs: '',
      // 新增参数默认值
      threadsBatch: 0,
      repeatLastN: 0,
      typicalP: 1.0,
      ignoreEos: false,
      splitMode: '',
      tensorSplit: '',
      contBatching: true,
      cachePrompt: true,
      grammar: '',
      grammarFile: '',
      lora: '',
      loraScaled: '',
      chatTemplateKwargs: '',
      ropeScaling: '',
      ropeScale: 0,
      ropeFreqBase: 0,
      ropeFreqScale: 0,
      // 参数启用状态
      enabled: {
        ctxSize: true,
        batchSize: true,
        threads: true,
        threadsBatch: false,
        gpuLayers: true,
        temperature: true,
        topP: true,
        topK: true,
        repeatPenalty: true,
        repeatLastN: false,
        seed: false,
        nPredict: false,
        minP: true,
        typicalP: false,
        presencePenalty: false,
        frequencyPenalty: false,
        ignoreEos: false,
        uBatchSize: true,
        parallelSlots: true,
        contBatching: false,
        cachePrompt: false,
        kvCacheSize: true,
        kvCacheUnified: true,
        kvCacheTypeK: true,
        kvCacheTypeV: true,
        flashAttention: true,
        noMmap: true,
        lockMemory: false,
        splitMode: false,
        tensorSplit: false,
        grammar: false,
        grammarFile: false,
        lora: false,
        loraScaled: false,
        chatTemplate: false,
        chatTemplateKwargs: false,
        disableJinja: false,
        ropeScaling: false,
        ropeScale: false,
        ropeFreqBase: false,
        ropeFreqScale: false,
        contextShift: false,
        directIo: false,
        logitsAll: false,
        reranking: false,
        timeout: false,
        alias: false,
      },
    });
  };

  // Tooltip 交互
  const handleTooltipEnter = (key: string) => {
    setActiveTooltip(key);
  };

  const handleTooltipLeave = () => {
    setActiveTooltip(null);
  };

  // 参数控制组件 - 包含帮助按钮和启用/禁用开关
  const ParamControl = ({ paramKey, showToggle = true }: { paramKey: string; showToggle?: boolean }) => {
    const helpText = PARAM_HELP[paramKey as keyof typeof PARAM_HELP];
    const isEnabled = params.enabled?.[paramKey as keyof NonNullable<typeof params.enabled>] ?? false;
    const buttonRef = useRef<HTMLButtonElement>(null);
    const tooltipRef = useRef<HTMLDivElement>(null);
    const [position, setPosition] = useState({ top: 0, left: 0 });

    const updatePosition = () => {
      if (buttonRef.current && activeTooltip === paramKey) {
        const rect = buttonRef.current.getBoundingClientRect();
        setPosition({
          top: rect.top - 8,
          left: rect.left + rect.width / 2,
        });
      }
    };

    useEffect(() => {
      if (activeTooltip === paramKey) {
        updatePosition();
        const handleScroll = () => updatePosition();
        window.addEventListener('scroll', handleScroll, true);
        window.addEventListener('resize', updatePosition);
        return () => {
          window.removeEventListener('scroll', handleScroll, true);
          window.removeEventListener('resize', updatePosition);
        };
      }
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [activeTooltip, paramKey]);

    useEffect(() => {
      if (activeTooltip !== paramKey) return;

      const handleClickOutside = (e: MouseEvent) => {
        if (
          buttonRef.current &&
          !buttonRef.current.contains(e.target as Node) &&
          tooltipRef.current &&
          !tooltipRef.current.contains(e.target as Node)
        ) {
          setActiveTooltip(null);
        }
      };

      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [activeTooltip, paramKey]);

    const handleToggleTooltip = () => {
      setActiveTooltip(prev => prev === paramKey ? null : paramKey);
    };

    const handleToggleEnabled = () => {
      console.log('[ToggleEnabled] 点击按钮:', paramKey);
      console.log('[ToggleEnabled] 当前 enabled:', params.enabled);
      const currentValue = params.enabled?.[paramKey as keyof typeof params.enabled] ?? false;
      console.log('[ToggleEnabled] 当前值:', currentValue, '将变为:', !currentValue);

      setParams(prevParams => {
        const newValue = !(prevParams.enabled?.[paramKey as keyof typeof prevParams.enabled] ?? false);
        console.log('[ToggleEnabled] setParams 调用, 新值:', newValue);
        return {
          ...prevParams,
          enabled: {
            ...prevParams.enabled,
            [paramKey]: newValue,
          },
        };
      });
    };

    return (
      <div className="relative inline-flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
        {/* 启用/禁用开关 */}
        {showToggle && (
          <button
            type="button"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              console.log('[Toggle Button] onClick triggered, paramKey:', paramKey);
              console.log('[Toggle Button] isLoading:', isLoading, 'isEnabled:', isEnabled);
              console.log('[Toggle Button] event:', e);
              handleToggleEnabled();
              // 点击后移除焦点，防止焦点转移到其他元素
              (e.currentTarget as HTMLButtonElement).blur();
            }}
            onMouseDown={(e) => {
              // 防止焦点转移
              e.preventDefault();
            }}
            disabled={isLoading}
            className={cn(
              "inline-flex items-center justify-center w-6 h-6 rounded transition-all duration-200",
              "focus:outline-none focus:ring-1 focus:ring-blue-500 dark:focus:ring-blue-400",
              isEnabled
                ? "text-green-600 dark:text-green-400 hover:text-green-700 dark:hover:text-green-300 hover:bg-green-50 dark:hover:bg-green-900/20"
                : "text-gray-400 dark:text-gray-600 hover:text-gray-500 dark:hover:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800/20",
              "disabled:opacity-50 disabled:cursor-not-allowed",
              "active:scale-95",
              "select-none"
            )}
            aria-label={isEnabled ? `禁用 ${paramKey}` : `启用 ${paramKey}`}
            title={isEnabled ? '已启用 - 点击禁用' : '已禁用 - 点击启用'}
            data-param-key={paramKey}
            data-is-enabled={String(isEnabled)}
            data-is-loading={String(isLoading)}
          >
            {isEnabled ? (
              <ToggleRight className="w-4 h-4" />
            ) : (
              <ToggleLeft className="w-4 h-4" />
            )}
          </button>
        )}

        {/* 帮助按钮 */}
        <div className="relative inline-block">
          <button
            ref={buttonRef}
            type="button"
            onClick={handleToggleTooltip}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                handleToggleTooltip();
              }
            }}
            className={cn(
              "w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium",
              "flex items-center justify-center",
              "bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600",
              "hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40",
              "hover:text-blue-600 dark:hover:text-blue-400",
              "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1",
              "transition-all duration-200 cursor-help shadow-sm hover:shadow",
              activeTooltip === paramKey && "from-blue-50 to-blue-100 dark:from-blue-900/40 dark:to-blue-800/40 text-blue-600 dark:text-blue-400",
              !isEnabled && "opacity-50"
            )}
            aria-label={`查看 ${paramKey} 的帮助说明`}
            aria-expanded={activeTooltip === paramKey}
            aria-controls={`tooltip-${paramKey}`}
          >
            ?
          </button>

          {/* Tooltip */}
          {activeTooltip === paramKey && (
            <div
              ref={tooltipRef}
              id={`tooltip-${paramKey}`}
              role="tooltip"
              className="fixed z-[100]"
              style={{
                top: `${position.top}px`,
                left: `${position.left}px`,
                transform: 'translateX(-50%) translateY(-100%)',
                animation: 'tooltipFadeIn 0.2s ease-out forwards',
              }}
            >
              <style>{`
                @keyframes tooltipFadeIn {
                  from {
                    opacity: 0;
                    transform: 'translateX(-50%) translateY(-10%)';
                  }
                  to {
                    opacity: 1;
                    transform: 'translateX(-50%) translateY(-100%)';
                  }
                }
              `}</style>
              <div className="relative mb-1.5">
                <div className="max-w-xs px-4 py-3 bg-background/95 backdrop-blur-xl rounded-xl shadow-2xl border border-white/10">
                  <div className="flex items-start gap-3">
                    <Info className="w-4 h-4 text-blue-400 mt-0.5 flex-shrink-0" />
                    <p className="text-sm text-foreground leading-relaxed">
                      {helpText || '暂无说明'}
                    </p>
                  </div>
                </div>

                {/* 下方箭头 */}
                <div className="absolute -bottom-1.5 left-1/2 -translate-x-1/2">
                  <div className="w-0 h-0 border-l-[6px] border-l-transparent border-r-[6px] border-r-transparent border-t-[6px] border-t-gray-900/95 backdrop-blur-xl" />
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    );
  };

  const renderHelpButton = (paramKey: string, showToggle = true) => <ParamControl paramKey={paramKey} showToggle={showToggle} />;

  // 检查参数是否启用
  const isParamEnabled = (paramKey: string): boolean => {
    return params.enabled?.[paramKey as keyof NonNullable<typeof params.enabled>] ?? false;
  };

  // 获取禁用状态（如果参数未启用，则禁用输入）
  const getInputDisabled = (paramKey: string): boolean => {
    return isLoading || !isParamEnabled(paramKey);
  };

  // 下拉选择组件 - 用于布尔值和固定选项
  const SelectInput = ({
    value,
    onChange,
    disabled,
    children,
    className = '',
  }: {
    value: string | number | undefined;
    onChange: (e: React.ChangeEvent<HTMLSelectElement>) => void;
    disabled?: boolean;
    children: React.ReactNode;
    className?: string;
  }) => (
    <div className="relative">
      <select
        value={value ?? ''}
        onChange={onChange}
        disabled={disabled}
        className={cn(
          "w-full px-2 py-1.5 pr-8 text-sm",
          "border-2 border-border",
          "rounded-md bg-input",
          "text-foreground",
          "appearance-none cursor-pointer",
          "hover:border-blue-400 dark:hover:border-blue-500",
          "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "transition-colors",
          className
        )}
      >
        {children}
      </select>
      <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
    </div>
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-lg shadow-xl max-w-6xl w-full max-h-[90vh] flex flex-col">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b border-border flex-shrink-0">
          <h2 className="text-lg font-semibold text-foreground">
            加载模型配置
          </h2>
          <button
            onClick={onClose}
            disabled={isLoading}
            className="p-1 text-muted-foreground hover:text-foreground disabled:opacity-50"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* 预设配置按钮 */}
        <div className="px-4 py-3 border-b border-border bg-muted/50 flex-shrink-0">
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-sm font-medium text-foreground mr-2">预设配置:</span>
              {Object.entries(PRESETS).map(([key, preset]) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => applyPreset(preset.params)}
                  disabled={isLoading}
                  className={cn(
                    "px-3 py-1.5 text-sm font-medium rounded-md transition-all duration-200",
                    "border shadow-sm",
                    "hover:shadow-md hover:-translate-y-px active:translate-y-0",
                    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                    // 根据预设类型使用不同的样式
                    key === 'fast' && "bg-secondary text-secondary-foreground border-border hover:bg-secondary/80",
                    key === 'balanced' && "bg-primary/10 text-primary border-primary/30 hover:bg-primary/15",
                    key === 'performance' && "bg-accent text-accent-foreground border-border hover:bg-accent/80",
                    key === 'max' && "bg-destructive/10 text-destructive border-destructive/30 hover:bg-destructive/15",
                    "disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:translate-y-0 disabled:hover:shadow-sm"
                  )}
                  title={preset.description}
                >
                  {preset.name}
                </button>
              ))}
            </div>

            {/* 重置配置按钮 */}
            <button
              type="button"
              onClick={handleResetConfig}
              disabled={isLoading}
              className={cn(
                "flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-all duration-200",
                "border shadow-sm",
                "hover:shadow-md hover:-translate-y-px active:translate-y-0",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                "bg-muted text-muted-foreground border-border hover:bg-muted/80",
                "disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:translate-y-0 disabled:hover:shadow-sm"
              )}
              title="重置为默认配置"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              重置
            </button>
          </div>
        </div>

        {/* 表单内容 */}
        <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
          <div className="flex-1 overflow-y-auto p-4 min-h-0">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* 左列：基础配置 */}
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-foreground pb-2 border-b border-border">
                基础配置
              </h3>

              {/* 模型信息 */}
              <div className="space-y-3">
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    模型
                  </label>
                  <div className="px-3 py-2 bg-muted rounded-md text-foreground text-sm">
                    {modelName}
                  </div>
                  {modelPath && (
                    <div className="mt-1 text-xs text-muted-foreground truncate">
                      {modelPath}
                    </div>
                  )}
                </div>

                {/* Llama.cpp 版本选择 */}
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    Llama.cpp 版本
                  </label>
                  <SelectInput
                    value={params.llamaCppPath || ''}
                    onChange={(e) => setParams({ ...params, llamaCppPath: e.target.value })}
                    disabled={isLoading}
                    className="w-full"
                  >
                    {llamacppBackends.length > 0 ? (
                      llamacppBackends.map((backend: LlamacppBackend) => (
                        <option
                          key={backend.path}
                          value={backend.path}
                          disabled={!backend.available}
                        >
                          {backend.name}
                          {backend.description && ` (${backend.description})`}
                          {!backend.available && ' - 不可用'}
                        </option>
                      ))
                    ) : (
                      <option value="" disabled>
                        未配置 llama.cpp 后端
                      </option>
                    )}
                  </SelectInput>
                  {llamacppBackends.length === 0 && (
                    <p className="mt-1 text-xs text-yellow-600 dark:text-yellow-400">
                      请在服务器配置中添加 llama.cpp 路径
                    </p>
                  )}
                </div>

                {/* 能力开关 */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <label className="block text-sm font-medium text-foreground">
                      能力
                    </label>
                    <button
                      type="button"
                      onClick={() => {
                        autoDetectCapabilities.mutateAsync(modelId)
                          .then((result) => {
                            if (result?.capabilities) {
                              setParams(prev => ({
                                ...prev,
                                capabilities: result.capabilities
                              }));
                              const detectedList: string[] = [];
                              if (result.capabilities.thinking) detectedList.push('思考能力');
                              if (result.capabilities.tools) detectedList.push('工具调用');
                              if (result.capabilities.embedding) detectedList.push('嵌入');
                              if (result.capabilities.rerank) detectedList.push('重排序');

                              if (detectedList.length > 0) {
                                toast.success('检测完成', detectedList.join(', '));
                              } else {
                                toast.info('检测完成', '未检测到特殊能力');
                              }
                            }
                          })
                          .catch((error) => {
                            toast.error('检测失败', error instanceof Error ? error.message : '未知错误');
                          });
                      }}
                      disabled={!modelId || autoDetectCapabilities.isPending}
                      className={cn(
                        "flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-all duration-200",
                        "border shadow-sm",
                        "hover:shadow-md hover:-translate-y-px active:translate-y-0",
                        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                        "bg-muted text-muted-foreground border-border hover:bg-muted/80",
                        "disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:translate-y-0 disabled:hover:shadow-sm"
                      )}
                    >
                      {autoDetectCapabilities.isPending ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <Wand2 className="w-3.5 h-3.5" />
                      )}
                      自动检测
                    </button>
                  </div>
                  <div className="border border-border rounded-lg p-3 bg-card">
                    <div className="space-y-2">
                      {/* 聊天能力 */}
                      <div className="text-xs text-muted-foreground uppercase tracking-wide mb-1">聊天能力</div>
                      {[
                        { key: 'thinking', label: '思考能力' },
                        {key: 'tools', label: '工具使用' },
                      ].map(({ key, label }) => (
                        <label key={key} className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded hover:bg-accent">
                          <input
                            type="checkbox"
                            checked={params.capabilities?.[key as keyof NonNullable<typeof params.capabilities>] || false}
                            onChange={(e) => handleCapabilityChange(key, e.target.checked)}
                            disabled={isLoading || (params.reranking || params.capabilities?.embedding)}
                            className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                          />
                          <span>{label}</span>
                        </label>
                      ))}

                      {/* 非聊天能力 */}
                      <div className="text-xs text-muted-foreground uppercase tracking-wide mb-1 mt-2">非聊天能力</div>
                      {[
                        {key: 'translation', label: '直译' },
                        {key: 'embedding', label: '嵌入' },
                      ].map(({ key, label }) => (
                        <label key={key} className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded hover:bg-accent">
                          <input
                            type="checkbox"
                            checked={params.capabilities?.[key as keyof NonNullable<typeof params.capabilities>] || false}
                            onChange={(e) => handleCapabilityChange(key, e.target.checked)}
                            disabled={isLoading || ((params.capabilities?.thinking || params.capabilities?.tools) && key === 'embedding')}
                            className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                          />
                          <span>{label}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    选择模型支持的功能能力（互斥：思考/工具与嵌入）
                  </p>
                </div>

                {/* 主GPU选择 */}
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    主GPU
                  </label>
                  <SelectInput
                    value={params.mainGpu || 'default'}
                    onChange={(e) => setParams({ ...params, mainGpu: e.target.value })}
                    disabled={isLoading || gpus.length === 0}
                    className="w-full"
                  >
                    <option value="default">默认</option>
                    {gpus.map((gpu: SystemGPUInfo) => (
                      <option key={gpu.id} value={gpu.id}>
                        {gpu.name}
                      </option>
                    ))}
                  </SelectInput>
                  {gpus.length === 0 && (
                    <p className="mt-1 text-xs text-yellow-600 dark:text-yellow-400">
                      未检测到GPU，请确保ROCm正确安装
                    </p>
                  )}
                </div>

                {/* 设备选择 */}
                <div>
                  <label className="block text-sm font-medium text-foreground mb-2">
                    设备
                  </label>
                  <div className="border border-border rounded-lg p-3 bg-card">
                    {gpus.length > 0 ? (
                      <div className="space-y-1">
                        {gpus.map((gpu: SystemGPUInfo) => (
                          <label
                            key={gpu.id}
                            className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded"
                          >
                            <input
                              type="checkbox"
                              checked={gpu.available}
                              disabled={true}
                              className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                            />
                            <span className="flex-1">
                              <span className="font-medium">{gpu.id}</span>
                              <span className="text-muted-foreground mx-1">·</span>
                              <span className="text-muted-foreground">{gpu.name}</span>
                              {gpu.totalMemory && (
                                <span className="ml-2 text-muted-foreground">
                                  总计 {gpu.totalMemory}
                                </span>
                              )}
                              {gpu.freeMemory && (
                                <span className="ml-2 text-green-600 dark:text-green-400">
                                  可用 {gpu.freeMemory}
                                </span>
                              )}
                            </span>
                            {gpu.available ? (
                              <span className="text-xs text-green-600 dark:text-green-400">就绪</span>
                            ) : (
                              <span className="text-xs text-red-600 dark:text-red-400">不可用</span>
                            )}
                          </label>
                        ))}
                      </div>
                    ) : (
                      <div className="text-sm text-muted-foreground text-center py-2">
                        未检测到GPU设备
                      </div>
                    )}
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    选择用于模型加载的设备
                  </p>
                </div>

                {/* 节点选择（仅多节点环境显示） */}
                {onlineNodes.length > 0 && (
                  <div>
                    <label className="block text-sm font-medium text-foreground mb-1">
                      运行节点
                    </label>
                    <SelectInput
                      value={params.nodeId || 'auto'}
                      onChange={(e) => setParams({ 
                        ...params, 
                        nodeId: e.target.value === 'auto' ? undefined : e.target.value 
                      })}
                      disabled={isLoading}
                      className="w-full"
                    >
                      <option value="auto">
                        🎯 自动调度（推荐）- 系统选择最佳节点
                      </option>
                      <optgroup label="指定节点">
                        {onlineNodes.map((node: UnifiedNode) => (
                          <option key={node.id} value={node.id}>
                            {node.name} ({node.address}:{node.port})
                            {node.capabilities?.gpuCount && ` · ${node.capabilities.gpuCount} GPU`}
                            {node.resources?.gpuInfo?.[0] && 
                              ` · 显存 ${Math.round((node.resources.gpuInfo[0].totalMemory - node.resources.gpuInfo[0].usedMemory) / 1024 / 1024 / 1024)}GB 可用`
                            }
                          </option>
                        ))}
                      </optgroup>
                    </SelectInput>
                    <p className="mt-1 text-xs text-muted-foreground">
                      自动调度会根据 GPU 显存和负载选择最佳节点
                    </p>
                  </div>
                )}

                {/* 设备状态 */}
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    加载状态
                  </label>
                  <div className="px-3 py-2 bg-muted rounded-md min-h-[40px] text-sm">
                    {isLoading && loadingStatus === '加载中...' ? (
                      <span className="flex items-center gap-2">
                        <Loader2 className="w-4 h-4 animate-spin" />
                        {loadingStatus}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">{loadingStatus}</span>
                    )}
                  </div>
                </div>
              </div>
            </div>

            {/* 右列：高级参数 */}
            <div className="space-y-4">
              <h3 className="text-sm font-semibold text-foreground pb-2 border-b border-border">
                高级参数
              </h3>

              {/* 上下文与加速 */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  上下文与加速
                </h4>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      上下文窗口
                      {renderHelpButton('ctxSize')}
                    </label>
                    <NumberInput
                      value={params.ctxSize}
                      onChange={(v) => setParams({ ...params, ctxSize: v })}
                      disabled={getInputDisabled('ctxSize')}
                      min={0}
                      max={131072}
                      step={1}
                      placeholder="8192"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      Flash Attention
                      {renderHelpButton('flashAttention')}
                    </label>
                    <SelectInput
                      value={params.flashAttention ? 'on' : 'off'}
                      onChange={(e) => setParams({ ...params, flashAttention: e.target.value === 'on' })}
                      disabled={getInputDisabled('flashAttention')}
                    >
                      <option value="on">on</option>
                      <option value="off">off</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      禁用内存映射
                      {renderHelpButton('noMmap')}
                    </label>
                    <SelectInput
                      value={params.noMmap ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, noMmap: e.target.value === 'true' })}
                      disabled={getInputDisabled('noMmap')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      锁定物理内存
                      {renderHelpButton('lockMemory')}
                    </label>
                    <SelectInput
                      value={params.lockMemory ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, lockMemory: e.target.value === 'true' })}
                      disabled={getInputDisabled('lockMemory')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      GPU层数
                      {renderHelpButton('gpuLayers')}
                    </label>
                    <NumberInput
                      value={params.gpuLayers}
                      onChange={(v) => setParams({ ...params, gpuLayers: v })}
                      disabled={getInputDisabled('gpuLayers')}
                      min={-1}
                      max={999}
                      step={1}
                      placeholder="-1 表示全部"
                      allowMinusOne={true}
                    />
                  </div>
                </div>
              </div>

              {/* 采样参数 */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  采样参数
                </h4>
                <div className="grid grid-cols-3 gap-2">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      温度
                      {renderHelpButton('temperature')}
                    </label>
                    <NumberInput
                      value={params.temperature}
                      onChange={(v) => setParams({ ...params, temperature: v })}
                      disabled={getInputDisabled('temperature')}
                      min={0}
                      max={2}
                      step={0.1}
                      placeholder="0.7"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      Top-P
                      {renderHelpButton('topP')}
                    </label>
                    <NumberInput
                      value={params.topP}
                      onChange={(v) => setParams({ ...params, topP: v })}
                      disabled={getInputDisabled('topP')}
                      min={0}
                      max={1}
                      step={0.05}
                      placeholder="0.95"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      Top-K
                      {renderHelpButton('topK')}
                    </label>
                    <NumberInput
                      value={params.topK}
                      onChange={(v) => setParams({ ...params, topK: v })}
                      disabled={getInputDisabled('topK')}
                      min={1}
                      max={1000}
                      step={1}
                      placeholder="40"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      Min-P
                      {renderHelpButton('minP')}
                    </label>
                    <NumberInput
                      value={params.minP}
                      onChange={(v) => setParams({ ...params, minP: v })}
                      disabled={getInputDisabled('minP')}
                      min={0}
                      max={1}
                      step={0.01}
                      placeholder="0.05"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      重复惩罚
                      {renderHelpButton('repeatPenalty')}
                    </label>
                    <NumberInput
                      value={params.repeatPenalty}
                      onChange={(v) => setParams({ ...params, repeatPenalty: v })}
                      disabled={getInputDisabled('repeatPenalty')}
                      min={0}
                      max={2}
                      step={0.05}
                      placeholder="1.1"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      存在惩罚
                      {renderHelpButton('presencePenalty')}
                    </label>
                    <NumberInput
                      value={params.presencePenalty}
                      onChange={(v) => setParams({ ...params, presencePenalty: v })}
                      disabled={getInputDisabled('presencePenalty')}
                      min={0}
                      max={2}
                      step={0.1}
                      placeholder="0.0"
                    />
                  </div>
                </div>
              </div>

              {/* 批处理与并发 */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  批处理与并发
                </h4>
                <div className="grid grid-cols-4 gap-2">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      批次大小
                      {renderHelpButton('batchSize')}
                    </label>
                    <NumberInput
                      value={params.batchSize}
                      onChange={(v) => setParams({ ...params, batchSize: v })}
                      disabled={getInputDisabled('batchSize')}
                      min={64}
                      max={16384}
                      step={64}
                      placeholder="4096"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      微批大小
                      {renderHelpButton('uBatchSize')}
                    </label>
                    <NumberInput
                      value={params.uBatchSize}
                      onChange={(v) => setParams({ ...params, uBatchSize: v })}
                      disabled={getInputDisabled('uBatchSize')}
                      min={64}
                      max={8192}
                      step={64}
                      placeholder="512"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      并发槽位
                      {renderHelpButton('parallelSlots')}
                    </label>
                    <NumberInput
                      value={params.parallelSlots}
                      onChange={(v) => setParams({ ...params, parallelSlots: v })}
                      disabled={getInputDisabled('parallelSlots')}
                      min={1}
                      max={128}
                      step={1}
                      placeholder="4"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      线程数
                      {renderHelpButton('threads')}
                    </label>
                    <NumberInput
                      value={params.threads}
                      onChange={(v) => setParams({ ...params, threads: v })}
                      disabled={getInputDisabled('threads')}
                      min={-1}
                      max={256}
                      step={1}
                      placeholder="-1 表示自动"
                      allowMinusOne={true}
                    />
                  </div>
                </div>
              </div>

              {/* KV缓存 */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  KV缓存
                </h4>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      缓存大小
                      {renderHelpButton('kvCacheSize')}
                    </label>
                    <NumberInput
                      value={params.kvCacheSize}
                      onChange={(v) => setParams({ ...params, kvCacheSize: v })}
                      disabled={getInputDisabled('kvCacheSize')}
                      min={0}
                      max={131072}
                      step={1}
                      placeholder="8192"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      统一缓存
                      {renderHelpButton('kvCacheUnified')}
                    </label>
                    <SelectInput
                      value={params.kvCacheUnified ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, kvCacheUnified: e.target.value === 'true' })}
                      disabled={getInputDisabled('kvCacheUnified')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="text-xs font-medium text-foreground mb-1">
                      KV类型K
                    </label>
                    <SelectInput
                      value={params.kvCacheTypeK || 'f16'}
                      onChange={(e) => setParams({ ...params, kvCacheTypeK: e.target.value })}
                      disabled={getInputDisabled('kvCacheTypeK')}
                    >
                      <option value="f32">f32</option>
                      <option value="f16">f16 (默认)</option>
                      <option value="bf16">bf16</option>
                      <option value="q8_0">q8_0</option>
                      <option value="q5_0">q5_0</option>
                      <option value="q5_1">q5_1</option>
                      <option value="q4_0">q4_0</option>
                      <option value="q4_1">q4_1</option>
                      <option value="iq4_nl">iq4_nl</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="text-xs font-medium text-foreground mb-1">
                      KV类型V
                    </label>
                    <SelectInput
                      value={params.kvCacheTypeV || 'f16'}
                      onChange={(e) => setParams({ ...params, kvCacheTypeV: e.target.value })}
                      disabled={getInputDisabled('kvCacheTypeV')}
                    >
                      <option value="f32">f32</option>
                      <option value="f16">f16 (默认)</option>
                      <option value="bf16">bf16</option>
                      <option value="q8_0">q8_0</option>
                      <option value="q5_0">q5_0</option>
                      <option value="q5_1">q5_1</option>
                      <option value="q4_0">q4_0</option>
                      <option value="q4_1">q4_1</option>
                      <option value="iq4_nl">iq4_nl</option>
                    </SelectInput>
                  </div>
                </div>
              </div>

              {/* 其他参数 */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  其他参数
                </h4>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-xs font-medium text-foreground mb-1">
                      随机种子
                    </label>
                    <NumberInput
                      value={params.seed}
                      onChange={(v) => setParams({ ...params, seed: v })}
                      disabled={getInputDisabled('seed')}
                      min={-1}
                      max={4294967295}
                      step={1}
                      placeholder="-1 表示随机"
                      allowMinusOne={true}
                    />
                  </div>

                  <div>
                    <label className="text-xs font-medium text-foreground mb-1">
                      Max Tokens
                    </label>
                    <NumberInput
                      value={params.nPredict}
                      onChange={(v) => setParams({ ...params, nPredict: v })}
                      disabled={isLoading}
                      min={-1}
                      max={65536}
                      step={64}
                      placeholder="-1 表示无限"
                      allowMinusOne={true}
                    />
                  </div>

                  <div>
                    <label className="text-xs font-medium text-foreground mb-1">
                      DirectIO
                    </label>
                    <SelectInput
                      value={params.directIo || 'default'}
                      onChange={(e) => setParams({ ...params, directIo: e.target.value })}
                      disabled={isLoading}
                    >
                      <option value="default">default</option>
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div className="col-span-2">
                    <label className="text-xs font-medium text-foreground mb-1">
                      额外参数
                    </label>
                    <textarea
                      value={params.extraArgs || ''}
                      onChange={(e) => setParams({ ...params, extraArgs: e.target.value })}
                      disabled={isLoading}
                      rows={2}
                      placeholder="例如: --timeout 30 --grp-attn-n 8"
                      className="w-full px-2 py-1.5 text-sm border-2 border-border rounded-md bg-input text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 resize-y"
                    />
                    <p className="text-xs text-muted-foreground mt-1">
                      输入额外的命令行参数，用空格分隔
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
          </div>

          {/* 按钮区域 - 固定在底部 */}
          <div className="flex justify-end items-center gap-3 px-4 py-3 border-t border-border bg-card flex-shrink-0">
            {/* 取消按钮 - 右侧最左 */}
            <Button variant="outline" onClick={onClose} disabled={isLoading}>
              取消
            </Button>

            {/* 估算显存 + 结果 */}
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                onClick={async () => {
                  setEstimateResult('计算中...');

                  try {
                    const result = await estimateVRAM.mutateAsync({
                      modelId,
                      llamaBinPath: params.llamaCppPath || '/home/user/workspace/llama.cpp/build-rocm/bin',
                      ctxSize: params.ctxSize,
                      batchSize: params.batchSize,
                      uBatchSize: params.uBatchSize,
                      parallel: params.parallelSlots,
                      flashAttention: params.flashAttention,
                      kvUnified: params.kvCacheUnified,
                      cacheTypeK: params.kvCacheTypeK,
                      cacheTypeV: params.kvCacheTypeV,
                    });

                    if (result.vramGB) {
                      setEstimateResult(`约需 ${result.vramGB} GB 显存`);
                    } else if (result.error) {
                      setEstimateResult(`估算失败: ${result.error}`);
                    } else {
                      setEstimateResult('估算失败');
                    }
                  } catch (error) {
                    setEstimateResult(`估算出错: ${error instanceof Error ? error.message : '未知错误'}`);
                  }
                }}
                disabled={isLoading || estimateVRAM.isPending}
              >
                {estimateVRAM.isPending ? '计算中...' : '估算显存'}
              </Button>
              {estimateResult && (
                <span className="text-sm text-muted-foreground px-2 py-1 bg-muted rounded">{estimateResult}</span>
              )}
            </div>

            {/* 保存配置 */}
            <Button
              variant="secondary"
              onClick={handleSaveConfig}
              disabled={isLoading || saveStatus === 'saving'}
              className={cn(
                saveStatus === 'saved' && 'bg-green-600 text-white hover:bg-green-700',
                saveStatus === 'error' && 'bg-red-600 text-white hover:bg-red-700'
              )}
            >
              {saveStatus === 'saving' ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  保存中...
                </>
              ) : saveStatus === 'saved' ? (
                <>
                  ✓ 已保存
                </>
              ) : saveStatus === 'error' ? (
                <>
                  ✗ 保存失败
                </>
              ) : (
                <>
                  <Save className="w-4 h-4" />
                  保存配置
                </>
              )}
            </Button>

            {/* 开始加载 - 主操作 */}
            <Button type="submit" disabled={isLoading}>
              {isLoading ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  加载中...
                </>
              ) : (
                '开始加载'
              )}
            </Button>
          </div>
      </form>
    </div>
  </div>
  );
}
