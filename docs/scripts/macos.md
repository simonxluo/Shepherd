# macOS 脚本

## 架构支持

| 架构 | 适用设备 |
|---|---|
| arm64 | M1/M2/M3/M4 (Apple Silicon) |
| amd64 | Intel Mac |
| universal | 通用二进制（合并 arm64 + amd64） |

## 编译

```bash
make build                              # 自动检测当前架构
BUILD_UNIVERSAL=true make build-all     # 编译通用二进制
```

输出：
- `build/shepherd-darwin-arm64` (Apple Silicon)
- `build/shepherd-darwin-amd64` (Intel)
- `build/shepherd-darwin-universal` (通用)

## 依赖安装

推荐使用 Homebrew：

```bash
brew install go
```

或从 Go 官网下载安装器。

## Gatekeeper 处理

首次运行可能被 macOS Gatekeeper 拦截：

```bash
# 方法 1：移除隔离属性
xattr -cr build/shepherd

# 方法 2：系统偏好设置 → 安全性与隐私 → 允许运行
```

## 代码签名

```bash
# 创建自签名证书（钥匙串访问 → 证书助理）
# 然后签名：
CODESIGN_IDENTITY=" Shepherd Developer" codesign --sign "$CODESIGN_IDENTITY" build/shepherd
```

## LaunchAgent 自动启动

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.shepherd</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/shepherd</string>
        <string>serve</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

```bash
cp com.shepherd.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.shepherd.plist
```

## 环境变量

| 变量 | 说明 |
|---|---|
| `GOPROXY` | Go 模块代理 |
| `BUILD_UNIVERSAL` | 编译通用二进制 |
| `CODESIGN_IDENTITY` | 代码签名身份 |
| `LLAMACPP_SERVER_PATH` | 自定义 llama.cpp 路径 |

## macOS 版本支持

macOS 15 Sequoia → macOS 11 Big Sur

## 故障排除

| 问题 | 解决方案 |
|---|---|
| Gatekeeper 隔离 | `xattr -cr build/shepherd` |
| Go 版本过低 | `brew upgrade go` |
| Apple Silicon Rosetta | 确认 `uname -m` 输出 arm64 |
| Xcode 工具缺失 | `xcode-select --install` |
