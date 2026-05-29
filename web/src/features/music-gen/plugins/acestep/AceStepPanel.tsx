import { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Music, Loader2, AlertCircle, Settings2, ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { AvailableModelList } from '@/features/creative/AvailableModelList';
import { useAvailableModels } from '@/features/creative/hooks';
import { toast } from '@/hooks/useToast';
import type { MusicPluginPanelProps, MusicGenRequest } from '../../types';

const AUDIO_FORMATS = [
  { value: 'wav', label: 'WAV' },
  { value: 'mp3', label: 'MP3' },
];

const VOCAL_LANGUAGES = [
  { value: 'en', label: 'English' },
  { value: 'zh', label: '中文' },
  { value: 'ja', label: '日本語' },
  { value: 'ko', label: '한국어' },
  { value: 'es', label: 'Español' },
  { value: 'fr', label: 'Français' },
  { value: 'de', label: 'Deutsch' },
  { value: 'instrumental', label: 'Instrumental (No Vocals)' },
];

const KEY_SCALES = [
  { value: '', label: 'Auto' },
  { value: 'C major', label: 'C major' },
  { value: 'C minor', label: 'C minor' },
  { value: 'D major', label: 'D major' },
  { value: 'D minor', label: 'D minor' },
  { value: 'E major', label: 'E major' },
  { value: 'E minor', label: 'E minor' },
  { value: 'F major', label: 'F major' },
  { value: 'F minor', label: 'F minor' },
  { value: 'G major', label: 'G major' },
  { value: 'G minor', label: 'G minor' },
  { value: 'A major', label: 'A major' },
  { value: 'A minor', label: 'A minor' },
  { value: 'B major', label: 'B major' },
  { value: 'B minor', label: 'B minor' },
];

const TIME_SIGNATURES = [
  { value: '', label: 'Auto' },
  { value: '4/4', label: '4/4' },
  { value: '3/4', label: '3/4' },
  { value: '6/8', label: '6/8' },
  { value: '2/4', label: '2/4' },
  { value: '5/4', label: '5/4' },
  { value: '7/8', label: '7/8' },
];

export function AceStepPanel({
  model: selectedModel,
  matchedModels,
  onGenerate,
  isGenerating,
  onModelChange,
}: MusicPluginPanelProps) {
  const { t } = useTranslation();
  const availableModels = useAvailableModels('music');

  const aceStepAvailableModels = useMemo(
    () => availableModels.filter((m) => {
      const nameLower = (m.name || m.id || '').toLowerCase();
      return nameLower.includes('ace-step') || nameLower.includes('acestep');
    }),
    [availableModels]
  );

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';

  // Basic fields
  const [prompt, setPrompt] = useState('');
  const [lyrics, setLyrics] = useState('');
  const [duration, setDuration] = useState(30);
  const [responseFormat, setResponseFormat] = useState('wav');
  const [vocalLanguage, setVocalLanguage] = useState('en');

  // Advanced fields
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [bpm, setBpm] = useState('');
  const [keyScale, setKeyScale] = useState('');
  const [timeSignature, setTimeSignature] = useState('');
  const [inferenceSteps, setInferenceSteps] = useState(8);
  const [guidanceScale, setGuidanceScale] = useState(7.0);
  const [seed, setSeed] = useState('');

  const handleGenerate = () => {
    if (!modelName) {
      toast.warning(t('musicGen.selectModelWarning', '请选择模型'));
      return;
    }
    if (!prompt.trim()) {
      toast.warning(t('musicGen.promptRequired', '请输入音乐描述'));
      return;
    }

    const payload: MusicGenRequest = {
      model: modelName,
      prompt: prompt.trim(),
      duration,
      response_format: responseFormat,
      vocal_language: vocalLanguage,
      inference_steps: inferenceSteps,
      guidance_scale: guidanceScale,
      task_type: 'text2music',
    };

    if (lyrics.trim()) {
      payload.lyrics = lyrics.trim();
    }
    if (bpm) {
      payload.bpm = parseInt(bpm, 10) || undefined;
    }
    if (keyScale) {
      payload.key_scale = keyScale;
    }
    if (timeSignature) {
      payload.time_signature = timeSignature;
    }
    if (seed) {
      payload.seed = parseInt(seed, 10) || undefined;
    }

    onGenerate(payload);
  };

  if (matchedModels.length === 0) {
    return (
      <div className="space-y-4">
        <div className="flex flex-col items-center gap-3 py-6 px-4 rounded-lg border border-dashed border-orange-300 bg-orange-50/50 dark:border-orange-700 dark:bg-orange-950/20">
          <AlertCircle className="w-8 h-8 text-orange-500" />
          <div className="text-center">
            <p className="text-sm font-medium text-foreground">
              {t('musicGen.aceStep.notDetected', '未检测到已加载的 ACE-Step 模型')}
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              {t('musicGen.aceStep.notDetectedHint', '请从下方列表中选择一个 ACE-Step 模型进行加载')}
            </p>
          </div>
        </div>
        <AvailableModelList
          models={aceStepAvailableModels}
          emptyText={t('musicGen.aceStep.noModels', '未扫描到 ACE-Step 模型')}
          emptyHint={t('musicGen.aceStep.noModelsHint', '请确认已配置 ACE-Step 模型路径并完成扫描')}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Model selection */}
      <ModelSelect
        models={matchedModels}
        value={modelName}
        onValueChange={onModelChange}
        placeholder={t('musicGen.aceStep.selectModel', '选择 ACE-Step 模型')}
        label={t('musicGen.aceStep.modelLabel', 'ACE-Step 模型')}
        showBackend
      />

      {/* Music description / caption */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('musicGen.aceStep.promptLabel', '音乐描述 (Caption)')}
        </label>
        <Textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder={t('musicGen.aceStep.promptPlaceholder', '描述音乐的风格、情绪、乐器编排等，例如: A dreamy indie folk song with acoustic guitar and soft vocals...')}
          className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
          rows={3}
        />
      </div>

      {/* Lyrics */}
      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('musicGen.aceStep.lyricsLabel', '歌词 (Lyrics)')}
        </label>
        <Textarea
          value={lyrics}
          onChange={(e) => setLyrics(e.target.value)}
          placeholder={t('musicGen.aceStep.lyricsPlaceholder', '输入歌词，留空则生成纯音乐...')}
          className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
          rows={4}
        />
      </div>

      {/* Duration and Language */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('musicGen.durationLabel', '时长')}: {duration}s
          </label>
          <Slider
            value={[duration]}
            onValueChange={([val]) => setDuration(val)}
            min={5}
            max={300}
            step={5}
            className="w-full mt-2"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('musicGen.aceStep.vocalLanguage', '演唱语言')}
          </label>
          <Select value={vocalLanguage} onValueChange={setVocalLanguage}>
            <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {VOCAL_LANGUAGES.map((l) => (
                <SelectItem key={l.value} value={l.value}>{l.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Output format */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('musicGen.formatLabel', '输出格式')}
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
        <div>
          <label className="block text-sm font-medium mb-1.5">
            {t('musicGen.aceStep.inferenceSteps', '推理步数')}: {inferenceSteps}
          </label>
          <Slider
            value={[inferenceSteps]}
            onValueChange={([val]) => setInferenceSteps(val)}
            min={1}
            max={50}
            step={1}
            className="w-full mt-2"
          />
        </div>
      </div>

      {/* Advanced settings */}
      <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
        <CollapsibleTrigger asChild>
          <Button variant="ghost" className="w-full justify-between p-0 h-auto text-sm font-medium hover:bg-transparent">
            <span className="flex items-center gap-2">
              <Settings2 className="w-4 h-4" />
              {t('musicGen.aceStep.advanced', '高级设置')}
            </span>
            <ChevronDown className={`w-4 h-4 transition-transform ${advancedOpen ? 'rotate-180' : ''}`} />
          </Button>
        </CollapsibleTrigger>
        <CollapsibleContent className="space-y-4 pt-2">
          {/* Guidance scale */}
          <div>
            <label className="block text-sm font-medium mb-1.5">
              {t('musicGen.aceStep.guidanceScale', 'Guidance Scale')}: {guidanceScale.toFixed(1)}
            </label>
            <Slider
              value={[guidanceScale]}
              onValueChange={([val]) => setGuidanceScale(val)}
              min={1}
              max={15}
              step={0.5}
              className="w-full mt-2"
            />
          </div>

          {/* BPM + Seed */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('musicGen.aceStep.bpm', 'BPM')}
              </label>
              <Input
                value={bpm}
                onChange={(e) => setBpm(e.target.value)}
                placeholder={t('musicGen.aceStep.bpmPlaceholder', '留空自动检测')}
                type="number"
                min={40}
                max={240}
                className="bg-background"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('musicGen.aceStep.seed', 'Seed')}
              </label>
              <Input
                value={seed}
                onChange={(e) => setSeed(e.target.value)}
                placeholder={t('musicGen.aceStep.seedPlaceholder', '留空随机')}
                type="number"
                className="bg-background"
              />
            </div>
          </div>

          {/* Key scale + Time signature */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('musicGen.aceStep.keyScale', '调性')}
              </label>
              <Select value={keyScale} onValueChange={setKeyScale}>
                <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                  <SelectValue placeholder="Auto" />
                </SelectTrigger>
                <SelectContent>
                  {KEY_SCALES.map((k) => (
                    <SelectItem key={k.value || '__auto'} value={k.value || ' '}>{k.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('musicGen.aceStep.timeSignature', '拍号')}
              </label>
              <Select value={timeSignature} onValueChange={setTimeSignature}>
                <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                  <SelectValue placeholder="Auto" />
                </SelectTrigger>
                <SelectContent>
                  {TIME_SIGNATURES.map((ts) => (
                    <SelectItem key={ts.value || '__auto'} value={ts.value || ' '}>{ts.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>

      {/* Generate button */}
      <Button
        onClick={handleGenerate}
        disabled={isGenerating || !modelName || !prompt.trim()}
        className="w-full"
      >
        {isGenerating ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            {t('musicGen.generating', '生成中...')}
          </>
        ) : (
          <>
            <Music className="w-4 h-4 mr-2" />
            {t('musicGen.generate', '生成音乐')}
          </>
        )}
      </Button>
    </div>
  );
}
