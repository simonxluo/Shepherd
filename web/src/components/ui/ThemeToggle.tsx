import { Sun, Moon, Monitor, ChevronDown } from 'lucide-react';
import { useUIStore, type Theme } from '@/stores/uiStore';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';

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

  const currentTheme = themeOptions.find((option) => option.value === theme);
  const CurrentIcon = currentTheme?.icon || Monitor;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            'flex items-center gap-1.5 rounded-lg px-2.5 py-1.5',
            'transition-all duration-200',
            'border border-border/60 hover:border-border/80',
            'bg-muted/30 hover:bg-muted/50',
            'focus:outline-none focus:ring-2 focus:ring-ring focus:border-primary/50'
          )}
          aria-label={`选择主题（当前：${currentTheme?.label}）`}
          title={currentTheme?.label}
        >
          <CurrentIcon size={16} />
          <ChevronDown
            size={12}
            className="transition-transform duration-200 text-muted-foreground"
          />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-32">
        {themeOptions.map((option) => {
          const Icon = option.icon;
          const isSelected = option.value === theme;

          return (
            <DropdownMenuItem
              key={option.value}
              onClick={() => setTheme(option.value)}
              className={cn(
                'flex items-center gap-2 text-xs',
                isSelected && 'font-medium'
              )}
            >
              <Icon size={12} className={cn(
                'shrink-0',
                isSelected ? 'text-primary' : 'text-foreground'
              )} />
              <span className="truncate">
                {option.label}
              </span>
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
