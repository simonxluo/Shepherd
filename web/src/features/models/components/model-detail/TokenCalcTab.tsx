import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { applyTemplate, tokenizeText } from '@/features/models/model-detail-api';

interface TokenCalcTabProps {
  modelId: string;
  isLoaded: boolean;
}

export function TokenCalcTab({ modelId, isLoaded }: TokenCalcTabProps) {
  const { t } = useTranslation();
  const [input, setInput] = useState('');
  const [prompt, setPrompt] = useState('');
  const [tokenCount, setTokenCount] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState('');

  const handleCalculate = useCallback(async () => {
    if (!input.trim()) return;
    setLoading(true);
    setStatus('');
    try {
      const messages = [{ role: 'user', content: input }];
      const templateRes = await applyTemplate(modelId, messages);
      const generatedPrompt = templateRes.prompt;
      setPrompt(generatedPrompt);

      const tokenRes = await tokenizeText(modelId, generatedPrompt);
      setTokenCount(tokenRes.tokens.length);
    } catch {
      setStatus(t('modelDetail.tokenCalc.error', '计算失败'));
    } finally {
      setLoading(false);
    }
  }, [modelId, input, t]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.ctrlKey && e.key === 'Enter') {
      e.preventDefault();
      handleCalculate();
    }
  };

  if (!isLoaded) {
    return (
      <div className="flex items-center justify-center p-8 text-muted-foreground text-sm">
        {t('modelDetail.tokenCalc.notLoaded', '模型未加载，无法计算 Token')}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 p-4 h-full">
      {/* Controls */}
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={handleCalculate} disabled={loading || !input.trim()}>
          {loading
            ? t('modelDetail.tokenCalc.calculating', '计算中...')
            : t('modelDetail.tokenCalc.calculate', '计算')}
        </Button>
        <span className="text-xs text-muted-foreground">Ctrl+Enter</span>
        {tokenCount !== null && (
          <span className="text-sm font-medium ml-auto">
            {t('modelDetail.tokenCalc.count', 'Token 数')}: {tokenCount}
          </span>
        )}
        {status && <span className="text-xs text-destructive ml-2">{status}</span>}
      </div>

      {/* Text areas */}
      <div className="grid grid-cols-2 gap-3 flex-1 min-h-[250px]">
        <textarea
          className="w-full h-full rounded-md border border-input bg-background p-3 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={t('modelDetail.tokenCalc.inputPlaceholder', '输入文本...')}
        />
        <textarea
          className="w-full h-full rounded-md border border-input bg-muted/30 p-3 text-sm font-mono resize-none"
          value={prompt}
          readOnly
          placeholder={t('modelDetail.tokenCalc.promptPlaceholder', '生成的 Prompt 将显示在这里')}
        />
      </div>
    </div>
  );
}
