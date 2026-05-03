import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Type, WrapText, Search, X, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';

type FontSize = 'xxs' | 'xs' | 'small' | 'normal';

interface LogPanelProps {
  id: string;
  title: string;
  logData: string;
  onClear?: () => void;
}

const FONT_SIZE_CLASSES: Record<FontSize, string> = {
  xxs: 'text-[0.5rem] leading-[0.65rem]',
  xs: 'text-[0.75rem] leading-[1rem]',
  small: 'text-[0.875rem] leading-[1.25rem]',
  normal: 'text-base leading-[1.5rem]',
};

const FONT_SIZES: FontSize[] = ['xxs', 'xs', 'small', 'normal'];

function getLevelColor(line: string): string {
  if (line.includes(' ERROR ') || line.includes('"ERROR"')) return 'text-red-500';
  if (line.includes(' WARN ') || line.includes('"WARN"')) return 'text-yellow-500';
  if (line.includes(' DEBUG ') || line.includes('"DEBUG"')) return 'text-muted-foreground';
  if (line.includes(' FATAL ') || line.includes('"FATAL"')) return 'text-red-700 font-bold';
  return '';
}

export function LogPanel({ title, logData, onClear }: LogPanelProps) {
  const { t } = useTranslation();
  const [fontSize, setFontSize] = useState<FontSize>('small');
  const [wrapText, setWrapText] = useState(false);
  const [showFilter, setShowFilter] = useState(false);
  const [filterRegex, setFilterRegex] = useState('');
  const [userScrolledUp, setUserScrolledUp] = useState(false);
  const preRef = useRef<HTMLPreElement>(null);

  const handleScroll = useCallback(() => {
    if (!preRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = preRef.current;
    setUserScrolledUp(scrollHeight - scrollTop - clientHeight > 40);
  }, []);

  useEffect(() => {
    if (preRef.current && logData && !userScrolledUp) {
      preRef.current.scrollTop = preRef.current.scrollHeight;
    }
  }, [logData, userScrolledUp]);

  const toggleFontSize = useCallback(() => {
    setFontSize(prev => {
      const idx = FONT_SIZES.indexOf(prev);
      return FONT_SIZES[(idx + 1) % FONT_SIZES.length];
    });
  }, []);

  const filteredLogs = useMemo(() => {
    if (!filterRegex) return logData;
    try {
      const regex = new RegExp(filterRegex, 'i');
      return logData.split('\n').filter(line => regex.test(line)).join('\n');
    } catch {
      return logData;
    }
  }, [logData, filterRegex]);

  const lines = useMemo(() => filteredLogs.split('\n'), [filteredLogs]);

  return (
    <div className="rounded-lg overflow-hidden flex flex-col bg-muted/30 h-full">
      <div className="flex items-center justify-between px-3 py-2 border-b bg-muted/50">
        <h3 className="text-sm font-medium">{title}</h3>
        <div className="flex gap-1 items-center">
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={toggleFontSize} title={t('logs.fontSize', '字体大小')}>
            <Type className="w-3.5 h-3.5" />
          </Button>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setWrapText(p => !p)} title={t('logs.wrapText', '自动换行')}>
            <WrapText className={`w-3.5 h-3.5 ${wrapText ? 'text-primary' : ''}`} />
          </Button>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setShowFilter(p => !p)} title={t('logs.filter', '过滤')}>
            <Search className={`w-3.5 h-3.5 ${showFilter ? 'text-primary' : ''}`} />
          </Button>
          {onClear && (
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClear} title={t('logs.clear', '清空')}>
              <Trash2 className="w-3.5 h-3.5" />
            </Button>
          )}
        </div>
      </div>

      {showFilter && (
        <div className="flex gap-2 items-center px-3 py-1.5 border-b bg-muted/20">
          <input
            type="text"
            className="flex-1 text-xs border border-border rounded px-2 py-1 bg-background outline-none focus:ring-1 focus:ring-primary"
            placeholder={t('logs.filterPlaceholder', '正则过滤...')}
            value={filterRegex}
            onChange={e => setFilterRegex(e.target.value)}
          />
          {filterRegex && (
            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => setFilterRegex('')}>
              <X className="w-3 h-3" />
            </Button>
          )}
        </div>
      )}

      <div className="flex-1 overflow-hidden">
        <pre
          ref={preRef}
          onScroll={handleScroll}
          className={`h-full overflow-auto p-3 font-mono ${FONT_SIZE_CLASSES[fontSize]} ${wrapText ? 'whitespace-pre-wrap break-all' : 'whitespace-pre'}`}
        >
          {lines.map((line, i) => (
            <div key={i} className={getLevelColor(line)}>{line || '\u00A0'}</div>
          ))}
        </pre>
      </div>
    </div>
  );
}
