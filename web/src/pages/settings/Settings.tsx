import React from 'react';
import { useState, useEffect, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Settings as SettingsIcon,
  Zap,
  Toolbox,
  Info,
  FolderOpen,
  Globe,
  Shield,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { PathConfigPanel } from '@/features/settings/components/PathConfigPanel';
import { ApiConfigCard, type ApiConfig } from '@/features/settings/components/ApiConfigCard';
import { McpSettingsPanel } from '@/features/settings/components/McpSettingsPanel';
import { compatibilityApi } from '@/lib/api/compatibility';
import { apiClient } from '@/lib/api/client';
import { useServerInfo } from '@/hooks/useServerInfo';
import { toast } from '@/hooks/useToast';

/**
 * Settings tab type
 */
type SettingsTab = 'general' | 'paths' | 'benchmark' | 'mcp' | 'about';

const settingsMenuItems = [
  { id: 'general', icon: SettingsIcon, labelKey: 'settings.menu.general' },
  { id: 'paths', icon: FolderOpen, labelKey: 'settings.menu.paths' },
  { id: 'benchmark', icon: Zap, labelKey: 'settings.menu.benchmark' },
  { id: 'mcp', icon: Toolbox, labelKey: 'settings.menu.mcp' },
  { id: 'about', icon: Info, labelKey: 'settings.menu.about' },
] as const;

/**
 * Settings page
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');

  return (
    <div className="h-full text-foreground">
      {/* Header */}
      <div className="border-b px-5 py-3">
        <h1 className="text-xl font-semibold">{t('settings.title')}</h1>
      </div>

      {/* Settings content */}
      <div className="flex h-[calc(100%-53px)]">
        {/* Sidebar menu */}
        <div className="w-48 border-r bg-background p-3">
          <nav className="space-y-1" role="tablist" aria-label={t('settings.menu.ariaLabel')}>
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
                  <span>{t(item.labelKey)}</span>
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
          {activeTab === 'mcp' && <McpSettingsPanel />}
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
  const { t } = useTranslation();
  const [ollamaEnabled, setOllamaEnabled] = useState(false);
  const [ollamaPort, setOllamaPort] = useState(11434);
  const [lmstudioEnabled, setLmstudioEnabled] = useState(false);
  const [lmstudioPort, setLmstudioPort] = useState(1234);
  const [modelBindHost, setModelBindHost] = useState('0.0.0.0');
  const [bindHostLoading, setBindHostLoading] = useState(false);

  const [saveDone, setSaveDone] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [hasChanges, setHasChanges] = useState(false);
  const [isAutoDisabling, setIsAutoDisabling] = useState(false);

  const saveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Load config on mount
  useEffect(() => {
    const loadConfig = async () => {
      try {
        const [compatResponse, configResponse] = await Promise.all([
          compatibilityApi.get(),
          apiClient.get<{ success: boolean; data?: { server?: { model_bind_host?: string } } }>('/config'),
        ]);
        if (compatResponse.success && compatResponse.data) {
          setOllamaEnabled(compatResponse.data.ollama.enabled);
          setOllamaPort(compatResponse.data.ollama.port);
          setLmstudioEnabled(compatResponse.data.lmstudio.enabled);
          setLmstudioPort(compatResponse.data.lmstudio.port);
        }
        if (configResponse.success && configResponse.data?.server?.model_bind_host) {
          setModelBindHost(configResponse.data.server.model_bind_host);
        }
      } catch (error) {
        console.error('加载兼容性配置失败:', error);
        toast.error(t('settings.loadFailed'), t('settings.loadFailedDesc'));
      } finally {
        setIsLoading(false);
      }
    };

    loadConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const markChanged = useCallback(() => {
    setHasChanges(true);
    setSaveDone(false);
  }, []);

  // Auto-save with 2s debounce
  useEffect(() => {
    // Skip save during auto-disable
    if (isLoading || !hasChanges || isAutoDisabling) return;

    if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);

    saveTimeoutRef.current = setTimeout(async () => {
      try {
        const response = await compatibilityApi.update({
          ollama: { enabled: ollamaEnabled, port: ollamaPort },
          lmstudio: { enabled: lmstudioEnabled, port: lmstudioPort },
        });

        if (response.success) {
          setSaveDone(true);
          setHasChanges(false);
        } else {
          const errorMsg = response.error || t('common.unknownError');
          const serviceName = response.service === 'ollama' ? 'Ollama API' : 'LM Studio API';

          toast.error(t('settings.serviceFailed', { service: serviceName }), errorMsg);

          if (response.autoDisabled && response.data) {
            if (response.service === 'ollama') {
              setOllamaEnabled(response.data.ollama.enabled);
            } else if (response.service === 'lmstudio') {
              setLmstudioEnabled(response.data.lmstudio.enabled);
            }
          }
        }
      } catch (error) {
        console.error('保存兼容性配置失败:', error);
        toast.error(t('settings.saveFailed'), t('settings.saveFailedDesc'));
      }
    }, 2000);

    return () => {
      if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
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
      t('settings.connectionFailed', { service: serviceName }),
      t('settings.connectionFailedDesc', { port })
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
        toast.success(t('settings.serviceDisabled', { service: serviceName }), t('settings.serviceDisabledDesc'));
      } else {
        toast.error(t('settings.disableFailed', { service: serviceName }), response.error || t('common.unknownError'));
      }
    } catch (error) {
      console.error('自动禁用服务失败:', error);
      toast.error(t('settings.autoDisableFailed'), t('settings.autoDisableFailedDesc'));
    } finally {
      // Delay clearing flag to ensure state updates complete
      setTimeout(() => {
        setIsAutoDisabling(false);
      }, 100);
    }
  }, [ollamaEnabled, ollamaPort, lmstudioEnabled, lmstudioPort, toast]);

  const handleBindHostChange = useCallback(async (host: string) => {
    if (host === modelBindHost) return;
    setBindHostLoading(true);
    try {
      const response = await apiClient.put<{ success: boolean; data?: { restart_required?: boolean }; error?: string }>('/config', {
        model_bind_host: host,
      });
      if (response.success) {
        setModelBindHost(host);
        toast.success(
          t('settings.bindHost.saveSuccess'),
          t('settings.bindHost.restartHint')
        );
      } else {
        toast.error(t('settings.bindHost.saveFailed'), response.error || '');
      }
    } catch (error) {
      console.error('更新绑定地址失败:', error);
      toast.error(t('settings.bindHost.saveFailed'), t('settings.saveFailedDesc'));
    } finally {
      setBindHostLoading(false);
    }
  }, [modelBindHost, t]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-foreground">
        <div className="text-center">
          <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-current border-r-transparent motion-reduce:animate-[spin_1.5s_linear_infinite]" />
          <p className="text-sm text-muted-foreground mt-3">{t('settings.loadingConfig')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl space-y-4 text-foreground">
      <div>
        <h2 className="text-lg font-semibold ">{t('settings.apiCompatibility')}</h2>
        <p className="text-xs text-muted-foreground">
          {t('settings.apiCompatibilityDesc')}
        </p>
      </div>

      {/* Ollama config card */}
      <ApiConfigCard
        type="ollama"
        config={{ enabled: ollamaEnabled, port: ollamaPort }}
        onConfigChange={handleOllamaChange}
        saveDone={saveDone}
        onTestConnection={handleTestConnection}
        onConnectionFailed={handleConnectionFailed}
      />

      {/* LM Studio config card */}
      <ApiConfigCard
        type="lmstudio"
        config={{ enabled: lmstudioEnabled, port: lmstudioPort }}
        onConfigChange={handleLmstudioChange}
        saveDone={saveDone}
        onTestConnection={handleTestConnection}
        onConnectionFailed={handleConnectionFailed}
      />

      {/* Model bind host config */}
      <div className="mt-6">
        <h2 className="text-lg font-semibold">{t('settings.bindHost.title')}</h2>
        <p className="text-xs text-muted-foreground mb-3">
          {t('settings.bindHost.description')}
        </p>
        <div className="flex gap-3">
          <button
            type="button"
            disabled={bindHostLoading}
            onClick={() => handleBindHostChange('0.0.0.0')}
            className={cn(
              'flex-1 flex items-center gap-2 rounded-lg border-2 p-3 transition-all',
              modelBindHost === '0.0.0.0'
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-muted-foreground/50'
            )}
          >
            <Globe size={18} className={modelBindHost === '0.0.0.0' ? 'text-primary' : 'text-muted-foreground'} />
            <div className="text-left">
              <div className="text-sm font-medium">0.0.0.0</div>
              <div className="text-xs text-muted-foreground">{t('settings.bindHost.allInterfaces')}</div>
            </div>
          </button>
          <button
            type="button"
            disabled={bindHostLoading}
            onClick={() => handleBindHostChange('127.0.0.1')}
            className={cn(
              'flex-1 flex items-center gap-2 rounded-lg border-2 p-3 transition-all',
              modelBindHost === '127.0.0.1'
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-muted-foreground/50'
            )}
          >
            <Shield size={18} className={modelBindHost === '127.0.0.1' ? 'text-primary' : 'text-muted-foreground'} />
            <div className="text-left">
              <div className="text-sm font-medium">127.0.0.1</div>
              <div className="text-xs text-muted-foreground">{t('settings.bindHost.localhostOnly')}</div>
            </div>
          </button>
        </div>
      </div>

    </div>
  );
}

/**
 * Benchmark panel
 */
function BenchmarkPanel() {
  const { t } = useTranslation();
  return (
    <div className="flex h-full items-center justify-center text-foreground">
      <div className="text-center">
        <Zap size={48} className="mx-auto mb-4 text-muted-foreground" />
        <h3 className="text-lg font-semibold">{t('settings.benchmark.title')}</h3>
        <p className="text-sm text-muted-foreground mt-2">
          {t('settings.benchmark.inDevelopment')}
        </p>
      </div>
    </div>
  );
}

/**
 * About panel
 */
function AboutPanel() {
  const { t } = useTranslation();
  const { data: serverInfo, isLoading } = useServerInfo();

  const formatBuildTime = (buildTime: string | undefined) => {
    if (!buildTime || buildTime === 'unknown') return t('settings.about.unknown');
    try {
      const date = new Date(buildTime);
      return date.toISOString().split('T')[0];
    } catch {
      return buildTime;
    }
  };

  const formatGitCommit = (commit: string | undefined) => {
    if (!commit || commit === 'unknown') return t('settings.about.unknown');
    return commit.length > 8 ? commit.substring(0, 8) : commit;
  };

  const formatRole = (role: string | undefined) => {
    if (!role) return t('settings.about.unknown');
    const roleMap: Record<string, string> = {
      master: t('settings.about.roleMaster'),
      client: t('settings.about.roleClient'),
      hybrid: t('settings.about.roleHybrid'),
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
        <p className="text-sm text-muted-foreground">{t('settings.about.description')}</p>
      </div>

      <div className="rounded-lg border bg-card p-4 space-y-2">
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <div className="inline-block h-5 w-5 animate-spin rounded-full border-2 border-solid border-current border-r-transparent" />
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">{t('settings.about.version')}</span>
              <span className="font-mono text-sm font-medium">
                {serverInfo?.version || t('settings.about.unknown')}
              </span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">{t('settings.about.buildTime')}</span>
              <span className="font-mono text-xs">
                {formatBuildTime(serverInfo?.buildTime)}
              </span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">{t('settings.about.gitCommit')}</span>
              <span className="font-mono text-xs">
                {formatGitCommit(serverInfo?.gitCommit)}
              </span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">{t('settings.about.runMode')}</span>
              <span className="font-mono text-xs">
                {formatRole(serverInfo?.role)}
              </span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">{t('settings.about.goVersion')}</span>
              <span className="font-mono text-xs">{serverInfo?.goVersion || t('settings.about.unknown')}</span>
            </div>
            <div className="flex items-center justify-between py-1.5 border-b">
              <span className="text-sm text-muted-foreground">{t('settings.about.reactVersion')}</span>
              <span className="font-mono text-xs">{React.version}</span>
            </div>
            <div className="flex items-center justify-between py-1.5">
              <span className="text-sm text-muted-foreground">{t('settings.about.license')}</span>
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
