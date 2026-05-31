import { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Volume2, Loader2, Settings2, ChevronDown, AlertCircle, Play, Upload, Trash2, Mic } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectSeparator, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { AvailableModelList } from '@/features/creative/AvailableModelList';
import { useAvailableModels, BACKEND_LABELS } from '@/features/creative/hooks';
import {
  useTTSConfig,
  useAutoTranscribe,
} from '../../hooks';
import type { TTSRequest, TTSConfig } from '../../types';
import { RefAudioInput } from '../../components/RefAudioInput';
import { ConfigManager } from '../../components/ConfigManager';
import { toast } from '@/hooks/useToast';
import { useLoadModel, useModels, useAllModelCapabilities } from '@/features/models';
import { LoadModelDialog } from '@/features/models/components/LoadModelDialog';
import { cn } from '@/lib/utils';
import type { TTSPluginPanelProps } from '../../types';
import { listVoices, uploadVoice, deleteVoice, type VoiceInfo } from '@/lib/api/voices';

const VOXCPM2_LANGUAGES = [
  // 30 official languages (alphabetical)
  { value: 'Arabic', group: 'official', label: 'Arabic' },
  { value: 'Burmese', group: 'official', label: 'Burmese' },
  { value: 'Chinese', group: 'official', label: 'Chinese (普通话)' },
  { value: 'Danish', group: 'official', label: 'Danish' },
  { value: 'Dutch', group: 'official', label: 'Dutch' },
  { value: 'English', group: 'official', label: 'English' },
  { value: 'Finnish', group: 'official', label: 'Finnish' },
  { value: 'French', group: 'official', label: 'French' },
  { value: 'German', group: 'official', label: 'German' },
  { value: 'Greek', group: 'official', label: 'Greek' },
  { value: 'Hebrew', group: 'official', label: 'Hebrew' },
  { value: 'Hindi', group: 'official', label: 'Hindi' },
  { value: 'Indonesian', group: 'official', label: 'Indonesian' },
  { value: 'Italian', group: 'official', label: 'Italian' },
  { value: 'Japanese', group: 'official', label: 'Japanese' },
  { value: 'Khmer', group: 'official', label: 'Khmer' },
  { value: 'Korean', group: 'official', label: 'Korean' },
  { value: 'Lao', group: 'official', label: 'Lao' },
  { value: 'Malay', group: 'official', label: 'Malay' },
  { value: 'Norwegian', group: 'official', label: 'Norwegian' },
  { value: 'Polish', group: 'official', label: 'Polish' },
  { value: 'Portuguese', group: 'official', label: 'Portuguese' },
  { value: 'Russian', group: 'official', label: 'Russian' },
  { value: 'Spanish', group: 'official', label: 'Spanish' },
  { value: 'Swahili', group: 'official', label: 'Swahili' },
  { value: 'Swedish', group: 'official', label: 'Swedish' },
  { value: 'Tagalog', group: 'official', label: 'Tagalog' },
  { value: 'Thai', group: 'official', label: 'Thai' },
  { value: 'Turkish', group: 'official', label: 'Turkish' },
  { value: 'Vietnamese', group: 'official', label: 'Vietnamese' },
  // 9 Chinese dialects
  { value: '四川话', group: 'dialect', label: '四川话 (Sichuanese)' },
  { value: '粤语', group: 'dialect', label: '粤语 (Cantonese)' },
  { value: '吴语', group: 'dialect', label: '吴语 (Wu)' },
  { value: '东北话', group: 'dialect', label: '东北话 (Northeastern)' },
  { value: '河南话', group: 'dialect', label: '河南话 (Henan)' },
  { value: '陕西话', group: 'dialect', label: '陕西话 (Shanxi)' },
  { value: '山东话', group: 'dialect', label: '山东话 (Shandong)' },
  { value: '天津话', group: 'dialect', label: '天津话 (Tianjin)' },
  { value: '闽南话', group: 'dialect', label: '闽南话 (Min Nan)' },
];

export function VoxCPM2Panel(props: TTSPluginPanelProps) {
  const {
    model: selectedModel,
    matchedModels,
    onGenerate,
    isGenerating,
    streamState,
    onModelChange,
    refAudioOverride,
    modelStatus,
    fullModelId,
    voiceRefreshTrigger,
  } = props;
  const { t } = useTranslation();
  const availableModels = useAvailableModels('tts');
  const loadModel = useLoadModel();
  const autoTranscribe = useAutoTranscribe();
  const [showLoadDialog, setShowLoadDialog] = useState(false);

  // ASR 模型：获取所有具有 ASR 能力的模型（包括已加载和未加载）
  const { data: allModels = [] } = useModels();
  const allModelIds = useMemo(() => allModels.map((m) => m.id), [allModels]);
  const capsResults = useAllModelCapabilities(allModelIds);
  const asrModels = useMemo(() => {
    return allModels.filter((m, i) => {
      const caps = capsResults[i]?.data;
      return caps?.asr === true;
    });
  }, [allModels, capsResults]);

  const [selectedAsrModelId, setSelectedAsrModelId] = useState<string>('');
  const [showAsrLoadDialog, setShowAsrLoadDialog] = useState(false);

  // 默认选中第一个 ASR 模型
  useEffect(() => {
    if (!selectedAsrModelId && asrModels.length > 0) {
      setSelectedAsrModelId(asrModels[0].id);
    }
  }, [asrModels, selectedAsrModelId]);

  // 选中的 ASR 模型信息
  const selectedAsrModel = useMemo(
    () => asrModels.find((m) => m.id === selectedAsrModelId),
    [asrModels, selectedAsrModelId]
  );
  const asrModelName = selectedAsrModel ? (selectedAsrModel.alias || selectedAsrModel.name || selectedAsrModel.id) : '';
  const isAsrModelRunning = selectedAsrModel?.isLoaded;
  const isAsrModelLoading = selectedAsrModel?.status === 'loading' || selectedAsrModel?.status === 'unloading';
  const isAsrModelStopped = selectedAsrModel?.status === 'stopped' || (!selectedAsrModel?.isLoaded && selectedAsrModel?.status !== 'loading');

  // Filter available models to only show VoxCPM2-related ones
  const voxcpmAvailableModels = useMemo(
    () => availableModels.filter((m) => {
      const nameLower = (m.name || m.id || '').toLowerCase();
      return nameLower.includes('voxcpm');
    }),
    [availableModels]
  );

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';
  const modelIdForConfig = selectedModel?.id || '';

  // Model status derived state
  const isModelRunning = !modelStatus || modelStatus === 'running';
  const isModelLoading = modelStatus === 'loading' || modelStatus === 'unloading';
  const isModelStopped = modelStatus === 'stopped';
  const isModelError = modelStatus === 'error';

  // --- State (no mode selector — fields are always visible) ---
  const [input, setInput] = useState('');
  const [refAudio, setRefAudio] = useState('');
  const [refText, setRefText] = useState('');
  const [instructions, setInstructions] = useState('');
  const [seed, setSeed] = useState('');
  const [maxNewTokens, setMaxNewTokens] = useState('');
  const [language, setLanguage] = useState('auto');
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [isTranscribing, setIsTranscribing] = useState(false);

  // Voice management
  const [uploadedVoices, setUploadedVoices] = useState<VoiceInfo[]>([]);
  const [isLoadingVoices, setIsLoadingVoices] = useState(false);
  const [isUploadingVoice, setIsUploadingVoice] = useState(false);
  const [selectedVoice, setSelectedVoice] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Reference audio collapsible
  const [refAudioOpen, setRefAudioOpen] = useState(false);

  const loadVoices = useCallback(async () => {
    if (!modelName) return;
    setIsLoadingVoices(true);
    try {
      const res = await listVoices(modelName);
      setUploadedVoices(res.uploaded_voices || []);
    } catch {
      // model may not be loaded yet
    } finally {
      setIsLoadingVoices(false);
    }
  }, [modelName]);

  const handleUploadVoice = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !modelName) return;
    const voiceName = file.name.replace(/\.[^.]+$/, '');
    setIsUploadingVoice(true);
    try {
      await uploadVoice(modelName, file, voiceName);
      toast.success(t('tts.voxcpm2.voiceUploaded', 'Voice uploaded'));
      await loadVoices();
    } catch (err) {
      toast.error(t('tts.voxcpm2.voiceUploadFailed', 'Upload failed'), (err as Error).message);
    } finally {
      setIsUploadingVoice(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  }, [modelName, loadVoices, t]);

  const handleDeleteVoice = useCallback(async (voiceName: string) => {
    if (!modelName) return;
    try {
      await deleteVoice(modelName, voiceName);
      toast.success(t('tts.voxcpm2.voiceDeleted', 'Voice deleted'));
      if (selectedVoice === voiceName) setSelectedVoice('');
      await loadVoices();
    } catch (err) {
      toast.error(t('tts.voxcpm2.voiceDeleteFailed', 'Delete failed'), (err as Error).message);
    }
  }, [modelName, selectedVoice, loadVoices, t]);

  // Load voices when model is running
  useEffect(() => { if (isModelRunning) loadVoices(); }, [isModelRunning, loadVoices]);

  // Reload voices when external trigger changes (e.g., after saving voice from history)
  useEffect(() => {
    if (voiceRefreshTrigger && voiceRefreshTrigger > 0 && isModelRunning) loadVoices();
  }, [voiceRefreshTrigger, isModelRunning, loadVoices]);

  const isStreamActive = streamState === 'streaming' || streamState === 'playing';

  const { ttsConfig, saveConfig, deleteConfig } = useTTSConfig(modelIdForConfig);

  const backendLabel = selectedModel?.backendType
    ? BACKEND_LABELS[selectedModel.backendType] || selectedModel.backendType
    : '';

  // --- Config restore ---
  /* eslint-disable react-hooks/set-state-in-effect -- restoring state from persisted config */
  useEffect(() => {
    if (ttsConfig) {
      if (ttsConfig.instructions !== undefined) setInstructions(ttsConfig.instructions);
      if (ttsConfig.refAudio !== undefined) setRefAudio(ttsConfig.refAudio);
      if (ttsConfig.refText !== undefined) setRefText(ttsConfig.refText);
      if (ttsConfig.seed !== undefined) setSeed(ttsConfig.seed);
      if (ttsConfig.maxNewTokens !== undefined) setMaxNewTokens(ttsConfig.maxNewTokens);
      if (ttsConfig.language !== undefined) setLanguage(ttsConfig.language || 'auto');
    } else {
      // 模型切换后无 config 时清除残留状态
      setInstructions('');
      setRefAudio('');
      setRefText('');
      setSeed('');
      setMaxNewTokens('');
      setLanguage('auto');
    }
  }, [ttsConfig, modelIdForConfig]);
  /* eslint-enable react-hooks/set-state-in-effect */

  // --- refAudioOverride: set ref audio and expand section ---
  /* eslint-disable react-hooks/set-state-in-effect -- syncing external ref audio override */
  useEffect(() => {
    if (!refAudioOverride) return;
    setRefAudio(refAudioOverride);
    setRefAudioOpen(true);
  }, [refAudioOverride]); // eslint-disable-line react-hooks/exhaustive-deps
  /* eslint-enable react-hooks/set-state-in-effect */

  // --- Config persistence ---
  const getCurrentConfig = useCallback((): TTSConfig => ({
    stream: true,
    responseFormat: 'pcm',
    instructions: instructions || undefined,
    refAudio: refAudio || undefined,
    refText: refText || undefined,
    seed: seed || undefined,
    maxNewTokens: maxNewTokens || undefined,
    language: language === 'auto' ? undefined : language || undefined,
  }), [instructions, refAudio, refText, seed, maxNewTokens, language]);

  const handleSaveToServer = useCallback(() => {
    if (!modelIdForConfig) return;
    saveConfig.mutate({ modelId: modelIdForConfig, config: getCurrentConfig() as unknown as import('@/types/model').LoadModelParams });
  }, [modelIdForConfig, getCurrentConfig, saveConfig]);

  const handleDeleteFromServer = useCallback(() => {
    if (!modelIdForConfig) return;
    deleteConfig.mutate(modelIdForConfig, {
      onSuccess: () => toast.success(t('tts.configDeleted', 'Config deleted')),
      onError: (err: Error) => toast.error(t('tts.deleteFailed', 'Delete failed'), err.message),
    });
  }, [modelIdForConfig, deleteConfig, t]);

  const handleLoadConfig = useCallback((cfg: TTSConfig) => {
    if (cfg.instructions !== undefined) setInstructions(cfg.instructions);
    if (cfg.refAudio !== undefined) setRefAudio(cfg.refAudio);
    if (cfg.refText !== undefined) setRefText(cfg.refText);
    if (cfg.seed !== undefined) setSeed(cfg.seed);
    if (cfg.maxNewTokens !== undefined) setMaxNewTokens(cfg.maxNewTokens);
    if (cfg.language !== undefined) setLanguage(cfg.language || 'auto');
  }, []);

  // --- Generate logic ---
  const handleGenerate = useCallback(() => {
    if (!modelName) {
      toast.warning(t('tts.selectModelWarning', 'Please select a model'));
      return;
    }
    if (isModelStopped) {
      setShowLoadDialog(true);
      return;
    }
    if (isModelLoading) {
      toast.info(t('tts.modelLoading', 'Model is loading, please wait...'));
      return;
    }
    if (isModelError) {
      toast.error(t('tts.modelError', 'Model encountered an error, please check model status'));
      return;
    }
    if (!input.trim()) {
      toast.warning(t('tts.inputRequired', 'Please enter text'));
      return;
    }

    const payload: TTSRequest = {
      model: modelName,
      input: input.trim(),
      response_format: 'pcm',
      stream: true,
    };

    if (selectedVoice) {
      payload.voice = selectedVoice;
    } else if (refAudio) {
      payload.ref_audio = refAudio;
    }

    if (refText.trim()) payload.ref_text = refText.trim();
    if (instructions.trim()) payload.instructions = instructions.trim();
    const parsedSeed = parseInt(seed, 10);
    if (seed !== '' && !Number.isNaN(parsedSeed)) payload.seed = parsedSeed;
    const parsedTokens = parseInt(maxNewTokens, 10);
    if (maxNewTokens !== '' && !Number.isNaN(parsedTokens)) payload.max_new_tokens = parsedTokens;
    if (language && language !== 'auto') payload.language = language;

    onGenerate(payload);
  }, [modelName, input, refAudio, refText, instructions,
       selectedVoice, seed, maxNewTokens, language,
       isModelStopped, isModelLoading, isModelError, onGenerate, t]);

  // --- Auto-transcribe ---
  const handleAutoTranscribe = useCallback(() => {
    if (!selectedAsrModel || !asrModelName) {
      toast.warning(t('tts.voxcpm2.noAsrModel'));
      return;
    }
    if (!refAudio) return;
    if (isAsrModelStopped) {
      setShowAsrLoadDialog(true);
      return;
    }
    if (isAsrModelLoading) {
      toast.info(t('tts.modelLoading', 'ASR model is loading, please wait...'));
      return;
    }
    setIsTranscribing(true);
    autoTranscribe.mutate(
      { audioSource: refAudio, asrModelName },
      {
        onSuccess: (text) => {
          setRefText(text);
          setIsTranscribing(false);
        },
        onError: () => {
          setIsTranscribing(false);
          toast.error(t('tts.voxcpm2.transcribeFailed'));
        },
      }
    );
  }, [selectedAsrModel, asrModelName, refAudio, isAsrModelStopped, isAsrModelLoading, autoTranscribe, t]);

  // --- Empty state ---
  if (matchedModels.length === 0) {
    return (
      <div className="space-y-4">
        <div className="flex flex-col items-center gap-3 py-6 px-4 rounded-lg border border-dashed border-orange-300 bg-orange-50/50 dark:border-orange-700 dark:bg-orange-950/20">
          <AlertCircle className="w-8 h-8 text-orange-500" />
          <div className="text-center">
            <p className="text-sm font-medium text-foreground">
              {t('tts.voxcpm2.notDetected')}
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              {t('tts.voxcpm2.notDetectedHint')}
            </p>
          </div>
        </div>
        <AvailableModelList
          models={voxcpmAvailableModels}
          emptyText={t('tts.voxcpm2.noModels')}
          emptyHint={t('tts.voxcpm2.noModelsHint')}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Model selection */}
      <div>
        <ModelSelect
          models={matchedModels}
          value={modelName}
          onValueChange={(v) => {
            onModelChange(v);
            setRefAudio('');
            setSelectedVoice('');
            setRefText('');
            setInstructions('');
            setSeed('');
            setMaxNewTokens('');
            setLanguage('auto');
          }}
          placeholder={t('tts.selectModel', 'Select TTS model')}
          label={t('tts.modelLabel', 'TTS Model')}
          showBackend
        />
        {backendLabel && (
          <p className="text-xs text-muted-foreground mt-1">
            {t('tts.backend', 'Backend')}: {backendLabel}
          </p>
        )}
        {/* Model status indicator */}
        {modelName && !isModelRunning && (
          <div className={cn(
            'flex items-center gap-2 mt-2 px-3 py-2 rounded-md text-sm',
            isModelLoading && 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400',
            isModelStopped && 'bg-orange-500/10 text-orange-600 dark:text-orange-400',
            isModelError && 'bg-red-500/10 text-red-600 dark:text-red-400',
          )}>
            {isModelLoading && <Loader2 className="w-4 h-4 animate-spin" />}
            {isModelStopped && <AlertCircle className="w-4 h-4" />}
            {isModelError && <AlertCircle className="w-4 h-4" />}
            <span>
              {isModelLoading && t('tts.modelLoading', 'Model is loading, please wait...')}
              {isModelStopped && t('tts.modelNotLoaded', 'Model not loaded')}
              {isModelError && t('tts.modelError', 'Model encountered an error')}
            </span>
            {isModelStopped && fullModelId && (
              <Button
                variant="outline"
                size="sm"
                className="ml-auto h-7 text-xs gap-1"
                onClick={() => setShowLoadDialog(true)}
              >
                <Play className="w-3 h-3" />
                {t('tts.loadModel', 'Load Model')}
              </Button>
            )}
          </div>
        )}
      </div>

      {/* Input Text — always visible */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.inputLabel', 'Input Text')}
        </label>
        <Textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder={t('tts.inputPlaceholder', 'Enter text to convert to speech...')}
          className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
          rows={4}
        />
      </div>

      {/* Language selector — always visible */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.languageLabel', 'Language')}
        </label>
        <Select value={language} onValueChange={setLanguage}>
          <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
            <SelectValue placeholder={t('tts.languageAuto', 'Auto Detect')} />
          </SelectTrigger>
          <SelectContent position="popper" className="!max-h-[300px]">
            <SelectItem value="auto">{t('tts.languageAuto', 'Auto Detect')}</SelectItem>
            <SelectSeparator />
            <SelectGroup>
              <SelectLabel>{t('tts.languageOfficial', 'Official Languages')}</SelectLabel>
              {VOXCPM2_LANGUAGES.filter(l => l.group === 'official').map(l => (
                <SelectItem key={l.value} value={l.value}>{l.label}</SelectItem>
              ))}
            </SelectGroup>
            <SelectSeparator />
            <SelectGroup>
              <SelectLabel>{t('tts.languageDialects', 'Chinese Dialects')}</SelectLabel>
              {VOXCPM2_LANGUAGES.filter(l => l.group === 'dialect').map(l => (
                <SelectItem key={l.value} value={l.value}>{l.label}</SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      {/* Reference Audio — collapsible, default folded */}
      <Collapsible open={refAudioOpen} onOpenChange={setRefAudioOpen}>
        <CollapsibleTrigger asChild>
          <Button variant="ghost" className="w-full justify-between p-0 h-auto text-sm font-medium hover:bg-transparent">
            <span className="flex items-center gap-2">
              <Mic className="w-4 h-4" />
              {t('tts.voxcpm2.refAudioSection', 'Reference Audio')}
            </span>
            <ChevronDown className={`w-4 h-4 transition-transform ${refAudioOpen ? 'rotate-180' : ''}`} />
          </Button>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-2">
          {/* Voice Library */}
          <div className="space-y-2 rounded-lg border p-3">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">
                {t('tts.voxcpm2.voiceLibrary', 'Voice Library')}
              </label>
              <div className="flex items-center gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="audio/*"
                  className="hidden"
                  onChange={handleUploadVoice}
                />
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 text-xs gap-1"
                  disabled={!isModelRunning || isUploadingVoice}
                  onClick={() => fileInputRef.current?.click()}
                >
                  {isUploadingVoice ? <Loader2 className="w-3 h-3 animate-spin" /> : <Upload className="w-3 h-3" />}
                  {t('tts.voxcpm2.uploadVoice', 'Upload')}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 text-xs"
                  disabled={isLoadingVoices}
                  onClick={loadVoices}
                >
                  {isLoadingVoices ? <Loader2 className="w-3 h-3 animate-spin" /> : null}
                  {t('tts.voxcpm2.refresh', 'Refresh')}
                </Button>
              </div>
            </div>

            {uploadedVoices.length > 0 ? (
              <div className="space-y-1.5">
                {/* "None" option to clear selection */}
                <label className={cn(
                  'flex items-center gap-2 px-2.5 py-2 rounded-md border cursor-pointer text-sm transition-colors',
                  !selectedVoice ? 'border-primary bg-primary/5' : 'border-border hover:bg-muted/50',
                )}>
                  <input
                    type="radio"
                    name="voice-select"
                    className="accent-primary"
                    checked={!selectedVoice}
                    onChange={() => setSelectedVoice('')}
                  />
                  <Mic className="w-3.5 h-3.5 text-muted-foreground" />
                  <span className="flex-1">{t('tts.voxcpm2.useRefAudio', 'Use reference audio below')}</span>
                </label>
                {uploadedVoices.map((v) => (
                  <label key={v.name} className={cn(
                    'flex items-center gap-2 px-2.5 py-2 rounded-md border cursor-pointer text-sm transition-colors',
                    selectedVoice === v.name ? 'border-primary bg-primary/5' : 'border-border hover:bg-muted/50',
                  )}>
                    <input
                      type="radio"
                      name="voice-select"
                      className="accent-primary"
                      checked={selectedVoice === v.name}
                      onChange={() => setSelectedVoice(v.name)}
                    />
                    <Mic className="w-3.5 h-3.5 text-muted-foreground" />
                    <span className="flex-1 truncate">{v.name}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0 text-muted-foreground hover:text-destructive"
                      onClick={(e) => { e.preventDefault(); handleDeleteVoice(v.name); }}
                    >
                      <Trash2 className="w-3 h-3" />
                    </Button>
                  </label>
                ))}
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">
                {t('tts.voxcpm2.noVoices', 'No uploaded voices. Upload an audio file to use as voice reference.')}
              </p>
            )}
          </div>

          {/* Ref Audio Input — only when no uploaded voice is selected */}
          {!selectedVoice && (
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.refAudio', 'Reference Audio')}
              </label>
              <RefAudioInput value={refAudio} onChange={setRefAudio} />
              <p className="text-xs text-muted-foreground mt-1">
                {t('tts.voxcpm2.refAudioHint')}
              </p>
            </div>
          )}

          {/* Ref Text — auto-transcribe button with ASR model selector */}
          <div>
            {/* ASR 模型选择行 */}
            {asrModels.length > 0 && (
              <div className="flex items-center gap-2 mb-2">
                <Select value={selectedAsrModelId} onValueChange={setSelectedAsrModelId}>
                  <SelectTrigger className="flex-1 h-8 text-xs">
                    <SelectValue placeholder={t('tts.voxcpm2.selectAsrModel', 'Select ASR model')} />
                  </SelectTrigger>
                  <SelectContent position="popper">
                    {asrModels.map((m) => (
                      <SelectItem key={m.id} value={m.id}>
                        <span className="flex items-center gap-1.5">
                          {m.alias || m.displayName || m.name}
                          {m.isLoaded ? (
                            <span className="text-green-500">●</span>
                          ) : (
                            <span className="text-muted-foreground">○</span>
                          )}
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {/* ASR 模型状态 */}
                {selectedAsrModel && (
                  <div className={cn(
                    'flex items-center gap-1 text-xs shrink-0',
                    isAsrModelRunning && 'text-green-600 dark:text-green-400',
                    isAsrModelLoading && 'text-yellow-600 dark:text-yellow-400',
                    isAsrModelStopped && 'text-orange-600 dark:text-orange-400',
                  )}>
                    {isAsrModelRunning && <span>● {t('tts.modelRunning', 'Running')}</span>}
                    {isAsrModelLoading && <><Loader2 className="w-3 h-3 animate-spin" />{t('tts.modelLoading', 'Loading...')}</>}
                    {isAsrModelStopped && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 text-xs gap-1 text-orange-600 dark:text-orange-400 hover:text-orange-700"
                        onClick={() => setShowAsrLoadDialog(true)}
                      >
                        <Play className="w-3 h-3" />
                        {t('tts.loadModel', 'Load')}
                      </Button>
                    )}
                  </div>
                )}
              </div>
            )}
            {asrModels.length === 0 && (
              <p className="text-xs text-muted-foreground mb-2">
                {t('tts.voxcpm2.noAsrModel', 'No ASR models available. Please scan or download an ASR model first.')}
              </p>
            )}
            <div className="flex items-center justify-between mb-1.5">
              <label className="block text-sm font-medium">
                {t('tts.refText', 'Reference Audio Transcription')}
              </label>
              <Button
                variant="outline"
                size="sm"
                className="h-7 text-xs gap-1"
                disabled={!refAudio || !asrModelName || isTranscribing || !isAsrModelRunning}
                onClick={handleAutoTranscribe}
              >
                {isTranscribing ? (
                  <>
                    <Loader2 className="w-3 h-3 animate-spin" />
                    {t('tts.voxcpm2.transcribing')}
                  </>
                ) : (
                  t('tts.voxcpm2.autoTranscribe')
                )}
              </Button>
            </div>
            <Textarea
              value={refText}
              onChange={(e) => setRefText(e.target.value)}
              placeholder={t('tts.refTextPlaceholder', 'Enter transcription of the reference audio (optional)')}
              rows={2}
              className="bg-background"
            />
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* Instructions — optional voice description / style */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.voxcpm2.instructionsLabel', 'Instructions')}
        </label>
        <Textarea
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
          placeholder={t('tts.voxcpm2.instructionsPlaceholder', 'Describe voice style, e.g., (A warm male voice, gentle tone) or "speak softly"...')}
          className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
          rows={2}
        />
        <p className="text-xs text-muted-foreground mt-1">
          {t('tts.voxcpm2.instructionsHint', 'Optional. Used for voice design or style control.')}
        </p>
      </div>

      {/* Advanced: seed and maxNewTokens */}
      <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
        <CollapsibleTrigger asChild>
          <Button variant="ghost" className="w-full justify-between p-0 h-auto text-sm font-medium hover:bg-transparent">
            <span className="flex items-center gap-2">
              <Settings2 className="w-4 h-4" />
              {t('tts.advanced', 'Advanced Settings')}
            </span>
            <ChevronDown className={`w-4 h-4 transition-transform ${advancedOpen ? 'rotate-180' : ''}`} />
          </Button>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-2">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.seed', 'Seed')}
              </label>
              <Input
                value={seed}
                onChange={(e) => setSeed(e.target.value)}
                placeholder={t('tts.seedPlaceholder', 'Leave empty for random')}
                type="number"
                className="bg-background"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.maxNewTokens', 'Max New Tokens')}
              </label>
              <Input
                value={maxNewTokens}
                onChange={(e) => setMaxNewTokens(e.target.value)}
                placeholder={t('tts.maxNewTokens', 'Leave empty for default')}
                type="number"
                className="bg-background"
              />
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* Generate button */}
      <Button
        onClick={handleGenerate}
        disabled={isGenerating || !modelName || isModelLoading || isModelError}
        className="w-full"
      >
        {isGenerating ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            {isStreamActive ? t('tts.streamingInProgress', 'Streaming...') : t('tts.generating', 'Generating...')}
          </>
        ) : isModelLoading ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            {t('tts.modelLoading', 'Model is loading...')}
          </>
        ) : isModelStopped ? (
          <>
            <Volume2 className="w-4 h-4 mr-2" />
            {t('tts.loadModelToGenerate', 'Load Model & Generate')}
          </>
        ) : isModelError ? (
          <>
            <AlertCircle className="w-4 h-4 mr-2" />
            {t('tts.modelError', 'Model error')}
          </>
        ) : (
          <>
            <Volume2 className="w-4 h-4 mr-2" />
            {t('tts.generate', 'Generate Speech')}
          </>
        )}
      </Button>

      {/* Config management */}
      {modelName && (
        <ConfigManager
          modelName={modelName}
          modelId={modelIdForConfig}
          getCurrentConfig={getCurrentConfig}
          onLoadConfig={handleLoadConfig}
          onSaveToServer={handleSaveToServer}
          onDeleteFromServer={handleDeleteFromServer}
          hasServerConfig={!!ttsConfig}
        />
      )}

      {/* Load TTS model dialog */}
      {showLoadDialog && fullModelId && (
        <LoadModelDialog
          modelId={fullModelId}
          modelName={modelName}
          backendType={selectedModel?.backendType}
          isOpen={showLoadDialog}
          onClose={() => setShowLoadDialog(false)}
          onConfirm={(params) => {
            loadModel.mutate({ modelId: fullModelId, ...params });
            setShowLoadDialog(false);
          }}
        />
      )}

      {/* Load ASR model dialog */}
      {showAsrLoadDialog && selectedAsrModel && (
        <LoadModelDialog
          modelId={selectedAsrModel.id}
          modelName={selectedAsrModel.alias || selectedAsrModel.displayName || selectedAsrModel.name}
          modelPath={selectedAsrModel.path}
          backendType={selectedAsrModel.backendType}
          isOpen={showAsrLoadDialog}
          onClose={() => setShowAsrLoadDialog(false)}
          onConfirm={(params) => {
            loadModel.mutate({ modelId: selectedAsrModel.id, ...params });
            setShowAsrLoadDialog(false);
          }}
        />
      )}
    </div>
  );
}
