import { useState, useRef, useEffect } from 'react';
import { X, Play, Square, Send, MessageSquare, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface TestModelDialogProps {
  isOpen: boolean;
  onClose: () => void;
  modelId: string;
  modelName: string;
  isModelLoaded: boolean;
  onLoadModel: () => void;
}

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  isStreaming?: boolean;
}

export function TestModelDialog({
  isOpen,
  onClose,
  modelId,
  modelName,
  isModelLoaded,
  onLoadModel,
}: TestModelDialogProps) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  if (!isOpen) return null;

  const handleSend = async () => {
    if (!input.trim() || isLoading) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setIsLoading(true);
    setError('');

    const assistantMessage: Message = {
      id: (Date.now() + 1).toString(),
      role: 'assistant',
      content: '',
      timestamp: new Date(),
      isStreaming: true,
    };

    setMessages((prev) => [...prev, assistantMessage]);

    try {
      const response = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          model: modelId,
          messages: [{ role: 'user', content: userMessage.content }],
          stream: true,
          max_tokens: 1024,
          temperature: 0.7,
        }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();

      if (!reader) {
        throw new Error('No response body');
      }

      let fullContent = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value);
        const lines = chunk.split('\n');

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6);
            if (data === '[DONE]') continue;

            try {
              const parsed = JSON.parse(data);
              const content = parsed.choices?.[0]?.delta?.content || '';
              if (content) {
                fullContent += content;
                setMessages((prev) =>
                  prev.map((msg) =>
                    msg.id === assistantMessage.id
                      ? { ...msg, content: fullContent }
                      : msg
                  )
                );
              }
            } catch (e) {
              // Ignore parse errors
            }
          }
        }
      }

      setMessages((prev) =>
        prev.map((msg) =>
          msg.id === assistantMessage.id
            ? { ...msg, isStreaming: false }
            : msg
        )
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : '请求失败');
      setMessages((prev) =>
        prev.filter((msg) => msg.id !== assistantMessage.id)
      );
    } finally {
      setIsLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const clearMessages = () => {
    setMessages([]);
    setError('');
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-card rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-border">
          <div className="flex items-center gap-2">
            <MessageSquare className="w-5 h-5 text-blue-600" />
            <div>
              <h2 className="text-lg font-semibold text-foreground">
                测试模型
              </h2>
              <p className="text-sm text-muted-foreground">
                {modelName}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {messages.length > 0 && (
              <Button
                variant="outline"
                size="sm"
                onClick={clearMessages}
              >
                清空对话
              </Button>
            )}
            <button
              onClick={onClose}
              className="p-1 text-muted-foreground hover:text-foreground"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Model not loaded warning */}
        {!isModelLoaded && (
          <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border-b border-yellow-200 dark:border-yellow-800">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-yellow-800 dark:text-yellow-200">
                <Loader2 className="w-4 h-4" />
                <span>模型未加载，请先加载模型后再测试</span>
              </div>
              <Button
                size="sm"
                onClick={onLoadModel}
                className="bg-yellow-600 hover:bg-yellow-700 text-white"
              >
                <Play className="w-4 h-4 mr-1" />
                加载模型
              </Button>
            </div>
          </div>
        )}

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 min-h-[300px] max-h-[500px]">
          {messages.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
              <MessageSquare className="w-12 h-12 mb-4 opacity-50" />
              <p>开始测试模型</p>
              <p className="text-sm mt-1">输入消息并发送</p>
            </div>
          ) : (
            messages.map((message) => (
              <div
                key={message.id}
                className={cn(
                  'flex gap-3',
                  message.role === 'user' ? 'justify-end' : 'justify-start'
                )}
              >
                <div
                  className={cn(
                    'max-w-[80%] rounded-lg p-3',
                    message.role === 'user'
                      ? 'bg-blue-600 text-white'
                      : 'bg-muted text-foreground'
                  )}
                >
                  <div className="text-sm whitespace-pre-wrap">
                    {message.content || (message.isStreaming ? '<span class="animate-pulse">...</span>' : '')}
                  </div>
                  <div
                    className={cn(
                      'text-xs mt-1',
                      message.role === 'user'
                        ? 'text-blue-200'
                        : 'text-muted-foreground'
                    )}
                  >
                    {message.timestamp.toLocaleTimeString()}
                  </div>
                </div>
              </div>
            ))
          )}
          {error && (
            <div className="p-3 bg-red-100 dark:bg-red-900/20 text-red-700 dark:text-red-300 rounded-lg">
              错误: {error}
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* Input */}
        <div className="p-4 border-t border-border">
          <div className="flex gap-2">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={isModelLoaded ? '输入消息...' : '请先加载模型'}
              disabled={!isModelLoaded || isLoading}
              className="flex-1 px-3 py-2 border border-border rounded-md bg-input text-foreground placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
              rows={2}
            />
            <Button
              onClick={handleSend}
              disabled={!isModelLoaded || isLoading || !input.trim()}
              className="self-end"
            >
              {isLoading ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Send className="w-4 h-4" />
              )}
            </Button>
          </div>
          <p className="mt-2 text-xs text-muted-foreground">
            按 Enter 发送，Shift + Enter 换行
          </p>
        </div>
      </div>
    </div>
  );
}
