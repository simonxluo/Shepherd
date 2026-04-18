import { AlertTriangle, X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useAlertDialog } from '@/providers/AlertDialog';
import { Button } from '@/components/ui/button';

export function AlertDialog() {
  const { state, resolve, close } = useAlertDialog();

  if (!state) return null;

  const isDestructive = state.variant === 'destructive';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop overlay */}
      <div
        className="fixed inset-0 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200"
        onClick={close}
      />

      {/* Dialog content */}
      <div
        className={cn(
          'relative bg-card text-card-foreground rounded-xl shadow-2xl border border-border',
          'max-w-md w-full',
          'animate-in fade-in-0 zoom-in-95 duration-200',
          'p-6'
        )}
      >
        {/* Close button */}
        <button
          onClick={close}
          className="absolute top-4 right-4 text-muted-foreground hover:text-foreground transition-colors"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Icon */}
        <div className={cn('flex items-center justify-center w-12 h-12 rounded-full mb-4', isDestructive ? 'bg-red-100 dark:bg-red-900/30' : 'bg-blue-100 dark:bg-blue-900/30')}>
          <AlertTriangle className={cn('w-6 h-6', isDestructive ? 'text-red-600 dark:text-red-400' : 'text-blue-600 dark:text-blue-400')} />
        </div>

        {/* Title */}
        <h2 className={cn('text-lg font-semibold mb-2', isDestructive ? 'text-destructive' : 'text-foreground')}>
          {state.title}
        </h2>

        {/* Description */}
        <p className="text-sm text-muted-foreground mb-6">
          {state.description}
        </p>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3">
          <Button
            variant="outline"
            onClick={close}
            className="min-w-[80px]"
          >
            {state.cancelText || '取消'}
          </Button>
          <Button
            variant={isDestructive ? 'destructive' : 'default'}
            onClick={() => resolve(true)}
            className="min-w-[80px]"
          >
            {state.confirmText || '确认'}
          </Button>
        </div>
      </div>
    </div>
  );
}

export { AlertDialogProvider, type AlertDialogOptions } from '@/providers/AlertDialog';
