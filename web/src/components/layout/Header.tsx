import { Menu } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { ThemeToggle } from '@/components/ui/ThemeToggle';
import { LanguageToggle } from '@/components/ui/LanguageToggle';
import { NotificationToggle } from '@/components/ui/NotificationToggle';
import { useUIStore } from '@/stores/uiStore';

export function Header() {
  const { t } = useTranslation();
  const { toggleMobileMenu } = useUIStore();

  return (
    <header className="flex h-16 items-center justify-between border-b bg-background px-4 md:px-6">
      {/* Mobile hamburger */}
      <Button
        variant="ghost"
        size="icon"
        className="md:hidden"
        onClick={toggleMobileMenu}
        aria-label={t('sidebar.toggle' as any)}
      >
        <Menu size={20} />
      </Button>

      {/* Desktop spacer */}
      <div className="hidden md:block" />

      <div className="flex items-center gap-2">
        <LanguageToggle />
        <ThemeToggle />
        <NotificationToggle />
      </div>
    </header>
  );
}
