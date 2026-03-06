# 脚本目录迁移指南

Shepherd 脚本已按操作系统重组到子目录中。

## 📁 新目录结构

```
scripts/
├── linux/   # Linux 脚本
├── macos/   # macOS 脚本
└── windows/ # Windows 脚本
```

## 🔄 路径变化

### 编译脚本

| 旧路径 | 新路径 (Linux) | 新路径 (macOS) | 新路径 (Windows) |
|--------|---------------|---------------|-----------------|
| `./scripts/build.sh` | `./scripts/linux/build.sh` | `./scripts/macos/build.sh` | `scripts\windows\build.bat` |

### 运行脚本

| 旧路径 | 新路径 (Linux) | 新路径 (macOS) | 新路径 (Windows) |
|--------|---------------|---------------|-----------------|
| `./scripts/run.sh` | `./scripts/linux/run.sh` | `./scripts/macos/run.sh` | `scripts\windows\run.bat` |

### Web 脚本

| 旧路径 | 新路径 (Linux) | 新路径 (macOS) | 新路径 (Windows) |
|--------|---------------|---------------|-----------------|
| `./scripts/web.sh` | `./scripts/linux/web.sh` | `./scripts/macos/web.sh` | `scripts\windows\web.bat` |

## 📝 更新文档

### Markdown

```markdown
<!-- 旧 -->
运行: `./scripts/run.sh standalone`

<!-- 新 (Linux) -->
运行: `./scripts/linux/run.sh standalone`
```

### Makefile

```makefile
# 使用 OS 检测
UNAME_S := $(shell uname -s)

ifeq ($(UNAME_S),Linux)
    SCRIPT_DIR := scripts/linux
endif
ifeq ($(UNAME_S),Darwin)
    SCRIPT_DIR := scripts/macos
endif

build:
	./$(SCRIPT_DIR)/build.sh
```

## 📚 详细文档

- [Linux 脚本](scripts/linux.md)
- [macOS 脚本](scripts/macos.md)
- [Windows 脚本](scripts/windows.md)
