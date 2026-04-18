import { useState, useRef, useEffect } from 'react';
import { MessageSquare, Plus, Trash2 } from 'lucide-react';
import { ChatMessage } from '@/features/chat/components/ChatMessage';
import { ChatInput } from '@/features/chat/components/ChatInput';
import { useStreamingChat, getLoadedModels } from '@/features/chat/hooks';
import type { ChatMessage as ChatMessageType } from '@/features/chat';
import { useToast } from '@/hooks/useToast';
import { useAlertDialog } from '@/providers/AlertDialog';

/**
 * Chat page
 */
export function ChatPage() {
  const toast = useToast();
  const alertDialog = useAlertDialog();

  const [messages, setMessages] = useState<ChatMessageType[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [currentResponse, setCurrentResponse] = useState('');
  const [models, setModels] = useState<string[]>([]);
  const [selectedModel, setSelectedModel] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const [isInitialized, setIsInitialized] = useState(false);

  const streamingChat = useStreamingChat();

  // Load available models
  useEffect(() => {
    getLoadedModels().then(setModels).catch(console.error);
  }, []);

  // Auto-scroll to bottom — only on message changes, skip initialization
  useEffect(() => {
    // Skip initialization phase
    if (!isInitialized) {
      setIsInitialized(true);
      return;
    }

    // Scroll only when there are messages or streaming
    if (messages.length > 0 || currentResponse) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, currentResponse, isInitialized]);

  const handleSend = (content: string) => {
    if (!selectedModel) {
      toast.warning('请先选择模型', '请从下拉列表中选择一个已加载的模型');
      return;
    }

    const userMessage: ChatMessageType = {
      role: 'user',
      content,
      timestamp: Date.now(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setIsStreaming(true);
    setCurrentResponse('');

    streamingChat.mutate(
      {
        model: selectedModel,
        messages: [...messages, userMessage],
        temperature: 0.7,
        topP: 0.95,
        onChunk: (chunk) => {
          setCurrentResponse((prev) => prev + chunk);
        },
        onComplete: (message) => {
          const assistantMessage: ChatMessageType = {
            role: 'assistant',
            content: message,
            timestamp: Date.now(),
          };
          setMessages((prev) => [...prev, assistantMessage]);
          setCurrentResponse('');
          setIsStreaming(false);
        },
        onError: (error) => {
          console.error('Chat error:', error);
          setCurrentResponse('');
          setIsStreaming(false);
        },
      },
      {
        onError: (error) => {
          console.error('Mutation error:', error);
          setIsStreaming(false);
        },
      }
    );
  };

  const handleStop = () => {
    setIsStreaming(false);
    setCurrentResponse('');
  };

  const handleNewChat = async () => {
    if (messages.length > 0) {
      const confirmed = await alertDialog.confirm({
        title: '新建对话',
        description: '确定要开始新对话吗？当前对话将被清空。',
      });
      if (!confirmed) return;
    }
    setMessages([]);
    setCurrentResponse('');
  };

  const handleClearHistory = async () => {
    const confirmed = await alertDialog.confirm({
      title: '清空对话',
      description: '确定要清空对话历史吗？此操作不可撤销。',
      variant: 'destructive',
    });
    if (confirmed) {
      setMessages([]);
      setCurrentResponse('');
    }
  };

  return (
    <div className="h-full flex flex-col bg-background text-foreground">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b bg-muted/30">
        <div className="flex items-center gap-3">
          <MessageSquare className="w-5 h-5 text-primary" />
          <h1 className="text-lg font-semibold">AI 聊天</h1>
        </div>

        <div className="flex items-center gap-2">
          {/* Model selector */}
          <select
            value={selectedModel}
            onChange={(e) => setSelectedModel(e.target.value)}
            className="px-3 py-1.5 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent"
          >
            <option value="">选择模型</option>
            {models.map((model) => (
              <option key={model} value={model}>
                {model}
              </option>
            ))}
          </select>

          <button
            onClick={handleNewChat}
            className="p-2 text-muted-foreground hover:bg-accent rounded transition-colors"
            title="新建对话"
          >
            <Plus className="w-5 h-5" />
          </button>

          {messages.length > 0 && (
            <button
              onClick={handleClearHistory}
              className="p-2 text-muted-foreground hover:bg-accent rounded transition-colors"
              title="清空对话"
            >
              <Trash2 className="w-5 h-5" />
            </button>
          )}
        </div>
      </div>

      {/* Message list */}
      <div
        ref={messagesContainerRef}
        className="flex-1 overflow-y-auto"
      >
        {messages.length === 0 && !currentResponse ? (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
            <MessageSquare className="w-16 h-16 mb-4 opacity-50" />
            <p className="text-lg mb-2">开始对话</p>
            <p className="text-sm">选择一个模型，然后输入消息开始聊天</p>
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
                  timestamp: Date.now(),
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
        disabled={!selectedModel || isStreaming}
        isStreaming={isStreaming}
        placeholder={selectedModel ? '输入消息... (按 Enter 发送)' : '请先选择一个模型'}
      />
    </div>
  );
}
