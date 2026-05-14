import { QueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { APIError } from '@/lib/api/client';
import i18n from '@/lib/i18n';

function handleGlobalMutationError(error: Error) {
  if (error instanceof APIError && error.handled) {
    return;
  }

  let title = i18n.t('error.operationFailed');
  let description = error.message || i18n.t('common.unknownError');

  if (error instanceof APIError) {
    switch (error.code) {
      case 'INVALID_REQUEST':
        title = i18n.t('error.invalidRequest');
        break;
      case 'PERMISSION_DENIED':
        title = i18n.t('error.permissionDenied');
        break;
      case 'NOT_AUTHENTICATED':
        title = i18n.t('error.notAuthenticated');
        break;
      case 'NOT_FOUND':
      case 'NODE_NOT_FOUND':
        title = i18n.t('error.notFound');
        break;
      case 'CONFLICT':
        title = i18n.t('error.conflict');
        break;
      case 'RESOURCE_EXHAUSTED':
        title = i18n.t('error.resourceExhausted');
        break;
      case 'TIMEOUT':
        title = i18n.t('error.timeout');
        break;
      default:
        if (error.status === 0 || error.status >= 500) {
          title = i18n.t('error.serverError');
          description = i18n.t('error.serverErrorDesc');
        }
    }
  }

  toast.error(title, { description, duration: 5000 });
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000,
      refetchOnWindowFocus: false,
      retry: 1,
      gcTime: 5 * 60 * 1000,
    },
    mutations: {
      retry: 0,
      onError: handleGlobalMutationError,
    },
  },
});
