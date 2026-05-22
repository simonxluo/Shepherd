import { useTranslation } from 'react-i18next';
import { Copy, Download, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { toast } from '@/hooks/useToast';

interface OutputPanelProps {
  content: string;
  isLoading: boolean;
  fileName: string | null;
}

export function OutputPanel({ content, isLoading, fileName }: OutputPanelProps) {
  const { t } = useTranslation();

  const handleCopy = async () => {
    if (!content) return;
    try {
      await navigator.clipboard.writeText(content);
      toast.success(t('benchmark.copied'));
    } catch {
      toast.error('Failed to copy');
    }
  };

  const handleExport = () => {
    if (!content) return;
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName || 'benchmark-output.txt';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast.success(t('benchmark.exported'));
  };

  return (
    <div className="flex-1 flex flex-col min-h-0">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-border bg-muted/30">
        <span className="text-xs text-muted-foreground">
          {fileName || t('benchmark.output')}
        </span>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            onClick={handleCopy}
            disabled={!content}
          >
            <Copy className="w-3 h-3 mr-1" />
            {t('benchmark.copy')}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            onClick={handleExport}
            disabled={!content}
          >
            <Download className="w-3 h-3 mr-1" />
            {t('benchmark.export')}
          </Button>
        </div>
      </div>

      {/* Output content */}
      <div className="flex-1 overflow-auto">
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : content ? (
          <pre className="p-3 text-xs font-mono text-foreground whitespace-pre-wrap break-all leading-relaxed">
            {content}
          </pre>
        ) : (
          <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
            {t('benchmark.noOutput')}
          </div>
        )}
      </div>
    </div>
  );
}
