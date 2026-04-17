import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { UserProfileDialog } from '@/components/layout/UserProfileDialog';
import { UserSettingsDialog } from '@/components/layout/UserSettingsDialog';

export function MainLayout() {
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />

      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />

        <main className="flex-1 overflow-y-auto bg-background p-6">
          <Outlet />
        </main>
      </div>

      <UserProfileDialog />
      <UserSettingsDialog />
    </div>
  );
}
