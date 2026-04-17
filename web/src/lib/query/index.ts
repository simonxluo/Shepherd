import { QueryClient } from '@tanstack/react-query';

/**
 * React Query 客户端配置
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 数据在 30 秒内视为新鲜
      staleTime: 30 * 1000,
      // 窗口重新聚焦时不自动重新获取
      refetchOnWindowFocus: false,
      // 失败时重试一次
      retry: 1,
      // 缓存时间 5 分钟
      gcTime: 5 * 60 * 1000,
    },
    mutations: {
      // 失败时重试一次
      retry: 1,
    },
  },
});
