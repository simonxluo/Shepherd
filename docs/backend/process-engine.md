# 进程引擎

## 设计目标

通用进程引擎，通过命令字符串 + 宏替换机制管理 llama.cpp 进程。后端无关设计——理论上可管理任何命令行程序。

## 进程状态机

```
Stopped → Starting → Ready → Stopping → Stopped
                ↓         ↓
              Error ← ─── Error
```

状态转换使用 `sync/atomic` 的 CAS 操作确保线程安全。

## 宏替换系统

构建命令时替换预定义宏：

| 宏 | 说明 |
|---|---|
| `${PORT}` | 分配的端口号 |
| `${MODEL_PATH}` | 模型文件路径 |
| `${MODEL_ID}` | 模型标识符 |
| `${PID}` | 进程 PID |
| `${env.XXX}` | 环境变量 |

支持全局和模型级别的自定义宏，优先级：模型级 > 全局级 > 内置。

## 进程生命周期

```
宏替换 → exec.CommandContext → stdout/stderr 管道 → 健康检查轮询
                                                         ↓
                                            指数退避 (1s → 5s) 直到就绪
```

## 进程分组

三个维度控制模型间的互斥和共存关系：

| 维度 | 说明 |
|---|---|
| `swap` | 同组内互斥，加载新模型自动卸载旧模型 |
| `exclusive` | 加载时驱逐其他组的模型 |
| `persistent` | 不可被其他组驱逐 |

未分组的模型使用默认组。

## TTL 自动卸载

- 定时检查（1s 间隔）
- 通过 WaitGroup 跟踪进行中的请求
- 卸载时广播 SSE 事件通知前端

## 反向代理

`httputil.ReverseProxy` 按请求体中的 `model` 字段路由到对应进程：

- 可配置超时
- 信号量控制并发（默认容量 10）

## 停止策略

| 策略 | 触发场景 | 行为 |
|---|---|---|
| `cmdStop` | 配置了自定义停止命令 | 执行自定义命令 |
| 默认 | 一般停止 | SIGTERM → 10s 超时 → SIGKILL |
| `WaitForInflight` | TTL/组交换 | 等待进行中请求完成 |
| `Immediately` | API 卸载/系统关闭 | 立即停止 |

## 进程监控

| 指标 | 来源 |
|---|---|
| CPU | `/proc/pid/stat` |
| 内存 RSS | `/proc/pid/statm` |
| 日志输出 | 环形缓冲区（100 行） |
| 指标回调 | 注册回调函数 |

## 零外部依赖

进程引擎完全基于 Go 标准库构建：`os/exec`、`net/http/httputil`、`os/signal`、`sync/atomic`、`context`。不依赖任何第三方 SDK。
