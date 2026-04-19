import {
  createContext,
  useCallback,
  useRef,
  useState,
  useEffect,
  useMemo,
} from 'react';
import type {
  WebSocketContextValue,
  WebSocketProviderProps,
  WebSocketMessage,
  WebSocketConnectionStatus,
  WebSocketClientOptions,
} from '@/types/websocket';
import { WebSocketClient } from '@/lib/websocket';
import { apiClient } from '@/lib/api/client';

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

type EventHandler<T = unknown> = (data: T) => void;
type EventSubscriptions = Map<string, Set<EventHandler<unknown>>>;

export function WebSocketProvider({
  children,
  url: propUrl,
  autoConnect = true,
  options = {},
  onConnect,
  onDisconnect,
  onError,
}: WebSocketProviderProps): React.ReactElement {
  const [connectionStatus, setConnectionStatus] = useState<WebSocketConnectionStatus>('disconnected');
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);

  const clientRef = useRef<WebSocketClient | null>(null);
  const subscriptionsRef = useRef<EventSubscriptions>(new Map());
  const mountedRef = useRef(true);

  const optionsRef = useRef(options);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const onErrorRef = useRef(onError);
  useEffect(() => {
    optionsRef.current = options;
    onConnectRef.current = onConnect;
    onDisconnectRef.current = onDisconnect;
    onErrorRef.current = onError;
  });

  const getWebSocketUrl = useCallback(() => {
    if (propUrl) return propUrl;

    const baseUrl = apiClient.getBaseUrl();
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return baseUrl.replace(/^https?:/, protocol).replace(/\/api$/, '/ws');
  }, [propUrl]);

  const createClient = useCallback(() => {
    const wsUrl = getWebSocketUrl();
    const opts = optionsRef.current;

    const clientOptions: WebSocketClientOptions = {
      url: wsUrl,
      maxReconnectAttempts: opts.maxReconnectAttempts ?? 5,
      initialReconnectDelay: opts.initialReconnectDelay ?? 1000,
      maxReconnectDelay: opts.maxReconnectDelay ?? 30000,
      heartbeatInterval: opts.heartbeatInterval ?? 30000,
      heartbeatTimeout: opts.heartbeatTimeout ?? 10000,
      autoReconnect: opts.autoReconnect ?? true,
      debug: opts.debug ?? false,
    };

    const client = new WebSocketClient(clientOptions);

    client.setHandlers({
      onStatusChange: (status) => {
        if (mountedRef.current) {
          setConnectionStatus(status);
        }
      },
      onMessage: (message) => {
        if (!mountedRef.current) return;

        setLastMessage(message);

        if (message.type === 'event' && 'eventType' in (message.payload as Record<string, unknown>)) {
          const payload = message.payload as { eventType: string; data: unknown };
          const handlers = subscriptionsRef.current.get(payload.eventType);
          if (handlers) {
            handlers.forEach((handler) => handler(payload.data));
          }
        }

        const genericHandlers = subscriptionsRef.current.get('*');
        if (genericHandlers) {
          genericHandlers.forEach((handler) => handler(message));
        }
      },
      onReconnecting: (attempt) => {
        if (mountedRef.current) {
          setReconnectAttempts(attempt);
        }
      },
      onReconnectFailed: () => {
        if (mountedRef.current) {
          setReconnectAttempts(clientOptions.maxReconnectAttempts ?? 5);
          onErrorRef.current?.(new Error('WebSocket reconnection failed'));
        }
      },
      onOpen: () => {
        if (mountedRef.current) {
          onConnectRef.current?.();
        }
      },
      onClose: () => {
        if (mountedRef.current) {
          onDisconnectRef.current?.();
        }
      },
    });

    return client;
  }, [getWebSocketUrl]);

  const connect = useCallback(() => {
    if (!clientRef.current) {
      clientRef.current = createClient();
    }
    clientRef.current.connect();
  }, [createClient]);

  const disconnect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect();
    }
  }, []);

  const sendMessage = useCallback(<T = unknown,>(message: WebSocketMessage<T>) => {
    if (clientRef.current) {
      clientRef.current.send(message);
    }
  }, []);

  const sendRaw = useCallback((data: string | ArrayBuffer | Blob) => {
    if (clientRef.current) {
      clientRef.current.sendRaw(data);
    }
  }, []);

  const subscribe = useCallback(<T = unknown,>(eventType: string, handler: (data: T) => void) => {
    const subscriptions = subscriptionsRef.current;

    if (!subscriptions.has(eventType)) {
      subscriptions.set(eventType, new Set());
    }

    const handlers = subscriptions.get(eventType)!;
    handlers.add(handler as EventHandler<unknown>);

    return () => {
      handlers.delete(handler as EventHandler<unknown>);
      if (handlers.size === 0) {
        subscriptions.delete(eventType);
      }
    };
  }, []);

  const unsubscribeAll = useCallback(() => {
    subscriptionsRef.current.clear();
  }, []);

  useEffect(() => {
    mountedRef.current = true;

    if (autoConnect) {
      connect();
    }

    const subs = subscriptionsRef.current;
    return () => {
      mountedRef.current = false;
      if (clientRef.current) {
        clientRef.current.disconnect();
        clientRef.current = null;
      }
      subs.clear();
    };
  }, [autoConnect, connect]);

  const value = useMemo<WebSocketContextValue>(
    () => ({
      connectionStatus,
      lastMessage,
      sendMessage,
      sendRaw,
      connect,
      disconnect,
      reconnectAttempts,
      isConnected: connectionStatus === 'connected',
      isConnecting: connectionStatus === 'connecting',
      subscribe,
      unsubscribeAll,
    }),
    [
      connectionStatus,
      lastMessage,
      sendMessage,
      sendRaw,
      connect,
      disconnect,
      reconnectAttempts,
      subscribe,
      unsubscribeAll,
    ]
  );

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
}
