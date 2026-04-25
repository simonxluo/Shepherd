# Linux 脚本

## 编译

```bash
make build                # 编译当前平台
make build VERSION=v1.0   # 指定版本
```

输出：`build/shepherd`

## 启动

```bash
# 默认启动（hybrid 模式）
make run

# 指定配置文件
./build/shepherd serve --config config/custom.yaml

# 启动前编译
make run
```

### 一键启动（前后端）

```bash
# 使用 CLI（推荐）
./build/shepherd serve --web --build

# 使用旧脚本
./scripts/linux/run.sh -b --web
```

## 前端开发

```bash
./scripts/linux/web.sh dev      # 开发服务器
./scripts/linux/web.sh build    # 生产编译
./scripts/linux/web.sh preview  # 预览生产构建
```

## 停止

```bash
./build/shepherd stop           # 优雅停止
./build/shepherd stop --force   # 强制停止

./scripts/linux/stop_all.sh     # 旧脚本
```

## 配置同步

```bash
./scripts/linux/sync-web-config.sh        # 单次同步
./scripts/linux/watch-sync-config.sh      # 监视同步
```

## systemd 服务

```ini
[Unit]
Description=Shepherd
After=network.target

[Service]
Type=simple
ExecStart=/path/to/shepherd serve
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable shepherd
sudo systemctl start shepherd
```

## 环境变量

| 变量 | 说明 |
|---|---|
| `GOPROXY` | Go 模块代理 |
| `LLAMACPP_SERVER_PATH` | 自定义 llama.cpp 二进制路径 |

## 架构支持

- x86_64 (amd64)
- ARM64 (aarch64)
- RISC-V (riscv64)

## 故障排除

| 问题 | 解决方案 |
|---|---|
| 编译失败 | 检查 Go 版本 ≥ 1.25，`GOPROXY` 设置 |
| 权限不足 | `chmod +x build/shepherd` |
| 端口占用 | `lsof -i :9190`，`kill <PID>` |
