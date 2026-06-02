import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { BACKEND_LABELS } from '@/features/creative/hooks';

interface BackendSelectProps {
  value: string;
  onChange: (id: string) => void;
  availableIds?: string[];
}

const ALL_BACKENDS = ['llamacpp', 'vllm', 'vllmomni'] as const;

export function BackendSelect({ value, onChange, availableIds }: BackendSelectProps) {
  const ids = availableIds && availableIds.length > 0 ? availableIds : [...ALL_BACKENDS];

  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="h-8 text-xs w-[180px]">
        <SelectValue placeholder="Select backend" />
      </SelectTrigger>
      <SelectContent>
        {ids.map((id) => (
          <SelectItem key={id} value={id}>
            {BACKEND_LABELS[id] || id}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
