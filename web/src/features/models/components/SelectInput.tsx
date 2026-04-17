import { ChevronDown } from 'lucide-react';
import { cn } from '@/lib/utils';

interface SelectInputProps {
  value: string | number | undefined;
  onChange: (e: React.ChangeEvent<HTMLSelectElement>) => void;
  disabled?: boolean;
  children: React.ReactNode;
  className?: string;
}

export function SelectInput({
  value,
  onChange,
  disabled,
  children,
  className = '',
}: SelectInputProps) {
  return (
    <div className="relative">
      <select
        value={value ?? ''}
        onChange={onChange}
        disabled={disabled}
        className={cn(
          "w-full px-2 py-1.5 pr-8 text-sm",
          "border-2 border-border",
          "rounded-md bg-input",
          "text-foreground",
          "appearance-none cursor-pointer",
          "hover:border-blue-400 dark:hover:border-blue-500",
          "focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "transition-colors",
          className
        )}
      >
        {children}
      </select>
      <ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
    </div>
  );
}
