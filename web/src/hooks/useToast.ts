import { toast as sonnerToast } from 'sonner';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

const variantMap = {
  success: sonnerToast.success,
  error: sonnerToast.error,
  warning: sonnerToast.warning,
  info: sonnerToast.info,
} as const;

export function useToast() {
  const showToast = (type: ToastType, title: string, description?: string, duration?: number) => {
    variantMap[type](title, { description, duration });
  };

  return {
    toast: showToast,
    success: (title: string, description?: string, duration?: number) => showToast('success', title, description, duration),
    error: (title: string, description?: string, duration?: number) => showToast('error', title, description, duration),
    warning: (title: string, description?: string, duration?: number) => showToast('warning', title, description, duration),
    info: (title: string, description?: string, duration?: number) => showToast('info', title, description, duration),
  };
}
