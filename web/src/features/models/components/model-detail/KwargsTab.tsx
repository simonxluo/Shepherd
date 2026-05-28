import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import {
  getModelKwargs,
  saveModelKwargs,
  deleteModelKwargs,
} from '@/features/models/model-detail-api';

interface KwargsTabProps {
  modelId: string;
}

export function KwargsTab({ modelId }: KwargsTabProps) {
  const { t } = useTranslation();
  const [jsonText, setJsonText] = useState('{}');
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');

  const loadKwargs = useCallback(async () => {
    try {
      const kwargs = await getModelKwargs(modelId);
      setJsonText(JSON.stringify(kwargs || {}, null, 2));
      setError('');
    } catch {
      setStatus(t('modelDetail.kwargs.loadError', '加载失败'));
    }
  }, [modelId, t]);

  useEffect(() => {
    loadKwargs();
  }, [loadKwargs]);

  const handleApply = async () => {
    setError('');
    try {
      const parsed = JSON.parse(jsonText);
      await saveModelKwargs(modelId, parsed);
      setStatus(t('modelDetail.kwargs.saved', '已应用'));
      setTimeout(() => setStatus(''), 2000);
    } catch (e) {
      if (e instanceof SyntaxError) {
        setError(t('modelDetail.kwargs.invalidJson', 'JSON 格式无效'));
      } else {
        setStatus(t('modelDetail.kwargs.saveError', '保存失败'));
      }
    }
  };

  const handleClear = async () => {
    try {
      await deleteModelKwargs(modelId);
      setJsonText('{}');
      setError('');
      setStatus(t('modelDetail.kwargs.cleared', '已清除'));
      setTimeout(() => setStatus(''), 2000);
    } catch {
      setStatus(t('modelDetail.kwargs.clearError', '清除失败'));
    }
  };

  return (
    <div className="flex flex-col gap-3 p-4 h-full">
      {/* Actions */}
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={handleApply}>
          {t('modelDetail.kwargs.apply', '应用')}
        </Button>
        <Button variant="destructive" size="sm" onClick={handleClear}>
          {t('modelDetail.kwargs.clear', '清除')}
        </Button>
        {status && <span className="text-xs text-muted-foreground ml-2">{status}</span>}
        {error && <span className="text-xs text-destructive ml-2">{error}</span>}
      </div>

      {/* JSON editor */}
      <textarea
        className="flex-1 min-h-[250px] w-full rounded-md border border-input bg-background p-3 text-sm font-mono resize-none focus:outline-none focus:ring-2 focus:ring-ring"
        value={jsonText}
        onChange={(e) => {
          setJsonText(e.target.value);
          setError('');
        }}
        spellCheck={false}
        placeholder={t('modelDetail.kwargs.placeholder', '输入 JSON...')}
      />
    </div>
  );
}
