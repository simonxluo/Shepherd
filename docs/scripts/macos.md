# Shepherd macOS 脚本

本目录包含 Shepherd 项目在 macOS 系统上的构建和运行脚本。

## 📁 脚本列表

| 脚本 | 说明 |
|------|------|
| [build.sh](./build.sh) | 编译 macOS 版本（支持 Intel 和 Apple Silicon）|
| [run.sh](./run.sh) | 运行 macOS 版本 |
| [web.sh](./web.sh) | 启动 Web 前端开发服务器 |

## 🚀 快速开始

### 1. 安装依赖

#### 方法 1: 使用 Homebrew (推荐)

```bash
# 安装 Homebrew (如果尚未安装)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 安装 Go
brew install go

# 验证安装
go version
```

#### 方法 2: 官方安装包

从 [Go 官网](https://go.dev/dl/) 下载 macOS 安装包并安装。

### 2. 编译项目

```bash
# 自动检测架构 (Intel: amd64, Apple Silicon: arm64)
./scripts/macos/build.sh

# 指定版本
./scripts/macos/build.sh v0.1.3

# 构建 Universal Binary (同时支持 Intel 和 Apple Silicon)
BUILD_UNIVERSAL=true ./scripts/macos/build.sh
```

编译输出：
- `build/shepherd-darwin-arm64` (Apple Silicon M1/M2/M3)
- `build/shepherd-darwin-amd64` (Intel)
- `build/shepherd-darwin-universal` (Universal Binary)

### 3. 运行项目

```bash
# 单机模式
./scripts/macos/run.sh standalone

# Master 模式
./scripts/macos/run.sh master

# Client 模式
./scripts/macos/run.sh client --master http://192.168.1.100:9190

# 运行前先编译
./scripts/macos/run.sh standalone -b

# 跳过 Gatekeeper 验证（解决隔离问题）
./scripts/macos/run.sh standalone --no-gatekeeper
```

### 4. Web 前端开发

```bash
# 启动开发服务器
./scripts/macos/web.sh dev

# 构建生产版本
./scripts/macos/web.sh build

# 预览构建结果
./scripts/macos/web.sh preview
```

## 🔧 支持的架构

- **ARM64**: Apple Silicon (M1, M2, M3, M1 Pro, M1 Max, M1 Ultra, M2 Pro, M2 Max, M2 Ultra)
- **x86_64**: Intel 处理器
- **Universal Binary**: 同时支持 Intel 和 Apple Silicon

## 🔐 代码签名

### 自签名代码

```bash
# 创建自签名证书
# 1. 打开 "钥匙串访问"
# 2. 菜单: 钥匙串访问 > 证书助理 > 创建证书
# 3. 名称: Shepherd Developer ID
# 4. 类型: 代码签名
# 5. 勾选: 让我覆盖这些默认设置

# 使用证书签名
CODESIGN_IDENTITY="Shepherd Developer ID" ./scripts/macos/build.sh
```

### 移除隔离属性 (Gatekeeper)

如果遇到无法打开应用的问题：

```bash
# 方法 1: 使用脚本参数
./scripts/macos/run.sh standalone --no-gatekeeper

# 方法 2: 手动移除
xattr -cr build/shepherd-darwin-arm64

# 方法 3: 允许任何来源 (macOS 12 及更早)
sudo spctl --master-disable
```

## 📝 环境变量

| 变量 | 说明 |
|------|------|
| `GOPROXY` | Go 模块代理 (默认: https://goproxy.cn,direct) |
| `RUN_TESTS` | 设置为 `true` 在编译后运行测试 |
| `BUILD_UNIVERSAL` | 设置为 `true` 构建 Universal Binary |
| `CODESIGN_IDENTITY` | 代码签名证书身份 |
| `SHEPHERD_CLIENT_NAME` | Client 节点名称 |
| `SHEPHERD_CLIENT_TAGS` | Client 节点标签 |

## 🛠️ Launch Agent (开机自启)

创建 `~/Library/LaunchAgents/com.shepherd.server.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.shepherd.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/Shepherd/build/shepherd-darwin-arm64</string>
        <string>--mode=standalone</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>WorkingDirectory</key>
    <string>/Applications/Shepherd</string>
    <key>StandardOutPath</key>
    <string>/tmp/shepherd.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/shepherd.error</string>
</dict>
</plist>
```

加载服务：

```bash
launchctl load ~/Library/LaunchAgents/com.shepherd.server.plist
launchctl start com.shepherd.server
```

## 🔍 故障排查

### Gatekeeper 隔离问题

```bash
# 检查是否有隔离属性
xattr -l build/shepherd-darwin-arm64

# 移除隔离属性
xattr -cr build/shepherd-darwin-arm64
```

### 编译失败

```bash
# 检查 Go 版本 (需要 1.21+)
go version

# 更新 Go
brew upgrade go

# 清理模块缓存
go clean -modcache
```

### Apple Silicon 特定问题

```bash
# 确认架构
uname -m

# 应该显示: arm64

# 如果编译为 x86_64，检查 Rosetta 2
arch -x86_64 uname -m

# 安装 Rosetta 2 (如果需要)
softwareupdate --install-rosetta
```

### 构建问题

```bash
# 更新 Xcode Command Line Tools
softwareupdate --all --install --force

# 或单独安装
xcode-select --install
```

## 📚 相关文档

- [主 README](../../README.md)
- [Linux 脚本](../linux/README.md)
- [Windows 脚本](../windows/README.md)

## 🍎 macOS 版本支持

| macOS 版本 | 支持状态 | 备注 |
|-----------|---------|------|
| macOS 15 Sequoia | ✅ 支持 | 需要最新 Xcode Tools |
| macOS 14 Sonoma | ✅ 支持 | 推荐 |
| macOS 13 Ventura | ✅ 支持 | |
| macOS 12 Monterey | ⚠️ 支持 | 可能需要更新 Xcode Tools |
| macOS 11 Big Sur | ⚠️ 有限支持 | 需要更新 Xcode Tools |

## 💡 提示

1. **Universal Binary**: 如果需要在 Intel 和 Apple Silicon 之间共享二进制，使用 `BUILD_UNIVERSAL=true`
2. **性能优化**: Apple Silicon 设备直接使用 arm64 版本可获得最佳性能
3. **Rosetta 2**: 仅在无法编译 arm64 版本时使用 Rosetta 2 运行 x86_64 版本
4. **代码签名**: 发布到外部的应用应该使用 Apple Developer ID 进行签名
