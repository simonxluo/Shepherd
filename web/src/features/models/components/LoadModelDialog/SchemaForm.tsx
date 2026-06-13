import { useState, useMemo, Fragment } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { ParamDef } from '@/lib/api/backends';

interface SchemaFormProps {
  params: ParamDef[];
  values: Record<string, unknown>;
  enabled: Record<string, boolean>;
  onChange: (key: string, value: unknown) => void;
  onToggle: (key: string, on: boolean) => void;
}

export function SchemaForm({ params, values, enabled, onChange, onToggle }: SchemaFormProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);

  const groups = useMemo(() => {
    const map = new Map<string, ParamDef[]>();
    for (const p of params) {
      const g = p.group || 'other';
      if (!map.has(g)) map.set(g, []);
      map.get(g)!.push(p);
    }
    return map;
  }, [params]);

  const hasAdvanced = params.some((p) => p.advanced);

  return (
    <div className="space-y-4">
      {Array.from(groups.entries()).map(([group, defs]) => {
        const visible = defs.filter((d) => !d.advanced || showAdvanced);
        if (visible.length === 0) return null;
        return (
          <fieldset key={group} className="space-y-2">
            <legend className="text-xs font-medium uppercase text-muted-foreground tracking-wider">
              {group}
            </legend>
            {visible.map((def) => (
              <ParamRow
                key={def.jsonName}
                def={def}
                value={values[def.jsonName]}
                isEnabled={enabled[def.jsonName] !== false}
                onChange={(v) => onChange(def.jsonName, v)}
                onToggle={(on) => onToggle(def.jsonName, on)}
              />
            ))}
          </fieldset>
        );
      })}
      {hasAdvanced && (
        <button
          type="button"
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
          onClick={() => setShowAdvanced(!showAdvanced)}
        >
          {showAdvanced ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          {showAdvanced ? 'Hide advanced' : 'Show advanced'}
        </button>
      )}
    </div>
  );
}

interface ParamRowProps {
  def: ParamDef;
  value: unknown;
  isEnabled: boolean;
  onChange: (value: unknown) => void;
  onToggle: (on: boolean) => void;
}

function ParamRow({ def, value, isEnabled, onChange, onToggle }: ParamRowProps) {
  return (
    <div className="flex items-center gap-2">
      <Switch
        checked={isEnabled}
        onCheckedChange={onToggle}
        className="shrink-0 scale-75"
      />
      <label className="text-sm min-w-[120px] truncate" title={def.description}>
        {def.name}
      </label>
      <div className="flex-1">
        <ParamInput def={def} value={value} disabled={!isEnabled} onChange={onChange} />
      </div>
    </div>
  );
}

interface ParamInputProps {
  def: ParamDef;
  value: unknown;
  disabled: boolean;
  onChange: (value: unknown) => void;
}

function ParamInput({ def, value, disabled, onChange }: ParamInputProps) {
  if (def.options && def.options.length > 0) {
    return (
      <Select
        value={String(value ?? '')}
        onValueChange={(v) => onChange(v)}
        disabled={disabled}
      >
        <SelectTrigger className="h-8 text-xs">
          <SelectValue placeholder="Select..." />
        </SelectTrigger>
        <SelectContent>
          {def.options.map((opt) => (
            <SelectItem key={String(opt)} value={String(opt)}>
              {String(opt)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }

  switch (def.type) {
    case 'bool':
      return (
        <Switch
          checked={Boolean(value)}
          onCheckedChange={(v) => onChange(v)}
          disabled={disabled}
        />
      );
    case 'int':
    case 'float':
      return (
        <Input
          type="number"
          className="h-8 text-xs"
          value={value != null ? String(value) : ''}
          disabled={disabled}
          min={def.min}
          max={def.max}
          step={def.type === 'float' ? 0.01 : 1}
          onChange={(e) => {
            const v = e.target.value;
            if (v === '') {
              onChange(def.type === 'int' ? 0 : 0.0);
              return;
            }
            onChange(def.type === 'int' ? parseInt(v, 10) : parseFloat(v));
          }}
        />
      );
    case 'string':
    default:
      return (
        <Input
          className="h-8 text-xs"
          value={String(value ?? '')}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
        />
      );
  }
}
