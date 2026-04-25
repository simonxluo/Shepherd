# 模型管理

## 模型状态机

```
Discovered → Unloaded → Loading → Loaded → Unloading → Unloaded
                          ↓         ↓
                        Error ← ─── Error
```

| 状态 | 值 | 说明 |
|---|---|---|
| StateUnloaded | 0 | 已发现，未加载 |
| StateLoading | 1 | 正在加载 |
| StateLoaded | 2 | 已加载运行中 |
| StateUnloading | 3 | 正在卸载 |
| StateError | 4 | 加载/运行错误 |

状态持久化在 SQLite 中。

## 核心类型

### Model

发现的 GGUF 模型实体：

| 字段 | 说明 |
|---|---|
| ID | SHA256 路径哈希 |
| Name | 文件名（不含扩展名） |
| DisplayName | 显示名（可自定义） |
| Alias | 别名（用于 API 调用） |
| Path | 文件路径 |
| Size / TotalSize | 单文件/总大小 |
| ShardCount / ShardFiles | 分片信息 |
| MmprojPath | 多模态投影文件 |
| Metadata | GGUF 元数据 |
| Favourite | 是否收藏 |
| LoadCount | 加载次数 |

### LoadRequest

模型加载请求，包含 70+ 个可配置字段：

| 类别 | 参数 |
|---|---|
| 基础 | port, ctxSize, batchSize, threads |
| GPU | gpuLayers, devices, mainGPU |
| 采样 | temperature, topP, topK, repeatPenalty, minP, seed |
| 性能 | flashAttention, noMmap, ubatchSize, parallelSlots |
| KV Cache | kvCacheTypeK/V, kvCacheUnified, kvCacheSize |
| 模板 | disableJinja, chatTemplate, contextShift |
| 视觉 | mmprojPath, enableVision |
| 服务器 | noWebUI, enableMetrics, timeout, alias |

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
CAS 状态转换 (Unloaded → Loading)
    ↓
ProcessEngine 构建命令
    ↓
健康检查轮询（指数退避）
    ↓
CAS 状态转换 (Loading → Loaded)
    ↓
SSE 广播通知
```

支持同步 (`Load`) 和异步 (`LoadAsync`) 两种模式。

**动态超时**：`1min/GB`，最小 5 分钟，最大 30 分钟。

## 端口分配

- 可配置范围（默认 8081-9000）
- 双重冲突检测：内存分配表 + TCP 连接探测
- 线程安全（`sync.Mutex`）

## 元数据持久化

SQLite WAL 模式存储：

| 数据 | 说明 |
|---|---|
| alias | 用户自定义别名 |
| favourite | 收藏标记 |
| load_config | 加载参数预设 |
| capabilities | 能力覆盖 |
| usage_stats | 使用统计 |

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

通过 `sync.WaitGroup` 跟踪进行中的推理请求：

- `InflightAdd()` / `InflightDone()`：增减计数
- `InflightWait()`：等待所有请求完成
- `AcquireSlot()` / `ReleaseSlot()`：并发槽位控制
- `AddTokens()`：Token 计数统计
