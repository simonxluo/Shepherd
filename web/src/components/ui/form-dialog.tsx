import type React from 'react';
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
 * FormDialog - 表单对话框
 *
 * 用于需要用户填写表单并提交的场景，如编辑别名、创建下载等。
 * 基于 shadcn Dialog 组件封装，提供标题、内容插槽、提交/取消按钮。
 *
 * @example
 * ```tsx
 * <FormDialog
 *   open={showEdit}
 *   onOpenChange={setShowEdit}
 *   title="编辑别名"
 *   description="设置别名后，模型将以别名显示在列表中"
 *   submitText="保存"
 *   isLoading={isSaving}
 *   onSubmit={handleSave}
 * >
 *   <input ... />
 * </FormDialog>
 * ```
 */

export interface FormDialogProps {
  /** 是否打开对话框 */
  open: boolean;
  /** 打开/关闭状态变更回调 */
  onOpenChange: (open: boolean) => void;
  /** 对话框标题 */
  title: string;
  /** 对话框描述文字 */
  description?: string;
  /** 标题图标，放在标题文字前 */
  icon?: React.ReactNode;
  /** 提交按钮文字，默认 "提交" */
  submitText?: string;
  /** 取消按钮文字，默认 "取消" */
  cancelText?: string;
  /** 是否加载中 */
  isLoading?: boolean;
  /** 提交按钮是否禁用（除 isLoading 外的额外禁用条件） */
  isSubmitDisabled?: boolean;
  /** 表单提交回调，调用者需负责阻止默认行为 */
  onSubmit: (e: React.FormEvent) => void;
  /** 表单内容 */
  children: React.ReactNode;
  /** 提交按钮额外 className */
  submitClassName?: string;
  /** 取消按钮额外 className */
  cancelClassName?: string;
  /** 对话框内容区域额外 className */
  contentClassName?: string;
}

export function FormDialog({
  open,
  onOpenChange,
  title,
  description,
  icon,
  submitText = '提交',
  cancelText = '取消',
  isLoading = false,
  isSubmitDisabled = false,
  onSubmit,
  children,
  submitClassName,
  cancelClassName,
  contentClassName,
}: FormDialogProps) {
  const handleCancel = () => {
    if (!isLoading) {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className={contentClassName}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {icon}
            {title}
          </DialogTitle>
          {description && (
            <DialogDescription>{description}</DialogDescription>
          )}
        </DialogHeader>

        <form onSubmit={onSubmit}>
          <div className="p-6 space-y-4">
            {children}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={handleCancel}
              disabled={isLoading}
              className={cancelClassName}
            >
              {cancelText}
            </Button>
            <Button
              type="submit"
              disabled={isLoading || isSubmitDisabled}
              className={cn('min-w-[80px]', submitClassName)}
            >
              {isLoading ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  {submitText}...
                </>
              ) : (
                submitText
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
