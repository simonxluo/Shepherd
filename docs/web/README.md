# Shepherd Web 前端

现代化的 React + TypeScript 前端应用，用于 Shepherd llama.cpp 模型管理系统。

## 技术栈

- **框架**: React 19.2.0 + TypeScript 5 + Vite 7
- **路由**: React Router v7
- **状态**: Zustand + React Query
- **样式**: Tailwind CSS v4 + shadcn/ui

## 快速开始

```bash
# 使用项目脚本（推荐）
./scripts/linux/web.sh dev

# 或直接运行
cd web
npm install
npm run dev
```

## 配置

前端配置文件位于 `web/config.yaml`（项目根目录）。

### 修改后端地址

```yaml
backend:
  urls:
    - "http://localhost:9190"
    - "http://192.168.1.100:9190"
  currentIndex: 0
```

### 同步配置

修改 `config.yaml` 后运行：

```bash
./scripts/sync-web-config.sh
```

## 开发命令

```bash
npm run dev        # 开发服务器
npm run build      # 生产构建
npm run preview    # 预览构建
npm run type-check # 类型检查
npm run lint       # 代码检查
```

## 项目结构

```
src/
├── components/    # 可复用组件
│   ├── ui/        # 基础 UI 组件
│   └── layout/    # 布局组件
├── pages/         # 路由页面
├── features/      # 功能模块
├── stores/        # Zustand 状态
├── hooks/         # 自定义 Hooks
├── lib/           # 工具库
└── types/         # TypeScript 类型
```

## 部署

### 开发模式

```bash
# 终端 1: 启动后端
./build/shepherd standalone

# 终端 2: 启动前端
npm run dev
```

### 生产模式

```bash
# 构建前端
npm run build

# 部署到 Nginx
cp -r dist/* /var/www/html/
```

### 后端托管

```bash
# 构建并复制到后端
npm run build
cp -r dist/* ../internal/server/web/

# 启动后端（同时托管前端）
./build/shepherd standalone
```

## 开发规范

### 文件命名

- 组件: `PascalCase.tsx` (例: `ModelCard.tsx`)
- 工具: `camelCase.ts` (例: `apiClient.ts`)
- Hooks: `camelCase` with `use` 前缀 (例: `useSSE.ts`)

### 导入顺序

```typescript
// 1. React 核心
import { useState } from 'react';

// 2. 第三方库
import { useQuery } from '@tanstack/react-query';

// 3. 内部导入
import { Button } from '@/components/ui/button';
```

### TypeScript 规范

- 优先使用 `type` 导入类型
- 避免使用 `any`
- 接口用于对象，类型别名用于联合类型

## 架构说明

### 前端独立架构

Web 前端采用完全独立架构，可连接任意后端：

```
┌─────────────────────────────────────────────┐
│            Web 前端 (独立运行)              │
│                                             │
│  web/config.yaml (前端配置)                 │
│  ↓                                          │
│  API Client (动态连接后端)                  │
│  ↓                                          │
│  HTTP 请求 (CORS)                           │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          后端服务器 (9190)                   │
│  /api/models, /api/events, /v1/*           │
└─────────────────────────────────────────────┘
```

### API 集成

开发环境下 Vite 自动代理 API 请求：

```javascript
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:9190',
    '/v1': 'http://localhost:9190'
  }
}
```

### 数据获取

```typescript
// 使用 React Query Hook
const { data, isLoading, error } = useModels();

// 使用 Mutation
const { mutate: loadModel } = useLoadModel();
```

## 故障排除

### 端口被占用

```bash
npm run dev -- --port 4000
```

### 类型错误

```bash
rm -rf node_modules/.vite dist
npm run build
```

### 依赖问题

```bash
rm -rf node_modules package-lock.json
npm install
```

## 更多文档

- [部署指南](deployment.md)
