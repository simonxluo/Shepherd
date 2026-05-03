import { useCallback, type JSX } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useUIStore } from '@/stores/uiStore';
import { useConfig } from '@/lib/config';
import { cn } from '@/lib/utils';
import {
  LayoutDashboard,
  Package,
  Download,
  MessageSquare,
  Network,
  ScrollText,
  Wand2,
  Settings,
  Ellipsis,
  X,
} from 'lucide-react';

const bottomNavItems = [
  { path: '/', icon: LayoutDashboard, labelKey: 'sidebar.dashboard', feature: 'dashboard' },
  { path: '/models', icon: Package, labelKey: 'sidebar.models', feature: 'models' },
  { path: '/chat', icon: MessageSquare, labelKey: 'sidebar.chat', feature: 'chat' },
  { path: '/downloads', icon: Download, labelKey: 'sidebar.downloads', feature: 'downloads' },
];

const moreNavItems = [
  { path: '/cluster', icon: Network, labelKey: 'sidebar.cluster', feature: 'cluster' },
  { path: '/multimodal', icon: Wand2, labelKey: 'sidebar.multimodal', feature: 'multimodal' },
  { path: '/logs', icon: ScrollText, labelKey: 'sidebar.logs', feature: 'logs' },
  { path: '/settings', icon: Settings, labelKey: 'sidebar.settings', feature: 'settings' },
];

export function MobileBottomNav(): JSX.Element {
  const location = useLocation();
  const { t } = useTranslation();
  const config = useConfig();
  const { morePanelOpen, setMorePanelOpen } = useUIStore();

  const visibleItems = bottomNavItems.filter(
    (item) => config.features[item.feature as keyof typeof config.features] !== false
  );
  const visibleMore = moreNavItems.filter(
    (item) => config.features[item.feature as keyof typeof config.features] !== false
  );

  const closeMore = useCallback(() => setMorePanelOpen(false), [setMorePanelOpen]);

  return (
    <>
      {/* More panel overlay */}
      {morePanelOpen && (
        <div className="fixed inset-0 z-50 flex flex-col justify-end">
          <div className="absolute inset-0 bg-black/50" onClick={closeMore} />
          <div className="relative z-10 bg-background rounded-t-xl shadow-lg animate-in slide-in-from-bottom">
            <div className="flex items-center justify-between px-4 py-3 border-b">
              <span className="font-semibold text-lg">{t('common.more' as any, '更多')}</span>
              <button onClick={closeMore} className="p-1 rounded-lg hover:bg-muted">
                <X size={20} />
              </button>
            </div>
            <nav className="p-2 pb-6">
              <ul className="space-y-1">
                {visibleMore.map((item) => {
                  const Icon = item.icon;
                  const isActive = location.pathname === item.path;
                  return (
                    <li key={item.path}>
                      <Link
                        to={item.path}
                        onClick={closeMore}
                        className={cn(
                          'flex items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition-colors',
                          isActive
                            ? 'bg-primary text-primary-foreground'
                            : 'text-foreground hover:bg-muted'
                        )}
                      >
                        <Icon size={20} />
                        <span>{t(item.labelKey as any)}</span>
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </nav>
          </div>
        </div>
      )}

      {/* Bottom navigation bar */}
      <nav className="fixed bottom-0 left-0 right-0 z-40 flex items-center justify-around h-14 border-t bg-background shadow-[0_-1px_3px_rgba(0,0,0,0.1)]">
        {visibleItems.map((item) => {
          const Icon = item.icon;
          const isActive = location.pathname === item.path;
          return (
            <Link
              key={item.path}
              to={item.path}
              className={cn(
                'flex flex-col items-center justify-center gap-0.5 py-1 px-3 text-[11px] font-medium transition-colors min-w-0',
                isActive
                  ? 'text-primary'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              <Icon size={22} />
              <span className="truncate">{t(item.labelKey as any)}</span>
            </Link>
          );
        })}
        {visibleMore.length > 0 && (
          <button
            onClick={() => setMorePanelOpen(true)}
            className={cn(
              'flex flex-col items-center justify-center gap-0.5 py-1 px-3 text-[11px] font-medium transition-colors min-w-0',
              visibleMore.some((i) => i.path === location.pathname)
                ? 'text-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Ellipsis size={22} />
            <span className="truncate">{t('common.more' as any, '更多')}</span>
          </button>
        )}
      </nav>
    </>
  );
}
