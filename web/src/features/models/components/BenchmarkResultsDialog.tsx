import { useState, useEffect, useCallback } from 'react';
import { FileText, Loader2, RefreshCw, Trash2, PlusCircle } from 'lucide-react';
import { cn, formatBytes } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { toast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';
import type { BenchmarkResultFile, BenchmarkResult } from '@/types';

interface BenchmarkResultsDialogProps {
  isOpen: boolean;
  onClose: () => void;
  modelId: string;
  modelName: string;
}

/**
 * Benchmark results comparison dialog
 * Left panel: result file list, Right panel: content comparison area
 */
export function BenchmarkResultsDialog({
  isOpen,
  onClose,
  modelId,
  modelName,
}: BenchmarkResultsDialogProps) {
  const alertDialog = useAlertDialog();

  const [resultFiles, setResultFiles] = useState<BenchmarkResultFile[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [resultContent, setResultContent] = useState<string>('');
  const [loadingFile, setLoadingFile] = useState<string | null>(null);

  const loadResultFiles = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await fetch(`/api/models/benchmark/tasks?modelId=${encodeURIComponent(modelId)}`);
      const data = await response.json();

      if (data.success && data.data?.benchmarks) {
        // Map benchmark tasks to result file format for display
        setResultFiles(data.data.benchmarks.map((b: { id: string; status: string; createdAt: string }) => ({
          name: b.id,
          size: 0,
          modified: b.createdAt || '',
          status: b.status,
        })));
      } else {
        setResultFiles([]);
      }
    } catch (error) {
      console.error('Failed to load benchmark results:', error);
      toast.error('加载测试结果列表失败', error instanceof Error ? error.message : '未知错误');
      setResultFiles([]);
    } finally {
      setIsLoading(false);
    }
  }, [modelId, toast]);

  useEffect(() => {
    if (isOpen && modelId) {
      loadResultFiles();
      setResultContent('');
    }
  }, [isOpen, modelId, loadResultFiles]);

  const appendResult = (result: BenchmarkResult, fileName: string) => {
    let text = '';

    text += '---\n';
    text += `文件: ${fileName}\n`;
    text += `模型: ${result.modelName || modelName}\n`;
    text += `模型ID: ${result.modelId || modelId}\n`;

    if (result.commandStr) {
      text += `\n命令:\n${result.commandStr}\n`;
    } else if (result.command && result.command.length) {
      text += `\n命令:\n${result.command.join(' ')}\n`;
    }

    if (result.exitCode != null) {
      text += `\n退出码: ${result.exitCode}\n`;
    }

    if (result.savedPath) {
      text += `\n保存文件: ${result.savedPath}\n`;
    }

    if (result.metrics) {
      text += `\n性能指标:\n`;
      if (result.metrics.tps != null) {
        text += `  - TPS (tokens/s): ${result.metrics.tps}\n`;
      }
      if (result.metrics.promptTps != null) {
        text += `  - Prompt TPS: ${result.metrics.promptTps}\n`;
      }
      if (result.metrics.totalTokens != null) {
        text += `  - Total Tokens: ${result.metrics.totalTokens}\n`;
      }
      if (result.metrics.loadTime != null) {
        text += `  - Load Time: ${result.metrics.loadTime} ms\n`;
      }
      if (result.metrics.memoryUsage != null) {
        text += `  - Memory Usage: ${result.metrics.memoryUsage} MB\n`;
      }
    }

    if (result.rawOutput) {
      text += `\n原始输出:\n${result.rawOutput}\n`;
    }

    setResultContent(prev => {
      const separator = prev.trim().length > 0 ? '\n\n' : '';
      return prev + separator + text;
    });
  };

  const loadBenchmarkResult = async (fileName: string) => {
    setLoadingFile(fileName);
    try {
      const response = await fetch(`/api/models/benchmark/tasks/${encodeURIComponent(fileName)}`);
      const data = await response.json();

      if (data.success && data.data) {
        appendResult(data.data as BenchmarkResult, fileName);
        toast.success('已追加测试结果');
      } else {
        toast.error('加载测试结果失败', data.error || '未知错误');
      }
    } catch (error) {
      console.error('Failed to load benchmark result:', error);
      toast.error('加载测试结果失败', error instanceof Error ? error.message : '网络错误');
    } finally {
      setLoadingFile(null);
    }
  };

  const deleteBenchmarkResult = async (fileName: string) => {
    const confirmed = await alertDialog.confirm({
      title: '删除测试结果',
      description: `确定要删除测试结果文件 "${fileName}" 吗？此操作不可恢复。`,
      confirmText: '删除',
      cancelText: '取消',
      variant: 'destructive',
    });

    if (!confirmed) return;

    try {
      const response = await fetch(`/api/models/benchmark/tasks/${encodeURIComponent(fileName)}`, {
        method: 'DELETE',
      });
      const data = await response.json();

      if (data.success) {
        setResultFiles(prev => prev.filter(f => f.name !== fileName));
        toast.success('测试结果已删除');
      } else {
        toast.error('删除测试结果失败', data.error || '未知错误');
      }
    } catch (error) {
      console.error('Failed to delete benchmark result:', error);
      toast.error('删除测试结果失败', error instanceof Error ? error.message : '网络错误');
    }
  };

  const clearContent = () => {
    setResultContent('');
  };

  if (!isOpen) return null;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="sm:max-w-6xl max-h-[85vh] flex flex-col p-0">
        {/* Header */}
        <DialogHeader className="px-4 py-3 border-b border-border flex-shrink-0">
          <DialogTitle className="flex items-center gap-2">
            <FileText className="w-5 h-5 text-blue-500" />
            模型测试结果对比
          </DialogTitle>
        </DialogHeader>

        {/* Content - two-column layout */}
        <div className="flex-1 flex gap-4 p-4 min-h-0 overflow-hidden">
          {/* Left: file list */}
          <div className="w-1/3 min-w-[280px] border border-border rounded-lg overflow-hidden bg-muted/30 flex flex-col">
            <div className="px-3 py-2 border-b border-border text-sm font-medium text-foreground bg-muted/50 flex items-center justify-between">
              <span>测试结果文件</span>
              <Button
                size="sm"
                variant="ghost"
                className="h-6 px-2 text-xs"
                onClick={loadResultFiles}
                disabled={isLoading}
              >
                <RefreshCw className={cn("w-3 h-3", isLoading && "animate-spin")} />
              </Button>
            </div>
            <div className="flex-1 overflow-y-auto">
              {isLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
                  <span className="ml-2 text-sm text-muted-foreground">加载中...</span>
                </div>
              ) : resultFiles.length > 0 ? (
                <div className="divide-y divide-border">
                  {resultFiles.map((file, index) => (
                    <div
                      key={index}
                      className="p-3 hover:bg-accent/50 transition-colors"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-1.5 text-sm font-medium text-foreground truncate">
                            <FileText className="w-4 h-4 flex-shrink-0 text-muted-foreground" />
                            <span className="truncate" title={file.name}>{file.name}</span>
                          </div>
                          <div className="mt-1 text-xs text-muted-foreground space-y-0.5">
                            <div>修改时间: {file.modified || '-'}</div>
                            <div>大小: {formatBytes(file.size)}</div>
                          </div>
                        </div>
                        <div className="flex flex-col gap-1 flex-shrink-0">
                          <Button
                            size="sm"
                            variant="default"
                            className="h-7 px-2 text-xs"
                            onClick={() => loadBenchmarkResult(file.name)}
                            disabled={loadingFile === file.name}
                          >
                            {loadingFile === file.name ? (
                              <Loader2 className="w-3 h-3 animate-spin" />
                            ) : (
                              <>
                                <PlusCircle className="w-3 h-3 mr-1" />
                                追加
                              </>
                            )}
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            className="h-7 px-2 text-xs text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950"
                            onClick={() => deleteBenchmarkResult(file.name)}
                          >
                            <Trash2 className="w-3 h-3 mr-1" />
                            删除
                          </Button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-sm text-muted-foreground text-center py-8 px-4">
                  <FileText className="w-10 h-10 mx-auto mb-2 opacity-50" />
                  <p>未找到测试结果文件</p>
                </div>
              )}
            </div>
          </div>

          {/* Right: content display */}
          <div className="flex-1 flex flex-col min-w-0 border border-border rounded-lg overflow-hidden">
            <div className="px-3 py-2 border-b border-border flex items-center justify-between bg-muted/50">
              <div className="text-sm text-muted-foreground">
                当前模型: <span className="font-medium text-foreground">{modelName}</span>
              </div>
              <Button
                size="sm"
                variant="ghost"
                className="h-7 px-2 text-xs"
                onClick={clearContent}
                disabled={!resultContent}
              >
                清空内容
              </Button>
            </div>
            <pre className="flex-1 overflow-auto p-3 text-xs bg-gray-900 text-gray-100 font-mono whitespace-pre-wrap break-all">
              {resultContent || <span className="text-gray-500">点击左侧文件的「追加」按钮查看测试结果，支持追加多个结果进行对比</span>}
            </pre>
          </div>
        </div>

        {/* Footer */}
        <DialogFooter className="px-4 py-3 border-t border-border bg-card flex-shrink-0">
          <Button variant="secondary" onClick={onClose}>
            关闭
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
