import { useState, useCallback, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { benchmarksApi } from '@/lib/api/benchmarks';
import type { BenchmarkParam, BenchmarkHistoryFile, BenchmarkTask, Model } from '@/types';
import { getFieldName } from '../lib/commandBuilder';

/**
 * Fetch benchmark params list hook
 */
export function useBenchmarkParams() {
  return useQuery<BenchmarkParam[]>({
    queryKey: ['benchmark', 'params'],
    queryFn: async () => {
      const response = await benchmarksApi.getParams();
      return response.params || [];
    },
    staleTime: 30 * 60 * 1000,
  });
}

/**
 * Fetch Llama.cpp versions list hook
 */
export function useLlamaCppVersions() {
  return useQuery<Array<{ path: string; name?: string; description?: string }>>({
    queryKey: ['llamacpp', 'versions'],
    queryFn: async () => {
      const response = await benchmarksApi.getLlamaCppVersions();
      return response.data?.items || [];
    },
    staleTime: 10 * 60 * 1000,
  });
}

/**
 * Fetch benchmark history files for a model
 */
export function useBenchmarkHistory(modelId: string | undefined) {
  return useQuery<BenchmarkHistoryFile[]>({
    queryKey: ['benchmark', 'history', modelId],
    queryFn: async () => {
      if (!modelId) return [];
      const response = await benchmarksApi.listHistory(modelId);
      return response.data?.files || [];
    },
    enabled: !!modelId,
    staleTime: 30 * 1000,
  });
}

/**
 * Delete benchmark history file mutation
 */
export function useDeleteHistoryFile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (fileName: string) => {
      const response = await benchmarksApi.deleteHistoryFile(fileName);
      if (!response.success) {
        throw new Error('Failed to delete file');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'history'] });
    },
  });
}

/**
 * Create benchmark task hook
 */
export function useCreateBenchmarkTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { modelId: string; llamaBinPath: string; cmd: string; args: string[] }) => {
      const response = await benchmarksApi.create(params);
      if (!response.success) {
        throw new Error(response.error || 'Failed to create benchmark task');
      }
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'history'] });
    },
  });
}

/**
 * Fetch running benchmark tasks with smart polling
 */
export function useBenchmarkTasks() {
  return useQuery<BenchmarkTask[]>({
    queryKey: ['benchmark', 'tasks'],
    queryFn: async () => {
      const response = await benchmarksApi.listTasks();
      return response.data?.tasks || [];
    },
    refetchInterval: (query) => {
      const tasks = query.state.data;
      const hasRunning = tasks?.some(t => t.status === 'running' || t.status === 'pending');
      return hasRunning ? 3000 : false;
    },
    staleTime: 1000,
  });
}

/**
 * Cancel a benchmark task
 */
export function useCancelBenchmarkTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (taskId: string) => {
      return benchmarksApi.cancelTask(taskId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['benchmark', 'tasks'] });
    },
  });
}

/**
 * Main benchmark page state management hook
 */
export function useBenchmarkState(models: Model[]) {
  const [selectedModelId, setSelectedModelId] = useState<string | undefined>();
  const [llamaCppPath, setLlamaCppPath] = useState<string>('');
  const [selectedHistoryFile, setSelectedHistoryFile] = useState<string | null>(null);
  const [outputContent, setOutputContent] = useState<string>('');
  const [isOutputLoading, setIsOutputLoading] = useState(false);

  // Param state
  const [enabledMap, setEnabledMap] = useState<Record<string, boolean>>({});
  const [valueMap, setValueMap] = useState<Record<string, string>>({});
  const [isParamsModalOpen, setIsParamsModalOpen] = useState(false);

  // Device state
  const [availableDevices, setAvailableDevices] = useState<string[]>([]);
  const [selectedDeviceIndices, setSelectedDeviceIndices] = useState<number[]>([]);
  const [mainGpu, setMainGpu] = useState(0);

  const selectedModel = models.find(m => m.id === selectedModelId);

  // Initialize param defaults when params load
  const initializeParamDefaults = useCallback((params: BenchmarkParam[]) => {
    const newEnabled: Record<string, boolean> = {};
    const newValues: Record<string, string> = {};

    for (const p of params) {
      const fieldName = getFieldName(p);
      // defaultEnabled: undefined or true means enabled
      newEnabled[fieldName] = p.defaultEnabled !== false;
      newValues[fieldName] = p.defaultValue || '';
    }

    setEnabledMap(newEnabled);
    setValueMap(newValues);
  }, []);

  // Load history file content
  const loadHistoryFile = useCallback(async (fileName: string) => {
    setIsOutputLoading(true);
    setSelectedHistoryFile(fileName);
    try {
      const response = await benchmarksApi.getHistoryFile(fileName);
      if (response.success && response.data) {
        setOutputContent(response.data.rawOutput);
      } else {
        setOutputContent('Failed to load file content');
      }
    } catch {
      setOutputContent('Error loading file');
    } finally {
      setIsOutputLoading(false);
    }
  }, []);

  // Load devices
  const loadDevices = useCallback(async (path: string) => {
    if (!path) return;
    try {
      const response = await fetch(
        `/api/model/device/list?llamaBinPath=${encodeURIComponent(path)}`
      );
      const data = await response.json();
      if (data.success && data.data?.devices) {
        const devices = data.data.devices as string[];
        setAvailableDevices(devices);
        setSelectedDeviceIndices(devices.map((_, i) => i));
      }
    } catch {
      setAvailableDevices([]);
      setSelectedDeviceIndices([]);
    }
  }, []);

  // When llama.cpp path changes, reload devices
  useEffect(() => {
    if (llamaCppPath) {
      loadDevices(llamaCppPath);
    }
  }, [llamaCppPath, loadDevices]);

  return {
    // Model selection
    selectedModelId,
    setSelectedModelId,
    selectedModel,

    // LlamaCpp
    llamaCppPath,
    setLlamaCppPath,

    // Params
    enabledMap,
    setEnabledMap,
    valueMap,
    setValueMap,
    isParamsModalOpen,
    setIsParamsModalOpen,
    initializeParamDefaults,

    // Devices
    availableDevices,
    selectedDeviceIndices,
    setSelectedDeviceIndices,
    mainGpu,
    setMainGpu,

    // History / Output
    selectedHistoryFile,
    setSelectedHistoryFile,
    outputContent,
    setOutputContent,
    isOutputLoading,
    loadHistoryFile,
  };
}
