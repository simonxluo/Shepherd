import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  getChatModels,
  listConversations,
  getConversation,
  createConversation,
  deleteConversation,
  createMessage,
  streamingChatCompletion,
  type ChatModelInfo,
  type Conversation,
  type Message,
  type StreamingChatParams,
} from '@/lib/api/chat';

// ===== Query keys =====
export const chatKeys = {
  models: ['chat', 'models'] as const,
  conversations: ['chat', 'conversations'] as const,
  conversation: (id: string) => ['chat', 'conversations', id] as const,
};

// ===== Model hooks =====

export function useChatModels() {
  return useQuery({
    queryKey: chatKeys.models,
    queryFn: getChatModels,
    refetchInterval: 10_000,
    staleTime: 5_000,
  });
}

// ===== Conversation hooks =====

export function useConversations() {
  return useQuery({
    queryKey: chatKeys.conversations,
    queryFn: () => listConversations(50, 0).then((r) => r.items ?? []),
  });
}

export function useConversation(id: string | null) {
  return useQuery({
    queryKey: chatKeys.conversation(id ?? ''),
    queryFn: () => getConversation(id!),
    enabled: !!id,
  });
}

export function useCreateConversation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { model: string; title?: string }) =>
      createConversation(body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: chatKeys.conversations });
    },
  });
}

export function useDeleteConversation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: deleteConversation,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: chatKeys.conversations });
    },
  });
}

export function useSaveMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      conversationId,
      role,
      content,
    }: {
      conversationId: string;
      role: string;
      content: string;
    }) => createMessage(conversationId, { role, content }),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: chatKeys.conversation(variables.conversationId),
      });
      qc.invalidateQueries({ queryKey: chatKeys.conversations });
    },
  });
}

// ===== Streaming chat hook =====

export interface UseStreamingChatOptions {
  onChunk?: (text: string) => void;
  onComplete?: (fullText: string) => void;
  onError?: (error: Error) => void;
}

export function useStreamingChat(opts?: UseStreamingChatOptions) {
  let abortController: AbortController | null = null;

  const send = (params: Omit<StreamingChatParams, 'onChunk' | 'onComplete' | 'onError' | 'signal'>) => {
    abortController?.abort();
    abortController = new AbortController();

    streamingChatCompletion({
      ...params,
      signal: abortController.signal,
      onChunk: opts?.onChunk ?? (() => {}),
      onComplete: opts?.onComplete ?? (() => {}),
      onError: opts?.onError ?? (() => {}),
    });
  };

  const abort = () => {
    abortController?.abort();
    abortController = null;
  };

  return { send, abort };
}

// ===== Re-export types =====
export type { ChatModelInfo, Conversation, Message };
