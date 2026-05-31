import { useState, useRef, useMemo, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Music, Play, Pause, Download, Clock, HardDrive } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn, formatBytes } from '@/lib/utils';
import { useLoadedModels } from '@/features/creative/hooks';
import { useMusicGeneration } from '@/features/music-gen/hooks';
import { musicRegistry } from '@/features/music-gen/registry';
import { toast } from '@/hooks/useToast';
import type { MusicGenRequest, MusicPluginPanelProps } from '@/features/music-gen/types';

// Import plugins to register them
import '@/features/music-gen/plugins/generic';
import '@/features/music-gen/plugins/acestep';

export function MusicGenPage() {
  const { t } = useTranslation();
  const { data: allModels = [] } = useLoadedModels();
  const musicModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.music),
    [allModels]
  );

  const musicGen = useMusicGeneration();
  const audioRef = useRef<HTMLAudioElement>(null);
  const prevAudioUrlRef = useRef<string | null>(null);

  // Plugin system
  const plugins = useMemo(() => musicRegistry.getAllPlugins(), []);
  const [activePluginId, setActivePluginId] = useState<string>(plugins[0]?.id || 'generic');
  const [modelByPlugin, setModelByPlugin] = useState<Record<string, string>>({});

  // Audio state
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [audioBlob, setAudioBlob] = useState<Blob | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [generationTime, setGenerationTime] = useState<number | null>(null);
  const [currentTime, setCurrentTime] = useState(0);
  const [audioDuration, setAudioDuration] = useState(0);
  const [responseFormat, setResponseFormat] = useState('wav');

  // 组件卸载时释放 blob URL，避免内存泄漏
  useEffect(() => {
    return () => {
      if (prevAudioUrlRef.current) {
        URL.revokeObjectURL(prevAudioUrlRef.current);
        prevAudioUrlRef.current = null;
      }
    };
  }, []);

  // Current plugin and matched models
  const activePlugin = useMemo(
    () => plugins.find((p) => p.id === activePluginId) || plugins[0],
    [plugins, activePluginId]
  );

  const matchedModels = useMemo(
    () => (activePlugin ? musicRegistry.getModelsForPlugin(activePlugin, musicModels) : []),
    [activePlugin, musicModels]
  );

  const currentModelName = modelByPlugin[activePluginId] || '';
  const selectedModel = useMemo(
    () => matchedModels.find((m) => (m.alias || m.name) === currentModelName) || matchedModels[0] || null,
    [matchedModels, currentModelName]
  );

  const handleModelChange = useCallback((modelName: string) => {
    setModelByPlugin((prev) => ({
      ...prev,
      [activePluginId]: modelName,
    }));
  }, [activePluginId]);

  const handleGenerate = useCallback((payload: MusicGenRequest) => {
    setResponseFormat(payload.response_format || 'wav');
    const startTime = Date.now();

    musicGen.mutate(payload, {
      onSuccess: ({ blob, contentType }) => {
        const elapsed = (Date.now() - startTime) / 1000;
        setGenerationTime(elapsed);
        const typedBlob = new Blob([blob], { type: contentType });
        setAudioBlob(typedBlob);
        const url = URL.createObjectURL(typedBlob);
        // 先设置新 URL，再释放旧的，避免 <audio> 引用已释放的 URL
        setAudioUrl(url);
        if (prevAudioUrlRef.current && prevAudioUrlRef.current !== url) {
          URL.revokeObjectURL(prevAudioUrlRef.current);
        }
        prevAudioUrlRef.current = url;
        setCurrentTime(0);
        toast.success(t('musicGen.generateSuccess', '音乐生成完成'));
      },
      onError: (error) => {
        toast.error(t('musicGen.generateFailed', '音乐生成失败'), error.message);
      },
    });
  }, [musicGen, t]);

  const handlePlayPause = () => {
    if (!audioRef.current) return;
    if (isPlaying) {
      audioRef.current.pause();
    } else {
      audioRef.current.play().catch(() => {
        // 自动播放策略阻止时，onPause/onPlay 事件会同步状态
      });
    }
    // 不在这里设置 isPlaying，由 <audio> 的 onPlay/onPause 事件同步状态
  };

  const handleDownload = () => {
    if (!audioUrl) return;
    const ext = responseFormat || 'wav';
    const a = document.createElement('a');
    a.href = audioUrl;
    a.download = `music_${Date.now()}.${ext}`;
    a.click();
  };

  const handleTimeUpdate = () => {
    if (audioRef.current) {
      setCurrentTime(audioRef.current.currentTime);
    }
  };

  const handleLoadedMetadata = () => {
    if (audioRef.current) {
      setAudioDuration(audioRef.current.duration);
    }
  };

  const progressPercent = audioDuration > 0 ? (currentTime / audioDuration) * 100 : 0;

  const formatTime = (seconds: number) => {
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  // Build panel props
  const panelProps: MusicPluginPanelProps = {
    model: selectedModel,
    matchedModels,
    onGenerate: handleGenerate,
    isGenerating: musicGen.isPending,
    onModelChange: handleModelChange,
  };

  const PanelComponent = activePlugin?.component;

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex-1 flex overflow-hidden">
        {/* Center: main content area */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Header */}
          <div className="mb-6">
            <h1 className="text-2xl font-bold text-foreground">
              {t('musicGen.title', '音乐生成')}
            </h1>
            <p className="text-sm text-muted-foreground mt-1">
              {t('musicGen.description', '通过 AI 模型从文本描述生成音乐')}
            </p>
          </div>

          {/* Operation UI — centered */}
          <div className="max-w-3xl mx-auto">
            <div className="space-y-6">
              {/* Plugin panel */}
              {PanelComponent && <PanelComponent {...panelProps} />}

              {/* Result */}
              {audioUrl && (
                <div className="border rounded-lg overflow-hidden">
                  {/* Result header */}
                  <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
                    <h3 className="text-sm font-medium">
                      {t('musicGen.result', '生成结果')}
                    </h3>
                    <span className="text-xs text-muted-foreground">
                      {isPlaying ? t('musicGen.playing', '播放中') : t('musicGen.paused', '已暂停')}
                    </span>
                  </div>

                  {/* Audio player */}
                  <div className="p-4 space-y-4">
                    <audio
                      ref={audioRef}
                      src={audioUrl}
                      onEnded={() => setIsPlaying(false)}
                      onPause={() => setIsPlaying(false)}
                      onPlay={() => setIsPlaying(true)}
                      onTimeUpdate={handleTimeUpdate}
                      onLoadedMetadata={handleLoadedMetadata}
                    />

                    {/* Progress bar */}
                    <div className="space-y-1.5">
                      <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                        <div
                          className="h-full bg-primary rounded-full transition-all duration-200"
                          style={{ width: `${progressPercent}%` }}
                        />
                      </div>
                      <div className="flex justify-between text-xs text-muted-foreground">
                        <span>{formatTime(currentTime)}</span>
                        <span>{audioDuration > 0 ? formatTime(audioDuration) : '--:--'}</span>
                      </div>
                    </div>

                    {/* Controls */}
                    <div className="flex items-center gap-2">
                      <Button variant="outline" size="icon" onClick={handlePlayPause}>
                        {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                      </Button>
                      <Button variant="outline" size="icon" onClick={handleDownload} title={t('musicGen.download', '下载音频')}>
                        <Download className="w-4 h-4" />
                      </Button>
                    </div>

                    {/* Metrics */}
                    <div className="grid grid-cols-3 gap-3">
                      {generationTime !== null && (
                        <div className="flex items-center gap-2 px-3 py-2 rounded-md bg-muted/50 text-xs">
                          <Clock className="w-3.5 h-3.5 text-muted-foreground" />
                          <div>
                            <div className="text-muted-foreground">{t('musicGen.generationTime', '生成耗时')}</div>
                            <div className="font-medium">{generationTime.toFixed(1)}s</div>
                          </div>
                        </div>
                      )}
                      {audioBlob && (
                        <div className="flex items-center gap-2 px-3 py-2 rounded-md bg-muted/50 text-xs">
                          <HardDrive className="w-3.5 h-3.5 text-muted-foreground" />
                          <div>
                            <div className="text-muted-foreground">{t('musicGen.fileSize', '文件大小')}</div>
                            <div className="font-medium">{formatBytes(audioBlob.size)}</div>
                          </div>
                        </div>
                      )}
                      {audioDuration > 0 && (
                        <div className="flex items-center gap-2 px-3 py-2 rounded-md bg-muted/50 text-xs">
                          <Music className="w-3.5 h-3.5 text-muted-foreground" />
                          <div>
                            <div className="text-muted-foreground">{t('musicGen.audioDuration', '音频时长')}</div>
                            <div className="font-medium">{formatTime(audioDuration)}</div>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right: vertical tab bar */}
        {plugins.length > 1 && (
          <div className="flex flex-col w-[120px] border-l bg-muted/30">
            {plugins.map((plugin) => {
              const isActive = plugin.id === activePluginId;
              return (
                <button
                  key={plugin.id}
                  onClick={() => setActivePluginId(plugin.id)}
                  className={cn(
                    'relative text-left px-3 py-3 text-sm transition-colors',
                    'hover:bg-accent/50',
                    isActive
                      ? 'bg-primary/10 font-medium text-foreground'
                      : 'text-muted-foreground'
                  )}
                >
                  {isActive && (
                    <span className="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-6 bg-primary rounded-r" />
                  )}
                  <span className="block truncate">
                    {t(plugin.labelKey, plugin.labelFallback)}
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
