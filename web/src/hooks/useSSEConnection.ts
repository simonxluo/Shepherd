/**
 * 通用 SSE 连接管理 Hook
 *
 * 封装 EventSource 的创建、自动重连（指数退避）、连接清理等通用逻辑。
 * useSSE 和 useLogs 共用此 hook，各自只关注事件处理。
 */

import { useEffect, useRef, useCallback, useState } from 'react';
import { apiClient } from '@/lib/api/client';

/** SSE 连接状态 */
export type SSEConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error';

/** useSSEConnection 配置选项 */
export interface UseSSEConnectionOptions {
  /** SSE 端点路径（相对于 API base URL），例如 '/events' 或 '/logs/stream' */
  url: string;
  /** URL 查询参数（追加到 url 后面） */
  params?: Record<string, string>;
  /** 重连基础间隔（毫秒），默认 1000 */
  reconnectInterval?: number;
  /** 最大重连次数，默认 10 */
  maxReconnectAttempts?: number;
  /** 连接打开回调。isReconnect 为 true 表示这是一次重连（非首次连接） */
  onOpen?: (isReconnect: boolean) => void;
  /** 收到消息回调 */
  onMessage?: (event: MessageEvent) => void;
  /** 发生错误回调（仅在达到最大重连次数时触发） */
  onError?: (error: Event) => void;
}

/** useSSEConnection 返回值 */
export interface UseSSEConnectionReturn {
  /** 当前连接状态 */
  connectionState: SSEConnectionState;
  /** 手动连接/重连（重置重连计数） */
  connect: () => void;
  /** 断开连接 */
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

  // 使用 ref 保存回调，避免因回调变化触发重连
  const onOpenRef = useRef(onOpen);
  const onMessageRef = useRef(onMessage);
  const onErrorRef = useRef(onError);
  onOpenRef.current = onOpen;
  onMessageRef.current = onMessage;
  onErrorRef.current = onError;

  /**
   * 清除重连定时器
   */
  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  /**
   * 断开 SSE 连接
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
   * 连接到 SSE 端点
   */
  const connect = useCallback(() => {
    // 如果已经连接或正在连接，不重复连接
    if (
      eventSourceRef.current?.readyState === EventSource.OPEN ||
      eventSourceRef.current?.readyState === EventSource.CONNECTING
    ) {
      return;
    }

    // 清理旧连接
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    // 清除待处理的重连
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

        // 检查是否为正常关闭
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
   * 手动重连：重置重连计数后连接
   */
  const manualReconnect = useCallback(() => {
    reconnectAttemptsRef.current = 0;
    disconnect();
    connect();
  }, [connect, disconnect]);

  // 组件挂载时连接，卸载时断开
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
