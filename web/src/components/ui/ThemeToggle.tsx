import { Sun, Moon, Monitor, ChevronDown } from 'lucide-react';
import { useTranslation } from 'react-i18next';
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
  labelKey: string;
  icon: typeof Sun;
}

const themeOptions: ThemeOption[] = [
  { value: 'light', labelKey: 'theme.light', icon: Sun },
  { value: 'dark', labelKey: 'theme.dark', icon: Moon },
  { value: 'system', labelKey: 'theme.system', icon: Monitor },
];

export function ThemeToggle() {
  const { t } = useTranslation();
  const { theme, setTheme } = useUIStore();

  const currentTheme = themeOptions.find((option) => option.value === theme);
  const CurrentIcon = currentTheme?.icon || Monitor;
  const currentLabel = currentTheme ? t(currentTheme.labelKey) : '';

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
          aria-label={currentLabel}
          title={currentLabel}
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
                {t(option.labelKey)}
              </span>
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
