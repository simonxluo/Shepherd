import { useState, useEffect } from 'react';
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

  // 加载路径列表
  const loadPaths = async () => {
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
      console.error('加载路径失败:', error);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadPaths();
  }, [type]);

  // 添加路径
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
      console.error('添加路径失败:', error);
      throw error;
    }
  };

  // 更新路径
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
      console.error('更新路径失败:', error);
      throw error;
    }
  };

  // 删除路径
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
      console.error('删除路径失败:', error);
      const message = error instanceof Error ? error.message : '删除失败';
      toast.error('删除失败', message);
    }
  };

  // 测试路径
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
      console.error('测试路径失败:', error);
      toast.error('测试错误', '测试路径时发生错误');
    }
  };

  // 打开添加对话框
  const handleOpenAddDialog = () => {
    setEditingPath(undefined);
    setIsDialogOpen(true);
  };

  // 打开编辑对话框
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
      {/* 标题和添加按钮 */}
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

      {/* 路径列表 */}
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

      {/* 编辑对话框 */}
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
