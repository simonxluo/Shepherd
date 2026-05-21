import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import rehypeRaw from 'rehype-raw';
import { User, Bot, Copy, Check, RefreshCw, Clock, Zap } from 'lucide-react';
import { useState, useRef, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { cn } from '@/lib/utils';
import type { GenerationStats } from '@/lib/api/chat';

interface ChatMessageData {
  role: 'user' | 'assistant' | 'system';
  content: string;
  images?: string[];
  timestamp?: number;
  stats?: GenerationStats;
}

interface ChatMessageProps {
  message: ChatMessageData;
  isStreaming?: boolean;
  onRegenerate?: () => void;
}

function CodeBlockCopyButton({ code }: { code: string }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      onClick={handleCopy}
      className="flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors rounded hover:bg-accent"
      title={t('chat.message.copyCode')}
    >
      {copied ? (
        <>
          <Check className="w-3 h-3 text-green-600" />
          <span className="text-green-600">{t('chat.message.copied')}</span>
        </>
      ) : (
        <>
          <Copy className="w-3 h-3" />
          <span>{t('chat.message.copyCode')}</span>
        </>
      )}
    </button>
  );
}

function extractTextContent(children: React.ReactNode): string {
  if (typeof children === 'string') return children;
  if (Array.isArray(children)) return children.map(extractTextContent).join('');
  if (children && typeof children === 'object' && 'props' in children) {
    return extractTextContent((children as React.ReactElement<{ children?: React.ReactNode }>).props.children);
  }
  return '';
}

function StatsDisplay({ stats }: { stats: GenerationStats }) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-3 mt-2 text-xs text-muted-foreground">
      <span className="flex items-center gap-1">
        <Zap className="w-3 h-3" />
        {stats.tokensPerSecond.toFixed(1)} {t('chat.stats.tokensPerSec')}
      </span>
      <span className="flex items-center gap-1">
        <Clock className="w-3 h-3" />
        {(stats.durationMs / 1000).toFixed(1)}s
      </span>
      {stats.completionTokens > 0 && (
        <span>
          {stats.completionTokens} {t('chat.stats.tokens')}
        </span>
      )}
    </div>
  );
}

export function ChatMessage({ message, isStreaming, onRegenerate }: ChatMessageProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const isUser = message.role === 'user';
  const isSystem = message.role === 'system';

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(message.content);
    setCopied(true);
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setCopied(false), 2000);
  }, [message.content]);

  if (isSystem) {
    return (
      <div className="flex gap-3 p-4 bg-blue-500/5 border-b border-blue-500/10">
        <div className="flex shrink-0 w-8 h-8 rounded-full items-center justify-center bg-blue-500/20 text-blue-600 dark:text-blue-400">
          <Bot className="w-5 h-5" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="font-medium text-sm text-blue-600 dark:text-blue-400">
              {t('chat.message.system')}
            </span>
          </div>
          <p className="text-sm text-muted-foreground whitespace-pre-wrap">{message.content}</p>
        </div>
      </div>
    );
  }

  return (
    <div
      className={cn(
        'flex gap-3 p-4',
        isUser ? 'bg-muted/30' : 'bg-muted/10'
      )}
    >
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

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-medium">
            {isUser ? t('chat.message.you') : t('chat.message.aiAssistant')}
          </span>
          {message.timestamp && (
            <span className="text-xs text-muted-foreground">
              {new Date(message.timestamp).toLocaleTimeString()}
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
              <>
                {message.images && message.images.length > 0 && (
                  <div className="flex gap-2 mb-2 flex-wrap">
                    {message.images.map((img, i) => (
                      <img
                        key={i}
                        src={img}
                        alt={t('chat.message.imageAlt', { index: i + 1 })}
                        className="max-w-[200px] max-h-[200px] object-contain rounded-lg border cursor-pointer"
                        onClick={() => window.open(img, '_blank')}
                      />
                    ))}
                  </div>
                )}
                <p className="whitespace-pre-wrap">{message.content}</p>
              </>
            ) : (
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeHighlight, rehypeRaw]}
                components={{
                  code: ({ className, children, ...props }: React.HTMLAttributes<HTMLElement> & { inline?: boolean; node?: unknown }) => {
                    return !props.inline ? (
                      <code className={className} {...props}>
                        {children}
                      </code>
                    ) : (
                      <code
                        className="px-1.5 py-0.5 bg-muted rounded text-sm font-mono"
                        {...props}
                      >
                        {children}
                      </code>
                    );
                  },
                  pre: ({ children }) => {
                    // Extract language from code child className
                    let language = '';
                    let codeText = '';
                    if (children && typeof children === 'object' && 'props' in children) {
                      const codeEl = children as React.ReactElement<{ className?: string; children?: React.ReactNode }>;
                      const cls = codeEl.props.className ?? '';
                      const match = cls.match(/language-(\w+)/);
                      if (match) language = match[1];
                      codeText = extractTextContent(codeEl.props.children);
                    }

                    return (
                      <div className="relative rounded-lg overflow-hidden my-3">
                        <div className="flex items-center justify-between px-3 py-1.5 bg-muted/80 border-b border-border/50">
                          <span className="text-xs text-muted-foreground font-mono">
                            {language || 'code'}
                          </span>
                          <CodeBlockCopyButton code={codeText} />
                        </div>
                        <pre className="overflow-x-auto p-4 bg-muted rounded-b-lg !mt-0 !rounded-t-none">
                          {children}
                        </pre>
                      </div>
                    );
                  },
                }}
              >
                {message.content}
              </ReactMarkdown>
            )}
          </div>

          {/* Message action bar */}
          {message.content && !isStreaming && (
            <div className="flex items-center gap-1 mt-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
              <button
                onClick={handleCopy}
                className="flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors rounded hover:bg-accent"
                title={t('chat.message.copy')}
              >
                {copied ? (
                  <Check className="w-3.5 h-3.5 text-green-600" />
                ) : (
                  <Copy className="w-3.5 h-3.5" />
                )}
              </button>
              {!isUser && onRegenerate && (
                <button
                  onClick={onRegenerate}
                  className="flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors rounded hover:bg-accent"
                  title={t('chat.message.regenerate')}
                >
                  <RefreshCw className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          )}
        </div>

        {/* Generation stats */}
        {!isUser && message.stats && !isStreaming && (
          <StatsDisplay stats={message.stats} />
        )}
      </div>
    </div>
  );
}
