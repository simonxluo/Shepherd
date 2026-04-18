import { Sun, Moon, Monitor, ChevronDown } from 'lucide-react';
import { useState, useRef, useEffect } from 'react';
import { useUIStore, type Theme } from '@/stores/uiStore';
import { cn } from '@/lib/utils';

interface ThemeOption {
  value: Theme;
  label: string;
  icon: typeof Sun;
}

const themeOptions: ThemeOption[] = [
  { value: 'light', label: '浅色模式', icon: Sun },
  { value: 'dark', label: '深色模式', icon: Moon },
  { value: 'system', label: '跟随系统', icon: Monitor },
];

export function ThemeToggle() {
  const { theme, setTheme } = useUIStore();
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const currentTheme = themeOptions.find((option) => option.value === theme);
  const CurrentIcon = currentTheme?.icon || Monitor;

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger button */}
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex items-center gap-1.5 rounded-lg px-2.5 py-1.5',
          'transition-all duration-200',
          'border border-border/40 hover:border-border/60',
          'bg-background/50 hover:bg-background/80',
          'hover:bg-accent',
          'focus:outline-none focus:ring-2 focus:ring-ring focus:border-primary/50'
        )}
        aria-label={`选择主题（当前：${currentTheme?.label}）`}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        title={currentTheme?.label}
      >
        <CurrentIcon size={16} />
        <ChevronDown
          size={12}
          className={cn(
            'transition-transform duration-200 text-muted-foreground',
            isOpen && 'rotate-180'
          )}
        />
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div
          className={cn(
            'absolute right-0 top-full z-50 mt-1 w-32',
            'rounded-lg border bg-popover shadow-md overflow-hidden',
            'animate-in fade-in-0 zoom-in-95'
          )}
          role="listbox"
          aria-label="主题选项"
        >
          <div className="py-0.5">
            {themeOptions.map((option) => {
              const Icon = option.icon;
              const isSelected = option.value === theme;

              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => {
                    setTheme(option.value);
                    setIsOpen(false);
                  }}
                  className={cn(
                    'flex w-full items-center gap-2 px-3 py-1.5 text-xs',
                    'transition-colors duration-150',
                    'hover:bg-accent hover:text-accent-foreground',
                    isSelected && 'bg-accent',
                    'focus:outline-none focus:bg-accent'
                  )}
                  role="option"
                  aria-selected={isSelected}
                >
                  <Icon size={12} className={cn(
                    'shrink-0',
                    isSelected ? 'text-primary' : 'text-foreground'
                  )} />
                  <span className={cn(
                    'truncate',
                    isSelected && 'font-medium'
                  )}>
                    {option.label}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
