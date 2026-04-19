import { useState, useEffect, useCallback } from 'react';
import { Plus, FolderOpen } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PathItem } from './PathItem';
import { PathEditDialog } from './PathEditDialog';
import { llamacppPathsApi, modelPathsApi } from '@/lib/api/paths';
import type { LlamaCppPathConfig, ModelPathConfig } from '@/lib/config';
import { useToast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';

interface PathConfigPanelProps {
  type: 'llamacpp' | 'models';
}

export function PathConfigPanel({ type }: PathConfigPanelProps) {
  const toast = useToast();
  const alertDialog = useAlertDialog();

  const [paths, setPaths] = useState<(LlamaCppPathConfig | ModelPathConfig)[]>(
    []
  );
  const [isLoading, setIsLoading] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingPath, setEditingPath] = useState<
    LlamaCppPathConfig | ModelPathConfig | undefined
  >(undefined);

  const loadPaths = useCallback(async () => {
    setIsLoading(true);
    try {
      const response =
        type === 'llamacpp'
          ? await llamacppPathsApi.list()
          : await modelPathsApi.list();

      if (response.success) {
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

  const handleAdd = async (data: LlamaCppPathConfig | ModelPathConfig) => {
    try {
      const response =
        type === 'llamacpp'
          ? await llamacppPathsApi.add(data as LlamaCppPathConfig)
          : await modelPathsApi.add(data as ModelPathConfig);

      if (response.success) {
        await loadPaths();
      } else {
        throw new Error(response.error || '添加失败');
      }
    } catch (error) {
      console.error('Failed to add path:', error);
      throw error;
    }
  };

  const handleUpdate = async (data: LlamaCppPathConfig | ModelPathConfig) => {
    try {
      const response =
        type === 'llamacpp'
          ? await llamacppPathsApi.update(data as LlamaCppPathConfig)
          : await modelPathsApi.update(data as ModelPathConfig);

      if (response.success) {
        await loadPaths();
      } else {
        throw new Error(response.error || '更新失败');
      }
    } catch (error) {
      console.error('Failed to update path:', error);
      throw error;
    }
  };

  const handleRemove = async (path: LlamaCppPathConfig | ModelPathConfig) => {
    const confirmed = await alertDialog.confirm({
      title: '删除路径',
      description: `确定要删除路径 "${path.name || path.path}" 吗？`,
      variant: 'destructive',
    });

    if (!confirmed) return;

    try {
      const response =
        type === 'llamacpp'
          ? await llamacppPathsApi.remove(path.path)
          : await modelPathsApi.remove(path.path);

      if (response.success) {
        await loadPaths();
        toast.success('删除成功', '路径已成功删除');
      } else {
        throw new Error(response.error || '删除失败');
      }
    } catch (error) {
      console.error('Failed to delete path:', error);
      const message = error instanceof Error ? error.message : '删除失败';
      toast.error('删除失败', message);
    }
  };

  const handleTest = async (path: LlamaCppPathConfig | ModelPathConfig) => {
    if (type !== 'llamacpp') return;

    try {
      const response = await llamacppPathsApi.test(path.path);

      if (response.success && response.data?.valid) {
        toast.success('测试成功', response.data.message || 'llama.cpp 路径测试通过');
      } else {
        toast.error('测试失败', response.data?.error || '未知错误');
      }
    } catch (error) {
      console.error('Failed to test path:', error);
      toast.error('测试错误', '测试路径时发生错误');
    }
  };

  const handleOpenAddDialog = () => {
    setEditingPath(undefined);
    setIsDialogOpen(true);
  };

  const handleOpenEditDialog = (path: LlamaCppPathConfig | ModelPathConfig) => {
    setEditingPath(path);
    setIsDialogOpen(true);
  };

  const title = type === 'llamacpp' ? 'llama.cpp 路径' : '模型目录';
  const description =
    type === 'llamacpp'
      ? '配置 llama.cpp 可执行文件所在的目录'
      : '配置用于扫描和管理模型文件的目录';

  return (
    <div className="space-y-3">
      {/* Title and add button */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold">{title}</h3>
          <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
        </div>
        <Button size="sm" onClick={handleOpenAddDialog} className="h-7 px-2.5 text-xs">
          <Plus size={14} className="mr-1" />
          添加路径
        </Button>
      </div>

      {/* Path list */}
      {isLoading ? (
        <div className="flex items-center justify-center py-6 text-xs text-muted-foreground">
          加载中...
        </div>
      ) : paths?.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-8 text-center border border-dashed rounded-lg">
          <FolderOpen size={36} className="text-muted-foreground mb-2" />
          <p className="text-xs text-muted-foreground">暂无配置的路径</p>
          <p className="text-xs text-muted-foreground mt-1">
            点击上方按钮添加{title}
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
              onTest={type === 'llamacpp' ? () => handleTest(path) : undefined}
            />
          ))}
        </div>
      )}

      {/* Edit dialog */}
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
