import { ThemeToggle } from '@/components/ui/ThemeToggle';
import { LanguageToggle } from '@/components/ui/LanguageToggle';
import { NotificationToggle } from '@/components/ui/NotificationToggle';

export function Header() {
  return (
    <header className="flex h-16 items-center justify-end border-b bg-background px-6">
      <div className="flex items-center gap-2">
        <LanguageToggle />
        <ThemeToggle />
        <NotificationToggle />
      </div>
    </header>
  );
}
