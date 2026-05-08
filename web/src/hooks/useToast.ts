import { toast as sonnerToast } from 'sonner';

type ToastType = 'success' | 'error' | 'warning' | 'info';

function showToast(type: ToastType, title: string, description?: string, duration?: number) {
  switch (type) {
    case 'success':
      sonnerToast.success(title, { description, duration });
      break;
    case 'error':
      sonnerToast.error(title, { description, duration });
      break;
    case 'warning':
      sonnerToast.warning(title, { description, duration });
      break;
    case 'info':
      sonnerToast.info(title, { description, duration });
      break;
  }
}

export const toast = {
  success: (title: string, description?: string, duration?: number) => showToast('success', title, description, duration),
  error: (title: string, description?: string, duration?: number) => showToast('error', title, description, duration),
  warning: (title: string, description?: string, duration?: number) => showToast('warning', title, description, duration),
  info: (title: string, description?: string, duration?: number) => showToast('info', title, description, duration),
};
