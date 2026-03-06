# Shepherd 脚本总览

Shepherd 项目提供跨平台的构建和运行脚本。

## 📁 目录结构

```
scripts/
├── linux/              # Linux 脚本
├── macos/              # macOS 脚本
├── windows/            # Windows 脚本
├── build-all.sh        # 跨平台编译
├── release.sh          # 发布打包
└── README.md
```

## 🚀 快速开始

### Linux

```bash
./scripts/linux/build.sh          # 编译
./scripts/linux/run.sh standalone  # 运行
./scripts/linux/web.sh dev        # Web 前端
```

### macOS

```bash
./scripts/macos/build.sh
./scripts/macos/run.sh standalone
./scripts/macos/web.sh dev
```

### Windows

```batch
scripts\windows\build.bat
scripts\windows\run.bat standalone
scripts\windows\web.bat dev
```

## 🔧 跨平台构建

```bash
# 构建所有平台
./scripts/build-all.sh v0.2.0

# 创建发布包
./scripts/release.sh v0.2.0
```

## 📝 详细文档

- [Linux 脚本](scripts/linux.md)
- [macOS 脚本](scripts/macos.md)
- [Windows 脚本](scripts/windows.md)
- [迁移指南](migration.md)
