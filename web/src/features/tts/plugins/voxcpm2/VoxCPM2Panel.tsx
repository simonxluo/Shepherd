import { useState, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Volume2, Loader2, Settings2, ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { AvailableModelList } from '@/features/creative/AvailableModelList';
import { useAvailableModels, BACKEND_LABELS } from '@/features/creative/hooks';
import {
  useTTSConfig,
  getTTSModelFeatures,
  type TTSRequest,
  type TTSConfig,
} from '../../hooks';
import { RefAudioInput } from '../../components/RefAudioInput';
import { ConfigManager } from '../../components/ConfigManager';
import { toast } from '@/hooks/useToast';
import type { TTSPluginPanelProps } from '../../types';

export function VoxCPM2Panel({
  model: selectedModel,
  matchedModels,
  onGenerate,
  isGenerating,
  streamState,
  onModelChange,
  refAudioOverride,
}: TTSPluginPanelProps) {
  const { t } = useTranslation();
  const availableModels = useAvailableModels('tts');

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';
  const modelIdForConfig = selectedModel?.id || '';

  const [input, setInput] = useState('');
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
  const [language, setLanguage] = useState('');
  const [emotion, setEmotion] = useState('default');
  const [cfgValue, setCfgValue] = useState('2');
  const [inferenceTimesteps, setInferenceTimesteps] = useState('10');

  const features = useMemo(
    () => (selectedModel ? getTTSModelFeatures(selectedModel) : null),
    [selectedModel]
  );

  const { ttsConfig, saveConfig, deleteConfig } = useTTSConfig(modelIdForConfig);

  const backendLabel = selectedModel?.backendType
    ? BACKEND_LABELS[selectedModel.backendType] || selectedModel.backendType
    : '';

  const isStreamActive = streamState === 'streaming' || streamState === 'playing';

  // Restore config from server
  useEffect(() => {
    if (ttsConfig) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      if (ttsConfig.instructions !== undefined) setInstructions(ttsConfig.instructions);
      if (ttsConfig.refAudio !== undefined) setRefAudio(ttsConfig.refAudio);
      if (ttsConfig.refText !== undefined) setRefText(ttsConfig.refText);
      if (ttsConfig.promptAudio !== undefined) setPromptAudio(ttsConfig.promptAudio);
      if (ttsConfig.promptText !== undefined) setPromptText(ttsConfig.promptText);
      if (ttsConfig.ultimateCloning !== undefined) setUltimateCloning(ttsConfig.ultimateCloning);
      if (ttsConfig.seed !== undefined) setSeed(ttsConfig.seed);
      if (ttsConfig.maxNewTokens !== undefined) setMaxNewTokens(ttsConfig.maxNewTokens);
      if (ttsConfig.language !== undefined) setLanguage(ttsConfig.language);
      if (ttsConfig.emotion !== undefined) setEmotion(ttsConfig.emotion);
      if (ttsConfig.cfgValue !== undefined) setCfgValue(ttsConfig.cfgValue);
      if (ttsConfig.inferenceTimesteps !== undefined) setInferenceTimesteps(ttsConfig.inferenceTimesteps);
    }
  }, [ttsConfig]);

  // Sync external ref audio override from history panel
  useEffect(() => {
    if (refAudioOverride) {
      setRefAudio(refAudioOverride);
    }
  }, [refAudioOverride]);

  const getCurrentConfig = useCallback((): TTSConfig => ({
    stream: true,
    responseFormat: 'pcm',
    instructions: instructions || undefined,
    refAudio: refAudio || undefined,
    refText: refText || undefined,
    promptAudio: promptAudio || undefined,
    promptText: promptText || undefined,
    ultimateCloning: ultimateCloning || undefined,
    seed: seed || undefined,
    maxNewTokens: maxNewTokens || undefined,
    language: language || undefined,
    emotion: emotion === 'default' ? undefined : emotion,
    cfgValue: cfgValue || undefined,
    inferenceTimesteps: inferenceTimesteps || undefined,
  }), [instructions, refAudio, refText, promptAudio, promptText, ultimateCloning,
       seed, maxNewTokens, language, emotion, cfgValue, inferenceTimesteps]);

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
    if (cfg.instructions !== undefined) setInstructions(cfg.instructions);
    if (cfg.refAudio !== undefined) setRefAudio(cfg.refAudio);
    if (cfg.refText !== undefined) setRefText(cfg.refText);
    if (cfg.promptAudio !== undefined) setPromptAudio(cfg.promptAudio);
    if (cfg.promptText !== undefined) setPromptText(cfg.promptText);
    if (cfg.ultimateCloning !== undefined) setUltimateCloning(cfg.ultimateCloning);
    if (cfg.seed !== undefined) setSeed(cfg.seed);
    if (cfg.maxNewTokens !== undefined) setMaxNewTokens(cfg.maxNewTokens);
    if (cfg.language !== undefined) setLanguage(cfg.language);
    if (cfg.emotion !== undefined) setEmotion(cfg.emotion);
    if (cfg.cfgValue !== undefined) setCfgValue(cfg.cfgValue);
    if (cfg.inferenceTimesteps !== undefined) setInferenceTimesteps(cfg.inferenceTimesteps);
  }, []);

  const handleGenerate = useCallback(() => {
    if (!modelName) {
      toast.warning(t('tts.selectModelWarning', 'Please select a model'));
      return;
    }
    if (!input.trim() && !ultimateCloning) {
      toast.warning(t('tts.inputRequired', 'Please enter text'));
      return;
    }
    if (ultimateCloning && !promptAudio) {
      toast.warning(t('tts.inputRequired', 'Missing audio'));
      return;
    }

    const payload: TTSRequest = {
      model: modelName,
      input: ultimateCloning ? (promptText || '') : input.trim(),
      response_format: 'pcm',
      stream: true,
    };

    if (instructions.trim()) {
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
    if (language) payload.language = language;
    if (emotion && emotion !== 'default') payload.emotion = emotion;

    if (cfgValue) {
      const val = parseFloat(cfgValue);
      if (!isNaN(val)) payload.cfg_value = val;
    }
    if (inferenceTimesteps) {
      const val = parseInt(inferenceTimesteps, 10);
      if (!isNaN(val)) payload.inference_timesteps = val;
    }

    onGenerate(payload);
  }, [modelName, input, instructions, refAudio, refText, promptAudio, promptText,
      ultimateCloning, seed, maxNewTokens, language, emotion, cfgValue, inferenceTimesteps,
      onGenerate, t]);

  if (matchedModels.length === 0) {
    return (
      <AvailableModelList
        models={availableModels}
        emptyText={t('creative.noScannedModels')}
        emptyHint={t('creative.noScannedModelsHint')}
      />
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
            setUltimateCloning(false);
            setRefAudio('');
            setPromptAudio('');
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

      {/* Text input (hidden in ultimate cloning mode) */}
      {!ultimateCloning && (
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
      )}

      {/* Style instructions */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('tts.instructions', 'Style Instructions')}
        </label>
        <Input
          value={instructions}
          onChange={(e) => setInstructions(e.target.value)}
          placeholder={t('tts.instructionsPlaceholder', 'Enter style instructions, e.g.: Read in a gentle tone...')}
          className="bg-background"
        />
      </div>

      {/* Language */}
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

      {/* Voice cloning */}
      <Collapsible open={cloningOpen} onOpenChange={setCloningOpen}>
        <CollapsibleTrigger asChild>
          <Button variant="ghost" className="w-full justify-between p-0 h-auto text-sm font-medium hover:bg-transparent">
            {t('tts.voiceCloning', 'Voice Cloning')}
            <ChevronDown className={`w-4 h-4 transition-transform ${cloningOpen ? 'rotate-180' : ''}`} />
          </Button>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-2">
          {features?.supportsUltimateCloning && (
            <div className="flex items-center gap-3">
              <Switch checked={ultimateCloning} onCheckedChange={setUltimateCloning} />
              <div>
                <Label className="text-sm font-medium">
                  {t('tts.ultimateCloning', 'Ultimate Cloning Mode')}
                </Label>
                <p className="text-xs text-muted-foreground">
                  {t('tts.ultimateCloningDesc', 'Use prompt_audio + prompt_text for precise voice cloning')}
                </p>
              </div>
            </div>
          )}

          {ultimateCloning ? (
            <>
              <div>
                <label className="block text-sm font-medium mb-1.5">
                  {t('tts.refAudio', 'Cloning Audio')}
                </label>
                <RefAudioInput value={promptAudio} onChange={setPromptAudio} />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1.5">
                  {t('tts.refText', 'Audio Transcription')}
                </label>
                <Textarea
                  value={promptText}
                  onChange={(e) => setPromptText(e.target.value)}
                  placeholder={t('tts.refTextPlaceholder', 'Enter transcription of the audio')}
                  rows={2}
                  className="bg-background"
                />
              </div>
            </>
          ) : (
            <>
              <div>
                <label className="block text-sm font-medium mb-1.5">
                  {t('tts.refAudio', 'Reference Audio')}
                </label>
                <RefAudioInput value={refAudio} onChange={setRefAudio} />
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
            </>
          )}
        </CollapsibleContent>
      </Collapsible>

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
          {/* Emotion */}
          {features?.supportsEmotion && (
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.emotionLabel', 'Emotion')}
              </label>
              <Select value={emotion} onValueChange={setEmotion}>
                <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                  <SelectValue placeholder={t('tts.emotionDefault', 'Default')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="default">{t('tts.emotionDefault', 'Default')}</SelectItem>
                  <SelectItem value="happy">{t('tts.emotionHappy', 'Happy')}</SelectItem>
                  <SelectItem value="sad">{t('tts.emotionSad', 'Sad')}</SelectItem>
                  <SelectItem value="angry">{t('tts.emotionAngry', 'Angry')}</SelectItem>
                  <SelectItem value="gentle">{t('tts.emotionGentle', 'Gentle')}</SelectItem>
                  <SelectItem value="surprised">{t('tts.emotionSurprised', 'Surprised')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          {/* Seed + Max tokens */}
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

          {/* CFG guidance */}
          {features?.supportsCfgValue && (
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.cfgLabel', 'CFG Guidance')}: {cfgValue}
              </label>
              <Slider
                value={[parseFloat(cfgValue) || 2]}
                onValueChange={([val]) => setCfgValue(String(val))}
                min={1}
                max={5}
                step={0.5}
                className="w-full mt-2"
              />
            </div>
          )}

          {/* Diffusion steps */}
          {features?.supportsInferenceTimesteps && (
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.stepsLabel', 'Diffusion Steps')}: {inferenceTimesteps}
              </label>
              <Slider
                value={[parseInt(inferenceTimesteps) || 10]}
                onValueChange={([val]) => setInferenceTimesteps(String(val))}
                min={4}
                max={30}
                step={1}
                className="w-full mt-2"
              />
            </div>
          )}
        </CollapsibleContent>
      </Collapsible>

      {/* Generate button */}
      <Button
        onClick={handleGenerate}
        disabled={isGenerating || !modelName || (!input.trim() && !ultimateCloning)}
        className="w-full"
      >
        {isGenerating ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            {isStreamActive ? t('tts.streamingInProgress', 'Streaming...') : t('tts.generating', 'Generating...')}
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
