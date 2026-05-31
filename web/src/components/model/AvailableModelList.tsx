import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Play } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { LoadModelDialog } from '@/features/models/components/LoadModelDialog';
import { useLoadModel } from '@/features/models';
import { formatBytes } from '@/lib/utils';
import { BACKEND_LABELS } from '@/lib/constants/model';
import type { Model } from '@/types/model';

interface AvailableModelListProps {
  models: Model[];
  emptyText: string;
  emptyHint: string;
}

export function AvailableModelList({ models, emptyText, emptyHint }: AvailableModelListProps) {
  const { t } = useTranslation();
  const loadModel = useLoadModel();
  const [dialogModel, setDialogModel] = useState<Model | null>(null);

  if (models.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground">
        <p>{emptyText}</p>
        <p className="text-sm mt-1">{emptyHint}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-muted-foreground">
        {t('creative.availableModels')}
      </h3>
      <div className="space-y-2">
        {models.map((model) => (
          <div
            key={model.id}
            className="flex items-center justify-between border rounded-lg px-4 py-3 hover:bg-accent/50 transition-colors"
          >
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">
                {model.alias || model.displayName || model.name}
              </p>
              <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
                <span>{formatBytes(model.totalSize || model.size)}</span>
                {model.backendType && (
                  <span>{BACKEND_LABELS[model.backendType] || model.backendType}</span>
                )}
                {model.metadata?.architecture && (
                  <span>{model.metadata.architecture}</span>
                )}
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setDialogModel(model)}
              className="ml-3 shrink-0"
            >
              <Play className="w-3.5 h-3.5 mr-1.5" />
              {t('creative.loadButton')}
            </Button>
          </div>
        ))}
      </div>

      {dialogModel && (
        <LoadModelDialog
          isOpen={!!dialogModel}
          onClose={() => setDialogModel(null)}
          onConfirm={(params) => {
            loadModel.mutate(params, {
              onSuccess: () => setDialogModel(null),
            });
          }}
          modelId={dialogModel.id}
          modelName={dialogModel.alias || dialogModel.displayName || dialogModel.name}
          modelPath={dialogModel.path}
          isLoading={loadModel.isPending}
          backendType={dialogModel.backendType}
        />
      )}
    </div>
  );
}
