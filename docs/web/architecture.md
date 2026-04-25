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
├── components/
│   ├── layout/            # 布局组件（Header, Sidebar, MainLayout）
│   └── ui/                # UI 基础组件（Button, Card, Dialog, Toast...）
├── features/              # 业务功能模块
│   ├── chat/              # 聊天（流式响应）
│   ├── cluster/           # 集群管理
│   ├── downloads/         # 下载管理
│   ├── logs/              # 日志查看
│   ├── models/            # 模型管理（最大模块，拆分为子文件）
│   └── settings/          # 系统设置
├── hooks/                 # 共享 Hooks
│   ├── useSSEConnection   # SSE 基础连接（指数退避重连）
│   ├── useSSE             # SSE → React Query 失效
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
└── types/                 # TypeScript 类型定义
```

## 状态管理

### 服务端状态（React Query）

所有从后端获取的数据通过 React Query 管理：

- `['models']` - 模型列表
- `['downloads']` - 下载任务
- `['clients']` - 集群节点
- `['tasks']` - 调度任务
- `['cluster']` - 集群概览
- `['system']` - 系统信息

### 客户端状态（Zustand）

| Store | 持久化 | 内容 |
|---|---|---|
| `useUIStore` | localStorage | 主题、侧边栏开关、视图模式 |
| `useUserStore` | localStorage | 用户信息、语言偏好 |
| `useToastStore` | 内存 | Toast 通知队列 |

## API 客户端

`ApiClient` 单例封装 `fetch()`：

```typescript
class ApiClient {
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
useSSEConnection (基础) → useSSE (Query 失效) → useLogStream (日志)
```

- `useSSEConnection`：管理 EventSource 连接，指数退避重连（最大 10 次）
- `useSSE`：监听 SSE 事件，自动失效对应 React Query 缓存
- 重连时全量刷新所有 Query

### WebSocket（辅助）

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
web.config.yaml (源) → sync-web-config.sh → public/config.yaml → Vite + configLoader.ts
```

配置文件包含后端 URL 列表，支持运行时切换。

## 功能模块模式

每个功能模块遵循统一结构：

```
features/<name>/
├── components/    # UI 组件
├── hooks.ts       # React Query hooks
└── index.ts       # 统一导出
```

模型模块因规模较大，拆分为子文件：

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

## 构建优化

- **代码分割**：react-vendor、query-vendor、ui-vendor 手动分块
- **Source Map**：生产构建启用
- **路径别名**：`@` → `./src`
