import { useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Music, Loader2, Play, Pause, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { useLoadedModels } from '@/features/creative/hooks';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { useMusicGeneration } from '@/features/music-gen/hooks';
import { toast } from '@/hooks/useToast';

const AUDIO_FORMATS = [
  { value: 'wav', label: 'WAV' },
  { value: 'mp3', label: 'MP3' },
];

export function MusicGenPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data: allModels = [] } = useLoadedModels();
  const musicModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.music),
    [allModels]
  );

  const musicGen = useMusicGeneration();
  const audioRef = useRef<HTMLAudioElement>(null);

  const [model, setModel] = useState('');
  const [prompt, setPrompt] = useState('');
  const [duration, setDuration] = useState(30);
  const [responseFormat, setResponseFormat] = useState('wav');
  const [temperature, setTemperature] = useState(0.7);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);

  const handleGenerate = () => {
    if (!model) {
      toast.warning(t('musicGen.selectModelWarning', '请选择模型'));
      return;
    }
    if (!prompt.trim()) {
      toast.warning(t('musicGen.promptRequired', '请输入音乐描述'));
      return;
    }

    if (audioUrl) {
      URL.revokeObjectURL(audioUrl);
    }

    musicGen.mutate(
      {
        model,
        prompt: prompt.trim(),
        duration,
        response_format: responseFormat,
        temperature,
      },
      {
        onSuccess: ({ blob, contentType }) => {
          const typedBlob = new Blob([blob], { type: contentType });
          const url = URL.createObjectURL(typedBlob);
          setAudioUrl(url);
          toast.success(t('musicGen.generateSuccess', '音乐生成完成'));
        },
        onError: (error) => {
          toast.error(t('musicGen.generateFailed', '音乐生成失败'), error.message);
        },
      }
    );
  };

  const handlePlayPause = () => {
    if (!audioRef.current) return;
    if (isPlaying) {
      audioRef.current.pause();
    } else {
      audioRef.current.play();
    }
    setIsPlaying(!isPlaying);
  };

  const handleDownload = () => {
    if (!audioUrl) return;
    const ext = responseFormat || 'wav';
    const a = document.createElement('a');
    a.href = audioUrl;
    a.download = `music_${Date.now()}.${ext}`;
    a.click();
  };

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          {t('musicGen.title', '音乐生成')}
        </h1>
        <p className="text-muted-foreground">
          {t('musicGen.description', '通过文本描述生成音乐')}
        </p>
      </div>

      {musicModels.length > 0 ? (
        <ModelSelect
          models={musicModels}
          value={model}
          onValueChange={setModel}
          placeholder={t('musicGen.selectModel', '选择音乐生成模型')}
          label={t('musicGen.modelLabel', '音乐生成模型')}
          showBackend
        />
      ) : (
        <div className="flex flex-col items-center rounded-lg border border-dashed p-8 text-center">
          <Music className="h-8 w-8 text-muted-foreground mb-2" />
          <p className="text-sm font-medium">{t('musicGen.noModels', '没有已加载的音乐生成模型')}</p>
          <p className="text-xs text-muted-foreground mt-1">{t('musicGen.noModelsHint', '请先加载一个支持音乐生成的模型')}</p>
          <Button
            variant="outline"
            size="sm"
            className="mt-3"
            onClick={() => navigate('/models')}
          >
            {t('musicGen.goToModels', '前往模型管理')}
          </Button>
        </div>
      )}

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
        disabled={musicGen.isPending || !model || !prompt.trim()}
        className="w-full"
      >
        {musicGen.isPending ? (
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

      {audioUrl && (
        <div className="border rounded-lg p-4 space-y-3">
          <h3 className="text-sm font-medium">{t('musicGen.result', '生成结果')}</h3>
          <audio
            ref={audioRef}
            src={audioUrl}
            onEnded={() => setIsPlaying(false)}
            onPause={() => setIsPlaying(false)}
            onPlay={() => setIsPlaying(true)}
          />
          <div className="flex items-center gap-2">
            <Button variant="outline" size="icon" onClick={handlePlayPause}>
              {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
            </Button>
            <Button variant="outline" size="icon" onClick={handleDownload} title={t('musicGen.download', '下载音频')}>
              <Download className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
