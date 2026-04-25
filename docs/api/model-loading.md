# 模型加载 API

## 加载模型

```
POST /api/models/{id}/load
```

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

### 完整参数参考

#### 基础参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| port | int | 自动分配 | 服务端口 |
| ctxSize | int | 512 | 上下文长度 |
| batchSize | int | 512 | 批处理大小 |
| threads | int | 4 | 线程数 |

#### GPU 参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| gpuLayers | int | 0 | GPU 层数（0=纯 CPU，-1=全部） |
| devices | []string | - | GPU 设备列表 |
| mainGPU | int | 0 | 主 GPU 序号 |

#### 采样参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| temperature | float | 0.7 | 温度 |
| topP | float | 0.9 | Top-P 采样 |
| topK | int | 40 | Top-K 采样 |
| repeatPenalty | float | 1.1 | 重复惩罚 |
| minP | float | 0.05 | Min-P 采样 |
| presencePenalty | float | 0 | 存在惩罚 |
| frequencyPenalty | float | 0 | 频率惩罚 |
| seed | int | -1 | 随机种子 |
| nPredict | int | -1 | 最大预测 token 数 |

#### 性能参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| flashAttention | bool | false | Flash Attention |
| noMmap | bool | false | 禁用 mmap |
| lockMemory | bool | false | 锁定内存 |
| ubatchSize | int | 512 | 微批次大小 |
| parallelSlots | int | 1 | 并行 slot 数 |

#### KV Cache 参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| kvCacheTypeK | string | f16 | K cache 类型 (f16/q8_0/q4_0) |
| kvCacheTypeV | string | f16 | V cache 类型 |
| kvCacheUnified | bool | false | 统一 KV cache |
| kvCacheSize | int | 0 | KV cache 大小 |

#### 模板参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| disableJinja | bool | false | 禁用 Jinja 模板 |
| chatTemplate | string | - | 自定义聊天模板 |
| chatTemplateFile | string | - | 模板文件路径 |
| contextShift | bool | true | 上下文移位 |

#### 视觉参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| mmprojPath | string | - | 多模态投影文件路径 |
| enableVision | bool | false | 启用视觉 |

#### 服务器参数

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| noWebUI | bool | false | 禁用内置 Web UI |
| enableMetrics | bool | false | 启用指标 |
| slotSavePath | string | - | Slot 保存路径 |
| cacheRAM | bool | false | RAM 缓存 |
| timeout | int | 600 | 超时（秒） |
| alias | string | - | 模型别名 |

#### 其他

| 参数 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| reranking | bool | false | 重排序模式 |
| customCmd | string | - | 自定义启动命令 |
| extraParams | string | - | 额外参数 |

### 响应

```json
{
  "success": true,
  "data": {
    "success": true,
    "modelId": "model-id",
    "port": 8081,
    "duration": "3.5s",
    "async": false,
    "alreadyLoaded": false
  }
}
```

### 示例

```bash
# 加载模型（GPU 99 层，4K 上下文）
curl -X POST http://localhost:9190/api/models/abc123/load \
  -H "Content-Type: application/json" \
  -d '{"ctxSize": 4096, "gpuLayers": 99}'

# 使用别名加载
curl -X POST http://localhost:9190/api/models/abc123/load \
  -d '{"alias": "my-model", "temperature": 0.5}'
```
