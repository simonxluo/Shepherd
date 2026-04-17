import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

export interface EmptyStateProps {
  /** Optional icon element rendered above the title */
  icon?: ReactNode;
  /** Title text */
  title: string;
  /** Optional description text */
  description?: string;
  /** Optional action element (e.g. a Button) */
  action?: ReactNode;
  /** Additional className for the outer container */
  className?: string;
}

/**
 * A reusable empty state placeholder with icon, title, description,
 * and an optional call-to-action.
 */
export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-12 text-muted-foreground', className)}>
      {icon}
      <p className="text-lg mb-2">{title}</p>
      {description && <p className="text-sm mb-4">{description}</p>}
      {action}
    </div>
  );
}
