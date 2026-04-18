import { useState, useRef, useEffect } from 'react';
import { X, Loader2, ChevronDown, Info, RotateCcw, ToggleLeft, ToggleRight, Save, Wand2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import type { LoadModelParams } from '@/types';
import { useGPUs, useModelCapabilities, useSetModelCapabilities, useLlamacppBackends, useEstimateVRAM, useModelLoadConfig, useSaveModelLoadConfig, useDeleteModelLoadConfig, useAutoDetectCapabilities, type SystemGPUInfo, type LlamacppBackend } from '@/features/models';
import { useOnlineNodes } from '@/features/cluster/hooks';
import type { UnifiedNode } from '@/types';
import { useToast } from '@/hooks/useToast';

// NumberInput component
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

  useEffect(() => {
    if (value !== undefined && String(value) !== inputValue) {
      setInputValue(String(value));
      setError('');
    }
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setInputValue(newValue);

    if (newValue === '') {
      setError('');
      return;
    }

    const num = Number(newValue);
    if (isNaN(num)) {
      setError('请输入有效数字');
      return;
    }

    if (min !== undefined && num < min && !(allowMinusOne && num === -1) && !(allowNegative && num < 0)) {
      setError(`最小值为 ${min}`);
      return;
    }
    if (max !== undefined && num > max) {
      setError(`最大值为 ${max}`);
      return;
    }

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

// Presets
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

// Parameter help descriptions
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
  seed: '随机种子，-1表示随机，固定值可复现结果',
  nPredict: '最大生成token数，-1表示无限',
  directIo: 'DirectIO模式，提升磁盘IO性能',
  disableJinja: '禁用Jinja模板处理',
  chatTemplate: '内置聊天模板名称或自定义模板',
  contextShift: '启用上下文移位，支持超长对话',
  extraArgs: '额外命令行参数',
  embedding: '嵌入向量模式，仅用于嵌入模型',
  noWebUI: '禁用内置Web界面',
  reasoning: '推理/思考模式：on启用，off禁用，auto自动检测',
  reasoningFormat: '推理输出格式：deepseek、auto等',
  reasoningBudget: '推理token预算，-1无限制，0立即结束',
  mmprojOffload: 'mmproj投影层GPU卸载',
  unloadAfterMinutes: '空闲自动卸载时间（分钟）。0=永不自动卸载，>0=自定义分钟数',
  concurrencyLimit: '最大并发请求数。0=不限，>0=自定义限制',
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
  const { data: onlineNodes = [] } = useOnlineNodes();

  const { data: llamacppBackends = [] } = useLlamacppBackends();

  const { data: loadConfigData, isLoading: isLoadingConfig } = useModelLoadConfig(isOpen ? modelId : '');
  const saveModelLoadConfig = useSaveModelLoadConfig();
  const deleteModelLoadConfig = useDeleteModelLoadConfig();

  const autoDetectCapabilities = useAutoDetectCapabilities();

  const toast = useToast();

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
    embedding: false,
    noWebUI: true,
    reasoning: 'auto',
    reasoningFormat: 'auto',
    reasoningBudget: -1,
    mmprojOffload: true,
    unloadAfterMinutes: 0,
    concurrencyLimit: 0,
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
      seed: true,
      nPredict: true,
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
      chatTemplate: true,
      chatTemplateKwargs: false,
      disableJinja: true,
      ropeScaling: false,
      ropeScale: false,
      ropeFreqBase: false,
      ropeFreqScale: false,
      contextShift: true,
      directIo: true,
      logitsAll: false,
      reranking: false,
      timeout: false,
      alias: false,
      embedding: false,
      noWebUI: true,
      reasoning: true,
      reasoningFormat: true,
      reasoningBudget: true,
      mmprojOffload: true,
      unloadAfterMinutes: true,
      concurrencyLimit: true,
    },
  });

  const [estimateResult, setEstimateResult] = useState<string | null>(null);
  const [loadingStatus, setLoadingStatus] = useState<string>('就绪');

  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');

  const [activeTooltip, setActiveTooltip] = useState<string | null>(null);

  useEffect(() => {
    console.log('[LoadModelDialog] params.enabled changed:', params.enabled);
  }, [params.enabled]);

  useEffect(() => {
    if (isOpen) {
      setSaveStatus('idle');
      setEstimateResult(null);
    }
  }, [isOpen]);

  const { data: gpuData } = useGPUs(params.llamaCppPath);
  const gpus = gpuData?.gpus || [];

  const { data: savedCapabilities } = useModelCapabilities(isOpen ? modelId : '');

  const setModelCapabilities = useSetModelCapabilities();

  const estimateVRAM = useEstimateVRAM();

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

  useEffect(() => {
    if (isOpen && loadConfigData && !isLoadingConfig) {
      if (loadConfigData.exists && loadConfigData.config) {
        const savedConfig = loadConfigData.config.config;
        setParams(prev => {
          const savedEnabled = (savedConfig as any).enabled;
          return {
            ...prev,
            ...(savedConfig as Partial<LoadModelParams>),
            enabled: savedEnabled || prev.enabled,
          };
        });
      }
    }
  }, [isOpen, loadConfigData, isLoadingConfig]);

  const handleCapabilityChange = (key: string, value: boolean) => {
    setParams(prev => {
      const currentCaps = prev.capabilities || {};

      let newCaps = { ...currentCaps, [key]: value };
      let newReranking = prev.reranking || false;

      // embedding and reranking are mutually exclusive with thinking/tools
      if (key === 'embedding' && value) {
        newReranking = false;
        newCaps.thinking = false;
        newCaps.tools = false;
      } else if (key === 'reranking' && value) {
        newCaps.embedding = false;
        newCaps.thinking = false;
        newCaps.tools = false;
      }

      if ((key === 'thinking' || key === 'tools') && value) {
        newCaps.embedding = false;
        newReranking = false;
      }

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

  const handleSaveConfig = async () => {
    setSaveStatus('saving');
    try {
      await saveModelLoadConfig.mutateAsync({
        modelId,
        config: params,
      });
      setSaveStatus('saved');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } catch (error) {
      console.error('Failed to save config:', error);
      setSaveStatus('error');
      setTimeout(() => setSaveStatus('idle'), 3000);
    }
  };

  if (!isOpen) return null;

  const filterEnabledParams = (allParams: LoadModelParams): Partial<LoadModelParams> => {
    const enabled = allParams.enabled;
    if (!enabled) return allParams;

    const filtered: Partial<LoadModelParams> = {
      modelId: allParams.modelId,
      nodeId: allParams.nodeId,
    };

    const paramKeys: (keyof LoadModelParams)[] = [
      'ctxSize', 'batchSize', 'threads', 'threadsBatch', 'gpuLayers',
      'temperature', 'topP', 'topK', 'repeatPenalty', 'repeatLastN',
      'seed', 'nPredict',
      'llamaCppPath', 'mainGpu',
      'flashAttention', 'noMmap', 'lockMemory', 'embedding',
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
      'noWebUI', 'reasoning', 'reasoningFormat', 'reasoningBudget', 'mmprojOffload',
      'unloadAfterMinutes', 'concurrencyLimit',
    ];

    for (const key of paramKeys) {
      const enabledKey = key as keyof NonNullable<LoadModelParams['enabled']>;
      if (enabled[enabledKey] === true && allParams[key] !== undefined) {
        (filtered as any)[key] = allParams[key];
      }
    }

    if (allParams.extraArgs) {
      filtered.extraArgs = allParams.extraArgs;
    }

    return filtered;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoadingStatus('加载中...');

    try {
      await saveModelLoadConfig.mutateAsync({
        modelId,
        config: params,
      });
    } catch (error) {
      console.error('Failed to save load config:', error);
    }

    const filteredParams = filterEnabledParams(params);
    onConfirm(filteredParams);
  };

  const applyPreset = (presetParams: Partial<LoadModelParams>) => {
    setParams(prev => ({ ...prev, ...presetParams }));
  };

  const handleResetConfig = async () => {
    try {
      await deleteModelLoadConfig.mutateAsync(modelId);
    } catch (error) {
      console.error('Failed to delete load config:', error);
    }

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
      embedding: false,
      noWebUI: true,
      reasoning: 'auto',
      reasoningFormat: 'auto',
      reasoningBudget: -1,
      mmprojOffload: true,
      unloadAfterMinutes: 0,
      concurrencyLimit: 0,
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
        seed: true,
        nPredict: true,
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
        chatTemplate: true,
        chatTemplateKwargs: false,
        disableJinja: true,
        ropeScaling: false,
        ropeScale: false,
        ropeFreqBase: false,
        ropeFreqScale: false,
        contextShift: true,
        directIo: true,
        extraArgs: true,
        logitsAll: false,
        reranking: false,
        timeout: false,
        alias: false,
        embedding: false,
        noWebUI: true,
        reasoning: true,
        reasoningFormat: true,
        reasoningBudget: true,
        mmprojOffload: true,
        unloadAfterMinutes: true,
        concurrencyLimit: true,
      },
    });
  };

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
        {/* Enable/disable toggle */}
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
              (e.currentTarget as HTMLButtonElement).blur();
            }}
            onMouseDown={(e) => {
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

        {/* Help button */}
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

                {/* Arrow */}
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

  const isParamEnabled = (paramKey: string): boolean => {
    return params.enabled?.[paramKey as keyof NonNullable<typeof params.enabled>] ?? false;
  };

  const getInputDisabled = (paramKey: string): boolean => {
    return isLoading || !isParamEnabled(paramKey);
  };

  // Dropdown select for booleans and fixed options
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
      <div className="bg-card rounded-lg shadow-xl border border-border max-w-5xl w-full max-h-[85vh] flex flex-col">
        {/* Header */}
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

        {/* Presets */}
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
                    "h-[34px] px-3 py-1.5 text-sm font-medium rounded-md transition-all duration-200",
                    "border shadow-sm",
                    "hover:shadow-md hover:-translate-y-px active:translate-y-0",
                    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                    // Different style per preset
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

            {/* Reset config */}
            <button
              type="button"
              onClick={handleResetConfig}
              disabled={isLoading}
              className={cn(
                "flex items-center justify-center gap-1.5 h-[34px] px-3 py-1.5 text-sm font-medium rounded-md transition-all duration-200",
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

        {/* Form content */}
        <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
          <div className="flex-1 min-h-0 p-4 overflow-hidden">
            <div className="flex flex-col lg:flex-row gap-6 h-full min-h-0">
            {/* Left column: basic config */}

            <div className="flex-1 space-y-4 overflow-y-auto pr-2 min-h-0" aria-label="基础配置区域">
              <h3 className="text-sm font-semibold text-foreground pb-2 border-b border-border">
                基础配置
              </h3>

              {/* Model info */}
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

                {/* Llama.cpp version */}
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

                {/* Capabilities */}
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
                        "flex items-center justify-center gap-1.5 h-[34px] px-3 py-1.5 text-sm font-medium rounded-md transition-all duration-200",
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
                      {/* Chat capabilities */}
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

                      {/* Non-chat capabilities */}
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

                {/* Main GPU */}
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

                {/* Devices */}
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

                {/* Node selection (multi-node only) */}
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

                {/* Device status */}
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

            {/* Right column: advanced params */}
            <div className="flex-1 space-y-4 overflow-y-auto pr-2 min-h-0" aria-label="高级参数区域">
              <h3 className="text-sm font-semibold text-foreground pb-2 border-b border-border">
                高级参数
              </h3>

              {/* Context & acceleration */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  上下文与加速
                </h4>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --ctx-size
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
                      --no-mmap
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
                      --embedding
                      {renderHelpButton('embedding')}
                    </label>
                    <SelectInput
                      value={params.embedding ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, embedding: e.target.value === 'true' })}
                      disabled={getInputDisabled('embedding')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --reranking
                      {renderHelpButton('reranking')}
                    </label>
                    <SelectInput
                      value={params.reranking ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, reranking: e.target.value === 'true' })}
                      disabled={getInputDisabled('reranking')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --gpu-layers
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

              {/* Sampling params */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  采样参数
                </h4>
                <div className="grid grid-cols-3 gap-3">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --temp
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
                      --repeat-penalty
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
                      --presence-penalty
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

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --frequency-penalty
                      {renderHelpButton('frequencyPenalty')}
                    </label>
                    <NumberInput
                      value={params.frequencyPenalty}
                      onChange={(v) => setParams({ ...params, frequencyPenalty: v })}
                      disabled={getInputDisabled('frequencyPenalty')}
                      min={0}
                      max={2}
                      step={0.1}
                      placeholder="0.0"
                    />
                  </div>
                </div>
              </div>

              {/* Batch & concurrency */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  批处理与并发
                </h4>
                <div className="grid grid-cols-4 gap-3">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --batch-size
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
                      --ubatch-size
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
                      --parallel(并发槽数)
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

              {/* Runtime management */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                  运行时管理
                </h4>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      空闲卸载时间 (分钟)
                      {renderHelpButton('unloadAfterMinutes')}
                    </label>
                    <NumberInput
                      value={params.unloadAfterMinutes}
                      onChange={(v) => setParams({ ...params, unloadAfterMinutes: v })}
                      disabled={getInputDisabled('unloadAfterMinutes')}
                      min={0}
                      max={10080}
                      step={1}
                      placeholder="0为永不自动卸载"
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      最大并发数
                      {renderHelpButton('concurrencyLimit')}
                    </label>
                    <NumberInput
                      value={params.concurrencyLimit}
                      onChange={(v) => setParams({ ...params, concurrencyLimit: v })}
                      disabled={getInputDisabled('concurrencyLimit')}
                      min={0}
                      max={10000}
                      step={1}
                      placeholder="0为不限"
                    />
                  </div>
                </div>
              </div>

                {/* KV cache */}
                <div className="space-y-3">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                    KV缓存
                  </h4>
                  <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --cache-ram
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
                      --kv-unified
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
                      -ctk
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
                      -ctv
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

                {/* Other params */}
                <div className="space-y-3">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                    其他参数
                  </h4>
                  <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --seed
                      {renderHelpButton('seed')}
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
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --n-predict
                      {renderHelpButton('nPredict')}
                    </label>
                    <NumberInput
                      value={params.nPredict}
                      onChange={(v) => setParams({ ...params, nPredict: v })}
                      disabled={getInputDisabled('nPredict')}
                      min={-1}
                      max={65536}
                      step={64}
                      placeholder="-1 表示无限"
                      allowMinusOne={true}
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --direct-io
                      {renderHelpButton('directIo')}
                    </label>
                    <SelectInput
                      value={params.directIo || 'default'}
                      onChange={(e) => setParams({ ...params, directIo: e.target.value })}
                      disabled={getInputDisabled('directIo')}
                    >
                      <option value="default">default</option>
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --no-webui
                      {renderHelpButton('noWebUI')}
                    </label>
                    <SelectInput
                      value={params.noWebUI ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, noWebUI: e.target.value === 'true' })}
                      disabled={getInputDisabled('noWebUI')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --no-jinja
                      {renderHelpButton('disableJinja')}
                    </label>
                    <SelectInput
                      value={params.disableJinja ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, disableJinja: e.target.value === 'true' })}
                      disabled={getInputDisabled('disableJinja')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --chat-template
                      {renderHelpButton('chatTemplate')}
                    </label>
                    <SelectInput
                      value={params.chatTemplate || ''}
                      onChange={(e) => setParams({ ...params, chatTemplate: e.target.value })}
                      disabled={getInputDisabled('chatTemplate')}
                    >
                      <option value="">(自动 - 使用模型元数据)</option>
                      {/* Bailing */}
                      <option value="bailing">bailing</option>
                      <option value="bailing-think">bailing-think</option>
                      <option value="bailing2">bailing2</option>
                      {/* ChatGLM */}
                      <option value="chatglm3">chatglm3</option>
                      <option value="chatglm4">chatglm4</option>
                      {/* DeepSeek */}
                      <option value="deepseek">deepseek</option>
                      <option value="deepseek2">deepseek2</option>
                      <option value="deepseek3">deepseek3</option>
                      {/* Exaone */}
                      <option value="exaone-moe">exaone-moe</option>
                      <option value="exaone3">exaone3</option>
                      <option value="exaone4">exaone4</option>
                      {/* Hunyuan */}
                      <option value="hunyuan-dense">hunyuan-dense</option>
                      <option value="hunyuan-moe">hunyuan-moe</option>
                      {/* Llama */}
                      <option value="llama2">llama2</option>
                      <option value="llama2-sys">llama2-sys</option>
                      <option value="llama2-sys-bos">llama2-sys-bos</option>
                      <option value="llama2-sys-strip">llama2-sys-strip</option>
                      <option value="llama3">llama3</option>
                      <option value="llama4">llama4</option>
                      {/* Mistral */}
                      <option value="mistral-v1">mistral-v1</option>
                      <option value="mistral-v3">mistral-v3</option>
                      <option value="mistral-v3-tekken">mistral-v3-tekken</option>
                      <option value="mistral-v7">mistral-v7</option>
                      <option value="mistral-v7-tekken">mistral-v7-tekken</option>
                      {/* Phi */}
                      <option value="phi3">phi3</option>
                      <option value="phi4">phi4</option>
                      {/* Vicuna */}
                      <option value="vicuna">vicuna</option>
                      <option value="vicuna-orca">vicuna-orca</option>
                      {/* Other templates */}
                      <option value="chatml">chatml</option>
                      <option value="command-r">command-r</option>
                      <option value="falcon3">falcon3</option>
                      <option value="gemma">gemma</option>
                      <option value="gigachat">gigachat</option>
                      <option value="glmedge">glmedge</option>
                      <option value="gpt-oss">gpt-oss</option>
                      <option value="granite">granite</option>
                      <option value="grok-2">grok-2</option>
                      <option value="kimi-k2">kimi-k2</option>
                      <option value="megrez">megrez</option>
                      <option value="minicpm">minicpm</option>
                      <option value="monarch">monarch</option>
                      <option value="openchat">openchat</option>
                      <option value="orion">orion</option>
                      <option value="pangu-embedded">pangu-embedded</option>
                      <option value="rwkv-world">rwkv-world</option>
                      <option value="seed_oss">seed_oss</option>
                      <option value="smolvlm">smolvlm</option>
                      <option value="solar-open">solar-open</option>
                      <option value="yandex">yandex</option>
                      <option value="zephyr">zephyr</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --context-shift
                      {renderHelpButton('contextShift')}
                    </label>
                    <SelectInput
                      value={params.contextShift ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, contextShift: e.target.value === 'true' })}
                      disabled={getInputDisabled('contextShift')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --reasoning
                      {renderHelpButton('reasoning')}
                    </label>
                    <SelectInput
                      value={params.reasoning || 'auto'}
                      onChange={(e) => setParams({ ...params, reasoning: e.target.value })}
                      disabled={getInputDisabled('reasoning')}
                    >
                      <option value="auto">auto</option>
                      <option value="on">on</option>
                      <option value="off">off</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --reasoning-format
                      {renderHelpButton('reasoningFormat')}
                    </label>
                    <SelectInput
                      value={params.reasoningFormat || 'auto'}
                      onChange={(e) => setParams({ ...params, reasoningFormat: e.target.value })}
                      disabled={getInputDisabled('reasoningFormat')}
                    >
                      <option value="auto">auto</option>
                      <option value="deepseek">deepseek</option>
                    </SelectInput>
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --reasoning-budget
                      {renderHelpButton('reasoningBudget')}
                    </label>
                    <NumberInput
                      value={params.reasoningBudget}
                      onChange={(v) => setParams({ ...params, reasoningBudget: v })}
                      disabled={getInputDisabled('reasoningBudget')}
                      min={-1}
                      max={100000}
                      step={256}
                      placeholder="-1 无限制"
                      allowMinusOne={true}
                    />
                  </div>

                  <div>
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      --no-mmproj-offload
                      {renderHelpButton('mmprojOffload')}
                    </label>
                    <SelectInput
                      value={params.mmprojOffload ? 'true' : 'false'}
                      onChange={(e) => setParams({ ...params, mmprojOffload: e.target.value === 'true' })}
                      disabled={getInputDisabled('mmprojOffload')}
                    >
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </SelectInput>
                  </div>

                  <div className="col-span-2">
                    <label className="flex items-center text-xs font-medium text-foreground mb-1">
                      其他参数
                      {renderHelpButton('extraArgs')}
                    </label>
                    <textarea
                      value={params.extraArgs || ''}
                      onChange={(e) => setParams({ ...params, extraArgs: e.target.value })}
                      disabled={getInputDisabled('extraArgs')}
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

          {/* Footer */}
          <div className="flex justify-end items-center gap-3 px-4 py-3 border-t border-border bg-card flex-shrink-0">
            {/* Cancel */}
            <Button variant="outline" onClick={onClose} disabled={isLoading}>
              取消
            </Button>

            {/* VRAM estimate */}
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

            {/* Save config */}
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

            {/* Load - primary action */}
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
