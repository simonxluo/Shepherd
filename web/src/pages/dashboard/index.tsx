import { useMemo } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useModels } from '@/features/models';
import { useDownloads, useDownloadStats } from '@/features/downloads/hooks';
import { useClients, useFilteredClients } from '@/features/cluster/hooks';
import { formatBytes } from '@/lib/utils';
import { Package, Download, Network, Activity } from 'lucide-react';
import type { Model } from '@/types';

/**
 * Dashboard page
 */
export function DashboardPage() {
  const { data: models = [], isLoading } = useModels();
  const { data: downloads = [], isLoading: downloadsLoading } = useDownloads();
  const downloadStats = useDownloadStats(downloads);
  const { data: clients = [], isLoading: clientsLoading } = useClients();
  const onlineClients = useFilteredClients(clients, { status: 'online' });

  // Sort by scan time, get 5 most recent models
  const recentModels = useMemo(() => {
    return [...models]
      .sort((a: Model, b: Model) => {
        // Sort by scan time descending (newest first)
        const aTime = new Date(a.scannedAt).getTime();
        const bTime = new Date(b.scannedAt).getTime();
        return bTime - aTime;
      })
      .slice(0, 5);
  }, [models]);

  const stats = [
    {
      title: '总模型数',
      value: models?.length || 0,
      icon: Package,
      description: '已扫描的模型',
    },
    {
      title: '已加载',
      value: models?.filter((m) => m.isLoaded).length || 0,
      icon: Activity,
      description: '正在运行的模型',
    },
    {
      title: '下载任务',
      value: downloadStats.active,
      icon: Download,
      description: '活跃的下载任务',
    },
    {
      title: '集群节点',
      value: onlineClients.length,
      icon: Network,
      description: '在线的客户端',
    },
  ];

  if (isLoading || downloadsLoading || clientsLoading) {
    return <div className="flex items-center justify-center h-full">加载中...</div>;
  }

  return (
    <div className="space-y-6">
      {/* Page title */}
      <div>
        <h1 className="text-3xl font-bold text-foreground">仪表盘</h1>
        <p className="text-muted-foreground font-medium">Shepherd 模型管理系统概览</p>
      </div>

      {/* Statistics cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <Card key={stat.title}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{stat.title}</CardTitle>
                <Icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stat.value}</div>
                <p className="text-xs text-muted-foreground">{stat.description}</p>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* Recent models */}
      <Card>
        <CardHeader>
          <CardTitle>最近模型</CardTitle>
          <CardDescription>最近扫描的模型列表</CardDescription>
        </CardHeader>
        <CardContent>
          {recentModels.length > 0 ? (
            <div className="space-y-4">
              {recentModels.map((model) => (
                <div key={model.id} className="flex items-center justify-between">
                  <div>
                    <div className="font-medium">{model.alias || model.name}</div>
                    <div className="text-sm text-muted-foreground">
                      {model.metadata.architecture} • {formatBytes(model.totalSize ?? model.size)}
                      {model.shardCount && model.shardCount > 1 && (
                        <span className="ml-1 text-xs text-muted-foreground">
                          ({model.shardCount} 分卷)
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {new Date(model.scannedAt).toLocaleDateString()}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center text-muted-foreground py-8">
              暂无模型数据
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
