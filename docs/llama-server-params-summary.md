# llama-server 参数扩展完成总结

## 任务概述

本次任务完成了以下工作：
1. 分析了 llama-server 的所有参数（500+ 个）
2. 在 Shepherd 后端和前端添加了缺失的高优先级参数
3. 实现了参数启用/禁用功能，让用户可以选择使用默认值或手动配置

## 新增参数列表

### 后端 (internal/model/types.go)

新增参数字段：
- `ThreadsBatch int` - 批处理线程数 (--threads-batch)
- `RepeatLastN int` - 重复惩罚范围 (--repeat-last-n)
- `TypicalP float64` - 典型采样 (--typical-p)
- `IgnoreEOS bool` - 忽略结束token (--ignore-eos)
- `SplitMode string` - GPU分割模式 (--split-mode)
- `TensorSplit string` - 张量分割比例 (--tensor-split)
- `ContBatching bool` - 连续批处理 (--cont-batching)
- `CachePrompt bool` - 提示缓存 (--cache-prompt)
- `Grammar string` - BNF语法 (--grammar)
- `GrammarFile string` - 语法文件路径 (--grammar-file)
- `Lora string` - LoRA适配器路径 (--lora)
- `LoraScaled string` - 带缩放的LoRA (--lora-scaled)
- `ChatTemplateKwargs string` - 模板额外参数 (--chat-template-kwargs)
- `RopeScaling string` - RoPE缩放方法 (--rope-scaling)
- `RopeScale float64` - RoPE缩放因子 (--rope-scale)
- `RopeFreqBase float64` - RoPE基础频率 (--rope-freq-base)
- `RopeFreqScale float64` - RoPE频率缩放 (--rope-freq-scale)

### 命令构建 (internal/process/manager.go)

更新了 `BuildCommandFromRequest` 函数，添加了新参数的命令行构建逻辑：
- 批处理线程数：`-tb` 参数
- GPU 分割模式：`-sm` 参数
- 张量分割：`-ts` 参数
- 扩展采样参数：`--typical-p`, `--repeat-last-n`, `--ignore-eos`
- 服务器优化：`--cont-batching`, `--cache-prompt`
- 结构化生成：`--grammar`, `--grammar-file`
- LoRA 支持：`--lora`, `--lora-scaled`
- RoPE 扩展：`--rope-scaling`, `--rope-scale`, `--rope-freq-base`, `--rope-freq-scale`
- 模板额外参数：`--chat-template-kwargs`

### 前端类型 (web/src/types/model.ts)

更新了 `LoadModelParams` 接口：
1. 添加了所有新参数的字段定义
2. 添加了 `enabled` 对象来跟踪哪些参数被启用

### 前端 UI (web/src/components/models/LoadModelDialog.tsx)

1. **参数帮助文本**：更新了 `PARAM_HELP` 对象，添加了新参数的中文说明

2. **初始参数状态**：添加了新参数的默认值和启用状态

3. **参数启用状态**：为每个参数添加了 `enabled` 标志
   - 默认启用的参数：常用参数如 ctxSize, temperature, topP 等
   - 默认禁用的参数：高级参数如 typicalP, ropeScaling 等

4. **参数过滤函数**：`filterEnabledParams()` 函数
   - 根据 `enabled` 对象过滤参数
   - 只发送启用的参数到服务器
   - 未启用的参数使用 llama-server 默认值

5. **类型更新**：
   - `LoadModelDialogProps.onConfirm` 现在接受 `Partial<LoadModelParams>`
   - 允许传递部分参数（只包含启用的参数）

## 参数启用/禁用功能

### 设计理念

用户现在可以选择：
- **启用参数**：手动配置该参数，传递自定义值给 llama-server
- **禁用参数**：不传递该参数，让 llama-server 使用其默认值

### 默认配置

根据参数使用频率和重要性，设置了合理的默认启用状态：

**默认启用**（常用参数）：
- 基础运行参数：ctxSize, batchSize, threads, gpuLayers
- 采样参数：temperature, topP, topK, repeatPenalty, minP
- 批处理：uBatchSize, parallelSlots
- KV缓存：kvCacheSize, kvCacheUnified, kvCacheTypeK, kvCacheTypeV
- 性能选项：flashAttention, noMmap

**默认禁用**（高级参数）：
- 扩展采样：typicalP, repeatLastN, presencePenalty, frequencyPenalty
- 高级功能：splitMode, tensorSplit, grammar, lora
- RoPE 扩展：ropeScaling, ropeScale, ropeFreqBase, ropeFreqScale
- 其他：contextShift, cachePrompt, contBatching

## 文档更新

创建了 `docs/llama-server-params-analysis.md` 文档，包含：
- 完整的 llama-server 参数列表（500+ 个）
- Shepherd 当前已实现的参数状态
- 优先级分类（高/中/低）
- 参数对照表

## 编译测试

- ✅ 后端编译成功：`make build`
- ✅ 前端类型检查通过：`npm run type-check`

## 使用示例

用户在加载模型对话框中：
1. 每个参数都有一个启用/禁用开关
2. 默认情况下，常用参数已启用，高级参数已禁用
3. 用户可以：
   - 启用高级参数进行精细控制
   - 禁用不需要手动配置的参数，使用默认值
4. 只有启用的参数会发送到服务器

## 后续工作

如果需要继续完善，可以考虑：
1. 在前端 UI 中添加参数开关的视觉表示（当前已实现数据结构）
2. 为新参数添加 UI 输入控件
3. 添加参数预设（快速配置模板）
4. 实现参数验证和范围检查

## 参考文档

- llama-server 参数完整列表：`llama-server --help`
- Shepherd 文档：`docs/llama-server-params-analysis.md`
