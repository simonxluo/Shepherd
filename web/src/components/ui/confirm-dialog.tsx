import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';

/**
 * ConfirmDialog - 确认操作对话框
 *
 * 用于需要用户确认的危险/重要操作，如删除、卸载等。
 * 基于 shadcn Dialog 组件封装，提供简洁的 props 接口。
 *
 * @example
 * ```tsx
 * <ConfirmDialog
 *   open={showConfirm}
 *   onOpenChange={setShowConfirm}
 *   title="删除模型"
 *   description="确定要删除这个模型吗？此操作不可恢复。"
 *   confirmText="删除"
 *   variant="destructive"
 *   isLoading={isDeleting}
 *   onConfirm={handleDelete}
 * />
 * ```
 */

export interface ConfirmDialogProps {
  /** 是否打开对话框 */
  open: boolean;
  /** 打开/关闭状态变更回调 */
  onOpenChange: (open: boolean) => void;
  /** 对话框标题 */
  title: string;
  /** 对话框描述文字 */
  description?: string;
  /** 确认按钮文字，默认 "确认" */
  confirmText?: string;
  /** 取消按钮文字，默认 "取消" */
  cancelText?: string;
  /** 确认按钮变体样式 */
  variant?: 'default' | 'destructive';
  /** 是否加载中 */
  isLoading?: boolean;
  /** 点击确认回调 */
  onConfirm: () => void;
  /** 确认按钮额外 className */
  confirmClassName?: string;
  /** 取消按钮额外 className */
  cancelClassName?: string;
  /** 对话框内容区域额外 className */
  contentClassName?: string;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmText = '确认',
  cancelText = '取消',
  variant = 'default',
  isLoading = false,
  onConfirm,
  confirmClassName,
  cancelClassName,
  contentClassName,
}: ConfirmDialogProps) {
  const handleConfirm = () => {
    if (!isLoading) {
      onConfirm();
    }
  };

  const handleCancel = () => {
    if (!isLoading) {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={cn('max-w-md', contentClassName)}>
        <DialogHeader>
          <DialogTitle
            className={cn(
              variant === 'destructive' && 'text-destructive'
            )}
          >
            {title}
          </DialogTitle>
          {description && (
            <DialogDescription>{description}</DialogDescription>
          )}
        </DialogHeader>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleCancel}
            disabled={isLoading}
            className={cancelClassName}
          >
            {cancelText}
          </Button>
          <Button
            variant={variant === 'destructive' ? 'destructive' : 'default'}
            onClick={handleConfirm}
            disabled={isLoading}
            className={cn('min-w-[80px]', confirmClassName)}
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                {confirmText}
              </>
            ) : (
              confirmText
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
