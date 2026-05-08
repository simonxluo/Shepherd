import { useQuery } from '@tanstack/react-query';
import { systemApi } from '@/lib/api/system';

export function useServerInfo() {
  return useQuery({
    queryKey: ['serverInfo'],
    queryFn: async () => {
      const res = await systemApi.getInfo();
      if (res.success && res.data) {
        return res.data;
      }
      return null;
    },
    staleTime: 5 * 60 * 1000,
  });
}
