/**
 * WebSocket type definitions.
 * Type-safe interfaces for real-time communication.
 */

/**
 * WebSocket connection status
 */
export type WebSocketConnectionStatus =
  | 'connecting'    // Connecting
  | 'connected'     // Connected
  | 'disconnected'  // Disconnected
  | 'reconnecting'  // Reconnecting
  | 'error';        // Connection error

/**
 * Base WebSocket message interface
 */
export interface WebSocketMessage<T = unknown> {
  type: 'ping' | 'pong' | 'event' | 'notification' | 'error';
  timestamp: number;
  payload: T;
}

/**
 * WebSocket client options
 */
export interface WebSocketClientOptions {
  /** WebSocket server URL */
  url: string;
  /** Heartbeat interval (ms), default 30000 */
  heartbeatInterval?: number;
  /** Heartbeat timeout (ms), default 10000 */
  heartbeatTimeout?: number;
  /** Max reconnect attempts, default 5 */
  maxReconnectAttempts?: number;
  /** Initial reconnect delay (ms), default 1000 */
  initialReconnectDelay?: number;
  /** Max reconnect delay (ms), default 30000 */
  maxReconnectDelay?: number;
  /** Auto reconnect, default true */
  autoReconnect?: boolean;
  /** Send auth payload on connect */
  authPayload?: Record<string, unknown>;
  /** Debug mode */
  debug?: boolean;
}

/**
 * WebSocket event handlers
 */
export interface WebSocketEventHandlers {
  onOpen?: (event: Event) => void;
  onClose?: (event: CloseEvent) => void;
  onError?: (event: Event) => void;
  onMessage?: (message: WebSocketMessage) => void;
  onReconnecting?: (attempt: number, delay: number) => void;
  onReconnectFailed?: () => void;
  onStatusChange?: (status: WebSocketConnectionStatus) => void;
}

/**
 * WebSocket context value
 */
export interface WebSocketContextValue {
  /** Current connection status */
  connectionStatus: WebSocketConnectionStatus;
  /** Last received message */
  lastMessage: WebSocketMessage | null;
  /** Send message */
  sendMessage: <T = unknown>(message: WebSocketMessage<T>) => void;
  /** Send raw data */
  sendRaw: (data: string | ArrayBuffer | Blob) => void;
  /** Manual connect */
  connect: () => void;
  /** Manual disconnect */
  disconnect: () => void;
  /** Reconnect attempts */
  reconnectAttempts: number;
  /** Connected */
  isConnected: boolean;
  /** Connecting */
  isConnecting: boolean;
  /** Subscribe to specific event type */
  subscribe: <T = unknown>(
    eventType: string,
    handler: (data: T) => void
  ) => () => void;
  /** Unsubscribe all */
  unsubscribeAll: () => void;
}

/**
 * WebSocket Provider Props
 */
export interface WebSocketProviderProps {
  /** Child components */
  children: React.ReactNode;
  /** WebSocket server URL (optional; read from config by default) */
  url?: string;
  /** Auto connect, default true */
  autoConnect?: boolean;
  /** Custom options */
  options?: Partial<WebSocketClientOptions>;
  /** Connection success callback */
  onConnect?: () => void;
  /** Disconnection callback */
  onDisconnect?: () => void;
  /** Error callback */
  onError?: (error: Error) => void;
}
