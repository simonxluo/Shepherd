import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import {
  getModelChatTemplate,
  saveModelChatTemplate,
  deleteModelChatTemplate,
  getModelDefaultChatTemplate,
} from '@/features/models/model-detail-api';

interface ChatTemplateTabProps {
  modelId: string;
}

export function ChatTemplateTab({ modelId }: ChatTemplateTabProps) {
  const { t } = useTranslation();
  const [template, setTemplate] = useState('');
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState('');

  const loadTemplate = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getModelChatTemplate(modelId);
      setTemplate(data.chatTemplate || '');
    } catch {
      setStatus(t('modelDetail.chatTemplate.loadFailed', '加载失败'));
    } finally {
      setLoading(false);
    }
  }, [modelId, t]);

  useEffect(() => { loadTemplate(); }, [loadTemplate]);

  const handleDefault = async () => {
    try {
      const data = await getModelDefaultChatTemplate(modelId);
      if (data.exists) {
        setTemplate(data.chatTemplate);
        setStatus(t('modelDetail.chatTemplate.defaultLoaded', '已加载默认模板'));
      } else {
        setStatus(t('modelDetail.chatTemplate.noDefault', '该模型未提供默认模板'));
      }
      setTimeout(() => setStatus(''), 3000);
    } catch {
      setStatus(t('modelDetail.chatTemplate.loadFailed', '加载失败'));
    }
  };

  const handleSave = async () => {
    if (!template.trim()) {
      setStatus(t('modelDetail.chatTemplate.empty', '模板不能为空'));
      setTimeout(() => setStatus(''), 3000);
      return;
    }
    try {
      await saveModelChatTemplate(modelId, template);
      setStatus(t('modelDetail.chatTemplate.saved', '已保存'));
      setTimeout(() => setStatus(''), 3000);
    } catch {
      setStatus(t('modelDetail.chatTemplate.saveFailed', '保存失败'));
    }
  };

  const handleDelete = async () => {
    if (!confirm(t('modelDetail.chatTemplate.confirmDelete', '确定要删除已保存的聊天模板吗？'))) return;
    try {
      await deleteModelChatTemplate(modelId);
      setTemplate('');
      setStatus(t('modelDetail.chatTemplate.deleted', '已删除'));
      setTimeout(() => setStatus(''), 3000);
    } catch {
      setStatus(t('modelDetail.chatTemplate.deleteFailed', '删除失败'));
    }
  };

  if (loading) {
    return <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">{t('common.loading', '加载中...')}</div>;
  }

  return (
    <div className="flex flex-col h-full gap-3">
      <div className="flex items-center gap-2 flex-wrap">
        <Button variant="outline" size="sm" onClick={handleDefault}>{t('modelDetail.chatTemplate.default', '默认')}</Button>
        <Button variant="outline" size="sm" onClick={loadTemplate}>{t('common.refresh', '刷新')}</Button>
        <Button variant="outline" size="sm" onClick={handleSave}>{t('common.save', '保存')}</Button>
        <Button variant="destructive" size="sm" onClick={handleDelete}>{t('common.delete', '删除')}</Button>
        {status && <span className="text-xs text-muted-foreground ml-2">{status}</span>}
      </div>
      <textarea
        className="flex-1 min-h-[300px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono resize-y"
        value={template}
        onChange={e => setTemplate(e.target.value)}
        placeholder={t('modelDetail.chatTemplate.placeholder', '(可选) 输入自定义聊天模板...')}
      />
    </div>
  );
}
