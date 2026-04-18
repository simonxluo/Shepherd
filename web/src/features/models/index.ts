/**
 * Models feature - unified re-export
 *
 * All hooks are re-exported here so that external imports like
 * `import { useXxx } from '@/features/models/hooks'` continue to work.
 * New code can also import directly from the sub-module files.
 */

// 扫描相关
export { useScanModels } from './scan';

// 模型查询 & 加载/卸载相关
export {
  useModels,
  useModel,
  useLoadModel,
  useUnloadModel,
  useUpdateModelAlias,
  useSetModelFavourite,
  useFilteredModels,
} from './load';

// 压测相关
export {
  useBenchmarkParams,
  useLlamaCppVersions,
  useCreateBenchmark,
} from './benchmark';

// GPU / llama.cpp 后端 / 能力检测 / 显存估算相关
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

// 模型加载配置相关
export {
  useModelLoadConfig,
  useSaveModelLoadConfig,
  useDeleteModelLoadConfig,
} from './config';
