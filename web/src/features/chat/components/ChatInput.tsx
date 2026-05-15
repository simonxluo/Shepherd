import { useState, useRef, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Send, Square, ImageIcon, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

const MAX_HEIGHT = 320;
const MAX_IMAGE_SIZE = 2048;
const JPEG_QUALITY = 0.8;

interface ChatInputProps {
  onSend: (content: string, images?: string[]) => void;
  onStop?: () => void;
  disabled?: boolean;
  isStreaming?: boolean;
  placeholder?: string;
  supportsVision?: boolean;
}

async function compressImage(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      const img = new Image();
      img.onload = () => {
        let { width, height } = img;
        if (width > MAX_IMAGE_SIZE || height > MAX_IMAGE_SIZE) {
          const ratio = Math.min(MAX_IMAGE_SIZE / width, MAX_IMAGE_SIZE / height);
          width = Math.round(width * ratio);
          height = Math.round(height * ratio);
        }

        const canvas = document.createElement('canvas');
        canvas.width = width;
        canvas.height = height;
        const ctx = canvas.getContext('2d')!;
        ctx.drawImage(img, 0, 0, width, height);
        resolve(canvas.toDataURL('image/jpeg', JPEG_QUALITY));
      };
      img.onerror = reject;
      img.src = e.target?.result as string;
    };
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

export function ChatInput({
  onSend,
  onStop,
  disabled = false,
  isStreaming = false,
  placeholder,
  supportsVision = false,
}: ChatInputProps) {
  const { t } = useTranslation();
  const [content, setContent] = useState('');
  const [images, setImages] = useState<string[]>([]);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
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
    if ((!trimmed && images.length === 0) || disabled || isStreaming) return;

    onSend(trimmed, images.length > 0 ? images : undefined);
    setContent('');
    setImages([]);

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

  const handleImageSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;

    for (const file of Array.from(files)) {
      if (!file.type.startsWith('image/')) continue;
      try {
        const dataUrl = await compressImage(file);
        setImages((prev) => [...prev, dataUrl]);
      } catch {
        // Skip failed images
      }
    }

    // Reset input so same file can be re-selected
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const removeImage = (index: number) => {
    setImages((prev) => prev.filter((_, i) => i !== index));
  };

  const canSend = (content.trim().length > 0 || images.length > 0) && !disabled;

  return (
    <form onSubmit={handleSubmit} className="border-t p-4 bg-background">
      {/* Image previews */}
      {images.length > 0 && (
        <div className="flex gap-2 mb-3 flex-wrap">
          {images.map((img, i) => (
            <div key={i} className="relative group">
              <img
                src={img}
                alt={t('chat.message.uploadedImageAlt', { index: i + 1 })}
                className="w-16 h-16 object-cover rounded-lg border"
              />
              <button
                type="button"
                onClick={() => removeImage(i)}
                className="absolute -top-1.5 -right-1.5 w-5 h-5 bg-destructive text-destructive-foreground rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <X className="w-3 h-3" />
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="flex items-end gap-3">
        {/* Image upload button */}
        {supportsVision && !isStreaming && (
          <>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              multiple
              className="hidden"
              onChange={handleImageSelect}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="rounded-full shrink-0"
              title={t('chat.message.uploadImage')}
              onClick={() => fileInputRef.current?.click()}
            >
              <ImageIcon className="w-5 h-5" />
            </Button>
          </>
        )}

        <div className="flex-1 relative">
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder || t('chat.placeholderInput')}
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
            title={t('chat.message.stopGeneration')}
            className="rounded-full shrink-0"
          >
            <Square className="w-4 h-4" />
          </Button>
        ) : (
          <Button
            type="submit"
            disabled={!canSend}
            variant="default"
            size="icon"
            title={t('chat.message.sendMessage')}
            className="rounded-full shrink-0"
          >
            <Send className="w-4 h-4" />
          </Button>
        )}
      </div>
    </form>
  );
}
