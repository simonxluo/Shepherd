import { useState, useRef } from 'react';
import { Volume2, Loader2, Play, Pause, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Slider } from '@/components/ui/slider';
import { Progress } from '@/components/ui/progress';
import { useTTS } from '../hooks';
import { useToast } from '@/hooks/useToast';

interface TTSPanelProps {
  models: Array<{ id: string; name: string; alias?: string }>;
}

export function TTSPanel({ models }: TTSPanelProps) {
  const toast = useToast();
  const tts = useTTS();
  const audioRef = useRef<HTMLAudioElement>(null);

  const [model, setModel] = useState('');
  const [input, setInput] = useState('');
  const [voice, setVoice] = useState('');
  const [speed, setSpeed] = useState(1);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [audioBlob, setAudioBlob] = useState<Blob | null>(null);

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
        speed: speed !== 1 ? speed : undefined,
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
          toast.error('语音合成失败', error.message);
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
    const a = document.createElement('a');
    a.href = audioUrl;
    a.download = `tts_${Date.now()}.mp3`;
    a.click();
  };

  return (
    <div className="space-y-6">
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1.5">TTS 模型</label>
          <Select
            value={model}
            onValueChange={setModel}
          >
            <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
              <SelectValue placeholder="选择 TTS 模型" />
            </SelectTrigger>
            <SelectContent>
              {models.map((m) => (
                <SelectItem key={m.id} value={m.alias || m.name}>
                  {m.alias || m.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
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
            <Input
              type="text"
              value={voice}
              onChange={(e) => setVoice(e.target.value)}
              placeholder="可选，如：alloy, echo"
              className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent"
            />
          </div>
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
