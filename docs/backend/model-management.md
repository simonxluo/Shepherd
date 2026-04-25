# 模型管理

## 模型状态机

```
Unloaded → Loading → Loaded → Unloading → Unloaded
              ↓         ↓
            Error ← ─── Error
```

| 状态 | 值 | 说明 |
|---|---|---|
| StateUnloaded | 0 | 未加载 |
| StateLoading | 1 | 正在加载 |
| StateLoaded | 2 | 已加载运行中 |
| StateUnloading | 3 | 正在卸载 |
| StateError | 4 | 加载/运行错误 |

合法的状态转换：

```
Unloaded  → Loading
Loading   → Loaded, Error, Unloading
Loaded    → Unloading, Error
Unloading → Unloaded, Error
Error     → Unloaded, Loading
```

状态转换通过 `sync.Mutex` 保护的 `swapState` / `transitionTo` 方法完成，确保线程安全。

## 核心类型

### Model

发现的 GGUF 模型实体：

| 字段 | 类型 | 说明 |
|---|---|---|
| ID | string | SHA256 路径哈希 |
| Name | string | 文件名（不含扩展名） |
| DisplayName | string | 显示名（处理重复） |
| Alias | string | 别名（用于 API 调用） |
| Description | string | 模型描述/卡片 |
| Path | string | 文件路径 |
| PathPrefix | string | 路径前缀（区分重复模型） |
| Size | int64 | 单文件大小 |
| Favourite | bool | 是否收藏 |
| Tags | []string | 标签分类 |
| License | string | 模型许可证 |
| Author | string | 模型作者/组织 |
| Downloads | int | 下载计数 |
| Metadata | *gguf.Metadata | GGUF 元数据 |
| MmprojPath | string | 多模态投影文件路径 |
| MmprojMeta | *gguf.Metadata | mmproj GGUF 元数据 |
| ShardCount | int | 分卷数量（0 = 非分卷） |
| ShardFiles | []string | 分卷文件路径列表 |
| TotalSize | int64 | 分卷总大小 |
| ScannedAt | time.Time | 扫描时间 |
| SourcePath | string | 原始扫描路径 |
| SourceType | string | 来源类型（"local"/"huggingface"/"modelscope"） |
| LoadCount | int | 加载次数 |
| LastLoaded | time.Time | 最后加载时间 |
| TotalTokens | int64 | 总 Token 生成数 |

### ModelStatus

运行时状态（不持久化）：

| 字段 | 说明 |
|---|---|
| State | LoadState（Unloaded/Loading/Loaded/Unloading/Error） |
| ProcessID | 进程管理器中的模型 ID |
| Port | 绑定端口 |
| CtxSize | 实际上下文大小 |
| LoadedAt | 加载完成时间 |
| Error | 错误信息 |
| LoadWait | 加载等待 WaitGroup |
| InflightWg / InflightCount | 进行中请求跟踪 |
| ConcurrencySem / ConcurrencyLimit | 并发槽位控制 |
| UnloadAfter | 空闲自动卸载时间 |
| TotalPromptTokens / TotalCompletionTokens | Token 统计 |

### LoadRequest

模型加载请求，包含约 55 个可配置字段：

| 类别 | 参数 |
|---|---|
| 基础 | modelId, nodeId, ctxSize, batchSize, threads |
| GPU | gpuLayers, devices, mainGPU |
| 采样 | temperature, topP, topK, repeatPenalty, minP, seed, nPredict |
| 性能 | flashAttention, noMmap, lockMemory, ubatchSize, parallelSlots |
| KV Cache | kvCacheTypeK/V, kvCacheUnified, kvCacheSize |
| 模板 | chatTemplateFile, disableJinja, chatTemplate, contextShift |
| 视觉 | mmprojPath, enableVision |
| 服务器 | noWebUI, enableMetrics, slotSavePath, cacheRam, timeout, alias |
| 扩展采样 | logitsAll, reranking, presencePenalty, frequencyPenalty, repeatLastN, typicalP, ignoreEOS |
| 多 GPU | splitMode, tensorSplit |
| 优化 | contBatching, cachePrompt |
| 结构化生成 | grammar, grammarFile |
| LoRA | lora, loraScaled |
| RoPE 缩放 | ropeScaling, ropeScale, ropeFreqBase, ropeFreqScale |
| 运行时管理 | unloadAfterMinutes, concurrencyLimit |
| 自定义 | llamaCppPath（CustomCmd）, extraArgs（ExtraParams）, chatTemplateKwargs, directIo |

## 模型扫描

扫描流程：

1. 遍历配置的模型目录（信号量 ≤10 并发）
2. 解析 GGUF 元数据（通过 `infra/gguf`）
3. 基于 SHA256(路径) 生成唯一 ID
4. 检测 mmproj 伴随文件（多模态）
5. 合并分片模型（`name-00001-of-00006.gguf`）
6. JSON 缓存 + mtime 优化（未修改的文件跳过解析）

## 能力检测

基于关键词扫描模型名称、GGUF 架构、chat_template：

| 能力 | 关键词 |
|---|---|
| Thinking | "thinking", "think", "reason" 等 |
| Tools | "tool", "function", "agent" 等 |
| Embedding | "embed", "bge", "e5" 等 |
| Rerank | "rerank", "cross-encoder" 等 |

**互斥规则**：Embedding/Rerank 与 Thinking/Tools 互斥。如果检测到 Embedding 或 Rerank，自动禁用 Thinking 和 Tools。

## 加载序列

```
状态转换 (Unloaded → Loading)
    ↓
BuildCommandFromRequest 构建命令
    ↓
ProcessManager.Start() 启动进程
    ↓
健康检查轮询（指数退避）
    ↓
状态转换 (Loading → Loaded)
    ↓
SSE 广播通知
```

支持同步 (`Load`) 和异步 (`LoadAsync`) 两种模式。

## 端口分配

- 可配置范围（默认 8081-9000）
- 双重冲突检测：内存分配表 + TCP 连接探测
- 线程安全（`sync.Mutex`）

## 元数据持久化

通过 `infra/storage` 的 `Store` 接口持久化。

### ModelMetadata（用户自定义元数据）

| 字段 | 类型 | 说明 |
|---|---|---|
| ModelID | string | 模型 ID（主键） |
| NodeID | string | 节点 ID |
| StoragePath | string | 存储路径（分布式系统） |
| Alias | string | 用户自定义别名 |
| Favourite | bool | 收藏标记 |
| Tags | []string | 标签列表 |
| Description | string | 用户描述 |
| LoadCount | int | 加载次数 |
| LastLoaded | *time.Time | 最后加载时间 |
| TotalTokens | int64 | 总 Token 生成数 |
| Capabilities | *Capabilities | 能力覆盖（自动检测或用户定义） |
| CreatedAt / UpdatedAt | time.Time | 记录时间戳 |

### ModelLoadConfig（加载参数预设，独立实体）

| 字段 | 类型 | 说明 |
|---|---|---|
| ID | string | 配置 ID |
| NodeID | string | 节点 ID |
| ModelID | string | 模型 ID |
| ModelName | string | 模型名称 |
| Config | map[string]interface{} | LoadModelParams JSON |
| CreatedAt / UpdatedAt | time.Time | 时间戳 |

`load_config` 是独立的存储实体（`ModelLoadConfig`），不是 `ModelMetadata` 的一部分。

## Benchmark

- 任务创建与隔离进程执行
- 结果存储与跨配置比较
- 支持参数预设配置

## 惰性加载

`EnsureLoaded(modelID)` 实现惰性加载：

1. 如果已加载 → 直接返回
2. 如果正在加载 → 等待加载完成
3. 如果未加载 → 触发加载并等待

## Inflight 跟踪

通过 `sync.WaitGroup` 和 `atomic.Int32` 跟踪进行中的推理请求：

- `InflightWg` / `InflightCount`：增减计数
- `ConcurrencySem`：并发槽位控制（`ConcurrencyLimit` 限制）
- `AddTokens()`：Token 计数统计
