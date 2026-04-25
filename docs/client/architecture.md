# 桌面客户端（规划中）

> **状态**：尚未实现。无 `electron/` 或 `client/` 目录。

## 设计目标

为 Windows/macOS 提供原生桌面客户端，最大化 WebUI 代码复用。

## 架构方案

```
Electron Main Process (Node.js)
  ├── 管理 Go 后端生命周期（spawn shepherd.exe）
  ├── BrowserWindow 加载 http://localhost:9190
  ├── 系统托盘
  └── 自动更新（electron-updater + GitHub Releases）

Renderer Process
  └── 直接加载 http://localhost:9190 WebUI（无需修改前端代码）

IPC Bridge (preload/contextBridge)
  ├── backend:status / start / stop
  ├── window:minimize
  └── update:check / available / downloaded
```

## 启动流程

```
Electron 启动 → spawn shepherd.exe → 健康检查循环 → 创建 BrowserWindow → 加载 localhost:9190
```

## 关闭流程

```
SIGTERM → shepherd 优雅关闭 (10s) → SIGKILL 回退
```

## 打包

- **工具**：electron-builder
- **内嵌**：Go 交叉编译的二进制文件
- **格式**：NSIS 安装包 + 便携版（Windows）

## 未来扩展

- macOS (.dmg)
- Linux (AppImage / .deb)
- 系统托盘右键菜单
- 后台运行模式
- 离线错误检测与自动重启（最多 3 次）
