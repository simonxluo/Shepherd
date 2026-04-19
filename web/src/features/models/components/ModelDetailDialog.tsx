import { useState } from 'react';
import { X, ArrowLeft, Star, Copy, Check, Info } from 'lucide-react';
import { useModel } from '@/features/models';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import type { Model } from '@/types';

interface ModelDetailDialogProps {
  isOpen: boolean;
  onClose: () => void;
  modelId: string;
  modelName: string;
}

/**
 * Format file size
 */
function formatSize(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${size.toFixed(2)} ${units[unitIndex]}`;
}

/**
 * Format time duration
 */
/**
 * Detail row component
 */
function DetailRow({ label, value, onCopy }: { label: string; value: string | number | undefined; onCopy?: () => void }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    if (onCopy && value) {
      navigator.clipboard.writeText(String(value));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="flex items-center justify-between py-2.5 px-4 hover:bg-muted/50 rounded-md group">
      <span className="text-sm text-muted-foreground font-medium">{label}</span>
      <div className="flex items-center gap-2">
        <span className="text-sm text-foreground font-mono">{value || '-'}</span>
        {onCopy && value && (
          <button
            onClick={handleCopy}
            className="opacity-0 group-hover:opacity-100 transition-opacity"
          >
            {copied ? (
              <Check className="w-3.5 h-3.5 text-green-500" />
            ) : (
              <Copy className="w-3.5 h-3.5 text-muted-foreground hover:text-foreground" />
            )}
          </button>
        )}
      </div>
    </div>
  );
}

/**
 * Detail section component
 */
function DetailSection({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="mb-6">
      <div className="flex items-center gap-2 mb-3 px-1">
        {icon}
        <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      </div>
      <div className="bg-muted/30 rounded-lg overflow-hidden">
        {children}
      </div>
    </div>
  );
}

/**
 * Model detail dialog
 */
export function ModelDetailDialog({ isOpen, onClose, modelId, modelName }: ModelDetailDialogProps) {
  const { data: model, isLoading } = useModel(modelId);

  if (!isOpen) return null;

  const modelData = model as Model | undefined;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-2xl max-h-[90vh] overflow-hidden bg-card rounded-xl shadow-2xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              className="p-1.5 hover:bg-muted rounded-lg transition-colors"
            >
              <ArrowLeft className="w-5 h-5 text-muted-foreground" />
            </button>
            <h2 className="text-xl font-bold text-foreground">{modelData?.alias || modelData?.displayName || modelName}</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 hover:bg-muted rounded-lg transition-colors"
          >
            <X className="w-5 h-5 text-muted-foreground" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-2" />
                <p className="text-sm text-muted-foreground">加载中...</p>
              </div>
            </div>
          ) : modelData ? (
            <div className="space-y-4">
              {/* Basic info */}
              <DetailSection title="基本信息" icon={<Info className="w-4 h-4 text-blue-500" />}>
                <DetailRow label="架构" value={modelData.metadata.architecture} />
                <DetailRow label="量化" value={modelData.metadata.quantization || modelData.metadata.fileTypeDescriptor} />
                <DetailRow label="大小" value={formatSize(modelData.totalSize ?? modelData.size)} />
                {modelData.shardCount && modelData.shardCount > 1 && (
                  <DetailRow label="分片数" value={`${modelData.shardCount} 个文件`} />
                )}
                <DetailRow label="路径" value={modelData.path} onCopy={() => {}} />
                <DetailRow label="ID" value={modelData.id} onCopy={() => {}} />
                {modelData.mmprojPath && (
                  <DetailRow label="视觉模型路径" value={modelData.mmprojPath} onCopy={() => {}} />
                )}
              </DetailSection>

              {/* Metadata */}
              {modelData.metadata && (
                <DetailSection title="元数据" icon={<Info className="w-4 h-4 text-purple-500" />}>
                  <DetailRow label="文件类型描述" value={modelData.metadata.fileTypeDescriptor} />
                  {modelData.metadata.url && (
                    <DetailRow label="来源 URL" value={modelData.metadata.url} onCopy={() => {}} />
                  )}
                  {modelData.metadata.author && (
                    <DetailRow label="作者" value={modelData.metadata.author} />
                  )}
                  {modelData.metadata.parameters && (
                    <DetailRow label="参数量" value={`${(modelData.metadata.parameters / 1e9).toFixed(1)}B`} />
                  )}
                  {modelData.metadata.contextLength && (
                    <DetailRow label="上下文长度" value={`${modelData.metadata.contextLength.toLocaleString()}`} />
                  )}
                  {modelData.metadata.embeddingLength && (
                    <DetailRow label="嵌入维度" value={modelData.metadata.embeddingLength.toLocaleString()} />
                  )}
                  {modelData.metadata.layerCount && (
                    <DetailRow label="层数" value={modelData.metadata.layerCount} />
                  )}
                  {modelData.metadata.headCount && (
                    <DetailRow label="注意力头数" value={modelData.metadata.headCount} />
                  )}
                  {modelData.metadata.license && (
                    <DetailRow label="许可证" value={modelData.metadata.license} />
                  )}
                </DetailSection>
              )}

              {/* Status info */}
              <DetailSection title="状态信息" icon={<Info className="w-4 h-4 text-green-500" />}>
                <DetailRow label="状态" value={
                  modelData.status === 'running' ? '运行中' :
                  modelData.status === 'loading' ? '加载中' :
                  modelData.status === 'stopped' ? '已停止' :
                  modelData.status === 'unloading' ? '卸载中' : '错误'
                } />
                <DetailRow label="已加载" value={modelData.isLoaded ? '是' : '否'} />
                {modelData.port && (
                  <DetailRow label="端口" value={modelData.port} />
                )}
                {modelData.slots && modelData.slots.length > 0 && (
                  <DetailRow label="处理槽位" value={`${modelData.slots.length} 个`} />
                )}
                <DetailRow label="扫描时间" value={new Date(modelData.scannedAt).toLocaleString('zh-CN')} />
                <DetailRow label="收藏" value={modelData.favourite ? '是' : '否'} />
              </DetailSection>

              {/* Tags */}
              {modelData.tags && modelData.tags.length > 0 && (
                <DetailSection title="标签" icon={<Info className="w-4 h-4 text-orange-500" />}>
                  <div className="flex flex-wrap gap-2 px-4 py-3">
                    {modelData.tags.map((tag, index) => (
                      <span
                        key={index}
                        className="px-2.5 py-1 text-xs font-medium bg-muted text-muted-foreground rounded-md"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                </DetailSection>
              )}
            </div>
          ) : (
            <div className="text-center py-12 text-muted-foreground">
              未找到模型信息
            </div>
          )}
        </div>

        {/* Footer */}
        {modelData && (
          <div className="flex items-center justify-between px-6 py-4 border-t border-border bg-muted/30">
            <div className="flex items-center gap-2">
              <Star
                className={cn(
                  'w-5 h-5 transition-colors',
                  modelData.favourite ? 'text-yellow-500 fill-current' : 'text-muted-foreground'
                )}
              />
              <span className="text-sm text-muted-foreground">
                {modelData.favourite ? '已收藏' : '未收藏'}
              </span>
            </div>
            <Button onClick={onClose} variant="default" size="sm">
              关闭
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
