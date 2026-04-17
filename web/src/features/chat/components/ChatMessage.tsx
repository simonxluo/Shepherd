import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import rehypeRaw from 'rehype-raw';
import { User, Bot, Copy, Check } from 'lucide-react';
import { useState } from 'react';
import { cn } from '@/lib/utils';
import type { ChatMessage as ChatMessageType } from '@/features/chat';

interface ChatMessageProps {
  message: ChatMessageType;
  isStreaming?: boolean;
}

export function ChatMessage({ message, isStreaming }: ChatMessageProps) {
  const [copied, setCopied] = useState(false);
  const isUser = message.role === 'user';

  const handleCopy = async () => {
    await navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div
      className={cn(
        'flex gap-3 p-4',
        isUser ? 'bg-muted/30' : 'bg-muted/10'
      )}
    >
      {/* 头像 */}
      <div
        className={cn(
          'flex shrink-0 w-8 h-8 rounded-full items-center justify-center',
          isUser
            ? 'bg-primary text-primary-foreground'
            : 'bg-gradient-to-br from-purple-500 to-pink-500 text-white'
        )}
      >
        {isUser ? <User className="w-5 h-5" /> : <Bot className="w-5 h-5" />}
      </div>

      {/* 内容 */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-medium">
            {isUser ? '你' : 'AI 助手'}
          </span>
          {message.timestamp && (
            <span className="text-xs text-muted-foreground">
              {new Date(message.timestamp).toLocaleTimeString('zh-CN')}
            </span>
          )}
        </div>

        <div className="relative group">
          <div
            className={cn(
              'prose dark:prose-invert max-w-none',
              isStreaming && 'after:content-["▌"] after:animate-pulse'
            )}
          >
            {isUser ? (
              <p className="whitespace-pre-wrap">{message.content}</p>
            ) : (
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeHighlight, rehypeRaw]}
                components={{
                  code: ({ className, children, ...props }: any) => {
                    return !props.inline ? (
                      <div className="relative">
                        <code className={className} {...props}>
                          {children}
                        </code>
                      </div>
                    ) : (
                      <code
                        className="px-1.5 py-0.5 bg-muted rounded text-sm font-mono"
                        {...props}
                      >
                        {children}
                      </code>
                    );
                  },
                  pre: ({ children }) => (
                    <pre className="overflow-x-auto p-4 bg-muted rounded-lg">
                      {children}
                    </pre>
                  ),
                }}
              >
                {message.content}
              </ReactMarkdown>
            )}
          </div>

          {/* 复制按钮 */}
          {!isUser && message.content && (
            <button
              onClick={handleCopy}
              className="absolute top-0 right-0 p-1.5 opacity-0 group-hover:opacity-100 transition-opacity rounded bg-muted hover:bg-accent"
              title="复制"
            >
              {copied ? (
                <Check className="w-4 h-4 text-green-600" />
              ) : (
                <Copy className="w-4 h-4" />
              )}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
