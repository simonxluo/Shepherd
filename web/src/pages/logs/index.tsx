import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Server, Monitor, Wifi, WifiOff } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { Button } from '@/components/ui/button';
import { LogPanel } from '@/features/logs/components/LogPanel';
import { useLogStream } from '@/features/logs/hooks';
import type { UnifiedNode } from '@/types/node';

type Tab = 'local' | 'nodes';

export function LogsPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useState<Tab>('local');
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const { data: serverConfig } = useQuery({
    queryKey: ['server', 'config'],
    queryFn: async () => {
      const res = await apiClient.get<{ success: boolean; data: { role: string } }>('/config');
      return res.data;
    },
    staleTime: 5 * 60 * 1000,
  });

  const isMaster = serverConfig?.role === 'master';

  const { data: nodes = [] } = useQuery<UnifiedNode[]>({
    queryKey: ['cluster', 'nodes'],
    queryFn: async () => {
      const res = await apiClient.get<{ success: boolean; data: { nodes: UnifiedNode[] } }>('/nodes');
      return res.data.nodes.filter(n => n.status === 'online');
    },
    enabled: isMaster,
    refetchInterval: 10000,
  });

  const localStream = useLogStream();
  const selectedNode = nodes.find(n => n.id === selectedNodeId);
  const nodeUrl = selectedNode ? `http://${selectedNode.address}:${selectedNode.port}/api` : undefined;
  const nodeStream = useLogStream(nodeUrl);

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      <div className="flex items-center justify-between px-4 py-3 border-b">
        <h1 className="text-2xl font-bold text-foreground">{t('logs.title', '日志')}</h1>
        <div className="flex items-center gap-2">
          <Button
            variant={tab === 'local' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setTab('local')}
          >
            <Monitor className="w-4 h-4 mr-1" />
            {t('logs.local', '本机')}
          </Button>
          {isMaster && (
            <Button
              variant={tab === 'nodes' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setTab('nodes')}
            >
              <Server className="w-4 h-4 mr-1" />
              {t('logs.nodes', '节点')}
            </Button>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-hidden flex">
        {tab === 'nodes' && isMaster && (
          <div className="w-56 border-r flex flex-col bg-muted/20">
            <div className="px-3 py-2 text-xs text-muted-foreground border-b">
              {t('logs.nodeList', '节点列表')} ({nodes.length})
            </div>
            <div className="flex-1 overflow-y-auto">
              {nodes.length === 0 ? (
                <div className="px-3 py-4 text-sm text-muted-foreground text-center">
                  {t('logs.noNodes', '无在线节点')}
                </div>
              ) : (
                nodes.map(node => (
                  <button
                    key={node.id}
                    onClick={() => setSelectedNodeId(node.id)}
                    className={`w-full text-left px-3 py-2.5 text-sm border-b transition-colors flex items-center gap-2 ${
                      selectedNodeId === node.id
                        ? 'bg-accent text-accent-foreground'
                        : 'hover:bg-accent/50'
                    }`}
                  >
                    <span className="flex-shrink-0">
                      {node.status === 'online' ? (
                        <Wifi className="w-3.5 h-3.5 text-green-500" />
                      ) : (
                        <WifiOff className="w-3.5 h-3.5 text-muted-foreground" />
                      )}
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-medium">{node.name || node.id}</div>
                      <div className="text-xs text-muted-foreground truncate">
                        {node.address}:{node.port}
                      </div>
                    </div>
                  </button>
                ))
              )}
            </div>
          </div>
        )}

        <div className="flex-1 overflow-hidden p-2">
          {tab === 'local' ? (
            <LogPanel
              id="local"
              title={t('logs.localTitle', '本机日志')}
              logData={localStream.logs}
              onClear={localStream.clear}
            />
          ) : selectedNodeId ? (
            <LogPanel
              id={`node-${selectedNodeId}`}
              title={`${t('logs.nodeLogs', '节点日志')} - ${selectedNode?.name || selectedNodeId}`}
              logData={nodeStream.logs}
              onClear={nodeStream.clear}
            />
          ) : (
            <div className="flex items-center justify-center h-full text-muted-foreground">
              <div className="text-center">
                <Server className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>{t('logs.selectNode', '请选择一个节点查看日志')}</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
