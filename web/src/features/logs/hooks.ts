import { useEffect, useRef, useCallback, useState } from 'react';
import { apiClient } from '@/lib/api/client';

const LOG_LENGTH_LIMIT = 200 * 1024;

export function useLogStream(nodeUrl?: string) {
  const [logs, setLogs] = useState('');
  const [connected, setConnected] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  const readerRef = useRef<ReadableStreamDefaultReader<Uint8Array> | null>(null);

  const appendLog = useCallback((newData: string) => {
    setLogs(prev => {
      const updated = prev + newData;
      return updated.length > LOG_LENGTH_LIMIT ? updated.slice(-LOG_LENGTH_LIMIT) : updated;
    });
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    abortRef.current = controller;

    const url = nodeUrl
      ? `${nodeUrl}/logs/stream/text`
      : '/api/logs/stream/text';
    let cancelled = false;

    const connect = async () => {
      try {
        const response = await fetch(url, {
          signal: controller.signal,
          headers: { Accept: 'text/plain' },
        });

        if (!response.ok || !response.body) {
          setConnected(false);
          return;
        }

        setConnected(true);
        setLogs('');

        const reader = response.body.getReader();
        readerRef.current = reader;
        const decoder = new TextDecoder();

        while (!cancelled) {
          const { done, value } = await reader.read();
          if (done) break;
          appendLog(decoder.decode(value, { stream: true }));
        }
      } catch (err: unknown) {
        if (err instanceof Error && err.name === 'AbortError') return;
        setConnected(false);
      }
    };

    connect();

    return () => {
      cancelled = true;
      controller.abort();
      readerRef.current?.cancel().catch(() => {});
      readerRef.current = null;
      setConnected(false);
    };
  }, [nodeUrl, appendLog]);

  const clear = useCallback(() => setLogs(''), []);

  return { logs, connected, clear };
}
