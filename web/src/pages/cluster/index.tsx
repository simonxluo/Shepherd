import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Search, Radar, Server, CheckCircle2, XCircle, Clock, AlertCircle, AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  useClients,
  useClusterTasks,
  useNetworkScan,
  useClusterOverview,
  useFilteredClients,
  useServerConfig,
} from '@/features/cluster/hooks';
import { ClientCard } from '@/features/cluster/components/ClientCard';
import { cn } from '@/lib/utils';
import type { ClientStatus, ClusterTask } from '@/types';
import { useAlertDialog } from '@/providers/AlertDialog';

/**
 * Cluster management page
 */
export function ClusterPage() {
  // All hooks must be called before conditional returns
  const { t } = useTranslation();
  const alertDialog = useAlertDialog();
  const { data: serverConfig } = useServerConfig();
  const { data: clients, isLoading: clientsLoading } = useClients();
  const { data: tasks, isLoading: tasksLoading } = useClusterTasks() as { data: ClusterTask[] | undefined, isLoading: boolean };
  const { data: overview } = useClusterOverview();
  const networkScan = useNetworkScan();

  // UI state
  const [activeTab, setActiveTab] = useState<'clients' | 'tasks'>('clients');
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<ClientStatus | ''>('');

  const filteredClients = useFilteredClients(clients, {
    search,
    status: statusFilter || undefined,
  });

  // Show notice when not in Master/Hybrid mode
  if (serverConfig && serverConfig.role !== 'master' && serverConfig.role !== 'hybrid') {
    return (
      <div className="flex flex-col items-center justify-center h-[calc(100vh-8rem)]">
        <AlertCircle className="w-16 h-16 text-yellow-500 mb-4" />
        <h2 className="text-2xl font-bold text-foreground mb-2">{t('cluster.unavailable')}</h2>
        <p className="text-muted-foreground text-center max-w-md">
          {t('cluster.unavailableDesc')}
        </p>
        <p className="text-sm text-muted-foreground mt-4">
          {t('cluster.unavailableHint')}
        </p>
      </div>
    );
  }

  const handleScan = async () => {
    const confirmed = await alertDialog.confirm({
      title: t('cluster.scanNetwork'),
      description: t('cluster.scanNetworkConfirm'),
    });
    if (confirmed) {
      networkScan.mutate({});
    }
  };

  return (
    <div className="space-y-6">
      {/* Title and actions */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">{t('cluster.title')}</h1>
          <p className="text-muted-foreground">
            {t('cluster.summary', { total: overview?.totalClients || 0, online: overview?.onlineClients || 0 })}
          </p>
        </div>

        <Button
          onClick={handleScan}
          disabled={networkScan.isPending}
          variant="default"
          size="sm"
        >
          <Radar className={cn('w-4 h-4', networkScan.isPending && 'animate-spin')} />
          {t('cluster.scanNetwork')}
        </Button>
      </div>

      {/* Statistics cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="p-4 bg-card rounded-lg border border-border">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
              <Server className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <div className="text-2xl font-bold text-foreground">
                {overview?.totalClients || 0}
              </div>
              <div className="text-sm text-muted-foreground">{t('cluster.stats.totalClients')}</div>
            </div>
          </div>
        </div>

        <div className="p-4 bg-card rounded-lg border border-border">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
              <CheckCircle2 className="w-5 h-5 text-green-600 dark:text-green-400" />
            </div>
            <div>
              <div className="text-2xl font-bold text-foreground">
                {overview?.onlineClients || 0}
              </div>
              <div className="text-sm text-muted-foreground">{t('cluster.stats.online')}</div>
            </div>
          </div>
        </div>

        <div className="p-4 bg-card rounded-lg border border-border">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-yellow-100 dark:bg-yellow-900/30 rounded-lg">
              <Clock className="w-5 h-5 text-yellow-600 dark:text-yellow-400" />
            </div>
            <div>
              <div className="text-2xl font-bold text-foreground">
                {overview?.busyClients || 0}
              </div>
              <div className="text-sm text-muted-foreground">{t('cluster.stats.busy')}</div>
            </div>
          </div>
        </div>

        <div className="p-4 bg-card rounded-lg border border-border">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
              <XCircle className="w-5 h-5 text-purple-600 dark:text-purple-400" />
            </div>
            <div>
              <div className="text-2xl font-bold text-foreground">
                {overview?.runningTasks || 0}
              </div>
              <div className="text-sm text-muted-foreground">{t('cluster.stats.runningTasks')}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-border">
        <nav className="flex gap-6">
          <Button
            onClick={() => setActiveTab('clients')}
            variant={activeTab === 'clients' ? 'default' : 'ghost'}
            size="sm"
            className={cn(
              'rounded-t-lg rounded-b-none border-b-2',
              activeTab === 'clients'
                ? 'border-primary'
                : 'border-transparent hover:border-transparent'
            )}
          >
            {t('cluster.tabs.clients')} ({clients?.length || 0})
          </Button>
          <Button
            onClick={() => setActiveTab('tasks')}
            variant={activeTab === 'tasks' ? 'default' : 'ghost'}
            size="sm"
            className={cn(
              'rounded-t-lg rounded-b-none border-b-2',
              activeTab === 'tasks'
                ? 'border-primary'
                : 'border-transparent hover:border-transparent'
            )}
          >
            {t('cluster.tabs.tasks')} ({tasks?.length || 0})
          </Button>
        </nav>
      </div>

      {/* Search and filter */}
      {activeTab === 'clients' && (
        <div className="flex items-center gap-3 p-4 bg-card rounded-lg border border-border">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('cluster.searchPlaceholder')}
              className="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-input text-foreground placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <Select
            value={statusFilter}
            onValueChange={(v) => setStatusFilter(v as ClientStatus | '')}
          >
            <SelectTrigger className="px-3 py-2 border border-border rounded-lg bg-input text-foreground w-[140px]">
              <SelectValue placeholder={t('cluster.filter.allStatuses')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="online">{t('cluster.status.online')}</SelectItem>
              <SelectItem value="busy">{t('cluster.status.busy')}</SelectItem>
              <SelectItem value="offline">{t('cluster.status.offline')}</SelectItem>
              <SelectItem value="error">{t('cluster.status.error')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      )}

      {/* Client list */}
      {activeTab === 'clients' && (
        <>
          {clientsLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
            </div>
          ) : filteredClients.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <Server className="w-12 h-12 mb-4" />
              <p className="text-lg mb-2">{t('cluster.noClients')}</p>
              <p className="text-sm">{t('cluster.noClientsHint')}</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {filteredClients.map((client) => (
                <ClientCard key={client.id} client={client} />
              ))}
            </div>
          )}
        </>
      )}

      {/* Task list */}
      {activeTab === 'tasks' && (
        <>
          {tasksLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
            </div>
          ) : !tasks || tasks.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <Clock className="w-12 h-12 mb-4" />
              <p className="text-lg">{t('cluster.noTasks')}</p>
            </div>
          ) : (
            <div className="space-y-3">
              {tasks.map((task) => (
                <div
                  key={task.id}
                  className="p-4 bg-card rounded-lg border border-border"
                >
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <h3 className="font-medium text-foreground">{task.type}</h3>
                      <p className="text-sm text-muted-foreground">ID: {task.id}</p>
                    </div>
                    <span
                      className={cn(
                        'px-2 py-1 rounded text-xs font-medium',
                        task.status === 'completed'
                          ? 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300'
                          : task.status === 'running'
                            ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                            : task.status === 'failed'
                              ? 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
                              : 'bg-gray-100 text-gray-700 dark:bg-gray-700/50 dark:text-gray-300'
                      )}
                    >
                      {task.status}
                    </span>
                  </div>

                  {task.assignedTo && (
                    <div className="text-sm text-muted-foreground">
                      {t('cluster.assignedTo')}: {task.assignedTo}
                    </div>
                  )}

                  {task.error && (
                    <Alert variant="destructive" className="mt-2">
                      <AlertTriangle className="h-4 w-4" />
                      <AlertDescription className="text-sm">{task.error}</AlertDescription>
                    </Alert>
                  )}
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}
