import {
  LayoutDashboard,
  Package,
  Download,
  MessageSquare,
  Network,
  ScrollText,
  Volume2,
  Mic,
  Image as ImageIcon,
  Music,
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
  { path: '/tts', icon: Volume2, labelKey: 'sidebar.tts', feature: 'tts' },
  { path: '/asr', icon: Mic, labelKey: 'sidebar.asr', feature: 'asr' },
  { path: '/image-gen', icon: ImageIcon, labelKey: 'sidebar.imageGen', feature: 'imageGen' },
  { path: '/music-gen', icon: Music, labelKey: 'sidebar.musicGen', feature: 'musicGen' },
  { path: '/logs', icon: ScrollText, labelKey: 'sidebar.logs', feature: 'logs' },
  { path: '/settings', icon: Settings, labelKey: 'sidebar.settings', feature: 'settings' },
];
