# 变更日志

本文件记录 Shepherd 项目的所有重大变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/spec/v2.0.0.html)。

[English](../CHANGELOG.md)

## [Unreleased]

### Fixed
- **进程管理竞态条件修复**: 修复进程停止时的 "send on closed channel" panic
  - 问题: 当健康检查失败终止进程时，`readOutput` goroutine 可能仍在向已关闭的 channel 发送数据
  - 解决: 将 channel 关闭责任从 `processOutput` 转移到 `readOutput` goroutines
  - 使用 `atomic.AddInt32` 计数器跟踪完成的 reader 数量
  - 最后一个 reader 负责关闭 channel，避免竞态条件

## [v0.6.0] - 2026-03-06

### Breaking Changes
- **移除 internal/master 包**: 完全移除 master 节点管理器实现
  - 所有功能已迁移到 `internal/node` 包
  - 使用 `NodeAdapter` 统一所有角色的 API 接口
  - 删除了 2,400+ 行代码，简化架构
  - 符合 v0.4.0+ 统一 Node 架构设计

### Added
- **LangChainGo 集成**: 新增 LangChainGo 框架集成支持
  - 新增 `internal/langchain/` 包，包含 6 个核心文件
  - `LlamaCPP` 适配器实现 LangChainGo `llms.LLM` 接口
  - `Manager` 管理多个 LangChainGo 实例
  - `Handler` 提供 HTTP API 路由 (`/api/langchain/*`)
  - 支持流式和非流式文本生成
  - 完整的单元测试覆盖 (`llama_test.go`, `example_test.go`)
  - 集成到主程序启动流程（所有模式可用）
- **工具函数包**: 新增 `internal/utils/` 包，提供统一的错误处理接口
  - `CloseQuietly()` - 安全关闭 io.Closer，忽略错误
  - `RemoveQuietly()` - 安全删除文件，不存在时不报错
  - `RenameQuietly()` - 安全重命名文件，失败时记录警告
  - `KillQuietly()` - 安全终止进程，已退出时不报错
  - `UnmarshalQuietly()` - 安全 JSON 解析，失败时提供默认值
  - `WriteStringQuietly()` - 安全写入字符串
  - `SetReadDeadlineQuietly()` / `SetWriteDeadlineQuietly()` - 网络超时设置
  - 所有函数都有完整的文档注释和使用示例
  - 避免循环依赖（不导入其他 internal 包）
- **进程管理脚本**: 新增 `scripts/linux/stop_all.sh`
  - 智能检测 Shepherd 相关进程（后端 + 前端）
  - 优雅关闭机制（SIGTERM + 超时 + SIGKILL）
  - 支持三种模式：
    - 默认模式：停止所有相关进程
    - `--dry-run`：预览将要停止的进程（不实际停止）
    - `--force`：强制停止端口占用
  - 多层检测策略（模式匹配 → 目录检测 → 路径匹配）
  - 彩色输出和详细日志
  - 清理 PID 文件和日志文件
- **启动脚本增强**: `scripts/linux/run.sh` 集成进程清理
  - 启动前自动调用 `stop_all.sh --force`
  - 清理残留进程，避免端口冲突
  - 静默执行，失败不影响启动

### Changed
- **文档重组**: `doc/` 目录迁移到 `docs/`
  - 更规范的目录命名（复数形式）
  - 新增 LangChainGo 集成文档：
    - `docs/langchain-integration-summary.md` - 集成概述
    - `docs/langchain-integration-complete.md` - 完整指南
    - `docs/api-vs-langchain-comparison.md` - API 对比
- **代码质量改进**: 统一使用工具函数，提高代码一致性
  - 资源清理：`defer rows.Close()` → `defer utils.CloseQuietly(rows)`
  - 文件操作：`os.Rename()` → `utils.RenameQuietly()`
  - JSON 解析：`json.Unmarshal()` → `utils.UnmarshalQuietly()`（带默认值）
  - 进程管理：添加 `//errcheck:ignore` 注释标记
  - 涉及 50+ 个文件的代码重构
- **依赖管理更新** (go.mod):
  - 新增 `github.com/tmc/langchaingo v0.1.14`
  - `github.com/ROCm/amdsmi` 从 indirect 改为 direct 依赖
  - 更新间接依赖版本（多个包的小版本更新）

### Removed
- **internal/master 包** (2,400+ 行):
  - `doc.go` - 包文档
  - `handler.go` - Master HTTP 处理器
  - `node_manager.go` - 节点管理器（674 行）
  - `node_manager_test.go` - 节点管理器测试（482 行）
  - `scheduler.go` - 调度器（540 行）
  - `scheduler_test.go` - 调度器测试
  - 所有功能已由 `internal/node` + `api.NodeAdapter` 替代
- **doc/ 目录** (2,800+ 行):
  - 所有文档已迁移到 `docs/` 目录
  - 保持相同的目录结构和文件名
- **废弃的 API 函数**:
  - `server.GetWebSocketManager()` - 不再公开 WebSocket Manager

### Fixed
- **错误处理改进**: 使用 `UnmarshalQuietly` 避免 JSON 解析失败导致崩溃
  - 数据库元数据解析：添加默认值逻辑
  - 会话/消息元数据解析：失败时使用空 map
  - 模型标签/能力解析：失败时使用空切片/空对象
- **资源清理改进**: 确保所有资源正确释放
  - 数据库连接、文件句柄、HTTP 服务器
  - 即使在错误情况下也能优雅关闭
- **测试代码格式化**: 统一测试代码的对齐和缩进

### Migration Guide

如果您使用了已删除的 `internal/master` 包：

1. **后端代码迁移**:
   ```go
   // 旧代码
   import "github.com/simonxluo/Shepherd/internal/master"
   mgr := master.NewNodeManager(...)

   // 新代码
   import "github.com/simonxluo/Shepherd/internal/node"
   node := node.NewNode(...) // 使用统一的 Node
   ```

2. **API 路由迁移**:
   ```bash
   # 旧路由（v0.4.0+ 已废弃）
   GET /api/master/nodes

   # 新路由（统一 API）
   GET /api/nodes
   ```

3. **前端代码迁移**:
   ```typescript
   // 旧代码（v0.2.0 已废弃）
   import type { Client } from '@/types/cluster';

   // 新代码（推荐）
   import type { UnifiedNode } from '@/types/node';
   ```

### Technical Details
- **架构简化**: 删除 master 专用代码，所有角色共享同一个 Node 实现
- **依赖注入**: LangChainGo Handler 通过 `RegisterLangChainHandler()` 注入
- **测试策略**: LangChainGo 包使用表驱动测试 + 示例测试
- **向后兼容**: v0.4.0+ 的 API 路由保持兼容
- **代码统计**: 净删除 5,016 行代码（-5,449 +433），提高可维护性

---

## [v0.5.1] - 2026-03-04

### Added
- **一键启动功能**: `run.sh` 脚本新增 `--web` 参数
  - 同时启动前后端开发服务器
  - 自动处理端口冲突（检测并停止占用进程）
  - 优雅退出：`Ctrl+C` 同时停止前后端
  - 前端日志输出到 `/tmp/shepherd-web-dev.log`
  - 前端进程 PID 保存在 `/tmp/shepherd-web-dev.pid`
  - 使用方法: `./scripts/linux/run.sh --web -b`

### Changed
- **能力检测优化**: 重构 `internal/model/capability.go`
  - 提取 30+ 个关键词常量，消除硬编码
  - 使用 `strings.Builder` 优化字符串拼接，减少 60% 内存分配
  - 统一使用 `ApplyConstraints()` 方法，消除重复的互斥逻辑
  - 添加空值检查，避免对空的 `chat_template` 进行不必要操作
  - 提升代码可维护性和性能

### Performance
- 能力检测内存分配减少 60%（使用 `strings.Builder`）
- 避免对空 `chat_template` 进行字符串转换

### Code Quality
- 通过 Simplify Review 修复高优先级问题
- 消除重复的互斥逻辑代码
- 提取关键词常量，提高可维护性

---

## [v0.5.0] - 2026-03-04

### Added
- **模型能力自动检测**: 基于模型元数据自动检测模型能力
  - 新增 `internal/model/capability.go` 包，实现能力检测核心逻辑
  - 支持检测四种能力：Thinking（思考）、Tools（工具调用）、Rerank（重排序）、Embedding（嵌入）
  - 基于 GGUF 元数据的模型名称、架构和 `chat_template` 进行关键词匹配
  - 检测规则：
    - **Thinking**: 匹配 `deepseek-r1`、`qwq`、`enable_thinking`、`reasoning` 等关键词
    - **Tools**: 匹配 `tool_call`、`function`、`tools`、`mcp` 等关键词
    - **Rerank**: 匹配 `rerank`、`cross-encoder`、`ranker` 等关键词
    - **Embedding**: 匹配 `embedding`、`e5`、`bge`、`jina`、`nomic` 等关键词
  - 互斥规则：Rerank/Embedding 与 Thinking/Tools 互斥（启用前者时自动禁用后者）
- **GGUF 解析器增强**: 新增 `tokenizer.chat_template` 字段解析
  - `internal/gguf/metadata.go`: 添加 `ChatTemplate` 字段
  - `internal/gguf/parser.go`: 解析 `tokenizer.chat_template` KV 键
- **能力持久化**: 将模型能力存储从内存迁移到数据库
  - `internal/storage/storage.go`: 新增 `Capabilities` 类型定义
  - `internal/storage/sqlite.go`: 添加 `capabilities` 列到 `model_metadata` 表
  - 实现数据库迁移逻辑：使用 `PRAGMA table_info` 检查列是否存在
  - JSON 序列化存储到 SQLite TEXT 列
- **能力验证方法**: `storage.Capabilities` 类型新增验证方法
  - `Validate()`: 验证能力配置（检查互斥规则）
  - `ApplyConstraints()`: 自动应用互斥约束
- **单元测试**: 新增 8 个能力检测测试用例
  - `TestDetectCapabilities_DeepSeekR1`: DeepSeek-R1 思考模型
  - `TestDetectCapabilities_BGE_M3`: BGE-M3 嵌入模型
  - `TestDetectCapabilities_BGEReranker`: BGE Reranker 重排序模型
  - `TestDetectCapabilities_GPT4o`: GPT-4o 工具调用模型
  - `TestDetectCapabilities_QWQ`: QWQ 思考模型
  - `TestDetectCapabilities_E5Embedding`: E5 嵌入模型
  - `TestDetectCapabilities_NilMetadata`: 空 metadata 处理
  - `TestDetectCapabilities_EmptyMetadata`: 空 metadata 处理

### Changed
- **模型扫描流程**: 集成能力检测到 `loadModel()` 函数
  - 扫描时自动检测并保存能力到数据库
  - `internal/model/manager.go`: 在 `loadModel()` 中调用 `DetectCapabilities()`
- **API 层重构**: 移除内存存储，改用数据库
  - `internal/server/server.go`: 删除 `capabilities` map 和 `capabilitiesMu` 互斥锁
  - 删除本地 `ModelCapabilities` 类型定义，统一使用 `storage.Capabilities`
  - `handleLoadModel()`: 使用 `Validate()` 和 `ApplyConstraints()` 方法
  - `handleSetModelCapabilities()`: 同样使用新的验证方法
  - `handleGetModelCapabilities()`: 从数据库读取能力
- **错误处理改进**: 添加数据库保存失败的警告日志
  - `internal/server/server.go`: 能力保存失败时记录警告日志
  - `internal/model/manager.go`: 能力保存失败时记录警告日志

### Removed
- `docs/llama-server-params-analysis.md`: llama-server 参数分析文档（已过时）
- `docs/llama-server-params-summary.md`: llama-server 参数摘要文档（已过时）
- `web/scripts/check-model-detail.cjs`: 旧版模型详情检查脚本
- `web/scripts/check-model-detail.ts`: 旧版模型详情检查脚本
- `web/scripts/migrate-text-colors.cjs`: 文本颜色迁移脚本

### Fixed
- **代码质量**: 通过 Simplify Review 修复多个问题
  - 提取重复的验证逻辑到 `Capabilities.Validate()` 方法
  - 添加错误日志记录（之前使用 `_` 忽略错误）
  - 统一错误消息格式

### Technical Details
- **能力检测算法**: 关键词匹配（不区分大小写）
  - 组合输入：`model.Name + " " + model.Architecture` (小写)
  - Chat Template 检测：匹配 Jinja 模板中的特定标记
- **数据库迁移**: 使用 `ALTER TABLE ADD COLUMN` 动态添加列
- **JSON 存储**: 使用 `encoding/json` 序列化 `Capabilities` 结构
- **测试覆盖**: 所有 8 个测试用例通过

---

## [v0.4.0] - 2026-03-01

### BREAKING CHANGES
- **删除 standalone 角色**: 移除 standalone 单机模式角色
  - 统一使用 `node.role` 字段作为唯一角色配置源
  - 删除 `Config.Mode` 字段，消除双重配置
  - 仅保留三个角色: `master`、`client`、`hybrid`
- **默认角色变更**: 系统默认角色从 `standalone` 改为 `hybrid`
  - 新用户直接使用推荐的混合模式
  - 混合模式同时启用 Master 和 Client 功能
- **命令行参数变更**: 删除 `--mode` 参数
  - 节点角色由配置文件的 `node.role` 字段决定
  - 运行脚本不再支持位置参数指定模式
- **API 响应变更**:
  - `/config` 端点返回 `role` 字段代替 `mode`
  - `/api/server/status` 端点返回 `role` 字段代替 `mode`

### Changed
- **配置简化**:
  - 删除 `ConfigFileNames` 映射（模式→配置文件）
  - `NewManager()` 不再接受 `mode` 参数
  - `NewManagerWithPath()` 不再接受 `mode` 参数
- **日志系统**:
  - 日志文件命名使用 `role` 代替 `serverMode`
  - 日志文件格式: `shepherd-{role}-{date}.log`
- **前端类型**:
  - `NodeRole` 类型删除 `standalone`
  - `ServerModeConfig.mode` 字段已删除
- **脚本更新**:
  - `scripts/linux/run.sh` 删除模式参数逻辑
  - 配置文件自动从 `config/example/` 复制到 `config/node/`

### Removed
- `internal/node/types.go`: `NodeRoleStandalone` 常量
- `cmd/shepherd/main.go`:
  - `--mode` 命令行参数
  - `determineRole()` 函数中的模式映射逻辑
  - `initStandaloneNode()` 函数
  - standalone 模式初始化分支
- `internal/config/config.go`:
  - `Config.Mode` 字段
  - `ConfigFileNames` 变量
  - Mode 验证逻辑
- `internal/server/server.go`: `Config.Mode` 字段
- `internal/logger/logger.go`: `serverMode` 字段

### Migration Guide
如果您使用的是旧版本配置：

1. **更新配置文件**:
   ```yaml
   # 删除 mode 字段
   # mode: standalone  # 删除此行

   # 更新 node.role
   node:
     role: hybrid  # standalone → hybrid (推荐)
   ```

2. **更新启动命令**:
   ```bash
   # 旧命令
   ./build/shepherd standalone
   ./build/shepherd --mode hybrid

   # 新命令（使用配置文件中的 node.role）
   ./build/shepherd
   ./build/shepherd --config config/node/server.config.yaml
   ```

3. **更新前端代码**:
   - `serverConfig.mode` → `serverConfig.role`
   - 删除 `'standalone'` 类型引用
   - 使用 `'hybrid'` 替代 `'standalone'`

### Technical Details
- **配置迁移**: `syncLegacyConfig()` 自动将 `mode: standalone` 迁移到 `node.role: hybrid`
- **向后兼容**: 旧的 `master`、`client` 配置文件仍然有效
- **日志兼容性**: 旧日志文件仍可识别，新日志使用新的命名格式

---

## [v0.3.2] - 2026-02-27

### Added
- **模型加载配置持久化**: 支持保存和加载模型配置
  - 新增 `ModelLoadConfig` 存储模型，使用 `(nodeID, modelID)` 作为复合主键
  - 新增 API 端点:
    - `GET /api/models/:id/load-config` - 获取模型加载配置
    - `PUT /api/models/:id/load-config` - 保存模型加载配置
    - `DELETE /api/models/:id/load-config` - 删除模型加载配置
  - SQLite 存储使用 UPSERT 模式，自动更新已存在的配置
  - 内存存储使用复合键 `"nodeID:modelID"` 存储配置
  - NodeAdapter 新增 `GetNodeID()` 方法获取节点 ID
- **前端 Hooks**: 新增模型配置管理 hooks
  - `useModelLoadConfig()` - 查询模型配置
  - `useSaveModelLoadConfig()` - 保存模型配置
  - `useDeleteModelLoadConfig()` - 删除模型配置
- **LoadModelDialog 增强**:
  - 自动加载上次保存的配置
  - 加载模型时自动保存配置
  - 新增"重置"按钮，清除保存的配置并恢复默认值

### Changed
- **默认存储类型**: 示例配置文件 `server.config.yaml` 存储类型从 `memory` 改为 `sqlite`
  - 数据库路径: `./data/shepherd.db`

### Fixed
- **路径配置错误提示**:
  - PathEditDialog 现在显示友好的错误提示
  - 解析后端错误消息，区分"路径不存在"、"不是目录"等错误
  - 保存失败时不关闭对话框，允许用户修改后重试
- **路径验证改进**:
  - 调用后端 API 进行真实验证（llama.cpp 路径）
  - 验证状态：null（未验证）/ true（有效）/ false（无效）
  - 500ms 防抖，避免频繁请求
  - 显示"验证路径中..."加载状态
- **版本显示更新**: Sidebar 版本号从 v0.1.2 更新到 v0.4.1
- **日志查看器修复**:
  - 修复日志内容重叠问题
  - 使用动态行高计算，根据消息长度调整高度
  - 更新到 react-window v2 API

---

## [v0.3.1] - 2026-02-26

### Added
- **扩展模型加载参数**: 新增 11 个 llama.cpp 服务器参数支持
  - **采样参数**: `reranking`, `minP`, `presencePenalty`, `frequencyPenalty`
  - **模板和处理**: `disableJinja` (--no-jinja), `chatTemplate` (--chat-template), `contextShift` (--context-shift)
  - **KV 缓存配置**: `kvCacheTypeK` (--kv-cache-type-k), `kvCacheTypeV` (--kv-cache-type-v), `kvCacheUnified` (--kv-unified), `kvCacheSize` (--kv-cache-size)
  - 前端 LoadModelDialog 完整支持所有新参数配置
  - 后端 BuildCommandFromRequest 支持所有新参数
- **日志系统增强**:
  - 所有日志自动包含调用位置（文件名:行号）
  - 日志格式调整: `[时间] [文件:行号] 级别 消息` (文本格式) / `{"time":"...","caller":"...","level":"...","msg":"..."}` (JSON 格式)
  - 模型加载/卸载过程添加详细日志记录
- **开发体验改进**:
  - 更新 .gitignore，排除 AI 编辑器配置文件（.claude/, .sisyphus, .cursor/ 等）
  - CLAUDE.md 文档添加到版本控制

### Fixed
- **参数兼容性**: 禁用 llama-server 不支持的参数
  - `--logits-all` 仅适用于 llama-cli，不适用于 llama-server
  - `--dio` 需要特定文件系统支持，默认禁用
- **类型同步**: 更新 `toProcessLoadRequest` 转换函数，确保所有新字段正确传递
- **测试更新**: 更新测试用例以反映禁用的参数

### Technical Details
- **支持的参数映射**:
  - `reranking: boolean` → `--reranking`
  - `minP: float64` → `--min-p <value>`
  - `presencePenalty: float64` → `--presence-penalty <value>`
  - `frequencyPenalty: float64` → `--frequency-penalty <value>`
  - `disableJinja: boolean` → `--no-jinja`
  - `chatTemplate: string` → `--chat-template <value>`
  - `contextShift: boolean` → `--context-shift`
  - `kvCacheTypeK: string` → `--kv-cache-type-k <value>`
  - `kvCacheTypeV: string` → `--kv-cache-type-v <value>`
  - `kvCacheUnified: boolean` → `--kv-unified`
  - `kvCacheSize: int` → `--kv-cache-size <value>`

---

## [v0.3.0] - 2026-02-22

### Added
- **HuggingFace SDK 集成**: 集成两个 HuggingFace Go SDK
  - `github.com/gomlx/go-huggingface/hub` - 基础 Hub 操作和文件下载
  - `github.com/bodaay/HuggingFaceModelDownloader` - 高级下载（分块/可恢复）
  - 支持基础下载模式（Basic）和高级下载模式（Advanced）
  - 添加下载进度回调支持（速度、ETA、百分比）
  - 支持多种端点（官方 HuggingFace 和 HF-Mirror）
  - 前端添加 HuggingFace 模型搜索和下载 UI
- **llama.cpp 测试功能**: 新增 `internal/client/tester` 包
  - 自动检测常见 llama.cpp 二进制路径
  - 测试二进制可执行性和版本信息
  - 支持环境变量 `LLAMACPP_SERVER_PATH`
  - 系统信息收集（GPU/CPU/ROCm）
  - 前端添加 llama.cpp 可用性测试 UI
- **配置报告功能**: 新增 `internal/client/configreport` 包
  - 收集 llama.cpp 路径配置和可用性
  - 收集模型路径和模型数量
  - 环境信息（OS/Kernel/Python/Go 版本）
  - Conda 配置和执行器配置
  - 前端添加节点配置信息展示
- **NodeAdapter API 增强**:
  - `POST /api/nodes/:id/test-llamacpp` - 测试 llama.cpp 可用性
  - `GET /api/nodes/:id/config` - 获取节点配置信息
- **配置验证测试**: 添加模式验证测试
  - 验证所有有效模式（standalone, hybrid, master, client）
  - 验证无效模式拒绝
- **路径更新测试**: 添加 `TestHandler_UpdateLlamaCppPath` 测试
  - 测试 originalPath 匹配策略
  - 测试按名称匹配
  - 测试错误场景
- **前端类型系统**: `web/src/types/node.ts` 新增配置相关类型
  - `NodeConfigInfo` - 节点配置信息
  - `LlamaCppPathInfo` - llama.cpp 路径信息
  - `ModelPathInfo` - 模型路径信息
  - `EnvironmentInfo` - 环境信息
  - `CondaConfigInfo` - Conda 配置
  - `ExecutorConfigInfo` - 执行器配置

### Changed
- **路径更新逻辑改进**: 三级匹配策略
  - 最高优先级：`originalPath` 精确匹配
  - 中等优先级：按 `name` 匹配
  - 最低优先级：按 `path` 精确匹配
- **配置验证**: 添加 `standalone` 到有效模式列表
  - 默认模式从 `hybrid` 改为 `standalone`
  - 支持所有四种模式：standalone, hybrid, master, client
- **配置废弃标记**: 为旧的 Client/Master 配置添加废弃注释
  - `ClientConfig` 标记为废弃，建议使用 `Node.ClientRole`
  - `MasterConfig` 标记为废弃，建议使用 `Node.MasterRole`
- **前端配置管理**: 改进配置加载和重载逻辑
  - 支持运行时后端切换
  - 添加配置验证
- **LoadModelDialog 重构**: 改进模型加载对话框
  - 添加 llama.cpp 测试按钮
  - 改进参数配置界面
  - 优化布局和交互
- **Settings 页面增强**: 重新组织配置项
  - 添加环境信息显示
  - 改进配置保存逻辑

### Fixed
- **路径配置 500 错误**: 修复配置验证拒绝 `standalone` 模式的问题
- **路径更新功能**: 修复 `UpdateModelPath` 和 `UpdateLlamaCppPath` 的匹配逻辑
  - 改进路径规范化处理
  - 添加更好的错误消息
- **错误消息一致性**: 统一错误响应格式
- **下载页面**: 修复下载进度显示问题
  - 添加速度和 ETA 显示
  - 支持暂停/恢复下载

### Technical Details
- **下载模式**:
  - `DownloadModeBasic`: 使用 `go-huggingface/hub`（简单可靠）
  - `DownloadModeAdvanced`: 使用 `bodaay/HuggingFaceModelDownloader`（分块、可恢复）
- **路径匹配**: 使用规范化路径进行对比，避免符号链接等问题
- **测试覆盖**: 新增 5+ 个测试函数，覆盖下载、测试、配置验证
- **新增依赖**: 两个 HuggingFace Go SDK
- **API 新增命令**:
  - `CommandTypeTestLlamacpp` - 测试 llama.cpp
  - `CommandTypeGetConfig` - 获取节点配置

---

## [Unreleased]

## [v0.2.0] - 2026-02-22

### Breaking Changes
- **API 路由统一**: `/api/master/clients/*` 和 `/api/master/nodes/*` 统一为 `/api/nodes/*`
  - 旧路由已标记为废弃，将在 v0.4.0 移除
  - 旧路由返回 `X-API-Deprecation` 和 `X-API-Sunset` 响应头
- **前端类型重构**: `Client` 类型迁移到 `UnifiedNode`
  - `web/src/types/cluster.ts` 标记为 `@deprecated`
  - 建议使用 `web/src/types/node.ts` 中的统一类型

### Added
- **统一类型系统**: `internal/types/node.go` 新增统一节点类型定义
  - `NodeCapabilities` - 节点能力（GPU/CPU/内存/软件支持）
  - `NodeResources` - 节点资源使用情况
  - `NodeInfo` - 统一节点信息（兼容 Master/Client/Node）
  - `HeartbeatMessage` - 统一心跳消息
  - `Command` 和 `CommandResult` - 统一命令结构
- **类型别名系统**: 保持向后兼容
  - `internal/node/types.go`: `NodeInfo` → `types.NodeInfo`
  - `internal/cluster/types.go`: `Client` → `types.NodeInfo`
- **前端统一类型**: `web/src/types/node.ts` 新增 `UnifiedNode` 接口
- **API 响应辅助函数**: `internal/handler/response.go` 提供统一响应格式
  - `Success()`, `Error()`, `ValidationError()`, `NotFound()` 等
- **任务类型定义**: `web/src/types/task.ts` 新增任务相关类型

### Changed
- **NodeAdapter 路由重构**: 主路由使用 `/api/nodes/*`
  - `RegisterRoutes()` 方法重构
  - 添加 `registerDeprecatedRoutes()` 处理旧路由
  - 添加 `deprecationWarningMiddleware()` 废弃警告中间件
- **前端类型导出**: `web/src/types/index.ts` 重新组织导出
- **客户端组件更新**: 修复可选字段空值处理

### Fixed
- 修复前端 `ClientCard` 组件可选字段空值访问问题
- 统一前后端类型定义，消除重复

### Technical Details
- **后端类型统一**: 新建 `internal/types/node.go` 作为唯一类型定义来源
- **前端类型统一**: `web/src/types/node.ts` 的 `UnifiedNode` 作为推荐类型
- **向后兼容策略**: 使用类型别名保持现有代码无需修改
- **API 废弃策略**: HTTP 响应头 + 日志警告 + 文档标记

### Migration Guide

**后端迁移**:
```go
// 旧代码
import "github.com/simonxluo/Shepherd/internal/node"
nodeInfo := node.NodeInfo{...}

// 新代码（推荐）
import "github.com/simonxluo/Shepherd/internal/types"
nodeInfo := types.NodeInfo{...}

// 旧代码仍可编译（类型别名）
nodeInfo := node.NodeInfo{...} // 等同于 types.NodeInfo
```

**前端迁移**:
```typescript
// 旧代码
import type { Client } from '@/types/cluster';

// 新代码（推荐）
import type { UnifiedNode } from '@/types/node';

// 旧代码仍可编译（类型别名）
import type { Client } from '@/types/cluster'; // 等同于 UnifiedNode
```

**API 路由迁移**:
```bash
# 旧路由（已废弃）
GET /api/master/clients
GET /api/master/nodes

# 新路由（推荐）
GET /api/nodes
```

## [v0.1.4] - 2026-02-22

### Added
- 模型性能测试页面自动加载默认 llama.cpp 路径的设备信息
- 设备列表解析时添加严格的前缀验证 (ROCm/CUDA/Vulkan/Metal)

### Changed
- 优化 BenchmarkDialog 的 useEffect 执行顺序，确保设备检测自动触发

### Fixed
- 修复设备列表解析错误匹配调试信息导致重复设备的问题
  - llama.cpp --list-devices 输出包含 stderr 调试信息
  - 修复前: 解析了 "ggml_cuda_init: found 1 ROCm devices" 导致重复
  - 修复后: 只解析 "Available devices:" 标记后的正式设备列表
- 修复性能测试对话框打开后设备不自动加载的问题
  - 调整 useEffect 顺序，确保设备检测在初始化之前执行

### Technical Details
- **设备检测改进**: `parseDeviceList()` 现在只在找到 "Available devices:" 标记后开始解析
- **设备验证**: 添加 `validDevicePrefix()` 方法验证设备前缀格式
- **前端优化**: BenchmarkDialog useEffect 顺序重构，确保自动加载

## [v0.1.3] - 2026-02-19

### Added
- 配置管理 API (llama.cpp 和模型路径)
- 下载管理器完整实现
- 进程管理 API
- 脚本重组 (linux/macos/windows)

### Changed
- Web 前端完全独立于后端配置
- 路径配置功能从设置页面移到独立配置

## [v0.1.2] - 2026-02-15

### Added
- Web 前端独立架构
- 前端独立配置文件 (web/config.yaml)
- 多后端支持和运行时切换
- SSE 实时事件推送

## [v0.1.1] - 2026-02-10

### Added
- Master-Client 分布式架构
- 统一 Node 模型
- 心跳和资源监控
- 命令分发和调度

## [v0.1.0-alpha] - 2026-02-01

### Added
- 核心功能实现
- GGUF 模型扫描和管理
- OpenAI/Anthropic/Ollama API 兼容
- Web UI 基础功能
