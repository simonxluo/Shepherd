import { QueryClient } from '@tanstack/react-query';
import { useToastStore } from '@/stores/toast';
import { APIError } from '@/lib/api/client';

function handleGlobalMutationError(error: Error) {
  if (error instanceof APIError && error.handled) {
    return;
  }

  const { addToast } = useToastStore.getState();

  let title = '操作失败';
  let description = error.message || '未知错误';

  if (error instanceof APIError) {
    switch (error.code) {
      case 'INVALID_REQUEST':
        title = '请求参数错误';
        break;
      case 'PERMISSION_DENIED':
        title = '权限不足';
        break;
      case 'NOT_AUTHENTICATED':
        title = '未认证';
        break;
      case 'NOT_FOUND':
      case 'NODE_NOT_FOUND':
        title = '资源未找到';
        break;
      case 'CONFLICT':
        title = '操作冲突';
        break;
      case 'RESOURCE_EXHAUSTED':
        title = '资源不足';
        break;
      case 'TIMEOUT':
        title = '请求超时';
        break;
      default:
        if (error.status === 0 || error.status >= 500) {
          title = '服务器错误';
          description = '服务暂时不可用，请稍后重试';
        }
    }
  }

  addToast({ type: 'error', title, description, duration: 5000 });
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
      retry: 1,
      onError: handleGlobalMutationError,
    },
  },
});
