import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { BACKEND_LABELS, type LoadedModel } from '../hooks';

interface ModelSelectProps {
  models: LoadedModel[];
  value: string;
  onValueChange: (value: string) => void;
  placeholder: string;
  label?: string;
  showBackend?: boolean;
}

/**
 * Shared model selection dropdown for multimodal panels
 */
export function ModelSelect({
  models,
  value,
  onValueChange,
  placeholder,
  label,
  showBackend = false,
}: ModelSelectProps) {
  return (
    <div>
      {label && (
        <label className="block text-sm font-medium mb-1.5">{label}</label>
      )}
      <Select value={value} onValueChange={onValueChange}>
        <SelectTrigger className="w-full px-3 py-2 border rounded-md bg-background text-sm focus:ring-2 focus:ring-ring focus:border-transparent">
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {models.map((m) => (
            <SelectItem key={m.id} value={m.alias || m.name}>
              {m.alias || m.name}
              {showBackend && m.backendType && (
                <span className="ml-2 text-xs text-muted-foreground">
                  ({BACKEND_LABELS[m.backendType] || m.backendType})
                </span>
              )}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
