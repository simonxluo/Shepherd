import { useState, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Volume2, Loader2, Settings2, ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectSeparator, SelectTrigger, SelectValue } from '@/components/ui/select';
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

const VOXCPM2_LANGUAGES = [
  // Auto detect
  { value: '', group: 'auto', label: 'tts.languageAuto', fallback: 'Auto Detect' },
  // 30 official languages (alphabetical)
  { value: 'Arabic', group: 'official', label: 'Arabic' },
  { value: 'Burmese', group: 'official', label: 'Burmese' },
  { value: 'Chinese', group: 'official', label: 'Chinese' },
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
  { value: '陕西方言', group: 'dialect', label: '陕西方言 (Shaanxi)' },
  { value: '山东话', group: 'dialect', label: '山东话 (Shandong)' },
  { value: '天津话', group: 'dialect', label: '天津话 (Tianjin)' },
  { value: '闽南话', group: 'dialect', label: '闽南话 (Min Nan)' },
];

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
  const [cfgCutoffRatio, setCfgCutoffRatio] = useState('1');
  const [swaySamplingCoef, setSwaySamplingCoef] = useState('1');

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
      if (ttsConfig.cfgCutoffRatio !== undefined) setCfgCutoffRatio(ttsConfig.cfgCutoffRatio);
      if (ttsConfig.swaySamplingCoef !== undefined) setSwaySamplingCoef(ttsConfig.swaySamplingCoef);
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
    cfgCutoffRatio: cfgCutoffRatio !== '1' ? cfgCutoffRatio : undefined,
    swaySamplingCoef: swaySamplingCoef !== '1' ? swaySamplingCoef : undefined,
  }), [instructions, refAudio, refText, promptAudio, promptText, ultimateCloning,
       seed, maxNewTokens, language, emotion, cfgValue, inferenceTimesteps,
       cfgCutoffRatio, swaySamplingCoef]);

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
    if (cfg.cfgCutoffRatio !== undefined) setCfgCutoffRatio(cfg.cfgCutoffRatio);
    if (cfg.swaySamplingCoef !== undefined) setSwaySamplingCoef(cfg.swaySamplingCoef);
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

    // extra_params for cfg_cutoff_ratio and sway_sampling_coef
    const extraParams: Record<string, unknown> = {};
    if (cfgCutoffRatio) {
      const val = parseFloat(cfgCutoffRatio);
      if (!isNaN(val) && val !== 1.0) extraParams.cfg_cutoff_ratio = val;
    }
    if (swaySamplingCoef) {
      const val = parseFloat(swaySamplingCoef);
      if (!isNaN(val) && val !== 1.0) extraParams.sway_sampling_coef = val;
    }
    if (Object.keys(extraParams).length > 0) {
      payload.extra_params = extraParams;
    }

    onGenerate(payload);
  }, [modelName, input, instructions, refAudio, refText, promptAudio, promptText,
      ultimateCloning, seed, maxNewTokens, language, emotion, cfgValue, inferenceTimesteps,
      cfgCutoffRatio, swaySamplingCoef,
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
        <Select value={language} onValueChange={setLanguage}>
          <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
            <SelectValue placeholder={t('tts.languageAuto', 'Auto Detect')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">{t('tts.languageAuto', 'Auto Detect')}</SelectItem>
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

          {/* CFG Cutoff Ratio */}
          {features?.supportsCfgCutoffRatio && (
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.cfgCutoffRatio', 'CFG Cutoff Ratio')}: {cfgCutoffRatio}
              </label>
              <Slider
                value={[parseFloat(cfgCutoffRatio) || 1]}
                onValueChange={([val]) => setCfgCutoffRatio(String(val))}
                min={0}
                max={1}
                step={0.05}
                className="w-full mt-2"
              />
            </div>
          )}

          {/* Sway Sampling Coefficient */}
          {features?.supportsSwaySampling && (
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('tts.swaySamplingCoef', 'Sway Sampling Coefficient')}
              </label>
              <Input
                value={swaySamplingCoef}
                onChange={(e) => setSwaySamplingCoef(e.target.value)}
                placeholder="1.0"
                type="number"
                min="0"
                step="0.1"
                className="bg-background"
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
