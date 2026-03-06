# Shepherd Linux 脚本

本目录包含 Shepherd 项目在 Linux 系统上的构建和运行脚本。

## 📁 脚本列表

| 脚本 | 说明 |
|------|------|
| [build.sh](./build.sh) | 编译 Linux 版本 |
| [run.sh](./run.sh) | 运行 Linux 版本 |
| [web.sh](./web.sh) | 启动 Web 前端开发服务器 |
| [sync-web-config.sh](./sync-web-config.sh) | 同步 Web 配置 |
| [watch-sync-config.sh](./watch-sync-config.sh) | 监视并自动同步配置 |

## 🚀 快速开始

### 1. 安装依赖

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install golang git

# Fedora/RHEL
sudo dnf install golang git

# Arch Linux
sudo pacman -S go git
```

### 2. 编译项目

```bash
# 从项目根目录
./scripts/linux/build.sh

# 或指定版本
./scripts/linux/build.sh v0.1.3
```

编译输出：`build/shepherd` (amd64) 或 `build/shepherd-linux-arm64` (ARM64)

### 3. 运行项目

```bash
# 单机模式
./scripts/linux/run.sh standalone

# Master 模式
./scripts/linux/run.sh master

# Client 模式
./scripts/linux/run.sh client --master http://192.168.1.100:9190

# 运行前先编译
./scripts/linux/run.sh standalone -b
```

### 4. Web 前端开发

```bash
# 启动开发服务器
./scripts/linux/web.sh dev

# 构建生产版本
./scripts/linux/web.sh build

# 预览构建结果
./scripts/linux/web.sh preview
```

## 🔧 支持的架构

- **x86_64 (amd64)**: Intel/AMD 64位处理器
- **ARM64 (aarch64)**: ARM 64位处理器
- **RISC-V**: RISC-V 64位处理器

## 📝 环境变量

| 变量 | 说明 |
|------|------|
| `GOPROXY` | Go 模块代理 (默认: https://goproxy.cn,direct) |
| `RUN_TESTS` | 设置为 `true` 在编译后运行测试 |
| `SHEPHERD_CLIENT_NAME` | Client 节点名称 |
| `SHEPHERD_CLIENT_TAGS` | Client 节点标签 |

## 🛠️ 系统服务 (systemd)

创建 systemd 服务单元文件 `/etc/systemd/system/shepherd.service`:

```ini
[Unit]
Description=Shepherd Model Server
After=network.target

[Service]
Type=simple
User=shepherd
WorkingDirectory=/opt/shepherd
ExecStart=/opt/shepherd/build/shepherd --mode=standalone
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启用服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable shepherd
sudo systemctl start shepherd
sudo systemctl status shepherd
```

## 🔍 故障排查

### 编译失败

```bash
# 检查 Go 版本
go version

# 清理模块缓存
go clean -modcache

# 更新 Go 模块
go mod tidy
```

### 权限问题

```bash
# 添加执行权限
chmod +x ./scripts/linux/*.sh

# 二进制文件执行权限
chmod +x ./build/shepherd
```

### 端口占用

```bash
# 检查端口占用
sudo ss -tulpn | grep :9190

# 停止占用端口的进程
sudo kill <PID>
```

## 📚 相关文档

- [主 README](../../README.md)
- [macOS 脚本](../macos/README.md)
- [Windows 脚本](../windows/README.md)
