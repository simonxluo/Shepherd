# Shepherd Windows 脚本

本目录包含 Shepherd 项目在 Windows 系统上的构建和运行脚本。

## 📁 脚本列表

| 脚本 | 说明 |
|------|------|
| [build.bat](./build.bat) | 编译 Windows 版本 |
| [run.bat](./run.bat) | 运行 Windows 版本 |
| [web.bat](./web.bat) | 启动 Web 前端开发服务器 |

## 🚀 快速开始

### 1. 安装依赖

#### 方法 1: 使用 Chocolatey (推荐)

```powershell
# 以管理员身份运行 PowerShell
# 安装 Chocolatey (如果尚未安装)
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# 安装 Go
choco install golang git

# 验证安装
go version
```

#### 方法 2: 官方安装包

从以下网站下载并安装：

- **Go**: [https://go.dev/dl/](https://go.dev/dl/)
- **Git**: [https://git-scm.com/downloads/win](https://git-scm.com/downloads/win)

### 2. 编译项目

```batch
REM 从项目根目录
scripts\windows\build.bat

REM 或指定版本
scripts\windows\build.bat v0.1.3
```

编译输出：`build\shepherd-windows-amd64.exe` (AMD64) 或 `build\shepherd-windows-arm64.exe` (ARM64)

### 3. 运行项目

```batch
REM 单机模式
scripts\windows\run.bat standalone

REM Master 模式
scripts\windows\run.bat master

REM Client 模式
scripts\windows\run.bat client --master http://192.168.1.100:9190

REM 运行前先编译
scripts\windows\run.bat standalone -b
```

### 4. Web 前端开发

```batch
REM 启动开发服务器
scripts\windows\web.bat dev

REM 构建生产版本
scripts\windows\web.bat build

REM 预览构建结果
scripts\windows\web.bat preview
```

## 🔧 支持的架构

- **x86_64 (amd64)**: Intel/AMD 64位处理器
- **ARM64**: ARM 64位处理器 (Windows 11 on ARM)

## 📝 环境变量

### 系统环境变量

| 变量 | 说明 |
|------|------|
| `GOPROXY` | Go 模块代理 (默认: https://goproxy.cn,direct) |
| `RUN_TESTS` | 设置为 `true` 在编译后运行测试 |
| `SHEPHERD_CLIENT_NAME` | Client 节点名称 |
| `SHEPHERD_CLIENT_TAGS` | Client 节点标签 |

### PowerShell 设置环境变量

```powershell
# 临时设置（当前会话）
$env:GOPROXY = "https://goproxy.cn,direct"

# 永久设置
[System.Environment]::SetEnvironmentVariable('GOPROXY', 'https://goproxy.cn,direct', 'User')
```

### CMD 设置环境变量

```cmd
REM 临时设置（当前会话）
set GOPROXY=https://goproxy.cn,direct

REM 永久设置
setx GOPROXY "https://goproxy.cn,direct"
```

## 🛠️ Windows 服务

### 使用 NSSM (Non-Sucking Service Manager)

1. **下载 NSSM**: [https://nssm.cc/download](https://nssm.cc/download)

2. **安装服务**:

```batch
REM 以管理员身份运行 CMD
nssm install Shepherd C:\Path\To\Shepherd\build\shepherd-windows-amd64.exe --mode standalone

REM 设置工作目录
nssm set Shepherd AppDirectory C:\Path\To\Shepherd

REM 设置标准输出日志
nssm set Shepherd AppStdout C:\Path\To\Shepherd\logs\shepherd.log

REM 设置错误日志
nssm set Shepherd AppStderr C:\Path\To\Shepherd\logs\shepherd.error

REM 设置自动启动
nssm set Shepherd Start SERVICE_AUTO_START

REM 启动服务
nssm start Shepherd
```

3. **管理服务**:

```batch
REM 停止服务
nssm stop Shepherd

REM 重启服务
nssm restart Shepherd

REM 删除服务
nssm remove Shepherd confirm
```

### 使用 SC 命令 (Windows 内置)

```batch
REM 以管理员身份运行 CMD
sc create Shepherd binPath= "C:\Path\To\Shepherd\build\shepherd-windows-amd64.exe --mode standalone" start= auto
sc start Shepherd

REM 停止服务
sc stop Shepherd

REM 删除服务
sc delete Shepherd
```

## 🔍 防火墙配置

### 添加防火墙规则

```powershell
# 以管理员身份运行 PowerShell
New-NetFirewallRule -DisplayName "Shepherd Server" `
    -Direction Inbound `
    -LocalPort 9190 `
    -Protocol TCP `
    -Action Allow
```

或使用 CMD：

```batch
REM 以管理员身份运行 CMD
netsh advfirewall firewall add rule name="Shepherd Server" dir=in action=allow protocol=TCP localport=9190
```

## 🔍 故障排查

### 编译失败

```batch
REM 检查 Go 版本
go version

REM 清理模块缓存
go clean -modcache

REM 更新 Go 模块
go mod tidy

REM 检查环境变量
echo %GOPROXY%
echo %GOROOT%
echo %GOPATH%
```

### 权限问题

```batch
REM 以管理员身份运行 CMD 或 PowerShell
# 右键点击 CMD/PowerShell -> 以管理员身份运行

REM 或使用 runas 命令
runas /user:Administrator "cmd /k scripts\windows\build.bat"
```

### 端口占用

```batch
REM 检查端口占用
netstat -ano | findstr :9190

REM 停止占用端口的进程
taskkill /PID <进程ID> /F

REM 或使用 PowerShell
Get-NetTCPConnection -LocalPort 9190 | Select-Object OwningProcess
Stop-Process -Id <进程ID> -Force
```

### Windows Defender 误杀

如果 Windows Defender 将二进制文件识别为威胁：

1. **添加排除项**:
   - 打开 Windows 安全中心
   - 病毒和威胁防护 -> 管理设置
   - 排除项 -> 添加或删除排除项
   - 添加文件夹: `C:\Path\To\Shepherd\build`

2. **或使用 PowerShell**:
```powershell
# 以管理员身份运行
Add-MpPreference -ExclusionPath "C:\Path\To\Shepherd\build"
```

### PowerShell 执行策略

```powershell
# 检查执行策略
Get-ExecutionPolicy

# 临时设置为 RemoteSigned（推荐）
Set-ExecutionPolicy -Scope Process -ExecutionPolicy RemoteSigned

# 永久设置（需要管理员权限）
Set-ExecutionPolicy RemoteSigned
```

## 📚 相关文档

- [主 README](../../README.md)
- [Linux 脚本](../linux/README.md)
- [macOS 脚本](../macos/README.md)

## 💡 提示

1. **PowerShell vs CMD**: 推荐使用 PowerShell，功能更强大
2. **行尾符**: Git 配置 `core.autocrlf=true` 避免行尾符问题
3. **路径分隔符**: Windows 使用反斜杠 `\`，但 Go 也能正确处理正斜杠 `/`
4. **长路径**: Windows 有 260 字符路径限制，启用长路径支持：
```powershell
# 以管理员身份运行
New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem" -Name "LongPathsEnabled" -Value 1 -PropertyType DWORD -Force
```

## 🪟 Windows 版本支持

| Windows 版本 | 支持状态 | 备注 |
|-------------|---------|------|
| Windows 11 | ✅ 支持 | 推荐 |
| Windows 10 (22H2) | ✅ 支持 | 推荐 |
| Windows 10 (21H2/21H1) | ✅ 支持 | |
| Windows 10 (2004/20H2/21H1) | ⚠️ 支持 | 可能需要更新 |
| Windows Server 2022 | ✅ 支持 | |
| Windows Server 2019 | ⚠️ 支持 | 需要 .NET Framework 4.8+ |

## 🔧 可选工具

### Windows Terminal

推荐使用 Windows Terminal 获得更好的终端体验：

- **安装**: Microsoft Store 搜索 "Windows Terminal"
- **主题**: 支持自定义主题和配色方案
- **多标签页**: 同时运行多个命令行窗口

### Git for Windows

- **安装**: [https://git-scm.com/downloads/win](https://git-scm.com/downloads/win)
- **功能**: Git Bash, Git GUI, Git integration
- **提示**: 安装时选择 "Use Git from the Windows Command Prompt"

### VS Code

- **安装**: [https://code.visualstudio.com/](https://code.visualstudio.com/)
- **扩展**: Go, PowerShell, GitLens
