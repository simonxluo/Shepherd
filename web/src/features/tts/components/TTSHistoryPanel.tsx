import { useState, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Play, Pause, Download, Trash2, Star, Clock, Mic, Check, X, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useTTSHistory, useToggleTTSFavourite, useDeleteTTSHistory } from '../historyHooks';
import { getTTSAudioUrl, type TTSHistoryItem } from '../api';
import { uploadVoice } from '@/lib/api/voices';
import { toast } from '@/hooks/useToast';

interface TTSHistoryPanelProps {
  /** Callback when user selects an item as reference audio */
  onUseAsReference?: (audioUrl: string) => void;
  /** Whether current plugin supports ref_audio */
  supportsRefAudio?: boolean;
  /** Whether current plugin supports voice library (save-as-voice) */
  supportsVoiceLibrary?: boolean;
  /** Model name for voice upload API */
  modelName?: string;
  /** Callback after a voice has been registered from history */
  onVoiceRegistered?: () => void;
}

export function TTSHistoryPanel({
  onUseAsReference,
  supportsRefAudio,
  supportsVoiceLibrary,
  modelName,
  onVoiceRegistered,
}: TTSHistoryPanelProps) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState<'all' | 'favourites'>('all');
  const [playingId, setPlayingId] = useState<string | null>(null);
  const audioRef = useRef<HTMLAudioElement>(null);

  // Text expansion state
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Save-as-voice state
  const [savingVoiceId, setSavingVoiceId] = useState<string | null>(null);
  const [voiceName, setVoiceName] = useState('');
  const [isSavingVoice, setIsSavingVoice] = useState(false);

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

  const handleExpandText = (id: string) => {
    setExpandedId((prev) => (prev === id ? null : id));
  };

  const handleStartSaveVoice = useCallback((item: TTSHistoryItem) => {
    const defaultName = item.inputText.slice(0, 20).replace(/[\n\r]/g, ' ').trim() || `voice_${Date.now()}`;
    setVoiceName(defaultName);
    setSavingVoiceId(item.id);
  }, []);

  const handleCancelSaveVoice = useCallback(() => {
    setSavingVoiceId(null);
    setVoiceName('');
  }, []);

  const handleConfirmSaveVoice = useCallback(async (item: TTSHistoryItem) => {
    if (!modelName || !voiceName.trim()) return;
    setIsSavingVoice(true);
    try {
      const resp = await fetch(getTTSAudioUrl(item.id));
      const blob = await resp.blob();
      const ext = item.format || 'wav';
      const file = new File([blob], `${voiceName.trim()}.${ext}`, { type: blob.type });
      await uploadVoice(modelName, file, voiceName.trim());
      toast.success(t('tts.voxcpm2.voiceSaved', 'Voice saved'));
      onVoiceRegistered?.();
      setSavingVoiceId(null);
      setVoiceName('');
    } catch (err) {
      toast.error(t('tts.voxcpm2.voiceSaveFailed', 'Failed to save voice'), (err as Error).message);
    } finally {
      setIsSavingVoice(false);
    }
  }, [modelName, voiceName, t, onVoiceRegistered]);

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
            {items.map((item) => {
              const isExpanded = expandedId === item.id;
              const isSavingThis = savingVoiceId === item.id;

              return (
                <div
                  key={item.id}
                  className="group rounded-md border border-transparent hover:border-border hover:bg-accent/50 transition-colors"
                >
                  {/* Row 1: Play button + text content + metadata */}
                  <div className="flex items-start gap-2 p-2">
                    {/* Play button */}
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 shrink-0 mt-0.5"
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
                      <p
                        className={`text-sm cursor-pointer select-none ${
                          isExpanded
                            ? 'max-h-24 overflow-y-auto whitespace-pre-wrap break-words'
                            : 'line-clamp-2'
                        }`}
                        onClick={() => handleExpandText(item.id)}
                        title={isExpanded ? t('tts.collapseText', 'Click to collapse') : t('tts.expandText', 'Click to expand')}
                      >
                        {item.inputText}
                      </p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        {item.model} &middot; {formatDuration(item.duration)} &middot; {formatDate(item.createdAt)}
                      </p>
                    </div>
                  </div>

                  {/* Row 2: Action buttons (hover-revealed, star always visible when favourited) */}
                  <div className={`flex items-center gap-0.5 px-2 pb-1.5 pl-11 transition-opacity ${
                    item.favourite ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'
                  }`}>
                    {/* Star (favourite toggle) */}
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6"
                      onClick={() => handleToggleFavourite(item)}
                      title={t('tts.toggleFavourite', 'Toggle favourite')}
                    >
                      <Star className={`w-3 h-3 ${item.favourite ? 'fill-yellow-400 text-yellow-400' : ''}`} />
                    </Button>

                    {/* Use as reference */}
                    {supportsRefAudio && onUseAsReference && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={() => onUseAsReference(getTTSAudioUrl(item.id))}
                        title={t('tts.useAsRef', 'Use as reference audio')}
                      >
                        <Mic className="w-3 h-3" />
                      </Button>
                    )}

                    {/* Save as voice */}
                    {supportsVoiceLibrary && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={() => handleStartSaveVoice(item)}
                        disabled={isSavingThis}
                        title={t('tts.saveVoice', 'Save as voice')}
                      >
                        <Mic className="w-3 h-3" />
                      </Button>
                    )}

                    {/* Download */}
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6"
                      onClick={() => handleDownload(item)}
                      title={t('tts.download', 'Download')}
                    >
                      <Download className="w-3 h-3" />
                    </Button>

                    {/* Delete */}
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 text-destructive hover:text-destructive"
                      onClick={() => handleDelete(item.id)}
                      title={t('tts.delete', 'Delete')}
                    >
                      <Trash2 className="w-3 h-3" />
                    </Button>
                  </div>

                  {/* Inline voice naming row */}
                  {isSavingThis && (
                    <div className="flex items-center gap-1.5 px-2 pb-2 pl-11">
                      <input
                        type="text"
                        value={voiceName}
                        onChange={(e) => setVoiceName(e.target.value)}
                        placeholder={t('tts.voiceNamePlaceholder', 'Voice name...')}
                        className="flex-1 h-7 px-2 text-xs border rounded bg-background focus:outline-none focus:ring-1 focus:ring-ring"
                        autoFocus
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') handleConfirmSaveVoice(item);
                          if (e.key === 'Escape') handleCancelSaveVoice();
                        }}
                        disabled={isSavingVoice}
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 shrink-0"
                        onClick={() => handleConfirmSaveVoice(item)}
                        disabled={isSavingVoice || !voiceName.trim()}
                        title={t('tts.saveVoice', 'Save')}
                      >
                        {isSavingVoice ? <Loader2 className="w-3 h-3 animate-spin" /> : <Check className="w-3 h-3" />}
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 shrink-0"
                        onClick={handleCancelSaveVoice}
                        disabled={isSavingVoice}
                        title={t('tts.cancelVoice', 'Cancel')}
                      >
                        <X className="w-3 h-3" />
                      </Button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
