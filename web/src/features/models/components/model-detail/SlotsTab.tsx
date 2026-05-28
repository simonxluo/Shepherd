import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { getModelSlots, type SlotInfo } from '@/features/models/model-detail-api';

interface SlotsTabProps {
  modelId: string;
  isLoaded: boolean;
}

export function SlotsTab({ modelId, isLoaded }: SlotsTabProps) {
  const { t } = useTranslation();
  const [slots, setSlots] = useState<SlotInfo[]>([]);
  const [selectedSlot, setSelectedSlot] = useState<number>(0);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState('');

  const loadSlots = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getModelSlots(modelId);
      setSlots(data);
      if (data.length > 0 && selectedSlot >= data.length) {
        setSelectedSlot(0);
      }
      setStatus('');
    } catch {
      setStatus(t('modelDetail.slots.loadError', '加载失败'));
    } finally {
      setLoading(false);
    }
  }, [modelId, selectedSlot, t]);

  useEffect(() => {
    if (isLoaded) {
      loadSlots();
    }
  }, [isLoaded, loadSlots]);

  if (!isLoaded) {
    return (
      <div className="flex items-center justify-center p-8 text-muted-foreground text-sm">
        {t('modelDetail.slots.notLoaded', '模型未加载，无法查看 Slots')}
      </div>
    );
  }

  const currentSlot = slots[selectedSlot];

  return (
    <div className="flex flex-col gap-3 p-4 h-full">
      {/* Controls */}
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={loadSlots} disabled={loading}>
          {t('modelDetail.slots.refresh', '刷新')}
        </Button>
        {slots.length > 0 && (
          <select
            className="h-8 rounded-md border border-input bg-background px-2 text-sm"
            value={selectedSlot}
            onChange={(e) => setSelectedSlot(Number(e.target.value))}
          >
            {slots.map((slot, idx) => (
              <option key={slot.id} value={idx}>
                Slot {slot.id}
              </option>
            ))}
          </select>
        )}
        {status && <span className="text-xs text-destructive ml-2">{status}</span>}
      </div>

      {/* Slot viewer */}
      {loading ? (
        <div className="flex items-center justify-center py-8">
          <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
        </div>
      ) : currentSlot ? (
        <pre className="flex-1 min-h-[250px] overflow-auto rounded-md border border-input bg-muted/30 p-3 text-xs font-mono whitespace-pre-wrap">
          {JSON.stringify(currentSlot, null, 2)}
        </pre>
      ) : (
        <div className="flex items-center justify-center py-8 text-muted-foreground text-sm">
          {t('modelDetail.slots.noSlots', '暂无 Slot 数据')}
        </div>
      )}
    </div>
  );
}
