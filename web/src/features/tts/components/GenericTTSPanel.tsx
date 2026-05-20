import { useState, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Volume2, Loader2, Settings2, ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { AvailableModelList } from '@/features/creative/AvailableModelList';
import { useAvailableModels, BACKEND_LABELS } from '@/features/creative/hooks';
import {
  useVoices,
  useTTSConfig,
  getTTSModelFeatures,
  type TTSRequest,
  type TTSConfig,
} from '../hooks';
import { RefAudioInput } from './RefAudioInput';
import { ConfigManager } from './ConfigManager';
import { toast } from '@/hooks/useToast';
import type { TTSPluginPanelProps } from '../types';

const FALLBACK_VOICES = ['alloy', 'echo', 'fable', 'onyx', 'nova', 'shimmer'];

const AUDIO_FORMATS: Array<{ value: string; label: string; i18nKey?: string }> = [
  { value: 'mp3', label: 'MP3' },
  { value: 'wav', label: 'WAV' },
  { value: 'opus', label: 'Opus' },
  { value: 'flac', label: 'FLAC' },
  { value: 'pcm', label: 'PCM', i18nKey: 'tts.pcmFormat' },
];

export function GenericTTSPanel({
  model: selectedModel,
  matchedModels,
  onGenerate,
  isGenerating,
  onModelChange,
}: TTSPluginPanelProps) {
  const { t } = useTranslation();
  const availableModels = useAvailableModels('tts');

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';
  const modelIdForConfig = selectedModel?.id || '';

  const [input, setInput] = useState('');
  const [voice, setVoice] = useState('');
  const [speed, setSpeed] = useState(1);
  const [responseFormat, setResponseFormat] = useState('mp3');
  const [stream, setStream] = useState(false);
  const [language, setLanguage] = useState('');
  const [refAudio, setRefAudio] = useState('');
  const [refText, setRefText] = useState('');
  const [seed, setSeed] = useState('');
  const [maxNewTokens, setMaxNewTokens] = useState('');
  const [advancedOpen, setAdvancedOpen] = useState(false);

  const features = useMemo(
    () => (selectedModel ? getTTSModelFeatures(selectedModel) : null),
    [selectedModel]
  );

  const { ttsConfig, saveConfig, deleteConfig } = useTTSConfig(modelIdForConfig);
  const { data: voices = [] } = useVoices(modelName);

  const backendLabel = selectedModel?.backendType
    ? BACKEND_LABELS[selectedModel.backendType] || selectedModel.backendType
    : '';

  // Restore config from server
  useEffect(() => {
    if (ttsConfig) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      if (ttsConfig.voice !== undefined) setVoice(ttsConfig.voice);
      if (ttsConfig.speed !== undefined) setSpeed(ttsConfig.speed);
      if (ttsConfig.responseFormat !== undefined) setResponseFormat(ttsConfig.responseFormat);
      if (ttsConfig.stream !== undefined) setStream(ttsConfig.stream);
      if (ttsConfig.refAudio !== undefined) setRefAudio(ttsConfig.refAudio);
      if (ttsConfig.refText !== undefined) setRefText(ttsConfig.refText);
      if (ttsConfig.seed !== undefined) setSeed(ttsConfig.seed);
      if (ttsConfig.maxNewTokens !== undefined) setMaxNewTokens(ttsConfig.maxNewTokens);
      if (ttsConfig.language !== undefined) setLanguage(ttsConfig.language);
    }
  }, [ttsConfig]);

  // Auto-adjust format on model change
  useEffect(() => {
    if (selectedModel) {
      const fmt = getTTSModelFeatures(selectedModel).defaultFormat;
      if (fmt === 'pcm') {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setResponseFormat('pcm');
        setStream(true);
      }
    }
  }, [selectedModel]);

  const getCurrentConfig = useCallback((): TTSConfig => ({
    voice: voice || undefined,
    speed: speed !== 1 ? speed : undefined,
    responseFormat,
    stream,
    refAudio: refAudio || undefined,
    refText: refText || undefined,
    seed: seed || undefined,
    maxNewTokens: maxNewTokens || undefined,
    language: language || undefined,
  }), [voice, speed, responseFormat, stream, refAudio, refText, seed, maxNewTokens, language]);

  const handleSaveToServer = useCallback(() => {
    if (!modelIdForConfig) return;
    // TTS config is stored in the same model config endpoint - cast to expected type
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
    if (cfg.voice !== undefined) setVoice(cfg.voice);
    if (cfg.speed !== undefined) setSpeed(cfg.speed);
    if (cfg.responseFormat !== undefined) setResponseFormat(cfg.responseFormat);
    if (cfg.stream !== undefined) setStream(cfg.stream);
    if (cfg.refAudio !== undefined) setRefAudio(cfg.refAudio);
    if (cfg.refText !== undefined) setRefText(cfg.refText);
    if (cfg.seed !== undefined) setSeed(cfg.seed);
    if (cfg.maxNewTokens !== undefined) setMaxNewTokens(cfg.maxNewTokens);
    if (cfg.language !== undefined) setLanguage(cfg.language);
  }, []);

  const handleGenerate = useCallback(() => {
    if (!modelName) {
      toast.warning(t('tts.selectModelWarning', 'Please select a model'));
      return;
    }
    if (!input.trim()) {
      toast.warning(t('tts.inputRequired', 'Please enter text'));
      return;
    }

    const supportsStreamPcm = features?.supportsStreamPcm ?? false;
    const useStreamPcm = stream && (responseFormat === 'pcm' || supportsStreamPcm);

    const payload: TTSRequest = {
      model: modelName,
      input: input.trim(),
      response_format: useStreamPcm ? 'pcm' : (responseFormat === 'pcm' ? 'wav' : responseFormat),
      speed: speed !== 1 ? speed : undefined,
      stream: useStreamPcm ? true : undefined,
    };

    if (features?.supportsVoiceSelection && voice) {
      payload.voice = voice;
    }

    if (refAudio) {
      payload.ref_audio = refAudio;
      payload.ref_text = refText || undefined;
    }

    if (seed) payload.seed = parseInt(seed, 10) || undefined;
    if (maxNewTokens) payload.max_new_tokens = parseInt(maxNewTokens, 10) || undefined;
    if (language) payload.language = language;

    onGenerate(payload);
  }, [modelName, input, voice, speed, responseFormat, stream, features, refAudio, refText, seed, maxNewTokens, language, onGenerate, t]);

  if (matchedModels.length === 0) {
    return (
      <AvailableModelList
        models={availableModels}
        emptyText={t('creative.noScannedModels')}
        emptyHint={t('creative.noScannedModelsHint')}
      />
    );
  }

  const supportsVoiceSelection = features?.supportsVoiceSelection ?? true;
  const supportsRefAudio = features?.supportsRefAudio ?? false;

  return (
    <div className="space-y-4">
      {/* Model selection */}
      <div>
        <ModelSelect
          models={matchedModels}
          value={modelName}
          onValueChange={(v) => {
            onModelChange(v);
            setVoice('');
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
      </div>

      {/* Text input */}
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

      {/* Voice + Format */}
      <div className="grid grid-cols-2 gap-4">
        {supportsVoiceSelection && (
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.voiceLabel', 'Voice')}
            </label>
            <Select value={voice} onValueChange={setVoice}>
              <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                <SelectValue placeholder={voices.length > 0 ? t('tts.selectVoice', 'Select voice') : t('tts.enterVoice', 'Enter voice name')} />
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
        <div className={supportsVoiceSelection ? '' : 'col-span-2'}>
          <label className="block text-sm font-medium mb-1.5">
            {t('tts.formatLabel', 'Output Format')}
          </label>
          <Select value={responseFormat} onValueChange={setResponseFormat}>
            <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {AUDIO_FORMATS.map((f) => (
                <SelectItem key={f.value} value={f.value}>
                  {f.i18nKey ? t(f.i18nKey, f.label) : f.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Language + Streaming */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('tts.languageLabel', 'Language')}
          </label>
          <Input
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
            placeholder={t('tts.languagePlaceholder', 'e.g., zh, en, ja')}
            className="bg-background"
          />
        </div>
        <div className="flex items-center gap-3 pt-6">
          <Switch checked={stream} onCheckedChange={setStream} />
          <label className="text-sm font-medium">
            {t('tts.streaming', 'Streaming')}
          </label>
        </div>
      </div>

      {/* Speed */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.speedLabel', 'Speed')}: {speed}x
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

      {/* Reference audio (for models that support it) */}
      {supportsRefAudio && (
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('tts.refAudio', 'Reference Audio')}
            </label>
            <RefAudioInput value={refAudio} onChange={setRefAudio} />
          </div>
          {refAudio && (
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
          )}
        </div>
      )}

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
                placeholder={t('tts.seedPlaceholder', 'Leave empty for random')}
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
        disabled={isGenerating || !modelName || !input.trim()}
        className="w-full"
      >
        {isGenerating ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            {t('tts.generating', 'Generating...')}
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
    </div>
  );
}
