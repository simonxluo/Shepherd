import { Loader2, Wand2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { SelectInput } from './SelectInput';
import type { LoadModelParams } from '@/types';
import type { SystemGPUInfo, LlamacppBackend } from '@/features/models';
import type { UnifiedNode } from '@/types';

interface BasicConfigPanelProps {
  modelName: string;
  modelPath?: string;
  params: LoadModelParams;
  onParamsChange: (params: LoadModelParams) => void;
  isLoading: boolean;
  loadingStatus: string;
  // Llama.cpp 后端
  llamacppBackends: LlamacppBackend[];
  // GPU 信息
  gpus: SystemGPUInfo[];
  // 能力相关
  capabilities: LoadModelParams['capabilities'];
  reranking: boolean;
  onCapabilityChange: (key: string, value: boolean) => void;
  autoDetectPending: boolean;
  onAutoDetect: () => void;
  hasModelId: boolean;
  // 节点
  onlineNodes: UnifiedNode[];
}

export function BasicConfigPanel({
  modelName,
  modelPath,
  params,
  onParamsChange,
  isLoading,
  loadingStatus,
  llamacppBackends,
  gpus,
  capabilities,
  reranking,
  onCapabilityChange,
  autoDetectPending,
  onAutoDetect,
  hasModelId,
  onlineNodes,
}: BasicConfigPanelProps) {
  return (
    <div className="flex-1 space-y-4 overflow-y-auto pr-2 min-h-0" aria-label="基础配置区域">
      <h3 className="text-sm font-semibold text-foreground pb-2 border-b border-border">
        基础配置
      </h3>

      {/* 模型信息 */}
      <div className="space-y-3">
        <div>
          <label className="block text-sm font-medium text-foreground mb-1">
            模型
          </label>
          <div className="px-3 py-2 bg-muted rounded-md text-foreground text-sm">
            {modelName}
          </div>
          {modelPath && (
            <div className="mt-1 text-xs text-muted-foreground truncate">
              {modelPath}
            </div>
          )}
        </div>

        {/* Llama.cpp 版本选择 */}
        <div>
          <label className="block text-sm font-medium text-foreground mb-1">
            Llama.cpp 版本
          </label>
          <SelectInput
            value={params.llamaCppPath || ''}
            onChange={(e) => onParamsChange({ ...params, llamaCppPath: e.target.value })}
            disabled={isLoading}
            className="w-full"
          >
            {llamacppBackends.length > 0 ? (
              llamacppBackends.map((backend: LlamacppBackend) => (
                <option
                  key={backend.path}
                  value={backend.path}
                  disabled={!backend.available}
                >
                  {backend.name}
                  {backend.description && ` (${backend.description})`}
                  {!backend.available && ' - 不可用'}
                </option>
              ))
            ) : (
              <option value="" disabled>
                未配置 llama.cpp 后端
              </option>
            )}
          </SelectInput>
          {llamacppBackends.length === 0 && (
            <p className="mt-1 text-xs text-yellow-600 dark:text-yellow-400">
              请在服务器配置中添加 llama.cpp 路径
            </p>
          )}
        </div>

        {/* 能力开关 */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="block text-sm font-medium text-foreground">
              能力
            </label>
            <button
              type="button"
              onClick={onAutoDetect}
              disabled={!hasModelId || autoDetectPending}
              className={cn(
                "flex items-center justify-center gap-1.5 h-[34px] px-3 py-1.5 text-sm font-medium rounded-md transition-all duration-200",
                "border shadow-sm",
                "hover:shadow-md hover:-translate-y-px active:translate-y-0",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                "bg-muted text-muted-foreground border-border hover:bg-muted/80",
                "disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:translate-y-0 disabled:hover:shadow-sm"
              )}
            >
              {autoDetectPending ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Wand2 className="w-3.5 h-3.5" />
              )}
              自动检测
            </button>
          </div>
          <div className="border border-border rounded-lg p-3 bg-card">
            <div className="space-y-2">
              {/* 聊天能力 */}
              <div className="text-xs text-muted-foreground uppercase tracking-wide mb-1">聊天能力</div>
              {[
                { key: 'thinking', label: '思考能力' },
                { key: 'tools', label: '工具使用' },
              ].map(({ key, label }) => (
                <label key={key} className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded hover:bg-accent">
                  <input
                    type="checkbox"
                    checked={capabilities?.[key as keyof NonNullable<typeof capabilities>] || false}
                    onChange={(e) => onCapabilityChange(key, e.target.checked)}
                    disabled={isLoading || (reranking || capabilities?.embedding)}
                    className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                  />
                  <span>{label}</span>
                </label>
              ))}

              {/* 非聊天能力 */}
              <div className="text-xs text-muted-foreground uppercase tracking-wide mb-1 mt-2">非聊天能力</div>
              {[
                { key: 'translation', label: '直译' },
                { key: 'embedding', label: '嵌入' },
              ].map(({ key, label }) => (
                <label key={key} className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded hover:bg-accent">
                  <input
                    type="checkbox"
                    checked={capabilities?.[key as keyof NonNullable<typeof capabilities>] || false}
                    onChange={(e) => onCapabilityChange(key, e.target.checked)}
                    disabled={isLoading || ((capabilities?.thinking || capabilities?.tools) && key === 'embedding')}
                    className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                  />
                  <span>{label}</span>
                </label>
              ))}
            </div>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            选择模型支持的功能能力（互斥：思考/工具与嵌入）
          </p>
        </div>

        {/* 主GPU选择 */}
        <div>
          <label className="block text-sm font-medium text-foreground mb-1">
            主GPU
          </label>
          <SelectInput
            value={params.mainGpu || 'default'}
            onChange={(e) => onParamsChange({ ...params, mainGpu: e.target.value })}
            disabled={isLoading || gpus.length === 0}
            className="w-full"
          >
            <option value="default">默认</option>
            {gpus.map((gpu: SystemGPUInfo) => (
              <option key={gpu.id} value={gpu.id}>
                {gpu.name}
              </option>
            ))}
          </SelectInput>
          {gpus.length === 0 && (
            <p className="mt-1 text-xs text-yellow-600 dark:text-yellow-400">
              未检测到GPU，请确保ROCm正确安装
            </p>
          )}
        </div>

        {/* 设备选择 */}
        <div>
          <label className="block text-sm font-medium text-foreground mb-2">
            设备
          </label>
          <div className="border border-border rounded-lg p-3 bg-card">
            {gpus.length > 0 ? (
              <div className="space-y-1">
                {gpus.map((gpu: SystemGPUInfo) => (
                  <label
                    key={gpu.id}
                    className="flex items-center gap-2 text-sm text-foreground cursor-pointer hover:bg-accent p-1 rounded"
                  >
                    <input
                      type="checkbox"
                      checked={gpu.available}
                      disabled={true}
                      className="rounded border-border text-blue-600 focus:ring-blue-500 w-4 h-4"
                    />
                    <span className="flex-1">
                      <span className="font-medium">{gpu.id}</span>
                      <span className="text-muted-foreground mx-1">·</span>
                      <span className="text-muted-foreground">{gpu.name}</span>
                      {gpu.totalMemory && (
                        <span className="ml-2 text-muted-foreground">
                          总计 {gpu.totalMemory}
                        </span>
                      )}
                      {gpu.freeMemory && (
                        <span className="ml-2 text-green-600 dark:text-green-400">
                          可用 {gpu.freeMemory}
                        </span>
                      )}
                    </span>
                    {gpu.available ? (
                      <span className="text-xs text-green-600 dark:text-green-400">就绪</span>
                    ) : (
                      <span className="text-xs text-red-600 dark:text-red-400">不可用</span>
                    )}
                  </label>
                ))}
              </div>
            ) : (
              <div className="text-sm text-muted-foreground text-center py-2">
                未检测到GPU设备
              </div>
            )}
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            选择用于模型加载的设备
          </p>
        </div>

        {/* 节点选择（仅多节点环境显示） */}
        {onlineNodes.length > 0 && (
          <div>
            <label className="block text-sm font-medium text-foreground mb-1">
              运行节点
            </label>
            <SelectInput
              value={params.nodeId || 'auto'}
              onChange={(e) => onParamsChange({
                ...params,
                nodeId: e.target.value === 'auto' ? undefined : e.target.value
              })}
              disabled={isLoading}
              className="w-full"
            >
              <option value="auto">
                🎯 自动调度（推荐）- 系统选择最佳节点
              </option>
              <optgroup label="指定节点">
                {onlineNodes.map((node: UnifiedNode) => (
                  <option key={node.id} value={node.id}>
                    {node.name} ({node.address}:{node.port})
                    {node.capabilities?.gpuCount && ` · ${node.capabilities.gpuCount} GPU`}
                    {node.resources?.gpuInfo?.[0] &&
                      ` · 显存 ${Math.round((node.resources.gpuInfo[0].totalMemory - node.resources.gpuInfo[0].usedMemory) / 1024 / 1024 / 1024)}GB 可用`
                    }
                  </option>
                ))}
              </optgroup>
            </SelectInput>
            <p className="mt-1 text-xs text-muted-foreground">
              自动调度会根据 GPU 显存和负载选择最佳节点
            </p>
          </div>
        )}

        {/* 设备状态 */}
        <div>
          <label className="block text-sm font-medium text-foreground mb-1">
            加载状态
          </label>
          <div className="px-3 py-2 bg-muted rounded-md min-h-[40px] text-sm">
            {isLoading && loadingStatus === '加载中...' ? (
              <span className="flex items-center gap-2">
                <Loader2 className="w-4 h-4 animate-spin" />
                {loadingStatus}
              </span>
            ) : (
              <span className="text-muted-foreground">{loadingStatus}</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
