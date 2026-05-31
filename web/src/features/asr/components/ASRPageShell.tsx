import { useCallback, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useLoadedModels } from '@/features/creative/hooks';
import { useModels } from '@/features/models';
import { useASRStore } from '@/stores/asrStore';
import { useASR } from '@/features/asr/hooks';
import { asrRegistry } from '@/features/asr/registry';
import { VerticalTabBar } from './VerticalTabBar';
import { toast } from '@/hooks/useToast';
import type { ASRPluginPanelProps } from '@/features/asr/types';
import type { ASRRequest } from '@/features/asr/types';

// Import plugins to register them
import '../plugins/generic';
import '../plugins/qwen3asr';

export function ASRPageShell() {
  const { t } = useTranslation();
  const { data: allModels = [] } = useLoadedModels();
  const { data: fullModelsList = [] } = useModels();

  // Filter to ASR-capable models only
  const asrModels = useMemo(
    () => allModels.filter((m) => m.capabilities?.asr),
    [allModels]
  );

  // Get registered plugins
  const plugins = useMemo(() => asrRegistry.getAllPlugins(), []);

  // Active tab state (persisted via Zustand)
  const activePluginId = useASRStore((s) => s.activePluginId);
  const setActivePluginId = useASRStore((s) => s.setActivePluginId);
  const modelByPlugin = useASRStore((s) => s.modelByPlugin);

  // ASR mutation
  const asr = useASR();

  // Current plugin and matched models
  const activePlugin = useMemo(
    () => plugins.find((p) => p.id === activePluginId) || plugins[0],
    [plugins, activePluginId]
  );

  const matchedModels = useMemo(
    () => (activePlugin ? asrRegistry.getModelsForPlugin(activePlugin, asrModels) : []),
    [activePlugin, asrModels]
  );

  // Current selected model for active plugin
  const currentModelName = modelByPlugin[activePluginId] || '';
  const selectedModel = useMemo(
    () => matchedModels.find((m) => (m.alias || m.name) === currentModelName) || matchedModels[0] || null,
    [matchedModels, currentModelName]
  );

  // Find selected model in full model list for status
  const fullModel = useMemo(
    () => selectedModel ? fullModelsList.find((m) => m.id === selectedModel.id) : undefined,
    [selectedModel, fullModelsList]
  );

  // Auto-select first matched model if none selected
  useEffect(() => {
    if (matchedModels.length > 0 && !currentModelName) {
      const firstModel = matchedModels[0];
      useASRStore.getState().setModelForPlugin(activePluginId, firstModel.alias || firstModel.name);
    }
  }, [matchedModels, currentModelName, activePluginId]);

  const handleModelChange = useCallback((modelName: string) => {
    useASRStore.getState().setModelForPlugin(activePluginId, modelName);
  }, [activePluginId]);

  // Unified transcribe handler
  const handleTranscribe = useCallback((payload: ASRRequest) => {
    asr.mutate(payload, {
      onError: (error) => {
        toast.error(t('asr.transcribeFailed', 'Transcription failed'), error.message);
      },
    });
  }, [asr, t]);

  // Build panel props
  const panelProps: ASRPluginPanelProps = {
    model: selectedModel,
    matchedModels,
    onTranscribe: handleTranscribe,
    isTranscribing: asr.isPending,
    result: asr.data ?? null,
    onModelChange: handleModelChange,
    modelStatus: fullModel?.status,
    fullModelId: fullModel?.id,
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
              {t('asr.title', 'Speech Recognition (ASR)')}
            </h1>
            <p className="text-sm text-muted-foreground mt-1">
              {t('asr.description', 'Upload audio files for speech recognition and transcription')}
            </p>
          </div>

          {/* Plugin panel */}
          <div className="max-w-3xl mx-auto">
            {PanelComponent && <PanelComponent {...panelProps} />}
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
