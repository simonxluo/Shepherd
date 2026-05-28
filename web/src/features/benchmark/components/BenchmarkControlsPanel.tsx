import { useTranslation } from 'react-i18next';
import { Play, Loader2, Settings2, XCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

interface BenchmarkControlsPanelProps {
  llamaCppPath: string;
  llamaCppVersions: Array<{ path: string; name?: string; description?: string }>;
  onLlamaCppPathChange: (path: string) => void;
  onRunBenchmark: () => void;
  onCancelBenchmark?: () => void;
  onOpenParams: () => void;
  isRunning: boolean;
  isDisabled: boolean;
  enabledParamsCount: number;
  totalParamsCount: number;
}

export function BenchmarkControlsPanel({
  llamaCppPath,
  llamaCppVersions,
  onLlamaCppPathChange,
  onRunBenchmark,
  onCancelBenchmark,
  onOpenParams,
  isRunning,
  isDisabled,
  enabledParamsCount,
  totalParamsCount,
}: BenchmarkControlsPanelProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-3 p-3 border-b border-border bg-muted/30">
      {/* Run / Cancel button */}
      {isRunning ? (
        <Button
          variant="destructive"
          size="sm"
          onClick={onCancelBenchmark}
          className="min-w-[90px]"
        >
          <XCircle className="w-4 h-4 mr-1.5" />
          {t('benchmark.cancel')}
        </Button>
      ) : (
        <Button
          variant="default"
          size="sm"
          onClick={onRunBenchmark}
          disabled={isDisabled}
          className="min-w-[90px]"
        >
          {isRunning ? (
            <Loader2 className="w-4 h-4 mr-1.5 animate-spin" />
          ) : (
            <Play className="w-4 h-4 mr-1.5" />
          )}
          {t('benchmark.run')}
        </Button>
      )}

      {/* Params button */}
      <Button
        variant="outline"
        size="sm"
        onClick={onOpenParams}
      >
        <Settings2 className="w-4 h-4 mr-1.5" />
        {t('benchmark.params')}
        <span className="ml-1.5 text-xs text-muted-foreground">
          ({enabledParamsCount}/{totalParamsCount})
        </span>
      </Button>

      {/* LlamaCpp version select */}
      <div className="flex-1 max-w-xs">
        <Select
          value={llamaCppPath || undefined}
          onValueChange={onLlamaCppPathChange}
        >
          <SelectTrigger className="h-8 text-sm">
            <SelectValue placeholder={t('benchmark.llamaCppVersion')} />
          </SelectTrigger>
          <SelectContent>
            {llamaCppVersions.filter((v) => v.path !== '').map((version) => (
              <SelectItem key={version.path} value={version.path}>
                {version.name || version.path}
                {version.description && ` (${version.description})`}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
