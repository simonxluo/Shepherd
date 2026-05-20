import { useState, useRef, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useLoadedModels } from '@/features/creative/hooks';
import { useTTS, getTTSModelFeatures, type TTSRequest } from '../hooks';
import { StreamAudioPlayer, type StreamState, type TTSStreamMetrics } from '../lib/StreamAudioPlayer';
import { ttsRegistry } from '../registry';
import { VerticalTabBar } from './VerticalTabBar';
import { TTSPlaybackArea } from './TTSPlaybackArea';
import { toast } from '@/hooks/useToast';
import type { TTSPluginPanelProps } from '../types';

// Import plugins to register them
import '../plugins/generic';
import '../plugins/voxcpm2';

export function TTSPageShell() {
  const { t } = useTranslation();
  const { data: allModels = [] } = useLoadedModels();

  // Filter to TTS-capable models only
  const ttsModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.tts),
    [allModels]
  );

  // Get registered plugins
  const plugins = useMemo(() => ttsRegistry.getAllPlugins(), []);

  // Active tab state
  const [activePluginId, setActivePluginId] = useState<string>(plugins[0]?.id || 'generic');

  // Selected model per plugin tab
  const [modelByPlugin, setModelByPlugin] = useState<Record<string, string>>({});

  // TTS mutation (non-stream)
  const tts = useTTS();

  // Stream player ref
  const playerRef = useRef<StreamAudioPlayer | null>(null);

  // Stream state
  const [streamState, setStreamState] = useState<StreamState>('idle');
  const [streamMetrics, setStreamMetrics] = useState<TTSStreamMetrics>({
    ttfp: null, rtf: null, audioDuration: 0, speedMultiplier: 0, bytesReceived: 0,
  });
  const [pcmChunks, setPcmChunks] = useState<Int16Array[]>([]);

  // Non-stream audio
  const [audioUrl, setAudioUrl] = useState<string | null>(null);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      playerRef.current?.destroy();
    };
  }, []);

  // Current plugin and matched models
  const activePlugin = useMemo(
    () => plugins.find((p) => p.id === activePluginId) || plugins[0],
    [plugins, activePluginId]
  );

  const matchedModels = useMemo(
    () => (activePlugin ? ttsRegistry.getModelsForPlugin(activePlugin, ttsModels) : []),
    [activePlugin, ttsModels]
  );

  // Current selected model for active plugin
  const currentModelName = modelByPlugin[activePluginId] || '';
  const selectedModel = useMemo(
    () => matchedModels.find((m) => (m.alias || m.name) === currentModelName) || matchedModels[0] || null,
    [matchedModels, currentModelName]
  );

  // Auto-select first matched model if none selected
  useEffect(() => {
    if (matchedModels.length > 0 && !currentModelName) {
      const firstModel = matchedModels[0];
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setModelByPlugin((prev) => ({
        ...prev,
        [activePluginId]: firstModel.alias || firstModel.name,
      }));
    }
  }, [matchedModels, currentModelName, activePluginId]);

  const handleModelChange = useCallback((modelName: string) => {
    setModelByPlugin((prev) => ({
      ...prev,
      [activePluginId]: modelName,
    }));
  }, [activePluginId]);

  const handleStopStream = useCallback(() => {
    playerRef.current?.stop();
  }, []);

  // Get features for current model (for sample rate)
  const currentFeatures = useMemo(
    () => (selectedModel ? getTTSModelFeatures(selectedModel) : { defaultSampleRate: 24000, defaultFormat: 'mp3', supportsStreamPcm: false }),
    [selectedModel]
  );

  // Unified generate handler
  const handleGenerate = useCallback(async (payload: TTSRequest) => {
    const isStreamRequest = payload.stream === true;

    if (isStreamRequest) {
      try {
        if (!playerRef.current) {
          playerRef.current = new StreamAudioPlayer(currentFeatures.defaultSampleRate);
          await playerRef.current.init();
        }

        playerRef.current.onStateChange = (state) => {
          setStreamState(state);
          // Sync pcmChunks when stream completes
          if (state === 'completed' || state === 'error' || state === 'idle') {
            setPcmChunks(playerRef.current?.pcmChunks ? [...playerRef.current.pcmChunks] : []);
          }
        };
        playerRef.current.onMetricsUpdate = (metrics) => {
          setStreamMetrics(metrics);
          // Keep pcmChunks in sync during streaming
          setPcmChunks(playerRef.current?.pcmChunks ? [...playerRef.current.pcmChunks] : []);
        };
        playerRef.current.onError = (err) => {
          toast.error(t('tts.generateFailed', 'Stream playback error'), err.message);
        };

        setPcmChunks([]);
        await playerRef.current.startStream('/v1/audio/speech', payload);
      } catch (err) {
        toast.error(t('tts.generateFailed', 'Initialization failed'), (err as Error).message);
      }
    } else {
      if (audioUrl) URL.revokeObjectURL(audioUrl);

      tts.mutate(payload, {
        onSuccess: ({ blob, contentType }) => {
          const typedBlob = new Blob([blob], { type: contentType });
          const url = URL.createObjectURL(typedBlob);
          setAudioUrl(url);
          toast.success(t('tts.generateSuccess', 'Speech synthesis complete'));
        },
        onError: (error) => {
          toast.error(t('tts.generateFailed', 'Speech synthesis failed'), error.message);
        },
      });
    }
  }, [currentFeatures.defaultSampleRate, audioUrl, tts, t]);

  const isStreamActive = streamState === 'streaming' || streamState === 'playing';
  const isGenerating = tts.isPending || isStreamActive;

  // Build panel props
  const panelProps: TTSPluginPanelProps = {
    model: selectedModel,
    matchedModels,
    onGenerate: handleGenerate,
    isGenerating,
    streamState,
    streamMetrics,
    audioUrl,
    onModelChange: handleModelChange,
  };

  const PanelComponent = activePlugin?.component;

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex-1 flex overflow-hidden">
        {/* Left: main content area */}
        <div className="flex-1 overflow-y-auto p-6">
          <div className="max-w-3xl mx-auto">
            <div className="mb-6">
              <h1 className="text-2xl font-bold text-foreground">
                {t('tts.title', 'Text-to-Speech (TTS)')}
              </h1>
            </div>

            <div className="space-y-6">
              {/* Plugin panel */}
              {PanelComponent && <PanelComponent {...panelProps} />}

              {/* Playback area (shared) */}
              <TTSPlaybackArea
                streamState={streamState}
                streamMetrics={streamMetrics}
                pcmChunks={pcmChunks}
                sampleRate={currentFeatures.defaultSampleRate}
                onStopStream={handleStopStream}
                audioUrl={audioUrl}
                responseFormat={currentFeatures.defaultFormat}
              />
            </div>
          </div>
        </div>

        {/* Right: vertical tab bar */}
        <VerticalTabBar
          plugins={plugins}
          activeId={activePluginId}
          onSelect={setActivePluginId}
        />
      </div>
    </div>
  );
}
