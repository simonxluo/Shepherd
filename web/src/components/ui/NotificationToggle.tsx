import { Bell, BellOff, ChevronDown, Check } from 'lucide-react';
import { useUserStore } from '@/stores/userStore';
import { useTranslation } from 'react-i18next';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';

export function NotificationToggle() {
  const { t } = useTranslation();
  const { settings, updateSettings } = useUserStore();

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
          aria-label={settings.notifications ? t('user.notificationsOn') : t('user.notificationsOff')}
          title={settings.notifications ? t('user.notificationsOn') : t('user.notificationsOff')}
        >
          {settings.notifications ? <Bell size={16} /> : <BellOff size={16} />}
          <ChevronDown
            size={12}
            className="transition-transform duration-200 text-muted-foreground"
          />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-36">
        <DropdownMenuItem
          onClick={() => updateSettings({ notifications: true })}
          className={cn(
            'flex items-center justify-between gap-2 text-xs',
            settings.notifications && 'font-medium'
          )}
        >
          <span className="flex items-center gap-2">
            <Bell size={12} className={cn(
              'shrink-0',
              settings.notifications ? 'text-primary' : 'text-foreground'
            )} />
            {t('user.notificationsOn')}
          </span>
          {settings.notifications && (
            <Check size={12} className="text-primary shrink-0" />
          )}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => updateSettings({ notifications: false })}
          className={cn(
            'flex items-center justify-between gap-2 text-xs',
            !settings.notifications && 'font-medium'
          )}
        >
          <span className="flex items-center gap-2">
            <BellOff size={12} className={cn(
              'shrink-0',
              !settings.notifications ? 'text-primary' : 'text-foreground'
            )} />
            {t('user.notificationsOff')}
          </span>
          {!settings.notifications && (
            <Check size={12} className="text-primary shrink-0" />
          )}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
