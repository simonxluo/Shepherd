import { useState, useEffect, useRef } from 'react';
import { FolderOpen, CheckCircle2, XCircle, Loader2, AlertCircle } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { DirectoryBrowser } from '@/features/settings/components/DirectoryBrowser';
import { cn } from '@/lib/utils';
import type { LlamaCppPathConfig, ModelPathConfig, BackendPathConfig, MultimodalPathConfig } from '@/lib/config';
import { useToast } from '@/hooks/useToast';
import { llamacppPathsApi } from '@/lib/api/paths';

type AnyPathConfig = LlamaCppPathConfig | ModelPathConfig | BackendPathConfig | MultimodalPathConfig;

interface PathEditDialogProps {
  open: boolean;
  type: 'llamacpp' | 'models' | 'vllm' | 'vllm_omni' | 'multimodal';
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
  const toast = useToast();
  const isEdit = !!path;
  const typeLabel = type === 'llamacpp' ? 'Llama.cpp'
    : type === 'models' ? '模型'
    : type === 'vllm' ? 'vLLM'
    : type === 'vllm_omni' ? 'vLLM-Omni'
    : '多模态模型';

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
        message: '请输入绝对路径',
      });
      return;
    }

    const validatePath = async () => {
      setPathValidation({ valid: null, checking: true, message: '验证路径中...' });

      try {
        let response;
        if (type === 'llamacpp') {
          response = await llamacppPathsApi.test(formData.path);
        } else {
          setPathValidation({
            valid: true,
            checking: false,
            message: '路径格式有效',
          });
          return;
        }

        if (response.success && response.data?.valid) {
          setPathValidation({
            valid: true,
            checking: false,
            message: '路径有效',
          });
        } else {
          setPathValidation({
            valid: false,
            checking: false,
            message: response.data?.error || '路径无效',
          });
        }
      } catch (error) {
        console.error('Path validation failed:', error);
        setPathValidation({
          valid: null,
          checking: false,
          message: '无法验证路径',
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
  }, [formData.path, type]);

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

      let errorMessage = '保存失败，请稍后重试';

      if (error instanceof Error) {
        const errorText = error.message;

        if (errorText.includes('path does not exist')) {
          const match = errorText.match(/path does not exist: (.+)/);
          const invalidPath = match ? match[1] : formData.path;
          errorMessage = `路径不存在: ${invalidPath}`;
        } else if (errorText.includes('not a directory')) {
          errorMessage = '路径不是一个目录';
        } else if (errorText.includes('already exists')) {
          errorMessage = '该路径已存在';
        } else if (errorText.includes('Invalid path')) {
          errorMessage = errorText.replace('Invalid path: ', '');
        } else {
          errorMessage = errorText;
        }
      } else if (typeof error === 'string') {
        errorMessage = error;
      }

      setSubmitError(errorMessage);
      toast.error('保存失败', errorMessage);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[480px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle className="text-base">
              {isEdit ? '编辑' : '添加'} {typeLabel} 路径
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 p-6">
            {/* Error alert */}
            {submitError && (
              <div className="flex items-start gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-md">
                <AlertCircle className="w-4 h-4 text-destructive mt-0.5 flex-shrink-0" />
                <div className="flex-1">
                  <p className="text-sm text-destructive font-medium">保存失败</p>
                  <p className="text-xs text-destructive/80 mt-1">{submitError}</p>
                </div>
              </div>
            )}

            {/* Path input (required) */}
            <div className="space-y-2">
              <label htmlFor="path" className="text-xs font-medium flex items-center gap-1.5">
                <FolderOpen size={12} className="text-muted-foreground" />
                路径 <span className="text-destructive">*</span>
              </label>
              <div className="flex gap-2">
                <div className="flex-1 relative">
                  <input
                    id="path"
                    type="text"
                    value={formData.path}
                    onChange={(e) =>
                      setFormData({ ...formData, path: e.target.value })
                    }
                    placeholder={
                      type === 'llamacpp' ? '/usr/local/bin/llama.cpp'
                      : type === 'models' ? '~/.cache/huggingface/hub'
                      : type === 'multimodal' ? './models/multimodal'
                      : '/usr/local/bin/'
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
                  浏览
                </Button>
              </div>
              <div className="flex items-center justify-between">
                <p className="text-[11px] text-muted-foreground">
                  {type === 'llamacpp'
                    ? 'llama.cpp 可执行文件所在目录的绝对路径'
                    : type === 'multimodal'
                    ? '包含 safetensors 多模态模型 (含 config.json) 的目录绝对路径'
                    : type === 'models'
                    ? '包含 GGUF 模型文件的目录绝对路径'
                    : `${typeLabel} 可执行文件所在目录的绝对路径`}
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
                名称 <span className="text-muted-foreground">（可选）</span>
              </label>
              <input
                id="name"
                type="text"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder={
                  type === 'llamacpp'
                    ? '例如：主构建目录'
                    : '例如：HuggingFace 缓存'
                }
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent transition-all"
              />
            </div>

            {/* Description input (optional) */}
            <div className="space-y-2">
              <label htmlFor="description" className="text-xs font-medium">
                描述 <span className="text-muted-foreground">（可选）</span>
              </label>
              <textarea
                id="description"
                value={formData.description}
                onChange={(e) =>
                  setFormData({ ...formData, description: e.target.value })
                }
                placeholder="添加备注信息..."
                rows={2}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent resize-none transition-all"
              />
            </div>
          </div>

          {/* Directory browser */}
          <DirectoryBrowser
            open={isBrowserOpen}
            initialPath={formData.path}
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
              取消
            </Button>
            <Button
              type="submit"
              disabled={isSaving || !formData.path.trim()}
              className="h-8 px-3 text-xs"
            >
              {isSaving ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
