import { useTranslation } from 'react-i18next';
import { useModel } from '@/features/models';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import type { Model } from '@/types';
import { OverviewTab } from './model-detail/OverviewTab';
import { SamplingTab } from './model-detail/SamplingTab';
import { ChatTemplateTab } from './model-detail/ChatTemplateTab';
import { TokenCalcTab } from './model-detail/TokenCalcTab';
import { KwargsTab } from './model-detail/KwargsTab';
import { SlotsTab } from './model-detail/SlotsTab';

interface ModelDetailDialogProps {
  isOpen: boolean;
  onClose: () => void;
  modelId: string;
  modelName: string;
}

function StatusBadge({ status }: { status: string }) {
  const colorMap: Record<string, string> = {
    running: 'bg-green-500/15 text-green-600 border-green-500/30',
    loading: 'bg-blue-500/15 text-blue-600 border-blue-500/30',
    stopped: 'bg-gray-500/15 text-gray-600 border-gray-500/30',
    unloading: 'bg-yellow-500/15 text-yellow-600 border-yellow-500/30',
    error: 'bg-red-500/15 text-red-600 border-red-500/30',
  };

  return (
    <span className={cn(
      'inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full border',
      colorMap[status] || colorMap.stopped
    )}>
      {status === 'running' && <span className="w-1.5 h-1.5 rounded-full bg-green-500 mr-1.5 animate-pulse" />}
      {status}
    </span>
  );
}

export function ModelDetailDialog({ isOpen, onClose, modelId, modelName }: ModelDetailDialogProps) {
  const { t } = useTranslation();
  const { data: model, isLoading } = useModel(modelId);
  const modelData = model as Model | undefined;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="sm:max-w-4xl max-h-[85vh] flex flex-col p-0">
        <DialogHeader className="px-6 py-4 border-b border-border shrink-0">
          <div className="flex items-center gap-3">
            <DialogTitle className="text-lg font-bold truncate">
              {modelData?.alias || modelData?.displayName || modelName}
            </DialogTitle>
            {modelData && <StatusBadge status={modelData.status} />}
          </div>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-2" />
              <p className="text-sm text-muted-foreground">{t('common.loading', '加载中...')}</p>
            </div>
          </div>
        ) : modelData ? (
          <Tabs defaultValue="overview" className="flex-1 min-h-0 px-6 pb-4 pt-2">
            <TabsList className="w-full justify-start" variant="line">
              <TabsTrigger value="overview">{t('modelDetail.tabs.overview', '概览')}</TabsTrigger>
              <TabsTrigger value="sampling">{t('modelDetail.tabs.sampling', '采样设置')}</TabsTrigger>
              <TabsTrigger value="template">{t('modelDetail.tabs.chatTemplate', '聊天模板')}</TabsTrigger>
              <TabsTrigger value="token">{t('modelDetail.tabs.tokenCalc', 'Token计算')}</TabsTrigger>
              <TabsTrigger value="kwargs">{t('modelDetail.tabs.kwargs', 'Kwargs')}</TabsTrigger>
              <TabsTrigger value="slots">{t('modelDetail.tabs.slots', 'Slots')}</TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="mt-3 overflow-y-auto max-h-[calc(85vh-180px)]">
              <OverviewTab model={modelData} />
            </TabsContent>

            <TabsContent value="sampling" className="mt-3 overflow-y-auto max-h-[calc(85vh-180px)]">
              <SamplingTab modelId={modelId} />
            </TabsContent>

            <TabsContent value="template" className="mt-3 overflow-y-auto max-h-[calc(85vh-180px)]">
              <ChatTemplateTab modelId={modelId} />
            </TabsContent>

            <TabsContent value="token" className="mt-3 overflow-hidden flex flex-col max-h-[calc(85vh-180px)]">
              <TokenCalcTab modelId={modelId} isLoaded={modelData.isLoaded} />
            </TabsContent>

            <TabsContent value="kwargs" className="mt-3 overflow-hidden flex flex-col max-h-[calc(85vh-180px)]">
              <KwargsTab modelId={modelId} />
            </TabsContent>

            <TabsContent value="slots" className="mt-3 overflow-hidden flex flex-col max-h-[calc(85vh-180px)]">
              <SlotsTab modelId={modelId} isLoaded={modelData.isLoaded} />
            </TabsContent>
          </Tabs>
        ) : (
          <div className="text-center py-16 text-muted-foreground">
            {t('modelDetail.notFound', '未找到模型信息')}
          </div>
        )}

        {modelData && (
          <div className="px-6 py-3 border-t border-border bg-muted/30 shrink-0 flex justify-end">
            <Button onClick={onClose} variant="default" size="sm">
              {t('common.close', '关闭')}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
