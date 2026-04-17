import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface LoadingStateProps {
  /** Optional hint text displayed below the spinner */
  text?: string;
  /** Additional className for the outer container */
  className?: string;
}

/**
 * A reusable loading indicator with an optional text label.
 * Uses lucide-react Loader2 with animate-spin.
 */
export function LoadingState({ text, className }: LoadingStateProps) {
  return (
    <div className={cn('flex items-center justify-center py-12', className)}>
      <div className="text-center">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600 mx-auto mb-2" />
        {text && <p className="text-muted-foreground">{text}</p>}
      </div>
    </div>
  );
}
