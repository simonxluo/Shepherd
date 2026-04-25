# GPU 检测

## 架构

采用 Provider 模式，通过 `Detector` 门类统一管理多个 GPU 厂商的检测实现。

```
Detector (门类)
  ├── NvidiaProvider  (nvidia-smi)
  ├── AMDProvider     (rocm-smi / amdsmi SDK)
  └── IntelProvider   (gopsutil + lspci)
```

## Provider 接口

```go
type Provider interface {
    Name() string                              // "nvidia" / "amd" / "intel"
    Vendor() string                            // "NVIDIA" / "AMD" / "Intel"
    IsAvailable() bool                         // 检测工具是否可用
    Detect(ctx context.Context) ([]Info, error)
    Update(ctx context.Context, gpu *Info) error // 原地更新动态指标
}
```

`Update` 接收 `*Info` 指针，原地修改动态字段（温度、利用率、显存使用），仅返回 `error`。

## GPU 信息模型

```go
type Info struct {
    Index         int     `json:"index"`
    Name          string  `json:"name"`
    Vendor        string  `json:"vendor"`
    TotalMemory   int64   `json:"totalMemory"` // bytes
    UsedMemory    int64   `json:"usedMemory"`  // bytes
    Temperature   float64 `json:"temperature"` // celsius
    Utilization   float64 `json:"utilization"` // percentage 0-100
    PowerUsage    float64 `json:"powerUsage"`  // watts
    DriverVersion string  `json:"driverVersion,omitempty"`
}
```

注意：`TotalMemory` 和 `UsedMemory` 类型为 `int64`，不是 `uint64`。

## 各厂商实现

### NVIDIA

- **检测工具**：`nvidia-smi --query-gpu` CSV 格式
- **超时**：检测 10s，更新 5s
- **字段**：完整的显存、温度、利用率、功耗、驱动版本

### AMD

- **多策略回退**：`rocminfo` → `rocm-smi`
- **条件编译**：`amdsmi` 通过 Go build tag 控制
- **ROCm 版本检测**：4 级回退策略查找 ROCm 安装路径

### Intel

- **可用性检测**：通过 `gopsutil` 的 `host.Info()` 获取内核版本，检查是否包含 `i915` 或 `intel` 关键字
- **设备名检测**：`lspci -nn` 输出中查找含 `vga` 和 `intel` 的行
- **限制**：仅支持设备发现，无动态指标（`Update` 为空操作）

## Detector 门类

```go
type Detector struct {
    providers []Provider
    logger    Logger
}
```

核心方法：

| 方法 | 说明 |
|---|---|
| `DetectAll(ctx)` | 遍历所有可用 Provider 并发检测 GPU |
| `Update(ctx, *Info)` | 根据 GPU 的 Vendor 字段找到对应 Provider 更新 |
| `GetAvailableProviders()` | 返回当前可用的 Provider 名称列表 |

## 环境能力检测

GPU 硬件检测在 `comm/gpu` 包中，以下环境能力检测在 `service/node/resource.go`（`ResourceMonitor`）中：

| 能力 | 检测方式 |
|---|---|
| llama.cpp 二进制 | 配置路径遍历，`--version` 解析版本，`--help` 检测 GPU 后端 |
| ROCm 版本 | `/opt/rocm/.info/version` → `hipcc` 路径提取 → `rocm-smi` 多级回退 |
| 内核版本 | `gopsutil host.Info().KernelVersion` |

## 设计决策

- **Provider 模式**：新增 GPU 厂商只需实现 Provider 接口
- **Build Tag 隔离**：AMD SDK 依赖通过 build tag 控制，不影响默认编译
- **自定义 Logger 接口**：`gpu.Logger` 避免 `comm/gpu` 循环依赖 `comm/logger`
