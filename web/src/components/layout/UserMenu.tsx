import { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { 
  User, 
  Settings, 
  LogOut, 
  ChevronUp,
  Shield,
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

interface UserMenuProps {
  sidebarOpen: boolean;
}

export function UserMenu({ sidebarOpen }: UserMenuProps) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  
  const { 
    user, 
    settings,
    logout,
    setShowProfileDialog,
    setShowSettingsDialog,
    updateSettings
  } = useUserStore();
  
  const { theme, setTheme } = useUIStore();

  // Close menu on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleLogout = () => {
    logout();
    setIsOpen(false);
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

  const getThemeIcon = () => {
    switch (theme) {
      case 'light': return <Sun className="w-4 h-4" />;
      case 'dark': return <Moon className="w-4 h-4" />;
      default: return <Monitor className="w-4 h-4" />;
    }
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

  // Collapsed state - avatar only
  if (!sidebarOpen) {
    return (
      <div className="relative" ref={menuRef}>
        <Button
          variant="ghost"
          size="icon"
          className="w-10 h-10 rounded-full"
          onClick={() => setIsOpen(!isOpen)}
        >
          {getAvatarUrl() ? (
            <img 
              src={getAvatarUrl()!} 
              alt={getDisplayName()}
              className="w-8 h-8 rounded-full object-cover"
            />
          ) : (
            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
              <User className="w-4 h-4 text-primary" />
            </div>
          )}
        </Button>

        {isOpen && (
          <div className="absolute bottom-full left-0 mb-2 w-48 bg-popover border rounded-lg shadow-lg py-1 z-50">
            <div className="px-3 py-2 border-b">
              <p className="font-medium text-sm truncate">{getDisplayName()}</p>
              {user?.email && (
                <p className="text-xs text-muted-foreground truncate">{user.email}</p>
              )}
            </div>
            
            <MenuItems
              onProfile={() => { setShowProfileDialog(true); setIsOpen(false); }}
              onSettings={() => { setShowSettingsDialog(true); setIsOpen(false); }}
              onTheme={handleToggleTheme}
              onNotifications={handleToggleNotifications}
              onLogout={handleLogout}
              theme={theme}
              notifications={settings.notifications}
              // @ts-ignore
              t={t}
            />
          </div>
        )}
      </div>
    );
  }

  // Expanded state
  return (
    <div className="relative" ref={menuRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors',
          'hover:bg-accent text-left',
          isOpen && 'bg-accent'
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
        
        <ChevronUp className={cn(
          'w-4 h-4 text-muted-foreground transition-transform flex-shrink-0',
          isOpen && 'rotate-180'
        )} />
      </button>

      {isOpen && (
        <div className="absolute bottom-full left-0 right-0 mb-2 bg-popover border rounded-lg shadow-lg py-1 z-50">
          <MenuItems
            onProfile={() => { setShowProfileDialog(true); setIsOpen(false); }}
            onSettings={() => { setShowSettingsDialog(true); setIsOpen(false); }}
            onTheme={handleToggleTheme}
            onNotifications={handleToggleNotifications}
            onLogout={handleLogout}
            theme={theme}
            notifications={settings.notifications}
            // @ts-ignore
            t={t}
          />
        </div>
      )}
    </div>
  );
}

interface MenuItemsProps {
  onProfile: () => void;
  onSettings: () => void;
  onTheme: () => void;
  onNotifications: () => void;
  onLogout: () => void;
  theme: string;
  notifications: boolean;
  t: (...args: unknown[]) => string;
}

function MenuItems({ 
  onProfile, 
  onSettings, 
  onTheme, 
  onNotifications,
  onLogout, 
  theme,
  notifications,
  t 
}: MenuItemsProps) {
  const getThemeLabel = () => {
    switch (theme) {
      case 'light': return t('theme.light');
      case 'dark': return t('theme.dark');
      default: return t('theme.system');
    }
  };

  return (
    <>
      <button
        onClick={onProfile}
        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent transition-colors"
      >
        <User className="w-4 h-4" />
        {t('user.profile')}
      </button>
      
      <button
        onClick={onSettings}
        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent transition-colors"
      >
        <Settings className="w-4 h-4" />
        {t('user.settings')}
      </button>
      
      <div className="border-t my-1" />
      
      <button
        onClick={onTheme}
        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent transition-colors"
      >
        {theme === 'light' && <Sun className="w-4 h-4" />}
        {theme === 'dark' && <Moon className="w-4 h-4" />}
        {theme === 'system' && <Monitor className="w-4 h-4" />}
        {t('user.theme')}: {getThemeLabel()}
      </button>
      
      <button
        onClick={onNotifications}
        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent transition-colors"
      >
        {notifications ? <Bell className="w-4 h-4" /> : <BellOff className="w-4 h-4" />}
        {notifications ? t('user.notificationsOn') : t('user.notificationsOff')}
      </button>
      
      <div className="border-t my-1" />
      
      <button
        onClick={onLogout}
        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent transition-colors text-destructive"
      >
        <LogOut className="w-4 h-4" />
        {t('user.logout')}
      </button>
    </>
  );
}
