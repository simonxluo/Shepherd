import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { MobileBottomNav } from './MobileBottomNav';
import { UserProfileDialog } from '@/components/layout/UserProfileDialog';
import { UserSettingsDialog } from '@/components/layout/UserSettingsDialog';
import { useUIStore } from '@/stores/uiStore';

export function MainLayout() {
  const { mobileMenuOpen, setMobileMenuOpen } = useUIStore();

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Desktop/Tablet inline sidebar */}
      <Sidebar />

      {/* Mobile sidebar overlay */}
      {mobileMenuOpen && (
        <div className="fixed inset-0 z-50 md:hidden">
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => setMobileMenuOpen(false)}
          />
          <Sidebar overlay />
        </div>
      )}

      {/* Main content area */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />

        <main className="flex-1 overflow-y-auto bg-background p-4 md:p-6 pb-20 md:pb-6">
          <Outlet />
        </main>
      </div>

      {/* Mobile bottom navigation */}
      <div className="md:hidden">
        <MobileBottomNav />
      </div>

      <UserProfileDialog />
      <UserSettingsDialog />
    </div>
  );
}
