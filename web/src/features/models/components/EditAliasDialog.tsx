import { useState } from 'react';
import { X, Pencil } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface EditAliasDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (alias: string) => void;
  modelId: string;
  modelName: string;
  currentAlias?: string;
  isLoading?: boolean;
}

export function EditAliasDialog({
  isOpen,
  onClose,
  onConfirm,
  modelName,
  currentAlias,
  isLoading = false,
}: EditAliasDialogProps) {
  const [alias, setAlias] = useState(currentAlias || '');

  if (!isOpen) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onConfirm(alias.trim());
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-lg shadow-xl max-w-md w-full">
        <div className="flex items-center justify-between p-4 border-b border-border">
          <div className="flex items-center gap-2">
            <Pencil className="w-5 h-5 text-blue-600" />
            <h2 className="text-lg font-semibold text-foreground">
              编辑模型别名
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
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              模型名称
            </label>
            <div className="px-3 py-2 bg-muted rounded-md text-foreground text-sm">
              {modelName}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              别名
            </label>
            <input
              type="text"
              value={alias}
              onChange={(e) => setAlias(e.target.value)}
              placeholder="输入模型别名（可选）"
              className="w-full px-3 py-2 border border-border rounded-md bg-input text-foreground placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              autoFocus
            />
            <p className="mt-1 text-xs text-muted-foreground">
              设置别名后，模型将以别名显示在列表中
            </p>
          </div>

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
              disabled={isLoading}
            >
              {isLoading ? '保存中...' : '保存'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
