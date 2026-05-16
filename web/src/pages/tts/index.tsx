import { useState, useRef, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Volume2, Loader2, Play, Pause, Download, ChevronDown, Settings2, Save, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { useLoadedModels, BACKEND_LABELS } from '@/features/creative/hooks';
import { ModelSelect } from '@/features/creative/ModelSelect';
import {
  useTTS,
  useVoices,
  useTTSConfig,
  getTTSModelFeatures,
  type TTSRequest,
  type TTSModelFeatures,
  type TTSConfig,
} from '@/features/tts/hooks';
import { StreamAudioPlayer, type StreamState, type TTSStreamMetrics } from '@/features/tts/lib/StreamAudioPlayer';
import { StreamPlaybackPanel } from '@/features/tts/components/StreamPlaybackPanel';
import { RefAudioInput } from '@/features/tts/components/RefAudioInput';
import { toast } from '@/hooks/useToast';
import { cn } from '@/lib/utils';

const FALLBACK_VOICES = ['alloy', 'echo', 'fable', 'onyx', 'nova', 'shimmer'];

const AUDIO_FORMATS = [
  { value: 'mp3', label: 'MP3' },
  { value: 'wav', label: 'WAV' },
  { value: 'opus', label: 'Opus' },
  { value: 'flac', label: 'FLAC' },
  { value: 'pcm', label: 'PCM (流式)' },
];

export function TTSPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data: allModels = [] } = useLoadedModels();
  const ttsModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.tts),
    [allModels]
  );

  const tts = useTTS();
  const audioRef = useRef<HTMLAudioElement>(null);
  const playerRef = useRef<StreamAudioPlayer | null>(null);

  const [model, setModel] = useState('');
  const [input, setInput] = useState('');
  const [voice, setVoice] = useState('');
  const [speed, setSpeed] = useState(1);
  const [responseFormat, setResponseFormat] = useState('mp3');
  const [stream, setStream] = useState(false);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);

  // VoxCPM2 扩展字段
  const [instructions, setInstructions] = useState('');
  const [refAudio, setRefAudio] = useState('');
  const [refText, setRefText] = useState('');
  const [promptAudio, setPromptAudio] = useState('');
  const [promptText, setPromptText] = useState('');
  const [ultimateCloning, setUltimateCloning] = useState(false);
  const [cloningOpen, setCloningOpen] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [seed, setSeed] = useState('');
  const [maxNewTokens, setMaxNewTokens] = useState('');

  // 流式播放状态
  const [streamState, setStreamState] = useState<StreamState>('idle');
  const [streamMetrics, setStreamMetrics] = useState<TTSStreamMetrics>({
    ttfp: null, rtf: null, audioDuration: 0, speedMultiplier: 0, bytesReceived: 0,
  });

  // 配置管理
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [configName, setConfigName] = useState('');
  const [selectedConfigName, setSelectedConfigName] = useState('');
  const [savedConfigs, setSavedConfigs] = useState<{ name: string; config: TTSConfig }[]>([]);

  const selectedModel = ttsModels.find((m) => (m.alias || m.name) === model);
  const modelIdForConfig = selectedModel?.id || '';
  const { ttsConfig, saveConfig, deleteConfig } = useTTSConfig(modelIdForConfig);

  const backendLabel = selectedModel?.backendType
    ? BACKEND_LABELS[selectedModel.backendType] || selectedModel.backendType
    : '';

  const features: TTSModelFeatures = selectedModel
    ? getTTSModelFeatures(selectedModel)
    : {
        supportsVoiceSelection: true,
        supportsInstructions: false,
        supportsRefAudio: false,
        supportsUltimateCloning: false,
        supportsStreamPcm: false,
        defaultSampleRate: 24000,
        defaultFormat: 'mp3',
      };

  // localStorage 命名预设
  const CONFIGS_STORAGE_KEY = model ? `shepherd:tts-configs:${model}` : '';

  const getSavedConfigs = useCallback((): { name: string; config: TTSConfig }[] => {
    if (!CONFIGS_STORAGE_KEY) return [];
    try {
      const data = localStorage.getItem(CONFIGS_STORAGE_KEY);
      return data ? JSON.parse(data) : [];
    } catch {
      return [];
    }
  }, [CONFIGS_STORAGE_KEY]);

  // 从服务端恢复配置
  useEffect(() => {
    if (ttsConfig) {
      if (ttsConfig.voice !== undefined) setVoice(ttsConfig.voice);
      if (ttsConfig.speed !== undefined) setSpeed(ttsConfig.speed);
      if (ttsConfig.responseFormat !== undefined) setResponseFormat(ttsConfig.responseFormat);
      if (ttsConfig.stream !== undefined) setStream(ttsConfig.stream);
      if (ttsConfig.instructions !== undefined) setInstructions(ttsConfig.instructions);
      if (ttsConfig.refAudio !== undefined) setRefAudio(ttsConfig.refAudio);
      if (ttsConfig.refText !== undefined) setRefText(ttsConfig.refText);
      if (ttsConfig.promptAudio !== undefined) setPromptAudio(ttsConfig.promptAudio);
      if (ttsConfig.promptText !== undefined) setPromptText(ttsConfig.promptText);
      if (ttsConfig.ultimateCloning !== undefined) setUltimateCloning(ttsConfig.ultimateCloning);
      if (ttsConfig.seed !== undefined) setSeed(ttsConfig.seed);
      if (ttsConfig.maxNewTokens !== undefined) setMaxNewTokens(ttsConfig.maxNewTokens);
    }
  }, [ttsConfig]);

  // 模型切换时刷新 localStorage 预设
  useEffect(() => {
    setSavedConfigs(getSavedConfigs());
    setConfigName('');
    setSelectedConfigName('');
    setSaveStatus('idle');
  }, [model, getSavedConfigs]);

  // 模型切换时自动调整格式
  useEffect(() => {
    if (selectedModel) {
      const fmt = getTTSModelFeatures(selectedModel).defaultFormat;
      if (fmt === 'pcm') {
        setResponseFormat('pcm');
        setStream(true);
      }
    }
  }, [selectedModel]);

  useEffect(() => {
    return () => {
      playerRef.current?.destroy();
    };
  }, []);

  const { data: voices = [] } = useVoices(model);

  // 保存配置到服务端
  const handleSaveToServer = useCallback(() => {
    if (!modelIdForConfig) return;
    const config: Record<string, unknown> = {
      voice: voice || undefined,
      speed: speed !== 1 ? speed : undefined,
      responseFormat,
      stream,
      instructions: instructions || undefined,
      refAudio: refAudio || undefined,
      refText: refText || undefined,
      promptAudio: promptAudio || undefined,
      promptText: promptText || undefined,
      ultimateCloning: ultimateCloning || undefined,
      seed: seed || undefined,
      maxNewTokens: maxNewTokens || undefined,
    };
    saveConfig.mutate({ modelId: modelIdForConfig, config: config as any });
  }, [modelIdForConfig, voice, speed, responseFormat, stream, instructions,
      refAudio, refText, promptAudio, promptText, ultimateCloning, seed, maxNewTokens, saveConfig]);

  // 从 localStorage 加载命名配置
  const handleLoadNamedConfig = useCallback((name: string) => {
    const configs = getSavedConfigs();
    const found = configs.find(c => c.name === name);
    if (found) {
      const cfg = found.config;
      if (cfg.voice !== undefined) setVoice(cfg.voice);
      if (cfg.speed !== undefined) setSpeed(cfg.speed);
      if (cfg.responseFormat !== undefined) setResponseFormat(cfg.responseFormat);
      if (cfg.stream !== undefined) setStream(cfg.stream);
      if (cfg.instructions !== undefined) setInstructions(cfg.instructions);
      if (cfg.refAudio !== undefined) setRefAudio(cfg.refAudio);
      if (cfg.refText !== undefined) setRefText(cfg.refText);
      if (cfg.promptAudio !== undefined) setPromptAudio(cfg.promptAudio);
      if (cfg.promptText !== undefined) setPromptText(cfg.promptText);
      if (cfg.ultimateCloning !== undefined) setUltimateCloning(cfg.ultimateCloning);
      if (cfg.seed !== undefined) setSeed(cfg.seed);
      if (cfg.maxNewTokens !== undefined) setMaxNewTokens(cfg.maxNewTokens);
      setSelectedConfigName(name);
      setConfigName(name);
    }
  }, [getSavedConfigs]);

  // 保存命名配置到 localStorage
  const handleSaveNamedConfig = useCallback(() => {
    if (!CONFIGS_STORAGE_KEY) return;
    const name = configName.trim();
    if (!name) {
      toast.error(t('tts.selectModelWarning', '请输入配置名称'));
      return;
    }
    const config: TTSConfig = {
      voice: voice || undefined,
      speed: speed !== 1 ? speed : undefined,
      responseFormat,
      stream,
      instructions: instructions || undefined,
      refAudio: refAudio || undefined,
      refText: refText || undefined,
      promptAudio: promptAudio || undefined,
      promptText: promptText || undefined,
      ultimateCloning: ultimateCloning || undefined,
      seed: seed || undefined,
      maxNewTokens: maxNewTokens || undefined,
    };
    try {
      const configs = getSavedConfigs();
      const idx = configs.findIndex(c => c.name === name);
      if (idx >= 0) {
        configs[idx] = { name, config };
      } else {
        configs.push({ name, config });
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
  }, [CONFIGS_STORAGE_KEY, configName, voice, speed, responseFormat, stream,
      instructions, refAudio, refText, promptAudio, promptText, ultimateCloning,
      seed, maxNewTokens, getSavedConfigs, t]);

  // 删除命名配置
  const handleDeleteNamedConfig = useCallback((name: string) => {
    if (!CONFIGS_STORAGE_KEY) return;
    const configs = getSavedConfigs().filter(c => c.name !== name);
    localStorage.setItem(CONFIGS_STORAGE_KEY, JSON.stringify(configs));
    setSavedConfigs([...configs]);
    if (selectedConfigName === name) {
      setSelectedConfigName('');
    }
  }, [CONFIGS_STORAGE_KEY, getSavedConfigs, selectedConfigName]);

  const handleGenerate = useCallback(async () => {
    if (!model) {
      toast.warning(t('tts.selectModelWarning', '请选择模型'));
      return;
    }
    if (!input.trim() && !ultimateCloning) {
      toast.warning(t('tts.inputRequired', '请输入文本'));
      return;
    }
    if (ultimateCloning && !promptAudio) {
      toast.warning(t('tts.inputRequired', '缺少音频'));
      return;
    }

    const useStreamPcm = stream && (responseFormat === 'pcm' || features.supportsStreamPcm);

    const payload: TTSRequest = {
      model,
      input: ultimateCloning ? (promptText || '') : input.trim(),
      response_format: useStreamPcm ? 'pcm' : (responseFormat === 'pcm' ? 'wav' : responseFormat),
      speed: speed !== 1 ? speed : undefined,
      stream: useStreamPcm ? true : undefined,
    };

    if (!useStreamPcm && features.supportsVoiceSelection) {
      payload.voice = voice || undefined;
    }

    if (features.supportsInstructions && instructions.trim()) {
      payload.instructions = instructions.trim();
    }

    if (ultimateCloning) {
      payload.prompt_audio = promptAudio;
      payload.prompt_text = promptText || undefined;
    } else if (refAudio) {
      payload.ref_audio = refAudio;
      payload.ref_text = refText || undefined;
    }

    if (seed) payload.seed = parseInt(seed, 10) || undefined;
    if (maxNewTokens) payload.max_new_tokens = parseInt(maxNewTokens, 10) || undefined;

    if (useStreamPcm) {
      try {
        if (!playerRef.current) {
          playerRef.current = new StreamAudioPlayer(features.defaultSampleRate);
          await playerRef.current.init();
        }

        playerRef.current.onStateChange = setStreamState;
        playerRef.current.onMetricsUpdate = setStreamMetrics;
        playerRef.current.onError = (err) => {
          toast.error(t('tts.generateFailed', '流式播放错误'), err.message);
        };

        await playerRef.current.startStream('/v1/audio/speech', payload);
      } catch (err) {
        toast.error(t('tts.generateFailed', '初始化失败'), (err as Error).message);
      }
    } else {
      if (audioUrl) URL.revokeObjectURL(audioUrl);

      tts.mutate(payload, {
        onSuccess: ({ blob, contentType }) => {
          const typedBlob = new Blob([blob], { type: contentType });
          const url = URL.createObjectURL(typedBlob);
          setAudioUrl(url);
          toast.success(t('tts.generateSuccess', '语音合成完成'));
          if (modelIdForConfig) {
            const config: Record<string, unknown> = {
              voice: voice || undefined,
              speed: speed !== 1 ? speed : undefined,
              responseFormat,
              stream,
              instructions: instructions || undefined,
              refAudio: refAudio || undefined,
              refText: refText || undefined,
              promptAudio: promptAudio || undefined,
              promptText: promptText || undefined,
              ultimateCloning: ultimateCloning || undefined,
              seed: seed || undefined,
              maxNewTokens: maxNewTokens || undefined,
            };
            saveConfig.mutate({ modelId: modelIdForConfig, config: config as any });
          }
        },
        onError: (error) => {
          toast.error(t('tts.generateFailed', '语音合成失败'), error.message);
        },
      });
    }
  }, [model, input, voice, speed, responseFormat, stream, features, instructions,
      refAudio, refText, promptAudio, promptText, ultimateCloning, seed, maxNewTokens,
      audioUrl, tts, modelIdForConfig, saveConfig, t]);

  const handlePlayPause = () => {
    if (!audioRef.current) return;
    if (isPlaying) {
      audioRef.current.pause();
    } else {
      audioRef.current.play();
    }
    setIsPlaying(!isPlaying);
  };

  const handleDownload = () => {
    if (!audioUrl) return;
    const ext = responseFormat === 'pcm' ? 'wav' : (responseFormat || 'mp3');
    const a = document.createElement('a');
    a.href = audioUrl;
    a.download = `tts_${Date.now()}.${ext}`;
    a.click();
  };

  const handleStopStream = () => {
    playerRef.current?.stop();
  };

  const isStreamActive = streamState === 'streaming' || streamState === 'playing';
  const isGenerating = tts.isPending || isStreamActive;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          {t('tts.title', '语音合成 (TTS)')}
        </h1>
        <p className="text-muted-foreground">
          {t('tts.description', '将文本转换为自然语音，支持流式播放和声音克隆')}
        </p>
      </div>

      <div className="space-y-4">
        {ttsModels.length > 0 ? (
          <div>
            <ModelSelect
              models={ttsModels}
              value={model}
              onValueChange={(v) => {
                setModel(v);
                setVoice('');
                setUltimateCloning(false);
                setRefAudio('');
                setPromptAudio('');
              }}
              placeholder={t('tts.selectModel', '选择 TTS 模型')}
              label={t('tts.modelLabel', 'TTS 模型')}
              showBackend
            />
            {backendLabel && (
              <p className="text-xs text-muted-foreground mt-1">
                {t('tts.backend', '后端')}: {backendLabel}
              </p>
            )}
          </div>
        ) : (
          <div className="flex flex-col items-center rounded-lg border border-dashed p-8 text-center">
            <Volume2 className="h-8 w-8 text-muted-foreground mb-2" />
            <p className="text-sm font-medium">{t('tts.noModels', '没有已加载的 TTS 模型')}</p>
            <p className="text-xs text-muted-foreground mt-1">{t('tts.noModelsHint', '请先加载一个支持语音合成的模型')}</p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3"
              onClick={() => navigate('/models')}
            >
              {t('tts.goToModels', '前往模型管理')}
            </Button>
          </div>
        )}

        {!ultimateCloning && (
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.inputLabel', '输入文本')}
            </label>
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={t('tts.inputPlaceholder', '输入要转换为语音的文本...')}
              className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
              rows={4}
            />
          </div>
        )}

        {features.supportsInstructions && (
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.instructions', '风格指令')}
            </label>
            <Input
              value={instructions}
              onChange={(e) => setInstructions(e.target.value)}
              placeholder={t('tts.instructionsPlaceholder', '输入风格指令，如：用温柔的语气朗读...')}
              className="bg-background"
            />
          </div>
        )}

        <div className="grid grid-cols-2 gap-4">
          {features.supportsVoiceSelection && (
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.voiceLabel', '语音 (Voice)')}
              </label>
              <Select value={voice} onValueChange={setVoice}>
                <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                  <SelectValue placeholder={voices.length > 0 ? t('tts.selectVoice', '选择语音') : t('tts.enterVoice', '输入语音名称')} />
                </SelectTrigger>
                <SelectContent>
                  {voices.length > 0
                    ? voices.map((v) => (
                        <SelectItem key={v.id} value={v.id}>
                          {v.name || v.id}
                        </SelectItem>
                      ))
                    : FALLBACK_VOICES.map((v) => (
                        <SelectItem key={v} value={v}>{v}</SelectItem>
                      ))
                  }
                </SelectContent>
              </Select>
            </div>
          )}
          <div className={features.supportsVoiceSelection ? '' : 'col-span-2'}>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.formatLabel', '输出格式')}
            </label>
            <Select value={responseFormat} onValueChange={setResponseFormat}>
              <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {AUDIO_FORMATS.map((f) => (
                  <SelectItem key={f.value} value={f.value}>{f.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.speedLabel', '语速')}: {speed}x
            </label>
            <Slider
              value={[speed]}
              onValueChange={([val]) => setSpeed(val)}
              min={0.25}
              max={4}
              step={0.25}
              className="w-full mt-2"
            />
          </div>
          <div className="flex items-center gap-3 pt-6">
            <Switch checked={stream} onCheckedChange={setStream} />
            <label className="text-sm font-medium">
              {t('tts.streaming', '流式生成')}
            </label>
          </div>
        </div>

        {features.supportsRefAudio && (
          <Collapsible open={cloningOpen} onOpenChange={setCloningOpen}>
            <CollapsibleTrigger asChild>
              <Button variant="ghost" className="w-full justify-between p-0 h-auto text-sm font-medium hover:bg-transparent">
                {t('tts.voiceCloning', '声音克隆')}
                <ChevronDown className={`w-4 h-4 transition-transform ${cloningOpen ? 'rotate-180' : ''}`} />
              </Button>
            </CollapsibleTrigger>
            <CollapsibleContent className="space-y-4 pt-2">
              {features.supportsUltimateCloning && (
                <div className="flex items-center gap-3">
                  <Switch checked={ultimateCloning} onCheckedChange={setUltimateCloning} />
                  <div>
                    <Label className="text-sm font-medium">
                      {t('tts.ultimateCloning', '终极克隆模式')}
                    </Label>
                    <p className="text-xs text-muted-foreground">
                      {t('tts.ultimateCloningDesc', '使用 prompt_audio + prompt_text 进行精确声音克隆')}
                    </p>
                  </div>
                </div>
              )}

              {ultimateCloning ? (
                <>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('tts.refAudio', '克隆音频')}
                    </label>
                    <RefAudioInput value={promptAudio} onChange={setPromptAudio} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('tts.refText', '音频转录文本')}
                    </label>
                    <Textarea
                      value={promptText}
                      onChange={(e) => setPromptText(e.target.value)}
                      placeholder={t('tts.refTextPlaceholder', '输入音频的转录文本')}
                      rows={2}
                      className="bg-background"
                    />
                  </div>
                </>
              ) : (
                <>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('tts.refAudio', '参考音频')}
                    </label>
                    <RefAudioInput value={refAudio} onChange={setRefAudio} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('tts.refText', '参考音频转录文本')}
                    </label>
                    <Textarea
                      value={refText}
                      onChange={(e) => setRefText(e.target.value)}
                      placeholder={t('tts.refTextPlaceholder', '输入参考音频的转录文本（可选）')}
                      rows={2}
                      className="bg-background"
                    />
                  </div>
                </>
              )}
            </CollapsibleContent>
          </Collapsible>
        )}

        <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
          <CollapsibleTrigger asChild>
            <Button variant="ghost" className="w-full justify-between p-0 h-auto text-sm font-medium hover:bg-transparent">
              <span className="flex items-center gap-2">
                <Settings2 className="w-4 h-4" />
                {t('tts.advanced', '高级设置')}
              </span>
              <ChevronDown className={`w-4 h-4 transition-transform ${advancedOpen ? 'rotate-180' : ''}`} />
            </Button>
          </CollapsibleTrigger>
          <CollapsibleContent className="space-y-4 pt-2">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-1.5">
                  {t('tts.seed', '随机种子')}
                </label>
                <Input
                  value={seed}
                  onChange={(e) => setSeed(e.target.value)}
                  placeholder={t('tts.seedPlaceholder', '留空为随机')}
                  type="number"
                  className="bg-background"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1.5">
                  {t('tts.maxNewTokens', '最大 Token 数')}
                </label>
                <Input
                  value={maxNewTokens}
                  onChange={(e) => setMaxNewTokens(e.target.value)}
                  placeholder="默认值"
                  type="number"
                  className="bg-background"
                />
              </div>
            </div>
          </CollapsibleContent>
        </Collapsible>

        <Button
          onClick={handleGenerate}
          disabled={isGenerating || !model || (!input.trim() && !ultimateCloning)}
          className="w-full"
        >
          {isGenerating ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              {isStreamActive ? '流式生成中...' : t('tts.generating', '生成中...')}
            </>
          ) : (
            <>
              <Volume2 className="w-4 h-4 mr-2" />
              {t('tts.generate', '生成语音')}
            </>
          )}
        </Button>

        {model && (
          <div className="border rounded-lg p-3 space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-medium text-muted-foreground">配置管理</h4>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleSaveToServer}
                  disabled={!modelIdForConfig}
                  className="text-xs h-7"
                >
                  <Save className="w-3.5 h-3.5 mr-1" />
                  保存到服务端
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    if (modelIdForConfig) {
                      deleteConfig.mutate(modelIdForConfig, {
                        onSuccess: () => toast.success('配置已删除'),
                        onError: (err: Error) => toast.error('删除失败', err.message),
                      });
                    }
                  }}
                  disabled={!modelIdForConfig || !ttsConfig}
                  className="text-xs h-7"
                >
                  <Trash2 className="w-3.5 h-3.5 mr-1" />
                  删除服务端配置
                </Button>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <select
                value={selectedConfigName}
                onChange={(e) => {
                  if (e.target.value) handleLoadNamedConfig(e.target.value);
                }}
                className={cn(
                  "h-8 px-2 text-sm border-2 border-border rounded-md flex-1",
                  "bg-input text-foreground",
                  "focus:outline-none focus:ring-2 focus:ring-blue-500"
                )}
              >
                <option value="">选择预设...</option>
                {savedConfigs.map(c => (
                  <option key={c.name} value={c.name}>{c.name}</option>
                ))}
              </select>
              {selectedConfigName && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDeleteNamedConfig(selectedConfigName)}
                  className="text-muted-foreground hover:text-destructive h-8 w-8 p-0"
                  title="删除此预设"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </Button>
              )}
              <div className="w-px h-6 bg-border" />
              <input
                type="text"
                value={configName}
                onChange={(e) => setConfigName(e.target.value)}
                placeholder="预设名称"
                className={cn(
                  "h-8 px-2 text-sm w-28 border-2 border-border rounded-md",
                  "bg-input text-foreground",
                  "focus:outline-none focus:ring-2 focus:ring-blue-500"
                )}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    handleSaveNamedConfig();
                  }
                }}
              />
              <Button
                variant="secondary"
                size="sm"
                onClick={handleSaveNamedConfig}
                className={cn(
                  "text-xs h-8",
                  saveStatus === 'saved' && 'bg-green-600 text-white hover:bg-green-700',
                  saveStatus === 'error' && 'bg-red-600 text-white hover:bg-red-700'
                )}
              >
                {saveStatus === 'saved' ? '✓ 已保存' : saveStatus === 'error' ? '✗ 失败' : '保存预设'}
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* 流式播放面板 */}
      <StreamPlaybackPanel
        state={streamState}
        metrics={streamMetrics}
        pcmChunks={playerRef.current?.pcmChunks ?? []}
        sampleRate={features.defaultSampleRate}
        onStop={handleStopStream}
      />

      {/* 非流式结果 */}
      {audioUrl && !isStreamActive && (
        <div className="border rounded-lg p-4 space-y-3">
          <h3 className="text-sm font-medium">{t('tts.result', '生成结果')}</h3>
          <audio
            ref={audioRef}
            src={audioUrl}
            onEnded={() => setIsPlaying(false)}
            onPause={() => setIsPlaying(false)}
            onPlay={() => setIsPlaying(true)}
          />
          <div className="flex items-center gap-2">
            <Button variant="outline" size="icon" onClick={handlePlayPause}>
              {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
            </Button>
            <Button variant="outline" size="icon" onClick={handleDownload} title={t('tts.download', '下载音频')}>
              <Download className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
