import { useTranslation } from 'react-i18next';
import { cn } from '@/lib/utils';
import type { TTSPlugin } from '../types';

interface VerticalTabBarProps {
  plugins: TTSPlugin[];
  activeId: string;
  onSelect: (id: string) => void;
}

export function VerticalTabBar({ plugins, activeId, onSelect }: VerticalTabBarProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col w-[120px] border-l bg-muted/30">
      {plugins.map((plugin) => {
        const isActive = plugin.id === activeId;
        return (
          <button
            key={plugin.id}
            onClick={() => onSelect(plugin.id)}
            className={cn(
              'relative text-left px-3 py-3 text-sm transition-colors',
              'hover:bg-accent/50',
              isActive
                ? 'bg-primary/10 font-medium text-foreground'
                : 'text-muted-foreground'
            )}
          >
            {/* Left indicator bar */}
            {isActive && (
              <span className="absolute left-0 top-1/2 -translate-y-1/2 w-0.5 h-6 bg-primary rounded-r" />
            )}
            <span className="block truncate">
              {t(plugin.labelKey, plugin.labelFallback)}
            </span>
          </button>
        );
      })}
    </div>
  );
}
