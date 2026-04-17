import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useSSEConnection } from './useSSEConnection';
import type { SSEEvent } from '@/types';
import type { UnifiedNode } from '@/types/node';

interface UseSSEOptions {
  onMessage?: (event: SSEEvent) => void;
  onError?: (error: Event) => void;
  onOpen?: () => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

export function useSSE(options: UseSSEOptions = {}) {
  const {
    onMessage,
    onError,
    onOpen,
    reconnectInterval = 1000,
    maxReconnectAttempts = 10,
  } = options;

  const queryClient = useQueryClient();

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const data: SSEEvent = JSON.parse(event.data);

        switch (data.type) {
          case 'modelLoad':
          case 'modelLoadStart':
          case 'modelStop':
            queryClient.invalidateQueries({ queryKey: ['models'] });
            break;
          case 'download_progress':
          case 'download_status':
            queryClient.invalidateQueries({ queryKey: ['downloads'] });
            break;
          case 'clientRegistered':
          case 'clientDisconnected':
            queryClient.invalidateQueries({ queryKey: ['clients'] });
            queryClient.invalidateQueries({ queryKey: ['cluster'] });
            break;
          case 'clientResourcesUpdated': {
            const clientData = data.data as { clientId: string; node: UnifiedNode };
            const { clientId, node } = clientData;

            queryClient.setQueryData(['cluster', 'clients', clientId], node);

            queryClient.setQueryData(['cluster', 'clients'], (old: unknown) => {
              if (Array.isArray(old)) {
                return old.map((client: Record<string, unknown>) =>
                  client.id === clientId
                    ? { ...node, lastSeen: new Date().toISOString() }
                    : client
                );
              }
              return old;
            });
            break;
          }
          case 'taskUpdate':
            queryClient.invalidateQueries({ queryKey: ['tasks'] });
            queryClient.invalidateQueries({ queryKey: ['cluster'] });
            break;
          case 'systemStatus':
            queryClient.invalidateQueries({ queryKey: ['system'] });
            break;
        }

        onMessage?.(data);
      } catch (error) {
        console.error('Failed to parse SSE event:', error);
      }
    },
    [onMessage, queryClient]
  );

  const handleOpen = useCallback(
    (isReconnect: boolean) => {
      if (isReconnect) {
        queryClient.invalidateQueries({ queryKey: ['models'] });
        queryClient.invalidateQueries({ queryKey: ['downloads'] });
        queryClient.invalidateQueries({ queryKey: ['clients'] });
        queryClient.invalidateQueries({ queryKey: ['cluster'] });
        queryClient.invalidateQueries({ queryKey: ['tasks'] });
        queryClient.invalidateQueries({ queryKey: ['system'] });
        queryClient.invalidateQueries({ queryKey: ['nodes'] });
      }
      onOpen?.();
    },
    [onOpen, queryClient]
  );

  const { connectionState } = useSSEConnection({
    url: '/events',
    onMessage: handleMessage,
    onOpen: handleOpen,
    onError,
    reconnectInterval,
    maxReconnectAttempts,
  });

  return { connectionState };
}
