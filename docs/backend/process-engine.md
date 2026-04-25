# 进程引擎

## 设计目标

进程引擎管理 llama.cpp 进程的生命周期，包括启动、停止和输出监控。通过 `BuildCommandFromRequest` 将结构化的 `LoadRequest` 参数转换为 `llama-server` 命令行。

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

`BuildCommandFromRequest(req *LoadRequest, binPath string)` 将 `LoadRequest` 结构体的字段映射为 `llama-server` 命令行参数：

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

## 进程监控

`Process` 提供以下监控能力：

| 方法 | 说明 |
|---|---|
| `IsRunning()` | 通过 `Signal(0)` 检查进程是否存活 |
| `GetPID()` | 获取进程 PID |
| `GetExitCode()` | 获取已退出进程的退出码 |
| `SetOutputHandler()` | 设置输出回调 |
| `Send(input)` | 向 stdin 写入数据 |
