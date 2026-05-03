import { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogAction,
  AlertDialogCancel,
} from '@/components/ui/alert-dialog';

export interface AlertDialogOptions {
  title: string;
  description: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'default' | 'destructive';
}

interface AlertDialogContextValue {
  confirm: (options: AlertDialogOptions) => Promise<boolean>;
}

const AlertDialogContext = createContext<AlertDialogContextValue | null>(null);

export function AlertDialogProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AlertDialogOptions | null>(null);
  const [resolver, setResolver] = useState<((value: boolean) => void) | null>(null);

  const confirm = useCallback((options: AlertDialogOptions): Promise<boolean> => {
    return new Promise((resolve) => {
      setState(options);
      setResolver(() => resolve);
    });
  }, []);

  const handleResolve = useCallback((value: boolean) => {
    resolver?.(value);
    setState(null);
    setResolver(null);
  }, [resolver]);

  const handleOpenChange = useCallback((open: boolean) => {
    if (!open) {
      handleResolve(false);
    }
  }, [handleResolve]);

  return (
    <AlertDialogContext.Provider value={{ confirm }}>
      {children}
      <AlertDialog open={state !== null} onOpenChange={handleOpenChange}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{state?.title}</AlertDialogTitle>
            <AlertDialogDescription>{state?.description}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{state?.cancelText || 'Cancel'}</AlertDialogCancel>
            <AlertDialogAction
              variant={state?.variant === 'destructive' ? 'destructive' : 'default'}
              onClick={() => handleResolve(true)}
            >
              {state?.confirmText || 'Confirm'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </AlertDialogContext.Provider>
  );
}

export function useAlertDialog() {
  const context = useContext(AlertDialogContext);
  if (!context) {
    throw new Error('useAlertDialog must be used within AlertDialogProvider');
  }
  return context;
}
