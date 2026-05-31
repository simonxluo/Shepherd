import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Download, Square } from 'lucide-react';
import { pcmToWav } from '@/features/tts/lib/pcmToWav';
import type { StreamState, TTSStreamMetrics } from '@/features/tts/lib/StreamAudioPlayer';

interface StreamPlaybackPanelProps {
  state: StreamState;
  metrics: TTSStreamMetrics;
  pcmChunks: Int16Array[];
  sampleRate: number;
  onStop: () => void;
}

export function StreamPlaybackPanel({
  state,
  metrics,
  pcmChunks,
  sampleRate,
  onStop,
}: StreamPlaybackPanelProps) {
  const { t } = useTranslation();

  if (state === 'idle') return null;

  const handleDownload = () => {
    if (pcmChunks.length === 0) return;
    const blob = pcmToWav(pcmChunks, sampleRate);
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `tts_stream_${Date.now()}.wav`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };

  const statusColor = {
    idle: 'bg-muted text-muted-foreground',
    streaming: 'bg-blue-500/10 text-blue-500',
    playing: 'bg-green-500/10 text-green-500',
    completed: 'bg-green-500/10 text-green-600',
    error: 'bg-red-500/10 text-red-500',
  }[state];

  const statusLabel = {
    idle: '',
    streaming: t('tts.streamStatus.streaming', '流式中'),
    playing: t('tts.streamStatus.playing', '播放中'),
    completed: t('tts.streamStatus.completed', '完成'),
    error: t('tts.streamStatus.error', '错误'),
  }[state];

  return (
    <div className="border rounded-lg p-4 space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className={`text-xs font-medium px-2 py-0.5 rounded ${statusColor}`}>
            {statusLabel}
          </span>
          {(state === 'streaming' || state === 'playing') && (
            <span className="text-xs text-muted-foreground">
              {formatBytes(metrics.bytesReceived)} {t('tts.received', '已接收')}
            </span>
          )}
        </div>
        {(state === 'streaming' || state === 'playing') && (
          <Button variant="outline" size="sm" onClick={onStop}>
            <Square className="w-3 h-3 mr-1" />
            {t('tts.stop', '停止')}
          </Button>
        )}
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
        <MetricCard label={t('tts.ttfp', '首包延迟')} value={metrics.ttfp !== null ? `${metrics.ttfp} ms` : '--'} />
        <MetricCard label={t('tts.rtf', '实时率')} value={metrics.rtf !== null ? `${metrics.rtf}x` : '--'} />
        <MetricCard label={t('tts.audioDuration', '音频时长')} value={metrics.audioDuration > 0 ? `${metrics.audioDuration} s` : '--'} />
        <MetricCard label={t('tts.speedMultiplier', '速度倍率')} value={metrics.speedMultiplier > 0 ? `${metrics.speedMultiplier}x` : '--'} />
      </div>

      {state === 'completed' && pcmChunks.length > 0 && (
        <Button variant="outline" size="sm" onClick={handleDownload} className="w-full">
          <Download className="w-4 h-4 mr-2" />
          {t('tts.downloadWav', '下载 WAV')}
        </Button>
      )}
    </div>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className="p-2 text-center">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="text-sm font-mono font-medium">{value}</div>
    </Card>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
