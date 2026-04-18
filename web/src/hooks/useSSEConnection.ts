/**
 * Generic SSE connection management hook.
 *
 * Wraps EventSource creation, auto-reconnect (exponential backoff), and cleanup.
 * Shared by useSSE and useLogs — consumers only handle events.
 */

import { useEffect, useRef, useCallback, useState } from 'react';
import { apiClient } from '@/lib/api/client';

/** SSE connection state */
export type SSEConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error';

/** useSSEConnection options */
export interface UseSSEConnectionOptions {
  /** SSE endpoint path (relative to API base URL), e.g. '/events' or '/logs/stream' */
  url: string;
  /** URL query parameters */
  params?: Record<string, string>;
  /** Base reconnect interval (ms), default 1000 */
  reconnectInterval?: number;
  /** Max reconnect attempts, default 10 */
  maxReconnectAttempts?: number;
  /** Connection open callback. isReconnect is true for reconnections */
  onOpen?: (isReconnect: boolean) => void;
  /** Message received callback */
  onMessage?: (event: MessageEvent) => void;
  /** Error callback (triggered only when max reconnect attempts reached) */
  onError?: (error: Event) => void;
}

/** useSSEConnection return value */
export interface UseSSEConnectionReturn {
  /** Current connection state */
  connectionState: SSEConnectionState;
  /** Manual connect/reconnect (resets reconnect count) */
  connect: () => void;
  /** Disconnect */
  disconnect: () => void;
}

export function useSSEConnection(options: UseSSEConnectionOptions): UseSSEConnectionReturn {
  const {
    url,
    params,
    reconnectInterval = 1000,
    maxReconnectAttempts = 10,
    onOpen,
    onMessage,
    onError,
  } = options;

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [connectionState, setConnectionState] = useState<SSEConnectionState>('disconnected');

  // Use refs for callbacks to avoid reconnection on callback changes
  const onOpenRef = useRef(onOpen);
  const onMessageRef = useRef(onMessage);
  const onErrorRef = useRef(onError);
  onOpenRef.current = onOpen;
  onMessageRef.current = onMessage;
  onErrorRef.current = onError;

  /**
   * Clear reconnect timer
   */
  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  /**
   * Disconnect SSE
   */
  const disconnect = useCallback(() => {
    clearReconnectTimer();

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    setConnectionState('disconnected');
  }, [clearReconnectTimer]);

  /**
   * Connect to SSE endpoint
   */
  const connect = useCallback(() => {
    // Skip if already connected or connecting
    if (
      eventSourceRef.current?.readyState === EventSource.OPEN ||
      eventSourceRef.current?.readyState === EventSource.CONNECTING
    ) {
      return;
    }

    // Clean up existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    // Clear pending reconnect
    clearReconnectTimer();

    try {
      const baseUrl = apiClient.getBaseUrl();
      let sseUrl = `${baseUrl}${url}`;

      if (params) {
        const searchParams = new URLSearchParams(params);
        sseUrl += `?${searchParams.toString()}`;
      }

      const es = new EventSource(sseUrl);

      es.addEventListener('open', () => {
        const wasReconnect = reconnectAttemptsRef.current > 0;
        setConnectionState('connected');
        reconnectAttemptsRef.current = 0;
        onOpenRef.current?.(wasReconnect);
      });

      es.addEventListener('message', (event: MessageEvent) => {
        onMessageRef.current?.(event);
      });

      es.addEventListener('error', (error: Event) => {
        const target = error.target as EventSource;

        // Check if cleanly closed
        if (target.readyState === EventSource.CLOSED) {
          setConnectionState('disconnected');
          return;
        }

        setConnectionState('error');

        if (reconnectAttemptsRef.current < maxReconnectAttempts) {
          const delay = Math.min(
            reconnectInterval * Math.pow(2, reconnectAttemptsRef.current),
            30000
          );

          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttemptsRef.current++;
            connect();
          }, delay);
        } else {
          console.error('SSE max reconnection attempts reached');
          es.close();
          eventSourceRef.current = null;
          onErrorRef.current?.(error);
        }
      });

      eventSourceRef.current = es;
      setConnectionState('connecting');
    } catch (error) {
      console.error('Failed to create EventSource:', error);
      setConnectionState('error');
    }
  }, [url, params, reconnectInterval, maxReconnectAttempts, clearReconnectTimer]);

  /**
   * Manual reconnect: reset counter then connect
   */
  const manualReconnect = useCallback(() => {
    reconnectAttemptsRef.current = 0;
    disconnect();
    connect();
  }, [connect, disconnect]);

  // Connect on mount, disconnect on unmount
  useEffect(() => {
    connect();
    return () => disconnect();
  }, [connect, disconnect]);

  return {
    connectionState,
    connect: manualReconnect,
    disconnect,
  };
}
