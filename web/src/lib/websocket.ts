import type {
  WebSocketClientOptions,
  WebSocketConnectionStatus,
  WebSocketEventHandlers,
  WebSocketMessage,
} from '@/types/websocket';

const DEFAULT_OPTIONS: Required<Omit<WebSocketClientOptions, 'url' | 'authPayload'>> & {
  authPayload?: Record<string, unknown>;
} = {
  heartbeatInterval: 30000,
  heartbeatTimeout: 10000,
  maxReconnectAttempts: 5,
  initialReconnectDelay: 1000,
  maxReconnectDelay: 30000,
  autoReconnect: true,
  debug: false,
};

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private options: Required<Omit<WebSocketClientOptions, 'url' | 'authPayload'>> & {
    url: string;
    authPayload?: Record<string, unknown>;
  };
  private handlers: WebSocketEventHandlers = {};
  private reconnectAttempts = 0;
  private reconnectTimeoutId: ReturnType<typeof setTimeout> | null = null;
  private heartbeatIntervalId: ReturnType<typeof setInterval> | null = null;
  private heartbeatTimeoutId: ReturnType<typeof setTimeout> | null = null;
  private status: WebSocketConnectionStatus = 'disconnected';
  private isIntentionallyClosed = false;
  private messageQueue: string[] = [];

  constructor(options: WebSocketClientOptions) {
    this.options = {
      ...DEFAULT_OPTIONS,
      ...options,
    } as Required<Omit<WebSocketClientOptions, 'authPayload'>> & {
      url: string;
      authPayload?: Record<string, unknown>;
    };
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN || this.ws?.readyState === WebSocket.CONNECTING) {
      this.log('Already connected or connecting');
      return;
    }

    this.isIntentionallyClosed = false;
    this.setStatus('connecting');

    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = this.options.url.replace(/^https?:/, protocol).replace(/^http:\/\//, 'ws://').replace(/^https:\/\//, 'wss://');
      
      this.log('Connecting to:', host);
      this.ws = new WebSocket(host);

      this.ws.onopen = this.handleOpen.bind(this);
      this.ws.onclose = this.handleClose.bind(this);
      this.ws.onerror = this.handleError.bind(this);
      this.ws.onmessage = this.handleMessage.bind(this);
    } catch (error) {
      this.log('Failed to create WebSocket:', error);
      this.setStatus('error');
      this.scheduleReconnect();
    }
  }

  disconnect(): void {
    this.isIntentionallyClosed = true;
    this.cleanup();
    this.setStatus('disconnected');
    this.reconnectAttempts = 0;
  }

  send<T = unknown>(message: WebSocketMessage<T>): void {
    const data = JSON.stringify(message);
    
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(data);
      this.log('Sent:', message);
    } else {
      this.log('Queuing message (not connected):', message);
      this.messageQueue.push(data);
    }
  }

  sendRaw(data: string | ArrayBuffer | Blob): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    }
  }

  getStatus(): WebSocketConnectionStatus {
    return this.status;
  }

  getReconnectAttempts(): number {
    return this.reconnectAttempts;
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  setHandlers(handlers: WebSocketEventHandlers): void {
    this.handlers = { ...this.handlers, ...handlers };
  }

  private handleOpen(event: Event): void {
    this.log('Connected');
    this.setStatus('connected');
    this.reconnectAttempts = 0;
    
    this.flushMessageQueue();
    
    if (this.options.authPayload) {
      this.send({
        type: 'event',
        timestamp: Date.now(),
        payload: { auth: this.options.authPayload },
      });
    }
    
    this.startHeartbeat();
    
    this.handlers.onOpen?.(event);
  }

  private handleClose(event: CloseEvent): void {
    this.log('Disconnected:', event.code, event.reason);
    this.cleanup();
    
    if (!this.isIntentionallyClosed) {
      this.setStatus('disconnected');
      this.scheduleReconnect();
    }
    
    this.handlers.onClose?.(event);
  }

  private handleError(event: Event): void {
    this.log('Error:', event);
    this.setStatus('error');
    this.handlers.onError?.(event);
  }

  private handleMessage(event: MessageEvent): void {
    try {
      const message: WebSocketMessage = JSON.parse(event.data);
      this.log('Received:', message);
      
      if (message.type === 'pong') {
        this.handlePong();
        return;
      }
      
      this.handlers.onMessage?.(message);
    } catch (error) {
      this.log('Failed to parse message:', error);
    }
  }

  private startHeartbeat(): void {
    this.stopHeartbeat();
    
    this.heartbeatIntervalId = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send({ type: 'ping', timestamp: Date.now(), payload: null });
        
        this.heartbeatTimeoutId = setTimeout(() => {
          this.log('Heartbeat timeout, reconnecting...');
          this.ws?.close();
        }, this.options.heartbeatTimeout);
      }
    }, this.options.heartbeatInterval);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatIntervalId) {
      clearInterval(this.heartbeatIntervalId);
      this.heartbeatIntervalId = null;
    }
    if (this.heartbeatTimeoutId) {
      clearTimeout(this.heartbeatTimeoutId);
      this.heartbeatTimeoutId = null;
    }
  }

  private handlePong(): void {
    if (this.heartbeatTimeoutId) {
      clearTimeout(this.heartbeatTimeoutId);
      this.heartbeatTimeoutId = null;
    }
  }

  private scheduleReconnect(): void {
    if (!this.options.autoReconnect || this.isIntentionallyClosed) {
      return;
    }

    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      this.log('Max reconnect attempts reached');
      this.setStatus('error');
      this.handlers.onReconnectFailed?.();
      return;
    }

    const delay = Math.min(
      this.options.initialReconnectDelay * Math.pow(2, this.reconnectAttempts),
      this.options.maxReconnectDelay
    );

    this.reconnectAttempts++;
    this.setStatus('reconnecting');
    this.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.options.maxReconnectAttempts})`);
    
    this.handlers.onReconnecting?.(this.reconnectAttempts, delay);

    this.reconnectTimeoutId = setTimeout(() => {
      this.connect();
    }, delay);
  }

  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0 && this.ws?.readyState === WebSocket.OPEN) {
      const data = this.messageQueue.shift();
      if (data) {
        this.ws.send(data);
      }
    }
  }

  private cleanup(): void {
    this.stopHeartbeat();
    
    if (this.reconnectTimeoutId) {
      clearTimeout(this.reconnectTimeoutId);
      this.reconnectTimeoutId = null;
    }
    
    if (this.ws) {
      this.ws.onopen = null;
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws.onmessage = null;
      
      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        this.ws.close(1000, 'Client disconnect');
      }
      this.ws = null;
    }
  }

  private setStatus(status: WebSocketConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      this.handlers.onStatusChange?.(status);
    }
  }

  private log(...args: unknown[]): void {
    if (this.options.debug) {
      console.log('[WebSocket]', ...args);
    }
  }
}

