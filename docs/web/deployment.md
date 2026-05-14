# 前端部署

## 架构特点

前端是完全独立的应用，通过 `config.yaml` 配置后端地址，可部署到任何静态服务器。

## 部署模式

### 1. 开发模式

前后端独立运行：

```bash
# 终端 1：后端
./build/shepherd serve

# 终端 2：前端
cd web && npm run dev
```

Vite 开发服务器将 `/api` 代理到 `http://localhost:9190`。

### 2. 生产模式（Nginx）

```bash
cd web && npm run build
# 部署 dist/ 到 Nginx
```

Nginx 配置要点：

```nginx
server {
    listen 3000;
    root /path/to/dist;

    # SPA 路由回退
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 反向代理
    location /api/ {
        proxy_pass http://localhost:9190;
    }

    # SSE 支持
    location /api/events {
        proxy_pass http://localhost:9190;
        proxy_set_header Connection '';
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding off;
    }

    # WebSocket 支持
    location /ws {
        proxy_pass http://localhost:9190;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### 3. 后端托管模式

使用 `shepherd serve --web --build` 一键编译并启动，后端直接服务前端静态文件，单端口访问（默认 9190）。

## 配置文件

`public/config.yaml` 控制前端行为：

```yaml
backend:
  urls:
    - "http://localhost:9190"
    - "https://api.example.com"
  currentIndex: 0
features:
  models: true
  downloads: true
  cluster: true
  logs: true
  chat: true
  settings: true
  dashboard: true
  multimodal: true
```

注意：`backend.urls` 是字符串数组，不是对象数组。

配置同步命令：

```bash
./scripts/linux/sync-web-config.sh
```

从 `config/example/web.config.yaml` 同步到 `web/public/config.yaml`。

## 后端地址切换

两种方式设置后端地址：

1. **配置文件**：编辑 `public/config.yaml` 中的 `backend.urls`，修改 `currentIndex` 选择当前后端
2. **运行时切换**：通过 `updateApiClientUrl(baseUrl)` （来自 `lib/api/client.ts`）动态切换 API 客户端地址

## CORS

后端默认启用 CORS。自定义配置：

```yaml
security:
  cors:
    allowedOrigins: ["http://localhost:3000"]
```

## 故障排除

| 问题 | 排查步骤 |
|---|---|
| 连接失败 | 检查 `config.yaml` URL，`curl /api/info`，确认 CORS |
| 配置不加载 | 确认 `public/config.yaml` 存在，清除浏览器缓存 |
| SSE 不工作 | `curl -N http://localhost:9190/api/events`，检查代理是否缓冲 |
| WebSocket 断开 | 检查代理超时配置，确认 Upgrade 头传递 |
