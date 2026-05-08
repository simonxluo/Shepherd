import { useState, useRef, useEffect, useCallback } from 'react';
import { MessageSquare, Plus, Trash2, Loader2, History, X, CircleDot } from 'lucide-react';
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
  useDeleteConversation,
  useSaveMessage,
  useStreamingChat,
} from '@/features/chat';
import { getConversation, type ChatModelInfo } from '@/lib/api/chat';
import { toast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';

interface DisplayMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: number;
}

export function ChatPage() {
  const alertDialog = useAlertDialog();

  // Models
  const { data: models = [], isLoading: modelsLoading } = useChatModels();
  const [selectedModel, setSelectedModel] = useState('');

  // Conversations
  const { data: conversations = [] } = useConversations();
  const [activeConvId, setActiveConvId] = useState<string | null>(null);
  const [showSidebar, setShowSidebar] = useState(false);
  const createConv = useCreateConversation();
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

      // Persist assistant message
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
      toast.error('聊天失败', error.message);
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

  // Find selected model info
  const selectedModelInfo = models.find(
    (m) => m.id === selectedModel || m.alias === selectedModel || m.name === selectedModel,
  );

  const handleSend = useCallback(
    (content: string) => {
      if (!selectedModel) {
        toast.warning('请先选择模型', '请从下拉列表中选择一个模型');
        return;
      }

      const userMessage: DisplayMessage = {
        role: 'user',
        content,
        timestamp: Date.now(),
      };

      const newMessages = [...messagesRef.current, userMessage];
      setMessages(newMessages);
      messagesRef.current = newMessages;
      setIsStreaming(true);
      setModelLoading(true);
      setStreamingStartTime(Date.now());
      setCurrentResponse('');

      // Persist user message
      if (activeConvId) {
        saveMessage.mutate({
          conversationId: activeConvId,
          role: 'user',
          content,
        });
      }

      const modelId = selectedModelInfo?.id ?? selectedModel;

      streaming.send({
        model: modelId,
        messages: newMessages.map((m) => ({ role: m.role, content: m.content })),
        temperature: 0.7,
        topP: 0.95,
      });

      // Model loading state clears after first chunk or timeout
      const loadingTimeout = setTimeout(() => setModelLoading(false), 15_000);
      // The model will be auto-loaded by the backend; once we get chunks, loading is done
      return () => clearTimeout(loadingTimeout);
    },
    [selectedModel, activeConvId, selectedModelInfo, saveMessage, streaming, toast],
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
        title: '新建对话',
        description: '确定要开始新对话吗？当前对话将被保留。',
      });
      if (!confirmed) return;
    }

    if (!selectedModel) {
      toast.warning('请先选择模型', '新建对话前请先选择一个模型');
      return;
    }

    // Create a new conversation on the server
    const modelId = selectedModelInfo?.id ?? selectedModel;
    createConv.mutate(
      { model: modelId, title: '新对话' },
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
      const data = await getConversation(convId);
      const msgs: DisplayMessage[] = (data.messages ?? []).map((m) => ({
        role: m.role as DisplayMessage['role'],
        content: m.content,
        timestamp: new Date(m.createdAt).getTime(),
      }));
      setMessages(msgs);
      messagesRef.current = msgs;
      setActiveConvId(convId);
      setCurrentResponse('');
      setShowSidebar(false);

      // Set model from conversation
      if (data.conversation?.model) {
        const match = models.find((m) => m.id === data.conversation.model);
        if (match) {
          setSelectedModel(match.id);
        }
      }
    } catch {
      toast.error('加载失败', '无法加载对话历史');
    }
  };

  const handleDeleteConversation = async (convId: string) => {
    const confirmed = await alertDialog.confirm({
      title: '删除对话',
      description: '确定要删除这个对话吗？此操作不可撤销。',
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
        toast.success('已删除', '对话已删除');
      },
    });
  };

  const getModelLabel = (model: ChatModelInfo) => {
    const name = model.alias || model.name;
    return model.isLoaded ? name : `${name} (未加载)`;
  };

  return (
    <div className="h-full flex bg-background text-foreground">
      {/* Conversation sidebar */}
      {showSidebar && (
        <div className="w-72 border-r flex flex-col bg-muted/20">
          <div className="flex items-center justify-between px-4 py-3 border-b">
            <h2 className="font-semibold text-sm">对话历史</h2>
            <Button variant="ghost" size="icon" onClick={() => setShowSidebar(false)}>
              <X className="w-4 h-4" />
            </Button>
          </div>
          <div className="flex-1 overflow-y-auto">
            {conversations.length === 0 ? (
              <p className="text-sm text-muted-foreground p-4 text-center">暂无对话历史</p>
            ) : (
              conversations.map((conv) => (
                <div
                  key={conv.id}
                  className={`flex items-center gap-2 px-4 py-2.5 cursor-pointer hover:bg-accent/50 transition-colors group ${
                    activeConvId === conv.id ? 'bg-accent' : ''
                  }`}
                  onClick={() => handleLoadConversation(conv.id)}
                >
                  <MessageSquare className="w-4 h-4 shrink-0 text-muted-foreground" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm truncate">{conv.title || '新对话'}</p>
                    <p className="text-xs text-muted-foreground">{conv.model}</p>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="opacity-0 group-hover:opacity-100 shrink-0 h-6 w-6"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteConversation(conv.id);
                    }}
                  >
                    <Trash2 className="w-3 h-3" />
                  </Button>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {/* Main chat area */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
          <div className="flex items-center gap-3">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setShowSidebar(!showSidebar)}
              title="对话历史"
            >
              <History className="w-5 h-5" />
            </Button>
            <MessageSquare className="w-5 h-5 text-primary" />
            <h1 className="text-lg font-semibold">AI 聊天</h1>
          </div>

          <div className="flex items-center gap-2">
            {/* Model selector */}
            <Select value={selectedModel} onValueChange={setSelectedModel}>
              <SelectTrigger className="w-[240px] px-3 py-1.5 border rounded-md bg-background text-sm">
                <SelectValue placeholder={modelsLoading ? '加载中...' : '选择模型'} />
              </SelectTrigger>
              <SelectContent>
                {models.map((model) => (
                  <SelectItem
                    key={model.id}
                    value={model.id}
                  >
                    <span className="flex items-center gap-2">
                      <CircleDot
                        className={`w-3 h-3 ${
                          model.isLoaded ? 'text-green-500' : 'text-muted-foreground'
                        }`}
                      />
                      {getModelLabel(model)}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Button
              variant="ghost"
              size="icon"
              onClick={handleNewChat}
              title="新建对话"
            >
              <Plus className="w-5 h-5" />
            </Button>

            {messages.length > 0 && (
              <Button
                variant="ghost"
                size="icon"
                onClick={async () => {
                  const confirmed = await alertDialog.confirm({
                    title: '清空对话',
                    description: '确定要清空当前对话吗？',
                    variant: 'destructive',
                  });
                  if (confirmed) {
                    setMessages([]);
                    setCurrentResponse('');
                  }
                }}
                title="清空对话"
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
            <span>正在加载模型 {selectedModelInfo?.name ?? selectedModel}，请稍候...</span>
          </div>
        )}

        {/* Message list */}
        <div className="flex-1 overflow-y-auto">
          {messages.length === 0 && !currentResponse ? (
            <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
              <MessageSquare className="w-16 h-16 mb-4 opacity-50" />
              <p className="text-lg mb-2">开始对话</p>
              <p className="text-sm">选择一个模型，然后输入消息开始聊天</p>
              {selectedModelInfo && !selectedModelInfo.isLoaded && (
                <p className="text-xs mt-2 text-yellow-500">
                  模型未加载，发送消息时将自动加载
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
          placeholder={
            !selectedModel
              ? '请先选择一个模型'
              : modelLoading
                ? '模型加载中...'
                : '输入消息... (按 Enter 发送)'
          }
        />
      </div>
    </div>
  );
}
