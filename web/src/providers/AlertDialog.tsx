import { createContext, useContext, useState, useCallback, ReactNode } from 'react';

export interface AlertDialogOptions {
  title: string;
  description: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'default' | 'destructive';
}

interface AlertDialogContextValue {
  confirm: (options: AlertDialogOptions) => Promise<boolean>;
  state: AlertDialogOptions | null;
  resolve: (value: boolean) => void;
  close: () => void;
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

  const close = useCallback(() => {
    handleResolve(false);
  }, [handleResolve]);

  return (
    <AlertDialogContext.Provider value={{ confirm, state, resolve: handleResolve, close }}>
      {children}
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
