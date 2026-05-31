import { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Music, Loader2, AlertCircle, Settings2, ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ModelSelect } from '@/components/model/ModelSelect';
import { AvailableModelList } from '@/components/model/AvailableModelList';
import { useAvailableModels } from '@/features/creative/hooks';
import { useMusicGenStore } from '@/stores/musicGenStore';
import { toast } from '@/hooks/useToast';
import type { MusicPluginPanelProps, MusicGenRequest } from '@/features/music-gen/types';
import { AUDIO_FORMATS } from '@/features/music-gen/constants';

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
  { value: '__auto', label: 'Auto' },
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
  { value: '__auto', label: 'Auto' },
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

  // 表单状态从 Zustand store 获取（跨页面持久化）
  const aceStepForm = useMusicGenStore((s) => s.aceStepForm);
  const setAceStepField = useMusicGenStore((s) => s.setAceStepField);
  const { prompt, lyrics, duration, responseFormat, vocalLanguage,
          bpm, keyScale, timeSignature, inferenceSteps, guidanceScale, seed } = aceStepForm;

  // 瞬态 UI 状态保留为本地 useState
  const [advancedOpen, setAdvancedOpen] = useState(false);

  const handleGenerate = () => {
    const form = useMusicGenStore.getState().aceStepForm;
    if (!modelName) {
      toast.warning(t('musicGen.selectModelWarning', '请选择模型'));
      return;
    }
    if (!form.prompt.trim()) {
      toast.warning(t('musicGen.promptRequired', '请输入音乐描述'));
      return;
    }

    const payload: MusicGenRequest = {
      model: modelName,
      prompt: form.prompt.trim(),
      duration: form.duration,
      response_format: form.responseFormat,
      vocal_language: form.vocalLanguage,
      inference_steps: form.inferenceSteps,
      guidance_scale: form.guidanceScale,
      task_type: 'text2music',
    };

    if (form.lyrics.trim()) {
      payload.lyrics = form.lyrics.trim();
    }
    if (form.bpm) {
      payload.bpm = parseInt(form.bpm, 10) || undefined;
    }
    if (form.keyScale && form.keyScale !== '__auto') {
      payload.key_scale = form.keyScale;
    }
    if (form.timeSignature && form.timeSignature !== '__auto') {
      payload.time_signature = form.timeSignature;
    }
    if (form.seed) {
      payload.seed = parseInt(form.seed, 10) || undefined;
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
          onChange={(e) => setAceStepField('prompt', e.target.value)}
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
          onChange={(e) => setAceStepField('lyrics', e.target.value)}
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
            onValueChange={([val]) => setAceStepField('duration', val)}
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
          <Select value={vocalLanguage} onValueChange={(v) => setAceStepField('vocalLanguage', v)}>
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
          <Select value={responseFormat} onValueChange={(v) => setAceStepField('responseFormat', v)}>
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
            onValueChange={([val]) => setAceStepField('inferenceSteps', val)}
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
              onValueChange={([val]) => setAceStepField('guidanceScale', val)}
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
                onChange={(e) => setAceStepField('bpm', e.target.value)}
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
                onChange={(e) => setAceStepField('seed', e.target.value)}
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
              <Select value={keyScale} onValueChange={(v) => setAceStepField('keyScale', v)}>
                <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                  <SelectValue placeholder="Auto" />
                </SelectTrigger>
                <SelectContent>
                  {KEY_SCALES.map((k) => (
                    <SelectItem key={k.value} value={k.value}>{k.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1.5">
                {t('musicGen.aceStep.timeSignature', '拍号')}
              </label>
              <Select value={timeSignature} onValueChange={(v) => setAceStepField('timeSignature', v)}>
                <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                  <SelectValue placeholder="Auto" />
                </SelectTrigger>
                <SelectContent>
                  {TIME_SIGNATURES.map((ts) => (
                    <SelectItem key={ts.value} value={ts.value}>{ts.label}</SelectItem>
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
        disabled={isGenerating || !modelName || !aceStepForm.prompt.trim()}
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
