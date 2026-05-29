import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Play, Loader2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useCreateBenchmarkV2, useBenchmarkV2History, useDeleteBenchmarkV2 } from '../hooks/useBenchmarkV2';
import { toast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';
import type { BenchmarkV2Record } from '@/types';

interface BenchmarkV2PanelProps {
  selectedModelId: string | undefined;
  isModelLoaded: boolean;
}

export function BenchmarkV2Panel({ selectedModelId, isModelLoaded }: BenchmarkV2PanelProps) {
  const { t } = useTranslation();
  const alertDialog = useAlertDialog();
  const [promptTokens, setPromptTokens] = useState(512);
  const [maxTokens, setMaxTokens] = useState(128);

  const createV2 = useCreateBenchmarkV2();
  const { data: records = [], isLoading } = useBenchmarkV2History(selectedModelId);
  const deleteRecord = useDeleteBenchmarkV2();

  const handleRun = () => {
    if (!selectedModelId) {
      toast.warning(t('benchmark.selectModel'));
      return;
    }
    if (!isModelLoaded) {
      toast.warning(t('benchmark.modelNotLoaded'));
      return;
    }

    createV2.mutate(
      { modelId: selectedModelId, promptTokens, maxTokens },
      {
        onSuccess: () => toast.success(t('benchmark.v2RunSuccess')),
        onError: (err) => toast.error(t('benchmark.v2RunFailed'), err.message),
      }
    );
  };

  const handleDelete = async (record: BenchmarkV2Record) => {
    if (record.lineNumber === undefined || !selectedModelId) return;
    const confirmed = await alertDialog.confirm({
      title: t('benchmark.deleteFile'),
      description: t('benchmark.deleteConfirm'),
      variant: 'destructive',
    });
    if (!confirmed) return;

    deleteRecord.mutate(
      { modelId: selectedModelId, lineNumber: record.lineNumber },
      {
        onSuccess: () => toast.success(t('benchmark.deleteSuccess')),
        onError: () => toast.error(t('benchmark.deleteFailed')),
      }
    );
  };

  const formatSpeed = (val: number | undefined): string => {
    if (val === undefined || val === null) return '-';
    return `${val.toFixed(2)} t/s`;
  };

  const formatMs = (val: number | undefined): string => {
    if (val === undefined || val === null) return '-';
    if (val >= 1000) return `${(val / 1000).toFixed(2)}s`;
    return `${val.toFixed(0)}ms`;
  };

  return (
    <div className="flex-1 flex flex-col min-h-0">
      {/* Controls */}
      <div className="p-3 border-b border-border bg-muted/30">
        <p className="text-xs text-muted-foreground mb-2">{t('benchmark.v2Desc')}</p>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5">
            <label className="text-xs text-muted-foreground whitespace-nowrap">{t('benchmark.promptTokens')}:</label>
            <Input
              type="number"
              value={promptTokens}
              onChange={(e) => setPromptTokens(Number(e.target.value) || 1)}
              className="h-7 w-24 text-xs"
              min={1}
            />
          </div>
          <div className="flex items-center gap-1.5">
            <label className="text-xs text-muted-foreground whitespace-nowrap">{t('benchmark.maxTokens')}:</label>
            <Input
              type="number"
              value={maxTokens}
              onChange={(e) => setMaxTokens(Number(e.target.value) || 1)}
              className="h-7 w-24 text-xs"
              min={1}
            />
          </div>
          <Button
            size="sm"
            onClick={handleRun}
            disabled={!selectedModelId || !isModelLoaded || createV2.isPending}
          >
            {createV2.isPending ? (
              <Loader2 className="w-4 h-4 mr-1.5 animate-spin" />
            ) : (
              <Play className="w-4 h-4 mr-1.5" />
            )}
            {t('benchmark.run')}
          </Button>
        </div>
        {!isModelLoaded && selectedModelId && (
          <p className="text-xs text-amber-600 dark:text-amber-400 mt-1.5">{t('benchmark.modelNotLoaded')}</p>
        )}
      </div>

      {/* Results table */}
      <div className="flex-1 overflow-auto">
        {isLoading ? (
          <div className="flex items-center justify-center p-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : records.length === 0 ? (
          <div className="flex items-center justify-center p-8 text-sm text-muted-foreground">
            {t('benchmark.v2NoRecords')}
          </div>
        ) : (
          <table className="w-full text-xs">
            <thead className="bg-muted/50 sticky top-0">
              <tr className="border-b border-border">
                <th className="px-3 py-2 text-left font-medium text-muted-foreground">Time</th>
                <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('benchmark.promptSpeed')}</th>
                <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('benchmark.genSpeed')}</th>
                <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('benchmark.promptTokens')}</th>
                <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('benchmark.maxTokens')}</th>
                <th className="px-3 py-2 text-right font-medium text-muted-foreground">{t('benchmark.totalTime')}</th>
                <th className="px-3 py-2 w-8"></th>
              </tr>
            </thead>
            <tbody>
              {[...records].reverse().map((record, idx) => {
                const promptMs = record.timings?.prompt_ms;
                const predictedMs = record.timings?.predicted_ms;
                const totalMs = (promptMs || 0) + (predictedMs || 0);
                return (
                  <tr key={idx} className="border-b border-border/50 hover:bg-muted/30">
                    <td className="px-3 py-1.5 text-muted-foreground">{record.timestamp}</td>
                    <td className="px-3 py-1.5 text-right font-mono">{formatSpeed(record.timings?.prompt_per_second)}</td>
                    <td className="px-3 py-1.5 text-right font-mono">{formatSpeed(record.timings?.predicted_per_second)}</td>
                    <td className="px-3 py-1.5 text-right">{record.timings?.prompt_n ?? record.promptTokens}</td>
                    <td className="px-3 py-1.5 text-right">{record.timings?.predicted_n ?? record.maxTokens}</td>
                    <td className="px-3 py-1.5 text-right font-mono">{formatMs(totalMs)}</td>
                    <td className="px-1 py-1.5">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 w-6 p-0 text-muted-foreground hover:text-destructive"
                        onClick={() => handleDelete(record)}
                      >
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
