import { type ButtonHTMLAttributes, forwardRef } from 'react';
import { cn } from '@/lib/utils';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' | 'icon-button';
  size?: 'default' | 'sm' | 'icon' | 'xs';
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = 'default', size = 'default', ...props }, ref) => {
    return (
      <button
        className={cn(
          // Base styles
          'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-all duration-200 ease-in-out',
          // Enhanced focus styles - dual focus ring (ring + border)
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:border focus-visible:border-ring/50',
          // Disabled styles
          'disabled:pointer-events-none disabled:opacity-50 disabled:cursor-not-allowed',
          // Active styles with shadow
          'active:scale-95 active:shadow-sm',
          // Shadow for depth
          'shadow-sm hover:shadow-md transition-shadow',
          {
            // Default variant
            'bg-primary text-primary-foreground border border-primary/20 hover:bg-primary/90 hover:border-primary/30 active:bg-primary/80': variant === 'default',

            // Destructive variant
            'bg-destructive text-destructive-foreground border border-destructive/20 hover:bg-destructive/90 hover:border-destructive/30 active:bg-destructive/80': variant === 'destructive',

            // Outline variant (2px border)
            'border-2 border-input bg-background hover:bg-accent hover:text-accent-foreground hover:border-border/70 active:bg-accent/80': variant === 'outline',

            // Secondary variant
            'bg-secondary text-secondary-foreground border border-secondary/50 hover:bg-secondary/80 hover:border-secondary/60 active:bg-secondary/70': variant === 'secondary',

            // Ghost variant
            'hover:bg-accent/80 hover:text-accent-foreground active:bg-accent/60': variant === 'ghost',

            // Link variant
            'text-primary underline-offset-4 hover:underline active:text-primary/80': variant === 'link',

            // Icon button variant (2px border)
            'border-2 border-border/50 bg-transparent hover:bg-accent/80 hover:border-border hover:shadow-sm active:bg-accent/70': variant === 'icon-button',


          },
          {
            'h-10 px-4 py-2': size === 'default',
            'h-9 rounded-md px-3 text-xs': size === 'sm',
            'h-10 w-10': size === 'icon',
            'h-7 px-2 text-xs': size === 'xs',
          },
          className
        )}
        ref={ref}
        {...props}
      />
    );
  }
);

Button.displayName = 'Button';

export { Button };
