import { useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Volume2, Loader2, Play, Pause, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Switch } from '@/components/ui/switch';
import { useLoadedModels, BACKEND_LABELS } from '@/features/creative/hooks';
import { ModelSelect } from '@/features/creative/ModelSelect';
import { useTTS, useVoices } from '@/features/tts/hooks';
import { toast } from '@/hooks/useToast';

const FALLBACK_VOICES = ['alloy', 'echo', 'fable', 'onyx', 'nova', 'shimmer'];

const AUDIO_FORMATS = [
  { value: 'mp3', label: 'MP3' },
  { value: 'wav', label: 'WAV' },
  { value: 'opus', label: 'Opus' },
  { value: 'flac', label: 'FLAC' },
];

export function TTSPage() {
  const { t } = useTranslation();
  const { data: allModels = [] } = useLoadedModels();
  const ttsModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.tts),
    [allModels]
  );

  const tts = useTTS();
  const audioRef = useRef<HTMLAudioElement>(null);

  const [model, setModel] = useState('');
  const [input, setInput] = useState('');
  const [voice, setVoice] = useState('');
  const [speed, setSpeed] = useState(1);
  const [responseFormat, setResponseFormat] = useState('mp3');
  const [stream, setStream] = useState(false);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);

  const { data: voices = [] } = useVoices(model);

  const selectedModel = ttsModels.find((m) => (m.alias || m.name) === model);
  const backendLabel = selectedModel?.backendType
    ? BACKEND_LABELS[selectedModel.backendType] || selectedModel.backendType
    : '';

  const handleGenerate = () => {
    if (!model) {
      toast.warning(t('tts.selectModelWarning', '请选择模型'));
      return;
    }
    if (!input.trim()) {
      toast.warning(t('tts.inputRequired', '请输入文本'));
      return;
    }

    if (audioUrl) {
      URL.revokeObjectURL(audioUrl);
    }

    tts.mutate(
      {
        model,
        input: input.trim(),
        voice: voice || undefined,
        response_format: responseFormat,
        speed: speed !== 1 ? speed : undefined,
        stream: stream ? true : undefined,
      },
      {
        onSuccess: ({ blob, contentType }) => {
          const typedBlob = new Blob([blob], { type: contentType });
          const url = URL.createObjectURL(typedBlob);
          setAudioUrl(url);
          toast.success(t('tts.generateSuccess', '语音合成完成'));
        },
        onError: (error) => {
          toast.error(t('tts.generateFailed', '语音合成失败'), error.message);
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
    const ext = responseFormat || 'mp3';
    const a = document.createElement('a');
    a.href = audioUrl;
    a.download = `tts_${Date.now()}.${ext}`;
    a.click();
  };

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl mx-auto">
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-foreground">
              {t('tts.title', '语音合成 (TTS)')}
            </h1>
          </div>

          {ttsModels.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <p>{t('tts.noModels', '没有已加载的 TTS 模型')}</p>
              <p className="text-sm mt-1">{t('tts.noModelsHint', '请先加载一个支持语音合成的模型')}</p>
            </div>
          ) : (
            <div className="space-y-6">
              <div className="space-y-4">
                <div>
                  <ModelSelect
                    models={ttsModels}
                    value={model}
                    onValueChange={(v) => { setModel(v); setVoice(''); }}
                    placeholder={t('tts.selectModel', '选择 TTS 模型')}
                    label={t('tts.modelLabel', 'TTS 模型')}
                    showBackend
                  />
                  {backendLabel && (
                    <p className="text-xs text-muted-foreground mt-1">
                      {t('tts.backend', '后端')}: {backendLabel}
                    </p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1.5">
                    {t('tts.inputLabel', '输入文本')}
                  </label>
                  <Textarea
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    placeholder={t('tts.inputPlaceholder', '输入要转换为语音的文本...')}
                    className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
                    rows={4}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('tts.voiceLabel', '语音 (Voice)')}
                    </label>
                    <Select value={voice} onValueChange={setVoice}>
                      <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                        <SelectValue placeholder={voices.length > 0 ? t('tts.selectVoice', '选择语音') : t('tts.enterVoice', '输入语音名称 (如: alloy)')} />
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
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('tts.formatLabel', '输出格式')}
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

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium mb-1.5">
                      {t('tts.speedLabel', '语速')}: {speed}x
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
                  <div className="flex items-center gap-3 pt-6">
                    <Switch checked={stream} onCheckedChange={setStream} />
                    <label className="text-sm font-medium">
                      {t('tts.streaming', '流式生成')}
                    </label>
                  </div>
                </div>

                <Button
                  onClick={handleGenerate}
                  disabled={tts.isPending || !model || !input.trim()}
                  className="w-full"
                >
                  {tts.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                      {t('tts.generating', '生成中...')}
                    </>
                  ) : (
                    <>
                      <Volume2 className="w-4 h-4 mr-2" />
                      {t('tts.generate', '生成语音')}
                    </>
                  )}
                </Button>
              </div>

              {audioUrl && (
                <div className="border rounded-lg p-4 space-y-3">
                  <h3 className="text-sm font-medium">{t('tts.result', '生成结果')}</h3>
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
                    <Button variant="outline" size="icon" onClick={handleDownload} title={t('tts.download', '下载音频')}>
                      <Download className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
