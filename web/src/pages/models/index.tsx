import { useState, useMemo } from 'react';
import { Search, RefreshCw, Grid3X3, List, Gauge, FileText, Star, Layers, MessageSquare, Volume2, Ear, Image, Music, Database, ArrowUpDown } from 'lucide-react';
import { useModels, useLoadModel, useUnloadModel, useSetModelFavourite, useUpdateModelAlias, useScanModels, useFilteredModels, useCreateBenchmark, useAllModelCapabilities, MODEL_CATEGORIES, getModelCategory } from '@/features/models';
import type { ModelCategory } from '@/features/models';
import { ModelCard } from '@/features/models/components/ModelCard';
import { LoadModelDialog } from '@/features/models/components/LoadModelDialog';
import { EditAliasDialog } from '@/features/models/components/EditAliasDialog';
import { BenchmarkDialog } from '@/features/models/components/BenchmarkDialog';
import { BenchmarkResultsDialog } from '@/features/models/components/BenchmarkResultsDialog';
import { ModelDetailDialog } from '@/features/models/components/ModelDetailDialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import type { Model, ModelStatus, BenchmarkConfig, LoadModelParams, ModelCapabilities } from '@/types';
import { useAlertDialog } from '@/providers/AlertDialog';
import { toast } from '@/hooks/useToast';
import { APIError } from '@/lib/api/client';
import { useUIStore } from '@/stores/uiStore';

const CATEGORY_ICONS: Record<ModelCategory, React.ElementType> = {
  all: Layers,
  chat: MessageSquare,
  tts: Volume2,
  asr: Ear,
  image: Image,
  music: Music,
  embedding: Database,
  rerank: ArrowUpDown,
};

const STATUS_CHIPS: { value: ModelStatus | ''; label: string; dot?: string }[] = [
  { value: '', label: '全部' },
  { value: 'running', label: '运行中', dot: 'bg-green-500' },
  { value: 'stopped', label: '已停止', dot: 'bg-gray-400' },
  { value: 'loading', label: '加载中', dot: 'bg-amber-500' },
  { value: 'error', label: '错误', dot: 'bg-red-500' },
];

export function ModelsPage() {
  const alertDialog = useAlertDialog();
  const { data: models = [], isLoading } = useModels();
  const loadModel = useLoadModel();
  const unloadModel = useUnloadModel();
  const setFavourite = useSetModelFavourite();
  const updateAlias = useUpdateModelAlias();
  const scanModels = useScanModels();
  const createBenchmark = useCreateBenchmark();

  const [search, setSearch] = useState('');
  const [category, setCategory] = useState<ModelCategory>('all');
  const [statusFilter, setStatusFilter] = useState<ModelStatus | ''>('');
  const [favouriteFilter, setFavouriteFilter] = useState(false);
  const { modelViewMode: viewMode, setModelViewMode: setViewMode } = useUIStore();

  const [dialogModel, setDialogModel] = useState<Model | null>(null);
  const [editAliasModel, setEditAliasModel] = useState<Model | null>(null);
  const [benchmarkModel, setBenchmarkModel] = useState<Model | null>(null);
  const [benchmarkResultsModel, setBenchmarkResultsModel] = useState<Model | null>(null);
  const [detailModel, setDetailModel] = useState<Model | null>(null);

  const modelIds = useMemo(() => models.map(m => m.id), [models]);
  const capsResults = useAllModelCapabilities(modelIds);

  const capabilitiesMap = useMemo(() => {
    const map: Record<string, ModelCapabilities> = {};
    modelIds.forEach((id, i) => {
      if (capsResults[i]?.data) {
        map[id] = capsResults[i].data!;
      }
    });
    return map;
  }, [modelIds, capsResults]);

  const categoryCounts = useMemo(() => {
    const counts: Record<ModelCategory, number> = { all: 0, chat: 0, tts: 0, asr: 0, image: 0, music: 0, embedding: 0, rerank: 0 };
    models.forEach(m => {
      counts.all++;
      const cat = getModelCategory(capabilitiesMap[m.id]);
      counts[cat]++;
    });
    return counts;
  }, [models, capabilitiesMap]);

  const filteredModels = useFilteredModels(models, {
    search,
    status: statusFilter || undefined,
    favourite: favouriteFilter || undefined,
    category,
    capabilitiesMap,
  });

  const handleLoadClick = (model: Model) => setDialogModel(model);

  const handleLoadConfirm = (params: Partial<LoadModelParams>) => {
    loadModel.mutate(params, {
      onSuccess: () => {
        // 异步加载：API 返回 202 表示已接受，模型正在后台加载
        toast.info('模型正在加载', `${params.modelId} 已提交加载请求，请稍候查看状态`);
        setDialogModel(null);
      },
      onError: (error) => {
        (error as APIError).handled = true;
        toast.error('模型加载失败', error.message || '未知错误');
      },
    });
  };

  const handleUnloadClick = async (modelId: string) => {
    const confirmed = await alertDialog.confirm({
      title: '卸载模型',
      description: '确定要卸载此模型吗？',
    });
    if (confirmed) {
      unloadModel.mutate(modelId, {
        onSuccess: () => {
          toast.success('模型卸载成功', `${modelId} 已成功停止`);
        },
        onError: (error) => {
          (error as APIError).handled = true;
          toast.error('模型卸载失败', error.message || '未知错误');
        },
      });
    }
  };

  const handleToggleFavourite = (modelId: string, favourite: boolean) => {
    setFavourite.mutate(
      { modelId, favourite: !favourite },
      {
        onSuccess: () => {
          toast.success(!favourite ? '已添加到收藏' : '已取消收藏', modelId);
        },
        onError: (error) => {
          (error as APIError).handled = true;
          toast.error('操作失败', error.message || '未知错误');
        },
      }
    );
  };

  const handleScan = () => {
    scanModels.mutate(undefined, {
      onSuccess: (data) => {
        const message = data?.message || `扫描完成，找到 ${data?.models_found || 0} 个模型`;
        toast.success('模型扫描成功', message);
      },
      onError: (error) => {
        (error as APIError).handled = true;
        toast.error('模型扫描失败', error.message || '未知错误');
      },
    });
  };

  const handleEditAlias = (model: Model) => setEditAliasModel(model);

  const handleAliasConfirm = (alias: string) => {
    if (editAliasModel) {
      updateAlias.mutate(
        { modelId: editAliasModel.id, alias },
        {
          onSuccess: () => {
            toast.success('别名更新成功', `模型别名已设置为 ${alias || '（空）'}`);
            setEditAliasModel(null);
          },
          onError: (error) => {
            (error as APIError).handled = true;
            toast.error('别名更新失败', error.message || '未知错误');
          },
        }
      );
    }
  };

  const handleBenchmarkModel = (model: Model) => setBenchmarkModel(model);

  const handleBenchmarkConfirm = async (config: BenchmarkConfig) => {
    const model = models.find(m => m.id === config.modelId);
    if (!model) {
      toast.error('模型未找到', '无法获取模型信息');
      return;
    }

    const cmdParts: string[] = [];
    cmdParts.push('-m', model.path);

    Object.entries(config.params).forEach(([key, value]) => {
      if (value === 'true') {
        cmdParts.push(key);
      } else if (value !== 'false' && value !== '') {
        cmdParts.push(key, String(value));
      }
    });
    if (config.devices && config.devices.length > 0 && config.devices.length < 999) {
      cmdParts.push('-dev', config.devices.join('/'));
    }
    const cmd = cmdParts.join(' ');

    createBenchmark.mutate(
      {
        modelId: config.modelId,
        llamaBinPath: config.llamaCppPath,
        cmd,
        args: cmdParts,
      },
      {
        onSuccess: (data) => {
          if (data) {
            toast.success('压测任务已创建', '正在运行...');
            setBenchmarkModel(null);
            const currentModel = models.find(m => m.id === config.modelId);
            if (currentModel) {
              setBenchmarkResultsModel(currentModel);
            }
          }
        },
        onError: (error) => {
          (error as APIError).handled = true;
          toast.error('创建压测任务失败', error.message);
        },
      }
    );
  };

  const handleViewBenchmarkResults = (model: Model) => setBenchmarkResultsModel(model);
  const handleShowDetail = (model: Model) => setDetailModel(model);

  return (
    <div className="space-y-4">
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

      <div className="flex items-center gap-2 flex-wrap">
        {MODEL_CATEGORIES.map(cat => {
          const Icon = CATEGORY_ICONS[cat.key];
          return (
            <button
              key={cat.key}
              onClick={() => setCategory(cat.key)}
              className={cn(
                "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium transition-colors border",
                category === cat.key
                  ? "bg-primary text-primary-foreground border-primary"
                  : "bg-card text-muted-foreground border-border hover:bg-accent hover:text-accent-foreground"
              )}
            >
              <Icon className="w-3.5 h-3.5" />
              {cat.label}
              <span className={cn(
                "text-xs tabular-nums",
                category === cat.key ? "text-primary-foreground/70" : "text-muted-foreground/70"
              )}>
                {categoryCounts[cat.key]}
              </span>
            </button>
          );
        })}

        <div className="flex-1" />

        <Button
          onClick={() => setFavouriteFilter(!favouriteFilter)}
          variant={favouriteFilter ? 'default' : 'outline'}
          size="sm"
          className={cn(
            favouriteFilter && 'border-yellow-500 bg-yellow-500 text-white hover:bg-yellow-600 dark:bg-yellow-600 dark:hover:bg-yellow-700 dark:border-yellow-700'
          )}
        >
          <Star className={cn('w-4 h-4', favouriteFilter && 'fill-current')} />
          只看收藏
        </Button>

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

      <div className="flex items-center gap-3 p-3 bg-card rounded-lg border border-border">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="搜索模型名称、架构..."
            className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-input text-foreground placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div className="flex items-center gap-1.5">
          {STATUS_CHIPS.map(s => (
            <button
              key={s.value}
              onClick={() => setStatusFilter(s.value)}
              className={cn(
                "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-colors border",
                statusFilter === s.value
                  ? "bg-primary text-primary-foreground border-primary"
                  : "bg-transparent text-muted-foreground border-border hover:bg-accent hover:text-accent-foreground"
              )}
            >
              {s.dot && (
                <span className={cn(
                  "w-1.5 h-1.5 rounded-full shrink-0",
                  statusFilter === s.value ? 'bg-primary-foreground' : s.dot
                )} />
              )}
              {s.label}
            </button>
          ))}
        </div>
      </div>

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
            'gap-4 sm:gap-5',
            viewMode === 'grid'
              ? 'grid grid-cols-1 sm:grid-cols-2'
              : 'flex flex-col'
          )}
        >
          {filteredModels.map((model) => (
            <ModelCard
              key={model.id}
              model={model}
              capabilities={capabilitiesMap[model.id]}
              onLoad={() => handleLoadClick(model)}
              onUnload={() => handleUnloadClick(model.id)}
              onToggleFavourite={() => handleToggleFavourite(model.id, model.favourite)}
              onShowDetail={() => handleShowDetail(model)}
              onEditAlias={() => handleEditAlias(model)}
              actions={
                <>
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

      {dialogModel && (
        <LoadModelDialog
          isOpen={!!dialogModel}
          onClose={() => setDialogModel(null)}
          onConfirm={handleLoadConfirm}
          modelId={dialogModel.id}
          modelName={dialogModel.alias || dialogModel.name}
          modelPath={dialogModel.path}
          isLoading={loadModel.isPending}
          backendType={dialogModel.backendType}
        />
      )}

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

      {benchmarkResultsModel && (
        <BenchmarkResultsDialog
          isOpen={!!benchmarkResultsModel}
          onClose={() => setBenchmarkResultsModel(null)}
          modelId={benchmarkResultsModel.id}
          modelName={benchmarkResultsModel.alias || benchmarkResultsModel.name}
        />
      )}

      {detailModel && (
        <ModelDetailDialog
          isOpen={!!detailModel}
          onClose={() => setDetailModel(null)}
          modelId={detailModel.id}
          modelName={detailModel.alias || detailModel.name}
        />
      )}
    </div>
  );
}
