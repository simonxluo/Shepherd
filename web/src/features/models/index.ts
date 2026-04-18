/**
 * Models feature - unified re-export
 *
 * All hooks are re-exported here so that external imports like
 * `import { useXxx } from '@/features/models/hooks'` continue to work.
 * New code can also import directly from the sub-module files.
 */

// Scan
export { useScanModels } from './scan';

// Model query & load/unload
export {
  useModels,
  useModel,
  useLoadModel,
  useUnloadModel,
  useUpdateModelAlias,
  useSetModelFavourite,
  useFilteredModels,
} from './load';

// Benchmark
export {
  useBenchmarkParams,
  useLlamaCppVersions,
  useCreateBenchmark,
} from './benchmark';

// GPU / llama.cpp backend / capabilities / VRAM estimation
export {
  useGPUs,
  useLlamacppBackends,
  useModelCapabilities,
  useSetModelCapabilities,
  useAutoDetectCapabilities,
  useEstimateVRAM,
  type SystemGPUInfo,
  type LlamacppBackend,
} from './capabilities';

// Model load config
export {
  useModelLoadConfig,
  useSaveModelLoadConfig,
  useDeleteModelLoadConfig,
} from './config';
