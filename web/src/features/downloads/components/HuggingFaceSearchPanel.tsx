import { useState } from 'react';
import { Search, Download, ExternalLink, Loader2, Settings, Key, Globe, File, Filter } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useHuggingFaceSearch, useModelRepoConfig, useAvailableEndpoints, useUpdateModelRepoConfig, useModelFiles } from '@/features/downloads/hooks';
import type { HuggingFaceModel } from '@/lib/api/downloads';
import { cn } from '@/lib/utils';

interface HuggingFaceSearchPanelProps {
  onDownload: (model: HuggingFaceModel, fileName?: string) => void;
}

export function HuggingFaceSearchPanel({ onDownload }: HuggingFaceSearchPanelProps) {
  const [query, setQuery] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [showSettings, setShowSettings] = useState(false);
  const [tokenInput, setTokenInput] = useState('');
  const [expandedModelId, setExpandedModelId] = useState<string | null>(null);
  const [pageSize, setPageSize] = useState<number>(20);
  const [searchFormat, setSearchFormat] = useState<string>('gguf'); // Default to GGUF
  
  const { data: searchResult, isLoading, error } = useHuggingFaceSearch(query, pageSize, searchFormat);
  const { data: config, isLoading: configLoading } = useModelRepoConfig();
  const { data: endpoints } = useAvailableEndpoints();
  const updateConfig = useUpdateModelRepoConfig();
  const handleSaveSettings = () => {
    const updates: { endpoint?: string; token?: string } = {};
    if (config && endpoints) {
      const selectedEndpoint = (document.getElementById('endpoint-select') as HTMLSelectElement)?.value;
      if (selectedEndpoint && selectedEndpoint !== config.endpoint) {
        updates.endpoint = selectedEndpoint;
      }
    }
    if (tokenInput) {
      updates.token = tokenInput;
    }
    if (Object.keys(updates).length > 0) {
      updateConfig.mutate(updates, {
        onSuccess: () => {
          setTokenInput('');
          setShowSettings(false);
        }
      });
    } else {
      setShowSettings(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchInput.trim()) {
      setQuery(searchInput.trim());
    }
  };

  const formatNumber = (num: number): string => {
    if (num >= 1000000) {
      return (num / 1000000).toFixed(1) + 'M';
    }
    if (num >= 1000) {
      return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
  };

  const formatFileSize = (bytes: number): string => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  return (
    <div className="space-y-4">
      {/* 搜索框和筛选器 */}
      <div className="flex items-center gap-2">
        <form onSubmit={handleSearch} className="flex gap-2 flex-1 items-center">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="搜索 HuggingFace 模型，如：llama、qwen、mistral..."
              className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-input text-foreground placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          
          {/* 每页条数筛选器 */}
          <select
            value={pageSize}
            onChange={(e) => {
              setPageSize(Number(e.target.value));
              if (searchInput.trim()) {
                setQuery(searchInput.trim());
              }
            }}
            className="px-3 py-2 border border-border rounded-md bg-input text-foreground text-sm"
            title="每页条数"
          >
            <option value={10}>10 条</option>
            <option value={20}>20 条</option>
            <option value={50}>50 条</option>
            <option value={100}>100 条</option>
          </select>
          
          {/* 格式筛选器 */}
          <select
            value={searchFormat}
            onChange={(e) => {
              setSearchFormat(e.target.value);
              if (searchInput.trim()) {
                setQuery(searchInput.trim());
              }
            }}
            className="px-3 py-2 border border-border rounded-md bg-input text-foreground text-sm"
            title="文件格式"
          >
            <option value="gguf">GGUF</option>
            <option value="safetensors">SafeTensors</option>
            <option value="onnx">ONNX</option>
            <option value="mlx">MLX</option>
            <option value="all">所有格式</option>
          </select>
          
          <Button
            type="submit"
            disabled={isLoading || !searchInput.trim()}
            className="whitespace-nowrap"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                搜索中...
              </>
            ) : (
              <>
                <Search className="w-4 h-4 mr-2" />
                搜索
              </>
            )}
          </Button>
        </form>

        {/* 设置按钮 */}
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setShowSettings(!showSettings)}
          title="搜索设置"
        >
          <Settings className="w-4 h-4" />
        </Button>
      </div>

      {/* 设置面板 */}
      {showSettings && (
        <div className="p-4 bg-card rounded-lg border border-border space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-foreground flex items-center gap-2">
              <Settings className="w-4 h-4" />
              搜索设置
            </h3>
            <button
              onClick={() => setShowSettings(false)}
              className="text-muted-foreground hover:text-foreground"
            >
              ✕
            </button>
          </div>
          
          {/* 端点选择 */}
          <div>
            <label className="flex items-center gap-2 text-sm font-medium text-foreground mb-1">
              <Globe className="w-4 h-4" />
              API 端点
            </label>
            <select
              id="endpoint-select"
              defaultValue={config?.endpoint || 'huggingface.co'}
              disabled={configLoading}
              className="w-full px-3 py-2 border border-border rounded-md bg-input text-foreground"
            >
              {endpoints && Object.entries(endpoints).map(([value, label]) => (
                <option key={value} value={value}>
                  {label} ({value})
                </option>
              ))}
              {!endpoints && (
                <>
                  <option value="huggingface.co">HuggingFace 官方 (huggingface.co)</option>
                  <option value="hf-mirror.com">HuggingFace 镜像 (hf-mirror.com)</option>
                </>
              )}
            </select>
            <p className="text-xs text-muted-foreground mt-1">
              如果官方站点访问缓慢，可尝试使用镜像站点
            </p>
          </div>

          {/* Token 配置 */}
          <div>
            <label className="flex items-center gap-2 text-sm font-medium text-foreground mb-1">
              <Key className="w-4 h-4" />
              Access Token (可选)
            </label>
            <input
              type="password"
              value={tokenInput}
              onChange={(e) => setTokenInput(e.target.value)}
              placeholder={config?.token ? '已配置 (输入新值替换)' : 'hf_...'}
              className="w-full px-3 py-2 border border-border rounded-md bg-input text-foreground"
            />
            <p className="text-xs text-muted-foreground mt-1">
              用于访问私有模型或提高速率限制，可在 HuggingFace 设置页面获取
            </p>
          </div>

          <div className="flex justify-end gap-2 pt-2 border-t border-border">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowSettings(false)}
            >
              取消
            </Button>
            <Button
              size="sm"
              onClick={handleSaveSettings}
              disabled={updateConfig.isPending}
            >
              {updateConfig.isPending ? '保存中...' : '保存'}
            </Button>
          </div>
        </div>
      )}

      {/* 搜索结果 */}
      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400">
          搜索失败: {error.message}
          {error.message?.includes('timeout') && (
            <p className="text-sm mt-1">
              建议尝试切换至镜像站点，点击上方的设置按钮修改端点
            </p>
          )}
        </div>
      )}

      {searchResult && searchResult.models.length === 0 && (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Search className="w-12 h-12 mb-4" />
          <p className="text-lg">未找到模型</p>
          <p className="text-sm">尝试使用不同的关键词搜索</p>
        </div>
      )}

      {searchResult && searchResult.models.length > 0 && (
        <div className="space-y-3">
          <div className="text-sm text-muted-foreground">
            找到 {searchResult.total} 个模型，显示前 {searchResult.models.length} 个
          </div>
          
          {searchResult.models.map((model) => (
            <div
              key={model.id}
              className="p-4 bg-card rounded-lg border border-border hover:border-blue-300 dark:hover:border-blue-700 transition-colors"
            >
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                  <h3 className="font-medium text-foreground truncate">
                    {model.modelId}
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    作者: {model.author}
                  </p>
                  
                  {/* 标签 */}
                  {model.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-2">
                      {model.tags.slice(0, 5).map((tag) => (
                        <span
                          key={tag}
                          className="px-2 py-0.5 text-xs bg-muted text-muted-foreground rounded"
                        >
                          {tag}
                        </span>
                      ))}
                      {model.tags.length > 5 && (
                        <span className="px-2 py-0.5 text-xs text-muted-foreground">
                          +{model.tags.length - 5}
                        </span>
                      )}
                    </div>
                  )}

                  {/* 统计 */}
                  <div className="flex items-center gap-4 mt-2 text-sm text-muted-foreground">
                    <span>下载: {formatNumber(model.downloads)}</span>
                    <span>点赞: {formatNumber(model.likes)}</span>
                  </div>
                </div>

                {/* 操作按钮 */}
                <div className="flex flex-col gap-2">
                  <Button
                    onClick={() => setExpandedModelId(expandedModelId === model.modelId ? null : model.modelId)}
                    size="sm"
                    className="whitespace-nowrap"
                  >
                    <Download className="w-4 h-4 mr-1" />
                    {expandedModelId === model.modelId ? '收起' : '下载'}
                  </Button>
                  <a
                    href={`https://${config?.endpoint || 'huggingface.co'}/${model.modelId}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={cn(
                      'inline-flex items-center justify-center px-3 py-1.5 text-sm font-medium rounded-md',
                      'bg-muted text-foreground',
                      'hover:bg-muted transition-colors'
                    )}
                  >
                    <ExternalLink className="w-4 h-4 mr-1" />
                    查看
                  </a>
                </div>
              </div>

              {/* 展开的文件列表 */}
              {expandedModelId === model.modelId && (
                <ModelFilesList 
                  model={model} 
                  onDownload={onDownload} 
                  formatFileSize={formatFileSize} 
                />
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ModelFilesList({ 
  model, 
  onDownload, 
  formatFileSize
  }: { 
  model: HuggingFaceModel; 
  onDownload: (model: HuggingFaceModel, fileName?: string) => void;
  formatFileSize: (bytes: number) => string;
  }) {
  const { data: files, isLoading, error } = useModelFiles('huggingface', model.modelId);
  const [fileFormat, setFileFormat] = useState<string>('gguf'); // Default to GGUF


  // 根据文件格式筛选
  const filteredFiles = files?.filter((file) => {
    if (fileFormat === 'all') return true;
    if (fileFormat === 'gguf') return file.name.toLowerCase().endsWith('.gguf');
    if (fileFormat === 'safetensors') return file.name.toLowerCase().endsWith('.safetensors');
    if (fileFormat === 'onnx') return file.name.toLowerCase().endsWith('.onnx');
    if (fileFormat === 'mlx') return file.name.toLowerCase().endsWith('.mlx') || file.name.toLowerCase().includes('mlx');
    if (fileFormat === 'other') {
      const lower = file.name.toLowerCase();
      return !lower.endsWith('.gguf') && !lower.endsWith('.safetensors') && !lower.endsWith('.mlx');
    }
    return true;
  });

  return (
    <div className="mt-4 pt-4 border-t border-border">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <h4 className="text-sm font-medium text-foreground flex items-center gap-2">
            <File className="w-4 h-4" />
            可用模型文件
          </h4>
          <select
            value={fileFormat}
            onChange={(e) => setFileFormat(e.target.value)}
            className="px-2 py-1 border border-border rounded text-xs bg-input text-foreground"
            title="文件格式"
          >
            <option value="gguf">GGUF</option>
            <option value="safetensors">SafeTensors</option>
            <option value="onnx">ONNX</option>
            <option value="mlx">MLX</option>
            <option value="all">所有格式</option>
          </select>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onDownload(model)}
          className="text-xs h-7"
        >
          手动输入文件名
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-6 text-sm text-muted-foreground">
          <Loader2 className="w-4 h-4 animate-spin mr-2" />
          加载文件列表...
        </div>
      ) : error ? (
        <div className="text-sm text-red-600 dark:text-red-400 py-2">
          加载失败: {error.message}
        </div>
      ) : !filteredFiles || filteredFiles.length === 0 ? (
        <div className="text-sm text-muted-foreground py-4 text-center bg-muted/50 rounded-md">
          {fileFormat === 'all' ? '未找到模型文件' : `未找到 ${fileFormat} 格式的模型文件`}
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-60 overflow-y-auto pr-2 custom-scrollbar">
          {filteredFiles.map((file) => {
            // 尝试从文件名中提取量化精度 (如 Q4_K_M, Q8_0)
            const quantMatch = file.name.match(/(q[1-8]_[0-1k_a-z]+|f16|f32)/i);
            const quant = quantMatch ? quantMatch[1].toUpperCase() : null;
            
            return (
              <div
                key={file.name}
                className="flex items-center justify-between p-2 rounded-md border border-border bg-card hover:border-blue-300 dark:hover:border-blue-700 transition-colors group"
              >
                <div className="flex-1 min-w-0 mr-3">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium truncate" title={file.name}>
                      {file.name}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                    <span>{formatFileSize(file.size)}</span>
                    {quant && (
                      <span className="px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded font-mono">
                        {quant}
                      </span>
                    )}
                  </div>
                </div>
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => onDownload(model, file.name)}
                  className="opacity-0 group-hover:opacity-100 transition-opacity h-8 px-3"
                >
                  下载
                </Button>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
