import { apiClient } from './client';

// ===== Types =====

export interface ChatModelInfo {
  id: string;
  name: string;
  alias?: string;
  state: string;
  isLoaded: boolean;
  isVision: boolean;
  port?: number;
}

export interface Conversation {
  id: string;
  model: string;
  title?: string;
  systemPrompt?: string;
  messageCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface Message {
  id: string;
  conversationId: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  tokenCount?: number;
  createdAt: string;
}

// ===== Streaming Chat =====

export interface ContentPart {
  type: 'text' | 'image_url';
  text?: string;
  image_url?: { url: string };
}

export interface StreamingChatParams {
  model: string;
  messages: { role: string; content: string | ContentPart[] }[];
  temperature?: number;
  maxTokens?: number;
  topP?: number;
  stop?: string[];
  signal?: AbortSignal;
  onChunk: (text: string) => void;
  onComplete: (fullText: string) => void;
  onError: (error: Error) => void;
}

/**
 * Chat API — unified object matching the pattern used by downloadsApi, systemApi, etc.
 */
export const chatApi = {
  // ===== Models =====

  getChatModels: async (): Promise<ChatModelInfo[]> => {
    const res = await apiClient.get<{ success: boolean; models: ChatModelInfo[] }>(
      '/chat/models',
    );
    return res.models ?? [];
  },

  // ===== Conversations =====

  listConversations: (
    limit = 50,
    offset = 0,
  ): Promise<{ items: Conversation[]; count: number }> =>
    apiClient.get('/conversations', { limit, offset }),

  getConversation: (
    id: string,
  ): Promise<{ conversation: Conversation; messages: Message[] }> =>
    apiClient.get(`/conversations/${id}`),

  createConversation: (body: {
    model: string;
    title?: string;
    systemPrompt?: string;
  }): Promise<{ conversation: Conversation }> =>
    apiClient.post('/conversations', body),

  updateConversation: (
    id: string,
    body: { title?: string; systemPrompt?: string },
  ): Promise<{ conversation: Conversation }> =>
    apiClient.put(`/conversations/${id}`, body),

  deleteConversation: async (id: string): Promise<void> => {
    await apiClient.delete(`/conversations/${id}`);
  },

  // ===== Messages =====

  createMessage: (
    conversationId: string,
    body: { role: string; content: string; tokenCount?: number; metadata?: Record<string, unknown> },
  ): Promise<{ message: Message }> =>
    apiClient.post(`/conversations/${conversationId}/messages`, body),

  // ===== Streaming Chat =====

  streamingChatCompletion: (params: StreamingChatParams): void => {
    const {
      model,
      messages,
      temperature = 0.7,
      maxTokens,
      topP,
      stop,
      signal,
      onChunk,
      onComplete,
      onError,
    } = params;

    (async () => {
      try {
        const response = await fetch('/api/chat/completions', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            model,
            messages,
            stream: true,
            temperature,
            max_tokens: maxTokens,
            top_p: topP,
            stop,
          }),
          signal,
        });

        if (!response.ok) {
          const errBody = await response.text();
          throw new Error(`HTTP ${response.status}: ${errBody}`);
        }

        const reader = response.body?.getReader();
        if (!reader) throw new Error('No reader available');

        const decoder = new TextDecoder();
        let fullText = '';
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split('\n');
          buffer = lines.pop() ?? '';

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed || !trimmed.startsWith('data: ')) continue;

            const data = trimmed.slice(6);
            if (data === '[DONE]') {
              onComplete(fullText);
              return;
            }

            try {
              const parsed = JSON.parse(data);
              const content = parsed.choices?.[0]?.delta?.content;
              if (content) {
                fullText += content;
                onChunk(content);
              }
            } catch {
              // skip malformed JSON
            }
          }
        }

        onComplete(fullText);
      } catch (err: unknown) {
        if (signal?.aborted) return;
        onError(err instanceof Error ? err : new Error('Unknown error'));
      }
    })();
  },
};
