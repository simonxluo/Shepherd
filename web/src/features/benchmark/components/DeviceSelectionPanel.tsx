import { useTranslation } from 'react-i18next';
import { Monitor } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

interface DeviceSelectionPanelProps {
  availableDevices: string[];
  selectedDeviceIndices: number[];
  mainGpu: number;
  onDeviceSelectionChange: (indices: number[]) => void;
  onMainGpuChange: (gpu: number) => void;
}

export function DeviceSelectionPanel({
  availableDevices,
  selectedDeviceIndices,
  mainGpu,
  onDeviceSelectionChange,
  onMainGpuChange,
}: DeviceSelectionPanelProps) {
  const { t } = useTranslation();

  if (availableDevices.length === 0) {
    return (
      <div className="text-xs text-muted-foreground italic py-1">
        {t('benchmark.llamaCppVersion')}
      </div>
    );
  }

  const handleToggleDevice = (index: number) => {
    if (selectedDeviceIndices.includes(index)) {
      onDeviceSelectionChange(selectedDeviceIndices.filter(i => i !== index));
    } else {
      onDeviceSelectionChange([...selectedDeviceIndices, index].sort((a, b) => a - b));
    }
  };

  const handleSelectAll = () => {
    onDeviceSelectionChange(availableDevices.map((_, i) => i));
  };

  const handleDeselectAll = () => {
    onDeviceSelectionChange([]);
  };

  const getDeviceId = (device: string): string => {
    const colonIdx = device.indexOf(':');
    return colonIdx >= 0 ? device.substring(0, colonIdx).trim() : device.trim();
  };

  return (
    <div className="flex items-start gap-3">
      {/* Header & actions */}
      <div className="flex items-center gap-2 flex-shrink-0 pt-0.5">
        <Monitor className="w-3.5 h-3.5 text-muted-foreground" />
        <span className="text-xs font-medium text-foreground">
          {t('benchmark.devices')}
        </span>
        <span className="text-xs text-muted-foreground">
          ({selectedDeviceIndices.length}/{availableDevices.length})
        </span>
        <Button variant="ghost" size="sm" className="h-5 px-1.5 text-xs" onClick={handleSelectAll}>
          {t('benchmark.selectAll')}
        </Button>
        <Button variant="ghost" size="sm" className="h-5 px-1.5 text-xs" onClick={handleDeselectAll}>
          {t('benchmark.deselectAll')}
        </Button>
      </div>

      {/* Device checkboxes */}
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 flex-1">
        {availableDevices.map((device, index) => (
          <label
            key={index}
            className="flex items-center gap-1.5 cursor-pointer text-xs hover:text-foreground text-muted-foreground"
          >
            <Checkbox
              checked={selectedDeviceIndices.includes(index)}
              onCheckedChange={() => handleToggleDevice(index)}
              className="h-3.5 w-3.5"
            />
            <span title={device}>{device}</span>
          </label>
        ))}
      </div>

      {/* Main GPU selector */}
      {availableDevices.length >= 2 && (
        <div className="flex items-center gap-1.5 flex-shrink-0">
          <span className="text-xs text-muted-foreground whitespace-nowrap">
            {t('benchmark.mainGpu')}:
          </span>
          <Select
            value={String(mainGpu)}
            onValueChange={(v) => onMainGpuChange(Number(v))}
          >
            <SelectTrigger className="h-6 text-xs w-auto min-w-[80px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {availableDevices.map((device, index) => (
                <SelectItem key={index} value={String(index)}>
                  {getDeviceId(device)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}
    </div>
  );
}
