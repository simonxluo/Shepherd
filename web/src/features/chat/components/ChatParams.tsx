import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Settings2, RotateCcw, ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useChatStore, type SamplingParams } from '@/stores/chatStore';
import { cn } from '@/lib/utils';

interface ParamSliderProps {
  label: string;
  value: number;
  min: number;
  max: number;
  step: number;
  onChange: (value: number) => void;
}

function ParamSlider({ label, value, min, max, step, onChange }: ParamSliderProps) {
  return (
    <div className="flex items-center gap-3">
      <label className="text-sm text-muted-foreground w-28 shrink-0">{label}</label>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => onChange(parseFloat(e.target.value))}
        className="flex-1 h-1.5 bg-muted rounded-full appearance-none cursor-pointer accent-primary"
      />
      <input
        type="number"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={(e) => {
          const v = parseFloat(e.target.value);
          if (!isNaN(v) && v >= min && v <= max) onChange(v);
        }}
        className="w-16 text-sm text-center border border-input rounded-md px-1.5 py-0.5 bg-background"
      />
    </div>
  );
}

export function ChatParams() {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const samplingParams = useChatStore((s) => s.samplingParams);
  const setSamplingParams = useChatStore((s) => s.setSamplingParams);
  const resetSamplingParams = useChatStore((s) => s.resetSamplingParams);

  const handleChange = (key: keyof SamplingParams) => (value: number) => {
    setSamplingParams({ [key]: value });
  };

  return (
    <div className="border-t bg-muted/10">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className={cn(
          'flex items-center gap-2 px-4 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors w-full',
          expanded && 'border-b'
        )}
      >
        <Settings2 className="w-4 h-4" />
        <span>{t('chat.params.title')}</span>
        {expanded ? <ChevronUp className="w-3.5 h-3.5 ml-auto" /> : <ChevronDown className="w-3.5 h-3.5 ml-auto" />}
      </button>

      {expanded && (
        <div className="px-4 py-3 space-y-3">
          <ParamSlider
            label={t('chat.params.temperature')}
            value={samplingParams.temperature}
            min={0}
            max={2}
            step={0.05}
            onChange={handleChange('temperature')}
          />
          <ParamSlider
            label={t('chat.params.topP')}
            value={samplingParams.topP}
            min={0}
            max={1}
            step={0.05}
            onChange={handleChange('topP')}
          />
          <ParamSlider
            label={t('chat.params.topK')}
            value={samplingParams.topK}
            min={0}
            max={200}
            step={1}
            onChange={handleChange('topK')}
          />
          <ParamSlider
            label={t('chat.params.maxTokens')}
            value={samplingParams.maxTokens}
            min={64}
            max={8192}
            step={64}
            onChange={handleChange('maxTokens')}
          />
          <ParamSlider
            label={t('chat.params.repeatPenalty')}
            value={samplingParams.repeatPenalty}
            min={0}
            max={2}
            step={0.05}
            onChange={handleChange('repeatPenalty')}
          />

          <div className="flex justify-end pt-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={resetSamplingParams}
              className="text-xs gap-1.5"
            >
              <RotateCcw className="w-3 h-3" />
              {t('chat.params.reset')}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
