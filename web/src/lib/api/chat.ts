import { apiClient } from './client';

//  Types

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

//  Generation Stats

export interface GenerationStats {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  tokensPerSecond: number;
  durationMs: number;
}

//  Streaming Chat

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
  topK?: number;
  repeatPenalty?: number;
  stop?: string[];
  signal?: AbortSignal;
  onChunk: (text: string) => void;
  onComplete: (fullText: string, stats?: GenerationStats) => void;
  onError: (error: Error) => void;
}

/**
 * Chat API — unified object matching the pattern used by downloadsApi, systemApi, etc.
 */
export const chatApi = {
  //  Models

  getChatModels: async (): Promise<ChatModelInfo[]> => {
    const res = await apiClient.get<{ success: boolean; models: ChatModelInfo[] }>(
      '/chat/models',
    );
    return res.models ?? [];
  },

  //  Conversations

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

  //  Messages

  createMessage: (
    conversationId: string,
    body: { role: string; content: string; tokenCount?: number; metadata?: Record<string, unknown> },
  ): Promise<{ message: Message }> =>
    apiClient.post(`/conversations/${conversationId}/messages`, body),

  //  Streaming Chat

  streamingChatCompletion: (params: StreamingChatParams): void => {
    const {
      model,
      messages,
      temperature = 0.7,
      maxTokens,
      topP,
      topK,
      repeatPenalty,
      stop,
      signal,
      onChunk,
      onComplete,
      onError,
    } = params;

    (async () => {
      try {
        const body: Record<string, unknown> = {
          model,
          messages,
          stream: true,
          temperature,
          top_p: topP,
        };
        if (maxTokens !== undefined) body.max_tokens = maxTokens;
        if (topK !== undefined) body.top_k = topK;
        if (repeatPenalty !== undefined) body.repeat_penalty = repeatPenalty;
        if (stop !== undefined) body.stop = stop;

        const response = await fetch(`${apiClient.getBaseUrl()}/chat/completions`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
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
        let chunkCount = 0;
        const startTime = Date.now();
        let usage: { prompt_tokens?: number; completion_tokens?: number; total_tokens?: number } | undefined;
        const MAX_BUFFER_SIZE = 10 * 1024 * 1024; // 10MB 缓冲区上限，防止 [DONE] 丢失时内存无限增长

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          if (buffer.length > MAX_BUFFER_SIZE) {
            throw new Error(`流式缓冲区超出 ${MAX_BUFFER_SIZE / 1024 / 1024}MB 限制，服务端未发送 [DONE]`);
          }
          const lines = buffer.split('\n');
          buffer = lines.pop() ?? '';

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed || !trimmed.startsWith('data: ')) continue;

            const data = trimmed.slice(6);
            if (data === '[DONE]') {
              const durationMs = Date.now() - startTime;
              const stats: GenerationStats | undefined = usage
                ? {
                    promptTokens: usage.prompt_tokens ?? 0,
                    completionTokens: usage.completion_tokens ?? chunkCount,
                    totalTokens: usage.total_tokens ?? chunkCount,
                    tokensPerSecond: durationMs > 0
                      ? ((usage.completion_tokens ?? chunkCount) / durationMs) * 1000
                      : 0,
                    durationMs,
                  }
                : chunkCount > 0
                  ? {
                      promptTokens: 0,
                      completionTokens: chunkCount,
                      totalTokens: chunkCount,
                      tokensPerSecond: durationMs > 0 ? (chunkCount / durationMs) * 1000 : 0,
                      durationMs,
                    }
                  : undefined;
              onComplete(fullText, stats);
              return;
            }

            try {
              const parsed = JSON.parse(data);
              // Capture usage from chunk (some backends include in final chunk)
              if (parsed.usage) {
                usage = parsed.usage;
              }
              const content = parsed.choices?.[0]?.delta?.content;
              if (content) {
                fullText += content;
                chunkCount++;
                onChunk(content);
              }
            } catch {
              // skip malformed JSON
            }
          }
        }

        // If we got here without [DONE], still complete
        const durationMs = Date.now() - startTime;
        const stats: GenerationStats | undefined = chunkCount > 0
          ? {
              promptTokens: usage?.prompt_tokens ?? 0,
              completionTokens: usage?.completion_tokens ?? chunkCount,
              totalTokens: usage?.total_tokens ?? chunkCount,
              tokensPerSecond: durationMs > 0 ? (chunkCount / durationMs) * 1000 : 0,
              durationMs,
            }
          : undefined;
        onComplete(fullText, stats);
      } catch (err: unknown) {
        reader?.cancel().catch(() => {}); // Release ReadableStream to prevent connection leak
        if (signal?.aborted) return;
        onError(err instanceof Error ? err : new Error('Unknown error'));
      }
    })();
  },
};
