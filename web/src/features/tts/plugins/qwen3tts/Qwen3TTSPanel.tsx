import { useState, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Volume2, Loader2, Settings2, ChevronDown, AlertCircle, Play } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectSeparator, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ModelSelect } from '@/components/model/ModelSelect';
import { AvailableModelList } from '@/components/model/AvailableModelList';
import { useAvailableModels } from '@/features/creative/hooks';
import { BACKEND_LABELS } from '@/lib/constants/model';
import {
  useVoices,
  useTTSConfig,
} from '@/features/tts/hooks';
import type { TTSRequest, TTSConfig } from '@/features/tts/types';
import { RefAudioInput } from '@/features/tts/components/RefAudioInput';
import { ConfigManager } from '@/features/tts/components/ConfigManager';
import { toast } from '@/hooks/useToast';
import { useLoadModel } from '@/features/models';
import { LoadModelDialog } from '@/features/models/components/LoadModelDialog';
import { cn } from '@/lib/utils';
import type { TTSPluginPanelProps } from '@/features/tts/types';
import { useTTSStore } from '@/stores/ttsStore';

// Qwen3-TTS predefined speakers
const QWEN3_SPEAKERS = [
  { value: 'Vivian', label: 'Vivian', desc: 'tts.qwen3.speakerVivian', fallback: 'Bright young female (Chinese)' },
  { value: 'Serena', label: 'Serena', desc: 'tts.qwen3.speakerSerena', fallback: 'Warm gentle female (Chinese)' },
  { value: 'Uncle_Fu', label: 'Uncle Fu', desc: 'tts.qwen3.speakerUncleFu', fallback: 'Seasoned mellow male (Chinese)' },
  { value: 'Dylan', label: 'Dylan', desc: 'tts.qwen3.speakerDylan', fallback: 'Clear natural male (Beijing)' },
  { value: 'Eric', label: 'Eric', desc: 'tts.qwen3.speakerEric', fallback: 'Lively husky male (Sichuan)' },
  { value: 'Ryan', label: 'Ryan', desc: 'tts.qwen3.speakerRyan', fallback: 'Dynamic rhythmic male (English)' },
  { value: 'Aiden', label: 'Aiden', desc: 'tts.qwen3.speakerAiden', fallback: 'Sunny clear male (English)' },
  { value: 'Ono_Anna', label: 'Ono Anna', desc: 'tts.qwen3.speakerOnoAnna', fallback: 'Playful light female (Japanese)' },
  { value: 'Sohee', label: 'Sohee', desc: 'tts.qwen3.speakerSohee', fallback: 'Warm emotional female (Korean)' },
];

// Qwen3-TTS supported languages
const QWEN3_LANGUAGES = [
  { value: 'auto', label: 'tts.languageAuto', fallback: 'Auto Detect' },
  { value: 'chinese', label: 'Chinese (中文)' },
  { value: 'english', label: 'English' },
  { value: 'japanese', label: 'Japanese (日本語)' },
  { value: 'korean', label: 'Korean (한국어)' },
  { value: 'german', label: 'German (Deutsch)' },
  { value: 'french', label: 'French (Français)' },
  { value: 'russian', label: 'Russian (Русский)' },
  { value: 'portuguese', label: 'Portuguese (Português)' },
  { value: 'spanish', label: 'Spanish (Español)' },
  { value: 'italian', label: 'Italian (Italiano)' },
];

// Generation mode
type GenerationMode = 'custom_voice' | 'voice_design' | 'voice_clone';

export function Qwen3TTSPanel({
  model: selectedModel,
  matchedModels,
  onGenerate,
  isGenerating,
  streamState,
  onModelChange,
  refAudioOverride,
  modelStatus,
  fullModelId,
}: TTSPluginPanelProps) {
  const { t } = useTranslation();
  const availableModels = useAvailableModels('tts');
  const loadModel = useLoadModel();
  const [showLoadDialog, setShowLoadDialog] = useState(false);

  // Filter available models to only show Qwen3-TTS-related ones
  const qwen3AvailableModels = useMemo(
    () => availableModels.filter((m) => {
      const nameLower = (m.name || m.id || '').toLowerCase();
      return nameLower.includes('qwen3-tts') || nameLower.includes('qwen3tts') || nameLower.includes('qwen3_tts');
    }),
    [availableModels]
  );

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';
  const modelIdForConfig = selectedModel?.id || '';

  // Model status
  const isModelRunning = !modelStatus || modelStatus === 'running';
  const isModelLoading = modelStatus === 'loading' || modelStatus === 'unloading';
  const isModelStopped = modelStatus === 'stopped';
  const isModelError = modelStatus === 'error';
  const isStreamActive = streamState === 'streaming' || streamState === 'playing';

  // Form state from Zustand store
  const qwen3Form = useTTSStore((s) => s.qwen3Form);
  const setQwen3Field = useTTSStore((s) => s.setQwen3Field);
  const {
    input, language, mode, speaker, instructions, voiceDesignPrompt,
    refAudio, refText, fastCloneMode, temperature, topP, topK,
    repetitionPenalty, maxNewTokens, seed,
  } = qwen3Form;

  // Transient UI state (kept as local useState)
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [refAudioName, setRefAudioName] = useState('');

  const { data: voices = [] } = useVoices(modelName);
  const { ttsConfig, saveConfig, deleteConfig } = useTTSConfig(modelIdForConfig);

  const backendLabel = selectedModel?.backendType
    ? BACKEND_LABELS[selectedModel.backendType] || selectedModel.backendType
    : '';

  // Restore config
  useEffect(() => {
    if (ttsConfig) {
      useTTSStore.getState().hydrateFromServerConfig('qwen3tts', ttsConfig);
    }
  }, [ttsConfig]);

  // Sync external ref audio override
  useEffect(() => {
    if (refAudioOverride) {
      useTTSStore.getState().setQwen3Field('refAudio', refAudioOverride);
      useTTSStore.getState().setQwen3Field('mode', 'voice_clone');
    }
  }, [refAudioOverride]);

  const getCurrentConfig = useCallback((): TTSConfig => {
    const form = useTTSStore.getState().qwen3Form;
    return {
      voice: form.speaker || undefined,
      stream: true,
      responseFormat: 'pcm',
      instructions: form.instructions || undefined,
      refAudio: form.refAudio || undefined,
      refText: form.refText || undefined,
      language: form.language === 'auto' ? undefined : form.language,
    };
  }, []);

  const handleSaveToServer = useCallback(() => {
    if (!modelIdForConfig) return;
    saveConfig.mutate(getCurrentConfig());
  }, [modelIdForConfig, getCurrentConfig, saveConfig]);

  const handleDeleteFromServer = useCallback(() => {
    if (!modelIdForConfig) return;
    deleteConfig.mutate(modelIdForConfig, {
      onSuccess: () => toast.success(t('tts.configDeleted', 'Config deleted')),
      onError: (err: Error) => toast.error(t('tts.deleteFailed', 'Delete failed'), err.message),
    });
  }, [modelIdForConfig, deleteConfig, t]);

  const handleLoadConfig = useCallback((cfg: TTSConfig) => {
    useTTSStore.getState().hydrateFromServerConfig('qwen3tts', cfg);
  }, []);

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
      toast.error(t('tts.modelError', 'Model encountered an error'));
      return;
    }

    const form = useTTSStore.getState().qwen3Form;

    if (!form.input.trim() && form.mode !== 'voice_clone') {
      toast.warning(t('tts.inputRequired', 'Please enter text'));
      return;
    }
    if (form.mode === 'voice_clone' && !form.refAudio) {
      toast.warning(t('tts.qwen3.refAudioRequired', 'Please provide reference audio'));
      return;
    }
    if (form.mode === 'voice_design' && !form.voiceDesignPrompt.trim()) {
      toast.warning(t('tts.qwen3.voiceDesignRequired', 'Please enter voice design description'));
      return;
    }

    const payload: TTSRequest = {
      model: modelName,
      input: form.input.trim(),
      response_format: 'pcm',
      stream: true,
    };

    // Language
    if (form.language && form.language !== 'auto') {
      payload.language = form.language;
    }

    // Mode-specific parameters
    if (form.mode === 'custom_voice') {
      payload.voice = form.speaker;
      if (form.instructions.trim()) {
        payload.instructions = form.instructions.trim();
      }
    } else if (form.mode === 'voice_design') {
      payload.instructions = form.voiceDesignPrompt.trim();
    } else if (form.mode === 'voice_clone') {
      payload.ref_audio = form.refAudio;
      if (form.refText.trim()) {
        payload.ref_text = form.refText.trim();
      }
    }

    // Sampling params
    if (form.temperature && parseFloat(form.temperature) !== 0.9) {
      payload.temperature = parseFloat(form.temperature);
    }
    if (form.topP && parseFloat(form.topP) !== 1.0) {
      payload.top_p = parseFloat(form.topP);
    }
    if (form.topK && parseInt(form.topK, 10) !== 50) {
      payload.top_k = parseInt(form.topK, 10);
    }
    if (form.repetitionPenalty && parseFloat(form.repetitionPenalty) !== 1.05) {
      payload.repetition_penalty = parseFloat(form.repetitionPenalty);
    }
    if (form.maxNewTokens) {
      payload.max_new_tokens = parseInt(form.maxNewTokens, 10) || undefined;
    }
    if (form.seed) {
      payload.seed = parseInt(form.seed, 10) || undefined;
    }
    if (form.mode === 'voice_clone' && form.fastCloneMode) {
      payload.x_vector_only_mode = true;
    }

    onGenerate(payload);
  }, [modelName, isModelStopped, isModelLoading, isModelError, onGenerate, t]);

  // Empty state
  if (matchedModels.length === 0) {
    return (
      <div className="space-y-4">
        <div className="flex flex-col items-center gap-3 py-6 px-4 rounded-lg border border-dashed border-orange-300 bg-orange-50/50 dark:border-orange-700 dark:bg-orange-950/20">
          <AlertCircle className="w-8 h-8 text-orange-500" />
          <div className="text-center">
            <p className="text-sm font-medium text-foreground">
              {t('tts.qwen3.notDetected', '未检测到已加载的 Qwen3-TTS 模型')}
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              {t('tts.qwen3.notDetectedHint', '请从下方列表中选择一个 Qwen3-TTS 模型进行加载')}
            </p>
          </div>
        </div>
        <AvailableModelList
          models={qwen3AvailableModels}
          emptyText={t('tts.qwen3.noModels', '未扫描到 Qwen3-TTS 模型')}
          emptyHint={t('tts.qwen3.noModelsHint', '请确认已配置 Qwen3-TTS 模型路径并完成扫描')}
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
            const store = useTTSStore.getState();
            store.setQwen3Field('speaker', 'Vivian');
            store.setQwen3Field('refAudio', '');
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
            {(isModelStopped || isModelError) && <AlertCircle className="w-4 h-4" />}
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

      {/* Generation mode selector */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.qwen3.modeLabel', '生成模式')}
        </label>
        <Select value={mode} onValueChange={(v) => setQwen3Field('mode', v as GenerationMode)}>
          <SelectTrigger className="w-full bg-background">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="custom_voice">
              {t('tts.qwen3.modeCustomVoice', '预设音色 (CustomVoice)')}
            </SelectItem>
            <SelectItem value="voice_design">
              {t('tts.qwen3.modeVoiceDesign', '语音设计 (VoiceDesign)')}
            </SelectItem>
            <SelectItem value="voice_clone">
              {t('tts.qwen3.modeVoiceClone', '声音克隆 (VoiceClone)')}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Text input */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.inputLabel', 'Input Text')}
        </label>
        <Textarea
          value={input}
          onChange={(e) => setQwen3Field('input', e.target.value)}
          placeholder={t('tts.inputPlaceholder', 'Enter text to convert to speech...')}
          className="w-full bg-background resize-none"
          rows={4}
        />
      </div>

      {/* Mode-specific controls */}
      {mode === 'custom_voice' && (
        <>
          {/* Speaker selection */}
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.qwen3.speakerLabel', '预设音色')}
            </label>
            <Select value={speaker} onValueChange={(v) => setQwen3Field('speaker', v)}>
              <SelectTrigger className="w-full bg-background">
                <SelectValue placeholder={t('tts.qwen3.selectSpeaker', 'Select speaker')} />
              </SelectTrigger>
              <SelectContent>
                {/* Check if backend returned voices */}
                {voices.length > 0 ? (
                  voices.map(v => (
                    <SelectItem key={v.id} value={v.id}>
                      {v.name}{v.description ? ` — ${v.description}` : ''}
                    </SelectItem>
                  ))
                ) : (
                  QWEN3_SPEAKERS.map(s => (
                    <SelectItem key={s.value} value={s.value}>
                      {s.label} — {t(s.desc, s.fallback)}
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
          </div>

          {/* Style instructions (1.7B only) */}
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.qwen3.instructLabel', '风格指令 (1.7B)')}
            </label>
            <Input
              value={instructions}
              onChange={(e) => setQwen3Field('instructions', e.target.value)}
              placeholder={t('tts.qwen3.instructPlaceholder', '可选: 用温柔的语气朗读 / Speak in a cheerful tone...')}
              className="bg-background"
            />
            <p className="text-xs text-muted-foreground mt-1">
              {t('tts.qwen3.instructHint', '仅 1.7B CustomVoice 模型支持风格指令控制')}
            </p>
          </div>
        </>
      )}

      {mode === 'voice_design' && (
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('tts.qwen3.voiceDesignLabel', '语音描述')}
          </label>
          <Textarea
            value={voiceDesignPrompt}
            onChange={(e) => setQwen3Field('voiceDesignPrompt', e.target.value)}
            placeholder={t('tts.qwen3.voiceDesignPlaceholder', '用自然语言描述想要的声音特征，如：\n年轻女性，声音甜美温柔，语速适中，带有轻微的撒娇语气...\nA warm male voice, mid-30s, with a gentle authoritative tone...')}
            className="w-full bg-background resize-none"
            rows={4}
          />
          <p className="text-xs text-muted-foreground mt-1">
            {t('tts.qwen3.voiceDesignHint', '通过自然语言描述创建全新声音，支持中英文描述')}
          </p>
        </div>
      )}

      {mode === 'voice_clone' && (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.qwen3.refAudioLabel', '参考音频')}
            </label>
            <RefAudioInput value={refAudio} onChange={(v, name) => { setQwen3Field('refAudio', v); setRefAudioName(name || ''); }} fileName={refAudioName} />
            <p className="text-xs text-muted-foreground mt-1">
              {t('tts.qwen3.refAudioHint', '建议 3 秒以上清晰语音')}
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.qwen3.refTextLabel', '参考音频转录')}
            </label>
            <Textarea
              value={refText}
              onChange={(e) => setQwen3Field('refText', e.target.value)}
              placeholder={t('tts.qwen3.refTextPlaceholder', '输入参考音频对应的文本（ICL 模式需要，快速模式可选）')}
              rows={2}
              className="bg-background"
            />
          </div>
          <div className="flex items-center gap-3">
            <Switch checked={fastCloneMode} onCheckedChange={(v) => setQwen3Field('fastCloneMode', v)} />
            <div>
              <Label className="text-sm font-medium">
                {t('tts.qwen3.fastClone', '快速克隆模式')}
              </Label>
              <p className="text-xs text-muted-foreground">
                {t('tts.qwen3.fastCloneDesc', '仅提取音色向量，速度更快但相似度稍低')}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Language */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.languageLabel', 'Language')}
        </label>
        <Select value={language} onValueChange={(v) => setQwen3Field('language', v)}>
          <SelectTrigger className="w-full bg-background">
            <SelectValue placeholder={t('tts.languageAuto', 'Auto Detect')} />
          </SelectTrigger>
          <SelectContent>
            {QWEN3_LANGUAGES.map(l => (
              <SelectItem key={l.value} value={l.value}>
                {l.fallback || l.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Advanced settings */}
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
          {/* Temperature */}
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.qwen3.temperature', 'Temperature')}: {temperature}
            </label>
            <Slider
              value={[parseFloat(temperature) || 0.9]}
              onValueChange={([val]) => setQwen3Field('temperature', String(val))}
              min={0.1}
              max={1.5}
              step={0.05}
              className="w-full mt-2"
            />
          </div>

          {/* Top-P */}
          <div>
            <label className="block text-sm font-medium mb-1.5">
              Top-P: {topP}
            </label>
            <Slider
              value={[parseFloat(topP) || 1.0]}
              onValueChange={([val]) => setQwen3Field('topP', String(val))}
              min={0.5}
              max={1.0}
              step={0.05}
              className="w-full mt-2"
            />
          </div>

          {/* Top-K */}
          <div>
            <label className="block text-sm font-medium mb-1.5">
              Top-K: {topK}
            </label>
            <Slider
              value={[parseInt(topK) || 50]}
              onValueChange={([val]) => setQwen3Field('topK', String(val))}
              min={1}
              max={100}
              step={1}
              className="w-full mt-2"
            />
          </div>

          {/* Repetition Penalty */}
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.qwen3.repetitionPenalty', 'Repetition Penalty')}: {repetitionPenalty}
            </label>
            <Slider
              value={[parseFloat(repetitionPenalty) || 1.05]}
              onValueChange={([val]) => setQwen3Field('repetitionPenalty', String(val))}
              min={1.0}
              max={2.0}
              step={0.05}
              className="w-full mt-2"
            />
          </div>

          {/* Seed + Max tokens */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.seed', 'Seed')}
              </label>
              <Input
                value={seed}
                onChange={(e) => setQwen3Field('seed', e.target.value)}
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
                onChange={(e) => setQwen3Field('maxNewTokens', e.target.value)}
                placeholder="2048"
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
        disabled={isGenerating || !modelName || (!input.trim() && mode !== 'voice_clone') || isModelLoading || isModelError}
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
