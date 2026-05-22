import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, FolderOpen } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PathItem } from './PathItem';
import { PathEditDialog } from './PathEditDialog';
import {
  llamacppPathsApi,
  modelPathsApi,
  vllmPathsApi,
  vllmOmniPathsApi,
  multimodalPathsApi,
} from '@/lib/api/paths';
import type {
  LlamaCppPathConfig,
  ModelPathConfig,
  BackendPathConfig,
  MultimodalPathConfig,
} from '@/lib/config';
import { toast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';

type PathType = 'llamacpp' | 'models' | 'vllm' | 'vllm_omni' | 'multimodal';
type AnyPathConfig = LlamaCppPathConfig | ModelPathConfig | BackendPathConfig | MultimodalPathConfig;

/**
 * Common interface for path API operations
 */
interface PathApi {
  list: () => Promise<{ success?: boolean; data?: { items?: AnyPathConfig[] }; error?: string }>;
  add: (data: AnyPathConfig) => Promise<{ success?: boolean; error?: string }>;
  update: (data: AnyPathConfig) => Promise<{ success?: boolean; error?: string }>;
  remove: (path: string) => Promise<{ success?: boolean; error?: string }>;
}

/**
 * Map path types to their corresponding API instances
 */
const PATH_API_MAP: Record<PathType, PathApi> = {
  llamacpp: llamacppPathsApi as unknown as PathApi,
  models: modelPathsApi as unknown as PathApi,
  vllm: vllmPathsApi as unknown as PathApi,
  vllm_omni: vllmOmniPathsApi as unknown as PathApi,
  multimodal: multimodalPathsApi as unknown as PathApi,
};

interface PathConfigPanelProps {
  type: PathType;
}

const PATH_META_KEYS = {
  llamacpp: {
    titleKey: 'settings.pathConfig.llamacpp.title',
    descriptionKey: 'settings.pathConfig.llamacpp.description',
  },
  models: {
    titleKey: 'settings.pathConfig.models.title',
    descriptionKey: 'settings.pathConfig.models.description',
  },
  vllm: {
    titleKey: 'settings.pathConfig.vllm.title',
    descriptionKey: 'settings.pathConfig.vllm.description',
  },
  vllm_omni: {
    titleKey: 'settings.pathConfig.vllm_omni.title',
    descriptionKey: 'settings.pathConfig.vllm_omni.description',
  },
  multimodal: {
    titleKey: 'settings.pathConfig.multimodal.title',
    descriptionKey: 'settings.pathConfig.multimodal.description',
  },
} as const satisfies Record<PathType, { titleKey: string; descriptionKey: string }>;

export function PathConfigPanel({ type }: PathConfigPanelProps) {
  const { t } = useTranslation();
  const alertDialog = useAlertDialog();

  const [paths, setPaths] = useState<AnyPathConfig[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingPath, setEditingPath] = useState<AnyPathConfig | undefined>(
    undefined
  );

  const loadPaths = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await PATH_API_MAP[type].list();
      if (response?.success) {
        setPaths(response.data?.items || []);
      }
    } catch (error) {
      console.error('Failed to load paths:', error);
    } finally {
      setIsLoading(false);
    }
  }, [type]);

  useEffect(() => {
    loadPaths();
  }, [loadPaths]);

  const handleAdd = async (data: AnyPathConfig) => {
    try {
      const response = await PATH_API_MAP[type].add(data);

      if (response?.success) {
        await loadPaths();
      } else {
        throw new Error(response?.error || t('settings.pathConfig.addFailed'));
      }
    } catch (error) {
      console.error('Failed to add path:', error);
      throw error;
    }
  };

  const handleUpdate = async (data: AnyPathConfig) => {
    try {
      const response = await PATH_API_MAP[type].update(data);

      if (response?.success) {
        await loadPaths();
      } else {
        throw new Error(response?.error || t('settings.pathConfig.updateFailed'));
      }
    } catch (error) {
      console.error('Failed to update path:', error);
      throw error;
    }
  };

  const handleRemove = async (path: AnyPathConfig) => {
    const confirmed = await alertDialog.confirm({
      title: t('settings.pathConfig.deletePath'),
      description: t('settings.pathConfig.deletePathConfirm', { name: path.name || path.path }),
      variant: 'destructive',
    });

    if (!confirmed) return;

    try {
      const response = await PATH_API_MAP[type].remove(path.path);

      if (response?.success) {
        await loadPaths();
        toast.success(t('settings.pathConfig.deleteSuccess'), t('settings.pathConfig.deleteSuccessDesc'));
      } else {
        throw new Error(response?.error || t('settings.pathConfig.deleteFailed'));
      }
    } catch (error) {
      console.error('Failed to delete path:', error);
      const message = error instanceof Error ? error.message : t('settings.pathConfig.deleteFailed');
      toast.error(t('settings.pathConfig.deleteFailed'), message);
    }
  };

  const handleTest = async (path: AnyPathConfig) => {
    if (type !== 'llamacpp' && type !== 'vllm' && type !== 'vllm_omni') return;

    try {
      let response;
      if (type === 'llamacpp') {
        response = await llamacppPathsApi.test(path.path);
      } else if (type === 'vllm') {
        response = await vllmPathsApi.test(path.path);
      } else {
        response = await vllmOmniPathsApi.test(path.path);
      }

      if (response.success && response.data?.valid) {
        toast.success(t('settings.pathConfig.testSuccess'), response.data.message || t('settings.pathConfig.testSuccessDesc'));
      } else {
        toast.error(t('settings.pathConfig.testFailed'), response.data?.error || t('common.unknownError'));
      }
    } catch (error) {
      console.error('Failed to test path:', error);
      toast.error(t('settings.pathConfig.testError'), t('settings.pathConfig.testErrorDesc'));
    }
  };

  const handleOpenAddDialog = () => {
    setEditingPath(undefined);
    setIsDialogOpen(true);
  };

  const handleOpenEditDialog = (path: AnyPathConfig) => {
    setEditingPath(path);
    setIsDialogOpen(true);
  };

  const { titleKey, descriptionKey } = PATH_META_KEYS[type];
  const title = t(titleKey);
  const description = t(descriptionKey);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold">{title}</h3>
          <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
        </div>
        <Button size="sm" onClick={handleOpenAddDialog} className="h-7 px-2.5 text-xs">
          <Plus size={14} className="mr-1" />
          {t('settings.pathConfig.addPath')}
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-6 text-xs text-muted-foreground">
          {t('settings.pathConfig.loading')}
        </div>
      ) : paths?.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-8 text-center border border-dashed rounded-lg">
          <FolderOpen size={36} className="text-muted-foreground mb-2" />
          <p className="text-xs text-muted-foreground">{t('settings.pathConfig.noPathsConfigured')}</p>
          <p className="text-xs text-muted-foreground mt-1">
            {t('settings.pathConfig.clickToAdd', { title })}
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {paths?.map((path, index) => (
            <PathItem
              key={`${path.path}-${index}`}
              path={path}
              onEdit={() => handleOpenEditDialog(path)}
              onRemove={() => handleRemove(path)}
              onTest={(type === 'llamacpp' || type === 'vllm' || type === 'vllm_omni') ? () => handleTest(path) : undefined}
            />
          ))}
        </div>
      )}

      <PathEditDialog
        open={isDialogOpen}
        type={type}
        path={editingPath}
        onSave={editingPath ? handleUpdate : handleAdd}
        onClose={() => setIsDialogOpen(false)}
      />
    </div>
  );
}
