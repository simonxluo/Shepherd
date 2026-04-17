import { QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { queryClient } from './lib/query';
import { MainLayout } from './components/layout/MainLayout';
import { DashboardPage } from './pages/dashboard';
import { ModelsPage } from './pages/models';
import { DownloadsPage } from './pages/downloads';
import { ChatPage } from './pages/chat';
import { ClusterPage } from './pages/cluster';
import { LogsPage } from './pages/logs';
import { SettingsPage } from './pages/settings';
import { useSSE } from './hooks/useSSE';
import { AlertDialogProvider } from './providers/AlertDialog';
import { AlertDialog } from './components/ui/alert-dialog';
import { Toaster } from './components/ui/toaster';
import { WebSocketProvider } from './providers/WebSocketProvider';

import 'highlight.js/styles/github-dark.css';

function AppContent() {
  useSSE({
    onMessage: (event) => {
      console.log('SSE Event:', event);
    },
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
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <WebSocketProvider
        autoConnect={false}
        options={{
          maxReconnectAttempts: 5,
          initialReconnectDelay: 1000,
          heartbeatInterval: 30000,
        }}
        onError={(error) => console.error('WebSocket error:', error)}
      >
        <AlertDialogProvider>
          <AppContent />
          <AlertDialog />
          <Toaster />
        </AlertDialogProvider>
      </WebSocketProvider>
    </QueryClientProvider>
  );
}

export default App;
