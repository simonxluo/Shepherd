import {
  LayoutDashboard,
  Package,
  Download,
  MessageSquare,
  Network,
  ScrollText,
  Wand2,
  Settings,
} from 'lucide-react';

export interface NavItem {
  path: string;
  icon: typeof LayoutDashboard;
  labelKey: string;
  feature: string;
}

export const navItems: NavItem[] = [
  { path: '/', icon: LayoutDashboard, labelKey: 'sidebar.dashboard', feature: 'dashboard' },
  { path: '/models', icon: Package, labelKey: 'sidebar.models', feature: 'models' },
  { path: '/downloads', icon: Download, labelKey: 'sidebar.downloads', feature: 'downloads' },
  { path: '/cluster', icon: Network, labelKey: 'sidebar.cluster', feature: 'cluster' },
  { path: '/chat', icon: MessageSquare, labelKey: 'sidebar.chat', feature: 'chat' },
  { path: '/multimodal', icon: Wand2, labelKey: 'sidebar.multimodal', feature: 'multimodal' },
  { path: '/logs', icon: ScrollText, labelKey: 'sidebar.logs', feature: 'logs' },
  { path: '/settings', icon: Settings, labelKey: 'sidebar.settings', feature: 'settings' },
];
