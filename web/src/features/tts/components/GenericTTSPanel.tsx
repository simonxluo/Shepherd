import { useState, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Volume2, Settings2, ChevronDown, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ModelSelect } from '@/components/model/ModelSelect';
import { AvailableModelList } from '@/components/model/AvailableModelList';
import { useAvailableModels } from '@/features/creative/hooks';
import { BACKEND_LABELS } from '@/lib/constants/model';
import {
  useVoices,
  useTTSConfig,
  getTTSModelFeatures,
} from '@/features/tts/hooks';
import type { TTSRequest, TTSConfig } from '@/features/tts/types';
import { RefAudioInput } from './RefAudioInput';
import { ConfigManager } from './ConfigManager';
import { toast } from '@/hooks/useToast';
import type { TTSPluginPanelProps } from '@/features/tts/types';
import { useTTSStore } from '@/stores/ttsStore';

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
  onCancel,
  isGenerating,
  onModelChange,
  refAudioOverride,
}: TTSPluginPanelProps) {
  const { t } = useTranslation();
  const availableModels = useAvailableModels('tts');

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';
  const modelIdForConfig = selectedModel?.id || '';

  const genericForm = useTTSStore((s) => s.genericForm);
  const setGenericField = useTTSStore((s) => s.setGenericField);
  const { input, voice, speed, responseFormat, language, refAudio, refText, seed, maxNewTokens } = genericForm;

  const [stream, setStream] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [refAudioName, setRefAudioName] = useState('');

  const features = useMemo(
    () => (selectedModel ? getTTSModelFeatures(selectedModel) : null),
    [selectedModel]
  );

  const { ttsConfig, saveConfig, deleteConfig } = useTTSConfig(modelIdForConfig);
  const { data: voices = [] } = useVoices(modelName);

  const backendLabel = selectedModel?.pluginId
    ? BACKEND_LABELS[selectedModel.pluginId] || selectedModel.pluginId
    : '';

  // Restore config from server
  useEffect(() => {
    if (ttsConfig) {
      useTTSStore.getState().hydrateFromServerConfig('generic', ttsConfig);
      if (ttsConfig.stream !== undefined) setStream(ttsConfig.stream);
    } else {
      useTTSStore.getState().resetGenericForm();
    }
  }, [ttsConfig, modelIdForConfig]);

  // Sync external ref audio override from history panel
  useEffect(() => {
    if (refAudioOverride) {
      setGenericField('refAudio', refAudioOverride);
    }
  }, [refAudioOverride]);

  // Auto-adjust format on model change
  useEffect(() => {
    if (selectedModel) {
      const fmt = getTTSModelFeatures(selectedModel).defaultFormat;
      if (fmt === 'pcm') {
        useTTSStore.getState().setGenericField('responseFormat', 'pcm');
        setStream(true);
      }
    }
  }, [selectedModel]);

  const getCurrentConfig = useCallback((): TTSConfig => {
    const { voice, speed, responseFormat, refAudio, refText, seed, maxNewTokens, language } = useTTSStore.getState().genericForm;
    return {
      voice: voice || undefined,
      speed: speed !== 1 ? speed : undefined,
      responseFormat,
      stream,
      refAudio: refAudio || undefined,
      refText: refText || undefined,
      seed: seed || undefined,
      maxNewTokens: maxNewTokens || undefined,
      language: language || undefined,
    };
  }, [stream]);

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
    useTTSStore.getState().hydrateFromServerConfig('generic', cfg);
    if (cfg.stream !== undefined) setStream(cfg.stream);
  }, []);

  const handleGenerate = useCallback(() => {
    if (!modelName) {
      toast.warning(t('tts.selectModelWarning', 'Please select a model'));
      return;
    }
    const { input, voice, speed, responseFormat, refAudio, refText, seed, maxNewTokens, language } = useTTSStore.getState().genericForm;
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

    const parsedSeed = parseInt(seed, 10);
    if (seed !== '' && !Number.isNaN(parsedSeed)) payload.seed = parsedSeed;
    const parsedTokens = parseInt(maxNewTokens, 10);
    if (maxNewTokens !== '' && !Number.isNaN(parsedTokens)) payload.max_new_tokens = parsedTokens;
    if (language) payload.language = language;

    onGenerate(payload);
  }, [modelName, stream, features, onGenerate, t]);

  if (matchedModels.length === 0) {
    return (
      <div className="space-y-4">
        <div className="text-center py-6">
          <Volume2 className="w-12 h-12 mx-auto mb-3 text-muted-foreground opacity-50" />
          <h3 className="text-lg font-medium mb-1">{t('tts.noModelsTitle', 'No TTS Models Loaded')}</h3>
          <p className="text-sm text-muted-foreground mb-4">
            {t('tts.noModelsDescription', 'Load a TTS-capable model to start generating speech. Available models are shown below.')}
          </p>
        </div>
        <AvailableModelList
          models={availableModels}
          emptyText={t('creative.noScannedModels')}
          emptyHint={t('creative.noScannedModelsHint')}
        />
      </div>
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
            useTTSStore.getState().resetGenericForm();
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
          value={genericForm.input}
          onChange={(e) => setGenericField('input', e.target.value)}
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
            <Select value={genericForm.voice} onValueChange={(v) => setGenericField('voice', v)}>
              <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                <SelectValue placeholder={t('tts.selectVoice', 'Select voice')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="default">{t('tts.voiceDefault', 'Default')}</SelectItem>
                {voices.filter(v => !v.isUploaded).map((v) => (
                  <SelectItem key={v.id} value={v.id}>
                    {v.name}
                  </SelectItem>
                ))}
                {voices.some(v => v.isUploaded) && (
                  <>
                    <SelectSeparator />
                    {voices.filter(v => v.isUploaded).map(v => (
                      <SelectItem key={v.id} value={v.id}>
                        {v.name}{v.description ? ` (${v.description})` : ''}
                      </SelectItem>
                    ))}
                  </>
                )}
              </SelectContent>
            </Select>
          </div>
        )}
        <div className={supportsVoiceSelection ? '' : 'col-span-2'}>
          <label className="block text-sm font-medium mb-1.5">
            {t('tts.formatLabel', 'Output Format')}
          </label>
          <Select value={genericForm.responseFormat} onValueChange={(v) => setGenericField('responseFormat', v)}>
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
            value={genericForm.language}
            onChange={(e) => setGenericField('language', e.target.value)}
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
          {t('tts.speedLabel', 'Speed')}: {genericForm.speed}x
        </label>
        <Slider
          value={[genericForm.speed]}
          onValueChange={([val]) => setGenericField('speed', val)}
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
            <RefAudioInput value={genericForm.refAudio} onChange={(v, name) => { setGenericField('refAudio', v); setRefAudioName(name || ''); }} fileName={refAudioName} />
          </div>
          {genericForm.refAudio && (
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.refText', 'Reference Audio Transcription')}
              </label>
              <Textarea
                value={genericForm.refText}
                onChange={(e) => setGenericField('refText', e.target.value)}
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
                value={genericForm.seed}
                onChange={(e) => setGenericField('seed', e.target.value)}
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
                value={genericForm.maxNewTokens}
                onChange={(e) => setGenericField('maxNewTokens', e.target.value)}
                placeholder={t('tts.maxNewTokensPlaceholder', 'Leave empty for default')}
                type="number"
                className="bg-background"
              />
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* Generate / Cancel button */}
      {isGenerating ? (
        <Button
          onClick={onCancel}
          variant="destructive"
          className="w-full"
        >
          <X className="w-4 h-4 mr-2" />
          {t('tts.cancel', 'Cancel')}
        </Button>
      ) : (
        <Button
          onClick={handleGenerate}
          disabled={!modelName || !genericForm.input.trim()}
          className="w-full"
        >
          <Volume2 className="w-4 h-4 mr-2" />
          {t('tts.generate', 'Generate Speech')}
        </Button>
      )}

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
