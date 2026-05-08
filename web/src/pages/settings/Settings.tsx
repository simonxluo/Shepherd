import { useState, useEffect, useRef, useCallback } from 'react';
import {
  Settings as SettingsIcon,
  Zap,
  Toolbox,
  Info,
  FolderOpen,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { PathConfigPanel } from '@/features/settings/components/PathConfigPanel';
import { ApiConfigCard, type ApiConfig } from '@/features/settings/components/ApiConfigCard';
import { compatibilityApi } from '@/lib/api/compatibility';
import { useServerInfo } from '@/hooks/useServerInfo';
import { useToast } from '@/hooks/useToast';

/**
 * Settings tab type
 */
type SettingsTab = 'general' | 'paths' | 'benchmark' | 'mcp' | 'about';

/**
 * Settings menu item
 */
interface SettingsMenuItem {
  id: SettingsTab;
  icon: typeof SettingsIcon;
  label: string;
}

const settingsMenuItems: SettingsMenuItem[] = [
  { id: 'general', icon: SettingsIcon, label: '通用设置' },
  { id: 'paths', icon: FolderOpen, label: '路径配置' },
  { id: 'benchmark', icon: Zap, label: '性能压测' },
  { id: 'mcp', icon: Toolbox, label: 'MCP 管理' },
  { id: 'about', icon: Info, label: '关于' },
];

/**
 * Settings page
 */
export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');

  return (
    <div className="h-full text-foreground">
      {/* Header */}
      <div className="border-b px-5 py-3">
        <h1 className="text-xl font-semibold">设置</h1>
      </div>

      {/* Settings content */}
      <div className="flex h-[calc(100%-53px)]">
        {/* Sidebar menu */}
        <div className="w-48 border-r bg-background p-3">
          <nav className="space-y-1" role="tablist" aria-label="设置菜单">
            {settingsMenuItems.map((item) => {
              const Icon = item.icon;
              const isActive = activeTab === item.id;

              return (
                <button
                  key={item.id}
                  type="button"
                  role="tab"
                  aria-selected={isActive}
                  onClick={() => setActiveTab(item.id)}
                  className={cn(
                    'flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-xs font-medium transition-all duration-200',
                    isActive
                      ? 'bg-primary text-primary-foreground shadow-sm'
                      : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                  )}
                >
                  <Icon size={16} />
                  <span>{item.label}</span>
                </button>
              );
            })}
          </nav>
        </div>

        {/* Content area */}
        <div className="flex-1 overflow-y-auto p-5">
          {activeTab === 'general' && <GeneralSettingsPanel />}
          {activeTab === 'paths' && <PathsSettingsPanel />}
          {activeTab === 'benchmark' && <BenchmarkPanel />}
          {activeTab === 'mcp' && <McpPanel />}
          {activeTab === 'about' && <AboutPanel />}
        </div>
      </div>
    </div>
  );
}

/**
 * General settings panel
 */
function GeneralSettingsPanel() {
  const toast = useToast();

  const [ollamaEnabled, setOllamaEnabled] = useState(false);
  const [ollamaPort, setOllamaPort] = useState(11434);
  const [lmstudioEnabled, setLmstudioEnabled] = useState(false);
  const [lmstudioPort, setLmstudioPort] = useState(1234);

  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'success' | 'error'>('idle');
  const [isLoading, setIsLoading] = useState(true);
  const [hasChanges, setHasChanges] = useState(false);
  const [isAutoDisabling, setIsAutoDisabling] = useState(false);

  const saveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const successTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Load config on mount
  useEffect(() => {
    const loadConfig = async () => {
      try {
        const response = await compatibilityApi.get();
        if (response.success && response.data) {
          setOllamaEnabled(response.data.ollama.enabled);
          setOllamaPort(response.data.ollama.port);
          setLmstudioEnabled(response.data.lmstudio.enabled);
          setLmstudioPort(response.data.lmstudio.port);
        }
      } catch (error) {
        console.error('加载兼容性配置失败:', error);
        toast.error('加载失败', '无法加载兼容性配置');
      } finally {
        setIsLoading(false);
      }
    };

    loadConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const markChanged = useCallback(() => {
    setHasChanges(true);
  }, []);

  // Auto-save with 2s debounce
  useEffect(() => {
    // Skip save during auto-disable
    if (isLoading || !hasChanges || isAutoDisabling) return;

    if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
    if (successTimeoutRef.current) clearTimeout(successTimeoutRef.current);

    saveTimeoutRef.current = setTimeout(async () => {
      setSaveStatus('saving');
      try {
        const response = await compatibilityApi.update({
          ollama: { enabled: ollamaEnabled, port: ollamaPort },
          lmstudio: { enabled: lmstudioEnabled, port: lmstudioPort },
        });

        if (response.success) {
          setSaveStatus('success');
          setHasChanges(false);
        } else {
          setSaveStatus('error');
          const errorMsg = response.error || '未知错误';
          const serviceName = response.service === 'ollama' ? 'Ollama API' : 'LM Studio API';

          toast.error(`${serviceName} 启动失败`, errorMsg);

          if (response.autoDisabled && response.data) {
            if (response.service === 'ollama') {
              setOllamaEnabled(response.data.ollama.enabled);
            } else if (response.service === 'lmstudio') {
              setLmstudioEnabled(response.data.lmstudio.enabled);
            }
          }
        }

        successTimeoutRef.current = setTimeout(() => {
          setSaveStatus('idle');
        }, 3000);
      } catch (error) {
        console.error('保存兼容性配置失败:', error);
        setSaveStatus('error');
        toast.error('保存失败', '无法保存兼容性配置，请检查网络连接');

        successTimeoutRef.current = setTimeout(() => {
          setSaveStatus('idle');
        }, 3000);
      }
    }, 2000);

    return () => {
      if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
      if (successTimeoutRef.current) clearTimeout(successTimeoutRef.current);
    };
  }, [ollamaEnabled, ollamaPort, lmstudioEnabled, lmstudioPort, isLoading, hasChanges, isAutoDisabling, toast]);

  const handleOllamaChange = useCallback((config: ApiConfig) => {
    setOllamaEnabled(config.enabled);
    setOllamaPort(config.port);
    markChanged();
  }, [markChanged]);

  const handleLmstudioChange = useCallback((config: ApiConfig) => {
    setLmstudioEnabled(config.enabled);
    setLmstudioPort(config.port);
    markChanged();
  }, [markChanged]);

  const handleTestConnection = useCallback(async (port: number, type: 'ollama' | 'lmstudio'): Promise<boolean> => {
    try {
      const response = await compatibilityApi.testConnection(port, type);
      return response.valid;
    } catch {
      return false;
    }
  }, []);

  const handleConnectionFailed = useCallback(async (type: 'ollama' | 'lmstudio', port: number) => {
    const serviceName = type === 'ollama' ? 'Ollama API' : 'LM Studio API';
    toast.error(
      `${serviceName} 连接失败`,
      `端口 ${port} 无响应，服务将自动禁用`
    );

    // Prevent auto-save from triggering during disable
    setIsAutoDisabling(true);

    try {
      // Disable service immediately
      const response = await compatibilityApi.update({
        ollama: {
          enabled: type === 'ollama' ? false : ollamaEnabled,
          port: ollamaPort,
        },
        lmstudio: {
          enabled: type === 'lmstudio' ? false : lmstudioEnabled,
          port: lmstudioPort,
        },
      });

      if (response.success) {
        // Update local state
        if (type === 'ollama') {
          setOllamaEnabled(false);
        } else {
          setLmstudioEnabled(false);
        }
        setHasChanges(false);
        toast.success(`${serviceName} 已禁用`, '配置已自动还原');
      } else {
        toast.error(`${serviceName} 禁用失败`, response.error || '未知错误');
      }
    } catch (error) {
      console.error('自动禁用服务失败:', error);
      toast.error('自动禁用失败', '请手动禁用服务');
    } finally {
      // Delay clearing flag to ensure state updates complete
      setTimeout(() => {
        setIsAutoDisabling(false);
      }, 100);
    }
  }, [ollamaEnabled, ollamaPort, lmstudioEnabled, lmstudioPort, toast]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-foreground">
        <div className="text-center">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent motion-reduce:animate-[spin_1.5s_linear_infinite]" />
          <p className="text-sm text-muted-foreground mt-3">加载配置中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl space-y-4 text-foreground">
      <div>
        <h2 className="text-lg font-semibold ">API 兼容性设置</h2>
        <p className="text-xs text-muted-foreground">
          配置 Ollama 和 LM Studio API 兼容层端口
        </p>
      </div>

      {/* Ollama config card */}
      <ApiConfigCard
        type="ollama"
        config={{ enabled: ollamaEnabled, port: ollamaPort }}
        onConfigChange={handleOllamaChange}
        saveStatus={saveStatus}
        onTestConnection={handleTestConnection}
        onConnectionFailed={handleConnectionFailed}
      />

      {/* LM Studio config card */}
      <ApiConfigCard
        type="lmstudio"
        config={{ enabled: lmstudioEnabled, port: lmstudioPort }}
        onConfigChange={handleLmstudioChange}
        saveStatus={saveStatus}
        onTestConnection={handleTestConnection}
        onConnectionFailed={handleConnectionFailed}
      />

      {/* Auto-save notice */}
      <div className="flex items-center justify-center py-2">
        <p className="text-xs text-muted-foreground">
          配置将自动保存
        </p>
      </div>
    </div>
  );
}

/**
 * Benchmark panel
 */
function BenchmarkPanel() {
  return (
    <div className="flex h-full items-center justify-center text-foreground">
      <div className="text-center">
        <Zap size={48} className="mx-auto mb-4 text-muted-foreground" />
        <h3 className="text-lg font-semibold">性能压测</h3>
        <p className="text-sm text-muted-foreground mt-2">
          性能压测功能开发中...
        </p>
      </div>
    </div>
  );
}

/**
 * MCP management panel
 */
function McpPanel() {
  return (
    <div className="flex h-full items-center justify-center text-foreground">
      <div className="text-center">
        <Toolbox size={48} className="mx-auto mb-4 text-muted-foreground" />
        <h3 className="text-lg font-semibold">MCP 管理</h3>
        <p className="text-sm text-muted-foreground mt-2">
          MCP (Model Context Protocol) 管理功能开发中...
        </p>
      </div>
    </div>
  );
}

/**
 * About panel
 */
function AboutPanel() {
  const { data: serverInfo, isLoading } = useServerInfo();

  const formatBuildTime = (buildTime: string | undefined) => {
    if (!buildTime || buildTime === 'unknown') return '未知';
    try {
      const date = new Date(buildTime);
      return date.toISOString().split('T')[0];
    } catch {
      return buildTime;
    }
  };

  const formatGitCommit = (commit: string | undefined) => {
    if (!commit || commit === 'unknown') return '未知';
    return commit.length > 8 ? commit.substring(0, 8) : commit;
  };

  const formatRole = (role: string | undefined) => {
    if (!role) return '未知';
    const roleMap: Record<string, string> = {
      master: '主节点',
      client: '工作节点',
      hybrid: '混合节点',
    };
    return roleMap[role] || role;
  };

  return (
    <div className="max-w-2xl mx-auto text-foreground">
      <div className="text-center mb-6">
        <div className="flex h-16 w-16 items-center justify-center rounded-xl bg-primary mx-auto mb-3 text-2xl">
          🐏
        </div>
        <h2 className="text-xl font-bold">Shepherd</h2>
        <p className="text-sm text-muted-foreground">高性能轻量级 llama.cpp 模型管理系统</p>
      </div>

      <div className="rounded-lg border bg-card p-4 space-y-2">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <div className="inline-block h-5 w-5 animate-spin rounded-full border-2 border-solid border-current border-r-transparent" />
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">版本</span>
              <span className="font-mono text-sm font-medium">
                {serverInfo?.version || '未知'}
              </span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">构建时间</span>
              <span className="font-mono text-xs">
                {formatBuildTime(serverInfo?.buildTime)}
              </span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">Git Commit</span>
              <span className="font-mono text-xs">
                {formatGitCommit(serverInfo?.gitCommit)}
              </span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">运行模式</span>
              <span className="font-mono text-xs">
                {formatRole(serverInfo?.role)}
              </span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">Go 版本</span>
              <span className="font-mono text-xs">1.25+</span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">React 版本</span>
              <span className="font-mono text-xs">19.x</span>
            </div>
            <div className="flex items-center justify-between py-1.5">
              <span className="text-sm text-muted-foreground">许可证</span>
              <span className="text-xs">Apache 2.0</span>
            </div>
          </>
        )}
      </div>

      <div className="mt-4 text-center text-xs text-muted-foreground">
        <p>© 2026 Shepherd Project. Licensed under Apache 2.0</p>
        <p className="mt-1">
          <a
            href="https://github.com/shepherd-project/shepherd"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary hover:underline"
          >
            GitHub Repository
          </a>
        </p>
      </div>
    </div>
  );
}

/**
 * Path configuration panel
 */
function PathsSettingsPanel() {
  return (
    <div className="max-w-3xl space-y-5 text-foreground">
      {/* llama.cpp path config */}
      <div className="rounded-lg border bg-card p-4">
        <PathConfigPanel type="llamacpp" />
      </div>

      <div className="border-t" />

      {/* Model path config */}
      <div className="rounded-lg border bg-card p-4">
        <PathConfigPanel type="models" />
      </div>

      <div className="border-t" />

      {/* Multimodal model path config */}
      <div className="rounded-lg border bg-card p-4">
        <PathConfigPanel type="multimodal" />
      </div>

      <div className="border-t" />

      {/* vLLM path config */}
      <div className="rounded-lg border bg-card p-4">
        <PathConfigPanel type="vllm" />
      </div>

      <div className="border-t" />

      {/* vLLM-Omni path config */}
      <div className="rounded-lg border bg-card p-4">
        <PathConfigPanel type="vllm_omni" />
      </div>
    </div>
  );
}
