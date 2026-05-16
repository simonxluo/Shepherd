import { useState } from 'react';
import { Save, Trash2, ChevronDown, ChevronRight, Loader2 } from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import type { ModelLoadConfigEntry } from '@/features/models/config';
import { toast } from '@/hooks/useToast';

interface SaveConfigDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (name: string) => void;
  existingConfigs: ModelLoadConfigEntry[];
  onDeleteConfig: (name: string) => void;
  modelName: string;
  isSaving?: boolean;
}

export function SaveConfigDialog({
  isOpen,
  onClose,
  onConfirm,
  existingConfigs,
  onDeleteConfig,
  modelName,
  isSaving = false,
}: SaveConfigDialogProps) {
  const [name, setName] = useState('');
  const [manageOpen, setManageOpen] = useState(false);
  const [deletingName, setDeletingName] = useState<string | null>(null);

  const isDuplicate = existingConfigs.some(c => c.name === name.trim());

  const handleSave = () => {
    const trimmed = name.trim();
    if (!trimmed) {
      toast.error('请输入配置名称');
      return;
    }
    onConfirm(trimmed);
    setName('');
    setManageOpen(false);
  };

  const handleDelete = (configName: string) => {
    setDeletingName(configName);
    onDeleteConfig(configName);
    setDeletingName(null);
  };

  const handleClose = () => {
    setName('');
    setManageOpen(false);
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) handleClose(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>保存配置 — {modelName}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="输入配置名称"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  handleSave();
                }
              }}
            />
            {isDuplicate && (
              <p className="text-xs text-amber-600">将覆盖已有配置「{name.trim()}」</p>
            )}
          </div>

          {existingConfigs.length > 0 && (
            <Collapsible open={manageOpen} onOpenChange={setManageOpen}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" size="sm" className="w-full justify-start text-muted-foreground">
                  {manageOpen ? <ChevronDown className="w-4 h-4 mr-1" /> : <ChevronRight className="w-4 h-4 mr-1" />}
                  管理已有配置 ({existingConfigs.length})
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div className="mt-2 space-y-1 max-h-48 overflow-y-auto">
                  {existingConfigs.map((config) => (
                    <div
                      key={config.name}
                      className="flex items-center justify-between px-3 py-2 rounded-md bg-muted/50"
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate">{config.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {config.updatedAt ? new Date(config.updatedAt).toLocaleString() : ''}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="ml-2 h-8 w-8 p-0 text-muted-foreground hover:text-destructive"
                        onClick={() => handleDelete(config.name)}
                        disabled={deletingName === config.name}
                      >
                        {deletingName === config.name ? (
                          <Loader2 className="w-4 h-4 animate-spin" />
                        ) : (
                          <Trash2 className="w-4 h-4" />
                        )}
                      </Button>
                    </div>
                  ))}
                </div>
              </CollapsibleContent>
            </Collapsible>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose} disabled={isSaving}>
            取消
          </Button>
          <Button onClick={handleSave} disabled={isSaving || !name.trim()}>
            {isSaving ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                保存中...
              </>
            ) : (
              <>
                <Save className="w-4 h-4" />
                保存
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
