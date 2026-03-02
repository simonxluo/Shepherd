import { useState } from 'react';
import { Search, RefreshCw, Filter, Grid3X3, List, Pencil, MessageSquare, Gauge, FileText } from 'lucide-react';
import { useModels, useLoadModel, useUnloadModel, useSetModelFavourite, useUpdateModelAlias, useScanModels, useFilteredModels, useCreateBenchmark } from '@/features/models/hooks';
import { ModelCard } from '@/components/models/ModelCard';
import { LoadModelDialog } from '@/components/models/LoadModelDialog';
import { EditAliasDialog } from '@/components/models/EditAliasDialog';
import { TestModelDialog } from '@/components/models/TestModelDialog';
import { BenchmarkDialog } from '@/components/models/BenchmarkDialog';
import { BenchmarkResultsDialog } from '@/components/models/BenchmarkResultsDialog';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import type { Model, ModelStatus, BenchmarkConfig, LoadModelParams } from '@/types';
import { useAlertDialog } from '@/hooks/useAlertDialog';
import { useToast } from '@/hooks/useToast';

/**
 * 模型管理页面
 */
export function ModelsPage() {
  const alertDialog = useAlertDialog();
  const toast = useToast();
  const { data: models = [], isLoading } = useModels();
  const loadModel = useLoadModel();
  const unloadModel = useUnloadModel();
  const setFavourite = useSetModelFavourite();
  const updateAlias = useUpdateModelAlias();
  const scanModels = useScanModels();
  const createBenchmark = useCreateBenchmark();

  // UI 状态
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<ModelStatus | ''>('');
  const [favouriteFilter, setFavouriteFilter] = useState(false);
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  // 加载对话框状态
  const [dialogModel, setDialogModel] = useState<Model | null>(null);

  // 编辑别名对话框状态
  const [editAliasModel, setEditAliasModel] = useState<Model | null>(null);

  // 测试模型对话框状态
  const [testModel, setTestModel] = useState<Model | null>(null);

  // 压测对话框状态
  const [benchmarkModel, setBenchmarkModel] = useState<Model | null>(null);

  // 压测结果对话框状态
  const [benchmarkResultsModel, setBenchmarkResultsModel] = useState<Model | null>(null);

  // 过滤模型
  const filteredModels = useFilteredModels(models, {
    search,
    status: statusFilter || undefined,
    favourite: favouriteFilter || undefined,
  });

  // 处理加载模型
  const handleLoadClick = (model: Model) => {
    setDialogModel(model);
  };

  const handleLoadConfirm = (params: LoadModelParams) => {
    loadModel.mutate(params, {
      onSuccess: () => {
        setDialogModel(null);
      },
    });
  };

  // 处理卸载模型
  const handleUnloadClick = async (modelId: string) => {
    const confirmed = await alertDialog.confirm({
      title: '卸载模型',
      description: '确定要卸载此模型吗？',
    });
    if (confirmed) {
      unloadModel.mutate(modelId);
    }
  };

  // 处理收藏切换
  const handleToggleFavourite = (modelId: string, favourite: boolean) => {
    setFavourite.mutate({ modelId, favourite: !favourite });
  };

  // 处理扫描
  const handleScan = () => {
    scanModels.mutate();
  };

  // 处理编辑别名
  const handleEditAlias = (model: Model) => {
    setEditAliasModel(model);
  };

  const handleAliasConfirm = (alias: string) => {
    if (editAliasModel) {
      updateAlias.mutate(
        { modelId: editAliasModel.id, alias },
        {
          onSuccess: () => {
            setEditAliasModel(null);
          },
        }
      );
    }
  };

  // 处理测试模型
  const handleTestModel = (model: Model) => {
    setTestModel(model);
  };

  // 处理压测模型
  const handleBenchmarkModel = (model: Model) => {
    setBenchmarkModel(model);
  };

  // 处理压测确认
  const handleBenchmarkConfirm = (config: BenchmarkConfig) => {
    // 构建命令字符串
    const cmdParts: string[] = [];
    Object.entries(config.params).forEach(([key, value]) => {
      if (value === 'true') {
        cmdParts.push(key);
      } else if (value !== 'false' && value !== '') {
        cmdParts.push(key, String(value));
      }
    });
    // 添加设备参数
    if (config.devices && config.devices.length > 0 && config.devices.length < 999) {
      cmdParts.push('-dev', config.devices.join('/'));
    }
    const cmd = cmdParts.join(' ');

    createBenchmark.mutate(
      {
        modelId: config.modelId,
        llamaBinPath: config.llamaCppPath,
        cmd,
      },
      {
        onSuccess: (data) => {
          if (data) {
            toast.success('压测任务已创建', '正在运行...');
            setBenchmarkModel(null);
            // 打开结果查看对话框
            const currentModel = models.find(m => m.id === config.modelId);
            if (currentModel) {
              setBenchmarkResultsModel(currentModel);
            }
          }
        },
        onError: (error) => {
          toast.error('创建压测任务失败', error.message);
        },
      }
    );
  };

  // 处理查看压测结果
  const handleViewBenchmarkResults = (model: Model) => {
    setBenchmarkResultsModel(model);
  };

  return (
    <div className="space-y-6">
      {/* 标题和操作 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">模型管理</h1>
          <p className="text-muted-foreground">
            共 {models?.length || 0} 个模型，已加载 {models?.filter((m) => m.isLoaded).length || 0} 个
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button
            onClick={handleScan}
            disabled={scanModels.isPending}
            variant="default"
            size="sm"
          >
            <RefreshCw className={cn('w-4 h-4', scanModels.isPending && 'animate-spin')} />
            扫描模型
          </Button>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="flex flex-wrap items-center gap-3 p-4 bg-card rounded-lg border border-border">
        {/* 搜索框 */}
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="搜索模型名称、架构..."
            className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-input text-foreground placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        {/* 状态过滤 */}
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as ModelStatus | '')}
          className="px-3 py-2 border border-border rounded-lg bg-input text-foreground"
        >
          <option value="">所有状态</option>
          <option value="running">运行中</option>
          <option value="stopped">已停止</option>
          <option value="loading">加载中</option>
          <option value="error">错误</option>
        </select>

        {/* 收藏过滤 */}
        <Button
          onClick={() => setFavouriteFilter(!favouriteFilter)}
          variant={favouriteFilter ? 'default' : 'outline'}
          size="sm"
          className={cn(
            favouriteFilter && 'border-yellow-500 bg-yellow-500 text-white hover:bg-yellow-600 dark:bg-yellow-600 dark:hover:bg-yellow-700 dark:border-yellow-700'
          )}
        >
          <Filter className="w-4 h-4" />
          只看收藏
        </Button>

        {/* 视图切换 */}
        <div className="flex items-center rounded-lg overflow-hidden border border-border/50">
          <Button
            onClick={() => setViewMode('grid')}
            variant={viewMode === 'grid' ? 'default' : 'ghost'}
            size="sm"
            className="rounded-none border-0"
          >
            <Grid3X3 className="w-4 h-4" />
          </Button>
          <Button
            onClick={() => setViewMode('list')}
            variant={viewMode === 'list' ? 'default' : 'ghost'}
            size="sm"
            className="rounded-none border-0"
          >
            <List className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* 模型列表 */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="text-center">
            <RefreshCw className="w-8 h-8 animate-spin text-blue-600 mx-auto mb-2" />
            <p className="text-muted-foreground">加载中...</p>
          </div>
        </div>
      ) : filteredModels.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <p className="text-lg mb-2">没有找到模型</p>
          <p className="text-sm">尝试调整搜索条件或扫描模型目录</p>
        </div>
      ) : (
        <div
          className={cn(
            'gap-3 sm:gap-4',
            viewMode === 'grid'
              ? 'grid grid-cols-1 sm:grid-cols-2'
              : 'flex flex-col'
          )}
        >
          {filteredModels.map((model) => (
            <ModelCard
              key={model.id}
              model={model}
              onLoad={() => handleLoadClick(model)}
              onUnload={() => handleUnloadClick(model.id)}
              onToggleFavourite={() => handleToggleFavourite(model.id, model.favourite)}
              actions={
                <>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleEditAlias(model)}
                    title="编辑别名"
                    className="h-8 w-8 sm:h-9 sm:w-9"
                  >
                    <Pencil className="w-3 h-3 sm:w-4 sm:h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleTestModel(model)}
                    title="测试模型"
                    className="h-8 w-8 sm:h-9 sm:w-9"
                  >
                    <MessageSquare className="w-3 h-3 sm:w-4 sm:h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleBenchmarkModel(model)}
                    title="性能测试"
                    className="h-8 w-8 sm:h-9 sm:w-9"
                  >
                    <Gauge className="w-3 h-3 sm:w-4 sm:h-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleViewBenchmarkResults(model)}
                    title="查看测试结果"
                    className="h-8 w-8 sm:h-9 sm:w-9"
                  >
                    <FileText className="w-3 h-3 sm:w-4 sm:h-4" />
                  </Button>
                </>
              }
            />
          ))}
        </div>
      )}

      {/* 加载对话框 */}
      {dialogModel && (
        <LoadModelDialog
          isOpen={!!dialogModel}
          onClose={() => setDialogModel(null)}
          onConfirm={handleLoadConfirm}
          modelId={dialogModel.id}
          modelName={dialogModel.alias || dialogModel.name}
          isLoading={loadModel.isPending}
        />
      )}

      {/* Edit Alias Dialog */}
      {editAliasModel && (
        <EditAliasDialog
          isOpen={!!editAliasModel}
          onClose={() => setEditAliasModel(null)}
          onConfirm={handleAliasConfirm}
          modelId={editAliasModel.id}
          modelName={editAliasModel.name}
          currentAlias={editAliasModel.alias}
          isLoading={updateAlias.isPending}
        />
      )}

      {/* Test Model Dialog */}
      {testModel && (
        <TestModelDialog
          isOpen={!!testModel}
          onClose={() => setTestModel(null)}
          modelId={testModel.id}
          modelName={testModel.alias || testModel.name}
          isModelLoaded={testModel.isLoaded}
          onLoadModel={() => {
            setTestModel(null);
            handleLoadClick(testModel);
          }}
        />
      )}

      {/* Benchmark Dialog */}
      {benchmarkModel && (
        <BenchmarkDialog
          isOpen={!!benchmarkModel}
          onClose={() => setBenchmarkModel(null)}
          onConfirm={handleBenchmarkConfirm}
          modelId={benchmarkModel.id}
          modelName={benchmarkModel.alias || benchmarkModel.name}
          isLoading={createBenchmark.isPending}
        />
      )}

      {/* Benchmark Results Dialog */}
      {benchmarkResultsModel && (
        <BenchmarkResultsDialog
          isOpen={!!benchmarkResultsModel}
          onClose={() => setBenchmarkResultsModel(null)}
          modelId={benchmarkResultsModel.id}
          modelName={benchmarkResultsModel.alias || benchmarkResultsModel.name}
        />
      )}
    </div>
  );
}
