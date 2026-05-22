import { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { ChevronDown, ChevronRight, RotateCcw } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import type { BenchmarkParam } from '@/types';
import { getFieldName } from '../lib/commandBuilder';

// Group display name mapping
const GROUP_KEYS: Record<string, string> = {
  'page.params.group.basic': 'basic',
  'page.params.group.test_data': 'test_data',
  'page.params.group.performance': 'performance',
  'page.params.group.cache_storage': 'cache_storage',
  'page.params.group.advanced': 'advanced',
};

interface ParamGroup {
  key: string;
  label: string;
  order: number;
  params: BenchmarkParam[];
}

interface BenchmarkParamsModalProps {
  isOpen: boolean;
  onClose: () => void;
  params: BenchmarkParam[];
  enabledMap: Record<string, boolean>;
  valueMap: Record<string, string>;
  onConfirm: (enabledMap: Record<string, boolean>, valueMap: Record<string, string>) => void;
}

/**
 * Wrapper that remounts the inner content each time the dialog opens,
 * ensuring fresh state initialization without calling setState in effects.
 */
export function BenchmarkParamsModal({
  isOpen,
  onClose,
  params,
  enabledMap,
  valueMap,
  onConfirm,
}: BenchmarkParamsModalProps) {
  if (!isOpen) return null;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose(); }}>
      <DialogContent className="sm:max-w-2xl max-h-[85vh] flex flex-col p-0">
        <BenchmarkParamsModalContent
          params={params}
          initialEnabledMap={enabledMap}
          initialValueMap={valueMap}
          onConfirm={onConfirm}
          onClose={onClose}
        />
      </DialogContent>
    </Dialog>
  );
}

interface ModalContentProps {
  params: BenchmarkParam[];
  initialEnabledMap: Record<string, boolean>;
  initialValueMap: Record<string, string>;
  onConfirm: (enabledMap: Record<string, boolean>, valueMap: Record<string, string>) => void;
  onClose: () => void;
}

function BenchmarkParamsModalContent({
  params,
  initialEnabledMap,
  initialValueMap,
  onConfirm,
  onClose,
}: ModalContentProps) {
  const { t } = useTranslation();

  // Local state initialized from props (component remounts on each open)
  const [localEnabled, setLocalEnabled] = useState<Record<string, boolean>>(() => ({ ...initialEnabledMap }));
  const [localValues, setLocalValues] = useState<Record<string, string>>(() => ({ ...initialValueMap }));
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(() => {
    const collapsed = new Set<string>();
    params.forEach(p => {
      if (p.groupCollapsed && p.group) {
        collapsed.add(p.group);
      }
    });
    return collapsed;
  });

  // Organize params into groups
  const groups = useMemo((): ParamGroup[] => {
    const groupMap = new Map<string, ParamGroup>();

    for (const param of params) {
      const groupKey = param.group || 'page.params.group.basic';
      if (!groupMap.has(groupKey)) {
        const translationKey = GROUP_KEYS[groupKey] || 'basic';
        groupMap.set(groupKey, {
          key: groupKey,
          label: t(`benchmark.groups.${translationKey}`, translationKey),
          order: param.groupOrder || 0,
          params: [],
        });
      }
      groupMap.get(groupKey)!.params.push(param);
    }

    // Sort params within each group
    for (const group of groupMap.values()) {
      group.params.sort((a, b) => (a.sort || 0) - (b.sort || 0));
    }

    // Sort groups by order
    return Array.from(groupMap.values()).sort((a, b) => a.order - b.order);
  }, [params, t]);

  const enabledCount = useMemo(() => {
    return Object.values(localEnabled).filter(Boolean).length;
  }, [localEnabled]);

  const toggleGroup = (groupKey: string) => {
    setCollapsedGroups(prev => {
      const next = new Set(prev);
      if (next.has(groupKey)) {
        next.delete(groupKey);
      } else {
        next.add(groupKey);
      }
      return next;
    });
  };

  const handleToggleEnabled = (fieldName: string) => {
    setLocalEnabled(prev => ({ ...prev, [fieldName]: !prev[fieldName] }));
  };

  const handleValueChange = (fieldName: string, value: string) => {
    setLocalValues(prev => ({ ...prev, [fieldName]: value }));
  };

  const handleReset = () => {
    const newEnabled: Record<string, boolean> = {};
    const newValues: Record<string, string> = {};
    for (const p of params) {
      const fieldName = getFieldName(p);
      newEnabled[fieldName] = p.defaultEnabled !== false;
      newValues[fieldName] = p.defaultValue || '';
    }
    setLocalEnabled(newEnabled);
    setLocalValues(newValues);
  };

  const handleConfirm = () => {
    onConfirm(localEnabled, localValues);
    onClose();
  };

  const renderParamField = (param: BenchmarkParam) => {
    const fieldName = getFieldName(param);
    const isEnabled = localEnabled[fieldName] ?? true;
    const value = localValues[fieldName] ?? param.defaultValue ?? '';

    // Get values list, handling both string[] and {value, label}[] formats
    const valuesList = param.values?.filter(v => {
      if (typeof v === 'string') return v !== '';
      return v.value !== '';
    }) ?? [];

    return (
      <div
        key={fieldName}
        className={cn(
          'flex items-start gap-2 p-2 rounded-md transition-colors',
          !isEnabled && 'opacity-50'
        )}
      >
        {/* Enable checkbox */}
        <Checkbox
          checked={isEnabled}
          onCheckedChange={() => handleToggleEnabled(fieldName)}
          className="mt-1.5 flex-shrink-0"
        />

        {/* Param content */}
        <div className="flex-1 min-w-0 space-y-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-foreground">
              {param.name}
            </span>
            {param.abbreviation && (
              <span className="text-xs text-muted-foreground">
                ({param.abbreviation})
              </span>
            )}
            {param.description && (
              <span
                className="text-xs text-muted-foreground cursor-help"
                title={param.description}
              >
                ?
              </span>
            )}
          </div>

          {/* Value input */}
          {valuesList.length > 0 ? (
            <Select
              value={value || undefined}
              onValueChange={(v) => handleValueChange(fieldName, v)}
              disabled={!isEnabled}
            >
              <SelectTrigger className="h-7 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {valuesList.map((v) => {
                  const optValue = typeof v === 'string' ? v : v.value;
                  const optLabel = typeof v === 'string' ? v : v.label;
                  return (
                    <SelectItem key={optValue} value={optValue}>
                      {optLabel}
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          ) : param.type === 'LOGIC' ? (
            <Select
              value={value || '0'}
              onValueChange={(v) => handleValueChange(fieldName, v)}
              disabled={!isEnabled}
            >
              <SelectTrigger className="h-7 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1">Enabled</SelectItem>
                <SelectItem value="0">Disabled</SelectItem>
              </SelectContent>
            </Select>
          ) : (
            <Input
              type={param.type === 'INTEGER' || param.type === 'FLOAT' ? 'number' : 'text'}
              value={value}
              onChange={(e) => handleValueChange(fieldName, e.target.value)}
              disabled={!isEnabled}
              className="h-7 text-xs"
            />
          )}
        </div>
      </div>
    );
  };

  return (
    <>
      <DialogHeader className="px-4 py-3 border-b border-border flex-shrink-0">
        <DialogTitle className="flex items-center justify-between">
          <span>{t('benchmark.paramsTitle')}</span>
          <span className="text-sm font-normal text-muted-foreground">
            {t('benchmark.paramsCount', { enabled: enabledCount, total: params.length })}
          </span>
        </DialogTitle>
      </DialogHeader>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {groups.map((group) => {
          const isCollapsed = collapsedGroups.has(group.key);

          return (
            <div key={group.key} className="border border-border rounded-lg overflow-hidden">
              {/* Group header */}
              <button
                onClick={() => toggleGroup(group.key)}
                className="w-full flex items-center gap-2 px-3 py-2 bg-muted/50 hover:bg-muted transition-colors text-left"
              >
                {isCollapsed ? (
                  <ChevronRight className="w-4 h-4 text-muted-foreground" />
                ) : (
                  <ChevronDown className="w-4 h-4 text-muted-foreground" />
                )}
                <span className="text-sm font-semibold text-foreground">
                  {group.label}
                </span>
                <span className="text-xs text-muted-foreground ml-auto">
                  {group.params.filter(p => localEnabled[getFieldName(p)]).length}/{group.params.length}
                </span>
              </button>

              {/* Group params */}
              {!isCollapsed && (
                <div className="p-2 grid grid-cols-1 md:grid-cols-2 gap-1">
                  {group.params.map(renderParamField)}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Footer */}
      <DialogFooter className="px-4 py-3 border-t border-border flex-shrink-0">
        <Button variant="ghost" size="sm" onClick={handleReset}>
          <RotateCcw className="w-4 h-4 mr-1.5" />
          {t('benchmark.reset')}
        </Button>
        <div className="flex-1" />
        <Button variant="outline" size="sm" onClick={onClose}>
          {t('benchmark.cancel')}
        </Button>
        <Button size="sm" onClick={handleConfirm}>
          {t('benchmark.confirm')}
        </Button>
      </DialogFooter>
    </>
  );
}
