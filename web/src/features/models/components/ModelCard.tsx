import { type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Star, Loader2, Play, Square, Info } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ModelIcon } from '@/features/models/components/ModelIcon';
import { cn, formatBytes } from '@/lib/utils';
import type { Model, ModelStatus, ModelCapabilities } from '@/types';

interface ModelCardProps {
  model: Model;
  capabilities?: ModelCapabilities;
  onLoad?: () => void;
  onUnload?: () => void;
  onToggleFavourite?: () => void;
  onShowDetail?: () => void;
  onEditAlias?: () => void;
  actions?: ReactNode;
}

const STATUS_LABEL_KEYS: Record<ModelStatus, string> = {
  stopped: 'models.statusLabels.stopped',
  loading: 'models.statusLabels.loading',
  running: 'models.statusLabels.running',
  unloading: 'models.statusLabels.unloading',
  error: 'models.statusLabels.error',
};

const CAPABILITY_BADGE_KEYS: Record<string, { labelKey: string; className: string }> = {
  thinking: { labelKey: 'models.capabilities.thinking', className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-500/20' },
  tools: { labelKey: 'models.capabilities.tools', className: 'bg-purple-500/10 text-purple-600 dark:text-purple-400 border-purple-500/20' },
  embedding: { labelKey: 'models.capabilities.embedding', className: 'bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20' },
  rerank: { labelKey: 'models.capabilities.rerank', className: 'bg-orange-500/10 text-orange-600 dark:text-orange-400 border-orange-500/20' },
  tts: { labelKey: 'models.capabilities.tts', className: 'bg-pink-500/10 text-pink-600 dark:text-pink-400 border-pink-500/20' },
  asr: { labelKey: 'models.capabilities.asr', className: 'bg-teal-500/10 text-teal-600 dark:text-teal-400 border-teal-500/20' },
  imageGeneration: { labelKey: 'models.capabilities.imageGeneration', className: 'bg-violet-500/10 text-violet-600 dark:text-violet-400 border-violet-500/20' },
  music: { labelKey: 'models.capabilities.music', className: 'bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 border-cyan-500/20' },
};

function hasAnyCapability(capabilities?: ModelCapabilities): boolean {
  if (!capabilities) return false;
  return !!(
    capabilities.thinking ||
    capabilities.tools ||
    capabilities.embedding ||
    capabilities.rerank ||
    capabilities.tts ||
    capabilities.asr ||
    capabilities.imageGeneration ||
    capabilities.music
  );
}

export function ModelCard({ model, capabilities, onLoad, onUnload, onToggleFavourite, onShowDetail, onEditAlias, actions }: ModelCardProps) {
  const { t } = useTranslation();
  const statusLabel = t(STATUS_LABEL_KEYS[model.status]);
  const isLoading = model.status === 'loading';
  const isLoaded = model.status === 'running';

  const quantizationLabel = model.metadata.fileTypeDescriptor || model.metadata.quantization || t('models.card.unknown');

  return (
    <div className="group flex items-center gap-4 px-4 py-4 bg-card hover:bg-accent/5 rounded-lg border border-border transition-all duration-200">
      <div className="flex items-center gap-3 flex-1 min-w-0">
        <button
          onClick={onToggleFavourite}
          className={cn(
            'shrink-0 p-1 rounded-full hover:bg-accent transition-colors',
            model.favourite && 'text-yellow-500'
          )}
        >
          <Star className={cn('w-5 h-5', model.favourite && 'fill-current')} />
        </button>

        <button
          onClick={onEditAlias}
          className={cn(
            'shrink-0 w-10 h-10 rounded-full flex items-center justify-center',
            'bg-muted hover:bg-accent transition-all duration-200',
            'relative overflow-hidden'
          )}
          title={t('models.card.editAlias')}
        >
          <ModelIcon
            architecture={model.metadata.architecture}
            className="w-5 h-5"
          />
        </button>

        <div className="flex-1 min-w-0">
          <h3 className="font-bold text-base text-foreground truncate">
            {model.alias || model.displayName || model.name}
          </h3>
          <div className="flex items-center gap-2 text-sm text-muted-foreground mt-0.5">
            <span className="truncate">{model.metadata.architecture}</span>
            <span className="shrink-0">|</span>
            <span className="truncate">{quantizationLabel}</span>
            <span className="shrink-0">|</span>
            <span className="shrink-0">{formatBytes(model.totalSize ?? model.size)}</span>
            {model.sourceType === 'huggingface' && (
              <>
                <span className="shrink-0">|</span>
                <span className="shrink-0 inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-500/10 text-blue-500">Safetensors</span>
              </>
            )}
          </div>
          {hasAnyCapability(capabilities) && (
            <div className="flex items-center gap-1 mt-1 flex-wrap">
              {Object.entries(CAPABILITY_BADGE_KEYS).map(([key, badge]) =>
                capabilities?.[key as keyof ModelCapabilities] ? (
                  <span
                    key={key}
                    className={cn(
                      'inline-flex items-center px-1.5 py-0 rounded text-[10px] font-medium border',
                      badge.className
                    )}
                  >
                    {t(badge.labelKey)}
                  </span>
                ) : null
              )}
            </div>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <div className={cn(
            'w-2 h-2 rounded-full',
            isLoaded ? 'bg-green-500' : 'bg-gray-400'
          )} />
          <span className="whitespace-nowrap">{statusLabel}</span>
        </div>

        {!isLoaded ? (
          <Button
            onClick={onLoad}
            disabled={isLoading}
            variant="default"
            className="min-w-[100px] h-9 px-4 justify-center rounded-lg"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span className="ml-1.5">{t('models.card.loadingBtn')}</span>
              </>
            ) : (
              <>
                <Play className="w-4 h-4" />
                <span className="ml-1.5">{t('models.card.startBtn')}</span>
              </>
            )}
          </Button>
        ) : (
          <Button
            onClick={onUnload}
            disabled={model.status === 'unloading'}
            variant="destructive"
            className="min-w-[100px] h-9 px-4 justify-center rounded-lg"
          >
            {model.status === 'unloading' ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span className="ml-1.5">{t('models.card.unloadingBtn')}</span>
              </>
            ) : (
              <>
                <Square className="w-4 h-4" />
                <span className="ml-1.5">{t('models.card.stopBtn')}</span>
              </>
            )}
          </Button>
        )}

        <div className="flex items-center gap-1 ml-1">
          <Button
            onClick={onShowDetail}
            variant="ghost"
            size="icon"
            title={t('models.card.modelDetail')}
            className="h-8 w-8 sm:h-9 sm:w-9"
          >
            <Info className="w-3 h-3 sm:w-4 sm:h-4" />
          </Button>
          {actions}
        </div>
      </div>
    </div>
  );
}
