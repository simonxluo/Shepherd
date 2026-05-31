import { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Music, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { AvailableModelList } from '@/features/creative/AvailableModelList';
import { useAvailableModels } from '@/features/creative/hooks';
import { toast } from '@/hooks/useToast';
import type { MusicPluginPanelProps } from '../../types';
import { AUDIO_FORMATS } from '../../constants';

export function GenericMusicPanel({
  model: selectedModel,
  matchedModels,
  onGenerate,
  isGenerating,
  onModelChange,
}: MusicPluginPanelProps) {
  const { t } = useTranslation();
  const availableModels = useAvailableModels('music');

  const genericAvailableModels = useMemo(
    () => availableModels.filter((m) => {
      const nameLower = (m.name || m.id || '').toLowerCase();
      return !nameLower.includes('ace-step') && !nameLower.includes('acestep');
    }),
    [availableModels]
  );

  const modelName = selectedModel ? (selectedModel.alias || selectedModel.name) : '';

  const [prompt, setPrompt] = useState('');
  const [duration, setDuration] = useState(30);
  const [responseFormat, setResponseFormat] = useState('wav');
  const [temperature, setTemperature] = useState(0.7);

  const handleGenerate = () => {
    if (!modelName) {
      toast.warning(t('musicGen.selectModelWarning', '请选择模型'));
      return;
    }
    if (!prompt.trim()) {
      toast.warning(t('musicGen.promptRequired', '请输入音乐描述'));
      return;
    }

    onGenerate({
      model: modelName,
      prompt: prompt.trim(),
      duration,
      response_format: responseFormat,
      temperature,
    });
  };

  if (matchedModels.length === 0) {
    return (
      <div className="space-y-4">
        <div className="flex flex-col items-center gap-3 py-6 px-4 rounded-lg border border-dashed border-muted-foreground/30">
          <AlertCircle className="w-8 h-8 text-muted-foreground" />
          <div className="text-center">
            <p className="text-sm font-medium text-foreground">
              {t('musicGen.noModelsLoaded', '未检测到已加载的音乐生成模型')}
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              {t('musicGen.noModelsLoadedHint', '请从下方列表中选择一个模型进行加载')}
            </p>
          </div>
        </div>
        <AvailableModelList
          models={genericAvailableModels}
          emptyText={t('creative.noScannedModels')}
          emptyHint={t('creative.noScannedModelsHint')}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <ModelSelect
        models={matchedModels}
        value={modelName}
        onValueChange={onModelChange}
        placeholder={t('musicGen.selectModel', '选择音乐生成模型')}
        label={t('musicGen.modelLabel', '音乐生成模型')}
        showBackend
      />

      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('musicGen.promptLabel', '音乐描述')}
        </label>
        <Textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          placeholder={t('musicGen.promptPlaceholder', '描述要生成的音乐风格、情绪、乐器等...')}
          className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
          rows={3}
        />
      </div>

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
      </div>

      <div>
        <label className="block text-sm font-medium mb-1.5">
          {t('musicGen.temperatureLabel', '温度')}: {temperature.toFixed(1)}
        </label>
        <Slider
          value={[temperature]}
          onValueChange={([val]) => setTemperature(val)}
          min={0}
          max={1}
          step={0.1}
          className="w-full mt-2"
        />
      </div>

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
