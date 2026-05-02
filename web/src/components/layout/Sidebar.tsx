import { Link, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useUIStore } from '@/stores/uiStore';
import { useConfig } from '@/lib/config';
import { cn } from '@/lib/utils';
import { UserMenu } from './UserMenu';
import {
  LayoutDashboard,
  Package,
  Download,
  MessageSquare,
  Network,
  ScrollText,
  Wand2,
  Settings,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';

const allNavItems = [
  { path: '/', icon: LayoutDashboard, labelKey: 'sidebar.dashboard', feature: 'dashboard' },
  { path: '/models', icon: Package, labelKey: 'sidebar.models', feature: 'models' },
  { path: '/downloads', icon: Download, labelKey: 'sidebar.downloads', feature: 'downloads' },
  { path: '/cluster', icon: Network, labelKey: 'sidebar.cluster', feature: 'cluster' },
  { path: '/chat', icon: MessageSquare, labelKey: 'sidebar.chat', feature: 'chat' },
  { path: '/multimodal', icon: Wand2, labelKey: 'sidebar.multimodal', feature: 'multimodal' },
  { path: '/logs', icon: ScrollText, labelKey: 'sidebar.logs', feature: 'logs' },
  { path: '/settings', icon: Settings, labelKey: 'sidebar.settings', feature: 'settings' },
];

export function Sidebar() {
  const location = useLocation();
  const { sidebarOpen, toggleSidebar } = useUIStore();
  const config = useConfig();
  const { t } = useTranslation();

  const navItems = allNavItems.filter(
    (item) => config.features[item.feature as keyof typeof config.features] !== false
  );

  return (
    <aside
      className={cn(
        'flex flex-col border-r bg-background transition-all duration-300',
        sidebarOpen ? 'w-64' : 'w-16'
      )}
    >
      <div className="flex h-16 items-center justify-between border-b px-4">
        {sidebarOpen && (
          <Link to="/" className="flex items-center gap-2 font-semibold">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              🐏
            </div>
            <span>Shepherd</span>
          </Link>
        )}
        <button
          onClick={toggleSidebar}
          className="ml-auto rounded-lg p-2 hover:bg-accent"
          aria-label={sidebarOpen ? t('sidebar.collapse') : t('sidebar.expand')}
        >
          {sidebarOpen ? <ChevronLeft size={18} /> : <ChevronRight size={18} />}
        </button>
      </div>

      <nav className="flex-1 overflow-y-auto py-4">
        <ul className="space-y-1 px-2">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.path;

            return (
              <li key={item.path}>
                <Link
                  to={item.path}
                  className={cn(
                    'relative flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200',
                    isActive
                      ? 'bg-primary text-primary-foreground shadow-sm'
                      : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                  )}
                >
                  {isActive && (
                    <span className="absolute left-0 top-1/2 -translate-y-1/2 h-8 w-1 rounded-r-full bg-current opacity-80" />
                  )}

                  <Icon size={20} />
                  {sidebarOpen && <span>{t(item.labelKey as any)}</span>}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      <div className="border-t p-4 space-y-4">
        <UserMenu sidebarOpen={sidebarOpen} />
        
        {sidebarOpen && (
          <div className="text-xs text-muted-foreground pt-2 border-t">
            <div>Shepherd v0.4.0</div>
            <div className="mt-1">{t('footer.copyright')}</div>
          </div>
        )}
      </div>
    </aside>
  );
}
