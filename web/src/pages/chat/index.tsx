import { useState, useRef, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { MessageSquare, Plus, Trash2, Loader2, History, X, CircleDot, Pencil, Check, Search } from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import { ChatMessage } from '@/features/chat/components/ChatMessage';
import { ChatInput } from '@/features/chat/components/ChatInput';
import {
  useChatModels,
  useConversations,
  useCreateConversation,
  useUpdateConversation,
  useDeleteConversation,
  useSaveMessage,
  useStreamingChat,
  type ChatModelInfo,
} from '@/features/chat';
import { chatApi } from '@/lib/api/chat';
import { toast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';
import { useChatStore } from '@/stores/chatStore';

interface DisplayMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  images?: string[];
  timestamp: number;
}

function groupConversations(conversations: { id: string; title?: string; model: string; updatedAt: string }[], t: (key: string) => string) {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const yesterday = today - 86400000;

  const groups: { label: string; items: typeof conversations }[] = [
    { label: t('chat.timeGroups.today'), items: [] },
    { label: t('chat.timeGroups.yesterday'), items: [] },
    { label: t('chat.timeGroups.earlier'), items: [] },
  ];

  for (const conv of conversations) {
    const t = new Date(conv.updatedAt).getTime();
    if (t >= today) {
      groups[0].items.push(conv);
    } else if (t >= yesterday) {
      groups[1].items.push(conv);
    } else {
      groups[2].items.push(conv);
    }
  }

  return groups.filter((g) => g.items.length > 0);
}

export function ChatPage() {
  const { t } = useTranslation();
  const alertDialog = useAlertDialog();

  // Zustand store — persists across route changes
  const activeConvId = useChatStore((s) => s.activeConvId);
  const selectedModel = useChatStore((s) => s.selectedModel);
  const showSidebar = useChatStore((s) => s.showSidebar);
  const setActiveConvId = useChatStore((s) => s.setActiveConvId);
  const setSelectedModel = useChatStore((s) => s.setSelectedModel);
  const setShowSidebar = useChatStore((s) => s.setShowSidebar);
  const toggleSidebar = useChatStore((s) => s.toggleSidebar);

  // Models
  const { data: models = [], isLoading: modelsLoading } = useChatModels();

  // Conversations
  const { data: conversations = [] } = useConversations();
  const createConv = useCreateConversation();
  const updateConv = useUpdateConversation();
  const deleteConv = useDeleteConversation();
  const saveMessage = useSaveMessage();

  // Chat state
  const [messages, setMessages] = useState<DisplayMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [currentResponse, setCurrentResponse] = useState('');
  const [modelLoading, setModelLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesRef = useRef<DisplayMessage[]>([]);
  const [streamingStartTime, setStreamingStartTime] = useState(0);
  const [sidebarSearch, setSidebarSearch] = useState('');
  const [editingConvId, setEditingConvId] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const initialLoadDone = useRef(false);

  const streaming = useStreamingChat({
    onChunk: (text) => {
      setCurrentResponse((prev) => prev + text);
    },
    onComplete: (fullText) => {
      const assistantMsg: DisplayMessage = {
        role: 'assistant',
        content: fullText,
        timestamp: Date.now(),
      };
      setMessages((prev) => {
        const updated = [...prev, assistantMsg];
        messagesRef.current = updated;
        return updated;
      });
      setCurrentResponse('');
      setIsStreaming(false);

      if (activeConvId) {
        saveMessage.mutate({
          conversationId: activeConvId,
          role: 'assistant',
          content: fullText,
        });
      }
    },
    onError: (error) => {
      console.error('Chat error:', error);
      toast.error(t('chat.error.chatFailed'), error.message);
      setCurrentResponse('');
      setIsStreaming(false);
      setModelLoading(false);
    },
  });

  // Auto-scroll
  useEffect(() => {
    if (messages.length > 0 || currentResponse) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, currentResponse]);

  // Sync messagesRef
  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  // Restore conversation on mount if activeConvId is set
  useEffect(() => {
    if (initialLoadDone.current) return;
    initialLoadDone.current = true;

    if (activeConvId) {
      chatApi.getConversation(activeConvId)
        .then((data) => {
          const msgs: DisplayMessage[] = (data.messages ?? []).map((m) => ({
            role: m.role as DisplayMessage['role'],
            content: m.content,
            timestamp: new Date(m.createdAt).getTime(),
          }));
          setMessages(msgs);
          messagesRef.current = msgs;

          if (data.conversation?.model) {
            const match = models.find((m) => m.id === data.conversation.model);
            if (match) setSelectedModel(match.id);
          }
        })
        .catch(() => {
          // Conversation may have been deleted
          setActiveConvId(null);
        });
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Find selected model info
  const selectedModelInfo = models.find(
    (m) => m.id === selectedModel || m.alias === selectedModel || m.name === selectedModel,
  );

  const ensureConversation = useCallback(async (): Promise<string | null> => {
    if (activeConvId) return activeConvId;

    const modelId = selectedModelInfo?.id ?? selectedModel;
    try {
      const data = await createConv.mutateAsync({ model: modelId, title: t('chat.newConversation') });
      setActiveConvId(data.conversation.id);
      return data.conversation.id;
    } catch {
      toast.error(t('chat.error.createFailed'), t('chat.error.createFailedDesc'));
      return null;
    }
  }, [activeConvId, selectedModelInfo, selectedModel, createConv, setActiveConvId, toast]);

  const handleSend = useCallback(
    async (content: string, images?: string[]) => {
      if (!selectedModel) {
        toast.warning(t('chat.error.selectModel'), t('chat.error.selectModelDesc'));
        return;
      }

      const userMessage: DisplayMessage = {
        role: 'user',
        content,
        images,
        timestamp: Date.now(),
      };

      const newMessages = [...messagesRef.current, userMessage];
      setMessages(newMessages);
      messagesRef.current = newMessages;
      setIsStreaming(true);
      setCurrentResponse('');

      // Only show loading indicator if model is NOT already loaded
      if (!selectedModelInfo?.isLoaded) {
        setModelLoading(true);
      }
      setStreamingStartTime(Date.now());

      // Auto-create conversation if needed
      const convId = await ensureConversation();

      // Save user message
      if (convId) {
        saveMessage.mutate({
          conversationId: convId,
          role: 'user',
          content,
          metadata: images?.length ? { images } : undefined,
        });

        // Update title on first message
        if (newMessages.filter((m) => m.role === 'user').length === 1) {
          const title = content.slice(0, 50) + (content.length > 50 ? '...' : '');
          updateConv.mutate({ id: convId, title });
        }
      }

      const modelId = selectedModelInfo?.id ?? selectedModel;

      // Build message content — multimodal if images present
      const apiMessages = newMessages.map((m) => {
        if (m.images?.length) {
          return {
            role: m.role,
            content: [
              { type: 'text' as const, text: m.content },
              ...m.images.map((url) => ({ type: 'image_url' as const, image_url: { url } })),
            ],
          };
        }
        return { role: m.role, content: m.content };
      });

      streaming.send({
        model: modelId,
        messages: apiMessages,
        temperature: 0.7,
        topP: 0.95,
      });

      const loadingTimeout = setTimeout(() => setModelLoading(false), 15_000);
      return () => clearTimeout(loadingTimeout);
    },
    [
      selectedModel,
      selectedModelInfo,
      ensureConversation,
      saveMessage,
      updateConv,
      streaming,
      toast,
    ],
  );

  // Clear model loading once we receive first chunk
  useEffect(() => {
    if (currentResponse && modelLoading) {
      setModelLoading(false);
    }
  }, [currentResponse, modelLoading]);

  const handleStop = () => {
    streaming.abort();
    setIsStreaming(false);
    setCurrentResponse('');
    setModelLoading(false);
  };

  const handleNewChat = async () => {
    if (messages.length > 0) {
      const confirmed = await alertDialog.confirm({
        title: t('chat.newChat'),
        description: t('chat.newChatConfirm'),
      });
      if (!confirmed) return;
    }

    if (!selectedModel) {
      toast.warning(t('chat.error.selectModel'), t('chat.error.selectModelForNew'));
      return;
    }

    const modelId = selectedModelInfo?.id ?? selectedModel;
    createConv.mutate(
      { model: modelId, title: t('chat.newConversation') },
      {
        onSuccess: (data) => {
          setActiveConvId(data.conversation.id);
          setMessages([]);
          setCurrentResponse('');
        },
      },
    );
  };

  const handleLoadConversation = async (convId: string) => {
    try {
      const data = await chatApi.getConversation(convId);
      const msgs: DisplayMessage[] = (data.messages ?? []).map((m) => ({
        role: m.role as DisplayMessage['role'],
        content: m.content,
        timestamp: new Date(m.createdAt).getTime(),
      }));
      setMessages(msgs);
      messagesRef.current = msgs;
      setActiveConvId(convId);
      setCurrentResponse('');

      if (data.conversation?.model) {
        const match = models.find((m) => m.id === data.conversation.model);
        if (match) setSelectedModel(match.id);
      }
    } catch {
      toast.error(t('chat.error.loadFailed'), t('chat.error.loadFailedDesc'));
    }
  };

  const handleDeleteConversation = async (convId: string) => {
    const confirmed = await alertDialog.confirm({
      title: t('chat.deleteConversation'),
      description: t('chat.deleteConversationConfirm'),
      variant: 'destructive',
    });
    if (!confirmed) return;

    deleteConv.mutate(convId, {
      onSuccess: () => {
        if (activeConvId === convId) {
          setActiveConvId(null);
          setMessages([]);
          setCurrentResponse('');
        }
        toast.success(t('chat.deleted'), t('chat.deletedDesc'));
      },
    });
  };

  const handleRenameConversation = (convId: string, currentTitle: string) => {
    setEditingConvId(convId);
    setEditTitle(currentTitle);
  };

  const handleSaveRename = (convId: string) => {
    if (editTitle.trim()) {
      updateConv.mutate({ id: convId, title: editTitle.trim() });
    }
    setEditingConvId(null);
  };

  const getModelLabel = (model: ChatModelInfo) => {
    const name = model.alias || model.name;
    return model.isLoaded ? name : `${name} (${t('chat.modelNotLoaded')})`;
  };

  const filteredConversations = sidebarSearch
    ? conversations.filter(
        (c) =>
          c.title?.toLowerCase().includes(sidebarSearch.toLowerCase()) ||
          c.model.toLowerCase().includes(sidebarSearch.toLowerCase()),
      )
    : conversations;

  const groupedConversations = groupConversations(filteredConversations, t);

  return (
    <div className="h-full flex bg-background text-foreground">
      {/* Conversation sidebar — always visible on md+ */}
      <div
        className={`w-72 border-r flex-col bg-muted/20 shrink-0 ${
          showSidebar ? 'flex' : 'hidden md:flex'
        }`}
      >
        <div className="flex items-center justify-between px-4 py-3 border-b">
          <h2 className="font-semibold text-sm">{t('chat.history')}</h2>
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={handleNewChat} title={t('chat.newChat')}>
              <Plus className="w-4 h-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 md:hidden"
              onClick={() => setShowSidebar(false)}
            >
              <X className="w-4 h-4" />
            </Button>
          </div>
        </div>

        {/* Search */}
        <div className="px-3 py-2 border-b">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
            <input
              type="text"
              placeholder={t('chat.searchConversation')}
              value={sidebarSearch}
              onChange={(e) => setSidebarSearch(e.target.value)}
              className="w-full pl-8 pr-3 py-1.5 text-sm bg-background border border-input rounded-md outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto">
          {conversations.length === 0 ? (
            <p className="text-sm text-muted-foreground p-4 text-center">{t('chat.noHistory')}</p>
          ) : (
            groupedConversations.map((group) => (
              <div key={group.label}>
                <div className="px-4 py-1.5 text-xs font-medium text-muted-foreground sticky top-0 bg-muted/20">
                  {group.label}
                </div>
                {group.items.map((conv) => (
                  <div
                    key={conv.id}
                    className={`flex items-center gap-2 px-4 py-2.5 cursor-pointer hover:bg-accent/50 transition-colors group ${
                      activeConvId === conv.id ? 'bg-accent' : ''
                    }`}
                    onClick={() => handleLoadConversation(conv.id)}
                  >
                    <MessageSquare className="w-4 h-4 shrink-0 text-muted-foreground" />
                    <div className="flex-1 min-w-0">
                      {editingConvId === conv.id ? (
                        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                          <input
                            type="text"
                            value={editTitle}
                            onChange={(e) => setEditTitle(e.target.value)}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') handleSaveRename(conv.id);
                              if (e.key === 'Escape') setEditingConvId(null);
                            }}
                            className="flex-1 text-sm bg-background border rounded px-1 py-0.5 outline-none focus:ring-1 focus:ring-ring"
                            autoFocus
                          />
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-5 w-5 shrink-0"
                            onClick={() => handleSaveRename(conv.id)}
                          >
                            <Check className="w-3 h-3" />
                          </Button>
                        </div>
                      ) : (
                        <p className="text-sm truncate">{conv.title || t('chat.newConversation')}</p>
                      )}
                      <p className="text-xs text-muted-foreground truncate">{conv.model}</p>
                    </div>
                    <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="shrink-0 h-6 w-6"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleRenameConversation(conv.id, conv.title || t('chat.newConversation'));
                        }}
                      >
                        <Pencil className="w-3 h-3" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="shrink-0 h-6 w-6"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteConversation(conv.id);
                        }}
                      >
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ))
          )}
        </div>
      </div>

      {/* Main chat area */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
          <div className="flex items-center gap-3">
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleSidebar}
              title={t('chat.history')}
              className="md:hidden"
            >
              <History className="w-5 h-5" />
            </Button>
            <MessageSquare className="w-5 h-5 text-primary" />
            <h1 className="text-lg font-semibold">{t('chat.aiChat')}</h1>
          </div>

          <div className="flex items-center gap-2">
            {/* Model selector */}
            <Select value={selectedModel} onValueChange={setSelectedModel}>
              <SelectTrigger className="w-[240px] px-3 py-1.5 border rounded-md bg-background text-sm">
                <SelectValue placeholder={modelsLoading ? t('common.loading') : t('chat.selectModel')} />
              </SelectTrigger>
              <SelectContent>
                {models.map((model) => (
                  <SelectItem key={model.id} value={model.id}>
                    <span className="flex items-center gap-2">
                      <CircleDot
                        className={`w-3 h-3 ${
                          model.isLoaded ? 'text-green-500' : 'text-muted-foreground'
                        }`}
                      />
                      {getModelLabel(model)}
                      {model.isVision && (
                        <span className="text-xs text-blue-500" title={t('chat.supportsImage')}>📷</span>
                      )}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Button variant="ghost" size="icon" onClick={handleNewChat} title={t('chat.newChat')} className="md:hidden">
              <Plus className="w-5 h-5" />
            </Button>

            {messages.length > 0 && (
              <Button
                variant="ghost"
                size="icon"
                onClick={async () => {
                  const confirmed = await alertDialog.confirm({
                    title: t('chat.clearChat'),
                    description: t('chat.clearChatConfirm'),
                    variant: 'destructive',
                  });
                  if (confirmed) {
                    setMessages([]);
                    setCurrentResponse('');
                  }
                }}
                title={t('chat.clearChat')}
              >
                <Trash2 className="w-5 h-5" />
              </Button>
            )}
          </div>
        </div>

        {/* Model loading indicator */}
        {modelLoading && (
          <div className="flex items-center gap-2 px-4 py-2 bg-yellow-500/10 border-b border-yellow-500/20 text-sm text-yellow-600 dark:text-yellow-400">
            <Loader2 className="w-4 h-4 animate-spin" />
            <span>{t('chat.loadingModel', { model: selectedModelInfo?.name ?? selectedModel })}</span>
          </div>
        )}

        {/* Message list */}
        <div className="flex-1 overflow-y-auto">
          {messages.length === 0 && !currentResponse ? (
            <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
              <MessageSquare className="w-16 h-16 mb-4 opacity-50" />
              <p className="text-lg mb-2">{t('chat.startConversation')}</p>
              <p className="text-sm">{t('chat.startConversationHint')}</p>
              {selectedModelInfo && !selectedModelInfo.isLoaded && (
                <p className="text-xs mt-2 text-yellow-500">
                  {t('chat.modelAutoLoad')}
                </p>
              )}
            </div>
          ) : (
            <div className="divide-y divide-border">
              {messages.map((message, index) => (
                <ChatMessage key={index} message={message} />
              ))}

              {/* Streaming response */}
              {currentResponse && (
                <ChatMessage
                  message={{
                    role: 'assistant',
                    content: currentResponse,
                    timestamp: streamingStartTime,
                  }}
                  isStreaming
                />
              )}
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* Input area */}
        <ChatInput
          onSend={handleSend}
          onStop={handleStop}
          disabled={!selectedModel || isStreaming || modelLoading}
          isStreaming={isStreaming || modelLoading}
          supportsVision={selectedModelInfo?.isVision ?? false}
          placeholder={
            !selectedModel
              ? t('chat.placeholderSelectModel')
              : modelLoading
                ? t('chat.placeholderLoading')
                : t('chat.placeholderInput')
          }
        />
      </div>
    </div>
  );
}
