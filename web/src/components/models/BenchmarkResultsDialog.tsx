import { useState } from 'react';
import { X, FileText, Trash2, Loader2, Gauge, Copy, Download } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import {
  useBenchmarkResults,
  useBenchmarkResult,
  useDeleteBenchmarkResult,
} from '@/features/models/hooks';
import type { BenchmarkResult, BenchmarkResultFile } from '@/types';
import { useToast } from '@/hooks/useToast';
import { useAlertDialog } from '@/hooks/useAlertDialog';

// 常量定义
const SEPARATOR_LENGTH = 60;
const SEPARATOR_CHAR = '=';
const FILE_SIZE_UNITS = ['B', 'KB', 'MB', 'GB'] as const;
const BYTES_IN_KB = 1024;

interface BenchmarkResultsDialogProps {
  isOpen: boolean;
  onClose: () => void;
  modelId: string;
  modelName: string;
}

/**
 * 压测结果查看对话框组件
 */
export function BenchmarkResultsDialog({
  isOpen,
  onClose,
  modelId,
  modelName,
}: BenchmarkResultsDialogProps) {
  const toast = useToast();
  const alertDialog = useAlertDialog();

  const { data: resultFiles = [], isLoading: listLoading } = useBenchmarkResults(
    isOpen ? modelId : ''
  );

  const deleteResult = useDeleteBenchmarkResult();

  const [selectedResults, setSelectedResults] = useState<BenchmarkResult[]>([]);
  const [displayContent, setDisplayContent] = useState<string>('');

  // 加载并追加结果到显示区域
  const handleAppendResult = async (fileName: string) => {
    try {
      const response = await fetch(`/api/models/benchmark/get?fileName=${encodeURIComponent(fileName)}`);
      const data = await response.json();
      if (data.success && data.data) {
        const result = data.data as BenchmarkResult;
        setSelectedResults((prev) => [...prev, result]);
        appendResultToDisplay(result);
      }
    } catch (error) {
      console.error('Failed to load result:', error);
      toast.error('加载结果失败', error instanceof Error ? error.message : '未知错误');
    }
  };

  // 追加结果到显示区域
  const appendResultToDisplay = (result: BenchmarkResult) => {
    const separator = '\n' + SEPARATOR_CHAR.repeat(SEPARATOR_LENGTH) + '\n';
    let text = displayContent || '';

    if (text.length > 0) {
      text += separator;
    }

    text += `文件: ${result.fileName}\n`;
    text += `模型: ${result.modelName}\n`;
    text += `模型ID: ${result.modelId}\n`;
    text += `时间: ${new Date(result.timestamp).toLocaleString('zh-CN')}\n`;
    text += `\n命令:\n${result.commandStr}\n`;
    text += `\n退出码: ${result.exitCode}\n`;
    text += `\n保存路径: ${result.savedPath}\n`;

    // 解析的性能指标
    if (result.metrics) {
      text += `\n性能指标:\n`;
      if (result.metrics.tps) text += `  - TPS: ${result.metrics.tps.toFixed(2)} tokens/s\n`;
      if (result.metrics.promptTps) text += `  - Prompt TPS: ${result.metrics.promptTps.toFixed(2)} tokens/s\n`;
      if (result.metrics.totalTokens) text += `  - Total Tokens: ${result.metrics.totalTokens}\n`;
      if (result.metrics.loadTime) text += `  - Load Time: ${result.metrics.loadTime} ms\n`;
      if (result.metrics.memoryUsage) text += `  - Memory Usage: ${result.metrics.memoryUsage} MB\n`;
    }

    text += `\n原始输出:\n${result.rawOutput}\n`;

    setDisplayContent(text);
  };

  // 删除结果
  const handleDeleteResult = async (fileName: string) => {
    const confirmed = await alertDialog.confirm({
      title: '删除测试结果',
      description: '确定要删除该测试结果吗？此操作不可撤销。',
      confirmText: '删除',
      cancelText: '取消',
      variant: 'destructive',
    });
    if (!confirmed) return;

    deleteResult.mutate(fileName, {
      onSuccess: () => {
        // 从已选结果中移除
        setSelectedResults((prev) =>
          prev.filter((r) => r.fileName !== fileName)
        );
        // 重新生成显示内容（移除已删除的结果）
        const newContent = selectedResults
          .filter((r) => r.fileName !== fileName)
          .map((r) => formatResultText(r))
          .join('\n' + SEPARATOR_CHAR.repeat(SEPARATOR_LENGTH) + '\n');
        setDisplayContent(newContent);
      },
    });
  };

  // 格式化单个结果为文本
  const formatResultText = (result: BenchmarkResult): string => {
    let text = `文件: ${result.fileName}\n`;
    text += `模型: ${result.modelName}\n`;
    text += `时间: ${new Date(result.timestamp).toLocaleString('zh-CN')}\n`;
    text += `\n命令:\n${result.commandStr}\n`;
    text += `\n原始输出:\n${result.rawOutput}\n`;
    return text;
  };

  // 清空显示内容
  const handleClearContent = () => {
    setDisplayContent('');
    setSelectedResults([]);
  };

  // 复制到剪贴板
  const handleCopyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(displayContent);
      toast.success('已复制到剪贴板');
    } catch (error) {
      toast.error('复制失败', error instanceof Error ? error.message : '未知错误');
    }
  };

  // 下载为文件
  const handleDownload = () => {
    const blob = new Blob([displayContent], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `benchmark-results-${modelId}-${Date.now()}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // 格式化文件大小
  const formatFileSize = (bytes: number): string => {
    let size = bytes;
    let unitIndex = 0;
    while (size >= BYTES_IN_KB && unitIndex < FILE_SIZE_UNITS.length - 1) {
      size /= BYTES_IN_KB;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${FILE_SIZE_UNITS[unitIndex]}`;
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-lg shadow-xl max-w-6xl w-full max-h-[85vh] flex flex-col">
        {/* 标题栏 */}
        <div className="flex items-center justify-between p-4 border-b border-border flex-shrink-0">
          <div className="flex items-center gap-2">
            <FileText className="w-5 h-5 text-blue-500" />
            <h2 className="text-lg font-semibold text-foreground">
              模型测试结果对比
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 text-muted-foreground hover:text-foreground"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* 内容区域 - 两列布局 */}
        <div className="flex-1 flex min-h-0">
          {/* 左侧：测试任务列表 */}
          <div className="w-80 border-r border-border flex flex-col bg-muted/50">
            <div className="flex items-center justify-between p-3 border-b border-border bg-muted">
              <h3 className="text-sm font-medium text-foreground">测试历史记录</h3>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => refetch()} title="刷新记录">
                <RefreshCw className="h-4 w-4" />
              </Button>
            </div>
            <div className="flex-1 overflow-y-auto">
              {listLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
                  <span className="ml-2 text-sm text-muted-foreground">加载中...</span>
                </div>
              ) : benchmarks.length === 0 ? (
                <div className="text-sm text-muted-foreground text-center py-8 px-4">
                  暂无测试记录
                </div>
              ) : (
                <div className="divide-y divide-border">
                  {benchmarks.map((task) => (
                    <div
                      key={task.id}
                      className="p-3 hover:bg-accent transition-colors"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-foreground truncate" title={task.command}>
                            {task.status === 'running' ? '⏳ 测试中...' : task.status === 'failed' ? '❌ 失败' : '✅ 完成'}
                          </div>
                          <div className="text-xs text-muted-foreground mt-1">
                            时间: {new Date(task.createdAt).toLocaleString('zh-CN')}
                          </div>
                          {task.metrics?.tokens_per_second && (
                            <div className="text-xs font-semibold text-green-600 mt-1">
                              TPS: {Number(task.metrics.tokens_per_second).toFixed(2)}
                            </div>
                          )}
                        </div>
                        <div className="flex flex-col gap-1">
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => handleAppendResult(task)}
                            className="px-2 py-1 text-xs h-7"
                          >
                            追加
                          </Button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* 右侧：结果显示区域 */}
          <div className="flex-1 flex flex-col min-w-0">
            {/* 工具栏 */}
            <div className="flex items-center justify-between p-3 border-b border-border bg-muted/50">
              <div className="text-sm text-foreground">
                当前模型: {modelName}
              </div>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={handleClearContent}
                  disabled={!displayContent}
                >
                  清空内容
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={handleCopyToClipboard}
                  disabled={!displayContent}
                >
                  <Copy className="w-4 h-4 mr-1" />
                  复制
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={handleDownload}
                  disabled={!displayContent}
                >
                  <Download className="w-4 h-4 mr-1" />
                  下载
                </Button>
              </div>
            </div>

            {/* 内容显示区 */}
            <div className="flex-1 overflow-auto p-4 bg-slate-900">
              {displayContent ? (
                <pre className="text-sm text-slate-100 whitespace-pre-wrap font-mono">
                  {displayContent}
                </pre>
              ) : (
                <div className="flex items-center justify-center h-full text-slate-400">
                  <div className="text-center">
                    <FileText className="w-12 h-12 mx-auto mb-2 opacity-50" />
                    <p>点击左侧"追加"按钮查看测试结果</p>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* 底部状态栏 */}
        {selectedResults.length > 0 && (
          <div className="px-4 py-2 border-t border-border bg-muted/50 text-xs text-muted-foreground">
            已加载 {selectedResults.length} 个结果，共 {displayContent.length} 字符
          </div>
        )}
      </div>
    </div>
  );
}
