import { useState, useEffect, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import {
  listSamplingConfigs,
  saveSamplingConfig,
  deleteSamplingConfig,
  getModelSamplingSelection,
  setModelSamplingSelection,
  type SamplingConfig,
} from '@/features/models/model-detail-api';

interface SamplingTabProps {
  modelId: string;
}

const SAMPLING_FIELDS = [
  { key: 'temperature', label: '温度', labelKey: 'modelDetail.sampling.temperature', type: 'number', step: '0.1' },
  { key: 'top_p', label: 'Top-P', labelKey: 'modelDetail.sampling.topP', type: 'number', step: '0.01' },
  { key: 'top_k', label: 'Top-K', labelKey: 'modelDetail.sampling.topK', type: 'number', step: '1' },
  { key: 'min_p', label: 'Min-P', labelKey: 'modelDetail.sampling.minP', type: 'number', step: '0.01' },
  { key: 'top_n_sigma', label: 'Top-N Sigma', labelKey: 'modelDetail.sampling.topNSigma', type: 'number', step: '0.1' },
  { key: 'presence_penalty', label: '存在惩罚', labelKey: 'modelDetail.sampling.presencePenalty', type: 'number', step: '0.1' },
  { key: 'repeat_penalty', label: '重复惩罚', labelKey: 'modelDetail.sampling.repeatPenalty', type: 'number', step: '0.1' },
  { key: 'frequency_penalty', label: '频率惩罚', labelKey: 'modelDetail.sampling.frequencyPenalty', type: 'number', step: '0.1' },
  { key: 'dry_multiplier', label: 'DRY 倍数', labelKey: 'modelDetail.sampling.dryMultiplier', type: 'number', step: '0.1' },
  { key: 'dry_base', label: 'DRY 底数', labelKey: 'modelDetail.sampling.dryBase', type: 'number', step: '0.01' },
  { key: 'seed', label: '随机种子', labelKey: 'modelDetail.sampling.seed', type: 'number', step: '1' },
] as const;

export function SamplingTab({ modelId }: SamplingTabProps) {
  const { t } = useTranslation();
  const [configs, setConfigs] = useState<Record<string, SamplingConfig>>({});
  const [selectedName, setSelectedName] = useState('');
  const [currentConfig, setCurrentConfig] = useState<SamplingConfig>({});
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState('');
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [cfgs, selection] = await Promise.all([
        listSamplingConfigs(),
        getModelSamplingSelection(modelId),
      ]);
      setConfigs(cfgs || {});
      setSelectedName(selection || '');
      if (selection && cfgs[selection]) {
        setCurrentConfig(cfgs[selection]);
      }
    } catch {
      setStatus(t('modelDetail.sampling.loadFailed', '加载失败'));
    } finally {
      setLoading(false);
    }
  }, [modelId, t]);

  useEffect(() => { loadData(); }, [loadData]);

  const handleSelectChange = async (name: string) => {
    setSelectedName(name);
    if (name && configs[name]) {
      setCurrentConfig(configs[name]);
    } else {
      setCurrentConfig({});
    }
    try {
      await setModelSamplingSelection(modelId, name);
    } catch { /* ignore */ }
  };

  const scheduleAutoSave = useCallback(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(async () => {
      if (!selectedName) return;
      try {
        await saveSamplingConfig(selectedName, currentConfig);
        setConfigs(prev => ({ ...prev, [selectedName]: { ...currentConfig } }));
      } catch { /* ignore */ }
    }, 400);
  }, [selectedName, currentConfig]);

  const handleFieldChange = (key: string, value: string) => {
    const numVal = value === '' ? undefined : Number(value);
    setCurrentConfig(prev => {
      const next = { ...prev };
      if (numVal === undefined || isNaN(numVal)) {
        delete next[key];
      } else {
        next[key] = numVal;
      }
      return next;
    });
    scheduleAutoSave();
  };

  const handleAdd = async () => {
    const name = prompt(t('modelDetail.sampling.newNamePrompt', '请输入新的采样配置名称'));
    if (!name?.trim()) return;
    try {
      await saveSamplingConfig(name.trim(), currentConfig);
      setConfigs(prev => ({ ...prev, [name.trim()]: { ...currentConfig } }));
      setSelectedName(name.trim());
      await setModelSamplingSelection(modelId, name.trim());
      setStatus(t('modelDetail.sampling.added', '已新增'));
      setTimeout(() => setStatus(''), 2000);
    } catch { setStatus(t('modelDetail.sampling.saveFailed', '保存失败')); }
  };

  const handleSave = async () => {
    if (!selectedName) return;
    try {
      await saveSamplingConfig(selectedName, currentConfig);
      await setModelSamplingSelection(modelId, selectedName);
      setConfigs(prev => ({ ...prev, [selectedName]: { ...currentConfig } }));
      setStatus(t('modelDetail.sampling.saved', '已保存'));
      setTimeout(() => setStatus(''), 2000);
    } catch { setStatus(t('modelDetail.sampling.saveFailed', '保存失败')); }
  };

  const handleDelete = async () => {
    if (!selectedName) return;
    if (!confirm(t('modelDetail.sampling.confirmDelete', '确定要删除此配置吗？') + `\n${selectedName}`)) return;
    try {
      await deleteSamplingConfig(selectedName);
      const next = { ...configs };
      delete next[selectedName];
      setConfigs(next);
      setSelectedName('');
      setCurrentConfig({});
      await setModelSamplingSelection(modelId, '');
    } catch { setStatus(t('modelDetail.sampling.deleteFailed', '删除失败')); }
  };

  if (loading) {
    return <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">{t('common.loading', '加载中...')}</div>;
  }

  return (
    <div className="space-y-4">
      <p className="text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2">
        {t('modelDetail.sampling.desc', '开启该功能后，将强制使用指定的采样配置，而忽略其它客户端中的采样设置。')}
      </p>

      <div className="flex items-center gap-2 flex-wrap">
        <select
          className="h-8 rounded-md border border-input bg-background px-3 text-sm"
          value={selectedName}
          onChange={e => handleSelectChange(e.target.value)}
        >
          <option value="">{t('modelDetail.sampling.off', '关闭功能')}</option>
          {Object.keys(configs).map(name => (
            <option key={name} value={name}>{name}</option>
          ))}
        </select>
        <Button variant="outline" size="sm" onClick={handleAdd}>{t('modelDetail.sampling.add', '新增')}</Button>
        <Button variant="outline" size="sm" onClick={handleSave} disabled={!selectedName}>{t('modelDetail.sampling.save', '保存')}</Button>
        <Button variant="destructive" size="sm" onClick={handleDelete} disabled={!selectedName}>{t('common.delete', '删除')}</Button>
        {status && <span className="text-xs text-muted-foreground ml-2">{status}</span>}
      </div>

      {selectedName && (
        <div className="grid grid-cols-2 gap-3">
          {SAMPLING_FIELDS.map(field => (
            <div key={field.key} className="space-y-1">
              <label className="text-xs text-muted-foreground">{t(field.labelKey, field.label)}</label>
              <input
                type="number"
                step={field.step}
                className="w-full h-8 rounded-md border border-input bg-background px-3 text-sm"
                value={currentConfig[field.key] ?? ''}
                onChange={e => handleFieldChange(field.key, e.target.value)}
                placeholder="-"
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
