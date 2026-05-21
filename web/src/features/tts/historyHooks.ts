import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  fetchTTSHistory,
  createTTSHistory,
  toggleTTSFavourite,
  deleteTTSHistory,
  type TTSHistoryListParams,
} from './api';

const TTS_HISTORY_KEY = ['tts-history'];

export function useTTSHistory(params?: TTSHistoryListParams) {
  return useQuery({
    queryKey: [...TTS_HISTORY_KEY, params],
    queryFn: () => fetchTTSHistory(params),
  });
}

export function useCreateTTSHistory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (formData: FormData) => createTTSHistory(formData),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TTS_HISTORY_KEY });
    },
  });
}

export function useToggleTTSFavourite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, favourite }: { id: string; favourite: boolean }) =>
      toggleTTSFavourite(id, favourite),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TTS_HISTORY_KEY });
    },
  });
}

export function useDeleteTTSHistory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteTTSHistory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: TTS_HISTORY_KEY });
    },
  });
}
