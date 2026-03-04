import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api/client';
import type {
  DistributedNode,
  NodeListResponse,
  NodeStats,
  NodeRegisterRequest,
  NodeRegisterResponse,
  SendCommandRequest,
  SendCommandResponse,
} from '@/types/node';

export function useNodes() {
  return useQuery({
    queryKey: ['nodes'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: NodeListResponse }>('/master/nodes');
      return response.data.nodes;
    },
    refetchInterval: 5000,
  });
}

export function useNode(nodeId: string) {
  return useQuery({
    queryKey: ['nodes', nodeId],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: { node: DistributedNode } }>(`/master/nodes/${nodeId}`);
      return response.data.node;
    },
    enabled: !!nodeId,
  });
}

export function useNodeStats() {
  return useQuery({
    queryKey: ['nodes', 'stats'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: NodeListResponse }>('/master/nodes');
      return response.data.stats;
    },
    refetchInterval: 5000,
  });
}

export function useNodesWithStats() {
  return useQuery({
    queryKey: ['nodes'],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: NodeListResponse }>('/master/nodes');
      return response.data;
    },
    refetchInterval: 5000,
  });
}

export function useRegisterNode() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (nodeInfo: NodeRegisterRequest) => {
      const response = await apiClient.post<{
        success: boolean;
        data: NodeRegisterResponse;
      }>('/master/nodes/register', nodeInfo);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
  });
}

export function useUnregisterNode() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (nodeId: string) => {
      const response = await apiClient.delete<{
        success: boolean;
        data?: { message: string };
      }>(`/master/nodes/${nodeId}`);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
  });
}

export function useSendCommand() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ nodeId, command }: { nodeId: string; command: SendCommandRequest }) => {
      const response = await apiClient.post<{
        success: boolean;
        data: SendCommandResponse;
      }>(`/master/nodes/${nodeId}/command`, command);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    },
  });
}

export function useNodesByRole(role: string) {
  return useQuery<DistributedNode[]>({
    queryKey: ['nodes', 'role', role],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: NodeListResponse }>('/master/nodes');
      return response.data.nodes.filter((node) => node.role === role);
    },
    enabled: !!role,
  });
}

export function useNodesByStatus(status: string) {
  return useQuery<DistributedNode[]>({
    queryKey: ['nodes', 'status', status],
    queryFn: async () => {
      const response = await apiClient.get<{ success: boolean; data: NodeListResponse }>('/master/nodes');
      return response.data.nodes.filter((node) => node.status === status);
    },
    enabled: !!status,
  });
}

export function useFilteredNodes(
  nodes: DistributedNode[] | undefined,
  filters: {
    search?: string;
    status?: string;
    role?: string;
    hasTag?: string;
  }
) {
  if (!nodes) return [];

  return nodes.filter((node) => {
    if (filters.search) {
      const search = filters.search.toLowerCase();
      const matchName = node.name ? node.name.toLowerCase().includes(search) : false;
      const matchAddress = node.address ? node.address.toLowerCase().includes(search) : false;
      if (!matchName && !matchAddress) return false;
    }

    if (filters.status && node.status !== filters.status) return false;

    if (filters.role && node.role !== filters.role) return false;

    if (filters.hasTag && !node.tags.includes(filters.hasTag)) return false;

    return true;
  });
}
