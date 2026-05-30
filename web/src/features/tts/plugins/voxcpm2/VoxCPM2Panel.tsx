import { useState, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Volume2, Loader2, Settings2, ChevronDown, AlertCircle, Play } from 'lucide-react';
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
import { useLoadModel } from '@/features/models';
import { LoadModelDialog } from '@/features/models/components/LoadModelDialog';
import { cn } from '@/lib/utils';
import type { TTSPluginPanelProps } from '../../types';

type VoxCPM2Mode = 'standard' | 'voice_design' | 'voice_clone' | 'ultimate_cloning';

const VOXCPM2_LANGUAGES = [
  // Auto detect
  { value: 'auto', group: 'auto', label: 'tts.languageAuto', fallback: 'Auto Detect' },
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
  } = props;
  const { t } = useTranslation();
  const availableModels = useAvailableModels('tts');
  const asrModels = useAvailableModels('asr');
  const loadModel = useLoadModel();
  const autoTranscribe = useAutoTranscribe();
  const [showLoadDialog, setShowLoadDialog] = useState(false);

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

  // --- Generation mode & state ---
  const [mode, setMode] = useState<VoxCPM2Mode>('standard');
  const [input, setInput] = useState('');
  const [voiceDesignPrompt, setVoiceDesignPrompt] = useState('');
  const [styleDescription, setStyleDescription] = useState('');
  const [refAudio, setRefAudio] = useState('');
  const [refText, setRefText] = useState('');
  const [seed, setSeed] = useState('');
  const [maxNewTokens, setMaxNewTokens] = useState('');
  const [language, setLanguage] = useState('auto');
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [isTranscribing, setIsTranscribing] = useState(false);

  const isStreamActive = streamState === 'streaming' || streamState === 'playing';

  const { ttsConfig, saveConfig, deleteConfig } = useTTSConfig(modelIdForConfig);

  const backendLabel = selectedModel?.backendType
    ? BACKEND_LABELS[selectedModel.backendType] || selectedModel.backendType
    : '';

  // --- Config restore ---
  /* eslint-disable react-hooks/set-state-in-effect -- restoring state from persisted config */
  useEffect(() => {
    if (!ttsConfig) return;
    if (ttsConfig.mode) {
      setMode(ttsConfig.mode as VoxCPM2Mode);
    }
    if (ttsConfig.voiceDesignPrompt !== undefined) setVoiceDesignPrompt(ttsConfig.voiceDesignPrompt);
    if (ttsConfig.styleDescription !== undefined) setStyleDescription(ttsConfig.styleDescription);
    if (ttsConfig.refAudio !== undefined) setRefAudio(ttsConfig.refAudio);
    if (ttsConfig.refText !== undefined) setRefText(ttsConfig.refText);
    if (ttsConfig.seed !== undefined) setSeed(ttsConfig.seed);
    if (ttsConfig.maxNewTokens !== undefined) setMaxNewTokens(ttsConfig.maxNewTokens);
    if (ttsConfig.language !== undefined) setLanguage(ttsConfig.language || 'auto');
  }, [ttsConfig]);
  /* eslint-enable react-hooks/set-state-in-effect */

  // --- refAudioOverride: auto-switch mode ---
  /* eslint-disable react-hooks/set-state-in-effect -- syncing external ref audio override */
  useEffect(() => {
    if (!refAudioOverride) return;
    setRefAudio(refAudioOverride);
    if (mode === 'standard' || mode === 'voice_design') {
      setMode('voice_clone');
    }
  }, [refAudioOverride]); // eslint-disable-line react-hooks/exhaustive-deps
  /* eslint-enable react-hooks/set-state-in-effect */

  // --- Config persistence ---
  const getCurrentConfig = useCallback((): TTSConfig => ({
    stream: true,
    responseFormat: 'pcm',
    mode,
    voiceDesignPrompt: voiceDesignPrompt || undefined,
    styleDescription: styleDescription || undefined,
    refAudio: refAudio || undefined,
    refText: refText || undefined,
    seed: seed || undefined,
    maxNewTokens: maxNewTokens || undefined,
    language: language === 'auto' ? undefined : language || undefined,
  }), [mode, voiceDesignPrompt, styleDescription, refAudio, refText,
       seed, maxNewTokens, language]);

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
    if (cfg.mode) {
      setMode(cfg.mode as VoxCPM2Mode);
    }
    if (cfg.refAudio !== undefined) setRefAudio(cfg.refAudio);
    if (cfg.refText !== undefined) setRefText(cfg.refText);
    if (cfg.seed !== undefined) setSeed(cfg.seed);
    if (cfg.maxNewTokens !== undefined) setMaxNewTokens(cfg.maxNewTokens);
    if (cfg.language !== undefined) setLanguage(cfg.language || 'auto');
    if (cfg.voiceDesignPrompt !== undefined) setVoiceDesignPrompt(cfg.voiceDesignPrompt);
    if (cfg.styleDescription !== undefined) setStyleDescription(cfg.styleDescription);
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

    // Mode-specific validation
    const needsInput = mode !== 'ultimate_cloning';
    if (needsInput && !input.trim()) {
      toast.warning(t('tts.inputRequired', 'Please enter text'));
      return;
    }
    if (mode === 'voice_clone' && !refAudio) {
      toast.warning(t('tts.voxcpm2.refAudioHint', 'Please provide reference audio'));
      return;
    }
    if (mode === 'ultimate_cloning' && !refAudio) {
      toast.warning(t('tts.voxcpm2.refAudioHint', 'Please provide reference audio'));
      return;
    }
    if (mode === 'voice_design' && !voiceDesignPrompt.trim()) {
      toast.warning(t('tts.voxcpm2.voiceDesignLabel', 'Please enter voice description'));
      return;
    }

    // Build input text with bracket convention
    let formattedInput = input.trim();
    if (mode === 'voice_design' && voiceDesignPrompt.trim()) {
      formattedInput = `(${voiceDesignPrompt.trim()})${formattedInput}`;
    }
    if (mode === 'voice_clone' && styleDescription.trim()) {
      formattedInput = `(${styleDescription.trim()})${formattedInput}`;
    }

    const payload: TTSRequest = {
      model: modelName,
      input: mode === 'ultimate_cloning' ? (refText || '') : formattedInput,
      response_format: 'pcm',
      stream: true,
    };

    // Mode-specific params
    if (mode === 'voice_clone' && refAudio) {
      payload.ref_audio = refAudio;
      payload.ref_text = refText || undefined;
    }
    if (mode === 'ultimate_cloning' && refAudio) {
      payload.ref_audio = refAudio;
      payload.ref_text = refText || undefined;
    }

    // Common params
    if (seed) payload.seed = parseInt(seed, 10) || undefined;
    if (maxNewTokens) payload.max_new_tokens = parseInt(maxNewTokens, 10) || undefined;
    if (language && language !== 'auto') payload.language = language;

    onGenerate(payload);
  }, [modelName, mode, input, voiceDesignPrompt, styleDescription,
       refAudio, refText,
       seed, maxNewTokens, language,
       isModelStopped, isModelLoading, isModelError,
       onGenerate, t]);

  // --- Auto-transcribe ---
  const handleAutoTranscribe = useCallback(() => {
    const asrModel = asrModels[0];
    if (!asrModel) {
      toast.warning(t('tts.voxcpm2.noAsrModel'));
      return;
    }
    if (!refAudio) return;
    setIsTranscribing(true);
    autoTranscribe.mutate(
      { audioSource: refAudio, asrModelName: asrModel.alias || asrModel.name },
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
  }, [asrModels, refAudio, autoTranscribe, t]);

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

      {/* Mode selector */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.voxcpm2.modeLabel')}
        </label>
        <Select value={mode} onValueChange={(v) => setMode(v as VoxCPM2Mode)}>
          <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="standard">{t('tts.voxcpm2.modeStandard')}</SelectItem>
            <SelectItem value="voice_design">{t('tts.voxcpm2.modeVoiceDesign')}</SelectItem>
            <SelectItem value="voice_clone">{t('tts.voxcpm2.modeVoiceClone')}</SelectItem>
            <SelectItem value="ultimate_cloning">{t('tts.voxcpm2.modeUltimateCloning')}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* ===== Standard mode ===== */}
      {mode === 'standard' && (
        <>
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
            <p className="text-xs text-muted-foreground mt-1">
              {t('tts.voxcpm2.voiceDesignHint')}
            </p>
          </div>
        </>
      )}

      {/* ===== Voice Design mode ===== */}
      {mode === 'voice_design' && (
        <>
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.voxcpm2.voiceDesignLabel')}
            </label>
            <Textarea
              value={voiceDesignPrompt}
              onChange={(e) => setVoiceDesignPrompt(e.target.value)}
              placeholder={t('tts.voxcpm2.voiceDesignPlaceholder')}
              className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
              rows={4}
            />
            <p className="text-xs text-muted-foreground mt-1">
              {t('tts.voxcpm2.voiceDesignHint')}
            </p>
          </div>
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
        </>
      )}

      {/* ===== Voice Clone mode ===== */}
      {mode === 'voice_clone' && (
        <>
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.refAudio', 'Reference Audio')}
            </label>
            <RefAudioInput value={refAudio} onChange={setRefAudio} />
            <p className="text-xs text-muted-foreground mt-1">
              {t('tts.voxcpm2.refAudioHint')}
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.refText', 'Reference Audio Transcription')}
            </label>
            <Textarea
              value={refText}
              onChange={(e) => setRefText(e.target.value)}
              placeholder={t('tts.refTextPlaceholder', 'Enter transcription of the reference audio (optional)')}
              rows={2}
              className="bg-background"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.voxcpm2.styleDescription')}
            </label>
            <Input
              value={styleDescription}
              onChange={(e) => setStyleDescription(e.target.value)}
              placeholder={t('tts.voxcpm2.stylePlaceholder')}
              className="bg-background"
            />
          </div>
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
        </>
      )}

      {/* ===== Ultimate Cloning mode ===== */}
      {mode === 'ultimate_cloning' && (
        <>
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.refAudio', 'Reference Audio')}
            </label>
            <RefAudioInput value={refAudio} onChange={setRefAudio} />
            <p className="text-xs text-muted-foreground mt-1">
              {t('tts.voxcpm2.refAudioHint')}
            </p>
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="block text-sm font-medium">
                {t('tts.refText', 'Reference Audio Transcription')}
              </label>
              <Button
                variant="outline"
                size="sm"
                className="h-7 text-xs gap-1"
                disabled={!refAudio || asrModels.length === 0 || isTranscribing}
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
              rows={3}
              className="bg-background"
            />
          </div>
        </>
      )}

      {/* Language: only for Standard and Voice Clone modes */}
      {(mode === 'standard' || mode === 'voice_clone') && (
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
      )}

      {/* Advanced: seed and maxNewTokens only */}
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

      {/* Load model dialog */}
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
    </div>
  );
}
