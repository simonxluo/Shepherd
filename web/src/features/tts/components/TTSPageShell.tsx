import { useState, useRef, useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { PanelLeftOpen } from 'lucide-react';
import { useLoadedModels } from '@/features/creative/hooks';
import { useModels } from '@/features/models';
import { Button } from '@/components/ui/button';
import { useTTS, getTTSModelFeatures, type TTSRequest } from '../hooks';
import { StreamAudioPlayer, type StreamState, type TTSStreamMetrics } from '../lib/StreamAudioPlayer';
import { ttsRegistry } from '../registry';
import { VerticalTabBar } from './VerticalTabBar';
import { TTSPlaybackArea } from './TTSPlaybackArea';
import { TTSHistoryPanel } from './TTSHistoryPanel';
import { useCreateTTSHistory } from '../historyHooks';
import { toast } from '@/hooks/useToast';
import { pcmToWav } from '../lib/pcmToWav';
import type { TTSPluginPanelProps } from '../types';

// Import plugins to register them
import '../plugins/generic';
import '../plugins/voxcpm2';
import '../plugins/qwen3tts';

export function TTSPageShell() {
  const { t } = useTranslation();
  const { data: allModels = [] } = useLoadedModels();
  const { data: fullModelsList = [] } = useModels();

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

  // History saving
  const createHistory = useCreateTTSHistory();

  // Track last generation payload for history saving
  const lastPayloadRef = useRef<TTSRequest | null>(null);

  // Stream player ref
  const playerRef = useRef<StreamAudioPlayer | null>(null);

  // AbortController for non-stream cancellation
  const abortControllerRef = useRef<AbortController | null>(null);

  // Stream state
  const [streamState, setStreamState] = useState<StreamState>('idle');
  const [streamMetrics, setStreamMetrics] = useState<TTSStreamMetrics>({
    ttfp: null, rtf: null, audioDuration: 0, speedMultiplier: 0, bytesReceived: 0,
  });
  const [pcmChunks, setPcmChunks] = useState<Int16Array[]>([]);

  // Non-stream audio (with ref for safe cleanup)
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const audioUrlRef = useRef<string | null>(null);

  const setAudioUrlSafe = useCallback((url: string | null) => {
    if (audioUrlRef.current) URL.revokeObjectURL(audioUrlRef.current);
    audioUrlRef.current = url;
    setAudioUrl(url);
  }, []);

  // Last used voice (for download naming)
  const [lastVoice, setLastVoice] = useState<string>('');

  // Auto-play toggle with localStorage persistence
  const [autoPlay, setAutoPlay] = useState(() => {
    try { return localStorage.getItem('shepherd-tts-autoplay') === 'true'; }
    catch { return false; }
  });

  useEffect(() => {
    try { localStorage.setItem('shepherd-tts-autoplay', String(autoPlay)); }
    catch { /* silent */ }
  }, [autoPlay]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      playerRef.current?.destroy();
      if (audioUrlRef.current) URL.revokeObjectURL(audioUrlRef.current);
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

  // 查找选中模型在完整模型列表中的状态
  const fullModel = useMemo(
    () => selectedModel ? fullModelsList.find((m) => m.id === selectedModel.id) : undefined,
    [selectedModel, fullModelsList]
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

  const handleCancel = useCallback(() => {
    abortControllerRef.current?.abort();
    abortControllerRef.current = null;
    playerRef.current?.stop();
  }, []);

  // Get features for current model (for sample rate)
  const currentFeatures = useMemo(
    () => (selectedModel ? getTTSModelFeatures(selectedModel) : { defaultSampleRate: 24000, defaultFormat: 'mp3', supportsStreamPcm: false, supportsRefAudio: false }),
    [selectedModel]
  );

  // Save to history helper
  const saveToHistory = useCallback(async (blob: Blob, payload: TTSRequest, format: string) => {
    try {
      const formData = new FormData();
      formData.append('audio', blob, `audio.${format}`);
      formData.append('metadata', JSON.stringify({
        model: payload.model,
        inputText: payload.input,
        format: format,
        duration: 0, // We don't know exact duration from blob
        params: payload,
      }));
      createHistory.mutate(formData);
    } catch {
      // Silent fail for history saving - non-critical
    }
  }, [createHistory]);

  // Unified generate handler
  const handleGenerate = useCallback(async (payload: TTSRequest) => {
    const isStreamRequest = payload.stream === true;
    lastPayloadRef.current = payload;
    setLastVoice(payload.voice || '');

    if (isStreamRequest) {
      try {
        if (!playerRef.current) {
          const player = new StreamAudioPlayer(currentFeatures.defaultSampleRate);
          await player.init();
          playerRef.current = player;
        }

        playerRef.current.onStateChange = (state) => {
          setStreamState(state);
          // Sync pcmChunks when stream completes
          if (state === 'completed' || state === 'error' || state === 'idle') {
            const chunks = playerRef.current?.pcmChunks ? [...playerRef.current.pcmChunks] : [];
            setPcmChunks(chunks);

            // Auto-save to history on stream completion
            if (state === 'completed' && chunks.length > 0 && lastPayloadRef.current) {
              const wavBlob = pcmToWav(chunks, currentFeatures.defaultSampleRate);
              saveToHistory(wavBlob, lastPayloadRef.current, 'wav');
            }
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
      setAudioUrlSafe(null);

      const controller = new AbortController();
      abortControllerRef.current = controller;

      tts.mutate({ ...payload, signal: controller.signal }, {
        onSuccess: ({ blob, contentType }) => {
          const typedBlob = new Blob([blob], { type: contentType });
          const url = URL.createObjectURL(typedBlob);
          setAudioUrlSafe(url);
          toast.success(t('tts.generateSuccess', 'Speech synthesis complete'));

          // Auto-play if enabled
          if (autoPlay) {
            const audio = new Audio(url);
            audio.play().catch(() => {});
          }

          // Auto-save to history
          const format = payload.response_format || 'mp3';
          saveToHistory(typedBlob, payload, format);
        },
        onError: (error) => {
          if (error.name === 'AbortError') return; // silent cancel
          toast.error(t('tts.generateFailed', 'Speech synthesis failed'), error.message);
        },
        onSettled: () => {
          abortControllerRef.current = null;
        },
      });
    }
  }, [currentFeatures.defaultSampleRate, tts, t, saveToHistory, setAudioUrlSafe, autoPlay]);

  const isStreamActive = streamState === 'streaming' || streamState === 'playing';
  const isGenerating = tts.isPending || isStreamActive;

  // Sidebar visibility for mobile
  const [showSidebar, setShowSidebar] = useState(false);

  // "Use as reference" handler for history panel
  const [refAudioOverride, setRefAudioOverride] = useState<string | undefined>();

  const handleUseAsReference = useCallback((audioUrl: string) => {
    setRefAudioOverride(audioUrl);
    toast.success(t('tts.refAudioSet', 'Reference audio set from history'));
  }, [t]);

  // Voice refresh trigger for plugin panels
  const [voiceRefreshTrigger, setVoiceRefreshTrigger] = useState(0);
  const handleVoiceRegistered = useCallback(() => setVoiceRefreshTrigger((v) => v + 1), []);

  // Build panel props
  const panelProps: TTSPluginPanelProps = {
    model: selectedModel,
    matchedModels,
    onGenerate: handleGenerate,
    onCancel: handleCancel,
    isGenerating,
    streamState,
    streamMetrics,
    audioUrl,
    onModelChange: handleModelChange,
    refAudioOverride,
    modelStatus: fullModel?.status,
    fullModelId: fullModel?.id,
    voiceRefreshTrigger,
  };

  const PanelComponent = activePlugin?.component;

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex-1 flex overflow-hidden">
        {/* Left: History sidebar */}
        <div className={`w-72 border-r flex-col bg-muted/20 shrink-0 ${showSidebar ? 'flex' : 'hidden md:flex'}`}>
          <TTSHistoryPanel
            onUseAsReference={handleUseAsReference}
            supportsRefAudio={currentFeatures.supportsRefAudio}
            supportsVoiceLibrary={currentFeatures.supportsRefAudio}
            modelName={selectedModel ? (selectedModel.alias || selectedModel.name) : ''}
            onVoiceRegistered={handleVoiceRegistered}
          />
        </div>

        {/* Center: main content area */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Header — left-aligned at the top */}
          <div className="mb-6 flex items-start gap-3">
            {/* Mobile sidebar toggle */}
            <Button
              variant="ghost"
              size="icon"
              className="md:hidden shrink-0 mt-0.5"
              onClick={() => setShowSidebar((v) => !v)}
            >
              <PanelLeftOpen className="w-5 h-5" />
            </Button>
            <div>
              <h1 className="text-2xl font-bold text-foreground">
                {t('tts.title', 'Text-to-Speech (TTS)')}
              </h1>
              <p className="text-muted-foreground">
                {t('tts.description', 'Convert text to natural speech with streaming playback and voice cloning')}
              </p>
            </div>
          </div>

          {/* Operation UI — centered */}
          <div className="max-w-3xl mx-auto">
            <div className="space-y-6">
              {/* Plugin panel - always render, let plugin handle no-model state */}
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
                voice={lastVoice}
                autoPlay={autoPlay}
                onAutoPlayChange={setAutoPlay}
              />
            </div>
          </div>
        </div>

        {/* Right: vertical tab bar - always shown regardless of model state */}
        <VerticalTabBar
          plugins={plugins}
          activeId={activePluginId}
          onSelect={setActivePluginId}
        />
      </div>
    </div>
  );
}
