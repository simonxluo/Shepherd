# Shepherd 分布式架构重构计划

## 🎯 目标
重构 Shepherd 后端为统一 Node 架构，每个节点既可作为 Master 也可作为 Client，支持动态角色切换，实现分布式模型服务。

## 📊 现有架构分析

### 当前问题
1. **角色分离**：Master 和 Client 是独立的可执行文件模式，不能动态切换
2. **信息收集不全**：心跳信息缺少 GPU 详情、llama.cpp 路径、模型位置等关键信息
3. **命令执行弱**：缺乏标准化的 Master 到 Client 的命令下发机制
4. **调度缺失**：没有跨节点选择最佳节点运行模型的能力

### 现有基础
- `internal/cluster/types.go`：已定义 Client, Capabilities, Heartbeat, Task 等数据结构
- `internal/client/master/connector.go`：已有 Client 连接 Master 的基础实现
- `internal/client/executor/executor.go`：已有任务执行器基础
- `internal/cluster/client/manager.go`：已有 Master 端 Client 管理

---

## 🏗️ 新架构设计

### 核心概念：统一 Node 模型

```
┌─────────────────────────────────────────────────────────────┐
│                         Node                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │  MasterRole  │  │  ClientRole  │  │  Standalone  │       │
│  │   (可选)     │  │   (可选)     │  │    (默认)    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              NodeIdentityManager                     │    │
│  │         管理节点角色、状态、元信息                    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Resource   │  │  Heartbeat   │  │   Command    │       │
│  │   Monitor    │  │   Manager    │  │   Executor   │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

### 角色说明

| 角色 | 描述 | 能力 |
|------|------|------|
| **Standalone** | 单机模式 | 所有功能本地执行 |
| **Master** | 主节点 | 管理 Client 节点，分发任务，聚合状态 |
| **Client** | 工作节点 | 向 Master 注册，执行命令，报告状态 |
| **Master+Client** | 混合模式 | 既可管理其他 Client，也可被上层 Master 管理 |

---

## 📦 模块设计

### 1. Node 核心模块 (`internal/node/`)

#### 1.1 NodeIdentityManager
管理节点的身份、角色和状态。

```go
type Node struct {
    ID            string                 `json:"id"`
    Name          string                 `json:"name"`
    Role          NodeRole               `json:"role"` // master/client/standalone
    Status        NodeStatus             `json:"status"`
    Address       string                 `json:"address"`
    Port          int                    `json:"port"`
    Version       string                 `json:"version"`
    Capabilities  *NodeCapabilities      `json:"capabilities"`
    Resources     *NodeResources         `json:"resources"`
    Metadata      map[string]string      `json:"metadata"`
    LastHeartbeat time.Time              `json:"lastHeartbeat"`
    ConnectedTo   string                 `json:"connectedTo,omitempty"` // 连接的 Master ID
    SubNodes      []string               `json:"subNodes,omitempty"`    // 作为 Master 时管理的 Client IDs
}

type NodeRole string
const (
    RoleStandalone NodeRole = "standalone"
    RoleMaster     NodeRole = "master"
    RoleClient     NodeRole = "client"
    RoleHybrid     NodeRole = "hybrid" // 既是 Master 又是 Client
)
```

#### 1.2 ResourceMonitor
全面监控节点资源，包括：
- CPU：核心数、使用率、频率
- GPU：型号、显存、使用率、温度（通过 nvidia-smi 或 rocm-smi）
- 内存：总量、已用、可用
- 磁盘：总量、已用、可用
- llama.cpp：多个路径、版本、支持的计算后端（CUDA/ROCm/Vulkan）
- 模型：已发现模型列表、大小、加载状态

```go
type NodeResources struct {
    CPU      CPUInfo       `json:"cpu"`
    GPUs     []GPUInfo     `json:"gpus"`      // 支持多 GPU
    Memory   MemoryInfo    `json:"memory"`
    Disk     DiskInfo      `json:"disk"`
    Llamacpp []LlamacppInfo `json:"llamacpp"` // 多个 llama.cpp 实例
    Models   []ModelInfo   `json:"models"`    // 可用模型
    Uptime   int64         `json:"uptime"`    // 秒
}

type GPUInfo struct {
    Index       int     `json:"index"`
    Name        string  `json:"name"`
    Vendor      string  `json:"vendor"`      // nvidia/amd/intel
    TotalMemory int64   `json:"totalMemory"` // bytes
    UsedMemory  int64   `json:"usedMemory"`
    Temperature float64 `json:"temperature"`
    Utilization float64 `json:"utilization"` // 0-100
    DriverVersion string `json:"driverVersion"`
}

type LlamacppInfo struct {
    Path           string   `json:"path"`
    Version        string   `json:"version"`
    BuildType      string   `json:"buildType"`      // cuda/rocm/vulkan/metal/cpu
    ComputeCapability string `json:"computeCapability,omitempty"`
    MaxVRAM        int64    `json:"maxVram,omitempty"`
    MaxContextSize int      `json:"maxContextSize,omitempty"`
    SupportsGPU    bool     `json:"supportsGpu"`
    Available      bool     `json:"available"`
}
```

#### 1.3 HeartbeatManager
管理 Client 到 Master 的心跳连接。

**设计要点**：
- 心跳间隔：默认 5 秒，可配置
- 超时检测：3 个心跳周期未收到视为离线
- 指数退避重连：1s, 2s, 4s, 8s... 最大 60s
- 心跳内容：资源快照、当前任务状态、健康检查结果

```go
type HeartbeatManager struct {
    node        *Node
    masterAddr  string
    interval    time.Duration
    timeout     time.Duration
    maxRetries  int
    
    client      *http.Client
    connected   bool
    lastSuccess time.Time
    retryCount  int
    
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
}

type HeartbeatMessage struct {
    NodeID      string          `json:"nodeId"`
    Timestamp   time.Time       `json:"timestamp"`
    Resources   *NodeResources  `json:"resources"`
    Status      NodeStatus      `json:"status"`
    ActiveTasks []TaskStatus    `json:"activeTasks"`
    Health      HealthStatus    `json:"health"`
}

type HealthStatus struct {
    Overall  string            `json:"overall"` // healthy/degraded/unhealthy
    Checks   map[string]bool   `json:"checks"`  // 各组件健康状态
    Issues   []string          `json:"issues,omitempty"`
}
```

#### 1.4 CommandExecutor
安全执行 Master 下发的命令。

**设计要点**：
- 命令签名验证（防止伪造）
- 白名单机制（只允许特定命令）
- 资源限制（CPU、内存、超时）
- 沙箱执行（可选，用于 Python 脚本）
- 实时输出流

```go
type CommandExecutor struct {
    node        *Node
    maxConcurrent int
    timeout     time.Duration
    allowedCommands map[string]bool // 白名单
    
    activeTasks map[string]*TaskExecution
    mu          sync.RWMutex
}

type Command struct {
    ID        string                 `json:"id"`
    Type      CommandType            `json:"type"`
    Payload   map[string]interface{} `json:"payload"`
    Signature string                 `json:"signature"` // 签名防篡改
    Timeout   int                    `json:"timeout"`   // 秒
    Priority  int                    `json:"priority"`  // 优先级
}

type CommandType string
const (
    CmdLoadModel   CommandType = "load_model"
    CmdUnloadModel CommandType = "unload_model"
    CmdRunLlamacpp CommandType = "run_llamacpp"
    CmdStopProcess CommandType = "stop_process"
    CmdUpdateConfig CommandType = "update_config"
    CmdCollectLogs CommandType = "collect_logs"
    CmdScanModels  CommandType = "scan_models"
)

type CommandResult struct {
    CommandID   string                 `json:"commandId"`
    Success     bool                   `json:"success"`
    Output      string                 `json:"output,omitempty"`
    Error       string                 `json:"error,omitempty"`
    Result      map[string]interface{} `json:"result,omitempty"`
    Duration    int64                  `json:"duration"` // ms
    ResourceUsage *ResourceUsage       `json:"resourceUsage,omitempty"`
}
```

### 2. Master 模块 (`internal/master/`)

#### 2.1 NodeManager
管理连接的 Client 节点。

```go
type NodeManager struct {
    nodes      map[string]*NodeInfo
    mu         sync.RWMutex
    
    // 事件广播
    eventCh    chan NodeEvent
    
    // 健康检查
    healthCheckInterval time.Duration
    timeoutThreshold    time.Duration
}

type NodeInfo struct {
    *node.Node
    LastHeartbeat time.Time
    MissedBeats   int
    AssignedTasks []string
    ConnectionQuality float64 // 连接质量评分
}
```

#### 2.2 Scheduler
智能调度模型到最佳节点。

```go
type Scheduler struct {
    nodeManager *NodeManager
    strategy    SchedulingStrategy
}

type SchedulingStrategy interface {
    SelectNode(nodes []*NodeInfo, model *model.Model) (*NodeInfo, error)
}

// 内置策略
// 1. ResourceBasedStrategy：选择资源最充足的节点
// 2. LoadBalancedStrategy：选择负载最低的节点
// 3. LocalityStrategy：优先选择本地有模型的节点
// 4. CostBasedStrategy：选择运行成本最低的节点
```

### 3. Client 模块 (`internal/client/`)

#### 3.1 MasterConnector
连接 Master 并维持心跳。

```go
type MasterConnector struct {
    node           *node.Node
    masterAddr     string
    heartbeatMgr   *node.HeartbeatManager
    commandHandler func(*node.Command) (*node.CommandResult, error)
    
    registered     bool
    reconnectAttempts int
    
    ctx            context.Context
    cancel         context.CancelFunc
    wg             sync.WaitGroup
}
```

### 4. 配置重构

#### 新配置结构

```yaml
# node.config.yaml
node:
  id: "auto"              # auto 表示自动生成
  name: "node-1"
  role: "standalone"      # standalone/master/client/hybrid
  
  # 作为 Master 时的配置
  master:
    enabled: false
    port: 9190
    api_key: ""
    ssl:
      enabled: false
      cert_path: ""
      key_path: ""
    
  # 作为 Client 时的配置
  client:
    enabled: false
    master_address: ""    # 连接的 Master 地址
    register_retry: 3
    heartbeat_interval: 5  # 秒
    heartbeat_timeout: 15  # 秒
    
  # 资源报告配置
  resources:
    monitor_interval: 5    # 秒
    report_gpu: true
    report_temperature: true
    gpu_backend: "auto"    # auto/nvidia/amd/intel
    
  # 命令执行配置
  executor:
    max_concurrent: 4
    task_timeout: 3600     # 秒
    allow_remote_stop: true
    allowed_commands:
      - load_model
      - unload_model
      - run_llamacpp
      - stop_process
      
  # llama.cpp 配置
  llamacpp:
    paths:
      - path: "/usr/local/bin/llama-server"
        version: "b4300"
        backend: "cuda"
        max_vram: 24576000000  # 24GB
      - path: "/opt/llama.cpp/build/bin/llama-server"
        version: "b4300"
        backend: "rocm"
        max_vram: 16384000000  # 16GB
        
# 模型路径（扫描）
models:
  paths:
    - "/models"
    - "/data/gguf"
  
  # 存储后端
storage:
  type: "sqlite"  # sqlite/memory/redis
  path: "data/shepherd.db"
```

---

## 🔧 实施计划

### Wave 1: 基础架构 (4-5 tasks)

**Task 1: 创建 Node 核心类型和接口**
- 文件: `internal/node/types.go`, `internal/node/node.go`
- 内容: Node 结构体、NodeRole、NodeStatus、NodeCapabilities

**Task 2: ResourceMonitor 资源监控**
- 文件: `internal/node/resource.go`
- 内容: CPU、GPU、内存、磁盘监控，llama.cpp 版本检测
- 依赖: gopsutil, nvidia-smi/rocm-smi 调用

**Task 3: HeartbeatManager 心跳管理**
- 文件: `internal/node/heartbeat.go`
- 内容: 定时心跳发送、重连逻辑、健康检查

**Task 4: CommandExecutor 命令执行**
- 文件: `internal/node/executor.go`
- 内容: 命令白名单、签名验证、资源限制、执行流

### Wave 2: Master 功能 (3-4 tasks)

**Task 5: NodeManager 节点管理**
- 文件: `internal/master/node_manager.go`
- 内容: Client 注册、心跳处理、离线检测、事件广播

**Task 6: Master API 端点**
- 文件: `internal/master/handler.go`
- 内容: /api/master/nodes, /api/master/heartbeat, /api/master/command

**Task 7: Scheduler 调度器**
- 文件: `internal/master/scheduler.go`
- 内容: 节点选择策略、负载均衡、模型位置感知

### Wave 3: Client 功能 (2-3 tasks)

**Task 8: MasterConnector 连接管理**
- 文件: `internal/client/connector.go`
- 内容: 向 Master 注册、维持心跳、接收命令

**Task 9: CommandHandler 命令处理**
- 文件: `internal/client/command_handler.go`
- 内容: 处理各种命令类型，调用本地 llama.cpp

### Wave 4: 集成与配置 (2-3 tasks)

**Task 10: 配置重构**
- 文件: `internal/config/config.go`
- 内容: 新配置结构、加载验证

**Task 11: 主程序重构**
- 文件: `cmd/shepherd/main.go`
- 内容: 根据 role 初始化 Node、MasterRole、ClientRole

**Task 12: API 端点集成**
- 文件: `internal/server/server.go`
- 内容: 注册 Master/Client API 路由

### Wave 5: 测试与文档 (2 tasks)

**Task 13: 单元测试**
- 文件: `*_test.go`
- 内容: ResourceMonitor、Heartbeat、Scheduler 测试

**Task 14: 集成测试与文档**
- 文件: `docs/distributed.md`
- 内容: 部署指南、架构图、最佳实践

---

## 🚀 关键设计决策

### 1. 通信协议
- **心跳**: HTTP POST /api/master/heartbeat，轻量级，JSON
- **命令下发**: HTTP POST /api/client/command，Master -> Client
- **实时事件**: WebSocket /api/events，双向流
- **文件传输**: HTTP multipart，用于模型分发（可选）

### 2. 安全设计
- **API Key**: Master 和 Client 共享密钥，用于签名验证
- **TLS**: 可选，用于生产环境
- **命令白名单**: Client 只执行预定义的命令类型
- **资源限制**: 沙箱、超时、内存限制

### 3. 容错设计
- **心跳超时**: 3 个周期未收到视为离线
- **任务重试**: 失败任务自动重试（最多 3 次）
- **断线重连**: 指数退避重连策略
- **健康检查**: 本地健康检查，有问题自动标记为不可用

### 4. 扩展性
- **插件系统**: 支持自定义调度策略（接口）
- **多 GPU**: 支持单节点多 GPU 选择和调度
- **多后端**: CUDA、ROCm、Vulkan、Metal 支持
- **混合云**: 支持本地和云端节点混合部署

---

## 📊 性能指标

| 指标 | 目标 |
|------|------|
| 心跳延迟 | < 10ms（同局域网） |
| 节点发现时间 | < 30s（100 节点集群） |
| 任务调度时间 | < 100ms |
| 资源监控精度 | 1 秒 |
| 最大节点数 | 1000+（Master 节点） |
| 最大并发任务 | 100+（单 Client） |

---

## 🎯 验收标准

1. ✅ 节点可以作为 Standalone、Master、Client、Hybrid 运行
2. ✅ Client 向 Master 发送心跳，包含完整资源信息
3. ✅ Master 可以向 Client 下发命令，Client 执行并返回结果
4. ✅ Master 可以根据资源情况选择最佳节点运行模型
5. ✅ 支持多 GPU 检测和调度
6. ✅ 支持 llama.cpp 多版本管理
7. ✅ 断线后自动重连，任务不丢失
8. ✅ 完整的单元测试覆盖

---

## 📅 时间估算

| Wave | 任务数 | 预估时间 | 并行度 |
|------|--------|----------|--------|
| Wave 1 | 4 | 2-3 天 | 4 |
| Wave 2 | 3 | 2 天 | 3 |
| Wave 3 | 2 | 1-2 天 | 2 |
| Wave 4 | 3 | 2 天 | 3 |
| Wave 5 | 2 | 1-2 天 | 2 |
| **总计** | **14** | **8-10 天** | - |

---

## 🔄 下一步行动

1. 审查本计划
2. 确定优先级（哪些功能必须，哪些可以延后）
3. 开始 Wave 1 实施
4. 运行 `/start-work` 启动执行

请问您想：
1. 修改计划中的某些部分？
2. 调整优先级？
3. 立即开始执行？
4. 保存计划并稍后开始？
