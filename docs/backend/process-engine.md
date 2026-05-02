# 进程引擎

## 设计目标

进程引擎管理 llama.cpp 及其他推理后端（vLLM、vLLM-Omni）进程的生命周期，包括启动、停止和输出监控。通过可插拔的 `Backend` 接口（`internal/service/model/backend/`）将结构化的 `LoadRequest` 参数转换为目标后端的命令行。每种后端实现 `Discover / BuildStartConfig / IsLoadComplete / CheckHealth / SupportsModel` 方法。

## 进程状态

`Process` 结构体使用 `Running bool` 字段跟踪运行状态，通过 `sync.Mutex` 保护。没有复杂的状态机，只有运行/未运行两种状态：

- 启动时设置 `Running = true`，并验证进程是否真正存活（500ms 后 `Signal(0)` 检查）
- 停止时设置 `Running = false`

```go
type Process struct {
    ID      string
    Name    string
    Cmd     string
    BinPath string
    PID     int
    Running bool
    CtxSize int
    Port    int
    // ...
}
```

## 命令构建

通过 `Backend.BuildStartConfig()` 将 `LoadRequest` 参数映射为命令行。不同后端生成不同的命令：

| 后端 | 命令示例 |
|---|---|
| llama.cpp | `llama-server --port ${PORT} -m ${MODEL_PATH} -ngl 99` |
| vLLM | `python -m vllm.entrypoints.openai.api_server --port ${PORT} --model ${MODEL_PATH}` |
| vLLM-Omni | `vllm-omni serve --port ${PORT} --model ${MODEL_PATH}`（多模态：TTS/ASR/图像生成） |

llama.cpp 的 `BuildCommandFromRequest(req *LoadRequest, binPath)` 具体步骤：

1. 查找 `llama-server` 二进制（`utils.FindLlamacppBinary(binPath, "server")`）
2. 构建基础参数：`-m`（模型路径）、`--port`、`--host 0.0.0.0`
3. 按类别追加可选参数：

| 类别 | 参数 |
|---|---|
| 上下文/批处理 | `-c`、`-b`、`-t`、`-tb` |
| GPU 配置 | `-ngl`、`-sm`、`-dev`、`-mg`、`-ts` |
| 多模态 | `--mmproj` |
| 性能标志 | `-fa on`、`--no-mmap`、`--mlock` |
| 服务器功能 | `--no-webui`、`--metrics`、`--slot-save-path`、`--cache-ram` |
| 聊天模板 | `--chat-template-file` |
| 批处理 | `--ubatch-size`、`--parallel` |
| KV 缓存 | `-ctk`、`-ctv`、`-kvu`、`--cache-ram` |
| 采样参数 | `--temp`、`--top-p`、`--top-k`、`--repeat-penalty`、`--min-p` 等 |
| 模板/处理 | `--no-jinja`、`--chat-template`、`--context-shift` |
| 结构化生成 | `--grammar`、`--grammar-file` |
| LoRA 适配器 | `--lora`、`--lora-scaled` |
| RoPE 缩放 | `--rope-scaling`、`--rope-scale`、`--rope-freq-base`、`--rope-freq-scale` |
| 优化 | `--cont-batching`/`--no-cont-batching`、`--cache-prompt` |
4. 使用 `quoteAndJoin()` 对参数正确引用拼接为命令字符串
5. 追加 `CustomCmd` 和 `ExtraParams`（如有）

## 进程生命周期

```
NewProcess() → Start()
                  ↓
        splitCommandLineArgs(cmd) 解析命令行
                  ↓
        exec.CommandContext() 创建子进程
                  ↓
        设置环境（LD_LIBRARY_PATH 加入 binPath 目录）
                  ↓
        stdout/stderr 管道 → outputChan → processOutput() 处理
                  ↓
        500ms 后 Signal(0) 验证进程存活
```

## Process Manager

`Manager` 管理多个进程实例，使用两个 map 区分状态：

- `processes map[string]*Process`：已加载（运行中）的进程
- `loading map[string]*Process`：正在启动的进程

核心方法：

| 方法 | 说明 |
|---|---|
| `Start(modelID, name, cmd, binPath)` | 启动新进程（先放 loading，成功后移入 processes） |
| `Stop(modelID)` | 停止指定进程 |
| `Get(modelID)` | 获取进程（先查 processes，再查 loading） |
| `List()` | 返回所有运行中进程的副本 |
| `ListAll()` | 返回 running + loading 两部分 |
| `StopAll()` | 停止所有进程（running + loading） |
| `IsRunning(modelID)` | 检查指定模型是否运行中 |

## 停止流程

```
Stop() 调用
    ↓
cancel() 取消 context
    ↓
关闭 stdin 管道
    ↓
SIGTERM 信号
    ↓
等待 5 秒
    ↓
超时则 SIGKILL 强制终止
    ↓
等待输出读取器结束（最多 2 秒）
```

## 输出处理

- stdout/stderr 各有一个 reader goroutine，将行发送到 `outputChan`（缓冲 100）
- `processOutput` goroutine 从 channel 读取并调用 `outputHandler` 回调
- 过滤噪声日志（`update_slots`、`log_server_r`）
- 当两个 reader 都完成后自动关闭 channel

## 反向代理模式

引擎使用 `httputil.ReverseProxy` 将推理请求转发到对应的上游进程，实现请求路由与模型进程的解耦。

**路由机制：** 仅从请求体的 `model` 字段提取模型标识，以此确定目标进程的监听端口，不解析完整请求体。

**并发控制：** 通过信号量通道（默认容量 10）限制并行转发请求数。超出限制时返回 HTTP 429 状态码，提示客户端稍后重试。

### 多模态请求代理

多模态请求（TTS/ASR/图像生成）通过与 Chat/Completion 相同的反向代理模式路由到后端进程。根据请求和响应的内容类型，使用不同的转发方法：

| 方法 | 用途 | 特点 |
|---|---|---|
| `ForwardRequest` | 图像生成（JSON 请求 → JSON 响应） | 标准 JSON 代理 |
| `ForwardBinaryRequest` | TTS（JSON 请求 → 二进制音频响应） | 透传 Content-Type，支持音频流 |
| `ForwardMultipartRequest` | ASR（multipart/form-data 上传音频文件） | 重组 multipart 表单并转发 |

所有转发方法均执行 inflight 请求追踪（WaitGroup）和并发槽位控制（AcquireSlot/ReleaseSlot），与文本推理请求保持一致的生命周期管理。

## 进程监控

`Process` 提供以下监控能力：

| 方法 | 说明 |
|---|---|
| `IsRunning()` | 通过 `Signal(0)` 检查进程是否存活 |
| `GetPID()` | 获取进程 PID |
| `GetExitCode()` | 获取已退出进程的退出码 |
| `SetOutputHandler()` | 设置输出回调 |
| `Send(input)` | 向 stdin 写入数据 |

## 设计决策

| 决策 | 理由 |
|---|---|
| cmd 字符串 + Backend 接口 | 宏替换保留最大灵活性，Backend 接口封装后端差异，新增后端仅需实现接口并注册 |
| 宏替换在配置加载时执行 | 运行时零开销，类型安全保持 |
| httputil.ReverseProxy | Go 标准库方案，成熟可靠，无额外依赖 |
| 信号量并发控制 | 轻量级限流，避免上游过载 |

## 相关文档

- [全局架构](architecture.md) — 系统整体架构与模块关系
- [模型管理](model-management.md) — 模型加载流程使用本引擎的进程管理能力
- [节点系统](node-system.md) — 跨节点指令触发远程进程操作
- [GPU 检测](gpu-detection.md) — llama-bench 设备列表与本引擎的设备参数映射
