import { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { FolderOpen, CheckCircle2, XCircle, Loader2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { DirectoryBrowser } from '@/features/settings/components/DirectoryBrowser';
import { cn } from '@/lib/utils';
import type { LlamaCppPathConfig, ModelPathConfig, BackendPathConfig, MultimodalPathConfig } from '@/lib/config';
import { toast } from '@/hooks/useToast';
import { llamacppPathsApi, vllmPathsApi, vllmOmniPathsApi } from '@/lib/api/paths';

type AnyPathConfig = LlamaCppPathConfig | ModelPathConfig | BackendPathConfig | MultimodalPathConfig;

interface PathEditDialogProps {
  open: boolean;
  type: 'llamacpp' | 'models' | 'vllm' | 'vllmomni' | 'multimodal';
  path?: AnyPathConfig;
  onSave: (path: AnyPathConfig) => Promise<void>;
  onClose: () => void;
}

export function PathEditDialog({
  open,
  type,
  path,
  onSave,
  onClose,
}: PathEditDialogProps) {
  const { t } = useTranslation();
  const isEdit = !!path;
  const typeLabel = type === 'llamacpp' ? t('settings.pathEdit.typeLlamacpp')
    : type === 'models' ? t('settings.pathEdit.typeModels')
    : type === 'vllm' ? t('settings.pathEdit.typeVllm')
    : type === 'vllmomni' ? t('settings.pathEdit.typeVllmOmni')
    : t('settings.pathEdit.typeMultimodal');

  const [formData, setFormData] = useState({
    name: '',
    path: '',
    description: '',
  });
  const [isSaving, setIsSaving] = useState(false);
  const [isBrowserOpen, setIsBrowserOpen] = useState(false);
  const [pathValidation, setPathValidation] = useState<{
    valid: boolean | null;
    checking: boolean;
    message?: string;
  }>({ valid: null, checking: false });
  const [submitError, setSubmitError] = useState<string | null>(null);
  const validationTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (path) {
      setFormData({
        name: path.name || '',
        path: path.path || '',
        description: path.description || '',
      });
    } else {
      setFormData({
        name: '',
        path: '',
        description: '',
      });
    }
  }, [path, open]);

  // Path validation via backend API
  useEffect(() => {
    if (validationTimeoutRef.current) {
      clearTimeout(validationTimeoutRef.current);
    }

    if (!formData.path) {
      setPathValidation({ valid: null, checking: false });
      return;
    }

    const isUnixPath = formData.path.startsWith('/');
    const isWindowsPath = /^[a-zA-Z]:\\/.test(formData.path);

    if (!isUnixPath && !isWindowsPath) {
      setPathValidation({
        valid: false,
        checking: false,
        message: t('settings.pathEdit.enterAbsolutePath'),
      });
      return;
    }

    const validatePath = async () => {
      setPathValidation({ valid: null, checking: true, message: t('settings.pathEdit.validatingPath') });

      try {
        let response;
        if (type === 'llamacpp') {
          response = await llamacppPathsApi.test(formData.path);
        } else if (type === 'vllm') {
          response = await vllmPathsApi.test(formData.path);
        } else if (type === 'vllmomni') {
          response = await vllmOmniPathsApi.test(formData.path);
        } else {
          // Model and multimodal paths are directories - basic format check is enough
          setPathValidation({
            valid: true,
            checking: false,
            message: t('settings.pathEdit.pathFormatValid'),
          });
          return;
        }

        if (response.success && response.data?.valid) {
          setPathValidation({
            valid: true,
            checking: false,
            message: response.data.message || t('settings.pathEdit.pathValid'),
          });
        } else {
          setPathValidation({
            valid: false,
            checking: false,
            message: response.data?.error || t('settings.pathEdit.pathInvalid'),
          });
        }
      } catch (error) {
        console.error('Path validation failed:', error);
        setPathValidation({
          valid: null,
          checking: false,
          message: t('settings.pathEdit.cannotValidatePath'),
        });
      }
    };

    // 500ms debounce
    validationTimeoutRef.current = setTimeout(validatePath, 500);

    return () => {
      if (validationTimeoutRef.current) {
        clearTimeout(validationTimeoutRef.current);
      }
    };
  }, [formData.path, type, t]);

  const handleDirectorySelect = (selectedPath: string) => {
    setFormData({ ...formData, path: selectedPath });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    setSubmitError(null);

    if (!formData.path.trim()) {
      return;
    }

    setIsSaving(true);
    try {
      const dataToSave = {
        ...(isEdit && { originalPath: path?.path }),
        path: formData.path.trim(),
        name: formData.name.trim() || undefined,
        description: formData.description.trim() || undefined,
      };

      await onSave(dataToSave);
      onClose();
    } catch (error) {
      console.error('Failed to save path:', error);

      let errorMessage = t('settings.pathEdit.saveFailedGeneric');

      if (error instanceof Error) {
        const errorText = error.message;

        if (errorText.includes('path does not exist')) {
          const match = errorText.match(/path does not exist: (.+)/);
          const invalidPath = match ? match[1] : formData.path;
          errorMessage = t('settings.pathEdit.pathNotExist', { path: invalidPath });
        } else if (errorText.includes('not a directory')) {
          errorMessage = t('settings.pathEdit.pathNotDirectory');
        } else if (errorText.includes('already exists')) {
          errorMessage = t('settings.pathEdit.pathAlreadyExists');
        } else if (errorText.includes('Invalid path')) {
          errorMessage = errorText.replace('Invalid path: ', '');
        } else {
          errorMessage = errorText;
        }
      } else if (typeof error === 'string') {
        errorMessage = error;
      }

      setSubmitError(errorMessage);
      toast.error(t('settings.pathEdit.saveFailedTitle'), errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  const actionLabel = isEdit ? t('settings.pathEdit.actionEdit') : t('settings.pathEdit.actionAdd');

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[480px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle className="text-base">
              {t('settings.pathEdit.dialogTitle', { action: actionLabel, type: typeLabel })}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 p-6">
            {/* Error alert */}
            {submitError && (
              <Alert variant="destructive">
                <XCircle className="h-4 w-4" />
                <AlertTitle>{t('settings.pathEdit.saveFailedTitle')}</AlertTitle>
                <AlertDescription>{submitError}</AlertDescription>
              </Alert>
            )}

            {/* Path input (required) */}
            <div className="space-y-2">
              <label htmlFor="path" className="text-xs font-medium flex items-center gap-1.5">
                <FolderOpen size={12} className="text-muted-foreground" />
                {t('settings.pathEdit.pathLabel')} <span className="text-destructive">*</span>
              </label>
              <div className="flex gap-2">
                <div className="flex-1 relative">
                  <Input
                    id="path"
                    type="text"
                    value={formData.path}
                    onChange={(e) =>
                      setFormData({ ...formData, path: e.target.value })
                    }
                    placeholder={
                      type === 'llamacpp' ? t('settings.pathEdit.placeholderLlamacpp')
                      : type === 'models' ? t('settings.pathEdit.placeholderModels')
                      : type === 'multimodal' ? t('settings.pathEdit.placeholderMultimodal')
                      : type === 'vllm' ? t('settings.pathEdit.placeholderVllm')
                      : t('settings.pathEdit.placeholderVllmOmni')
                    }
                    className={cn(
                      "w-full rounded-md border border-input bg-background px-3 py-2 pr-8 text-sm",
                      "focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent transition-all",
                      pathValidation.checking && "opacity-50"
                    )}
                    required
                  />
                  {/* Validation status icon */}
                  <div className="absolute right-2 top-1/2 -translate-y-1/2">
                    {pathValidation.checking ? (
                      <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
                    ) : formData.path && pathValidation.valid === true ? (
                      <CheckCircle2 className="w-4 h-4 text-green-500" />
                    ) : formData.path && pathValidation.valid === false ? (
                      <XCircle className="w-4 h-4 text-red-500" />
                    ) : null}
                  </div>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setIsBrowserOpen(true)}
                  className="h-9 px-3"
                >
                  <FolderOpen className="w-4 h-4 mr-1" />
                  {t('settings.pathEdit.browse')}
                </Button>
              </div>
              <div className="flex items-center justify-between">
                <p className="text-[11px] text-muted-foreground">
                  {type === 'llamacpp'
                    ? t('settings.pathEdit.pathHintLlamacpp')
                    : type === 'multimodal'
                    ? t('settings.pathEdit.pathHintMultimodal')
                    : type === 'models'
                    ? t('settings.pathEdit.pathHintModels')
                    : type === 'vllm'
                    ? t('settings.pathEdit.pathHintVllm')
                    : t('settings.pathEdit.pathHintVllmOmni')}
                </p>
                {pathValidation.message && (
                  <p className={cn(
                    "text-[11px]",
                    pathValidation.valid === true ? "text-green-600" :
                    pathValidation.valid === false ? "text-red-600" :
                    "text-muted-foreground"
                  )}>
                    {pathValidation.message}
                  </p>
                )}
              </div>
            </div>

            {/* Name input (optional) */}
            <div className="space-y-2">
              <label htmlFor="name" className="text-xs font-medium">
                {t('settings.pathEdit.nameLabel')} <span className="text-muted-foreground">{t('settings.pathEdit.optional')}</span>
              </label>
              <Input
                id="name"
                type="text"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder={
                  type === 'llamacpp'
                    ? t('settings.pathEdit.namePlaceholderLlamacpp')
                    : t('settings.pathEdit.namePlaceholderModels')
                }
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent transition-all"
              />
            </div>

            {/* Description input (optional) */}
            <div className="space-y-2">
              <label htmlFor="description" className="text-xs font-medium">
                {t('settings.pathEdit.descriptionLabel')} <span className="text-muted-foreground">{t('settings.pathEdit.optional')}</span>
              </label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
                placeholder={t('settings.pathEdit.descriptionPlaceholder')}
                rows={2}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent resize-none transition-all"
              />
            </div>
          </div>

          {/* Directory browser */}
          <DirectoryBrowser
            open={isBrowserOpen}
            initialPath={formData.path}
            allowFileSelection={type === 'llamacpp' || type === 'vllm' || type === 'vllmomni'}
            onSelect={handleDirectorySelect}
            onClose={() => setIsBrowserOpen(false)}
          />

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={onClose}
              disabled={isSaving}
              className="h-8 px-3 text-xs"
            >
              {t('common.cancel')}
            </Button>
            <Button
              type="submit"
              disabled={isSaving || !formData.path.trim()}
              className="h-8 px-3 text-xs"
            >
              {isSaving ? t('models.editAlias.saving') : t('common.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
