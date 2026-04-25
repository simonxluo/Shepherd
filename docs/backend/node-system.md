# 分布式节点系统

## 角色模型

| 角色 | 说明 | 子系统 |
|---|---|---|
| `master` | 主节点，管理客户端和调度任务 | ClientRegistry + CommandQueue |
| `client` | 客户端，连接到主节点并执行命令 | Registration + Heartbeat + Command |
| `hybrid` | 混合模式（默认），同时具备 master 和 client 能力 | 全部子系统 |

## 核心接口

### INode（15 个方法）

```go
type INode interface {
    // 身份信息
    ID() string
    Name() string
    Role() NodeRole
    Status() NodeStatus
    Address() string
    Port() int

    // 生命周期管理
    Start() error
    Stop() error
    IsRunning() bool

    // 健康检查
    Health() *HealthStatus

    // 配置
    GetConfig() *NodeConfig
    UpdateConfig(*NodeConfig) error

    // 能力和资源
    GetCapabilities() *NodeCapabilities
    GetResources() *NodeResources

    // 上下文
    Context() context.Context
}
```

### ClientRegistry（10 个方法）

```go
type ClientRegistry interface {
    Register(info *NodeInfo) error
    Unregister(nodeID string) error
    Get(nodeID string) (*NodeInfo, error)
    List() []*NodeInfo
    GetStats() *RegistryStats
    Find(predicate func(*NodeInfo) bool) []*NodeInfo
    UpdateStatus(nodeID string, status NodeStatus) error
    UpdateResources(nodeID string, resources *NodeResources) error
    GetOnlineClients() []*NodeInfo
    Cleanup(timeout time.Duration) int
}
```

### CommandQueue（8 个方法）

```go
type CommandQueue interface {
    Enqueue(nodeID string, cmd *Command) error
    Dequeue(nodeID string) (*Command, error)
    Peek(nodeID string) (*Command, error)
    Cancel(commandID string) error
    GetQueueSize(nodeID string) int
    ListQueuedCommands(nodeID string) []*Command
    ClearQueue(nodeID string) int
    RetryCommand(commandID string) error
}
```

### IResourceMonitor（9 个方法）

```go
type IResourceMonitor interface {
    Start() error
    Stop() error
    GetResources() *NodeResources
    GetSnapshot() *NodeResources
    Watch(callback func(*NodeResources))
    SetUpdateInterval(interval time.Duration)
    GetMetrics() *NodeMetrics
    GetGPUInfo() []GPUInfo
    GetLlamacppInfo() *LlamacppInfo
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
Client 构造 NodeInfo → POST /api/master/nodes/register → Master 记录
                                                             ↓
失败？重试延迟 5s，最多 MaxRetries 次 ←───────────────────── 失败
                                                             ↓
成功 → 启动 HeartbeatSubsystem → 定期 POST /api/master/nodes/:id/heartbeat
```

## 心跳协议

| 参数 | 值 |
|---|---|
| 发送间隔 | 5s（可配置，`NodeClientRoleConfig.HeartbeatInterval`） |
| 超时阈值 | 15s（`HeartbeatTimeout`） |
| HTTP 超时 | 10s |
| 最大重试 | 3 次（默认，`RegisterRetry`） |
| 内容 | 状态 + 资源快照 |

## 命令执行流程

```
API → Enqueue → Client 轮询 Dequeue → CommandExecutor (信号量 ≤4 并发)
                                           ↓
                                      超时控制执行
                                           ↓
                                      POST /api/command/result
```

支持 13 种命令类型：

| CommandType | 值 |
|---|---|
| LoadModel | `load_model` |
| UnloadModel | `unload_model` |
| RunLlamacpp | `run_llamacpp` |
| StopProcess | `stop_process` |
| UpdateConfig | `update_config` |
| CollectLogs | `collect_logs` |
| ScanModels | `scan_models` |
| StartTask | `start_task` |
| StopTask | `stop_task` |
| Restart | `restart` |
| Shutdown | `shutdown` |
| TestLlamacpp | `test_llamacpp` |
| GetConfig | `get_config` |

## 资源监控

`NodeResources` 结构体通过 `gopsutil` 和 `comm/gpu` Detector 采集：

| 字段 | 类型 | 说明 |
|---|---|---|
| CPUUsed / CPUTotal | int64 | millicores |
| MemoryUsed / MemoryTotal | int64 | bytes |
| DiskUsed / DiskTotal | int64 | bytes |
| GPUInfo | []gpu.Info | GPU 列表 |
| NetworkRx / NetworkTx | int64 | bytes per second |
| Uptime | int64 | seconds |
| LoadAverage | []float64 | 1/5/15 分钟平均 |
| ROCmVersion | string | ROCm 版本（AMD GPU） |
| KernelVersion | string | Linux 内核版本 |

## 环境能力检测

除了 GPU 硬件信息（`comm/gpu`），`service/node/resource.go` 还检测：

| 能力 | 检测方式 |
|---|---|
| llama.cpp 二进制 | 配置路径遍历 + `--version` / `--help` 解析 |
| ROCm 版本 | `/opt/rocm/.info/version` → `hipcc --version` → `rocm-smi` 多级回退 |
| 内核版本 | `gopsutil host.Info()` |

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
启动: Registration → Heartbeat → Commands → Resource（然后启动其余子系统）
停止: 反序
```

每个子系统实现 `Subsystem` 接口：

```go
type Subsystem interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    IsRunning() bool
}
```
