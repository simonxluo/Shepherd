import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { 
  Settings, 
  Moon, 
  Sun, 
  Monitor, 
  Bell, 
  Mail,
  Save,
  Palette,
  Sparkles,
  Check,
  Laptop,
  Globe
} from 'lucide-react';
import { useUserStore } from '@/stores/userStore';
import { useUIStore } from '@/stores/uiStore';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';

const languages = [
  { code: 'zh-CN', name: '简体中文', flag: '🇨🇳', region: '中国大陆' },
  { code: 'en-US', name: 'English', flag: '🇺🇸', region: 'United States' },
];

const themes = [
  { 
    id: 'light' as const, 
    name: 'theme.light', 
    icon: Sun,
    description: 'settings.themeLightDesc',
    gradient: 'from-amber-100 to-orange-50'
  },
  { 
    id: 'dark' as const, 
    name: 'theme.dark', 
    icon: Moon,
    description: 'settings.themeDarkDesc',
    gradient: 'from-slate-800 to-slate-900'
  },
  { 
    id: 'system' as const, 
    name: 'theme.system', 
    icon: Monitor,
    description: 'settings.themeSystemDesc',
    gradient: 'from-blue-100 to-indigo-100 dark:from-blue-900 dark:to-indigo-900'
  },
];

export function UserSettingsDialog() {
  const { t, i18n } = useTranslation();
  const { 
    showSettingsDialog, 
    setShowSettingsDialog, 
    settings, 
    updateSettings 
  } = useUserStore();
  const { theme, setTheme } = useUIStore();

  const [activeTab, setActiveTab] = useState('appearance');
  const [localSettings, setLocalSettings] = useState(settings);
  const [savedState, setSavedState] = useState<Record<string, boolean>>({});

  const handleSettingChange = <K extends keyof typeof localSettings>(
    key: K, 
    value: typeof localSettings[K]
  ) => {
    setLocalSettings(prev => ({ ...prev, [key]: value }));
  };

  const handleThemeChange = (newTheme: 'light' | 'dark' | 'system') => {
    setTheme(newTheme);
  };

  const handleSave = () => {
    updateSettings(localSettings);
    if (localSettings.language !== i18n.language) {
      i18n.changeLanguage(localSettings.language);
    }
    setSavedState({ ...savedState, [activeTab]: true });
    setTimeout(() => {
      setSavedState(prev => ({ ...prev, [activeTab]: false }));
    }, 2000);
  };

  const hasChanges = JSON.stringify(localSettings) !== JSON.stringify(settings);

  const handleClose = () => {
    setLocalSettings(settings);
    setShowSettingsDialog(false);
  };

  return (
    <Dialog open={showSettingsDialog} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-2xl p-0 overflow-hidden bg-gradient-to-br from-background to-muted/20">
        <DialogHeader className="px-6 pt-6 pb-2">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-xl bg-primary/10 text-primary">
              <Settings className="w-6 h-6" />
            </div>
            <div>
              <DialogTitle className="text-2xl font-bold">{t('user.settings')}</DialogTitle>
              <p className="text-sm text-muted-foreground mt-1">
                {t('user.settingsDescription')}
              </p>
            </div>
          </div>
        </DialogHeader>

        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <div className="px-6">
            <TabsList className="w-full grid grid-cols-3 bg-muted/50 p-1 rounded-xl">
              <TabsTrigger value="appearance" className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm">
                <Palette className="w-4 h-4 mr-2" />
                {t('settings.appearance')}
              </TabsTrigger>
              <TabsTrigger value="notifications" className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm">
                <Bell className="w-4 h-4 mr-2" />
                {t('settings.notifications')}
              </TabsTrigger>
              <TabsTrigger value="general" className="rounded-lg data-[state=active]:bg-background data-[state=active]:shadow-sm">
                <Sparkles className="w-4 h-4 mr-2" />
                {t('settings.general')}
              </TabsTrigger>
            </TabsList>
          </div>

          <div className="px-6 py-4 max-h-[60vh] overflow-y-auto">
            <TabsContent value="appearance" className="mt-0 space-y-4">
              <Card className="border-none shadow-md bg-gradient-to-br from-card to-card/50">
                <CardHeader className="pb-3">
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Palette className="w-5 h-5 text-primary" />
                    {t('settings.theme')}
                  </CardTitle>
                  <CardDescription>
                    {t('settings.themeDescription')}
                  </CardDescription>
                </CardHeader>
                <CardContent className="grid grid-cols-3 gap-3">
                  {themes.map((tItem) => {
                    const Icon = tItem.icon;
                    const isActive = theme === tItem.id;
                    return (
                      <button
                        key={tItem.id}
                        onClick={() => handleThemeChange(tItem.id)}
                        className={cn(
                          'relative flex flex-col items-center gap-3 p-4 rounded-xl border-2 transition-all duration-300 group',
                          'hover:scale-[1.02] hover:shadow-lg',
                          isActive 
                            ? 'border-primary bg-gradient-to-br shadow-md ' + tItem.gradient
                            : 'border-border/50 bg-card hover:border-primary/30'
                        )}
                      >
                        <div className={cn(
                          'p-3 rounded-full transition-colors',
                          isActive ? 'bg-primary text-primary-foreground' : 'bg-muted group-hover:bg-primary/10'
                        )}>
                          <Icon className="w-6 h-6" />
                        </div>
                        <div className="text-center">
                          <p className={cn(
                            'font-semibold text-sm',
                            isActive && tItem.id === 'dark' ? 'text-white' : ''
                          )}>
                            {/* @ts-ignore */}
                            {t(tItem.name)}
                          </p>
                          <p className={cn(
                            'text-xs mt-1',
                            isActive && tItem.id === 'dark' ? 'text-white/70' : 'text-muted-foreground'
                          )}>
                            {/* @ts-ignore */}
                            {t(tItem.description)}
                          </p>
                        </div>
                        {isActive && (
                          <div className="absolute top-2 right-2">
                            <Check className={cn(
                              'w-4 h-4',
                              tItem.id === 'dark' ? 'text-white' : 'text-primary'
                            )} />
                          </div>
                        )}
                      </button>
                    );
                  })}
                </CardContent>
              </Card>

              <Card className="border-none shadow-md">
                <CardHeader className="pb-3">
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Globe className="w-5 h-5 text-primary" />
                    {t('settings.language')}
                  </CardTitle>
                  <CardDescription>
                    {t('settings.languageDescription')}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-2">
                  {languages.map((lang) => {
                    const isActive = localSettings.language === lang.code;
                    return (
                      <button
                        key={lang.code}
                        onClick={() => handleSettingChange('language', lang.code)}
                        className={cn(
                          'w-full flex items-center gap-4 p-4 rounded-xl border-2 transition-all duration-200',
                          'hover:shadow-md hover:scale-[1.01]',
                          isActive 
                            ? 'border-primary bg-primary/5' 
                            : 'border-border/50 bg-card hover:border-primary/30'
                        )}
                      >
                        <span className="text-3xl">{lang.flag}</span>
                        <div className="flex-1 text-left">
                          <p className="font-semibold">{lang.name}</p>
                          <p className="text-sm text-muted-foreground">{lang.region}</p>
                        </div>
                        {isActive && (
                          <Badge variant="default" className="bg-primary">
                            <Check className="w-3 h-3 mr-1" />
                            {t('common.active')}
                          </Badge>
                        )}
                      </button>
                    );
                  })}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="notifications" className="mt-0 space-y-4">
              <Card className="border-none shadow-md">
                <CardHeader className="pb-3">
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Bell className="w-5 h-5 text-primary" />
                    {t('settings.notifications')}
                  </CardTitle>
                  <CardDescription>
                    {t('settings.notificationsDescription')}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <NotificationSetting
                    icon={Bell}
                    title={t('settings.pushNotifications')}
                    description={t('settings.pushNotificationsDesc')}
                    checked={localSettings.notifications}
                    onCheckedChange={(checked) => handleSettingChange('notifications', checked)}
                  />
                  
                  <NotificationSetting
                    icon={Laptop}
                    title={t('settings.desktopNotifications')}
                    description={t('settings.desktopNotificationsDesc')}
                    checked={localSettings.notifications}
                    onCheckedChange={(checked) => handleSettingChange('notifications', checked)}
                  />
                  
                  <NotificationSetting
                    icon={Mail}
                    title={t('settings.emailNotifications')}
                    description={t('settings.emailNotificationsDesc')}
                    checked={false}
                    onCheckedChange={() => {}}
                    disabled
                  />
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="general" className="mt-0 space-y-4">
              <Card className="border-none shadow-md">
                <CardHeader className="pb-3">
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Save className="w-5 h-5 text-primary" />
                    {t('settings.data')}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex items-center justify-between p-4 rounded-xl border bg-card">
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-primary/10">
                        <Save className="w-5 h-5 text-primary" />
                      </div>
                      <div>
                        <p className="font-medium">{t('settings.autoSave')}</p>
                        <p className="text-sm text-muted-foreground">
                          {t('settings.autoSaveDescription')}
                        </p>
                      </div>
                    </div>
                    <Switch
                      checked={localSettings.autoSave}
                      onCheckedChange={(checked) => handleSettingChange('autoSave', checked)}
                    />
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </div>
        </Tabs>

        <div className="px-6 py-4 border-t bg-muted/30 flex justify-between items-center">
          <div className="text-sm text-muted-foreground">
            {savedState[activeTab] ? (
              <span className="flex items-center gap-1 text-green-600">
                <Check className="w-4 h-4" />
                {t('common.saved')}
              </span>
            ) : hasChanges ? (
              t('common.unsavedChanges')
            ) : (
              t('common.allChangesSaved')
            )}
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleClose}>
              {t('common.close')}
            </Button>
            <Button 
              onClick={handleSave} 
              disabled={!hasChanges}
              className="gap-2"
            >
              {savedState[activeTab] ? (
                <>
                  <Check className="w-4 h-4" />
                  {t('common.saved')}
                </>
              ) : (
                t('common.save')
              )}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

interface NotificationSettingProps {
  icon: React.ElementType;
  title: string;
  description: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
}

function NotificationSetting({ 
  icon: Icon, 
  title, 
  description, 
  checked, 
  onCheckedChange,
  disabled 
}: NotificationSettingProps) {
  return (
    <div className={cn(
      'flex items-center justify-between p-4 rounded-xl border transition-all',
      checked && !disabled 
        ? 'border-primary/30 bg-primary/5' 
        : 'border-border/50 bg-card',
      disabled && 'opacity-60'
    )}>
      <div className="flex items-center gap-3">
        <div className={cn(
          'p-2 rounded-lg transition-colors',
          checked && !disabled ? 'bg-primary text-primary-foreground' : 'bg-muted'
        )}>
          <Icon className="w-5 h-5" />
        </div>
        <div>
          <p className="font-medium">{title}</p>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
      </div>
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        disabled={disabled}
      />
    </div>
  );
}
