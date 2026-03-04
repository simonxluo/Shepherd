# llama-server 参数分析与对比

本文档分析 llama-server 的所有参数，并与 Shepherd 当前已实现的参数进行对比。

## 参数状态说明

- ✅ 已实现
- 🔄 部分实现
- ❌ 未实现
- ⚠️ 已有实现但可能需要验证

---

## 1. 通用参数 (Common Params)

### 1.1 基础运行参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--model` | `-m` | string | - | 模型文件路径 | ✅ 自动 |
| `--ctx-size` | `-c` | int | 0 (从模型加载) | 上下文大小 | ✅ CtxSize |
| `--predict` | `-n` | int | -1 (无限) | 预测token数 | ✅ NPredict |
| `--batch-size` | `-b` | int | 2048 | 批大小 | ✅ BatchSize |
| `--ubatch-size` | `-ub` | int | 512 | 微批大小 | ✅ UBatchSize |
| `--threads` | `-t` | int | -1 | CPU线程数 | ✅ Threads |
| `--threads-batch` | `-tb` | int | 同--threads | 批处理线程数 | ❌ 缺失 |
| `--seed` | `-s` | int | -1 | 随机种子 | ✅ Seed |

### 1.2 CPU 相关参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--cpu-mask` | `-C` | string | "" | CPU亲和性掩码 | ❌ 缺失 |
| `--cpu-range` | `-Cr` | string | - | CPU范围 | ❌ 缺失 |
| `--cpu-strict` | - | bool | 0 | 严格CPU放置 | ❌ 缺失 |
| `--prio` | - | int | 0 | 进程优先级 | ❌ 缺失 |
| `--poll` | - | int | 0-100 | 轮询级别 | ❌ 缺失 |
| `--cpu-mask-batch` | `-Cb` | string | 同--cpu-mask | 批CPU掩码 | ❌ 缺失 |
| `--cpu-range-batch` | `-Crb` | string | - | 批CPU范围 | ❌ 缺失 |
| `--cpu-strict-batch` | - | bool | 同--cpu-strict | 批严格放置 | ❌ 缺失 |
| `--prio-batch` | - | int | 0 | 批优先级 | ❌ 缺失 |
| `--poll-batch` | - | bool | 同--poll | 批轮询 | ❌ 缺失 |
| `--numa` | - | string | - | NUMA优化 | ❌ 缺失 |

### 1.3 RoPE 相关参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--rope-scaling` | - | string | linear | RoPE缩放方法 | ❌ 缺失 |
| `--rope-scale` | - | float | - | RoPE缩放因子 | ❌ 缺失 |
| `--rope-freq-base` | - | float | 从模型加载 | RoPE基础频率 | ❌ 缺失 |
| `--rope-freq-scale` | - | float | - | RoPE频率缩放 | ❌ 缺失 |
| `--yarn-orig-ctx` | - | int | 0 | YaRN原始上下文 | ❌ 缺失 |
| `--yarn-ext-factor` | - | float | -1.00 | YaRN外推因子 | ❌ 缺失 |
| `--yarn-attn-factor` | - | float | -1.00 | YaRN注意力因子 | ❌ 缺失 |
| `--yarn-beta-slow` | - | float | -1.00 | YaRN beta慢 | ❌ 缺失 |
| `--yarn-beta-fast` | - | float | -1.00 | YaRN beta快 | ❌ 缺失 |

### 1.4 GPU 相关参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--device` | `-dev` | string[] | - | 设备列表 | ✅ Devices |
| `--n-gpu-layers` | `-ngl` | int/string | auto | GPU层数 | ✅ GPULayers |
| `--split-mode` | `-sm` | string | layer | 分割模式 | ❌ 缺失 |
| `--tensor-split` | `-ts` | string | - | 张量分割 | ❌ 缺失 |
| `--main-gpu` | `-mg` | int | 0 | 主GPU | ✅ MainGPU |
| `--fit` | - | string | on | 自动适配显存 | ❌ 缺失 |
| `--fit-target` | `-fitt` | string | 1024 | 适配目标边距 | ❌ 缺失 |
| `--fit-ctx` | `-fitc` | int | 4096 | 适配最小上下文 | ❌ 缺失 |

### 1.5 KV 缓存参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--cache-type-k` | `-ctk` | string | f16 | K缓存类型 | ✅ KVCacheTypeK |
| `--cache-type-v` | `-ctv` | string | f16 | V缓存类型 | ✅ KVCacheTypeV |
| `--kv-unified` | `-kvu` | bool | auto | 统一KV缓存 | ✅ KVCacheUnified |
| `--kv-offload` | `-kvo` | bool | enabled | KV卸载 | ❌ 缺失 |
| `--no-kv-offload` | `-nkvo` | bool | - | 禁用KV卸载 | ❌ 缺失 |
| `--repack` | - | bool | enabled | 权重重新打包 | ❌ 缺失 |
| `--no-repack` | `-nr` | bool | - | 禁用重新打包 | ❌ 缺失 |
| `--no-host` | - | bool | - | 绕过主机缓冲 | ❌ 缺失 |

### 1.6 内存相关参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--mlock` | - | bool | - | 锁定内存 | ✅ LockMemory |
| `--mmap` | - | bool | enabled | 内存映射 | ✅ NoMmap (反向) |
| `--no-mmap` | - | bool | - | 禁用内存映射 | ✅ NoMmap |
| `--direct-io` | `-dio` | bool | disabled | 直接IO | 🔄 DirectIo (类型错误) |
| `--no-direct-io` | `-ndio` | bool | - | 禁用直接IO | ❌ 缺失 |

### 1.7 模型相关参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--override-kv` | - | string | - | 覆盖模型元数据 | ❌ 缺失 |
| `--override-tensor` | `-ot` | string | - | 覆盖张量类型 | ❌ 缺失 |
| `--lora` | - | string | - | LoRA适配器路径 | ❌ 缺失 |
| `--lora-scaled` | - | string | - | 带缩放的LoRA | ❌ 缺失 |
| `--control-vector` | - | string | - | 控制向量 | ❌ 缺失 |
| `--control-vector-scaled` | - | string | - | 带缩放的控制向量 | ❌ 缺失 |
| `--control-vector-layer-range` | - | int,int | - | 控制向量层范围 | ❌ 缺失 |
| `--cpu-moe` | `-cmoe` | bool | - | MoE权重在CPU | ❌ 缺失 |
| `--n-cpu-moe` | `-ncmoe` | int | - | 前N层MoE在CPU | ❌ 缺失 |

### 1.8 其他通用参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--verbose-prompt` | - | bool | false | 详细提示 | ❌ 缺失 |
| `--swa-full` | - | bool | false | 完整SWA缓存 | ❌ 缺失 |
| `--flash-attn` | `-fa` | string | auto | Flash Attention | ✅ FlashAttention |
| `--perf` | - | bool | false | 性能计时 | ❌ 缺失 |
| `--no-perf` | - | bool | - | 禁用性能计时 | ❌ 缺失 |
| `--escape` | `-e` | bool | true | 处理转义序列 | ❌ 缺失 |
| `--no-escape` | - | bool | - | 不处理转义 | ❌ 缺失 |
| `--keep` | - | int | 0 | 保留token数 | ❌ 缺失 |
| `--check-tensors` | - | bool | false | 检查张量 | ❌ 缺失 |
| `--op-offload` | - | bool | true | 操作卸载 | ❌ 缺失 |
| `--no-op-offload` | - | bool | - | 禁用操作卸载 | ❌ 缺失 |
| `--rpc` | - | string | - | RPC服务器 | ❌ 缺失 |

---

## 2. 采样参数 (Sampling Params)

### 2.1 基础采样参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--temp` | - | float | 0.80 | 温度 | ✅ Temperature |
| `--top-k` | - | int | 40 | Top-K采样 | ✅ TopK |
| `--top-p` | - | float | 0.95 | Top-P采样 | ✅ TopP |
| `--min-p` | - | float | 0.05 | Min-P采样 | ✅ MinP |
| `--typical-p` | - | float | 1.00 | 典型采样 | ❌ 缺失 |
| `--top-nsigma` | - | float | -1.00 | Top-N Sigma | ❌ 缺失 |

### 2.2 惩罚参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--repeat-last-n` | - | int | 64 | 惩罚最后N个 | ❌ 缺失 |
| `--repeat-penalty` | - | float | 1.00 | 重复惩罚 | ✅ RepeatPenalty |
| `--presence-penalty` | - | float | 0.00 | 存在惩罚 | ✅ PresencePenalty |
| `--frequency-penalty` | - | float | 0.00 | 频率惩罚 | ✅ FrequencyPenalty |

### 2.3 高级采样参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--xtc-probability` | - | float | 0.00 | XTC概率 | ❌ 缺失 |
| `--xtc-threshold` | - | float | 0.10 | XTC阈值 | ❌ 缺失 |
| `--dry-multiplier` | - | float | 0.00 | DRY乘数 | ❌ 缺失 |
| `--dry-base` | - | float | 1.75 | DRY基础值 | ❌ 缺失 |
| `--dry-allowed-length` | - | int | 2 | DRY允许长度 | ❌ 缺失 |
| `--dry-penalty-last-n` | - | int | -1 | DRY惩罚最后N | ❌ 缺失 |
| `--dry-sequence-breaker` | - | string | - | DRY序列断点 | ❌ 缺失 |
| `--adaptive-target` | - | float | -1.00 | 自适应目标 | ❌ 缺失 |
| `--adaptive-decay` | - | float | 0.90 | 自适应衰减 | ❌ 缺失 |
| `--dynatemp-range` | - | float | 0.00 | 动态温度范围 | ❌ 缺失 |
| `--dynatemp-exp` | - | float | 1.00 | 动态温度指数 | ❌ 缺失 |
| `--mirostat` | - | int | 0 | Mirostat采样 | ❌ 缺失 |
| `--mirostat-lr` | - | float | 0.10 | Mirostat学习率 | ❌ 缺失 |
| `--mirostat-ent` | - | float | 5.00 | Mirostat熵 | ❌ 缺失 |
| `--logit-bias` | `-l` | string | - | Logit偏置 | ❌ 缺失 |

### 2.4 结构化生成

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--grammar` | - | string | - | BNF语法 | ❌ 缺失 |
| `--grammar-file` | - | string | - | 语法文件 | ❌ 缺失 |
| `--json-schema` | `-j` | string | - | JSON模式 | ❌ 缺失 |
| `--json-schema-file` | `-jf` | string | - | JSON模式文件 | ❌ 缺失 |

### 2.5 其他采样参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--samplers` | - | string | 默认序列 | 采样器序列 | ❌ 缺失 |
| `--sampler-seq` | - | string | edskypmxt | 简化采样序列 | ❌ 缺失 |
| `--ignore-eos` | - | bool | false | 忽略EOS | ❌ 缺失 |
| `--backend-sampling` | `-bs` | bool | false | 后端采样 | ❌ 缺失 |

---

## 3. 服务器特有参数 (Server-Specific Params)

### 3.1 多模态参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--mmproj` | `-mm` | string | - | 多模态投影文件 | ✅ MmprojPath |
| `--mmproj-url` | `-mmu` | string | - | 多模态URL | ❌ 缺失 |
| `--mmproj-auto` | - | bool | enabled | 自动多模态 | ❌ 缺失 |
| `--no-mmproj-auto` | - | bool | - | 禁用自动多模态 | ❌ 缺失 |
| `--mmproj-offload` | - | bool | enabled | 多模态GPU卸载 | ❌ 缺失 |
| `--no-mmproj-offload` | - | bool | - | 禁用多模态卸载 | ❌ 缺失 |
| `--image-min-tokens` | - | int | 从模型 | 图像最小token | ❌ 缺失 |
| `--image-max-tokens` | - | int | 从模型 | 图像最大token | ❌ 缺失 |

### 3.2 服务器配置

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--host` | - | string | 127.0.0.1 | 监听地址 | ✅ 自动分配 |
| `--port` | - | int | 8080 | 监听端口 | ✅ 自动分配 |
| `--path` | - | string | - | 静态文件路径 | ❌ 缺失 |
| `--api-prefix` | - | string | - | API前缀 | ❌ 缺失 |
| `--alias` | `-a` | string | - | 模型别名 | ✅ Alias |
| `--tags` | - | string | - | 模型标签 | ❌ 缺失 |
| `--timeout` | `-to` | int | 600 | 超时(秒) | ✅ Timeout |
| `--threads-http` | - | int | -1 | HTTP线程数 | ❌ 缺失 |

### 3.3 批处理和缓存

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--parallel` | `-np` | int | -1 (auto) | 并发槽位数 | ✅ ParallelSlots |
| `--cont-batching` | `-cb` | bool | enabled | 连续批处理 | ❌ 缺失 |
| `--no-cont-batching` | `-nocb` | bool | - | 禁用连续批处理 | ❌ 缺失 |
| `--cache-prompt` | - | bool | enabled | 提示缓存 | ❌ 缺失 |
| `--no-cache-prompt` | - | bool | - | 禁用提示缓存 | ❌ 缺失 |
| `--cache-reuse` | - | int | 0 | 缓存复用大小 | ❌ 缺失 |
| `--cache-ram` | `-cram` | int | 8192 | 缓存RAM限制 | ✅ CacheRAM |

### 3.4 端点配置

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--metrics` | - | bool | false | Prometheus指标 | ✅ EnableMetrics |
| `--props` | - | bool | false | 属性端点 | ❌ 缺失 |
| `--slots` | - | bool | enabled | 槽位端点 | ❌ 缺失 |
| `--no-slots` | - | bool | - | 禁用槽位端点 | ❌ 缺失 |
| `--slot-save-path` | - | string | - | 槽位保存路径 | ✅ SlotSavePath |

### 3.5 Web UI 配置

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--webui` | - | bool | enabled | 启用Web UI | ✅ NoWebUI (反向) |
| `--no-webui` | - | bool | - | 禁用Web UI | ✅ NoWebUI |
| `--webui-config` | - | string | - | Web UI配置 | ❌ 缺失 |
| `--webui-config-file` | - | string | - | Web UI配置文件 | ❌ 缺失 |

### 3.6 模型路由器

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--models-dir` | - | string | - | 模型目录 | ❌ 缺失 |
| `--models-preset` | - | string | - | 模型预设文件 | ❌ 缺失 |
| `--models-max` | - | int | 4 | 最大模型数 | ❌ 缺失 |
| `--models-autoload` | - | bool | enabled | 自动加载模型 | ❌ 缺失 |
| `--no-models-autoload` | - | bool | - | 禁用自动加载 | ❌ 缺失 |

### 3.7 特殊功能

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--embedding` | - | bool | false | 仅嵌入模式 | ✅ embedding能力 |
| `--rerank` | - | bool | false | 重排端点 | ✅ Reranking |
| `--api-key` | - | string | - | API密钥 | ❌ 缺失 |
| `--api-key-file` | - | string | - | API密钥文件 | ❌ 缺失 |
| `--ssl-key-file` | - | string | - | SSL密钥文件 | ❌ 缺失 |
| `--ssl-cert-file` | - | string | - | SSL证书文件 | ❌ 缺失 |
| `--reasoning-format` | - | string | auto | 推理格式 | ❌ 缺失 |
| `--reasoning-budget` | - | int | -1 | 推理预算 | ❌ 缺失 |

### 3.8 聊天模板

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--chat-template` | - | string | 从模型 | 聊天模板 | ✅ ChatTemplate |
| `--chat-template-file` | - | string | 从模型 | 聊天模板文件 | ✅ ChatTemplateFile |
| `--chat-template-kwargs` | - | string | - | 模板额外参数 | ❌ 缺失 |
| `--jinja` | - | bool | enabled | Jinja模板引擎 | ✅ DisableJinja (反向) |
| `--no-jinja` | - | bool | - | 禁用Jinja | ✅ DisableJinja |

### 3.9 其他服务器参数

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--media-path` | - | string | - | 媒体文件目录 | ❌ 缺失 |
| `--context-shift` | - | bool | disabled | 上下文移位 | ✅ ContextShift |
| `--no-context-shift` | - | bool | - | 禁用上下文移位 | ❌ 缺失 |
| `--prefill-assistant` | - | bool | enabled | 预填充助手 | ❌ 缺失 |
| `--no-prefill-assistant` | - | bool | - | 禁用预填充 | ❌ 缺失 |
| `--slot-prompt-similarity` | `-sps` | float | 0.10 | 槽位提示相似度 | ❌ 缺失 |
| `--lora-init-without-apply` | - | bool | false | 加载LoRA不应用 | ❌ 缺失 |
| `--sleep-idle-seconds` | - | int | -1 | 空闲睡眠秒数 | ❌ 缺失 |
| `--special` | `-sp` | bool | false | 特殊token输出 | ❌ 缺失 |
| `--warmup` | - | bool | enabled | 预热 | ❌ 缺失 |
| `--no-warmup` | - | bool | - | 禁用预热 | ❌ 缺失 |
| `--lookup-cache-static` | `-lcs` | string | - | 静态查找缓存 | ❌ 缺失 |
| `--lookup-cache-dynamic` | `-lcd` | string | - | 动态查找缓存 | ❌ 缺失 |
| `--ctx-checkpoints` | - | int | 8 | 上下文检查点 | ❌ 缺失 |
| `--swa-checkpoints` | - | int | 8 | SWA检查点 | ❌ 缺失 |

---

## 4. 投机解码参数 (Speculative Decoding)

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--model-draft` | `-md` | string | - | 草稿模型 | ❌ 缺失 |
| `--draft-max` | `--draft-n` | int | 16 | 最大草稿token | ❌ 缺失 |
| `--draft-min` | `--draft-n-min` | int | 0 | 最小草稿token | ❌ 缺失 |
| `--draft-p-min` | - | float | 0.75 | 最小草稿概率 | ❌ 缺失 |
| `--ctx-size-draft` | `-cd` | int | 0 | 草稿上下文 | ❌ 缺失 |
| `--device-draft` | `-devd` | string | - | 草稿设备 | ❌ 缺失 |
| `--gpu-layers-draft` | `-ngld` | int/string | auto | 草稿GPU层数 | ❌ 缺失 |
| `--threads-draft` | `-td` | int | 同--threads | 草稿线程数 | ❌ 缺失 |
| `--threads-batch-draft` | `-tbd` | int | - | 草稿批线程 | ❌ 缺失 |
| `--cache-type-k-draft` | `-ctkd` | string | f16 | 草稿K缓存类型 | ❌ 缺失 |
| `--cache-type-v-draft` | `-ctvd` | string | f16 | 草稿V缓存类型 | ❌ 缺失 |
| `--override-tensor-draft` | `-otd` | string | - | 覆盖草稿张量 | ❌ 缺失 |
| `--cpu-moe-draft` | `-cmoed` | bool | - | 草稿MoE在CPU | ❌ 缺失 |
| `--n-cpu-moe-draft` | `-ncmoed` | int | - | 草稿MoE层数 | ❌ 缺失 |
| `--spec-replace` | - | string,string | - | 规格替换 | ❌ 缺失 |
| `--spec-type` | - | string | none | 规格类型 | ❌ 缺失 |
| `--spec-ngram-size-n` | - | int | 12 | ngram大小N | ❌ 缺失 |
| `--spec-ngram-size-m` | - | int | 48 | ngram大小M | ❌ 缺失 |
| `--spec-ngram-min-hits` | - | int | 1 | ngram最小命中 | ❌ 缺失 |

---

## 5. 其他特殊模型

| 参数 | 短选项 | 类型 | 默认值 | 说明 | Shepherd |
|------|--------|------|--------|------|----------|
| `--model-vocoder` | `-mv` | string | - | 声码器模型 | ❌ 缺失 |
| `--tts-use-guide-tokens` | - | bool | - | TTS引导token | ❌ 缺失 |
| `--embd-gemma-default` | - | bool | - | 默认嵌入Gemma | ❌ 缺失 |
| `--fim-qwen-*` | - | bool | - | Qwen FIM模型 | ❌ 缺失 |
| `--gpt-oss-*` | - | bool | - | GPT-OSS模型 | ❌ 缺失 |
| `--vision-gemma-*` | - | bool | - | 视觉Gemma | ❌ 缺失 |

---

## 总结

### 已实现参数 (约 40 个)
- 基础运行参数: ctxSize, batchSize, threads, seed, nPredict
- GPU 参数: gpuLayers, devices, mainGPU
- 采样参数: temperature, topP, topK, minP, repeatPenalty, presencePenalty, frequencyPenalty
- KV 缓存: kvCacheTypeK/V, kvCacheUnified, kvCacheSize
- 批处理: uBatchSize, parallelSlots
- 性能: flashAttention, noMmap, lockMemory
- 服务器: timeout, alias, noWebUI, enableMetrics, slotSavePath, cacheRAM
- 多模态: mmprojPath, enableVision
- 模板: chatTemplate, chatTemplateFile, disableJinja
- 其他: contextShift, logitsAll, reranking

### 高优先级缺失参数 (推荐添加)
1. **threadsBatch** - 批处理线程数
2. **repeatLastN** - 重复惩罚的最后N个token
3. **typicalP** - 典型采样
4. **splitMode** - 多GPU分割模式
5. **tensorSplit** - 张量分割比例
6. **cachePrompt** - 提示缓存开关
7. **contBatching** - 连续批处理开关
8. **ignoreEOS** - 忽略结束token
9. **grammar** - BNF语法约束
10. **lora** - LoRA适配器支持

### 中优先级缺失参数
- RoPE 扩展参数 (rope-scaling, rope-scale 等)
- 高级采样 (xtc, dry, mirostat 等)
- 投机解码参数
- NUMA 和 CPU 亲和性参数

### 低优先级缺失参数
- 结构化生成 (json-schema)
- 模型路由器参数
- SSL/TLS 配置
- 日志详细控制

### 已实现但需要验证的参数
- **DirectIo**: 当前类型是 string，应该是 bool
- **KVCacheSize**: 需要确认单位是否正确
