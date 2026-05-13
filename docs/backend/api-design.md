# API 设计

## 统一响应格式

### 成功响应 `ApiResponse[T]`

```json
{
  "success": true,
  "data": T,
  "metadata": {
    "timestamp": "2026-01-01T00:00:00Z",
    "requestId": "uuid",
    "latency": 12
  }
}
```

### 分页响应 `PaginatedResponse[T]`

分页字段位于顶层，不包含 `totalPages`：

```json
{
  "success": true,
  "data": [T],
  "total": 100,
  "page": 1,
  "pageSize": 20,
  "metadata": {
    "timestamp": "...",
    "requestId": "..."
  }
}
```

### 错误响应 `ErrorInfo`

```json
{
  "success": false,
  "error": {
    "code": "NODE_NOT_FOUND",
    "message": "节点不存在",
    "details": "node id: xxx"
  }
}
```

### 错误码映射

| ErrorCode | HTTP Status | 说明 |
|---|---|---|
| NODE_NOT_FOUND | 404 | 节点不存在 |
| INVALID_REQUEST | 400 | 请求参数无效 |
| CONFLICT | 409 | 资源冲突 |
| TIMEOUT | 408 | 操作超时 |
| COMMAND_FAILED | 500 | 命令执行失败 |
| NOT_AUTHENTICATED | 401 | 未认证 |
| PERMISSION_DENIED | 403 | 权限不足 |
| RESOURCE_EXHAUSTED | 429 | 资源耗尽 |
| INTERNAL_ERROR | 500 | 内部错误 |

## 路由表

### 系统与配置

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/info` | 服务器信息 |
| GET | `/api/system/gpus` | GPU 列表 |
| GET | `/api/system/llamacpp-backends` | llama.cpp 后端列表 |
| GET | `/api/config` | 获取配置 |
| PUT | `/api/config` | 更新配置 |
| GET/POST/PUT/DELETE | `/api/config/llamacpp/paths` | llama.cpp 路径 CRUD |
| POST | `/api/config/llamacpp/paths/test` | 测试路径 |
| GET/POST/PUT/DELETE | `/api/config/models/paths` | 模型路径 CRUD |
| GET/PUT | `/api/config/storage` | 存储配置 |
| GET | `/api/config/storage/stats` | 存储统计 |
| GET/PUT | `/api/config/compatibility` | 兼容性配置 |
| POST | `/api/config/compatibility/test` | 测试连接 |

### 模型管理

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/models` | 模型列表 |
| GET | `/api/models/loaded` | 已加载模型 |
| GET | `/api/models/:id` | 模型详情 |
| POST | `/api/models/:id/load` | 加载模型 |
| POST | `/api/models/:id/unload` | 卸载模型 |
| PUT | `/api/models/:id/alias` | 设置别名 |
| PUT | `/api/models/:id/favourite` | 设置收藏 |
| GET/PUT/DELETE | `/api/models/:id/load-config` | 加载配置 CRUD |
| GET | `/api/models/capabilities/get` | 获取能力 |
| POST | `/api/models/capabilities/set` | 设置能力 |
| GET | `/api/models/capabilities/auto-detect` | 自动检测 |
| POST | `/api/models/vram/estimate` | VRAM 估算 |
| POST | `/api/model/scan` | 扫描模型 |
| GET | `/api/model/scan/status` | 扫描状态 |

### Benchmark

| Method | Path | 说明 |
|---|---|---|
| POST | `/api/models/benchmark` | 创建 Benchmark |
| GET | `/api/models/benchmark/tasks` | 任务列表 |
| GET | `/api/models/benchmark/tasks/:id` | 任务详情 |
| POST | `/api/models/benchmark/tasks/:id/cancel` | 取消任务 |
| GET/POST | `/api/models/benchmark/configs` | 配置列表/创建 |
| GET/DELETE | `/api/models/benchmark/configs/:name` | 配置详情/删除 |

### 下载管理

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/downloads` | 下载列表 |
| POST | `/api/downloads` | 创建下载 |
| GET | `/api/downloads/:id` | 下载详情 |
| POST | `/api/downloads/:id/pause` | 暂停 |
| POST | `/api/downloads/:id/resume` | 恢复 |
| DELETE | `/api/downloads/:id` | 删除 |

### 模型仓库

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/repo/files` | 仓库文件列表 |
| GET | `/api/repo/search` | 搜索模型 |
| GET/PUT | `/api/repo/config` | 仓库配置 |
| GET | `/api/repo/endpoints` | 可用端点 |

### 进程管理

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/processes` | 进程列表 |
| GET | `/api/processes/:id` | 进程详情 |
| POST | `/api/processes/:id/stop` | 停止进程 |

### 日志

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/logs/stream` | 日志 SSE 流 |
| GET | `/api/logs/entries` | 日志条目 |
| GET | `/api/logs/files` | 日志文件列表 |
| GET/DELETE | `/api/logs/files/:filename` | 文件内容/删除 |
| GET | `/api/logs/files/:filename/stats` | 文件统计 |

### 文件系统

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/system/filesystem` | 目录浏览 |
| POST | `/api/system/filesystem/validate` | 路径校验 |

### 会话

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/conversations` | 会话列表 |
| GET | `/api/conversations/:id` | 会话详情 |
| DELETE | `/api/conversations/:id` | 删除会话 |

### 实时通信

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/events` | SSE 事件流（主通道） |
| GET | `/ws` | WebSocket 连接（辅助） |

### 节点管理（条件注册）

| Method | Path | 说明 |
|---|---|---|
| POST | `/api/nodes/register` | 节点注册 |
| GET | `/api/nodes` | 节点列表 |
| GET | `/api/nodes/:id` | 节点详情 |
| DELETE | `/api/nodes/:id` | 注销节点 |
| POST | `/api/nodes/:id/command` | 发送命令 |
| GET | `/api/nodes/:id/commands` | 待执行命令 |
| POST | `/api/nodes/:id/heartbeat` | 心跳 |
| GET | `/api/nodes/:id/config` | 节点配置 |
| POST | `/api/nodes/:id/test` | 测试节点 |
| POST | `/api/heartbeat` | 心跳上报 |
| POST | `/api/command/result` | 命令结果上报 |
| POST/GET/DELETE | `/api/tasks[/:id]` | 任务管理 |
| POST | `/api/tasks/:id/retry` | 重试任务 |
| POST | `/api/scan` | 网络扫描 |
| GET | `/api/scan/status` | 扫描状态 |
| GET | `/api/overview` | 集群概览 |

### 内部聊天 API（前端使用）

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/langchain/chat/models` | 聊天模型列表（含加载状态） |
| POST | `/api/langchain/chat/completions` | 流式聊天补全（惰性加载） |

> 注：路径保留 `/api/langchain/` 前缀以兼容前端。后端实现已迁移至轻量 `handler/chat`，不依赖 LangChainGo SDK。

## 兼容性 API

所有兼容性 API 默认共用 WebPort（9190），但可通过配置为 Ollama 和 LM Studio 分配独立端口：

| 协议 | 默认端口 | 配置 | 路由 |
|---|---|---|---|
| OpenAI | 9190（WebPort） | — | `POST /v1/chat/completions`, `POST /v1/completions`, `GET /v1/models` |
| OpenAI 多模态 | 9190（WebPort） | — | `POST /v1/audio/speech`, `POST /v1/audio/transcriptions`, `POST /v1/audio/translations`, `POST /v1/images/generations` |
| Anthropic | 9190（WebPort） | `server.anthropic_port` | `POST /v1/messages` |
| Ollama | 11434 | `compatibility.ollama.port`（需 `enabled: true`） | `POST /api/chat`, `POST /api/tags` |
| LM Studio | 1234 | `compatibility.lmstudio.port`（需 `enabled: true`） | `POST /lmstudio/v1/*` |

### 多模态兼容 API 代理模式

TTS、ASR 和图像生成请求通过专用的 Handler（`AudioHandler`、`ImageHandler`）代理到后端模型进程。代理模式与 Chat/Completion 请求一致——从请求体的 `model` 字段解析模型标识，查找对应进程的监听端口，然后转发请求。

| 端点 | 请求格式 | 响应格式 | Handler 方法 |
|---|---|---|---|
| `/v1/audio/speech` | JSON（model, input, voice） | 二进制音频流 | `ForwardBinaryRequest` |
| `/v1/audio/transcriptions` | multipart/form-data（file + model） | JSON | `ForwardMultipartRequest` |
| `/v1/audio/translations` | multipart/form-data（file + model） | JSON | `ForwardMultipartRequest` |
| `/v1/images/generations` | JSON（model, prompt, size） | JSON（base64/URL） | `ForwardRequest` |

## SSE 事件类型

| 事件 | EventType 常量 | 触发时机 |
|---|---|---|
| heartbeat | `heartbeat` | 定时心跳 |
| systemStatus | `systemStatus` | 系统状态变更 |
| modelLoadStart | `modelLoadStart` | 模型开始加载 |
| modelLoad | `modelLoad` | 模型加载完成 |
| modelStop | `modelStop` | 模型停止 |
| model_slots | `model_slots` | 模型 slot 状态更新 |
| console | `console` | 控制台输出（Base64 编码） |
| download_update | `download_update` | 下载状态变更 |
| download_progress | `download_progress` | 下载进度 |
| scan_progress | `scan_progress` | 模型扫描进度 |
| scan_complete | `scan_complete` | 模型扫描完成 |

## 中间件链

请求按以下顺序经过中间件：

```
RequestID → Recovery → CORS → Logger → ErrorHandler → Handler
```

1. **RequestID**：生成 UUID，设置 `X-Request-ID` 头
2. **Recovery**：panic 恢复，返回 `INTERNAL_ERROR`
3. **CORS**：跨域处理，支持预检请求
4. **Logger**：记录请求方法、路径、状态码、延迟
5. **ErrorHandler**：捕获 `ErrorInfo` 类型，格式化 HTTP 响应

## 设计决策

- **泛型响应信封**：`ApiResponse[T]` 统一所有成功/错误响应格式
- **错误码分离**：ErrorCode 与 HTTP Status 独立，支持更细粒度错误区分
- **多协议端口**：不同协议可使用不同端口，通过配置开启，避免路由冲突
- **双实时通道**：SSE（主）+ WebSocket（辅助），SSE 用于服务端推送，WebSocket 用于双向通信
