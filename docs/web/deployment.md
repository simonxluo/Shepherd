# Web 前端独立部署指南

## 架构说明

Shepherd Web 前端采用**完全独立**的架构设计：

```
┌─────────────────────────────────────────────────────────┐
│                  Web 前端 (独立运行)                    │
│                                                          │
│  ┌────────────────────────────────────────────────┐    │
│  │  web/config.yaml (前端配置文件)               │    │
│  │  - backend.urls: 可配置多个后端地址           │    │
│  │  - features: 功能开关                        │    │
│  │  - ui: UI 配置                               │    │
│  └────────────────────────────────────────────────┘    │
│                         ↓                               │
│  ┌────────────────────────────────────────────────┐    │
│  │  API Client                                 │    │
│  │  - 动态连接到配置中的后端 URL                │    │
│  └────────────────────────────────────────────────┘    │
│                         ↓                               │
│  ┌────────────────────────────────────────────────┐    │
│  │         HTTP 请求 (CORS)                       │    │
│  └────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│              后端服务器 (9190 或其他)                   │
│                                                          │
│  提供 API:                                               │
│  - /api/models     - 模型管理                            │
│  - /api/downloads  - 下载管理                            │
│  - /api/events    - SSE 实时事件                         │
│  - /v1/*          - OpenAI 兼容 API                      │
└─────────────────────────────────────────────────────────┘
```

## 配置文件说明

### web/config.yaml

这是**前端的主配置文件**，完全由前端控制：

```yaml
# 目标后端服务器配置
backend:
  urls:
    - "http://localhost:9190"        # 本地开发
    - "http://192.168.1.100:9190"   # 局域网服务器
    - "https://api.example.com"     # 生产环境
  currentIndex: 0                   # 当前使用的后端索引

# 功能开关
features:
  models: true                      # 是否显示模型管理
  downloads: true                   # 是否显示下载管理
  cluster: true                     # 是否显示集群功能
```

## 部署方式

### 1. 开发模式（推荐）

前端和后端独立运行：

```bash
# 终端 1: 启动后端
./build/shepherd standalone

# 终端 2: 启动前端
cd web
npm run dev

# 访问: http://localhost:3000
```

前端会自动从 `config.yaml` 读取后端地址并连接。

### 2. 生产模式

前端构建后可独立部署：

```bash
# 构建前端
cd web
npm run build

# 部署到 Nginx/Apache 等 Web 服务器
cp -r dist/* /var/www/html/
```

然后配置 Nginx：

```nginx
server {
    listen 80;
    server_name your-domain.com;
    root /var/www/html;

    # 前端静态文件
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 转发到后端（可选，也可以让前端直连）
    location /api/ {
        proxy_pass http://backend:9190;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # SSE 支持
    location /api/events {
        proxy_pass http://backend:9190;
        proxy_set_header Connection '';
        proxy_buffering off;
        chunked_transfer_encoding on;
    }
}
```

### 3. 后端托管（简单部署）

后端服务器同时托管前端静态文件：

```bash
# 1. 构建前端
cd web
npm run build

# 2. 将 dist 目录内容复制到后端的 web 目录
cp -r dist/* ../internal/server/web/

# 3. 启动后端
./build/shepherd standalone

# 访问: http://localhost:9190
# 前端从同一服务器加载，后端配置生效
```

## 配置后端地址

### 方法 1: 修改配置文件

编辑 `web/config.yaml`：

```yaml
backend:
  urls:
    - "http://your-backend:9190"
    - "http://backup-backend:9190"
  currentIndex: 0
```

### 方法 2: 环境变量

创建 `.env` 文件：

```bash
VITE_BACKEND_URL=http://your-backend:9190
```

### 方法 3: 运行时切换

前端应用内添加"切换后端"功能，使用 `configLoader.switchBackend(index)`。

## CORS 配置

前端独立运行时，后端需要启用 CORS。Shepherd 后端默认已启用 CORS（允许所有源）。

如需自定义 CORS 配置，修改后端配置文件 `config/server.config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  web_port: 9190

security:
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
      - "https://your-frontend.com"
    allowed_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
```

## 多后端配置

支持配置多个后端，方便切换：

```yaml
backend:
  urls:
    - "http://dev-server:9190"      # 开发环境
    - "http://staging-server:9190"  # 测试环境
    - "https://api.production.com"  # 生产环境
  currentIndex: 0
```

## 故障排查

### 前端无法连接后端

1. 检查 `config.yaml` 中的后端 URL 是否正确
2. 检查后端服务器是否运行：`curl http://localhost:9190/api/info`
3. 检查浏览器控制台的 CORS 错误

### 配置文件不生效

1. 确认 `config.yaml` 在 `public/` 目录中
2. 清除浏览器缓存并重新加载
3. 检查前端控制台是否有配置加载错误

### SSE 事件不工作

1. 检查后端 SSE 端点：`curl -N http://localhost:9190/api/events`
2. 确认 `config.yaml` 中 `sse.endpoint` 配置正确
3. 检查网络代理是否阻止 SSE 连接

## 文件结构

```
web/
├── config.yaml              # 前端配置文件（用户编辑）
├── public/
│   └── config.yaml         # 配置副本（自动同步）
├── src/
│   ├── lib/
│   │   ├── configLoader.ts  # 配置加载器
│   │   ├── config.ts        # 配置导出
│   │   └── api/
│   │       └── client.ts     # API 客户端
│   └── main.tsx             # 应用入口
└── vite.config.ts          # Vite 配置（无代理）
```

## 总结

- ✅ 前端完全独立，可连接任意后端
- ✅ 后端仅提供数据 API，不控制前端配置
- ✅ 配置文件 `web/config.yaml` 由前端直接读取
- ✅ 支持多后端配置和运行时切换
- ✅ 灵活的部署方式（独立部署、后端托管、反向代理）
