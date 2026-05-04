import { useState, useRef, useCallback, useEffect } from 'react';
import { Send, Square } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

const MAX_HEIGHT = 320;

interface ChatInputProps {
  onSend: (content: string) => void;
  onStop?: () => void;
  disabled?: boolean;
  isStreaming?: boolean;
  placeholder?: string;
}

export function ChatInput({
  onSend,
  onStop,
  disabled = false,
  isStreaming = false,
  placeholder = '输入消息...',
}: ChatInputProps) {
  const [content, setContent] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [needsScroll, setNeedsScroll] = useState(false);

  const adjustHeight = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;

    el.style.height = 'auto';
    const scrollH = el.scrollHeight;
    const clamped = Math.min(scrollH, MAX_HEIGHT);
    el.style.height = `${clamped}px`;
    setNeedsScroll(scrollH > MAX_HEIGHT);
  }, []);

  useEffect(() => {
    adjustHeight();
  }, [content, adjustHeight]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = content.trim();
    if (!trimmed || disabled || isStreaming) return;

    onSend(trimmed);
    setContent('');

    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      setNeedsScroll(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="border-t p-4 bg-background">
      <div className="flex items-end gap-3">
        <div className="flex-1 relative">
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={disabled}
            className={cn(
              'w-full px-4 py-3 pr-14 bg-background border border-input rounded-2xl resize-none',
              'text-sm leading-relaxed outline-none',
              'transition-colors placeholder:text-muted-foreground',
              'focus:ring-2 focus:ring-ring focus:border-transparent',
              needsScroll ? 'overflow-y-auto' : 'overflow-y-hidden',
              disabled && 'opacity-50 cursor-not-allowed'
            )}
            rows={1}
            style={{ minHeight: '48px', maxHeight: `${MAX_HEIGHT}px` }}
          />

          {content.length > 0 && (
            <div className="absolute bottom-2 right-2 text-xs text-muted-foreground/60 pointer-events-none">
              {content.length.toLocaleString()}
            </div>
          )}
        </div>

        {isStreaming ? (
          <Button
            type="button"
            onClick={onStop}
            variant="destructive"
            size="icon"
            title="停止生成"
            className="rounded-full shrink-0"
          >
            <Square className="w-4 h-4" />
          </Button>
        ) : (
          <Button
            type="submit"
            disabled={!content.trim() || disabled}
            variant="default"
            size="icon"
            title="发送 (Enter)"
            className="rounded-full shrink-0"
          >
            <Send className="w-4 h-4" />
          </Button>
        )}
      </div>
    </form>
  );
}
