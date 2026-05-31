# VoxCPM2 TTS 插件代码修复计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按依赖顺序修复 VoxCPM2 TTS 插件及共享 TTS 子系统中的死代码、逻辑缺陷和边界情况问题。

**Architecture:** 修复按 6 层依赖关系组织：Layer 0 清理死代码 → Layer 1 修复基础工具 → Layer 2 修复核心状态管理（TTSPageShell） → Layer 3 修复 VoxCPM2 面板状态 → Layer 4 补全 UI 交互 → Layer 5 收尾边界情况。每层内的任务相互独立可并行，层间有依赖需顺序执行。

**Tech Stack:** React 19, TypeScript 5.9, TanStack Query v5, Zustand v5, Web Audio API

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `web/src/features/tts/hooks.ts` | Modify | 移除死代码 `saveVoicesCache` 和缓存基础设施 |
| `web/src/features/tts/plugins/voxcpm2/VoxCPM2Panel.tsx` | Modify | 移除死数据、修复状态治理、添加取消按钮 |
| `web/src/features/tts/lib/StreamAudioPlayer.ts` | Modify | 移除死回调、优化缓冲区拼接 |
| `web/src/features/tts/components/RefAudioInput.tsx` | Modify | 重构 audioFileToBase64、录音清理 |
| `web/src/features/tts/components/TTSPageShell.tsx` | Modify | 播放器重建、模型切换取消、autoPlay 修复 |
| `web/src/features/tts/components/TTSHistoryPanel.tsx` | Modify | 播放错误处理 |
| `web/src/features/tts/components/TTSPlaybackArea.tsx` | Modify | autoPlay 误触发修复 |

---

## Layer 0: Dead Code Cleanup（无依赖，纯删除）

### Task 1: 移除死代码和死数据

**Files:**
- Modify: `web/src/features/tts/hooks.ts:73-88`
- Modify: `web/src/features/tts/plugins/voxcpm2/VoxCPM2Panel.tsx:28`
- Modify: `web/src/features/tts/lib/StreamAudioPlayer.ts:37-38`

**1a. 移除 `hooks.ts` 中的缓存基础设施（从未写入）**

删除 `VOICES_CACHE_KEY` 常量、`getVoicesCache()` 函数、`saveVoicesCache()` 函数，以及 `useVoices` 中的 `placeholderData` 选项：

```ts
// hooks.ts — 删除以下全部代码块（约73-88行）:

// 删除:
const VOICES_CACHE_KEY = 'shepherd-tts-voices-cache';

function getVoicesCache(): Record<string, VoiceOption[]> {
  try {
    const saved = localStorage.getItem(VOICES_CACHE_KEY);
    return saved ? JSON.parse(saved) : {};
  } catch {
    return {};
  }
}

function saveVoicesCache(cache: Record<string, VoiceOption[]>) {
  try {
    localStorage.setItem(VOICES_CACHE_KEY, JSON.stringify(cache));
  } catch { /* silent */ }
}
```

同时在 `useVoices` 中删除 `placeholderData` 选项：

```ts
// useVoices 删除 placeholderData 配置项
// 删除:
    placeholderData: (): VoiceOption[] => {
      if (!model) return [];
      const cache = getVoicesCache();
      return cache[model] ?? [];
    },
```

**1b. 移除 `VoxCPM2Panel.tsx` VOXCPM2_LANGUAGES 中的 auto 条目**

删除数组第一条（`group: 'auto'`），因为 SelectContent 中已手动渲染 auto 选项：

```ts
// VoxCPM2Panel.tsx — 删除第 28 行:
  { value: 'auto', group: 'auto', label: 'tts.languageAuto', fallback: 'Auto Detect' },
```

数组应从 `// 30 official languages` 注释开始。

**1c. 移除 `StreamAudioPlayer.ts` 的死回调属性**

删除从未被任何调用方使用的两个回调：

```ts
// StreamAudioPlayer.ts — 删除以下两行（约37-38行）:
  onPlaybackStart?: () => void;
  onPlaybackEnd?: () => void;
```

同时删除 `startStream` 中调用它们的位置：

```ts
// 删除:
          this.onPlaybackStart?.();
```

```ts
// 删除:
      setTimeout(() => {
        if (this._state === 'completed') {
          this.onPlaybackEnd?.();
        }
      }, remaining + 200);
```

以及相关变量：

```ts
// 删除 estimatedPlayTime 计算块:
      const estimatedPlayTime = audioDuration * 1000;
      const playStart = this.firstChunkTime || this.startTime;
      const elapsed = performance.now() - playStart;
      const remaining = Math.max(0, estimatedPlayTime - elapsed);
```

- [ ] Step 1: 编辑 `hooks.ts` — 删除缓存基础设施和 placeholderData
- [ ] Step 2: 编辑 `VoxCPM2Panel.tsx` — 删除 VOXCPM2_LANGUAGES auto 条目
- [ ] Step 3: 编辑 `StreamAudioPlayer.ts` — 删除 onPlaybackStart/onPlaybackEnd 及其调用
- [ ] Step 4: 运行 `cd web && npx tsc --noEmit` 确认无类型错误
- [ ] Step 5: `git commit -m "chore(tts): remove dead code — unused cache, auto entry, playback callbacks"`

---

## Layer 1: 基础工具层修复（被上游组件依赖）

### Task 2: 重构 RefAudioInput 的 audioFileToBase64

**Files:**
- Modify: `web/src/features/tts/components/RefAudioInput.tsx:219-227`

当前实现逐字节拼接字符串再 `btoa()`，对大文件 O(n²) 性能且内存占用高。替换为 `FileReader.readAsDataURL`：

```ts
// RefAudioInput.tsx — 替换整个 audioFileToBase64 函数

async function audioFileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}
```

- [ ] Step 1: 编辑 `RefAudioInput.tsx` — 替换 audioFileToBase64 实现
- [ ] Step 2: 确认无类型错误
- [ ] Step 3: `git commit -m "perf(tts): use FileReader.readAsDataURL instead of manual base64 conversion"`

### Task 3: 优化 StreamAudioPlayer 缓冲区拼接

**Files:**
- Modify: `web/src/features/tts/lib/StreamAudioPlayer.ts:170-209`

当前每个 chunk 都创建新 Uint8Array 并复制全部已有数据，O(n²) 时间。改为用数组收集，只在需要对齐时拼接：

将 `startStream` 方法中的 buffer 处理逻辑替换：

```ts
// StreamAudioPlayer.ts — 替换 startStream 中的 buffer 处理

      // 替换: let buffer = new Uint8Array();
      // 为:
      let chunks: Uint8Array[] = [];
      let totalBytes = 0;
      let leftoverBytes = 0;
      let leftover = new Uint8Array(0);
      let hasFirstChunk = false;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        if (!hasFirstChunk) {
          this.firstChunkTime = performance.now();
          const ttfp = this.firstChunkTime - this.startTime;
          hasFirstChunk = true;
          this.setState('playing');
          this.updateMetrics({ ttfp: Math.round(ttfp) });
        }

        chunks.push(value);
        totalBytes += value.length;

        // 处理累积数据：合并 leftover + 所有 chunks，按 2 字节对齐
        const allBytes = leftoverBytes + totalBytes;
        const alignedLength = Math.floor(allBytes / 2) * 2;

        if (alignedLength >= 2 && alignedLength > leftoverBytes) {
          const merged = new Uint8Array(allBytes);
          let offset = 0;
          merged.set(leftover, offset);
          offset += leftoverBytes;
          for (const chunk of chunks) {
            merged.set(chunk, offset);
            offset += chunk.length;
          }

          const pcm = new Int16Array(merged.buffer, merged.byteOffset, alignedLength / 2);
          const pcmCopy = new Int16Array(pcm);
          this.pcmChunks.push(pcmCopy);
          this.sendChunk(pcmCopy);

          // 保留未对齐的尾部
          const remaining = allBytes - alignedLength;
          if (remaining > 0) {
            leftover = new Uint8Array(remaining);
            leftover.set(merged.subarray(alignedLength));
          } else {
            leftover = new Uint8Array(0);
          }
          leftoverBytes = remaining;
          chunks = [];
          totalBytes = 0;
        }

        this.updateMetrics({ bytesReceived: this._metrics.bytesReceived + value.length });
      }

      // 处理最终 leftover
      if (leftoverBytes >= 2) {
        const pcm = new Int16Array(leftover.buffer, leftover.byteOffset, Math.floor(leftoverBytes / 2));
        const pcmCopy = new Int16Array(pcm);
        this.pcmChunks.push(pcmCopy);
        this.sendChunk(pcmCopy);
      }
```

> **注意：** 这个优化保留相同的对齐逻辑语义，只是减少了中间数组分配次数。每次 sendChunk 后清空 chunks 数组，避免 O(n²) 的逐 chunk 复制。

- [ ] Step 1: 编辑 `StreamAudioPlayer.ts` — 替换 buffer 处理逻辑
- [ ] Step 2: 确认无类型错误
- [ ] Step 3: `git commit -m "perf(tts): optimize stream buffer concatenation from O(n²) to amortized O(n)"`

---

## Layer 2: 核心状态管理层（TTSPageShell — 所有面板依赖它）

### Task 4: 修复 TTSPageShell — 播放器重建 + 模型切换取消

**Files:**
- Modify: `web/src/features/tts/components/TTSPageShell.tsx:133-138, 151-154, 175-248`

这两个问题紧密耦合：模型切换时需要同时取消生成并重建播放器。

**4a. 提取 player 重建为独立函数，在 sampleRate 变化时触发重建**

在 `TTSPageShell` 中，将 player 创建逻辑提取出来，并在 `currentFeatures.defaultSampleRate` 变化时重建：

```ts
// TTSPageShell.tsx — 在 handleCancel 之后添加:

  // 当 sampleRate 变化时重建播放器
  useEffect(() => {
    if (playerRef.current) {
      playerRef.current.destroy();
      playerRef.current = null;
    }
  }, [currentFeatures.defaultSampleRate]);
```

**4b. 模型切换时取消进行中的生成**

修改 `handleModelChange`，在切换前取消当前操作：

```ts
// TTSPageShell.tsx — 替换 handleModelChange:

  const handleModelChange = useCallback((modelName: string) => {
    // 切换模型前取消进行中的生成
    handleCancel();
    setModelByPlugin((prev) => ({
      ...prev,
      [activePluginId]: modelName,
    }));
  }, [activePluginId, handleCancel]);
```

**4c. 在 handleGenerate 中始终重建 player（确保 sampleRate 正确）**

```ts
// TTSPageShell.tsx — 替换 handleGenerate 中 stream 分支的 player 创建逻辑:

      // 替换:
      //   if (!playerRef.current) {
      //     const player = new StreamAudioPlayer(currentFeatures.defaultSampleRate);
      //     await player.init();
      //     playerRef.current = player;
      //   }
      // 为:
        // 始终重建播放器以确保 sampleRate 正确
        if (playerRef.current) {
          playerRef.current.destroy();
        }
        const player = new StreamAudioPlayer(currentFeatures.defaultSampleRate);
        await player.init();
        playerRef.current = player;
```

- [ ] Step 1: 编辑 `TTSPageShell.tsx` — 添加 sampleRate 变化时的 player 重建 effect
- [ ] Step 2: 编辑 `TTSPageShell.tsx` — 修改 handleModelChange 添加 handleCancel 调用
- [ ] Step 3: 编辑 `TTSPageShell.tsx` — 修改 handleGenerate 中 player 创建逻辑
- [ ] Step 4: 确认无类型错误
- [ ] Step 5: `git commit -m "fix(tts): rebuild player on sampleRate change and cancel generation on model switch"`

---

## Layer 3: VoxCPM2 面板状态治理（依赖 Layer 2 的切换逻辑）

### Task 5: 修复 VoxCPM2Panel 状态治理

**Files:**
- Modify: `web/src/features/tts/plugins/voxcpm2/VoxCPM2Panel.tsx:133, 221-229, 313-314, 384-387`

**5a. 模型切换时清除 selectedVoice**

```tsx
// VoxCPM2Panel.tsx — 修改 ModelSelect 的 onValueChange（约384-387行）:

// 替换:
          onValueChange={(v) => {
            onModelChange(v);
            setRefAudio('');
          }}
// 为:
          onValueChange={(v) => {
            onModelChange(v);
            setRefAudio('');
            setSelectedVoice('');
            setRefText('');
            setInstructions('');
            setSeed('');
            setMaxNewTokens('');
            setLanguage('auto');
          }}
```

**5b. Config 恢复时模型切换清除残留状态**

```tsx
// VoxCPM2Panel.tsx — 修改 config 恢复 effect（约221-229行）:

// 替换:
  useEffect(() => {
    if (!ttsConfig) return;
    if (ttsConfig.instructions !== undefined) setInstructions(ttsConfig.instructions);
    if (ttsConfig.refAudio !== undefined) setRefAudio(ttsConfig.refAudio);
    if (ttsConfig.refText !== undefined) setRefText(ttsConfig.refText);
    if (ttsConfig.seed !== undefined) setSeed(ttsConfig.seed);
    if (ttsConfig.maxNewTokens !== undefined) setMaxNewTokens(ttsConfig.maxNewTokens);
    if (ttsConfig.language !== undefined) setLanguage(ttsConfig.language || 'auto');
  }, [ttsConfig]);

// 为:
  useEffect(() => {
    if (ttsConfig) {
      if (ttsConfig.instructions !== undefined) setInstructions(ttsConfig.instructions);
      if (ttsConfig.refAudio !== undefined) setRefAudio(ttsConfig.refAudio);
      if (ttsConfig.refText !== undefined) setRefText(ttsConfig.refText);
      if (ttsConfig.seed !== undefined) setSeed(ttsConfig.seed);
      if (ttsConfig.maxNewTokens !== undefined) setMaxNewTokens(ttsConfig.maxNewTokens);
      if (ttsConfig.language !== undefined) setLanguage(ttsConfig.language || 'auto');
    } else {
      // 模型切换后无 config 时清除残留状态
      setInstructions('');
      setRefAudio('');
      setRefText('');
      setSeed('');
      setMaxNewTokens('');
      setLanguage('auto');
    }
  }, [ttsConfig]);
```

> **注意：** 这个 effect 的依赖需要同时包含 `modelIdForConfig`，确保模型切换时触发：

```tsx
  }, [ttsConfig, modelIdForConfig]);
```

**5c. 修复 seed=0 和 maxNewTokens=0 被静默丢弃**

```tsx
// VoxCPM2Panel.tsx — 修改 handleGenerate 中的 seed/maxNewTokens 解析（约313-314行）:

// 替换:
    if (seed) payload.seed = parseInt(seed, 10) || undefined;
    if (maxNewTokens) payload.max_new_tokens = parseInt(maxNewTokens, 10) || undefined;

// 为:
    const parsedSeed = parseInt(seed, 10);
    if (seed !== '' && !Number.isNaN(parsedSeed)) payload.seed = parsedSeed;
    const parsedTokens = parseInt(maxNewTokens, 10);
    if (maxNewTokens !== '' && !Number.isNaN(parsedTokens)) payload.max_new_tokens = parsedTokens;
```

- [ ] Step 1: 编辑 `VoxCPM2Panel.tsx` — 模型切换时清除所有状态
- [ ] Step 2: 编辑 `VoxCPM2Panel.tsx` — config 恢复 effect 添加 else 分支 + modelIdForConfig 依赖
- [ ] Step 3: 编辑 `VoxCPM2Panel.tsx` — 修复 seed/maxNewTokens 解析
- [ ] Step 4: 确认无类型错误
- [ ] Step 5: `git commit -m "fix(tts): clear state on model switch and preserve seed=0/maxNewTokens=0"`

---

## Layer 4: VoxCPM2 UI 交互补全（依赖 Layer 3 状态正确）

### Task 6: 添加取消按钮 + voice 删除确认 + voice/refAudio 互斥提示

**Files:**
- Modify: `web/src/features/tts/plugins/voxcpm2/VoxCPM2Panel.tsx`

**6a. 添加取消按钮**

在 VoxCPM2Panel 的 props 解构中添加 `onCancel`，并将生成按钮改为与 GenericTTSPanel 一致的条件渲染：

```tsx
// VoxCPM2Panel.tsx — 修改 props 解构（约73-84行）:

// 在解构中添加 onCancel:
  const {
    model: selectedModel,
    matchedModels,
    onGenerate,
    onCancel,           // ← 新增
    isGenerating,
    streamState,
    onModelChange,
    refAudioOverride,
    modelStatus,
    fullModelId,
    voiceRefreshTrigger,
  } = props;
```

替换生成按钮区域（约727-758行）：

```tsx
// 替换整个 Button 块:
      {isGenerating ? (
        <Button
          onClick={onCancel}
          variant="destructive"
          className="w-full"
        >
          <X className="w-4 h-4 mr-2" />
          {t('tts.cancel', 'Cancel')}
        </Button>
      ) : (
        <Button
          onClick={handleGenerate}
          disabled={!modelName || isModelLoading || isModelError}
          className="w-full"
        >
          {isModelLoading ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              {t('tts.modelLoading', 'Model is loading...')}
            </>
          ) : isModelStopped ? (
            <>
              <Volume2 className="w-4 h-4 mr-2" />
              {t('tts.loadModelToGenerate', 'Load Model & Generate')}
            </>
          ) : isModelError ? (
            <>
              <AlertCircle className="w-4 h-4 mr-2" />
              {t('tts.modelError', 'Model error')}
            </>
          ) : (
            <>
              <Volume2 className="w-4 h-4 mr-2" />
              {t('tts.generate', 'Generate Speech')}
            </>
          )}
        </Button>
      )}
```

确保在文件顶部 import 中添加 `X`：

```tsx
import { Volume2, Loader2, Settings2, ChevronDown, AlertCircle, Play, Upload, Trash2, Mic, X } from 'lucide-react';
```

**6b. Voice 删除添加确认**

```tsx
// VoxCPM2Panel.tsx — 修改 handleDeleteVoice（约191-201行）:

// 替换:
  const handleDeleteVoice = useCallback(async (voiceName: string) => {
    if (!modelName) return;
    try {
      await deleteVoice(modelName, voiceName);
      toast.success(t('tts.voxcpm2.voiceDeleted', 'Voice deleted'));
      if (selectedVoice === voiceName) setSelectedVoice('');
      await loadVoices();
    } catch (err) {
      toast.error(t('tts.voxcpm2.voiceDeleteFailed', 'Delete failed'), (err as Error).message);
    }
  }, [modelName, selectedVoice, loadVoices, t]);

// 为:
  const handleDeleteVoice = useCallback(async (voiceName: string) => {
    if (!modelName) return;
    if (!window.confirm(t('tts.voxcpm2.confirmDeleteVoice', 'Delete voice "{{name}}"?', { name: voiceName }))) return;
    try {
      await deleteVoice(modelName, voiceName);
      toast.success(t('tts.voxcpm2.voiceDeleted', 'Voice deleted'));
      if (selectedVoice === voiceName) setSelectedVoice('');
      await loadVoices();
    } catch (err) {
      toast.error(t('tts.voxcpm2.voiceDeleteFailed', 'Delete failed'), (err as Error).message);
    }
  }, [modelName, selectedVoice, loadVoices, t]);
```

**6c. 当 selectedVoice 和 refAudio 同时有值时显示提示**

在 handleGenerate 的 payload 构建后、`onGenerate(payload)` 前，添加提示：

```tsx
// VoxCPM2Panel.tsx — 修改 handleGenerate payload 构建逻辑（约305-317行）:

// 替换:
    if (selectedVoice) {
      payload.voice = selectedVoice;
    } else if (refAudio) {
      payload.ref_audio = refAudio;
    }

// 为:
    if (selectedVoice) {
      payload.voice = selectedVoice;
      // 当同时设置了 refAudio 时提示用户
      if (refAudio) {
        toast.info(t('tts.voxcpm2.voiceOverridesRefAudio', 'Voice is set — reference audio will be ignored.'));
      }
    } else if (refAudio) {
      payload.ref_audio = refAudio;
    }
```

- [ ] Step 1: 编辑 `VoxCPM2Panel.tsx` — 添加 X import + onCancel 解构
- [ ] Step 2: 编辑 `VoxCPM2Panel.tsx` — 替换生成按钮为条件渲染（含取消按钮）
- [ ] Step 3: 编辑 `VoxCPM2Panel.tsx` — handleDeleteVoice 添加确认
- [ ] Step 4: 编辑 `VoxCPM2Panel.tsx` — handleGenerate 添加互斥提示
- [ ] Step 5: 确认无类型错误
- [ ] Step 6: `git commit -m "feat(tts): add cancel button, voice delete confirmation, and voice/refAudio conflict hint"`

### Task 7: 修复 `as unknown as` 类型强转

**Files:**
- Modify: `web/src/features/tts/hooks.ts:138-146`
- Modify: `web/src/features/tts/plugins/voxcpm2/VoxCPM2Panel.tsx:255`
- Modify: `web/src/features/tts/components/GenericTTSPanel.tsx:133`

**7a. 在 hooks.ts 中创建类型安全的 save wrapper**

```ts
// hooks.ts — 修改 useTTSConfig 返回值:

// 替换:
export function useTTSConfig(modelId: string) {
  const { data, isLoading } = useModelLoadConfig(modelId);
  const saveConfig = useSaveModelLoadConfig();
  const deleteConfig = useDeleteModelLoadConfig();

  const ttsConfig = (data?.exists && data.config) ? extractTTSConfig(data.config.config as Record<string, unknown>) : null;

  return { ttsConfig, isLoading, saveConfig, deleteConfig };
}

// 为:
export function useTTSConfig(modelId: string) {
  const { data, isLoading } = useModelLoadConfig(modelId);
  const rawSaveConfig = useSaveModelLoadConfig();
  const deleteConfig = useDeleteModelLoadConfig();

  const ttsConfig = (data?.exists && data.config) ? extractTTSConfig(data.config.config as Record<string, unknown>) : null;

  const saveTTSConfig = useCallback((config: TTSConfig) => {
    if (!modelId) return;
    rawSaveConfig.mutate({ modelId, config: config as unknown as import('@/types/model').LoadModelParams });
  }, [modelId, rawSaveConfig]);

  return { ttsConfig, isLoading, saveConfig: { ...rawSaveConfig, mutate: saveTTSConfig, mutateAsync: rawSaveConfig.mutateAsync }, deleteConfig };
}
```

需要在文件顶部添加 `useCallback` import（如果尚未存在）。

**7b. 更新 VoxCPM2Panel 和 GenericTTSPanel 使用新接口**

```tsx
// VoxCPM2Panel.tsx — 替换 handleSaveToServer（约253-256行）:

// 替换:
  const handleSaveToServer = useCallback(() => {
    if (!modelIdForConfig) return;
    saveConfig.mutate({ modelId: modelIdForConfig, config: getCurrentConfig() as unknown as import('@/types/model').LoadModelParams });
  }, [modelIdForConfig, getCurrentConfig, saveConfig]);

// 为:
  const handleSaveToServer = useCallback(() => {
    if (!modelIdForConfig) return;
    saveConfig.mutate(getCurrentConfig());
  }, [modelIdForConfig, getCurrentConfig, saveConfig]);
```

```tsx
// GenericTTSPanel.tsx — 替换 handleSaveToServer（约130-134行）:

// 替换:
  const handleSaveToServer = useCallback(() => {
    if (!modelIdForConfig) return;
    saveConfig.mutate({ modelId: modelIdForConfig, config: getCurrentConfig() as unknown as import('@/types/model').LoadModelParams });
  }, [modelIdForConfig, getCurrentConfig, saveConfig]);

// 为:
  const handleSaveToServer = useCallback(() => {
    if (!modelIdForConfig) return;
    saveConfig.mutate(getCurrentConfig());
  }, [modelIdForConfig, getCurrentConfig, saveConfig]);
```

- [ ] Step 1: 编辑 `hooks.ts` — 创建类型安全的 saveTTSConfig wrapper
- [ ] Step 2: 编辑 `VoxCPM2Panel.tsx` — 使用新接口
- [ ] Step 3: 编辑 `GenericTTSPanel.tsx` — 使用新接口
- [ ] Step 4: 确认无类型错误
- [ ] Step 5: `git commit -m "refactor(tts): encapsulate config type cast in useTTSConfig hook"`

---

## Layer 5: 边界情况收尾

### Task 8: 录音清理 + 历史播放错误处理 + autoPlay 修复

**Files:**
- Modify: `web/src/features/tts/components/RefAudioInput.tsx:51-78`
- Modify: `web/src/features/tts/components/TTSHistoryPanel.tsx:53-65`
- Modify: `web/src/features/tts/components/TTSPlaybackArea.tsx:50-55`

**8a. RefAudioInput 录音清理**

添加 cleanup effect 并在 toggleRecording 中使用 ref 追踪 stream：

```tsx
// RefAudioInput.tsx — 添加 stream ref（在 mediaRecorderRef 后面）:

  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);     // ← 新增
  const chunksRef = useRef<Blob[]>([]);
```

```tsx
// RefAudioInput.tsx — 添加 cleanup effect（在组件内任意位置）:

  // 组件卸载时清理录音资源
  useEffect(() => {
    return () => {
      if (mediaRecorderRef.current && mediaRecorderRef.current.state !== 'inactive') {
        mediaRecorderRef.current.stop();
      }
      streamRef.current?.getTracks().forEach((t) => t.stop());
    };
  }, []);
```

修改 toggleRecording 中的录音启动，保存 stream ref：

```tsx
// RefAudioInput.tsx — 修改 toggleRecording 录音部分:

// 替换:
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        const recorder = new MediaRecorder(stream);

// 为:
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        streamRef.current = stream;
        const recorder = new MediaRecorder(stream);
```

修改 recorder.onstop 清理 stream ref：

```tsx
// 替换:
        recorder.onstop = async () => {
          stream.getTracks().forEach((t) => t.stop());

// 为:
        recorder.onstop = async () => {
          stream.getTracks().forEach((t) => t.stop());
          streamRef.current = null;
```

**8b. TTSHistoryPanel 播放错误处理**

```tsx
// TTSHistoryPanel.tsx — 修改 handlePlay（约53-65行）:

// 替换:
  const handlePlay = (item: TTSHistoryItem) => {
    if (!audioRef.current) return;

    if (playingId === item.id) {
      audioRef.current.pause();
      setPlayingId(null);
      return;
    }

    audioRef.current.src = getTTSAudioUrl(item.id);
    audioRef.current.play();
    setPlayingId(item.id);
  };

// 为:
  const handlePlay = (item: TTSHistoryItem) => {
    if (!audioRef.current) return;

    if (playingId === item.id) {
      audioRef.current.pause();
      setPlayingId(null);
      return;
    }

    audioRef.current.src = getTTSAudioUrl(item.id);
    audioRef.current.play().catch(() => {
      toast.error(t('tts.playbackFailed', 'Playback failed'));
      setPlayingId(null);
    });
    setPlayingId(item.id);
  };
```

同时在 `<audio>` 元素上添加错误处理：

```tsx
// TTSHistoryPanel.tsx — 修改 audio 元素（约147-151行）:

// 替换:
      <audio
        ref={audioRef}
        onEnded={() => setPlayingId(null)}
        onPause={() => setPlayingId(null)}
      />

// 为:
      <audio
        ref={audioRef}
        onEnded={() => setPlayingId(null)}
        onPause={() => setPlayingId(null)}
        onError={() => {
          setPlayingId(null);
        }}
      />
```

**8c. TTSPlaybackArea autoPlay 误触发修复**

```tsx
// TTSPlaybackArea.tsx — 修改 autoPlay effect（约50-55行）:

// 替换:
  useEffect(() => {
    if (audioUrl && autoPlay && audioRef.current) {
      audioRef.current.load();
      audioRef.current.play().catch(() => {});
    }
  }, [audioUrl, autoPlay]);

// 为:
  const prevAudioUrlRef = useRef<string | null>(null);
  useEffect(() => {
    // 只在 audioUrl 变化时触发自动播放，不在 autoPlay 切换时触发
    if (audioUrl && audioUrl !== prevAudioUrlRef.current && autoPlay && audioRef.current) {
      audioRef.current.load();
      audioRef.current.play().catch(() => {});
    }
    prevAudioUrlRef.current = audioUrl;
  }, [audioUrl, autoPlay]);
```

- [ ] Step 1: 编辑 `RefAudioInput.tsx` — 添加 streamRef、cleanup effect、录音清理
- [ ] Step 2: 编辑 `TTSHistoryPanel.tsx` — 播放错误处理 + audio onError
- [ ] Step 3: 编辑 `TTSPlaybackArea.tsx` — autoPlay 误触发修复
- [ ] Step 4: 确认无类型错误
- [ ] Step 5: `git commit -m "fix(tts): recording cleanup, playback error handling, autoPlay debounce"`

---

## Self-Review Checklist

- [x] **Spec coverage:** 4 个严重问题、8 个中等问题、4 个轻微问题 → 全部有对应 Task
- [x] **Placeholder scan:** 无 TBD/TODO，每个 step 都有具体代码
- [x] **Type consistency:** 各 Task 中的函数签名和属性名保持一致
- [x] **Dependency order:** Layer 0→5 严格按依赖关系排列
