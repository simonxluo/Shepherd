# Shepherd

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](../LICENSE)

轻量高性能的分布式 llama.cpp 模型管理系统。

[English](../README.md)

---

## 特性

- 启动快 (<500ms)，内存占用低 (~30MB)
- 单一二进制，无运行时依赖
- 支持 Master-Client 分布式部署
- 多协议 API 兼容：OpenAI / Anthropic / Ollama / LM Studio
- Web 界面，支持中英文切换

## 模型管理

- 自动扫描多目录下的 GGUF 模型文件
- 加载/卸载支持丰富参数：
  - 上下文大小、批次大小、线程数、GPU 层数
  - 采样参数：温度、Top-P、Top-K、重复惩罚、Min-P
  - 性能优化：Flash Attention、内存锁定、UBatch、并行槽位
  - KV 缓存类型配置 (K/V)、统一缓存
  - 模板系统、GPU 多设备配置
- 模型收藏、别名、分卷自动识别
- 视觉模型 (mmproj) 支持

## 分布式架构

| 角色 | 说明 |
|------|------|
| **Hybrid**（默认） | 同时作为 Master 和 Client |
| **Master** | 中心管理节点 |
| **Client** | GPU 工作节点，向 Master 注册 |

节点可运行时切换角色。集群提供心跳监控（5秒间隔）、资源上报（CPU/GPU/内存/显存）和资源感知调度。

---

## 快速开始

### 从源码编译

```bash
git clone https://github.com/simonxluo/Shepherd.git
cd Shepherd
make build
```

预编译版本见 [Releases](https://github.com/simonxluo/Shepherd/releases) 页面。

### 配置

配置文件位于 `config/node/*.config.yaml`，通过 `node.role` 字段指定节点角色：

| node.role | 说明 |
|-----------|------|
| `hybrid` | 混合模式（默认） |
| `master` | 主节点 |
| `client` | 工作节点 |

### 运行

```bash
# 默认启动（hybrid 模式）
./build/shepherd

# 指定配置文件
./build/shepherd serve --config config/node/server.config.yaml

# 启动前端开发服务器
./build/shepherd serve --web

# 编译前端后启动
./build/shepherd serve --build --web

# 查看版本
./build/shepherd version
```

Web 界面：http://localhost:9190

---

## 集群部署

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Master 节点   │◄────┤  Hybrid 节点    │◄────┤  Client 节点    │
│   (端口 9190)   │     │ (端口 9190+9191)│     │                 │
└────────┬────────┘     └────────┬────────┘     └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│  Client 节点 1  │     │  Client 节点 2  │
│   (GPU 服务器)  │     │   (GPU 服务器)  │
└─────────────────┘     └─────────────────┘
```

1. 启动 Master 或 Hybrid 节点：
   ```bash
   ./build/shepherd serve --config config/node/server.config.yaml
   ```

2. 启动 Client 节点（配置中设置 `node.role: client` 和 `node.client_role.master_address`）：
   ```bash
   ./build/shepherd serve --config config/node/client.config.yaml
   ```

3. 查看集群状态：
   ```bash
   curl http://master:9190/api/nodes
   ```

---

## API 示例

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:9190/v1",
    api_key="dummy"
)

response = client.chat.completions.create(
    model="llama-2-7b-chat",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

---

## 开发

### 后端 (Go)

```bash
make build       # 编译
make lint        # 代码检查
make fmt         # 格式化
make tidy        # 整理依赖
make build-all   # 跨平台编译
make swag        # 生成 Swagger 文档
```

### 前端 (React + TypeScript)

```bash
cd web
npm install      # 安装依赖
npm run dev      # 开发服务器（端口 3000）
npm run build    # 生产构建
npm run lint     # ESLint 检查
npm run test     # 单元测试
```

---

## 许可证

Apache License 2.0，详见 [LICENSE](../LICENSE)。

## 致谢

- [llama.cpp](https://github.com/ggerganov/llama.cpp)

## 链接

- [问题反馈](https://github.com/simonxluo/Shepherd/issues)
- [讨论](https://github.com/simonxluo/Shepherd/discussions)
- [变更日志](CHANGELOG_zh.md)
