# 分布式节点系统

## 角色模型

| 角色 | 说明 | 子系统 |
|---|---|---|
| `master` | 主节点，管理客户端和调度任务 | ClientRegistry + CommandQueue |
| `client` | 客户端，连接到主节点并执行命令 | Registration + Heartbeat + Command |
| `hybrid` | 混合模式（默认），同时具备 master 和 client 能力 | 全部子系统 |

## 核心接口

```go
type INode interface {
    ID() string
    Name() string
    Role() NodeRole
    Status() NodeStatus
    Start() error
    Stop() error
    IsRunning() bool
    GetCapabilities() *NodeCapabilities
    GetResources() *NodeResources
}

type ClientRegistry interface {
    Register(info) error
    Unregister(id) error
    Get(id) (*ClientInfo, error)
    List() []*ClientInfo
    GetOnlineClients() []*ClientInfo
}

type CommandQueue interface {
    Enqueue(nodeID string, cmd Command) error
    Dequeue(nodeID string) (*Command, error)
    Cancel(id string) error
}

type IResourceMonitor interface {
    Start(ctx context.Context) error
    Stop()
    GetResources() *NodeResources
    GetMetrics() *NodeMetrics
}
```

## 节点状态机

```
offline → online → busy / degraded / disabled → error → offline
```

- **online**：正常运行
- **busy**：正在执行任务
- **degraded**：部分功能降级
- **disabled**：被管理员禁用
- **error**：发生错误

## 注册流程（Client 角色）

```
Client 构造 NodeInfo → POST /api/nodes/register → Master 记录
                                                    ↓
失败？指数退避重试 ←────────────────────────────── 失败
                                                    ↓
成功 → 启动 HeartbeatSubsystem → 定期 POST /api/nodes/:id/heartbeat
```

## 心跳协议

| 参数 | 值 |
|---|---|
| 发送间隔 | 30s（可配置） |
| 超时阈值 | 15s |
| 最大重试 | 5 次 |
| 内容 | 状态 + 资源快照 + 能力信息 |

## 命令执行流程

```
API → Enqueue → Client 轮询 Dequeue → CommandExecutor (信号量 ≤4 并发)
                                           ↓
                                      超时控制执行
                                           ↓
                                      POST /api/command/result
```

支持 12 种命令类型：LoadModel、UnloadModel、RunLlamacpp、StopProcess、UpdateConfig、CollectLogs、ScanModels、StartTask、StopTask、Restart、Shutdown、TestLlamacpp、GetConfig。

## 资源监控

通过 `gopsutil` 采集系统指标：

| 指标 | 来源 |
|---|---|
| CPU | millicores |
| 内存 | 已用/总量 |
| 磁盘 | 已用/总量 |
| 负载 | 1/5/15 分钟平均 |
| GPU | `comm/gpu` Detector |
| 运行时间 | `time.Since(startTime)` |

## 网络扫描

`cluster/scanner` 提供子网扫描功能：

- 配置扫描子网和端口范围
- HTTP 探测发现节点
- 支持自动发现模式（定时扫描）

## 任务调度

`cluster/scheduler` 提供三种调度策略：

| 策略 | 说明 |
|---|---|
| `round_robin` | 轮询分配 |
| `least_loaded` | 选择负载最低的节点 |
| `resource_aware` | 根据资源可用性选择 |

生产者-消费者调度循环，支持任务提交、取消、重试。

## 子系统生命周期

`SubsystemManager` 管理子系统的启动和停止顺序：

```
启动: Registration → Heartbeat → Commands → Resource
停止: 反序
```

每个子系统实现 `Subsystem` 接口（`Name()`、`Start()`、`Stop()`、`IsRunning()`）。
