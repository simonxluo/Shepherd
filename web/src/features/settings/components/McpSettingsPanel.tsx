import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, RefreshCw, Trash2, Server, Wrench, Settings2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  listMCPServers,
  addMCPServer,
  removeMCPServer,
  refreshMCPServer,
  getMCPConfig,
  updateMCPConfig,
  type MCPServerInfo,
  type MCPServerConfig,
  type MCPConfig,
  type MCPTransportType,
} from '@/lib/api/mcp';

export function McpSettingsPanel() {
  const { t } = useTranslation();
  const [servers, setServers] = useState<MCPServerInfo[]>([]);
  const [config, setConfig] = useState<MCPConfig | null>(null);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshingId, setRefreshingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [serversRes, configRes] = await Promise.all([
        listMCPServers(),
        getMCPConfig(),
      ]);
      setServers(serversRes.servers || []);
      setConfig(configRes);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load MCP data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleAddServer = async (serverConfig: MCPServerConfig) => {
    try {
      setError(null);
      await addMCPServer(serverConfig);
      setShowAddDialog(false);
      await fetchData();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to add server');
    }
  };

  const handleRemoveServer = async (id: string) => {
    try {
      setError(null);
      await removeMCPServer(id);
      await fetchData();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to remove server');
    }
  };

  const handleRefreshServer = async (id: string) => {
    try {
      setError(null);
      setRefreshingId(id);
      await refreshMCPServer(id);
      await fetchData();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to refresh server');
    } finally {
      setRefreshingId(null);
    }
  };

  const handleConfigChange = async (newConfig: MCPConfig) => {
    try {
      setError(null);
      await updateMCPConfig(newConfig);
      setConfig(newConfig);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to update config');
    }
  };

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center">
        <RefreshCw className="animate-spin text-muted-foreground" size={24} />
      </div>
    );
  }

  return (
    <div className="space-y-6 p-4 overflow-y-auto h-full">
      {/* Error Banner */}
      {error && (
        <div className="flex items-center justify-between rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="ml-2 text-xs hover:underline">✕</button>
        </div>
      )}
      {/* MCP Server Mode Toggle */}
      {config && (
        <section className="space-y-3">
          <div className="flex items-center gap-2">
            <Settings2 size={18} className="text-muted-foreground" />
            <h3 className="text-sm font-medium">{t('settings.mcp.serverMode')}</h3>
          </div>
          <p className="text-xs text-muted-foreground">{t('settings.mcp.serverModeDesc')}</p>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
            <ToggleItem
              label={t('settings.mcp.exposeTts')}
              checked={config.server.exposeTts}
              onChange={(v) =>
                handleConfigChange({ ...config, server: { ...config.server, exposeTts: v } })
              }
            />
            <ToggleItem
              label={t('settings.mcp.exposeAsr')}
              checked={config.server.exposeAsr}
              onChange={(v) =>
                handleConfigChange({ ...config, server: { ...config.server, exposeAsr: v } })
              }
            />
            <ToggleItem
              label={t('settings.mcp.exposeChat')}
              checked={config.server.exposeChat}
              onChange={(v) =>
                handleConfigChange({ ...config, server: { ...config.server, exposeChat: v } })
              }
            />
          </div>
          <ToggleItem
            label={t('settings.mcp.serverMode')}
            checked={config.server.enabled}
            onChange={(v) =>
              handleConfigChange({ ...config, server: { ...config.server, enabled: v } })
            }
          />
        </section>
      )}

      {/* MCP Client Settings */}
      {config && (
        <section className="space-y-3">
          <div className="flex items-center gap-2">
            <Wrench size={18} className="text-muted-foreground" />
            <h3 className="text-sm font-medium">{t('settings.mcp.clientSettings')}</h3>
          </div>
          <ToggleItem
            label={t('settings.mcp.clientEnabled')}
            checked={config.client.enabled}
            onChange={(v) =>
              handleConfigChange({ ...config, client: { ...config.client, enabled: v } })
            }
          />
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label className="text-xs text-muted-foreground">{t('settings.mcp.callTimeout')}</label>
              <input
                type="number"
                min={10}
                max={600}
                value={config.client.callTimeout}
                onChange={(e) =>
                  handleConfigChange({
                    ...config,
                    client: { ...config.client, callTimeout: parseInt(e.target.value) || 120 },
                  })
                }
                className="w-full mt-1 px-2 py-1 border rounded-md bg-background text-sm"
              />
            </div>
            <div>
              <label className="text-xs text-muted-foreground">{t('settings.mcp.readyTimeout')}</label>
              <input
                type="number"
                min={5}
                max={120}
                value={config.client.readyTimeout}
                onChange={(e) =>
                  handleConfigChange({
                    ...config,
                    client: { ...config.client, readyTimeout: parseInt(e.target.value) || 30 },
                  })
                }
                className="w-full mt-1 px-2 py-1 border rounded-md bg-background text-sm"
              />
            </div>
          </div>
        </section>
      )}

      {/* MCP Servers List */}
      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Server size={18} className="text-muted-foreground" />
            <h3 className="text-sm font-medium">{t('settings.mcp.servers')}</h3>
          </div>
          <button
            onClick={() => setShowAddDialog(true)}
            className="flex items-center gap-1 text-xs px-2 py-1 rounded bg-primary text-primary-foreground hover:bg-primary/90"
          >
            <Plus size={14} />
            {t('settings.mcp.addServer')}
          </button>
        </div>

        {servers.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            {t('settings.mcp.noServers')}
          </p>
        ) : (
          <div className="space-y-2">
            {servers.map((server) => (
              <ServerCard
                key={server.id}
                server={server}
                refreshing={refreshingId === server.id}
                onRefresh={() => handleRefreshServer(server.id)}
                onRemove={() => handleRemoveServer(server.id)}
              />
            ))}
          </div>
        )}
      </section>

      {/* Add Server Dialog */}
      {showAddDialog && (
        <AddServerDialog
          onAdd={handleAddServer}
          onClose={() => setShowAddDialog(false)}
        />
      )}
    </div>
  );
}

function ServerCard({
  server,
  refreshing,
  onRefresh,
  onRemove,
}: {
  server: MCPServerInfo;
  refreshing: boolean;
  onRefresh: () => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const toolCount = server.tools?.length ?? 0;

  return (
    <div className="border rounded-lg p-3 space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <StatusDot status={server.status} />
          <span className="text-sm font-medium">{server.name}</span>
          <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
            {server.type}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={onRefresh}
            disabled={refreshing}
            className="p-1 rounded hover:bg-muted disabled:opacity-50"
            title={t('settings.mcp.refreshTools')}
          >
            <RefreshCw size={14} className={refreshing ? 'animate-spin' : ''} />
          </button>
          <button
            onClick={onRemove}
            className="p-1 rounded hover:bg-destructive/10 text-destructive"
            title={t('settings.mcp.removeServer')}
          >
            <Trash2 size={14} />
          </button>
        </div>
      </div>
      <p className="text-xs text-muted-foreground truncate">{server.url}</p>
      {server.error && (
        <p className="text-xs text-destructive">{server.error}</p>
      )}
      <div className="flex items-center gap-1 text-xs text-muted-foreground">
        <Wrench size={12} />
        <span>{t('settings.mcp.toolCount', { count: toolCount })}</span>
      </div>
      {toolCount > 0 && (
        <div className="flex flex-wrap gap-1">
          {server.tools!.map((tool) => (
            <span
              key={tool.name}
              className="text-xs bg-muted px-1.5 py-0.5 rounded"
              title={tool.description}
            >
              {tool.name}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

function StatusDot({ status }: { status?: string }) {
  const color =
    status === 'connected'
      ? 'bg-green-500'
      : status === 'error'
        ? 'bg-red-500'
        : 'bg-gray-400';
  return <span className={`inline-block w-2 h-2 rounded-full ${color}`} />;
}

function ToggleItem({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-2 text-sm cursor-pointer">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="rounded border-input"
      />
      {label}
    </label>
  );
}

function AddServerDialog({
  onAdd,
  onClose,
}: {
  onAdd: (config: MCPServerConfig) => void;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [id, setId] = useState('');
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [type, setType] = useState<MCPTransportType>('sse');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || !url) return;
    onAdd({
      id,
      name: name || id,
      url,
      type,
      isActive: true,
    });
  };

  return (
    <Dialog open={true} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('settings.mcp.addServer')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="text-sm font-medium">ID</label>
            <input
              type="text"
              value={id}
              onChange={(e) => setId(e.target.value)}
              placeholder="my-mcp-server"
              className="w-full mt-1 px-3 py-2 border rounded-md bg-background text-sm"
              required
            />
          </div>
          <div>
            <label className="text-sm font-medium">{t('settings.mcp.serverName')}</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My MCP Server"
              className="w-full mt-1 px-3 py-2 border rounded-md bg-background text-sm"
            />
          </div>
          <div>
            <label className="text-sm font-medium">{t('settings.mcp.serverUrl')}</label>
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="http://localhost:3001/sse"
              className="w-full mt-1 px-3 py-2 border rounded-md bg-background text-sm"
              required
            />
          </div>
          <div>
            <label className="text-sm font-medium">{t('settings.mcp.transportType')}</label>
            <select
              value={type}
              onChange={(e) => setType(e.target.value as MCPTransportType)}
              className="w-full mt-1 px-3 py-2 border rounded-md bg-background text-sm"
            >
              <option value="sse">SSE</option>
              <option value="streamable-http">Streamable HTTP</option>
            </select>
          </div>
          <DialogFooter>
            <button
              type="button"
              onClick={onClose}
              className="px-3 py-1.5 text-sm border rounded-md hover:bg-muted"
            >
              {t('common.cancel')}
            </button>
            <button
              type="submit"
              className="px-3 py-1.5 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
            >
              {t('settings.mcp.addServer')}
            </button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
