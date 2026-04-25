# GPU 检测

## 架构

采用 Provider 模式，通过 `Detector` 门类统一管理多个 GPU 厂商的检测实现。

```
Detector (门类)
  ├── NvidiaProvider  (nvidia-smi)
  ├── AMDProvider     (rocm-smi / amdsmi SDK)
  └── IntelProvider   (lspci)
```

## Provider 接口

```go
type Provider interface {
    Name() string        // "nvidia" / "amd" / "intel"
    Vendor() string      // "NVIDIA" / "AMD" / "Intel"
    IsAvailable() bool   // 检测工具是否可用
    Detect(ctx context.Context) ([]Info, error)
    Update(ctx context.Context, gpu Info) (Info, error)
}
```

## GPU 信息模型

```go
type Info struct {
    Index         int     // GPU 序号
    Name          string  // GPU 名称
    Vendor        string  // 厂商
    TotalMemory   uint64  // 总显存 (bytes)
    UsedMemory    uint64  // 已用显存 (bytes)
    Temperature   float64 // 温度 (°C)
    Utilization   float64 // 利用率 (%)
    PowerUsage    float64 // 功耗 (W)
    DriverVersion string  // 驱动版本
}
```

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

- **检测工具**：`lspci -nn`
- **内核模块**：检查 `i915` 模块加载
- **限制**：仅支持设备发现，无动态指标

## 环境能力检测

除了 GPU 硬件信息，还检测以下环境：

| 能力 | 检测方式 |
|---|---|
| llama.cpp 二进制 | PATH + 常见路径搜索 |
| ROCm 版本 | `/opt/rocm.*` 多路径回退 |
| Python/Conda | `which python` / `which conda` |
| 内核版本 | `uname -r` |

## 设计决策

- **Provider 模式**：新增 GPU 厂商只需实现 Provider 接口
- **Build Tag 隔离**：AMD SDK 依赖通过 build tag 控制，不影响默认编译
- **自定义 Logger 接口**：避免 `comm/gpu` 循环依赖 `comm/logger`
