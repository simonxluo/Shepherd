import { useState } from 'react';
import { Plus, Trash2, Filter, CloudDownload, Search, Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  useDownloads,
  useCreateDownload,
  usePauseDownload,
  useResumeDownload,
  useCancelDownload,
  useRetryDownload,
  useClearCompletedDownloads,
  useFilteredDownloads,
  useDownloadStats,
} from '@/features/downloads/hooks';
import { DownloadCard } from '@/features/downloads/components/DownloadCard';
import { CreateDownloadDialog } from '@/features/downloads/components/CreateDownloadDialog';
import { HuggingFaceSearchPanel } from '@/features/downloads/components/HuggingFaceSearchPanel';
import type { DownloadState, DownloadSource } from '@/types';
import type { HuggingFaceModel } from '@/lib/api/downloads';
import { useAlertDialog } from '@/providers/AlertDialog';
import { cn } from '@/lib/utils';

export function DownloadsPage() {
  const alertDialog = useAlertDialog();
  const { data: downloads, isLoading } = useDownloads();
  const createDownload = useCreateDownload();
  const pauseDownload = usePauseDownload();
  const resumeDownload = useResumeDownload();
  const cancelDownload = useCancelDownload();
  const retryDownload = useRetryDownload();
  const clearCompleted = useClearCompletedDownloads();

  // UI 状态
  const [activeTab, setActiveTab] = useState<'local' | 'online'>('local');
  const [search, setSearch] = useState('');
  const [stateFilter, setStateFilter] = useState<DownloadState | ''>('');
  const [sourceFilter, setSourceFilter] = useState<DownloadSource | ''>('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [preFillParams, setPreFillParams] = useState<{ source: DownloadSource; repoId: string; fileName?: string } | null>(null);

  // 统计
  const stats = useDownloadStats(downloads);

  // 过滤下载任务
  const filteredDownloads = useFilteredDownloads(downloads, {
    search,
    state: stateFilter || undefined,
    source: sourceFilter || undefined,
  });

  // 处理创建下载
  const handleCreateDownload = (params: { source: DownloadSource; repoId: string }) => {
    createDownload.mutate(params as any, {
      onSuccess: () => {
        setDialogOpen(false);
        setPreFillParams(null);
      },
    });
  };

  // 处理从 HuggingFace 下载
  const handleHuggingFaceDownload = (model: HuggingFaceModel, fileName?: string) => {
    setPreFillParams({ source: 'huggingface', repoId: model.modelId, fileName });
    setDialogOpen(true);
  };

  // 处理清理已完成
  const handleClearCompleted = async () => {
    const confirmed = await alertDialog.confirm({
      title: '清理已完成',
      description: '确定要清理所有已完成的下载任务吗？',
    });
    if (confirmed) {
      clearCompleted.mutate();
    }
  };

  return (
    <div className="space-y-6">
      {/* 标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">下载管理</h1>
          <p className="text-muted-foreground">
            管理本地下载任务或从 HuggingFace 搜索模型
          </p>
        </div>

        {activeTab === 'local' && stats.completed > 0 && (
          <Button
            onClick={handleClearCompleted}
            variant="ghost"
            size="sm"
          >
            <Trash2 className="w-4 h-4" />
            清理已完成
          </Button>
        )}
      </div>

      {/* 标签切换 */}
      <div className="border-b border-border">
        <nav className="flex gap-6">
          <Button
            onClick={() => setActiveTab('local')}
            variant={activeTab === 'local' ? 'default' : 'ghost'}
            size="sm"
            className={cn(
              'rounded-t-lg rounded-b-none border-b-2',
              activeTab === 'local'
                ? 'border-primary'
                : 'border-transparent hover:border-transparent'
            )}
          >
            <Download className="w-4 h-4 mr-2" />
            本地下载 ({stats.total})
          </Button>
          <Button
            onClick={() => setActiveTab('online')}
            variant={activeTab === 'online' ? 'default' : 'ghost'}
            size="sm"
            className={cn(
              'rounded-t-lg rounded-b-none border-b-2',
              activeTab === 'online'
                ? 'border-primary'
                : 'border-transparent hover:border-transparent'
            )}
          >
            <Search className="w-4 h-4 mr-2" />
            在线搜索
          </Button>
        </nav>
      </div>

      {/* 本地下载内容 */}
      {activeTab === 'local' && (
        <>
          {/* 统计卡片 */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="p-4 bg-card rounded-lg border border-border">
              <div className="text-2xl font-bold text-foreground">{stats.total}</div>
              <div className="text-sm text-muted-foreground">总任务</div>
            </div>
            <div className="p-4 bg-card rounded-lg border border-border">
              <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">{stats.active}</div>
              <div className="text-sm text-muted-foreground">进行中</div>
            </div>
            <div className="p-4 bg-card rounded-lg border border-border">
              <div className="text-2xl font-bold text-green-600 dark:text-green-400">{stats.completed}</div>
              <div className="text-sm text-muted-foreground">已完成</div>
            </div>
            <div className="p-4 bg-card rounded-lg border border-border">
              <div className="text-2xl font-bold text-foreground">
                {((stats.downloadedBytes / (stats.totalBytes || 1)) * 100).toFixed(1)}%
              </div>
              <div className="text-sm text-muted-foreground">总进度</div>
            </div>
          </div>

          {/* 搜索和过滤 */}
          <div className="flex flex-wrap items-center gap-3 p-4 bg-card rounded-lg border border-border">
            <div className="relative flex-1 min-w-[200px]">
              <CloudDownload className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="搜索仓库 ID 或文件名..."
                className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-input text-foreground placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <select
              value={stateFilter}
              onChange={(e) => setStateFilter(e.target.value as DownloadState | '')}
              className="px-3 py-2 border border-border rounded-lg bg-input text-foreground"
            >
              <option value="">所有状态</option>
              <option value="downloading">下载中</option>
              <option value="paused">已暂停</option>
              <option value="completed">已完成</option>
              <option value="failed">失败</option>
            </select>

            <select
              value={sourceFilter}
              onChange={(e) => setSourceFilter(e.target.value as DownloadSource | '')}
              className="px-3 py-2 border border-border rounded-lg bg-input text-foreground"
            >
              <option value="">所有来源</option>
              <option value="huggingface">HuggingFace</option>
              <option value="modelscope">ModelScope</option>
            </select>
          </div>

          {/* 下载列表 */}
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-2" />
                <p className="text-muted-foreground">加载中...</p>
              </div>
            </div>
          ) : filteredDownloads.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <CloudDownload className="w-12 h-12 mb-4" />
              <p className="text-lg mb-2">暂无下载任务</p>
              <p className="text-sm mb-4">创建新任务开始下载模型</p>
              <Button
                onClick={() => {
                  setPreFillParams(null);
                  setDialogOpen(true);
                }}
                variant="default"
                size="sm"
              >
                <Plus className="w-4 h-4" />
                新建下载
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              {filteredDownloads.map((task) => (
                <DownloadCard
                  key={task.id}
                  task={task}
                  onPause={() => pauseDownload.mutate(task.id)}
                  onResume={() => resumeDownload.mutate(task.id)}
                  onCancel={() => cancelDownload.mutate(task.id)}
                  onRetry={() => retryDownload.mutate(task.id)}
                />
              ))}
            </div>
          )}
        </>
      )}

      {/* 在线搜索内容 */}
      {activeTab === 'online' && (
        <HuggingFaceSearchPanel onDownload={handleHuggingFaceDownload} />
      )}

      {/* 创建下载对话框 */}
      <CreateDownloadDialog
        isOpen={dialogOpen}
        onClose={() => {
          setDialogOpen(false);
          setPreFillParams(null);
        }}
        onConfirm={handleCreateDownload}
        isLoading={createDownload.isPending}
        preFill={preFillParams}
      />
    </div>
  );
}
