import { useTranslation } from 'react-i18next';
import { FileText, Trash2, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import type { BenchmarkHistoryFile } from '@/types';

interface HistoryPanelProps {
  files: BenchmarkHistoryFile[];
  isLoading: boolean;
  selectedFile: string | null;
  onSelectFile: (fileName: string) => void;
  onDeleteFile: (fileName: string) => void;
}

export function HistoryPanel({
  files,
  isLoading,
  selectedFile,
  onSelectFile,
  onDeleteFile,
}: HistoryPanelProps) {
  const { t } = useTranslation();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="w-4 h-4 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (files.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
        <FileText className="w-8 h-8 opacity-40 mb-2" />
        <p className="text-sm">{t('benchmark.noHistory')}</p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-border overflow-y-auto">
      {files.map((file) => (
        <div
          key={file.name}
          className={cn(
            'group flex items-center gap-2 px-3 py-2 cursor-pointer transition-colors',
            'hover:bg-accent/50',
            selectedFile === file.name && 'bg-accent'
          )}
          onClick={() => onSelectFile(file.name)}
        >
          <FileText className="w-3.5 h-3.5 flex-shrink-0 text-muted-foreground" />
          <div className="flex-1 min-w-0">
            <div className="text-xs font-medium text-foreground truncate">
              {file.name}
            </div>
            <div className="text-[10px] text-muted-foreground">
              {file.modified}
            </div>
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity text-destructive hover:text-destructive"
            onClick={(e) => {
              e.stopPropagation();
              onDeleteFile(file.name);
            }}
          >
            <Trash2 className="w-3 h-3" />
          </Button>
        </div>
      ))}
    </div>
  );
}
