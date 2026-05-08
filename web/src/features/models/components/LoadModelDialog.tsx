import { useState, useEffect } from 'react';
import { X, Loader2, Info, Save, Trash2, Wand2, ToggleRight, ToggleLeft } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { Switch } from '@/components/ui/switch';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import type { LoadModelParams } from '@/types';
import { useModels, useGPUs, useModelCapabilities, useSetModelCapabilities, useLlamacppBackends, useEstimateVRAM, useModelLoadConfig, useSaveModelLoadConfig, useDeleteModelLoadConfig, useAutoDetectCapabilities, type SystemGPUInfo, type LlamacppBackend } from '@/features/models';
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
  const [prevValue, setPrevValue] = useState(value);

  if (value !== prevValue) {
    setPrevValue(value);
    if (value !== undefined && String(value) !== inputValue) {
      setInputValue(String(value));
      setError('');
    }
  }

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
      <Input
        type="text"
        inputMode="numeric"
        value={inputValue}
        onChange={handleChange}
        onBlur={handleBlur}
        disabled={disabled}
        placeholder={placeholder}
        className={cn(
          "h-8 px-2 py-1.5 text-sm",
          error && "border-red-500 dark:border-red-500 focus-visible:border-red-500 focus-visible:ring-red-500/50",
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
  backendType?: string; // 推荐后端类型 (llamacpp/vllm/vllm_omni)
}

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
  // vLLM 参数帮助
  vllmMaxModelLen: '模型能处理的最大序列长度（token 数），即上下文窗口大小',
  vllmGpuMemUtil: 'GPU 显存使用比例（0-1），默认 0.92。值越大分配越多显存给模型',
  vllmDtype: '模型权重数据类型。auto 自动选择，float16/bfloat16 半精度，float32 全精度',
  vllmTensorParallel: '张量并行使用的 GPU 数量，多 GPU 时设置可加速推理',
  vllmTrustRemoteCode: '允许执行模型仓库中的自定义代码，存在安全风险',
  vllmServedModelName: 'API 中使用的模型名称别名，用于替换实际模型路径',
  vllmQuantization: '模型量化方法，如 awq/gptq 可减少显存占用',
  vllmMaxNumSeqs: '单个迭代批次中最大并发序列数',
  vllmMaxNumBatchedTokens: '单次迭代中处理的最大 token 数量',
  vllmPrefixCaching: '启用前缀缓存，对相同前缀的请求可复用 KV 缓存',
  vllmChunkedPrefill: '启用分块预填充，将长 prompt 分块处理以降低延迟',
  vllmPipelineParallel: '流水线并行组数，用于跨节点分布式推理',
  vllmDisableLogRequests: '禁用请求日志输出，减少日志量',
  vllmVideoPruningRate: '视频 token 裁剪率（0-1），用于多模态模型的视频输入优化',
  vllmMmTensorIPC: '启用多模态张量 IPC，优化多模态模型的数据传输',
};

export function LoadModelDialog({
  isOpen,
  onClose,
  onConfirm,
  modelId,
  modelName,
  modelPath,
  isLoading = false,
  backendType,
}: LoadModelDialogProps) {
  const { data: onlineNodes = [] } = useOnlineNodes();

  const { data: llamacppBackends = [] } = useLlamacppBackends();

  const { data: modelsData } = useModels();
  const allModels = modelsData ?? [];
  const currentModel = allModels.find(m => m.id === modelId);
  const modelMaxCtxSize = currentModel?.metadata?.contextLength;

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
      tts: false,
      asr: false,
      imageGeneration: false,
      music: false,
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
    specDecoding: {
      specType: 'none',
    },
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
      specDecoding: false,
      unloadAfterMinutes: true,
      concurrencyLimit: true,
      extraArgs: true,
    },
  });

  const [estimateResult, setEstimateResult] = useState<string | null>(null);
  const [loadingStatus, setLoadingStatus] = useState<string>('就绪');

  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');

  const [configName, setConfigName] = useState('');
  const [selectedConfigName, setSelectedConfigName] = useState('');

  const CONFIGS_STORAGE_KEY = `shepherd:model-configs:${modelId}`;

  const getSavedConfigs = (): {name: string, config: LoadModelParams}[] => {
    try {
      const data = localStorage.getItem(CONFIGS_STORAGE_KEY);
      return data ? JSON.parse(data) : [];
    } catch {
      return [];
    }
  };

  const [savedConfigs, setSavedConfigs] = useState(getSavedConfigs);

  const handleLoadNamedConfig = (name: string) => {
    const configs = getSavedConfigs();
    const found = configs.find(c => c.name === name);
    if (found) {
      const config = { ...found.config };
      // Backward compatibility: migrate old draft fields to new specDecoding
      if (!config.specDecoding && (config as Record<string, unknown>).draftModelId) {
        config.specDecoding = {
          specType: 'draft',
          specDraftModelId: (config as Record<string, unknown>).draftModelId as string,
          specDraftNMax: (config as Record<string, unknown>).draftMaxTokens as number || 16,
        };
      }
      setParams(prev => ({
        ...prev,
        ...config,
        modelId: prev.modelId,
        enabled: config.enabled || prev.enabled,
      }));
      setSelectedConfigName(name);
      setConfigName(name);
    }
  };

  const handleDeleteNamedConfig = (name: string) => {
    const configs = getSavedConfigs().filter(c => c.name !== name);
    localStorage.setItem(CONFIGS_STORAGE_KEY, JSON.stringify(configs));
    setSavedConfigs([...configs]);
    if (selectedConfigName === name) {
      setSelectedConfigName('');
    }
  };

  useEffect(() => {
    if (isOpen) {
      setSaveStatus('idle');
      setEstimateResult(null);
      setConfigName('');
      setSelectedConfigName('');
      setSavedConfigs(getSavedConfigs());
      autoDetectCapabilities.mutate(modelId);
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
          tts: savedCapabilities.tts || false,
          asr: savedCapabilities.asr || false,
          imageGeneration: savedCapabilities.imageGeneration || false,
          music: savedCapabilities.music || false,
        },
      }));
    }
  }, [isOpen, savedCapabilities]);

  useEffect(() => {
    if (isOpen && loadConfigData && !isLoadingConfig) {
      if (loadConfigData.exists && loadConfigData.config) {
        const savedConfig = loadConfigData.config.config;
        setParams(prev => {
          const savedEnabled = (savedConfig as Partial<LoadModelParams>).enabled;
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
    let newCaps: Record<string, boolean>;
    let newReranking: boolean;

    setParams(prev => {
      const currentCaps = prev.capabilities || {};
      newCaps = { ...currentCaps, [key]: value };
      newReranking = prev.reranking || false;

      // Multimodal capabilities are mutually exclusive with chat capabilities
      const multimodalKeys = ['tts', 'asr', 'imageGeneration', 'music'];
      if (multimodalKeys.includes(key) && value) {
        newCaps.thinking = false;
        newCaps.tools = false;
        // Also disable other multimodal keys
        for (const mk of multimodalKeys) {
          if (mk !== key) newCaps[mk] = false;
        }
      }

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
        for (const mk of multimodalKeys) {
          newCaps[mk] = false;
        }
      }

      return {
        ...prev,
        capabilities: newCaps,
        reranking: newReranking,
      };
    });

    setTimeout(() => {
      setModelCapabilities.mutate({
        modelId,
        capabilities: {
          thinking: newCaps!.thinking || false,
          tools: newCaps!.tools || false,
          rerank: newReranking!,
          embedding: newCaps!.embedding || false,
          tts: newCaps!.tts || false,
          asr: newCaps!.asr || false,
          imageGeneration: newCaps!.imageGeneration || false,
          music: newCaps!.music || false,
        },
      });
    }, 0);
  };

  const handleSaveConfig = () => {
    const name = configName.trim();
    if (!name) {
      toast.error('请输入配置名称');
      return;
    }
    try {
      const configs = getSavedConfigs();
      const idx = configs.findIndex(c => c.name === name);
      if (idx >= 0) {
        configs[idx] = { name, config: params };
      } else {
        configs.push({ name, config: params });
      }
      localStorage.setItem(CONFIGS_STORAGE_KEY, JSON.stringify(configs));
      setSavedConfigs([...configs]);
      setSelectedConfigName(name);
      setSaveStatus('saved');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } catch (error) {
      console.error('Failed to save config:', error);
      setSaveStatus('error');
      setTimeout(() => setSaveStatus('idle'), 3000);
    }
  };

  if (!isOpen) return null;

  const isLlamaCpp = !backendType || backendType === 'llamacpp';
  const isVllmOmni = backendType === 'vllm_omni';
  const backendLabel = isVllmOmni ? 'vLLM-Omni' : backendType === 'vllm' ? 'vLLM' : 'llama.cpp';

  const filterEnabledParams = (allParams: LoadModelParams): Partial<LoadModelParams> => {
    const enabled = allParams.enabled;
    if (!enabled) return allParams;

    const filtered: Partial<LoadModelParams> = {
      modelId: allParams.modelId,
      nodeId: allParams.nodeId,
    };

    // vLLM 后端：只发送 vLLM 相关参数
    if (!isLlamaCpp) {
      const vllmKeys: (keyof LoadModelParams)[] = [
        'ctxSize', 'maxModelLen',
        'dtype', 'gpuMemoryUtilization', 'tensorParallelSize', 'pipelineParallelSize',
        'trustRemoteCode', 'servedModelName', 'quantization',
        'maxNumSeqs', 'maxNumBatchedTokens',
        'enablePrefixCaching', 'enableChunkedPrefill', 'disableLogRequests',
        'enforceEager',
        'unloadAfterMinutes', 'concurrencyLimit',
      ];

      // vLLM-Omni 专属：始终启用 omni 模式
      if (isVllmOmni) {
        filtered.omni = true;
        vllmKeys.push('videoPruningRate', 'mmTensorIPC');
      }

      for (const key of vllmKeys) {
        if (allParams[key] !== undefined && allParams[key] !== '' && allParams[key] !== 0 && allParams[key] !== false) {
          (filtered as Record<string, unknown>)[key] = allParams[key];
        }
      }

      if (allParams.extraArgs) {
        filtered.extraArgs = allParams.extraArgs;
      }

      if (allParams.envVars && allParams.envVars.length > 0) {
        filtered.envVars = allParams.envVars;
      }

      filtered.backendType = backendType;
      return filtered;
    }

    // llama.cpp 后端：原有逻辑
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
        (filtered as Record<string, unknown>)[key] = allParams[key];
      }
    }

    if (allParams.extraArgs) {
      filtered.extraArgs = allParams.extraArgs;
    }

    // Speculative decoding
    if (allParams.specDecoding && allParams.specDecoding.specType && allParams.specDecoding.specType !== 'none') {
      filtered.specDecoding = allParams.specDecoding;
    }

    // 传递后端类型
    if (backendType && backendType !== 'llamacpp') {
      filtered.backendType = backendType;
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

  // Dropdown select using shadcn Select
  interface SelectOption {
    value: string;
    label: string;
    disabled?: boolean;
  }

  interface SelectOptionGroup {
    label: string;
    options: SelectOption[];
  }

  const SelectInput = ({
    value,
    onValueChange,
    disabled,
    options,
    groups,
    className = '',
  }: {
    value: string | number | undefined;
    onValueChange: (value: string) => void;
    disabled?: boolean;
    options?: SelectOption[];
    groups?: SelectOptionGroup[];
    className?: string;
  }) => (
    <Select
      value={String(value ?? '')}
      onValueChange={onValueChange}
      disabled={disabled}
    >
      <SelectTrigger className={cn("h-8 w-full text-sm", className)}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent position="popper" sideOffset={4}>
        {groups ? (
          groups.map((group) => (
            <SelectGroup key={group.label}>
              <SelectLabel>{group.label}</SelectLabel>
              {group.options.map((opt) => (
                <SelectItem key={opt.value} value={opt.value} disabled={opt.disabled}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectGroup>
          ))
        ) : (
          options?.map((opt) => (
            <SelectItem key={opt.value} value={opt.value} disabled={opt.disabled}>
              {opt.label}
            </SelectItem>
          ))
        )}
      </SelectContent>
    </Select>
  );

  // 非 llama.cpp 后端使用 vLLM 参数对话框
  const [showAdvancedVllm, setShowAdvancedVllm] = useState(false);

  if (!isLlamaCpp) {
    return (
      <Dialog open={isOpen} onOpenChange={(open) => { if (!open && !isLoading) onClose(); }}>
        <DialogContent className="sm:max-w-[700px] max-h-[85vh] p-0 overflow-hidden flex flex-col">
          <DialogHeader className="p-4 border-b border-border flex-shrink-0">
            <DialogTitle className="text-lg font-semibold text-foreground">
              加载模型配置 — {backendLabel}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
            <div className="flex flex-col gap-4 p-4 overflow-y-auto flex-1">
              {/* 模型信息 */}
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">模型</label>
                <div className="px-3 py-2 bg-muted rounded-md text-foreground text-sm">
                  {modelName}
                </div>
                {modelPath && (
                  <div className="mt-1 text-xs text-muted-foreground truncate">{modelPath}</div>
                )}
              </div>

              {/* 后端类型标签 */}
              <div className="flex items-center gap-2">
                <span className="text-xs px-2 py-1 rounded bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 font-medium">
                  {backendLabel}
                </span>
                <span className="text-xs text-muted-foreground">
                  此模型需要使用 {backendLabel} 后端加载
                </span>
              </div>

              {/* 基本参数 */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">基本参数</h4>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --max-model-len
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                              ?
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-xs">
                            <p className="text-sm">{PARAM_HELP.vllmMaxModelLen}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                    <NumberInput
                      value={params.maxModelLen || params.ctxSize}
                      onChange={(v) => setParams({ ...params, maxModelLen: v, ctxSize: v })}
                      disabled={isLoading}
                      min={0}
                      max={modelMaxCtxSize}
                      step={256}
                      placeholder="8192"
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --gpu-memory-utilization
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                              ?
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-xs">
                            <p className="text-sm">{PARAM_HELP.vllmGpuMemUtil}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                    <NumberInput
                      value={params.gpuMemoryUtilization ?? 0.92}
                      onChange={(v) => setParams({ ...params, gpuMemoryUtilization: v })}
                      disabled={isLoading}
                      min={0}
                      max={1}
                      step={0.01}
                      placeholder="0.92"
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --dtype
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                              ?
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-xs">
                            <p className="text-sm">{PARAM_HELP.vllmDtype}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                    <SelectInput
                      value={params.dtype || 'auto'}
                      onValueChange={(v) => setParams({ ...params, dtype: v === 'auto' ? '' : v })}
                      disabled={isLoading}
                      options={[
                        { value: 'auto', label: 'auto (默认)' },
                        { value: 'float16', label: 'float16' },
                        { value: 'bfloat16', label: 'bfloat16' },
                        { value: 'float32', label: 'float32' },
                      ]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --tensor-parallel-size
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                              ?
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-xs">
                            <p className="text-sm">{PARAM_HELP.vllmTensorParallel}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                    <NumberInput
                      value={params.tensorParallelSize}
                      onChange={(v) => setParams({ ...params, tensorParallelSize: v })}
                      disabled={isLoading}
                      min={1}
                      max={16}
                      step={1}
                      placeholder="1"
                    />
                  </div>

                  <div className="col-span-2">
                    <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded">
                      <input
                        type="checkbox"
                        checked={params.trustRemoteCode || false}
                        onChange={(e) => setParams({ ...params, trustRemoteCode: e.target.checked })}
                        disabled={isLoading}
                        className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                      />
                      <span>--trust-remote-code</span>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button type="button" className="w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                              ?
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-xs">
                            <p className="text-sm">{PARAM_HELP.vllmTrustRemoteCode}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </label>
                  </div>
                </div>
              </div>

              {/* 高级参数（折叠区域） */}
              <div className="space-y-3">
                <button
                  type="button"
                  onClick={() => setShowAdvancedVllm(!showAdvancedVllm)}
                  className="flex items-center gap-1.5 text-xs font-semibold text-muted-foreground uppercase hover:text-foreground transition-colors"
                >
                  {showAdvancedVllm ? <ToggleRight className="w-4 h-4" /> : <ToggleLeft className="w-4 h-4" />}
                  高级参数
                </button>

                {showAdvancedVllm && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                        --served-model-name
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmServedModelName}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>
                      <Input
                        value={params.servedModelName || ''}
                        onChange={(e) => setParams({ ...params, servedModelName: e.target.value })}
                        disabled={isLoading}
                        className="h-8 text-sm"
                        placeholder="模型名称别名"
                      />
                    </div>

                    <div>
                      <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                        --quantization
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmQuantization}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>
                      <SelectInput
                        value={params.quantization || 'auto'}
                        onValueChange={(v) => setParams({ ...params, quantization: v === 'auto' ? '' : v })}
                        disabled={isLoading}
                        options={[
                          { value: 'auto', label: 'auto (默认)' },
                          { value: 'awq', label: 'awq' },
                          { value: 'gptq', label: 'gptq' },
                          { value: 'gptq_marlin', label: 'gptq_marlin' },
                          { value: 'gptq_marlin_24', label: 'gptq_marlin_24' },
                          { value: 'aqlm', label: 'aqlm' },
                          { value: 'fp8', label: 'fp8' },
                          { value: 'bitsandbytes', label: 'bitsandbytes' },
                          { value: 'gguf', label: 'gguf' },
                        ]}
                      />
                    </div>

                    <div>
                      <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                        --max-num-seqs
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmMaxNumSeqs}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>
                      <NumberInput
                        value={params.maxNumSeqs}
                        onChange={(v) => setParams({ ...params, maxNumSeqs: v })}
                        disabled={isLoading}
                        min={1}
                        max={1024}
                        step={1}
                        placeholder="256"
                      />
                    </div>

                    <div>
                      <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                        --max-num-batched-tokens
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmMaxNumBatchedTokens}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>
                      <NumberInput
                        value={params.maxNumBatchedTokens}
                        onChange={(v) => setParams({ ...params, maxNumBatchedTokens: v })}
                        disabled={isLoading}
                        min={1}
                        max={1048576}
                        step={256}
                        placeholder="自动"
                      />
                    </div>

                    <div>
                      <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                        --pipeline-parallel-size
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmPipelineParallel}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>
                      <NumberInput
                        value={params.pipelineParallelSize}
                        onChange={(v) => setParams({ ...params, pipelineParallelSize: v })}
                        disabled={isLoading}
                        min={1}
                        max={16}
                        step={1}
                        placeholder="1"
                      />
                    </div>

                    <div className="flex flex-col justify-end gap-2">
                      <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded">
                        <input
                          type="checkbox"
                          checked={params.enablePrefixCaching || false}
                          onChange={(e) => setParams({ ...params, enablePrefixCaching: e.target.checked })}
                          disabled={isLoading}
                          className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                        />
                        <span>--enable-prefix-caching</span>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmPrefixCaching}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </label>

                      <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded">
                        <input
                          type="checkbox"
                          checked={params.enableChunkedPrefill || false}
                          onChange={(e) => setParams({ ...params, enableChunkedPrefill: e.target.checked })}
                          disabled={isLoading}
                          className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                        />
                        <span>--enable-chunked-prefill</span>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmChunkedPrefill}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </label>

                      <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded">
                        <input
                          type="checkbox"
                          checked={params.enforceEager || false}
                          onChange={(e) => setParams({ ...params, enforceEager: e.target.checked })}
                          disabled={isLoading}
                          className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                        />
                        <span>--enforce-eager</span>
                      </label>

                      <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded">
                        <input
                          type="checkbox"
                          checked={params.disableLogRequests || false}
                          onChange={(e) => setParams({ ...params, disableLogRequests: e.target.checked })}
                          disabled={isLoading}
                          className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                        />
                        <span>--disable-log-requests</span>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmDisableLogRequests}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </label>
                    </div>
                  </div>
                )}
              </div>

              {/* vLLM-Omni 专属参数 */}
              {isVllmOmni && (
                <div className="space-y-3">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                    vLLM-Omni 专属参数
                  </h4>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                        --video-pruning-rate
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="ml-1.5 w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmVideoPruningRate}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </div>
                      <NumberInput
                        value={params.videoPruningRate}
                        onChange={(v) => setParams({ ...params, videoPruningRate: v })}
                        disabled={isLoading}
                        min={0}
                        max={1}
                        step={0.01}
                        placeholder="0"
                      />
                    </div>

                    <div className="flex items-end">
                      <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded">
                        <input
                          type="checkbox"
                          checked={params.mmTensorIPC || false}
                          onChange={(e) => setParams({ ...params, mmTensorIPC: e.target.checked })}
                          disabled={isLoading}
                          className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                        />
                        <span>--mm-tensor-ipc</span>
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button type="button" className="w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium flex items-center justify-center bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200 cursor-help shadow-sm hover:shadow">
                                ?
                              </button>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs">
                              <p className="text-sm">{PARAM_HELP.vllmMmTensorIPC}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </label>
                    </div>
                  </div>
                </div>
              )}

              {/* GPU 选择 */}
              {gpus.length > 0 && (
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">主GPU</label>
                  <SelectInput
                    value={params.mainGpu || 'default'}
                    onValueChange={(v) => setParams({ ...params, mainGpu: v })}
                    disabled={isLoading}
                    options={[
                      { value: 'default', label: '默认' },
                      ...gpus.map((gpu: SystemGPUInfo) => ({
                        value: gpu.id,
                        label: gpu.name,
                      })),
                    ]}
                  />
                </div>
              )}

              {/* 运行时管理 */}
              <div className="space-y-3">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase">运行时管理</h4>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">空闲自动卸载(分钟)</label>
                    <NumberInput
                      value={params.unloadAfterMinutes}
                      onChange={(v) => setParams({ ...params, unloadAfterMinutes: v })}
                      disabled={isLoading}
                      min={0}
                      max={10080}
                      step={1}
                      placeholder="0为永不自动卸载"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">最大并发请求数</label>
                    <NumberInput
                      value={params.concurrencyLimit}
                      onChange={(v) => setParams({ ...params, concurrencyLimit: v })}
                      disabled={isLoading}
                      min={0}
                      max={10000}
                      step={1}
                      placeholder="0为不限"
                    />
                  </div>
                </div>
              </div>

              {/* 额外参数 */}
              <div>
                <label className="block text-xs font-medium text-foreground mb-1">额外参数</label>
                <textarea
                  value={params.extraArgs || ''}
                  onChange={(e) => setParams({ ...params, extraArgs: e.target.value })}
                  disabled={isLoading}
                  rows={2}
                  placeholder="附加命令行参数，如 --disable-log-requests"
                  className="w-full px-2 py-1.5 text-sm border-2 border-border rounded-md bg-input text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 resize-y"
                />
              </div>

              {/* 环境变量配置 */}
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-xs font-medium text-foreground">环境变量</label>
                  <button
                    type="button"
                    onClick={() => {
                      const current = params.envVars || [];
                      setParams({ ...params, envVars: [...current, ''] });
                    }}
                    disabled={isLoading}
                    className="text-xs text-blue-600 dark:text-blue-400 hover:underline disabled:opacity-50"
                  >
                    + 添加
                  </button>
                </div>
                <div className="space-y-1.5">
                  {(params.envVars || []).length === 0 ? (
                    <div className="text-xs text-muted-foreground py-2 text-center border border-dashed border-border rounded-md">
                      点击"添加"配置环境变量，如 LD_LIBRARY_PATH=/path/to/lib
                    </div>
                  ) : (
                    (params.envVars || []).map((env, idx) => (
                      <div key={idx} className="flex items-center gap-1.5">
                        <input
                          value={env}
                          onChange={(e) => {
                            const newEnvs = [...(params.envVars || [])];
                            newEnvs[idx] = e.target.value;
                            setParams({ ...params, envVars: newEnvs });
                          }}
                          disabled={isLoading}
                          placeholder="KEY=VALUE"
                          className="flex-1 px-2 py-1 text-sm border-2 border-border rounded-md bg-input text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 font-mono"
                        />
                        <button
                          type="button"
                          onClick={() => {
                            const newEnvs = (params.envVars || []).filter((_, i) => i !== idx);
                            setParams({ ...params, envVars: newEnvs });
                          }}
                          disabled={isLoading}
                          className="p-1 text-muted-foreground hover:text-red-500 transition-colors disabled:opacity-50"
                        >
                          <X className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>

            {/* 底部按钮 */}
            <div className="flex items-center justify-end gap-3 p-4 border-t border-border flex-shrink-0">
              <Button type="button" variant="outline" onClick={onClose} disabled={isLoading}>
                取消
              </Button>
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
        </DialogContent>
      </Dialog>
    );
  }

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
        tts: false,
        asr: false,
        imageGeneration: false,
        music: false,
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
      specDecoding: {
        specType: 'none',
      },
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
        specDecoding: false,
        unloadAfterMinutes: true,
        concurrencyLimit: true,
      },
    });
  };

  const ParamControl = ({ paramKey, showToggle = true }: { paramKey: string; showToggle?: boolean }) => {
    const helpText = PARAM_HELP[paramKey as keyof typeof PARAM_HELP];
    const isEnabled = params.enabled?.[paramKey as keyof NonNullable<typeof params.enabled>] ?? false;

    const handleToggleEnabled = () => {
      setParams(prevParams => {
        const newValue = !(prevParams.enabled?.[paramKey as keyof typeof prevParams.enabled] ?? false);
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
      <div className="relative inline-flex items-center gap-2 flex-shrink-0" onClick={(e) => e.stopPropagation()}>
        {/* Enable/disable toggle using shadcn Switch */}
        {showToggle && (
          <Switch
            checked={isEnabled}
            onCheckedChange={handleToggleEnabled}
            disabled={isLoading}
            size="sm"
            aria-label={isEnabled ? `禁用 ${paramKey}` : `启用 ${paramKey}`}
            className={cn(
              isEnabled
                ? "data-[state=checked]:bg-green-600 dark:data-[state=checked]:bg-green-500"
                : ""
            )}
          />
        )}

        {/* Help button using shadcn Tooltip */}
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                className={cn(
                  "w-2.5 h-2.5 rounded-full text-muted-foreground text-[10px] font-medium",
                  "flex items-center justify-center",
                  "bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600",
                  "hover:from-blue-50 hover:to-blue-100 dark:hover:from-blue-900/40 dark:hover:to-blue-800/40",
                  "hover:text-blue-600 dark:hover:text-blue-400",
                  "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1",
                  "transition-all duration-200 cursor-help shadow-sm hover:shadow",
                  !isEnabled && "opacity-50"
                )}
                aria-label={`查看 ${paramKey} 的帮助说明`}
              >
                ?
              </button>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-xs">
              <div className="flex items-start gap-3 px-1 py-0.5">
                <Info className="w-4 h-4 text-blue-400 mt-0.5 flex-shrink-0" />
                <p className="text-sm leading-relaxed">
                  {helpText || '暂无说明'}
                </p>
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
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

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open && !isLoading) onClose(); }}>
      <DialogContent className="sm:max-w-[95vw] max-h-[90vh] p-0 overflow-hidden flex flex-col" onInteractOutside={(e) => { if (isLoading) e.preventDefault(); }}>
        {/* Header */}
        <DialogHeader className="p-4 border-b border-border flex-shrink-0">
          <DialogTitle className="text-lg font-semibold text-foreground">
            加载模型配置
          </DialogTitle>
        </DialogHeader>

        {/* Form content */}
        <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0">
          <div className="flex flex-col flex-1 min-h-0 p-3 overflow-hidden">
            <div className="flex flex-col lg:flex-row gap-4 flex-1 min-h-0">
            {/* Left column: basic config */}

            <div className="flex-1 space-y-4 overflow-y-scroll pr-2 min-h-0 dialog-scrollable" aria-label="基础配置区域">
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
                    onValueChange={(v) => setParams({ ...params, llamaCppPath: v })}
                    disabled={isLoading}
                    className="w-full"
                    options={
                      llamacppBackends.length > 0
                        ? llamacppBackends.map((backend: LlamacppBackend) => ({
                            value: backend.path,
                            label: `${backend.name}${backend.description ? ` (${backend.description})` : ''}${!backend.available ? ' - 不可用' : ''}`,
                            disabled: !backend.available,
                          }))
                        : [{ value: '__placeholder__', label: '未配置 llama.cpp 后端', disabled: true }]
                    }
                  />
                  {llamacppBackends.length === 0 && (
                    <p className="mt-1 text-xs text-yellow-600 dark:text-yellow-400">
                      请在服务器配置中添加 llama.cpp 路径
                    </p>
                  )}
                </div>

                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <label className="text-xs font-medium text-muted-foreground">推测解码</label>
                    <Switch
                      checked={params.enabled?.specDecoding ?? false}
                      onCheckedChange={(checked) =>
                        setParams(prev => ({
                          ...prev,
                          enabled: { ...prev.enabled, specDecoding: checked },
                          specDecoding: checked ? { ...prev.specDecoding, specType: prev.specDecoding?.specType || 'draft' } : { specType: 'none' },
                        }))
                      }
                    />
                  </div>
                  {params.enabled?.specDecoding && (
                    <Select
                      value={params.specDecoding?.specType || 'draft'}
                      onValueChange={(val) =>
                        setParams(prev => ({
                          ...prev,
                          specDecoding: { ...prev.specDecoding, specType: val },
                        }))
                      }
                    >
                      <SelectTrigger className="h-8 text-xs">
                        <SelectValue placeholder="选择推测解码类型" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="draft">Draft 模型</SelectItem>
                        <SelectItem value="eagle3">Eagle3 模型</SelectItem>
                        <SelectItem value="ngram-simple">NGram Simple</SelectItem>
                        <SelectItem value="ngram-map-k">NGram Map-K</SelectItem>
                        <SelectItem value="ngram-map-k4v">NGram Map-K4V</SelectItem>
                        <SelectItem value="ngram-mod">NGram Mod</SelectItem>
                        <SelectItem value="ngram-cache">NGram Cache</SelectItem>
                      </SelectContent>
                    </Select>
                  )}
                  {params.enabled?.specDecoding && (params.specDecoding?.specType === 'draft' || params.specDecoding?.specType === 'eagle3') && (
                    <Select
                      value={params.specDecoding?.specDraftModelId || '_none'}
                      onValueChange={(val) =>
                        setParams(prev => ({
                          ...prev,
                          specDecoding: { ...prev.specDecoding, specDraftModelId: val === '_none' ? '' : val },
                        }))
                      }
                    >
                      <SelectTrigger className="h-8 text-xs">
                        <SelectValue placeholder="选择 Draft 模型" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="_none">不使用</SelectItem>
                        {(() => {
                          const mainModel = allModels.find(m => m.id === modelId);
                          const mainArch = mainModel?.metadata?.architecture;
                          if (!mainArch) return null;
                          const compatible = allModels.filter(
                            m => m.id !== modelId
                              && m.metadata?.architecture === mainArch
                              && m.status === 'stopped'
                          );
                          if (compatible.length === 0) {
                            return <SelectItem value="_empty" disabled>没有找到同架构的候选模型</SelectItem>;
                          }
                          return compatible.map(m => {
                            const sizeGB = m.totalSize
                              ? (m.totalSize / 1024 / 1024 / 1024).toFixed(1)
                              : (m.size / 1024 / 1024 / 1024).toFixed(1);
                            return (
                              <SelectItem key={m.id} value={m.id}>
                                {m.displayName} ({sizeGB}GB)
                              </SelectItem>
                            );
                          });
                        })()}
                      </SelectContent>
                    </Select>
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
                              if (result.capabilities.tts) detectedList.push('语音合成');
                              if (result.capabilities.asr) detectedList.push('语音识别');
                              if (result.capabilities.imageGeneration) detectedList.push('图像生成');
                              if (result.capabilities.music) detectedList.push('音乐生成');

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

                      {/* Multimodal capabilities */}
                      <div className="text-xs text-muted-foreground uppercase tracking-wide mb-1 mt-2">多模态能力</div>
                      {[
                        {key: 'tts', label: '语音合成 (TTS)' },
                        {key: 'asr', label: '语音识别 (ASR)' },
                        {key: 'imageGeneration', label: '图像生成' },
                        {key: 'music', label: '音乐生成' },
                      ].map(({ key, label }) => (
                        <label key={key} className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded hover:bg-accent">
                          <input
                            type="checkbox"
                            checked={params.capabilities?.[key as keyof NonNullable<typeof params.capabilities>] || false}
                            onChange={(e) => handleCapabilityChange(key, e.target.checked)}
                            disabled={isLoading || params.capabilities?.thinking || params.capabilities?.tools}
                            className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                          />
                          <span>{label}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    选择模型支持的功能能力（互斥：思考/工具与嵌入/TTS/ASR/图像生成）
                  </p>
                </div>

                {/* Main GPU */}
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    主GPU
                  </label>
                  <SelectInput
                    value={params.mainGpu || 'default'}
                    onValueChange={(v) => setParams({ ...params, mainGpu: v })}
                    disabled={isLoading || gpus.length === 0}
                    className="w-full"
                    options={[
                      { value: 'default', label: '默认' },
                      ...gpus.map((gpu: SystemGPUInfo) => ({
                        value: gpu.id,
                        label: gpu.name,
                      })),
                    ]}
                  />
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
                      onValueChange={(v) => setParams({
                        ...params,
                        nodeId: v === 'auto' ? undefined : v
                      })}
                      disabled={isLoading}
                      className="w-full"
                      options={[
                        { value: 'auto', label: '🎯 自动调度（推荐）- 系统选择最佳节点' },
                      ]}
                      groups={onlineNodes.length > 0 ? [{
                        label: '指定节点',
                        options: onlineNodes.map((node: UnifiedNode) => ({
                          value: node.id,
                          label: `${node.name} (${node.address}:${node.port})${node.capabilities?.gpuCount ? ` · ${node.capabilities.gpuCount} GPU` : ''}${node.resources?.gpuInfo?.[0] ? ` · 显存 ${Math.round((node.resources.gpuInfo[0].totalMemory - node.resources.gpuInfo[0].usedMemory) / 1024 / 1024 / 1024)}GB 可用` : ''}`,
                        })),
                      }] : undefined}
                    />
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
            <div className="flex-1 space-y-4 overflow-y-scroll pr-2 min-h-0 dialog-scrollable" aria-label="高级参数区域">
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --ctx-size
                      {renderHelpButton('ctxSize')}
                    </div>
                    <NumberInput
                      value={params.ctxSize}
                      onChange={(v) => setParams({ ...params, ctxSize: v })}
                      disabled={getInputDisabled('ctxSize')}
                      min={0}
                      max={modelMaxCtxSize}
                      step={1}
                      placeholder="8192"
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      Flash Attention
                      {renderHelpButton('flashAttention')}
                    </div>
                    <SelectInput
                      value={params.flashAttention ? 'on' : 'off'}
                      onValueChange={(v) => setParams({ ...params, flashAttention: v === 'on' })}
                      disabled={getInputDisabled('flashAttention')}
                      options={[{ value: 'on', label: 'on' }, { value: 'off', label: 'off' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --no-mmap
                      {renderHelpButton('noMmap')}
                    </div>
                    <SelectInput
                      value={params.noMmap ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, noMmap: v === 'true' })}
                      disabled={getInputDisabled('noMmap')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      锁定物理内存
                      {renderHelpButton('lockMemory')}
                    </div>
                    <SelectInput
                      value={params.lockMemory ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, lockMemory: v === 'true' })}
                      disabled={getInputDisabled('lockMemory')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --embedding
                      {renderHelpButton('embedding')}
                    </div>
                    <SelectInput
                      value={params.embedding ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, embedding: v === 'true' })}
                      disabled={getInputDisabled('embedding')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --reranking
                      {renderHelpButton('reranking')}
                    </div>
                    <SelectInput
                      value={params.reranking ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, reranking: v === 'true' })}
                      disabled={getInputDisabled('reranking')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --gpu-layers
                      {renderHelpButton('gpuLayers')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --temp
                      {renderHelpButton('temperature')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      Top-P
                      {renderHelpButton('topP')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      Top-K
                      {renderHelpButton('topK')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      Min-P
                      {renderHelpButton('minP')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --repeat-penalty
                      {renderHelpButton('repeatPenalty')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --presence-penalty
                      {renderHelpButton('presencePenalty')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --frequency-penalty
                      {renderHelpButton('frequencyPenalty')}
                    </div>
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
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --batch-size
                      {renderHelpButton('batchSize')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --ubatch-size
                      {renderHelpButton('uBatchSize')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --parallel
                      {renderHelpButton('parallelSlots')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      线程数
                      {renderHelpButton('threads')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      空闲卸载(分)
                      {renderHelpButton('unloadAfterMinutes')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      最大并发数
                      {renderHelpButton('concurrencyLimit')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --cache-ram
                      {renderHelpButton('kvCacheSize')}
                    </div>
                    <NumberInput
                      value={params.kvCacheSize}
                      onChange={(v) => setParams({ ...params, kvCacheSize: v })}
                      disabled={getInputDisabled('kvCacheSize')}
                      min={0}
                      max={modelMaxCtxSize}
                      step={1}
                      placeholder="8192"
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --kv-unified
                      {renderHelpButton('kvCacheUnified')}
                    </div>
                    <SelectInput
                      value={params.kvCacheUnified ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, kvCacheUnified: v === 'true' })}
                      disabled={getInputDisabled('kvCacheUnified')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <label className="text-xs font-medium text-foreground mb-1">
                      -ctk
                    </label>
                    <SelectInput
                      value={params.kvCacheTypeK || 'f16'}
                      onValueChange={(v) => setParams({ ...params, kvCacheTypeK: v })}
                      disabled={getInputDisabled('kvCacheTypeK')}
                      options={[
                        { value: 'f32', label: 'f32' },
                        { value: 'f16', label: 'f16 (默认)' },
                        { value: 'bf16', label: 'bf16' },
                        { value: 'q8_0', label: 'q8_0' },
                        { value: 'q5_0', label: 'q5_0' },
                        { value: 'q5_1', label: 'q5_1' },
                        { value: 'q4_0', label: 'q4_0' },
                        { value: 'q4_1', label: 'q4_1' },
                        { value: 'iq4_nl', label: 'iq4_nl' },
                      ]}
                    />
                  </div>

                  <div>
                    <label className="text-xs font-medium text-foreground mb-1">
                      -ctv
                    </label>
                    <SelectInput
                      value={params.kvCacheTypeV || 'f16'}
                      onValueChange={(v) => setParams({ ...params, kvCacheTypeV: v })}
                      disabled={getInputDisabled('kvCacheTypeV')}
                      options={[
                        { value: 'f32', label: 'f32' },
                        { value: 'f16', label: 'f16 (默认)' },
                        { value: 'bf16', label: 'bf16' },
                        { value: 'q8_0', label: 'q8_0' },
                        { value: 'q5_0', label: 'q5_0' },
                        { value: 'q5_1', label: 'q5_1' },
                        { value: 'q4_0', label: 'q4_0' },
                        { value: 'q4_1', label: 'q4_1' },
                        { value: 'iq4_nl', label: 'iq4_nl' },
                      ]}
                    />
                  </div>
                </div>
              </div>

                {/* Speculative Decoding Parameters */}
                {params.enabled?.specDecoding && params.specDecoding?.specType && params.specDecoding.specType !== 'none' && (
                <div className="space-y-3">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                    推测解码参数
                  </h4>

                  {/* draft type parameters */}
                  {params.specDecoding.specType === 'draft' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-draft-n-max</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>每轮推测中 draft 模型生成的最大 token 数（默认 16）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specDraftNMax ?? 16} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specDraftNMax: val } }))} min={1} max={256} step={1} placeholder="16" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-draft-n-min</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>每轮推测中 draft 模型生成的最小 token 数（默认 1）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specDraftNMin ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specDraftNMin: val } }))} min={0} max={256} step={1} placeholder="1" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-draft-p-split</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>Draft 模型拆分概率阈值（0.0-1.0）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specDraftPSplit ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specDraftPSplit: val } }))} min={0} max={1} step={0.01} placeholder="0.00" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-draft-p-min</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>Draft 模型最小概率阈值（0.0-1.0）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specDraftPMin ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specDraftPMin: val } }))} min={0} max={1} step={0.01} placeholder="0.00" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-draft-ctx-size</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>Draft 模型上下文大小</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specDraftCtxSize ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specDraftCtxSize: val } }))} min={0} step={256} placeholder="0" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-draft-ngl</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>Draft 模型 GPU 层数（-1 表示全部）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specDraftNgl ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specDraftNgl: val } }))} min={-1} max={999} step={1} placeholder="0" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-draft-device</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>Draft 模型运行设备（如 cuda:0）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <Input value={params.specDecoding?.specDraftDevice ?? ''} onChange={(e) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specDraftDevice: e.target.value } }))} placeholder="自动" className="h-7 text-xs" />
                    </div>
                  </div>
                  )}

                  {/* ngram-simple parameters */}
                  {params.specDecoding.specType === 'ngram-simple' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-simple-size-n</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>NGram 前缀长度（默认 1）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramSimpleSizeN ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramSimpleSizeN: val } }))} min={1} step={1} placeholder="1" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-simple-size-m</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>NGram 预测长度（默认 2）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramSimpleSizeM ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramSimpleSizeM: val } }))} min={1} step={1} placeholder="2" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-simple-min-hits</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>最小命中次数（默认 1）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramSimpleMinHits ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramSimpleMinHits: val } }))} min={1} step={1} placeholder="1" className="w-full h-7 text-xs" />
                    </div>
                  </div>
                  )}

                  {/* ngram-mod parameters */}
                  {params.specDecoding.specType === 'ngram-mod' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-mod-n-min</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>最小 NGram 长度（默认 1）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramModNMin ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramModNMin: val } }))} min={1} step={1} placeholder="1" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-mod-n-max</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>最大 NGram 长度（默认 64）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramModNMax ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramModNMax: val } }))} min={1} step={1} placeholder="64" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-mod-n-match</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>匹配长度（默认 2）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramModNMatch ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramModNMatch: val } }))} min={1} step={1} placeholder="2" className="w-full h-7 text-xs" />
                    </div>
                  </div>
                  )}

                  {/* ngram-map-k parameters */}
                  {params.specDecoding.specType === 'ngram-map-k' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-map-k-size-n</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>NGram 前缀长度（默认 1）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramMapKSizeN ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramMapKSizeN: val } }))} min={1} step={1} placeholder="1" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-map-k-size-m</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>NGram 预测长度（默认 2）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramMapKSizeM ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramMapKSizeM: val } }))} min={1} step={1} placeholder="2" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-map-k-min-hits</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>最小命中次数（默认 1）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramMapKMinHits ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramMapKMinHits: val } }))} min={1} step={1} placeholder="1" className="w-full h-7 text-xs" />
                    </div>
                  </div>
                  )}

                  {/* ngram-map-k4v parameters */}
                  {params.specDecoding.specType === 'ngram-map-k4v' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-map-k4v-size-n</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>NGram 前缀长度（默认 1）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramMapK4VSizeN ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramMapK4VSizeN: val } }))} min={1} step={1} placeholder="1" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-map-k4v-size-m</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>NGram 预测长度（默认 2）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramMapK4VSizeM ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramMapK4VSizeM: val } }))} min={1} step={1} placeholder="2" className="w-full h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--spec-ngram-map-k4v-min-hits</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>最小命中次数（默认 1）</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <NumberInput value={params.specDecoding?.specNgramMapK4VMinHits ?? 0} onChange={(val) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, specNgramMapK4VMinHits: val } }))} min={1} step={1} placeholder="1" className="w-full h-7 text-xs" />
                    </div>
                  </div>
                  )}

                  {/* ngram-cache parameters */}
                  {params.specDecoding.specType === 'ngram-cache' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--lookup-cache-static</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>静态查找缓存文件路径</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <Input value={params.specDecoding?.lookupCacheStatic ?? ''} onChange={(e) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, lookupCacheStatic: e.target.value } }))} placeholder="路径" className="h-7 text-xs" />
                    </div>
                    <div>
                      <div className="flex items-center gap-1.5 text-xs font-medium text-foreground mb-1">
                        <span>--lookup-cache-dynamic</span>
                        <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-3 w-3 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent>动态查找缓存文件路径</TooltipContent></Tooltip></TooltipProvider>
                      </div>
                      <Input value={params.specDecoding?.lookupCacheDynamic ?? ''} onChange={(e) => setParams(prev => ({ ...prev, specDecoding: { ...prev.specDecoding, lookupCacheDynamic: e.target.value } }))} placeholder="路径" className="h-7 text-xs" />
                    </div>
                  </div>
                  )}
                </div>
                )}

                {/* Other params */}
                <div className="space-y-3">
                  <h4 className="text-xs font-semibold text-muted-foreground uppercase">
                    其他参数
                  </h4>
                  <div className="grid grid-cols-2 gap-3">
                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --seed
                      {renderHelpButton('seed')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --n-predict
                      {renderHelpButton('nPredict')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --direct-io
                      {renderHelpButton('directIo')}
                    </div>
                    <SelectInput
                      value={params.directIo || 'default'}
                      onValueChange={(v) => setParams({ ...params, directIo: v })}
                      disabled={getInputDisabled('directIo')}
                      options={[{ value: 'default', label: 'default' }, { value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --no-webui
                      {renderHelpButton('noWebUI')}
                    </div>
                    <SelectInput
                      value={params.noWebUI ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, noWebUI: v === 'true' })}
                      disabled={getInputDisabled('noWebUI')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --no-jinja
                      {renderHelpButton('disableJinja')}
                    </div>
                    <SelectInput
                      value={params.disableJinja ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, disableJinja: v === 'true' })}
                      disabled={getInputDisabled('disableJinja')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --chat-template
                      {renderHelpButton('chatTemplate')}
                    </div>
                    <SelectInput
                      value={params.chatTemplate || '__auto__'}
                      onValueChange={(v) => setParams({ ...params, chatTemplate: v === '__auto__' ? '' : v })}
                      disabled={getInputDisabled('chatTemplate')}
                      options={[
                        { value: '__auto__', label: '(自动 - 使用模型元数据)' },
                        { value: 'bailing', label: 'bailing' },
                        { value: 'bailing-think', label: 'bailing-think' },
                        { value: 'bailing2', label: 'bailing2' },
                        { value: 'chatglm3', label: 'chatglm3' },
                        { value: 'chatglm4', label: 'chatglm4' },
                        { value: 'deepseek', label: 'deepseek' },
                        { value: 'deepseek2', label: 'deepseek2' },
                        { value: 'deepseek3', label: 'deepseek3' },
                        { value: 'exaone-moe', label: 'exaone-moe' },
                        { value: 'exaone3', label: 'exaone3' },
                        { value: 'exaone4', label: 'exaone4' },
                        { value: 'hunyuan-dense', label: 'hunyuan-dense' },
                        { value: 'hunyuan-moe', label: 'hunyuan-moe' },
                        { value: 'llama2', label: 'llama2' },
                        { value: 'llama2-sys', label: 'llama2-sys' },
                        { value: 'llama2-sys-bos', label: 'llama2-sys-bos' },
                        { value: 'llama2-sys-strip', label: 'llama2-sys-strip' },
                        { value: 'llama3', label: 'llama3' },
                        { value: 'llama4', label: 'llama4' },
                        { value: 'mistral-v1', label: 'mistral-v1' },
                        { value: 'mistral-v3', label: 'mistral-v3' },
                        { value: 'mistral-v3-tekken', label: 'mistral-v3-tekken' },
                        { value: 'mistral-v7', label: 'mistral-v7' },
                        { value: 'mistral-v7-tekken', label: 'mistral-v7-tekken' },
                        { value: 'phi3', label: 'phi3' },
                        { value: 'phi4', label: 'phi4' },
                        { value: 'vicuna', label: 'vicuna' },
                        { value: 'vicuna-orca', label: 'vicuna-orca' },
                        { value: 'chatml', label: 'chatml' },
                        { value: 'command-r', label: 'command-r' },
                        { value: 'falcon3', label: 'falcon3' },
                        { value: 'gemma', label: 'gemma' },
                        { value: 'gigachat', label: 'gigachat' },
                        { value: 'glmedge', label: 'glmedge' },
                        { value: 'gpt-oss', label: 'gpt-oss' },
                        { value: 'granite', label: 'granite' },
                        { value: 'grok-2', label: 'grok-2' },
                        { value: 'kimi-k2', label: 'kimi-k2' },
                        { value: 'megrez', label: 'megrez' },
                        { value: 'minicpm', label: 'minicpm' },
                        { value: 'monarch', label: 'monarch' },
                        { value: 'openchat', label: 'openchat' },
                        { value: 'orion', label: 'orion' },
                        { value: 'pangu-embedded', label: 'pangu-embedded' },
                        { value: 'rwkv-world', label: 'rwkv-world' },
                        { value: 'seed_oss', label: 'seed_oss' },
                        { value: 'smolvlm', label: 'smolvlm' },
                        { value: 'solar-open', label: 'solar-open' },
                        { value: 'yandex', label: 'yandex' },
                        { value: 'zephyr', label: 'zephyr' },
                      ]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --context-shift
                      {renderHelpButton('contextShift')}
                    </div>
                    <SelectInput
                      value={params.contextShift ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, contextShift: v === 'true' })}
                      disabled={getInputDisabled('contextShift')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --reasoning
                      {renderHelpButton('reasoning')}
                    </div>
                    <SelectInput
                      value={params.reasoning || 'auto'}
                      onValueChange={(v) => setParams({ ...params, reasoning: v })}
                      disabled={getInputDisabled('reasoning')}
                      options={[{ value: 'auto', label: 'auto' }, { value: 'on', label: 'on' }, { value: 'off', label: 'off' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --reasoning-format
                      {renderHelpButton('reasoningFormat')}
                    </div>
                    <SelectInput
                      value={params.reasoningFormat || 'auto'}
                      onValueChange={(v) => setParams({ ...params, reasoningFormat: v })}
                      disabled={getInputDisabled('reasoningFormat')}
                      options={[{ value: 'auto', label: 'auto' }, { value: 'deepseek', label: 'deepseek' }]}
                    />
                  </div>

                  <div>
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --reasoning-budget
                      {renderHelpButton('reasoningBudget')}
                    </div>
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
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      --no-mmproj-offload
                      {renderHelpButton('mmprojOffload')}
                    </div>
                    <SelectInput
                      value={params.mmprojOffload ? 'true' : 'false'}
                      onValueChange={(v) => setParams({ ...params, mmprojOffload: v === 'true' })}
                      disabled={getInputDisabled('mmprojOffload')}
                      options={[{ value: 'true', label: 'true' }, { value: 'false', label: 'false' }]}
                    />
                  </div>

                  <div className="col-span-2">
                    <div className="flex items-center text-xs font-medium text-foreground mb-1 whitespace-nowrap">
                      其他参数
                      {renderHelpButton('extraArgs')}
                    </div>
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
          <div className="flex justify-between items-center gap-3 px-4 py-3 border-t border-border bg-card flex-shrink-0">
            {/* Left: Config selection + VRAM estimate */}
            <div className="flex items-center gap-2">
              <select
                value={selectedConfigName}
                onChange={(e) => {
                  const val = e.target.value;
                  if (val) {
                    handleLoadNamedConfig(val);
                  }
                }}
                className={cn(
                  "h-9 px-3 text-sm border-2 border-border rounded-md",
                  "bg-input text-foreground",
                  "focus:outline-none focus:ring-2 focus:ring-blue-500"
                )}
              >
                <option value="">选择配置...</option>
                {savedConfigs.map(c => (
                  <option key={c.name} value={c.name}>{c.name}</option>
                ))}
              </select>
              {selectedConfigName && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDeleteNamedConfig(selectedConfigName)}
                  className="text-muted-foreground hover:text-destructive h-8 w-8 p-0"
                  title="删除此配置"
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              )}
              <div className="w-px h-6 bg-border" />
              <Button
                type="button"
                variant="outline"
                onClick={async () => {
                  setEstimateResult('计算中...');
                  try {
                    const result = await estimateVRAM.mutateAsync({
                      modelId,
                      ctxSize: params.ctxSize,
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

            {/* Right: Action buttons */}
            <div className="flex items-center gap-3">
              <Button type="button" variant="outline" onClick={onClose} disabled={isLoading}>
                取消
              </Button>

              {/* Save named config */}
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={configName}
                  onChange={(e) => setConfigName(e.target.value)}
                  placeholder="配置名称"
                  className={cn(
                    "h-9 px-3 text-sm w-32 border-2 border-border rounded-md",
                    "bg-input text-foreground",
                    "focus:outline-none focus:ring-2 focus:ring-blue-500"
                  )}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      handleSaveConfig();
                    }
                  }}
                />
                <Button
                  type="button"
                  variant="secondary"
                  onClick={handleSaveConfig}
                  disabled={isLoading}
                  className={cn(
                    saveStatus === 'saved' && 'bg-green-600 text-white hover:bg-green-700',
                    saveStatus === 'error' && 'bg-red-600 text-white hover:bg-red-700'
                  )}
                >
                  {saveStatus === 'saved' ? (
                    <>✓ 已保存</>
                  ) : saveStatus === 'error' ? (
                    <>✗ 保存失败</>
                  ) : (
                    <>
                      <Save className="w-4 h-4" />
                      保存配置
                    </>
                  )}
                </Button>
              </div>

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
          </div>
      </form>
      </DialogContent>
    </Dialog>
  );
}
