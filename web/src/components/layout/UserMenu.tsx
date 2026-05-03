import { useTranslation } from 'react-i18next';
import {
  User,
  Settings,
  LogOut,
  ChevronUp,
  Moon,
  Sun,
  Monitor,
  Bell,
  BellOff
} from 'lucide-react';
import { useUserStore } from '@/stores/userStore';
import { useUIStore } from '@/stores/uiStore';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from '@/components/ui/dropdown-menu';

interface UserMenuProps {
  sidebarOpen: boolean;
}

export function UserMenu({ sidebarOpen }: UserMenuProps) {
  const { t } = useTranslation();

  const {
    user,
    settings,
    logout,
    setShowProfileDialog,
    setShowSettingsDialog,
    updateSettings
  } = useUserStore();

  const { theme, setTheme } = useUIStore();

  const handleLogout = () => {
    logout();
  };

  const handleToggleTheme = () => {
    const themes: Array<'light' | 'dark' | 'system'> = ['light', 'dark', 'system'];
    const currentIndex = themes.indexOf(theme);
    const nextTheme = themes[(currentIndex + 1) % themes.length];
    setTheme(nextTheme);
  };

  const handleToggleNotifications = () => {
    updateSettings({ notifications: !settings.notifications });
  };

  const getDisplayName = () => {
    if (user?.displayName) return user.displayName;
    if (user?.username) return user.username;
    return t('user.guest');
  };

  const getAvatarUrl = () => {
    if (user?.avatar) return user.avatar;
    return null;
  };

  const getThemeLabel = () => {
    switch (theme) {
      case 'light': return t('theme.light');
      case 'dark': return t('theme.dark');
      default: return t('theme.system');
    }
  };

  const avatarContent = getAvatarUrl() ? (
    <img
      src={getAvatarUrl()!}
      alt={getDisplayName()}
      className="w-8 h-8 rounded-full object-cover"
    />
  ) : (
    <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
      <User className="w-4 h-4 text-primary" />
    </div>
  );

  // Collapsed state - avatar only
  if (!sidebarOpen) {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="w-10 h-10 rounded-full"
          >
            {avatarContent}
          </Button>
        </DropdownMenuTrigger>

        <DropdownMenuContent side="top" align="start" className="w-48">
          <DropdownMenuLabel>
            <p className="font-medium text-sm truncate">{getDisplayName()}</p>
            {user?.email && (
              <p className="text-xs text-muted-foreground truncate">{user.email}</p>
            )}
          </DropdownMenuLabel>
          <DropdownMenuSeparator />

          <DropdownMenuItem onClick={() => setShowProfileDialog(true)}>
            <User className="w-4 h-4" />
            {t('user.profile')}
          </DropdownMenuItem>

          <DropdownMenuItem onClick={() => setShowSettingsDialog(true)}>
            <Settings className="w-4 h-4" />
            {t('user.settings')}
          </DropdownMenuItem>

          <DropdownMenuSeparator />

          <DropdownMenuItem onClick={handleToggleTheme}>
            {theme === 'light' && <Sun className="w-4 h-4" />}
            {theme === 'dark' && <Moon className="w-4 h-4" />}
            {theme === 'system' && <Monitor className="w-4 h-4" />}
            {t('user.theme')}: {getThemeLabel()}
          </DropdownMenuItem>

          <DropdownMenuItem onClick={handleToggleNotifications}>
            {settings.notifications ? <Bell className="w-4 h-4" /> : <BellOff className="w-4 h-4" />}
            {settings.notifications ? t('user.notificationsOn') : t('user.notificationsOff')}
          </DropdownMenuItem>

          <DropdownMenuSeparator />

          <DropdownMenuItem variant="destructive" onClick={handleLogout}>
            <LogOut className="w-4 h-4" />
            {t('user.logout')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }

  // Expanded state
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          className={cn(
            'w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors',
            'hover:bg-accent text-left'
          )}
        >
          {getAvatarUrl() ? (
            <img
              src={getAvatarUrl()!}
              alt={getDisplayName()}
              className="w-8 h-8 rounded-full object-cover flex-shrink-0"
            />
          ) : (
            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
              <User className="w-4 h-4 text-primary" />
            </div>
          )}

          <div className="flex-1 min-w-0">
            <p className="font-medium text-sm truncate">{getDisplayName()}</p>
            <p className="text-xs text-muted-foreground truncate">
              {user?.role === 'admin' ? t('user.admin') : t('user.user')}
            </p>
          </div>

          <ChevronUp className="w-4 h-4 text-muted-foreground flex-shrink-0" />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent side="top" align="start" className="w-full min-w-[200px]">
        <DropdownMenuItem onClick={() => setShowProfileDialog(true)}>
          <User className="w-4 h-4" />
          {t('user.profile')}
        </DropdownMenuItem>

        <DropdownMenuItem onClick={() => setShowSettingsDialog(true)}>
          <Settings className="w-4 h-4" />
          {t('user.settings')}
        </DropdownMenuItem>

        <DropdownMenuSeparator />

        <DropdownMenuItem onClick={handleToggleTheme}>
          {theme === 'light' && <Sun className="w-4 h-4" />}
          {theme === 'dark' && <Moon className="w-4 h-4" />}
          {theme === 'system' && <Monitor className="w-4 h-4" />}
          {t('user.theme')}: {getThemeLabel()}
        </DropdownMenuItem>

        <DropdownMenuItem onClick={handleToggleNotifications}>
          {settings.notifications ? <Bell className="w-4 h-4" /> : <BellOff className="w-4 h-4" />}
          {settings.notifications ? t('user.notificationsOn') : t('user.notificationsOff')}
        </DropdownMenuItem>

        <DropdownMenuSeparator />

        <DropdownMenuItem variant="destructive" onClick={handleLogout}>
          <LogOut className="w-4 h-4" />
          {t('user.logout')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
