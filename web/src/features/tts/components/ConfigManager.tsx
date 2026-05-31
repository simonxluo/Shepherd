import { useState, useCallback, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Save, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { toast } from '@/hooks/useToast';
import { cn } from '@/lib/utils';
import type { TTSConfig } from '@/features/tts/types';

interface ConfigManagerProps {
  /** Model name used as localStorage key segment */
  modelName: string;
  /** Model ID for server-side config */
  modelId: string;
  /** Current config state to save */
  getCurrentConfig: () => TTSConfig;
  /** Apply a loaded config */
  onLoadConfig: (config: TTSConfig) => void;
  /** Server-side save handler */
  onSaveToServer: () => void;
  /** Server-side delete handler */
  onDeleteFromServer: () => void;
  /** Whether server-side config exists */
  hasServerConfig: boolean;
}

export function ConfigManager({
  modelName,
  modelId,
  getCurrentConfig,
  onLoadConfig,
  onSaveToServer,
  onDeleteFromServer,
  hasServerConfig,
}: ConfigManagerProps) {
  const { t } = useTranslation();
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [configName, setConfigName] = useState('');
  const [selectedConfigName, setSelectedConfigName] = useState('');
  const [savedConfigs, setSavedConfigs] = useState<{ name: string; config: TTSConfig }[]>([]);
  const statusTimerRef = useRef<ReturnType<typeof setTimeout>>(null);

  const CONFIGS_STORAGE_KEY = modelName ? `shepherd:tts-configs:${modelName}` : '';

  const getSavedConfigs = useCallback((): { name: string; config: TTSConfig }[] => {
    if (!CONFIGS_STORAGE_KEY) return [];
    try {
      const data = localStorage.getItem(CONFIGS_STORAGE_KEY);
      return data ? JSON.parse(data) : [];
    } catch {
      return [];
    }
  }, [CONFIGS_STORAGE_KEY]);

  // Refresh configs when model changes; clean up pending timer on unmount
  useEffect(() => {
    return () => {
      if (statusTimerRef.current !== null) clearTimeout(statusTimerRef.current);
    };
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setSavedConfigs(getSavedConfigs());
    setConfigName('');
    setSelectedConfigName('');
    setSaveStatus('idle');
  }, [modelName, getSavedConfigs]);

  const handleLoadNamedConfig = useCallback((name: string) => {
    const configs = getSavedConfigs();
    const found = configs.find((c) => c.name === name);
    if (found) {
      onLoadConfig(found.config);
      setSelectedConfigName(name);
      setConfigName(name);
    }
  }, [getSavedConfigs, onLoadConfig]);

  const handleSaveNamedConfig = useCallback(() => {
    if (!CONFIGS_STORAGE_KEY) return;
    const name = configName.trim();
    if (!name) {
      toast.error(t('tts.selectModelWarning', 'Please enter a config name'));
      return;
    }
    const config = getCurrentConfig();
    try {
      const configs = getSavedConfigs();
      const idx = configs.findIndex((c) => c.name === name);
      if (idx >= 0) {
        configs[idx] = { name, config };
      } else {
        configs.push({ name, config });
      }
      localStorage.setItem(CONFIGS_STORAGE_KEY, JSON.stringify(configs));
      setSavedConfigs([...configs]);
      setSelectedConfigName(name);
      setSaveStatus('saved');
      statusTimerRef.current = setTimeout(() => setSaveStatus('idle'), 3000);
    } catch (error) {
      console.error('Failed to save config:', error);
      setSaveStatus('error');
      statusTimerRef.current = setTimeout(() => setSaveStatus('idle'), 3000);
    }
  }, [CONFIGS_STORAGE_KEY, configName, getCurrentConfig, getSavedConfigs, t]);

  const handleDeleteNamedConfig = useCallback((name: string) => {
    if (!CONFIGS_STORAGE_KEY) return;
    const configs = getSavedConfigs().filter((c) => c.name !== name);
    localStorage.setItem(CONFIGS_STORAGE_KEY, JSON.stringify(configs));
    setSavedConfigs([...configs]);
    if (selectedConfigName === name) {
      setSelectedConfigName('');
    }
  }, [CONFIGS_STORAGE_KEY, getSavedConfigs, selectedConfigName]);

  return (
    <div className="border rounded-lg p-3 space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-muted-foreground">
          {t('tts.configManagement', 'Config Management')}
        </h4>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={onSaveToServer}
            disabled={!modelId}
            className="text-xs h-7"
          >
            <Save className="w-3.5 h-3.5 mr-1" />
            {t('tts.saveToServer', 'Save to Server')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onDeleteFromServer}
            disabled={!modelId || !hasServerConfig}
            className="text-xs h-7"
          >
            <Trash2 className="w-3.5 h-3.5 mr-1" />
            {t('tts.deleteServerConfig', 'Delete Server Config')}
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <select
          value={selectedConfigName}
          onChange={(e) => {
            if (e.target.value) handleLoadNamedConfig(e.target.value);
          }}
          className={cn(
            'h-8 px-2 text-sm border-2 border-border rounded-md flex-1',
            'bg-input text-foreground',
            'focus:outline-none focus:ring-2 focus:ring-blue-500'
          )}
        >
          <option value="">{t('tts.selectPreset', 'Select preset...')}</option>
          {savedConfigs.map((c) => (
            <option key={c.name} value={c.name}>{c.name}</option>
          ))}
        </select>
        {selectedConfigName && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => handleDeleteNamedConfig(selectedConfigName)}
            className="text-muted-foreground hover:text-destructive h-8 w-8 p-0"
            title={t('tts.deletePreset', 'Delete preset')}
          >
            <Trash2 className="w-3.5 h-3.5" />
          </Button>
        )}
        <div className="w-px h-6 bg-border" />
        <input
          type="text"
          value={configName}
          onChange={(e) => setConfigName(e.target.value)}
          placeholder={t('tts.presetName', 'Preset name')}
          className={cn(
            'h-8 px-2 text-sm w-28 border-2 border-border rounded-md',
            'bg-input text-foreground',
            'focus:outline-none focus:ring-2 focus:ring-blue-500'
          )}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              handleSaveNamedConfig();
            }
          }}
        />
        <Button
          variant="secondary"
          size="sm"
          onClick={handleSaveNamedConfig}
          className={cn(
            'text-xs h-8',
            saveStatus === 'saved' && 'bg-green-600 text-white hover:bg-green-700',
            saveStatus === 'error' && 'bg-red-600 text-white hover:bg-red-700'
          )}
        >
          {saveStatus === 'saved'
            ? `✓ ${t('tts.saved', 'Saved')}`
            : saveStatus === 'error'
              ? `✗ ${t('tts.saveFailed', 'Failed')}`
              : t('tts.savePreset', 'Save Preset')}
        </Button>
      </div>
    </div>
  );
}
