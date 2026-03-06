# 模型加载 API 文档

本文档详细说明 Shepherd 模型加载 API 的所有参数。

## API 端点

```
POST /api/models/{id}/load
```

## 请求参数

### 基础参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `port` | integer | 自动分配 | 模型服务端口 (8081-9000) |
| `ctxSize` | integer | 512 | 上下文大小 (tokens) |
| `batchSize` | integer | 512 | 批次大小 |
| `threads` | integer | 4 | 线程数 |

### GPU 配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `gpuLayers` | integer | 0 | GPU 层数 (99=全部) |
| `devices` | string[] | - | GPU 设备列表 (例: ["cuda:0", "cuda:1"]) |
| `mainGPU` | integer | 0 | 主 GPU 索引 |

### 采样参数

| 参数 | 类型 | 默认值 | llama.cpp 参数 | 说明 |
|------|------|--------|----------------|------|
| `temperature` | float64 | 0.70 | `--temp` | 采样温度 |
| `topP` | float64 | 0.95 | `--top-p` | Top-P 采样 |
| `topK` | integer | 40 | `--top-k` | Top-K 采样 |
| `repeatPenalty` | float64 | 1.10 | `--repeat-penalty` | 重复惩罚 |
| `minP` | float64 | 0.00 | `--min-p` | Min-P 采样 (新) |
| `presencePenalty` | float64 | 0.00 | `--presence-penalty` | Presence 惩罚 (新) |
| `frequencyPenalty` | float64 | 0.00 | `--frequency-penalty` | Frequency 惩罚 (新) |
| `seed` | integer | 0 | `--seed` | 随机种子 |
| `nPredict` | integer | -1 | `-n` | 最大生成长度 |

### 性能优化

| 参数 | 类型 | 默认值 | llama.cpp 参数 | 说明 |
|------|------|--------|----------------|------|
| `flashAttention` | boolean | false | `-fa` | Flash Attention |
| `noMmap` | boolean | false | `--no-mmap` | 禁用内存映射 |
| `lockMemory` | boolean | false | `--mlock` | 锁定内存 |
| `ubatchSize` | integer | 0 | `--ubatch-size` | 微批次大小 |
| `parallelSlots` | integer | 0 | `--parallel` | 并行槽数 |

### KV 缓存配置 (新)

| 参数 | 类型 | 默认值 | llama.cpp 参数 | 说明 |
|------|------|--------|----------------|------|
| `kvCacheTypeK` | string | - | `--kv-cache-type-k` | K 缓存类型 (f16/q8_0) |
| `kvCacheTypeV` | string | - | `--kv-cache-type-v` | V 缓存类型 (f16/q8_0) |
| `kvCacheUnified` | boolean | false | `--kv-unified` | 统一 KV 缓存 |
| `kvCacheSize` | integer | 0 | `--kv-cache-size` | KV 缓存大小 |

### 模板系统 (新)

| 参数 | 类型 | 默认值 | llama.cpp 参数 | 说明 |
|------|------|--------|----------------|------|
| `disableJinja` | boolean | false | `--no-jinja` | 禁用 Jinja 模板 |
| `chatTemplate` | string | - | `--chat-template` | 内置模板名称 |
| `chatTemplateFile` | string | - | `--chat-template-file` | 自定义模板文件 |
| `contextShift` | boolean | false | `--context-shift` | 启用上下文切换 |

### 视觉模型

| 参数 | 类型 | 默认值 | llama.cpp 参数 | 说明 |
|------|------|--------|----------------|------|
| `mmprojPath` | string | - | `--mmproj` | 多模态项目路径 |
| `enableVision` | boolean | false | - | 启用视觉能力 |

### 服务器配置

| 参数 | 类型 | 默认值 | llama.cpp 参数 | 说明 |
|------|------|--------|----------------|------|
| `noWebUI` | boolean | false | `--no-webui` | 禁用 Web UI |
| `enableMetrics` | boolean | false | `--metrics` | 启用 /metrics 端点 |
| `slotSavePath` | string | - | `--slot-save-path` | 槽位缓存目录 |
| `cacheRAM` | integer | 0 | `--cache-ram` | RAM 缓存限制 (MB, -1=无限) |
| `timeout` | integer | 0 | `--timeout` | 读写超时 (秒) |
| `alias` | string | - | `--alias` | 模型别名 |

### 其他参数

| 参数 | 类型 | 默认值 | llama.cpp 参数 | 说明 |
|------|------|--------|----------------|------|
| `reranking` | boolean | false | `--reranking` | 重排序模式 (新) |
| `customCmd` | string | - | - | 自定义命令字符串 |
| `extraParams` | string | - | - | 额外参数字符串 |

## 请求示例

### 基础加载

```bash
curl -X POST http://localhost:9190/api/models/llama-2-7b/load \
  -H "Content-Type: application/json" \
  -d '{
    "ctxSize": 2048,
    "gpuLayers": 99
  }'
```

### 完整参数加载

```bash
curl -X POST http://localhost:9190/api/models/llama-2-7b/load \
  -H "Content-Type: application/json" \
  -d '{
    "port": 8081,
    "ctxSize": 4096,
    "batchSize": 512,
    "threads": 8,
    "gpuLayers": 99,
    "temperature": 0.7,
    "topP": 0.9,
    "topK": 40,
    "repeatPenalty": 1.1,
    "minP": 0.05,
    "presencePenalty": 0.1,
    "frequencyPenalty": 0.2,
    "flashAttention": true,
    "ubatchSize": 128,
    "kvCacheTypeK": "f16",
    "kvCacheTypeV": "f16",
    "kvCacheUnified": true,
    "disableJinja": false,
    "chatTemplate": "chatml",
    "contextShift": true,
    "reranking": false
  }'
```

### 多 GPU 配置

```bash
curl -X POST http://localhost:9190/api/models/llama-2-70b/load \
  -H "Content-Type: application/json" \
  -d '{
    "devices": ["cuda:0", "cuda:1"],
    "gpuLayers": 99,
    "ctxSize": 4096
  }'
```

## 响应

### 成功响应

```json
{
  "success": true,
  "message": "Model loaded successfully",
  "data": {
    "id": "llama-2-7b",
    "port": 8081,
    "status": "running",
    "process": {
      "pid": 12345,
      "command": "llama-server -m /path/to/model.gguf --port 8081 ..."
    }
  }
}
```

### 错误响应

```json
{
  "success": false,
  "error": "Model already loaded",
  "code": "MODEL_ALREADY_LOADED"
}
```

## 生成的命令示例

执行加载后，Shepherd 会生成类似以下的 llama-server 命令：

```bash
llama-server \
  -m /path/to/model.gguf \
  --port 8081 \
  --host 0.0.0.0 \
  -c 4096 \
  -b 512 \
  -t 8 \
  -ngl 99 \
  --temp 0.70 \
  --top-p 0.90 \
  --top-k 40 \
  --repeat-penalty 1.10 \
  --min-p 0.05 \
  --presence-penalty 0.10 \
  --frequency-penalty 0.20 \
  -fa \
  --ubatch-size 128 \
  --kv-cache-type-k f16 \
  --kv-cache-type-v f16 \
  --kv-unified \
  --chat-template chatml \
  --context-shift
```

## 不支持的参数

以下参数在当前环境中已禁用：

| 参数 | 原因 |
|------|------|
| `logitsAll` (`--logits-all`) | 仅适用于 llama-cli，不适用于 llama-server |
| `directIo` (`--dio`) | 需要特定文件系统支持 |

## 注意事项

1. **端口分配**: 如果不指定 `port`，系统会自动分配 8081-9000 范围内的可用端口
2. **GPU 层数**: 设置 `gpuLayers: 99` 表示将所有层加载到 GPU
3. **内存锁定**: `lockMemory` 可提高性能但需要足够 RAM
4. **Flash Attention**: 建议在支持的硬件上启用以提高性能
5. **KV 缓存类型**: `f16` 精度更高，`q8_0` 更省内存
