import { useState, useRef } from 'react';
import { Volume2, Loader2, Play, Pause, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Progress } from '@/components/ui/progress';
import { Switch } from '@/components/ui/switch';
import { useTTS, useVoices, BACKEND_LABELS } from '../hooks';
import { useToast } from '@/hooks/useToast';

interface TTSPanelProps {
  models: Array<{ id: string; name: string; alias?: string; backendType?: string }>;
}

const FALLBACK_VOICES = ['alloy', 'echo', 'fable', 'onyx', 'nova', 'shimmer'];

const AUDIO_FORMATS = [
  { value: 'mp3', label: 'MP3' },
  { value: 'wav', label: 'WAV' },
  { value: 'opus', label: 'Opus' },
  { value: 'flac', label: 'FLAC' },
];

export function TTSPanel({ models }: TTSPanelProps) {
  const toast = useToast();
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
  const [audioBlob, setAudioBlob] = useState<Blob | null>(null);

  const { data: voices = [] } = useVoices(model);

  const selectedModel = models.find((m) => (m.alias || m.name) === model);
  const backendLabel = selectedModel?.backendType
    ? BACKEND_LABELS[selectedModel.backendType] || selectedModel.backendType
    : '';

  const handleGenerate = () => {
    if (!model) {
      toast.warning('请选择模型', '请从下拉列表中选择一个支持 TTS 的模型');
      return;
    }
    if (!input.trim()) {
      toast.warning('请输入文本', '请输入要转换为语音的文本');
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
          setAudioBlob(typedBlob);
          const url = URL.createObjectURL(typedBlob);
          setAudioUrl(url);
          toast.success('语音合成完成');
        },
        onError: (error) => {
          const msg = error.message || '';
          if (msg.includes('不支持 TTS')) {
            toast.error('模型不支持 TTS', '请选择一个标记为支持 TTS 的模型');
          } else if (msg.includes('不支持 TTS 端点')) {
            toast.error('后端不支持 TTS', '请使用 vLLM-Omni 后端加载 TTS 模型');
          } else {
            toast.error('语音合成失败', msg);
          }
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
    if (!audioBlob || !audioUrl) return;
    const ext = responseFormat || 'mp3';
    const a = document.createElement('a');
    a.href = audioUrl;
    a.download = `tts_${Date.now()}.${ext}`;
    a.click();
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">TTS 模型</label>
          <Select value={model} onValueChange={(v) => { setModel(v); setVoice(''); }}>
            <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
              <SelectValue placeholder="选择 TTS 模型" />
            </SelectTrigger>
            <SelectContent>
              {models.map((m) => (
                <SelectItem key={m.id} value={m.alias || m.name}>
                  {m.alias || m.name}
                  {m.backendType && (
                    <span className="ml-2 text-xs text-muted-foreground">({BACKEND_LABELS[m.backendType] || m.backendType})</span>
                  )}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {backendLabel && (
            <p className="text-xs text-muted-foreground mt-1">后端: {backendLabel}</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium mb-1.5">输入文本</label>
          <Textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="输入要转换为语音的文本..."
            className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
            rows={4}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1.5">语音 (Voice)</label>
            <Select value={voice} onValueChange={setVoice}>
              <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
                <SelectValue placeholder={voices.length > 0 ? '选择语音' : '输入语音名称 (如: alloy)'} />
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
            <label className="block text-sm font-medium mb-1.5">输出格式</label>
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
            <label className="block text-sm font-medium mb-1.5">语速: {speed}x</label>
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
            <label className="text-sm font-medium">流式生成</label>
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
              生成中...
            </>
          ) : (
            <>
              <Volume2 className="w-4 h-4 mr-2" />
              生成语音
            </>
          )}
        </Button>
      </div>

      {audioUrl && (
        <div className="border rounded-lg p-4 space-y-3">
          <h3 className="text-sm font-medium">生成结果</h3>
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
            <Progress value={100} className="flex-1 h-2" />
            <Button variant="outline" size="icon" onClick={handleDownload} title="下载音频">
              <Download className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
