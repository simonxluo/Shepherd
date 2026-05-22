import { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Search, Star } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import type { Model } from '@/types';

interface ModelListPanelProps {
  models: Model[];
  selectedModelId: string | undefined;
  onSelectModel: (modelId: string) => void;
}

export function ModelListPanel({ models, selectedModelId, onSelectModel }: ModelListPanelProps) {
  const { t } = useTranslation();
  const [search, setSearch] = useState('');

  const filteredModels = useMemo(() => {
    if (!search) return models;
    const lowerSearch = search.toLowerCase();
    return models.filter(
      m =>
        m.name.toLowerCase().includes(lowerSearch) ||
        (m.alias && m.alias.toLowerCase().includes(lowerSearch))
    );
  }, [models, search]);

  return (
    <div className="w-64 flex-shrink-0 border-r border-border flex flex-col bg-card">
      {/* Search */}
      <div className="p-3 border-b border-border">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
          <Input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t('benchmark.searchModels')}
            className="pl-8 h-8 text-sm"
          />
        </div>
      </div>

      {/* Model list */}
      <div className="flex-1 overflow-y-auto">
        {filteredModels.length === 0 ? (
          <div className="text-center py-8 text-sm text-muted-foreground">
            {t('benchmark.noModels')}
          </div>
        ) : (
          <div className="py-1">
            {filteredModels.map((model) => (
              <button
                key={model.id}
                onClick={() => onSelectModel(model.id)}
                className={cn(
                  'w-full text-left px-3 py-2 text-sm transition-colors',
                  'hover:bg-accent/50',
                  selectedModelId === model.id && 'bg-accent text-accent-foreground'
                )}
              >
                <div className="flex items-center gap-1.5">
                  {model.favourite && (
                    <Star className="w-3 h-3 text-yellow-500 fill-yellow-500 flex-shrink-0" />
                  )}
                  <span className="truncate font-medium">
                    {model.alias || model.name}
                  </span>
                </div>
                <div className="text-xs text-muted-foreground truncate mt-0.5">
                  {model.metadata?.quantization && (
                    <span className="mr-2">{model.metadata.quantization}</span>
                  )}
                  {model.metadata?.parameters && (
                    <span>{(model.metadata.parameters / 1e9).toFixed(1)}B</span>
                  )}
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
