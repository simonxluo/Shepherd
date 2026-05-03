import { Bug, FolderOpen, Edit, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { LlamaCppPathConfig, ModelPathConfig } from '@/lib/config';

interface PathItemProps {
  path: LlamaCppPathConfig | ModelPathConfig;
  onEdit: () => void;
  onRemove: () => void;
  onTest?: () => Promise<void>;
}

export function PathItem({
  path,
  onEdit,
  onRemove,
  onTest,
}: PathItemProps) {
  const displayName = path.name || path.path;
  const isTesting = false;

  const handleTest = async () => {
    if (onTest) {
      await onTest();
    }
  };

  return (
    <div className="group flex items-start gap-3 p-3 rounded-lg border bg-card hover:border-border/80 transition-all">
      {/* Icon */}
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
        <FolderOpen size={16} />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <h4 className="text-sm font-medium truncate">{displayName}</h4>
        </div>
        <p className="text-xs text-muted-foreground truncate mt-0.5">{path.path}</p>
        {path.description && (
          <p className="text-xs text-muted-foreground mt-1">{path.description}</p>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1.5 opacity-60 group-hover:opacity-100 transition-opacity">
        {onTest && (
          <Button
            variant="icon-button"
            size="icon"
            onClick={handleTest}
            disabled={isTesting}
            title="测试路径"
            className="h-8 w-8"
          >
            <Bug size={16} />
          </Button>
        )}
        <Button
          variant="icon-button"
          size="icon"
          onClick={onEdit}
          title="编辑"
          className="h-8 w-8"
        >
          <Edit size={16} />
        </Button>
        <Button
          variant="icon-button"
          size="icon"
          onClick={onRemove}
          className="text-destructive hover:text-destructive hover:bg-destructive/10 h-8 w-8"
          title="删除"
        >
          <Trash2 size={16} />
        </Button>
      </div>
    </div>
  );
}
