# Windows 脚本

## 编译

```bash
# PowerShell / CMD
make build
```

输出：`build/shepherd.exe`

## 依赖安装

推荐使用 Chocolatey：

```powershell
choco install golang git
```

或手动下载安装器：
- Go: https://go.dev/dl/
- Git: https://git-scm.com/

## 环境变量

### PowerShell

```powershell
$env:GOPROXY = "https://goproxy.cn,direct"
$env:LLAMACPP_SERVER_PATH = "C:\path\to\llama-server.exe"
```

### CMD

```cmd
set GOPROXY=https://goproxy.cn,direct
setx GOPROXY "https://goproxy.cn,direct"
```

## Windows 服务

### NSSM（推荐）

```powershell
# 安装
nssm install Shepherd "C:\shepherd\shepherd.exe" "serve"

# 配置
nssm set Shepherd AppDirectory "C:\shepherd"
nssm set Shepherd StartServiceAutoStart true

# 管理
nssm start Shepherd
nssm stop Shepherd
nssm status Shepherd
nssm remove Shepherd confirm
```

### SC 命令

```cmd
sc create Shepherd binPath= "C:\shepherd\shepherd.exe serve"
sc description Shepherd "Shepherd - llama.cpp Model Manager"
sc start Shepherd
sc stop Shepherd
sc delete Shepherd
```

## 防火墙配置

```powershell
# PowerShell
New-NetFirewallRule -DisplayName "Shepherd" -Direction Inbound -Protocol TCP -LocalPort 9190 -Action Allow
```

```cmd
# CMD
netsh advfirewall firewall add rule name="Shepherd" dir=in action=allow protocol=TCP localport=9190
```

## Windows Defender 排除

```powershell
Add-MpPreference -ExclusionPath "C:\shepherd"
Add-MpPreference -ExclusionProcess "shepherd.exe"
```

## 故障排除

| 问题 | 解决方案 |
|---|---|
| 编译失败 | 确认 Go ≥ 1.25，检查 GOPROXY |
| 权限不足 | 以管理员身份运行 PowerShell |
| 端口占用 | `netstat -ano | findstr :9190` → `taskkill /PID <pid>` |
| 执行策略 | `Set-ExecutionPolicy RemoteSigned -Scope CurrentUser` |
| Defender 误报 | 添加排除路径 |

## Windows 版本支持

Windows 11、Windows 10 (H2 版本)、Windows Server 2022/2019
