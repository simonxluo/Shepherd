import { useMutation } from '@tanstack/react-query';

/**
 * Chat message
 */
export interface ChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp?: number;
}

/**
 * Chat completion request parameters
 */
export interface ChatCompletionParams {
  model: string;
  messages: ChatMessage[];
  stream?: boolean;
  temperature?: number;
  topP?: number;
  topK?: number;
  maxTokens?: number;
  repeatPenalty?: number;
  stop?: string[];
}

/**
 * Streaming chat completion hook
 */
export function useStreamingChat() {
  return useMutation({
    mutationFn: async (
      params: ChatCompletionParams & {
        onChunk?: (chunk: string) => void;
        onComplete?: (message: string) => void;
        onError?: (error: Error) => void;
      }
    ) => {
      const { onChunk, onComplete, onError, ...requestParams } = params;

      try {
        const response = await fetch('/v1/chat/completions', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            ...requestParams,
            stream: true,
          }),
        });

        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }

        const reader = response.body?.getReader();
        const decoder = new TextDecoder();
        let fullMessage = '';
        let buffer = '';

        if (!reader) {
          throw new Error('No reader available');
        }

        while (true) {
          const { done, value } = await reader.read();

          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split('\n');
          buffer = lines.pop() ?? '';

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed) continue;

            if (trimmed.startsWith('data: ')) {
              const data = trimmed.slice(6);

              if (data === '[DONE]') {
                onComplete?.(fullMessage);
                return { success: true, message: fullMessage };
              }

              try {
                const parsed = JSON.parse(data);
                const content = parsed.choices?.[0]?.delta?.content;

                if (content) {
                  fullMessage += content;
                  onChunk?.(content);
                }
              } catch (e) {
                console.error('Failed to parse SSE data:', e);
              }
            }
          }
        }

        onComplete?.(fullMessage);
        return { success: true, message: fullMessage };
      } catch (error) {
        const err = error instanceof Error ? error : new Error('Unknown error');
        onError?.(err);
        throw err;
      }
    },
  });
}

/**
 * Fetch loaded model list for chat
 */
export async function getLoadedModels(): Promise<string[]> {
  try {
    const response = await fetch('/api/models/loaded');
    const data = await response.json();

    if (data.success && data.models) {
      return data.models.map((m: { alias?: string; name: string }) => m.alias || m.name);
    }

    return [];
  } catch (error) {
    console.error('Failed to get loaded models:', error);
    return [];
  }
}

/**
 * Chat history storage key
 */
export const CHAT_HISTORY_KEY = 'shepherd_chat_history';

/**
 * Chat history item
 */
export interface ChatHistoryItem {
  id: string;
  title: string;
  messages: ChatMessage[];
  model: string;
  createdAt: number;
  updatedAt: number;
}

/**
 * Load chat history
 */
export function loadChatHistory(): ChatHistoryItem[] {
  try {
    const stored = localStorage.getItem(CHAT_HISTORY_KEY);
    if (stored) {
      return JSON.parse(stored);
    }
  } catch (error) {
    console.error('Failed to load chat history:', error);
  }
  return [];
}

/**
 * Save chat history
 */
export function saveChatHistory(history: ChatHistoryItem[]): void {
  try {
    localStorage.setItem(CHAT_HISTORY_KEY, JSON.stringify(history));
  } catch (error) {
    console.error('Failed to save chat history:', error);
  }
}

/**
 * Create a new chat session
 */
export function createChatSession(model: string): ChatHistoryItem {
  return {
    id: `chat_${Date.now()}`,
    title: '新对话',
    messages: [],
    model,
    createdAt: Date.now(),
    updatedAt: Date.now(),
  };
}

/**
 * Delete a chat session
 */
export function deleteChatSession(sessionId: string): void {
  const history = loadChatHistory();
  const filtered = history.filter((item) => item.id !== sessionId);
  saveChatHistory(filtered);
}


