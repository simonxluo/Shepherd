import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import { AlertTriangle } from 'lucide-react';
import { useModels } from '@/features/models';
import { ModelListPanel } from './ModelListPanel';
import { BenchmarkControlsPanel } from './BenchmarkControlsPanel';
import { BenchmarkParamsModal } from './BenchmarkParamsModal';
import { BenchmarkV2Panel } from './BenchmarkV2Panel';
import { HistoryPanel } from './HistoryPanel';
import { OutputPanel } from './OutputPanel';
import { BenchmarkTaskPanel } from './BenchmarkTaskPanel';
import {
  useBenchmarkState,
  useBenchmarkParams,
  useLlamaCppVersions,
  useBenchmarkHistory,
  useDeleteHistoryFile,
  useCreateBenchmarkTask,
  useBenchmarkTasks,
  useCancelBenchmarkTask,
} from '../hooks/useBenchmarkState';
import { buildBenchArgs } from '../lib/commandBuilder';
import { toast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';
import { cn } from '@/lib/utils';

type BenchmarkTab = 'v1' | 'v2';

export function BenchmarkPage() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const alertDialog = useAlertDialog();
  const [activeTab, setActiveTab] = useState<BenchmarkTab>('v1');

  // Data queries
  const { data: models = [] } = useModels();
  const { data: benchmarkParams = [] } = useBenchmarkParams();
  const { data: llamaCppVersions = [] } = useLlamaCppVersions();

  // State
  const state = useBenchmarkState(models);
  const { data: historyFiles = [], isLoading: historyLoading } = useBenchmarkHistory(state.selectedModelId);
  const deleteHistoryFile = useDeleteHistoryFile();
  const createBenchmark = useCreateBenchmarkTask();

  // Check if selected model is loaded
  const isModelLoaded = state.selectedModel?.isLoaded ?? false;

  // Task status
  const { data: tasks = [] } = useBenchmarkTasks();
  const cancelTask = useCancelBenchmarkTask();
  const hasRunningTask = tasks.some(t => t.status === 'running' || t.status === 'pending');

  // Initialize from URL params
  useEffect(() => {
    const modelParam = searchParams.get('model');
    if (modelParam && models.length > 0) {
      const found = models.find(m => m.id === modelParam);
      if (found) {
        state.setSelectedModelId(found.id);
      }
    }
  }, [searchParams, models]); // eslint-disable-line react-hooks/exhaustive-deps

  // Initialize param defaults when params load
  useEffect(() => {
    if (benchmarkParams.length > 0 && Object.keys(state.enabledMap).length === 0) {
      state.initializeParamDefaults(benchmarkParams);
    }
  }, [benchmarkParams]); // eslint-disable-line react-hooks/exhaustive-deps

  // Initialize llama.cpp path
  useEffect(() => {
    if (!state.llamaCppPath && llamaCppVersions.length > 0) {
      state.setLlamaCppPath(llamaCppVersions[0].path);
    }
  }, [llamaCppVersions]); // eslint-disable-line react-hooks/exhaustive-deps

  // Count enabled params
  const enabledParamsCount = Object.values(state.enabledMap).filter(Boolean).length;

  // Handlers
  const handleRunBenchmark = () => {
    if (!state.selectedModel) {
      toast.warning(t('benchmark.selectModel'));
      return;
    }
    if (!state.llamaCppPath) {
      toast.warning(t('benchmark.llamaCppVersion'));
      return;
    }

    const args = buildBenchArgs(
      state.selectedModel.path,
      benchmarkParams,
      state.enabledMap,
      state.valueMap,
      state.availableDevices,
      state.selectedDeviceIndices,
      state.mainGpu
    );

    createBenchmark.mutate(
      {
        modelId: state.selectedModel.id,
        llamaBinPath: state.llamaCppPath,
        cmd: args.join(' '),
        args,
      },
      {
        onSuccess: () => {
          toast.success(t('benchmark.runSuccess'));
        },
        onError: (error) => {
          toast.error(t('benchmark.runFailed'), error.message);
        },
      }
    );
  };

  const handleDeleteFile = async (fileName: string) => {
    const confirmed = await alertDialog.confirm({
      title: t('benchmark.deleteFile'),
      description: t('benchmark.deleteConfirm'),
      variant: 'destructive',
    });
    if (!confirmed) return;

    deleteHistoryFile.mutate(fileName, {
      onSuccess: () => {
        toast.success(t('benchmark.deleteSuccess'));
        if (state.selectedHistoryFile === fileName) {
          state.setSelectedHistoryFile(null);
          state.setOutputContent('');
        }
      },
      onError: () => {
        toast.error(t('benchmark.deleteFailed'));
      },
    });
  };

  const handleParamsConfirm = (enabledMap: Record<string, boolean>, valueMap: Record<string, string>) => {
    state.setEnabledMap(enabledMap);
    state.setValueMap(valueMap);
  };

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="px-4 py-3 border-b border-border">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold text-foreground">{t('benchmark.title')}</h1>
            <div className="flex items-center gap-2 mt-1.5 text-xs text-amber-600 dark:text-amber-400">
              <AlertTriangle className="w-3.5 h-3.5" />
              <span>{t('benchmark.warning')}</span>
            </div>
          </div>
          {/* Tab switcher */}
          <div className="flex items-center gap-1 bg-muted rounded-md p-0.5">
            <button
              onClick={() => setActiveTab('v1')}
              className={cn(
                'px-3 py-1 text-xs font-medium rounded transition-colors',
                activeTab === 'v1'
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {t('benchmark.v1')}
            </button>
            <button
              onClick={() => setActiveTab('v2')}
              className={cn(
                'px-3 py-1 text-xs font-medium rounded transition-colors',
                activeTab === 'v2'
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {t('benchmark.v2')}
            </button>
          </div>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex min-h-0">
        {/* Left: Model list */}
        <ModelListPanel
          models={models}
          selectedModelId={state.selectedModelId}
          onSelectModel={state.setSelectedModelId}
        />

        {/* Right: Tab content */}
        <div className="flex-1 flex flex-col min-w-0">
          {activeTab === 'v1' ? (
            <>
              {/* V1: Controls bar */}
              <BenchmarkControlsPanel
                llamaCppPath={state.llamaCppPath}
                llamaCppVersions={llamaCppVersions}
                onLlamaCppPathChange={state.setLlamaCppPath}
                onRunBenchmark={handleRunBenchmark}
                onCancelBenchmark={() => {
                  const running = tasks.find(t => t.status === 'running');
                  if (running) cancelTask.mutate(running.id);
                }}
                onOpenParams={() => state.setIsParamsModalOpen(true)}
                isRunning={createBenchmark.isPending || hasRunningTask}
                isDisabled={!state.selectedModel || !state.llamaCppPath || createBenchmark.isPending}
                enabledParamsCount={enabledParamsCount}
                totalParamsCount={benchmarkParams.length}
                availableDevices={state.availableDevices}
                selectedDeviceIndices={state.selectedDeviceIndices}
                mainGpu={state.mainGpu}
                onDeviceSelectionChange={state.setSelectedDeviceIndices}
                onMainGpuChange={state.setMainGpu}
              />

              {/* V1: Results area */}
              <div className="flex-1 flex min-h-0">
                {/* History panel */}
                <div className="w-56 flex-shrink-0 border-r border-border overflow-hidden flex flex-col">
                  {/* Task status panel */}
                  {tasks.length > 0 && (
                    <div className="border-b border-border">
                      <div className="px-3 py-2 border-b border-border text-xs font-medium text-muted-foreground uppercase tracking-wider">
                        {t('benchmark.tasks')} ({tasks.length})
                      </div>
                      <BenchmarkTaskPanel tasks={tasks} />
                    </div>
                  )}
                  <div className="px-3 py-2 border-b border-border text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    {t('benchmark.history')}
                  </div>
                  <div className="flex-1 overflow-y-auto">
                    <HistoryPanel
                      files={historyFiles}
                      isLoading={historyLoading}
                      selectedFile={state.selectedHistoryFile}
                      onSelectFile={state.loadHistoryFile}
                      onDeleteFile={handleDeleteFile}
                    />
                  </div>
                </div>

                {/* Output panel */}
                <OutputPanel
                  content={state.outputContent}
                  isLoading={state.isOutputLoading}
                  fileName={state.selectedHistoryFile}
                />
              </div>
            </>
          ) : (
            /* V2: Server test panel */
            <BenchmarkV2Panel
              selectedModelId={state.selectedModelId}
              isModelLoaded={isModelLoaded}
            />
          )}
        </div>
      </div>

      {/* Params modal */}
      <BenchmarkParamsModal
        isOpen={state.isParamsModalOpen}
        onClose={() => state.setIsParamsModalOpen(false)}
        params={benchmarkParams}
        enabledMap={state.enabledMap}
        valueMap={state.valueMap}
        onConfirm={handleParamsConfirm}
      />
    </div>
  );
}
