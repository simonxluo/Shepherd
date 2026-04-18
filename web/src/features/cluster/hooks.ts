import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
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
 * Server config response type
 */
interface ServerConfigResponse {
  role: 'master' | 'client' | 'hybrid';
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
 * Server config hook
 */
export function useServerConfig() {
  return useQuery({
    queryKey: ['server', 'config'],
    queryFn: async (): Promise<ServerConfigResponse> => {
      const response = await apiClient.get<{ success: boolean; data: ServerConfigResponse }>('/config');
      return response.data;
    },
    staleTime: 5 * 60 * 1000,
    refetchInterval: false,
  });
}

/**
 * Get current cluster mode. Cluster features only available in master mode.
 */
function useClusterMode(): 'master' | 'client' | 'hybrid' {
  const { data: serverConfig } = useServerConfig();

  return serverConfig?.role || 'hybrid';
}

/**
 * Cluster overview hook
 */
export function useClusterOverview() {
  const mode = useClusterMode();

  return useQuery({
    queryKey: ['cluster', 'overview'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: ClusterOverview }>('/master/overview');
      return response.data;
    },
    staleTime: 10 * 1000,
    refetchInterval: 5000,
    enabled: mode === 'master',
  });
}

/**
 * Client list hook
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
    enabled: mode === 'master',
  });
}

/**
 * Single client hook
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
 * Cluster task list hook
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
    enabled: mode === 'master',
  });
}

/**
 * Network scan hook
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
 * Filter clients hook
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

    if (filters.status && client.status !== filters.status) return false;

    if (filters.hasTag && !client.tags.includes(filters.hasTag)) return false;

    return true;
  });
}

/**
 * Filter tasks hook
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

    if (filters.status && task.status !== filters.status) return false;

    if (filters.type && task.type !== filters.type) return false;

    if (filters.assignedTo && task.assignedTo !== filters.assignedTo) return false;

    return true;
  });
}


/**
 * Online nodes hook for node selection
 */
export function useOnlineNodes() {
  const mode = useClusterMode();

  return useQuery<UnifiedNode[]>({
    queryKey: ['cluster', 'nodes', 'online'],
    queryFn: async () => {
      const response = await apiClient.get<{
        success: boolean;
        data: {
          nodes: UnifiedNode[];
        };
      }>('/nodes');
      return response.data.nodes.filter(
        (node) => node.status === ('online' as NodeStatus)
      );
    },
    staleTime: 10 * 1000,
    refetchInterval: 5000,
    enabled: mode === 'master',
  });
}



/**
 * Node config hook
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
    staleTime: 60 * 1000,
  });
}

/**
 * Test node llama.cpp hook
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
      queryClient.invalidateQueries({
        queryKey: ['cluster', 'nodes', nodeId, 'config'],
      });
    },
  });
}