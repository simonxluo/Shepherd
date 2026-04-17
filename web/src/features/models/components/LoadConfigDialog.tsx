import { useState } from 'react';
import { X, FolderOpen } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';

interface ConfigOption {
  name: string;
  config: any;
  createdAt: string;
}

interface LoadConfigDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onLoad: (configName: string) => void;
  configs: ConfigOption[];
  isLoading?: boolean;
}

/**
 * 加载压测配置对话框组件
 */
export function LoadConfigDialog({
  isOpen,
  onClose,
  onLoad,
  configs,
  isLoading = false,
}: LoadConfigDialogProps) {
  const [selectedConfig, setSelectedConfig] = useState<string>('');

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (selectedConfig) {
      onLoad(selectedConfig);
      onClose();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-lg shadow-xl max-w-md w-full">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b border-border">
          <div className="flex items-center gap-2">
            <FolderOpen className="w-5 h-5 text-blue-500" />
            <h2 className="text-lg font-semibold text-foreground">
              加载压测配置
            </h2>
          </div>
          <button
            onClick={onClose}
            disabled={isLoading}
            className="p-1 text-muted-foreground hover:text-foreground disabled:opacity-50"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {/* 配置选择 */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">
              选择配置
            </label>
            {configs.length === 0 ? (
              <div className="px-3 py-4 bg-muted rounded-md text-center text-sm text-muted-foreground">
                暂无保存的配置
              </div>
            ) : (
              <div className="space-y-2 max-h-60 overflow-y-auto border border-border rounded-md p-2">
                {configs.map((config) => (
                  <label
                    key={config.name}
                    className={cn(
                      'flex items-start gap-3 p-3 rounded cursor-pointer transition-colors',
                      'hover:bg-accent',
                      selectedConfig === config.name && 'bg-accent border-2 border-blue-500'
                    )}
                  >
                    <input
                      type="radio"
                      name="config"
                      value={config.name}
                      checked={selectedConfig === config.name}
                      onChange={(e) => setSelectedConfig(e.target.value)}
                      disabled={isLoading}
                      className="mt-1 rounded border-border text-blue-600 focus:ring-blue-500"
                    />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-foreground">
                        {config.name}
                      </div>
                      <div className="text-xs text-muted-foreground mt-1">
                        创建时间: {new Date(config.createdAt).toLocaleString('zh-CN')}
                      </div>
                    </div>
                  </label>
                ))}
              </div>
            )}
          </div>

          {/* 按钮区域 */}
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
              disabled={!selectedConfig || isLoading}
            >
              {isLoading ? '加载中...' : '加载'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
