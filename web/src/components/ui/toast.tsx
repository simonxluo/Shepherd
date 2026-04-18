import { X } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { Toast } from '@/stores/toast';
import { ToastIcons, ToastStyles } from '@/stores/toast';

interface ToastProps extends Toast {
  onClose?: () => void;
}

export function Toast({ type, title, description, onClose }: ToastProps) {
  const IconComponent = ToastIcons[type];
  const styles = ToastStyles[type];

  return (
    <div
      className={cn(
        'flex items-start gap-3 p-4 rounded-lg shadow-lg border',
        'min-w-[300px] max-w-md',
        'animate-in slide-in-from-right-full fade-in duration-300',
        styles.container
      )}
    >
      {/* Icon */}
      <IconComponent className={cn('w-5 h-5 shrink-0 mt-0.5', styles.icon)} />

      {/* Content */}
      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold">{title}</p>
        {description && (
          <p className="text-sm opacity-90 mt-1">{description}</p>
        )}
      </div>

      {/* Close button */}
      {onClose && (
        <button
          onClick={onClose}
          className={cn(
            'shrink-0 opacity-70 hover:opacity-100',
            'transition-opacity',
            'rounded p-0.5 hover:bg-black/5 dark:hover:bg-white/5'
          )}
        >
          <X className="w-4 h-4" />
        </button>
      )}
    </div>
  );
}
