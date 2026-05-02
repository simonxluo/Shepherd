# Shepherd 后端架构

## 设计目标

Shepherd 是一个 Go 语言编写的分布式 llama.cpp 模型管理系统，核心目标：

- **统一管理**：单进程管理多个 llama.cpp 实例，提供模型加载/卸载/自动发现
- **多协议兼容**：同时提供 OpenAI、Anthropic、Ollama、LM Studio 兼容 API
- **多模态支持**：通过 vLLM-Omni 等多模态后端提供 TTS/ASR/图像生成能力（`/v1/audio/*`、`/v1/images/*`）
- **分布式调度**：支持 master/client/hybrid 三种节点角色，跨机器调度模型
- **纯 Go 实现**：SQLite 使用 modernc.org/sqlite（无 CGO 依赖），单二进制部署

在后端进程引擎层面，系统采用"命令字符串 + 宏替换"的后端无关设计。进程管理器不绑定任何特定的推理引擎实现，而是通过可插拔的后端注册表（`Backend` 接口）支持 llama.cpp、vLLM、vLLM-Omni 等多种推理引擎。每种后端实现统一的 `Discover / BuildStartConfig / IsLoadComplete / CheckHealth / SupportsModel` 接口，新增后端仅需添加实现类并注册，无需改动现有代码。

## 分层架构

```
┌─────────────────────────────────────────────────┐
│                 入口层 (cmd/)                     │
│           cmd/shepherd/main.go                   │
├─────────────────────────────────────────────────┤
│               API 层 (handler/ + router/)         │
│  ┌──────────┬──────────┬──────────┬────────────┐ │
│  │ OpenAI   │ Ollama   │Anthropic │ LM Studio  │ │
│  │ /v1/*    │ /api/*   │ /v1/*    │ /lmstudio/*│ │
│  │ 多模态*  │          │          │            │ │
│  └──────────┴──────────┴──────────┴────────────┘ │
│  middleware: RequestID → Recovery → CORS → Logger │
├─────────────────────────────────────────────────┤
│              服务层 (service/)                     │
│  ┌──────────┬──────────┬──────────┐              │
│  │  model   │   node   │ langchain│              │
│  │ 模型管理  │ 节点管理  │ LLM集成  │              │
│  │ backend/ │          │          │              │
│  └──────────┴──────────┴──────────┘              │
│  ┌──────────┬──────────┐                         │
│  │ cluster  │ cluster/ │                         │
│  │ 集群类型  │scheduler │                         │
│  └──────────┴──────────┘                         │
├─────────────────────────────────────────────────┤
│             基础设施层 (infra/)                    │
│  ┌────────┬────────┬───────┬────────┬─────────┐  │
│  │process │storage │ port  │download │  gguf   │  │
│  │进程管理 │持久存储 │端口分配│下载管理  │GGUF解析 │  │
│  └────────┴────────┴───────┴────────┴─────────┘  │
│  ┌────────────┐                                   │
│  │ modelrepo  │ HuggingFace/ModelScope 仓库集成   │
│  └────────────┘                                   │
├─────────────────────────────────────────────────┤
│             通信层 (comm/)                        │
│  config │ logger │ event │ types │ gpu │shutdown │
│  配置管理│ 日志   │ SSE   │ 类型  │GPU  │ 关闭   │
│  netutil│ utils  │       │       │     │         │
│  网络工具│通用工具 │       │       │     │         │
└─────────────────────────────────────────────────┘
```

\* 多模态端点：`/v1/audio/speech`、`/v1/audio/transcriptions`、`/v1/audio/translations`、`/v1/images/generations`

## 层间依赖规则

```
server → handler → service → comm
                   infra  → comm
                   infra  → service (单向: langchain → model)
```

**禁止**：handler 直接依赖 infra；service 之间互相依赖（除 langchain → model）。

## 包结构

| 包 | 路径 | 职责 |
|---|---|---|
| **入口** | `cmd/shepherd/` | main() 入口，App 初始化/启动/关闭 |
| **路由** | `internal/router/` | 集中式路由注册 `Setup()` |
| **中间件** | `internal/middleware/` | RequestID、Recovery、CORS、Logger、ErrorHandler |
| **服务器** | `internal/server/` | HTTP 服务器生命周期、WebSocket Hub |
| **处理器** | `internal/handler/` | HTTP 处理器 (Gin) + NodeAdapter |
| ↳ OpenAI | `handler/openai/` | `/v1/chat/completions`, `/v1/completions`, `/v1/models` |
| ↳ Audio | `handler/openai/` | `/v1/audio/speech`, `/v1/audio/transcriptions`, `/v1/audio/translations` |
| ↳ Image | `handler/openai/` | `/v1/images/generations` |
| ↳ Anthropic | `handler/anthropic/` | `/v1/messages` |
| ↳ Ollama | `handler/ollama/` | `/api/chat`, `/api/tags` |
| ↳ LM Studio | `handler/lmstudio/` | `/lmstudio/v1/*` |
| ↳ 兼容性 | `handler/compatibility/` | 兼容服务管理 |
| ↳ 文件系统 | `handler/filesystem/` | 目录浏览、路径校验 |
| ↳ 路径配置 | `handler/paths/` | llama.cpp/模型路径 CRUD |
| ↳ 存储 | `handler/storage/` | 存储配置、会话管理 |
| ↳ 基准测试 | `handler/benchmark/` | Benchmark 创建/查询/配置 |
| **模型服务** | `service/model/` | GGUF 模型管理 + 能力自动检测 |
| ↳ 后端注册表 | `service/model/backend/` | 可插拔后端（llama.cpp, vLLM, vLLM-Omni） |
| **节点服务** | `service/node/` | 统一节点（hybrid/master/client） |
| **集群服务** | `service/cluster/` | 集群类型、扫描器、调度器 |
| **LangChain** | `service/langchain/` | LangChainGo 集成 |
| **进程** | `infra/process/` | llama.cpp 进程生命周期 |
| **存储** | `infra/storage/` | SQLite/Memory 存储实现 |
| **端口** | `infra/port/` | 端口分配（默认 8081-9000） |
| **下载** | `infra/download/` | HuggingFace/URL 下载管理 |
| **GGUF** | `infra/gguf/` | GGUF 文件解析器 |
| **模型仓库** | `infra/modelrepo/` | HuggingFace/ModelScope 仓库集成 |
| **配置** | `comm/config/` | 配置加载/迁移/原子写入 |
| **事件** | `comm/event/` | SSE 事件管理器（主实时通道） |
| **类型** | `comm/types/` | 共享类型：NodeInfo、ErrorCode、ApiResponse[T] |
| **GPU** | `comm/gpu/` | GPU 检测（nvidia-smi / rocm-smi / lspci） |
| **日志** | `comm/logger/` | 结构化日志 + 文件轮转 + 环形缓冲区 + 实时流 |
| **关闭** | `comm/shutdown/` | 优先级优雅关闭 |
| **网络工具** | `comm/netutil/` | 本地 IP 探测（`GetBestLocalIP`） |
| **通用工具** | `comm/utils/` | 静默关闭/删除/重命名、进程信号、llama.cpp 二进制查找 |

## 初始化流程

从 `cli/run_server.go` 的 `Initialize()` 方法实际顺序：

```
Config → Logger → Process → Port → Storage → Model → LangChain → Node → Server → Shutdown
```

详细步骤：

1. **配置管理器**：`config.NewManager()` → `Load()`，默认路径 `config/server.config.yaml`（可通过 `SHEPHERD_CONFIG_DIR` 环境变量或 `--config` 参数覆盖）；文件不存在时使用 `DefaultConfig()`
2. **确定角色**：读取 `cfg.Node.Role`，默认 `hybrid`
3. **日志系统**：`logger.InitLogger()`（内含 LogMonitor 环形缓冲区）
4. **进程管理器**：`process.NewManager()`
5. **端口分配器**：`port.NewPortAllocator(base, max)`，范围从配置读取（默认 8081-9000）
6. **存储管理器**：如果配置中 `storage.Type` 为空，run_server 会覆盖为 SQLite（`./data/shepherd.db`，WAL 模式）；`DefaultConfig()` 默认类型为 `memory`
7. **模型管理器**：`model.NewManager()`，触发已保存模型加载 + TTL 检查器启动
8. **LangChain**：`langchain.NewManager()` + `NewHandler()`
9. **分布式组件**：根据角色初始化 Node（含子系统：注册、心跳、命令、资源监控）
10. **HTTP 服务器**：`server.NewServer()` → 注册处理器 → `SetupRoutes()`
11. **关闭管理器**：`shutdown.NewManager(10s)`，监听 SIGINT/SIGTERM/SIGQUIT

## 关闭流程

优先级从高到低执行关闭钩子：

| 优先级 | 名称 | 操作 |
|---|---|---|
| Critical (0) | http-server | 优雅停止 HTTP 服务器 |
| High (1) | node | 停止所有节点子系统 |
| High (1) | models | 关闭模型管理器（停止 TTL 检查器） |
| High (1) | storage | 关闭存储管理器 |
| Normal (2) | processes | 停止所有 llama.cpp 进程 |
| Low (3) | logger | 关闭日志系统 |

## 并发模型

- **RWMutex**：模型状态读写（`sync/atomic` 用于 LoadState CAS 转换）
- **Context + WaitGroup**：goroutine 生命周期管理
- **Semaphore**：扫描并发控制（≤10）、命令执行并发（≤4）
- **CAS 操作**：进程状态机转换（`compare-and-swap` 确保线程安全）

## 关键设计决策

1. **纯 Go SQLite**：使用 `modernc.org/sqlite`，无需 CGO，单二进制部署
2. **Manager 模式**：每个核心领域（模型、进程、存储、下载）封装为 Manager
3. **原子配置写入**：写入 `.tmp` 文件后 `rename()`，避免配置损坏
4. **SSE 优先**：`GET /api/events` 为主实时通道，WebSocket 为辅助
5. **Adapter 模式**：`NodeAdapter` 将 Node/Scheduler 桥接到 HTTP API
6. **Provider 模式**：GPU 检测通过 Provider 接口支持 NVIDIA/AMD/Intel
7. **可插拔后端注册表**：`Backend` 接口管理多种推理引擎（llama.cpp、vLLM、vLLM-Omni），新增后端只需实现接口并注册

## SDK 依赖

| 库 | 用途 |
|---|---|
| `gin-gonic/gin` | HTTP 框架 |
| `gorilla/websocket` | WebSocket |
| `gopkg.in/yaml.v3` | YAML 解析 |
| `modernc.org/sqlite` | 纯 Go SQLite |
| `shirou/gopsutil` | 系统资源监控 |
| `google/uuid` | 请求 ID |
| `gpustack/gguf-parser-go` | GGUF 元数据解析 |
| `ROCm/amdsmi` | AMD GPU 监控 (build tag) |
| `tmc/langchaingo` | LangChain Go |
| `swaggo/swag` | Swagger 文档生成 |
