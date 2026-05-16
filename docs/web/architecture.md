# 前端架构

## 技术栈

| 技术 | 版本 | 用途 |
|---|---|---|
| React | 19.2 | UI 框架 |
| TypeScript | 5.9 | 类型系统 |
| Vite | 7.3 | 构建工具 |
| React Router | v7 | 路由 |
| React Query | v5 | 服务端状态管理 |
| Zustand | v5 | 客户端状态管理 |
| Tailwind CSS | v4 | 样式 |
| i18next | 25 | 国际化 |
| Vitest | 3 | 测试 |

## 目录结构

```
web/src/
├── App.tsx                # 根组件 + Provider 树
├── main.tsx               # 入口（i18n 初始化、配置加载、渲染）
├── index.css              # Tailwind v4 + 主题变量
├── assets/                # 静态资源
├── components/
│   ├── layout/            # 布局组件（Header, Sidebar, MainLayout, UserMenu, UserProfileDialog, UserSettingsDialog）
│   └── ui/                # UI 基础组件（alert-dialog, badge, button, card, dialog, LanguageToggle, switch, tabs, ThemeToggle, toast, toaster）
├── features/              # 业务功能模块
│   ├── chat/              # 聊天（流式响应）
│   ├── cluster/           # 集群管理
│   ├── downloads/         # 下载管理
│   ├── logs/              # 日志查看
│   ├── models/            # 模型管理（最大模块，拆分为子文件）
│   └── settings/          # 系统设置（仅 components/，无 hooks.ts 或 index.ts）
├── hooks/                 # 共享 Hooks
│   ├── useSSEConnection   # SSE 基础连接（指数退避重连）
│   ├── useSSE             # SSE handling logic lives in App.tsx's handleSSEMessage callback
│   └── useToast           # Toast 通知
├── lib/
│   ├── api/               # API 客户端（单例 ApiClient）
│   ├── config/            # 配置加载器
│   ├── i18n/              # i18n 初始化
│   ├── query/             # React Query 客户端
│   ├── utils.ts           # 工具函数（cn, formatBytes）
│   └── websocket.ts       # WebSocket 客户端
├── locales/               # 翻译文件（zh-CN, en-US）
├── pages/                 # 路由页面
├── providers/             # Context Providers
│   ├── AlertDialog        # 确认对话框
│   └── WebSocketProvider  # WebSocket 管理
├── stores/                # Zustand 全局状态
│   ├── uiStore            # UI 状态（主题、侧边栏、视图模式）
│   ├── userStore          # 用户设置
│   └── toast              # Toast 通知状态
├── test/                  # 测试配置（setup.ts）
└── types/                 # TypeScript 类型定义
```

## 状态管理

### 服务端状态（React Query）

所有从后端获取的数据通过 React Query 管理：

**模型相关：**
- `['models']` - 模型列表
- `['models', modelId]` - 单个模型详情
- `['models', 'capabilities', modelId]` - 模型能力配置
- `['models', modelId, 'load-config']` - 模型加载配置
- `['benchmark', 'params']` - 压测参数列表
- `['llamacpp', 'versions']` - llama.cpp 版本列表
- `['benchmarks']` - 压测任务列表
- `['benchmark', 'results']` - 压测结果

**下载相关：**
- `['downloads']` - 下载任务
- `['model-files', source, repoId]` - 模型文件列表
- `['huggingface-search', query, limit, format]` - HuggingFace 搜索
- `['model-repo-config']` - 模型仓库配置
- `['model-repo-endpoints']` - 可用端点列表

**集群相关：**
- `['cluster', 'overview']` - 集群概览
- `['cluster', 'clients']` - 客户端列表
- `['cluster', 'clients', clientId]` - 单个客户端详情
- `['cluster', 'tasks']` - 调度任务
- `['cluster', 'nodes', 'online']` - 在线节点
- `['cluster', 'nodes', nodeId, 'config']` - 节点配置
- `['cluster', 'scan']` - 网络扫描

**系统相关：**
- `['server', 'config']` - 服务器配置
- `['system', 'gpus', llamaCppPath]` - GPU 信息
- `['system', 'llamacpp-backends']` - llama.cpp 后端列表
- `['system']` - 系统状态

### 客户端状态（Zustand）

| Store | 持久化 | 内容 |
|---|---|---|
| `useUIStore` | localStorage | 主题、侧边栏开关、视图模式 |
| `useUserStore` | localStorage | 用户信息、语言偏好 |

## API 客户端

`ApiClient` 单例封装 `fetch()`：

```typescript
class ApiClient {
  getBaseUrl(): string
  setBaseUrl(baseUrl: string): void
  get<T>(path, params?, signal?): Promise<T>
  post<T>(path, body?): Promise<T>
  put<T>(path, body): Promise<T>
  delete<T>(path): Promise<T>
}
```

- 统一错误处理：`APIError` 类包含 status、code、details
- 运行时 URL 切换：`updateApiClientUrl()` 支持动态更换后端
- 开发代理：Vite 将 `/api` 代理到 `http://localhost:9190`

## 实时通信

### SSE（主通道）

```
useSSEConnection (基础) → useSSE (Query 失效)
```

- `useSSEConnection`：管理 EventSource 连接，指数退避重连（最大 10 次）
- `useSSE`：SSE handling logic lives in App.tsx's handleSSEMessage callback
- 重连时全量刷新所有 Query（`models`、`downloads`、`clients`、`cluster`、`tasks`、`system`、`nodes`）

`useLogStream` 是 feature-specific hook（位于 `features/logs/hooks.ts`），使用 `fetch + ReadableStream` 消费 chunked 文本流（`/logs/stream/text`），不经过 SSE。支持传入节点 URL 查看远程节点日志。

### WebSocket（已废弃）

> **注意：WebSocket 系统已被移除/废弃。实时通信现在完全通过 SSE 实现。**

- `WebSocketProvider`：管理 WebSocket 生命周期
- 事件订阅系统：`subscribe(eventType, handler)` → 返回取消函数
- URL 自动推导：`http(s)://` → `ws(s)://`，`/api` → `/ws`
- 默认不自动连接（`autoConnect=false`）

## Provider 树

```
QueryClientProvider
  └── WebSocketProvider
        └── AlertDialogProvider
              ├── AppContent（BrowserRouter + useSSE）
              ├── AlertDialog
              └── Toaster
```

初始化顺序：i18n → configLoader → updateApiClientUrl → Provider 树渲染。

## 路由

| 路径 | 页面 |
|---|---|
| `/` | Dashboard |
| `/models` | 模型管理 |
| `/downloads` | 下载管理 |
| `/chat` | 聊天 |
| `/cluster` | 集群管理 |
| `/logs` | 日志 |
| `/multimodal` | 多模态 |
| `/settings` | 系统设置 |

## 主题系统

- **Tailwind CSS v4**，无配置文件，通过 `@theme` 定义 CSS 变量
- 亮色：白色/灰色
- 暗色：VSCode Dark+ 风格（`#1e1e1e` 背景）
- 三种模式：Light / Dark / System（跟随系统）

## 国际化

- 默认语言：`zh-CN`
- 支持语言：`zh-CN`、`en-US`
- 检测优先级：localStorage → navigator → fallback zh-CN
- 类型安全：通过 `i18n.d.ts` 增强类型

## 配置管线

```
config/example/web.config.yaml (源) → sync-web-config.sh → web/public/config.yaml → Vite + configLoader.ts
```

配置文件包含后端 URL 列表，支持运行时切换。

## 功能模块模式

大部分功能模块遵循统一结构：

```
features/<name>/
├── components/    # UI 组件
├── hooks.ts       # React Query hooks
└── index.ts       # 统一导出
```

**例外：**

- **settings/** — 仅包含 `components/`（ApiConfigCard, DirectoryBrowser, PathConfigPanel, PathEditDialog, PathItem），无 `hooks.ts` 或 `index.ts`
- **models/** — 因规模较大，拆分为子文件：

```
features/models/
├── components/       # UI 组件
├── load.ts           # 加载/卸载 hooks
├── scan.ts           # 扫描 hooks
├── benchmark.ts      # Benchmark hooks
├── capabilities.ts   # 能力检测 hooks
├── config.ts         # 加载配置 hooks
└── index.ts          # 统一导出
```

## 模型配置管理

前端使用双层配置系统管理模型加载参数：

### 服务端持久化

- 每模型单配置，UPSERT 语义（按 `(node_id, model_id)` 一对一存储）
- 通过 `features/models/config.ts` 中的 React Query hooks 访问：
  - `useModelLoadConfig(modelId)` — 查询已保存配置
  - `useSaveModelLoadConfig()` — 保存配置 mutation
  - `useDeleteModelLoadConfig()` — 删除配置 mutation
- Query key：`['models', modelId, 'load-config']`，10 分钟缓存

### 客户端命名预设

- 存储于 `localStorage`，key 格式：`shepherd:model-configs:${modelId}`
- 支持每个模型保存多个命名预设
- LoadModelDialog 中提供预设选择下拉框、保存/删除按钮
- 当前仅 llama.cpp 对话框实现了完整的命名预设功能

### 各页面状态

| 页面 | 服务端配置 | 客户端命名预设 |
|---|---|---|
| llama.cpp LoadModelDialog | 自动保存/恢复 | 完整支持 |
| vLLM/vLLM-Omni LoadModelDialog | 自动保存/恢复 | 完整支持 |
| Multimodal TTSPanel | 自动保存/恢复 | 完整支持 |

## 构建优化

- **代码分割**：react-vendor、query-vendor、ui-vendor 手动分块
- **Source Map**：生产构建启用
- **路径别名**：`@` → `./src`
