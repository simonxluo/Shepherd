import { useState, useEffect, useCallback } from 'react';
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
import { useToast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';

type PathType = 'llamacpp' | 'models' | 'vllm' | 'vllm_omni' | 'multimodal';
type AnyPathConfig = LlamaCppPathConfig | ModelPathConfig | BackendPathConfig | MultimodalPathConfig;

interface PathConfigPanelProps {
  type: PathType;
}

const PATH_META: Record<PathType, { title: string; description: string }> = {
  llamacpp: {
    title: 'llama.cpp 路径',
    description: '配置 llama.cpp 可执行文件所在的目录',
  },
  models: {
    title: '模型目录',
    description: '配置用于扫描和管理 GGUF 模型文件的目录',
  },
  vllm: {
    title: 'vLLM 路径',
    description: '配置 vLLM 可执行文件所在的目录',
  },
  vllm_omni: {
    title: 'vLLM-Omni 路径',
    description: '配置 vLLM-Omni (多模态 vLLM) 可执行文件所在的目录',
  },
  multimodal: {
    title: '多模态模型目录',
    description: '配置用于扫描 safetensors/多模态模型 (TTS/ASR/图像生成) 的目录',
  },
};

export function PathConfigPanel({ type }: PathConfigPanelProps) {
  const toast = useToast();
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
      let response;
      switch (type) {
        case 'llamacpp':
          response = await llamacppPathsApi.list();
          break;
        case 'models':
          response = await modelPathsApi.list();
          break;
        case 'vllm':
          response = await vllmPathsApi.list();
          break;
        case 'vllm_omni':
          response = await vllmOmniPathsApi.list();
          break;
        case 'multimodal':
          response = await multimodalPathsApi.list();
          break;
      }

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
      let response;
      switch (type) {
        case 'llamacpp':
          response = await llamacppPathsApi.add(data as LlamaCppPathConfig);
          break;
        case 'models':
          response = await modelPathsApi.add(data as ModelPathConfig);
          break;
        case 'vllm':
          response = await vllmPathsApi.add(data as BackendPathConfig);
          break;
        case 'vllm_omni':
          response = await vllmOmniPathsApi.add(data as BackendPathConfig);
          break;
        case 'multimodal':
          response = await multimodalPathsApi.add(data as MultimodalPathConfig);
          break;
      }

      if (response?.success) {
        await loadPaths();
      } else {
        throw new Error(response?.error || '添加失败');
      }
    } catch (error) {
      console.error('Failed to add path:', error);
      throw error;
    }
  };

  const handleUpdate = async (data: AnyPathConfig) => {
    try {
      let response;
      switch (type) {
        case 'llamacpp':
          response = await llamacppPathsApi.update(data as LlamaCppPathConfig);
          break;
        case 'models':
          response = await modelPathsApi.update(data as ModelPathConfig);
          break;
        case 'multimodal':
          response = await multimodalPathsApi.update(data as MultimodalPathConfig);
          break;
        default:
          throw new Error('此路径类型不支持更新');
      }

      if (response?.success) {
        await loadPaths();
      } else {
        throw new Error(response?.error || '更新失败');
      }
    } catch (error) {
      console.error('Failed to update path:', error);
      throw error;
    }
  };

  const handleRemove = async (path: AnyPathConfig) => {
    const confirmed = await alertDialog.confirm({
      title: '删除路径',
      description: `确定要删除路径 "${path.name || path.path}" 吗？`,
      variant: 'destructive',
    });

    if (!confirmed) return;

    try {
      let response;
      switch (type) {
        case 'llamacpp':
          response = await llamacppPathsApi.remove(path.path);
          break;
        case 'models':
          response = await modelPathsApi.remove(path.path);
          break;
        case 'vllm':
          response = await vllmPathsApi.remove(path.path);
          break;
        case 'vllm_omni':
          response = await vllmOmniPathsApi.remove(path.path);
          break;
        case 'multimodal':
          response = await multimodalPathsApi.remove(path.path);
          break;
      }

      if (response?.success) {
        await loadPaths();
        toast.success('删除成功', '路径已成功删除');
      } else {
        throw new Error(response?.error || '删除失败');
      }
    } catch (error) {
      console.error('Failed to delete path:', error);
      const message = error instanceof Error ? error.message : '删除失败';
      toast.error('删除失败', message);
    }
  };

  const handleTest = async (path: AnyPathConfig) => {
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

  const handleOpenEditDialog = (path: AnyPathConfig) => {
    setEditingPath(path);
    setIsDialogOpen(true);
  };

  const { title, description } = PATH_META[type];

  return (
    <div className="space-y-3">
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
