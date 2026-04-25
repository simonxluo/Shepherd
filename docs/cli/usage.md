# CLI 使用指南

## 全局命令

```
shepherd [command]
```

| 命令 | 说明 |
|---|---|
| `serve` | 启动 Shepherd 服务 |
| `build` | 编译 Shepherd 二进制 |
| `web` | 管理前端（dev/build/preview） |
| `stop` | 停止所有 Shepherd 进程 |
| `version` | 显示版本信息 |

## serve — 启动服务

```bash
shepherd serve [flags]
```

| Flag | 简写 | 默认值 | 说明 |
|---|---|---|---|
| `--config` | `-c` | 自动检测 | 配置文件路径 |
| `--web` | `-w` | false | 同时启动前端开发服务器 |
| `--build` | `-b` | false | 启动前先编译 |
| `--host` | - | 配置文件 | 监听地址 |
| `--port` | - | 配置文件 | 监听端口 |

### 示例

```bash
shepherd serve                          # 默认启动
shepherd serve --web                    # 同时启动前端
shepherd serve --build --web            # 编译 + 前端 + 后端
shepherd serve -c config/custom.yaml    # 指定配置
shepherd serve --port 8080              # 指定端口
```

## build — 编译

```bash
shepherd build [flags]
```

| Flag | 简写 | 默认值 | 说明 |
|---|---|---|---|
| `--version` | `-v` | dev | 版本号 |
| `--output` | `-o` | build/shepherd | 输出路径 |
| `--goos` | - | 当前系统 | 目标操作系统 |
| `--goarch` | - | 当前架构 | 目标架构 |
| `--cross` | - | false | 交叉编译 |
| `--universal` | - | false | macOS 通用二进制 |

### 示例

```bash
shepherd build                          # 编译当前平台
shepherd build -v v1.0.0                # 指定版本
shepherd build --cross --goos linux --goarch arm64  # 交叉编译
shepherd build --universal              # macOS 通用二进制
```

## web — 前端管理

```bash
shepherd web <subcommand>
```

| 子命令 | 说明 |
|---|---|
| `dev` | 启动前端开发服务器 |
| `build` | 编译前端生产版本 |
| `preview` | 预览生产构建 |

### 示例

```bash
shepherd web dev        # 开发
shepherd web build      # 编译
shepherd web preview    # 预览
```

## stop — 停止进程

```bash
shepherd stop [flags]
```

| Flag | 简写 | 默认值 | 说明 |
|---|---|---|---|
| `--force` | `-f` | false | 强制终止 (SIGKILL) |

### 示例

```bash
shepherd stop           # 优雅停止
shepherd stop --force   # 强制终止
```

## version — 版本信息

```bash
shepherd version
```

显示版本号、构建时间和 Git 提交哈希。
