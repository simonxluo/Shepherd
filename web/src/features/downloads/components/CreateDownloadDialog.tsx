import { useState, useMemo, useEffect } from 'react';
import { X, Cloud, Database, Loader2, File, Check } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { useModelFiles } from '@/features/downloads/hooks';
import type { DownloadSource, CreateDownloadParams } from '@/types';
import type { ModelFileInfo } from '@/lib/api/downloads';

interface CreateDownloadDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (params: CreateDownloadParams) => void;
  isLoading?: boolean;
  preFill?: { source: DownloadSource; repoId: string; fileName?: string } | null;
}

export function CreateDownloadDialog({
  isOpen,
  onClose,
  onConfirm,
  isLoading = false,
  preFill = null,
}: CreateDownloadDialogProps) {
  const [source, setSource] = useState<DownloadSource>(() => preFill?.source || 'huggingface');
  const [repoId, setRepoId] = useState(() => preFill?.repoId || '');
  const [fileName, setFileName] = useState(() => preFill?.fileName || '');
  const [path, setPath] = useState('');
  const [maxRetries, setMaxRetries] = useState('3');
  const [chunkSize, setChunkSize] = useState('');

  const [showFileBrowser, setShowFileBrowser] = useState(false);

  useEffect(() => {
    if (isOpen) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSource(preFill?.source || 'huggingface');
      setRepoId(preFill?.repoId || '');
      setFileName(preFill?.fileName || '');
      setPath('');
      setMaxRetries('3');
      setChunkSize('');
      setShowFileBrowser(false);
    }
  }, [isOpen, preFill]);

  const { data: files, isLoading: loadingFiles, error: filesError } = useModelFiles(
    source,
    repoId
  );

  const availableFiles = useMemo(() => {
    if (!files) return [];
    return files.filter((f) => f.name.toLowerCase().endsWith('.gguf'));
  }, [files]);

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const params: CreateDownloadParams = {
      source,
      repoId: repoId.trim(),
      path: path || undefined,
      fileName: fileName || undefined,
      maxRetries: Number(maxRetries) || 3,
      chunkSize: chunkSize ? Number(chunkSize) : undefined,
    };

    onConfirm(params);
  };

  const handleSelectFile = (file: ModelFileInfo) => {
    setFileName(file.name);
    setShowFileBrowser(false);
  };

  const formatFileSize = (bytes: number): string => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;

    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }

    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <h2 className="text-lg font-semibold text-foreground">
            创建下载任务
          </h2>
          <button
            onClick={onClose}
            disabled={isLoading}
            className="p-1 text-muted-foreground hover:text-foreground disabled:opacity-50"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">
              下载来源
            </label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setSource('huggingface')}
                className={cn(
                  'flex-1 flex items-center justify-center gap-2 px-4 py-3 rounded-md border-2 transition-colors',
                  source === 'huggingface'
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-border hover:border-primary'
                )}
              >
                <Cloud className="w-5 h-5" />
                <span className="font-medium">HuggingFace</span>
              </button>
              <button
                type="button"
                onClick={() => setSource('modelscope')}
                className={cn(
                  'flex-1 flex items-center justify-center gap-2 px-4 py-3 rounded-md border-2 transition-colors',
                  source === 'modelscope'
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-border hover:border-primary'
                )}
              >
                <Database className="w-5 h-5" />
                <span className="font-medium">ModelScope</span>
              </button>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              仓库 ID <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={repoId}
              onChange={(e) => setRepoId(e.target.value)}
              placeholder={source === 'huggingface' ? 'Qwen/Qwen2-7B-Instruct' : 'Qwen/Qwen2-7B-Instruct'}
              className="w-full px-3 py-2 border border-border rounded-md bg-input text-foreground"
              required
            />
            <p className="text-xs text-muted-foreground mt-1">
              例如: {source === 'huggingface' ? 'Qwen/Qwen2-7B-Instruct' : 'Qwen/Qwen2-7B-Instruct'}
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              文件名
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                value={fileName}
                onChange={(e) => setFileName(e.target.value)}
                placeholder="选择或输入 GGUF 文件名"
                className="flex-1 px-3 py-2 border border-border rounded-md bg-input text-foreground"
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowFileBrowser(!showFileBrowser)}
                disabled={!repoId || loadingFiles}
                className="whitespace-nowrap"
              >
                {showFileBrowser ? '隐藏' : '浏览文件'}
              </Button>
            </div>

            {showFileBrowser && (
              <div className="mt-2 p-3 bg-muted rounded-md border border-border max-h-48 overflow-y-auto">
                {loadingFiles ? (
                  <div className="flex items-center justify-center py-4 text-sm text-muted-foreground">
                    <Loader2 className="w-4 h-4 animate-spin mr-2" />
                    加载中...
                  </div>
                ) : filesError ? (
                  <div className="text-sm text-red-600 dark:text-red-400 py-2">
                    加载失败: {filesError.message}
                  </div>
                ) : availableFiles.length === 0 ? (
                  <div className="text-sm text-muted-foreground py-2">
                    {repoId ? '该仓库没有 GGUF 文件' : '请先输入仓库 ID'}
                  </div>
                ) : (
                  <div className="space-y-1">
                    <p className="text-xs text-muted-foreground mb-2">
                      找到 {availableFiles.length} 个 GGUF 文件:
                    </p>
                    {availableFiles.map((file) => (
                      <div
                        key={file.name}
                        onClick={() => handleSelectFile(file)}
                        className={cn(
                          'flex items-center gap-2 p-2 rounded cursor-pointer transition-colors',
                          'hover:bg-blue-50 dark:hover:bg-blue-900/20 border border-transparent hover:border-blue-300',
                          fileName === file.name && 'bg-blue-100 dark:bg-blue-900/30 border-blue-500'
                        )}
                      >
                        <File className="w-4 h-4 flex-shrink-0 text-blue-600 dark:text-blue-400" />
                        <span className="flex-1 text-sm truncate">{file.name}</span>
                        <span className="text-xs text-muted-foreground">
                          {formatFileSize(file.size)}
                        </span>
                        {fileName === file.name && (
                          <Check className="w-4 h-4 text-green-600 dark:text-green-400" />
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              保存路径（可选）
            </label>
            <input
              type="text"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder="/path/to/models"
              className="w-full px-3 py-2 border border-border rounded-md bg-input text-foreground"
            />
            <p className="text-xs text-muted-foreground mt-1">
              留空则使用默认路径
            </p>
          </div>

          <details className="group">
            <summary className="cursor-pointer text-sm font-medium text-foreground list-none flex items-center gap-2">
              <span className="transform group-open:rotate-90 transition-transform">▶</span>
              高级选项
            </summary>

            <div className="mt-3 space-y-3 pl-5">
              <div>
                <label className="block text-sm font-medium text-foreground mb-1">
                  最大重试次数
                </label>
                <input
                  type="number"
                  value={maxRetries}
                  onChange={(e) => setMaxRetries(e.target.value)}
                  className="w-full px-3 py-2 border border-border rounded-md bg-input text-foreground"
                  min={0}
                  max={10}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-foreground mb-1">
                  分块大小（字节）
                </label>
                <input
                  type="number"
                  value={chunkSize}
                  onChange={(e) => setChunkSize(e.target.value)}
                  placeholder="默认: 10485760 (10MB)"
                  className="w-full px-3 py-2 border border-border rounded-md bg-input text-foreground"
                  min={1024}
                  step={1024}
                />
              </div>
            </div>
          </details>

          <div className="flex justify-end gap-3 pt-4 border-t border-border">
            <Button
              type="button"
              variant="outline"
              onClick={onClose}
              disabled={isLoading}
            >
              取消
            </Button>
            <Button
              type="submit"
              disabled={isLoading || !repoId.trim()}
              variant="default"
            >
              {isLoading ? '创建中...' : '创建任务'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
