# 模型加载 API

## 加载模型

```
POST /api/models/{id}/load
```

异步加载指定模型。返回 HTTP **202 Accepted**，表示加载请求已被接受。

### 请求体

```json
{
  "ctxSize": 4096,
  "gpuLayers": 99,
  "threads": 4,
  "temperature": 0.7,
  "flashAttention": true,
  "parallelSlots": 4
}
```

### 响应

HTTP 状态码 **202 Accepted**。

```json
{
  "success": true,
  "data": {
    "id": "model-id",
    "status": "loading"
  },
  "metadata": {
    "timestamp": "2026-01-15T10:00:00Z",
    "requestId": "req-xxx"
  }
}
```

如果模型已经处于加载状态：

```json
{
  "success": true,
  "data": {
    "id": "model-id",
    "status": "loading"
  }
}
```

如果模型已加载完成：

```json
{
  "success": true,
  "data": {
    "id": "model-id",
    "status": "running",
    "port": 8081
  }
}
```

### 响应字段

| 字段 | 类型 | 说明 |
|---|---|---|
| id | string | 模型 ID（来自 URL 路径） |
| status | string | 当前状态：`loading` 或 `running` |
| port | int | 服务端口（仅当模型已加载时返回） |

### 示例

```bash
# 加载模型（GPU 99 层，4K 上下文）
curl -X POST http://localhost:9190/api/models/abc123/load \
  -H "Content-Type: application/json" \
  -d '{"ctxSize": 4096, "gpuLayers": 99}'

# 使用别名加载
curl -X POST http://localhost:9190/api/models/abc123/load \
  -H "Content-Type: application/json" \
  -d '{"alias": "my-model", "temperature": 0.5}'
```

## 完整参数参考

### 基础参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| ctxSize | int | 512 | 上下文长度 |
| batchSize | int | 512 | 批处理大小 |
| threads | int | 4 | 线程数 |
| nodeId | string | - | 指定运行节点 ID，为空表示自动调度 |

### GPU 参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| gpuLayers | int | 0 | GPU 层数（0=纯 CPU，-1=全部） |
| devices | []string | - | GPU 设备列表（如 `["cuda:0", "cuda:1"]`） |
| mainGpu | int | 0 | 主 GPU 序号（-mg 标志） |

### 采样参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| temperature | float | 0.7 | 温度 |
| topP | float | 0.95 | Top-P 采样 |
| topK | int | 40 | Top-K 采样 |
| repeatPenalty | float | 1.1 | 重复惩罚 |
| minP | float | - | Min-P 采样 |
| presencePenalty | float | - | 存在惩罚 |
| frequencyPenalty | float | - | 频率惩罚 |
| seed | int | - | 随机种子 |
| nPredict | int | - | 最大预测 token 数 |

### 性能参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| flashAttention | bool | false | Flash Attention（-fa 标志） |
| noMmap | bool | false | 禁用 mmap（--no-mmap 标志） |
| lockMemory | bool | false | 锁定内存（--mlock 标志） |
| uBatchSize | int | - | 微批次大小（--ubatch-size） |
| parallelSlots | int | - | 并行 slot 数（--parallel） |
| logitsAll | bool | false | 输出所有 logits（--logits-all） |
| contBatching | bool | false | 连续批处理（--cont-batching） |
| cachePrompt | bool | false | 缓存提示（--cache-prompt） |

### KV Cache 参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| kvCacheTypeK | string | - | K cache 类型（--kv-cache-type-k） |
| kvCacheTypeV | string | - | V cache 类型（--kv-cache-type-v） |
| kvCacheUnified | bool | false | 统一 KV cache（--kv-unified） |
| kvCacheSize | int | - | KV cache 大小（--kv-cache-size） |

### 模板参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| disableJinja | bool | false | 禁用 Jinja 模板（--jinja 反向） |
| chatTemplate | string | - | 自定义聊天模板（--chat-template） |
| chatTemplateFile | string | - | 模板文件路径（--chat-template-file） |
| chatTemplateKwargs | string | - | 模板参数（--chat-template-kwargs） |
| contextShift | bool | false | 上下文移位（--context-shift） |

### 视觉参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| mmprojPath | string | - | 多模态投影文件路径 |
| enableVision | bool | false | 启用视觉/多模态 |

### 服务器参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| noWebUI | bool | false | 禁用内置 Web UI（--no-webui） |
| enableMetrics | bool | false | 启用指标（--metrics） |
| slotSavePath | string | - | Slot 保存路径（--slot-save-path） |
| cacheRam | int | - | RAM 缓存大小 MB（--cache-ram） |
| timeout | int | - | 超时秒数（--timeout） |
| alias | string | - | 模型别名（--alias） |

### 自定义命令

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| llamaCppPath | string | - | 自定义 llama.cpp 二进制路径覆盖 |
| extraArgs | string | - | 追加到命令的额外 CLI 参数 |

### 多 GPU 配置

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| splitMode | string | - | 分割模式（none/layer/row，--split-mode） |
| tensorSplit | string | - | 张量分割比例（逗号分隔，--tensor-split） |

### RoPE 缩放参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| ropeScaling | string | - | RoPE 缩放类型（--rope-scaling） |
| ropeScale | float | - | RoPE 缩放因子（--rope-scale） |
| ropeFreqBase | float | - | RoPE 基础频率（--rope-freq-base） |
| ropeFreqScale | float | - | RoPE 频率缩放（--rope-freq-scale） |

### 扩展采样参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| repeatLastN | int | - | 重复惩罚窗口（--repeat-last-n） |
| typicalP | float | - | Typical-P 采样（--typical-p） |
| ignoreEos | bool | false | 忽略 EOS（--ignore-eos） |

### 结构化生成

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| grammar | string | - | GBNF 语法（--grammar） |
| grammarFile | string | - | 语法文件路径（--grammar-file） |

### LoRA 适配器

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| lora | string | - | LoRA 适配器路径（--lora） |
| loraScaled | string | - | 带缩放的 LoRA 适配器（--lora-scaled） |

### 线程配置

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| threadsBatch | int | - | 批处理线程数（--threads-batch） |

### 其他参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| reranking | bool | false | 重排序模式（--reranking） |
| directIo | string | - | 直接 I/O（--dio） |

### 运行时管理

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| unloadAfterMinutes | int | - | 空闲自动卸载时间（分钟，0=永不卸载） |
| concurrencyLimit | int | - | 最大并发请求数（0=无限制） |

## 加载配置管理

模型加载参数可以持久化到服务端，下次打开加载对话框时自动恢复。

### 获取加载配置

```
GET /api/models/{id}/load-config
```

**响应**（配置存在时）：

```json
{
  "success": true,
  "data": {
    "exists": true,
    "config": {
      "id": "cfg-abc123",
      "nodeId": "node-001",
      "modelId": "model-xyz",
      "modelName": "qwen3-8b",
      "config": {
        "ctxSize": 4096,
        "gpuLayers": 99,
        "temperature": 0.7,
        "flashAttention": true
      },
      "createdAt": "2026-01-15T10:00:00Z",
      "updatedAt": "2026-01-16T14:30:00Z"
    }
  }
}
```

**响应**（配置不存在时）：

```json
{
  "success": true,
  "data": {
    "exists": false,
    "config": null
  }
}
```

### 保存加载配置

```
PUT /api/models/{id}/load-config
```

UPSERT 语义：如果该 `(node_id, model_id)` 组合已存在配置则更新，否则新建。

**请求体**：

```json
{
  "config": {
    "ctxSize": 8192,
    "gpuLayers": 99,
    "temperature": 0.5,
    "flashAttention": true,
    "parallelSlots": 4
  }
}
```

**响应**：

```json
{
  "success": true,
  "data": {
    "id": "cfg-abc123",
    "nodeId": "node-001",
    "modelId": "model-xyz",
    "modelName": "qwen3-8b",
    "config": {
      "ctxSize": 8192,
      "gpuLayers": 99,
      "temperature": 0.5,
      "flashAttention": true,
      "parallelSlots": 4
    },
    "createdAt": "2026-01-15T10:00:00Z",
    "updatedAt": "2026-01-17T09:15:00Z"
  }
}
```

### 删除加载配置

```
DELETE /api/models/{id}/load-config
```

**响应**：

```json
{
  "success": true,
  "data": null
}
```
