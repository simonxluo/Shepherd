import { QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { useCallback } from 'react';
import { queryClient } from './lib/query';
import { MainLayout } from './components/layout/MainLayout';
import { DashboardPage } from './pages/dashboard';
import { ModelsPage } from './pages/models';
import { DownloadsPage } from './pages/downloads';
import { ChatPage } from './pages/chat';
import { ClusterPage } from './pages/cluster';
import { LogsPage } from './pages/logs';
import { MultimodalPage } from './pages/multimodal';
import { SettingsPage } from './pages/settings';
import { useSSEConnection } from './hooks/useSSEConnection';
import type { SSEEvent } from './types';
import type { UnifiedNode } from './types/node';
import { AlertDialogProvider } from './providers/AlertDialog';
import { AlertDialog } from './components/ui/alert-dialog';
import { Toaster } from './components/ui/sonner';
import { ErrorBoundary } from './components/ErrorBoundary';

import 'highlight.js/styles/github-dark.css';

function AppContent() {
  const handleSSEMessage = useCallback((event: MessageEvent) => {
    try {
      const data: SSEEvent = JSON.parse(event.data);

      switch (data.type) {
        case 'modelLoad':
        case 'modelLoadStart':
        case 'modelStop':
        case 'modelSlots':
          queryClient.invalidateQueries({ queryKey: ['models'] });
          break;
        case 'download_progress':
        case 'download_status':
          queryClient.invalidateQueries({ queryKey: ['downloads'] });
          break;
        case 'clientRegistered':
        case 'clientDisconnected':
          queryClient.invalidateQueries({ queryKey: ['clients'] });
          queryClient.invalidateQueries({ queryKey: ['cluster'] });
          break;
        case 'clientResourcesUpdated': {
          const clientData = data.data as { clientId: string; node: UnifiedNode };
          const { clientId, node } = clientData;

          queryClient.setQueryData(['cluster', 'clients', clientId], node);

          queryClient.setQueryData(['cluster', 'clients'], (old: unknown) => {
            if (Array.isArray(old)) {
              return old.map((client: Record<string, unknown>) =>
                client.id === clientId
                  ? { ...node, lastSeen: new Date().toISOString() }
                  : client
              );
            }
            return old;
          });
          break;
        }
        case 'taskUpdate':
          queryClient.invalidateQueries({ queryKey: ['tasks'] });
          queryClient.invalidateQueries({ queryKey: ['cluster'] });
          break;
        case 'systemStatus':
          queryClient.invalidateQueries({ queryKey: ['system'] });
          break;
      }
    } catch (error) {
      console.error('Failed to parse SSE event:', error);
    }
  }, []);

  const handleSSEOpen = useCallback((isReconnect: boolean) => {
    if (isReconnect) {
      queryClient.invalidateQueries({ queryKey: ['models'] });
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
      queryClient.invalidateQueries({ queryKey: ['clients'] });
      queryClient.invalidateQueries({ queryKey: ['cluster'] });
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['system'] });
      queryClient.invalidateQueries({ queryKey: ['nodes'] });
    }
  }, []);

  useSSEConnection({
    url: '/events',
    onMessage: handleSSEMessage,
    onOpen: handleSSEOpen,
  });

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<MainLayout />}>
          <Route index element={<DashboardPage />} />
          <Route path="models" element={<ModelsPage />} />
          <Route path="downloads" element={<DownloadsPage />} />
          <Route path="chat" element={<ChatPage />} />
          <Route path="cluster" element={<ClusterPage />} />
          <Route path="logs" element={<LogsPage />} />
          <Route path="multimodal" element={<MultimodalPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ErrorBoundary>
        <AlertDialogProvider>
          <AppContent />
          <AlertDialog />
          <Toaster />
        </AlertDialogProvider>
      </ErrorBoundary>
    </QueryClientProvider>
  );
}

export default App;
