import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import { useConfig } from '@/lib/config';
import type {
  Client,
  ClusterTask,
  ClientListResponse,
  TaskListResponse,
  ClusterOverview,
  ScanStatus,
  ClientStatus,
  ScheduleStrategy,
  UnifiedNode,
  NodeStatus,
  NodeConfigInfo,
  LlamacppTestResult,
} from '@/types';

/**
 * 服务器配置响应类型
 */
interface ServerConfigResponse {
  role: 'master' | 'client' | 'hybrid';  // 使用 role 代替 mode
  server: {
    host: string;
    web_port: number;
    anthropic_port: number;
    ollama_port: number;
    lm_studio_port: number;
  };
  storage: {
    type: string;
    sqlite: Record<string, unknown>;
  };
  models: {
    paths: string[];
    auto_scan: boolean;
  };
  node: {
    role: string;
    id: string;
    name: string;
  };
  llamacpp: {
    paths: Array<{ name: string; path: string; description: string }>;
  };
}

/**
 * 获取服务器配置 Hook
 */
export function useServerConfig() {
  return useQuery({
    queryKey: ['server', 'config'],
    queryFn: async (): Promise<ServerConfigResponse> => {
      const response = await apiClient.get<{ success: boolean; data: ServerConfigResponse }>('/config');
      return response.data;
    },
    staleTime: 5 * 60 * 1000, // 5 分钟
    refetchInterval: false,
  });
}

/**
 * 获取当前运行模式
 * 集群相关功能仅在 Master 模式下可用
 */
function useClusterMode(): 'master' | 'client' | 'hybrid' {
  const { data: serverConfig } = useServerConfig();

  // 从后端 API 获取节点角色，如果未获取到则默认为 hybrid
  return serverConfig?.role || 'hybrid';
}

/**
 * 集群概览 Hook
 */
export function useClusterOverview() {
  const mode = useClusterMode();

  return useQuery({
    queryKey: ['cluster', 'overview'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: ClusterOverview }>('/master/overview');
      return response.data;
    },
    staleTime: 10 * 1000, // 10 秒
    refetchInterval: 5000, // 每 5 秒刷新
    enabled: mode === 'master', // 仅在 Master 模式下启用
  });
}

/**
 * 客户端列表 Hook
 */
export function useClients() {
  const mode = useClusterMode();

  return useQuery({
    queryKey: ['cluster', 'clients'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: ClientListResponse }>('/master/clients');
      return response.data.clients;
    },
    staleTime: 10 * 1000,
    refetchInterval: 5000,
    enabled: mode === 'master', // 仅在 Master 模式下启用
  });
}

/**
 * 单个客户端 Hook
 */
export function useClient(clientId: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['cluster', 'clients', clientId],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: { client: Client } }>(`/master/clients/${clientId}`);
      return response.data.client;
    },
    enabled: !!clientId && options?.enabled !== false,
    refetchInterval: 3000,
  });
}

/**
 * 断开客户端 Hook
 */
export function useDisconnectClient() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (clientId: string) => {
      const response = await apiClient.delete<{ success: boolean }>(`/master/clients/${clientId}`);
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cluster'] });
    },
  });
}

/**
 * 任务列表 Hook
 */
export function useClusterTasks() {
  const mode = useClusterMode();

  return useQuery({
    queryKey: ['cluster', 'tasks'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: TaskListResponse }>('/master/tasks');
      return response.data.tasks;
    },
    staleTime: 5 * 1000,
    refetchInterval: 2000,
    enabled: mode === 'master', // 仅在 Master 模式下启用
  });
}

/**
 * 创建任务 Hook
 */
export function useCreateClusterTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (task: {
      type: string;
      payload: Record<string, unknown>;
      assignTo?: string;
    }) => {
      const response = await apiClient.post<{ success: boolean; data: { task: ClusterTask } }>('/master/tasks', task);
      return response.data.task;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cluster', 'tasks'] });
    },
  });
}

/**
 * 取消任务 Hook
 */
export function useCancelClusterTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (taskId: string) => {
      const response = await apiClient.delete<{ success: boolean }>(`/master/tasks/${taskId}`);
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cluster', 'tasks'] });
    },
  });
}

/**
 * 重试任务 Hook
 */
export function useRetryClusterTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (taskId: string) => {
      const response = await apiClient.post<{ success: boolean }>(`/master/tasks/${taskId}/retry`);
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cluster', 'tasks'] });
    },
  });
}

/**
 * 扫描网络 Hook
 */
export function useNetworkScan() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params?: {
      cidr?: string;
      portRange?: string;
      timeout?: number;
    }) => {
      const response = await apiClient.post<{ success: boolean }>('/master/scan', params || {});
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cluster', 'scan'] });
    },
  });
}

/**
 * 扫描状态 Hook
 */
export function useScanStatus() {
  return useQuery({
    queryKey: ['cluster', 'scan'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: ScanStatus }>('/master/scan/status');
      return response.data;
    },
    refetchInterval: (query) => {
      const data = query.state.data;
      return data?.running ? 1000 : false;
    },
  });
}

/**
 * 设置调度策略 Hook
 */
export function useSetScheduleStrategy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (strategy: ScheduleStrategy) => {
      const response = await apiClient.put<{ success: boolean }>('/master/schedule/strategy', {
        strategy,
      });
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cluster'] });
    },
  });
}

/**
 * 过滤客户端 Hook
 */
export function useFilteredClients(
  clients: Client[] | undefined,
  filters: {
    search?: string;
    status?: ClientStatus;
    hasTag?: string;
  }
) {
  if (!clients) return [];

  return clients.filter((client) => {
    // 搜索过滤
    if (filters.search) {
      const search = filters.search.toLowerCase();
      const matchName = client.name ? client.name.toLowerCase().includes(search) : false;
      const matchAddress = client.address ? client.address.toLowerCase().includes(search) : false;
      if (!matchName && !matchAddress) return false;
    }

    // 状态过滤
    if (filters.status && client.status !== filters.status) return false;

    // 标签过滤
    if (filters.hasTag && !client.tags.includes(filters.hasTag)) return false;

    return true;
  });
}

/**
 * 过滤任务 Hook
 */
export function useFilteredTasks(
  tasks: ClusterTask[] | undefined,
  filters: {
    search?: string;
    status?: string;
    type?: string;
    assignedTo?: string;
  }
) {
  if (!tasks) return [];

  return tasks.filter((task) => {
    // 搜索过滤
    if (filters.search) {
      const search = filters.search.toLowerCase();
      const matchId = task.id.toLowerCase().includes(search);
      if (!matchId) return false;
    }

    // 状态过滤
    if (filters.status && task.status !== filters.status) return false;

    // 类型过滤
    if (filters.type && task.type !== filters.type) return false;

    // 分配过滤
    if (filters.assignedTo && task.assignedTo !== filters.assignedTo) return false;

    return true;
  });
}


/**
 * 获取在线节点列表 Hook（用于节点选择）
 * 返回所有在线状态的节点，包含资源信息
 */
export function useOnlineNodes() {
  const mode = useClusterMode();

  return useQuery<UnifiedNode[]>({
    queryKey: ['cluster', 'nodes', 'online'],
    queryFn: async () => {
      // 使用 /nodes 端点获取所有节点
      const response = await apiClient.get<{
        success: boolean;
        data: {
          nodes: UnifiedNode[];
        };
      }>('/nodes');
      // 只返回在线节点
      return response.data.nodes.filter(
        (node) => node.status === ('online' as NodeStatus)
      );
    },
    staleTime: 10 * 1000, // 10 秒缓存
    refetchInterval: 5000, // 每 5 秒刷新
    enabled: mode === 'master', // 仅在 Master 模式下启用
  });
}



/**
 * 获取节点配置信息 Hook
 */
export function useNodeConfig(nodeId: string, options?: { enabled?: boolean }) {
  return useQuery<NodeConfigInfo>({
    queryKey: ['cluster', 'nodes', nodeId, 'config'],
    queryFn: async () => {
      const response = await apiClient.get<{
        success: boolean;
        data: NodeConfigInfo;
      }>(`/nodes/${nodeId}/config`);
      return response.data;
    },
    enabled: !!nodeId && options?.enabled !== false,
    staleTime: 60 * 1000, // 1 分钟缓存
  });
}

/**
 * 测试节点 llama.cpp Hook
 */
export function useTestNodeLlamacpp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (nodeId: string) => {
      const response = await apiClient.post<{
        success: boolean;
        data: LlamacppTestResult;
      }>(`/nodes/${nodeId}/test`);
      return response.data;
    },
    onSuccess: (_, nodeId) => {
      // 使节点配置缓存失效，以便获取最新状态
      queryClient.invalidateQueries({
        queryKey: ['cluster', 'nodes', nodeId, 'config'],
      });
    },
  });
}