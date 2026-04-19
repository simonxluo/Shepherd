import { useMemo } from 'react';
import { useToastStore } from '@/stores/toast';
import type { ToastType } from '@/stores/toast';

export function useToast() {
  const addToast = useToastStore((state) => state.addToast);

  return useMemo(() => {
    const toast = (
      type: ToastType,
      title: string,
      description?: string,
      duration?: number
    ) => {
      addToast({ type, title, description, duration });
    };

    return {
      toast,

      success: (title: string, description?: string, duration?: number) => {
        toast('success', title, description, duration);
      },

      error: (title: string, description?: string, duration?: number) => {
        toast('error', title, description, duration);
      },

      warning: (title: string, description?: string, duration?: number) => {
        toast('warning', title, description, duration);
      },

      info: (title: string, description?: string, duration?: number) => {
        toast('info', title, description, duration);
      },
    };
  }, [addToast]);
}
