import { useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Play, Pause, Download, CheckCircle2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { StreamPlaybackPanel } from './StreamPlaybackPanel';
import type { StreamState, TTSStreamMetrics } from '../lib/StreamAudioPlayer';

interface TTSPlaybackAreaProps {
  /** Stream playback state */
  streamState: StreamState;
  /** Stream metrics */
  streamMetrics: TTSStreamMetrics;
  /** PCM chunks from stream player */
  pcmChunks: Int16Array[];
  /** Sample rate for stream playback */
  sampleRate: number;
  /** Stop stream callback */
  onStopStream: () => void;
  /** Non-stream audio URL */
  audioUrl: string | null;
  /** Response format for download file extension */
  responseFormat: string;
}

export function TTSPlaybackArea({
  streamState,
  streamMetrics,
  pcmChunks,
  sampleRate,
  onStopStream,
  audioUrl,
  responseFormat,
}: TTSPlaybackAreaProps) {
  const { t } = useTranslation();
  const audioRef = useRef<HTMLAudioElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);

  const isStreamActive = streamState === 'streaming' || streamState === 'playing';

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
    const ext = responseFormat === 'pcm' ? 'wav' : (responseFormat || 'mp3');
    const a = document.createElement('a');
    a.href = audioUrl;
    a.download = `tts_${Date.now()}.${ext}`;
    a.click();
  };

  return (
    <>
      {/* Stream playback panel */}
      <StreamPlaybackPanel
        state={streamState}
        metrics={streamMetrics}
        pcmChunks={pcmChunks}
        sampleRate={sampleRate}
        onStop={onStopStream}
      />

      {/* Non-stream result */}
      {audioUrl && !isStreamActive && (
        <div className="border rounded-lg p-4 space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">{t('tts.result', 'Result')}</h3>
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <CheckCircle2 className="w-3 h-3 text-green-500" />
              {t('tts.autoSaved', 'Auto-saved')}
            </span>
          </div>
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
            <Button variant="outline" size="icon" onClick={handleDownload} title={t('tts.download', 'Download audio')}>
              <Download className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </>
  );
}
