import { useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Play, Pause, Download, Trash2, Star, Clock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useTTSHistory, useToggleTTSFavourite, useDeleteTTSHistory } from '../historyHooks';
import { getTTSAudioUrl, type TTSHistoryItem } from '../api';

interface TTSHistoryPanelProps {
  /** Callback when user selects an item as reference audio */
  onUseAsReference?: (audioUrl: string) => void;
  /** Whether current plugin supports ref_audio */
  supportsRefAudio?: boolean;
}

export function TTSHistoryPanel({ onUseAsReference, supportsRefAudio }: TTSHistoryPanelProps) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState<'all' | 'favourites'>('all');
  const [playingId, setPlayingId] = useState<string | null>(null);
  const audioRef = useRef<HTMLAudioElement>(null);

  const { data, isLoading } = useTTSHistory({
    limit: 50,
    favourite: filter === 'favourites' ? true : undefined,
  });

  const toggleFav = useToggleTTSFavourite();
  const deleteItem = useDeleteTTSHistory();

  const items = data?.items ?? [];

  const handlePlay = (item: TTSHistoryItem) => {
    if (!audioRef.current) return;

    if (playingId === item.id) {
      audioRef.current.pause();
      setPlayingId(null);
      return;
    }

    audioRef.current.src = getTTSAudioUrl(item.id);
    audioRef.current.play();
    setPlayingId(item.id);
  };

  const handleDownload = (item: TTSHistoryItem) => {
    const a = document.createElement('a');
    a.href = getTTSAudioUrl(item.id);
    a.download = `tts_${item.id}.${item.format || 'mp3'}`;
    a.click();
  };

  const handleDelete = (id: string) => {
    if (playingId === id) {
      audioRef.current?.pause();
      setPlayingId(null);
    }
    deleteItem.mutate(id);
  };

  const handleToggleFavourite = (item: TTSHistoryItem) => {
    toggleFav.mutate({ id: item.id, favourite: !item.favourite });
  };

  const formatDuration = (seconds: number) => {
    if (!seconds) return '--';
    const m = Math.floor(seconds / 60);
    const s = Math.round(seconds % 60);
    return m > 0 ? `${m}:${s.toString().padStart(2, '0')}` : `${s}s`;
  };

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr);
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
  };

  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <p className="text-sm text-muted-foreground">{t('tts.historyLoading', 'Loading history...')}</p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* Hidden audio element for playback */}
      <audio
        ref={audioRef}
        onEnded={() => setPlayingId(null)}
        onPause={() => setPlayingId(null)}
      />

      {/* Header with filter tabs */}
      <div className="flex items-center justify-between px-4 py-3 border-b shrink-0">
        <h3 className="text-sm font-medium flex items-center gap-1.5">
          <Clock className="w-4 h-4" />
          {t('tts.history', 'History')}
        </h3>
        <div className="flex gap-1">
          <Button
            variant={filter === 'all' ? 'default' : 'ghost'}
            size="sm"
            className="text-xs h-7"
            onClick={() => setFilter('all')}
          >
            {t('tts.historyAll', 'All')}
          </Button>
          <Button
            variant={filter === 'favourites' ? 'default' : 'ghost'}
            size="sm"
            className="text-xs h-7"
            onClick={() => setFilter('favourites')}
          >
            <Star className="w-3 h-3 mr-1" />
            {t('tts.historyFavourites', 'Favourites')}
          </Button>
        </div>
      </div>

      {/* History list */}
      {items.length === 0 ? (
        <div className="flex-1 flex items-center justify-center px-4">
          <p className="text-xs text-muted-foreground text-center">
            {filter === 'favourites'
              ? t('tts.noFavourites', 'No favourites yet')
              : t('tts.noHistory', 'No TTS history yet. Generated audio will appear here.')}
          </p>
        </div>
      ) : (
        <div className="flex-1 overflow-y-auto">
          <div className="space-y-1 p-2">
            {items.map((item) => (
              <div
                key={item.id}
                className="flex items-center gap-2 p-2 rounded hover:bg-accent/50 group transition-colors"
              >
                {/* Play button */}
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 shrink-0"
                  onClick={() => handlePlay(item)}
                >
                  {playingId === item.id ? (
                    <Pause className="w-3.5 h-3.5" />
                  ) : (
                    <Play className="w-3.5 h-3.5" />
                  )}
                </Button>

                {/* Text & meta */}
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{item.inputText}</p>
                  <p className="text-xs text-muted-foreground">
                    {item.model} &middot; {formatDuration(item.duration)} &middot; {formatDate(item.createdAt)}
                  </p>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                  {supportsRefAudio && onUseAsReference && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-xs h-7 px-2"
                      onClick={() => onUseAsReference(getTTSAudioUrl(item.id))}
                      title={t('tts.useAsRef', 'Use as reference audio')}
                    >
                      {t('tts.useAsRef', 'Ref')}
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => handleToggleFavourite(item)}
                    title={t('tts.toggleFavourite', 'Toggle favourite')}
                  >
                    <Star className={`w-3.5 h-3.5 ${item.favourite ? 'fill-yellow-400 text-yellow-400' : ''}`} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => handleDownload(item)}
                    title={t('tts.download', 'Download')}
                  >
                    <Download className="w-3.5 h-3.5" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-destructive hover:text-destructive"
                    onClick={() => handleDelete(item.id)}
                    title={t('tts.delete', 'Delete')}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
