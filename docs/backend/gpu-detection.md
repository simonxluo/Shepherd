# GPU 与环境检测

## 1. 设计目标

Shepherd GPU 检测子系统为分布式模型调度提供硬件能力数据支撑。系统通过 Provider 模式支持 NVIDIA、AMD、Intel 三大 GPU 厂商的检测与动态监控，每种厂商实现独立的检测逻辑和动态指标更新。检测数据通过 `gpu.Info` 结构体统一输出，供节点资源监控和任务调度使用。

系统还通过 llama-bench 工具获取本地 llama.cpp 可用设备列表，并结合 ROCm 版本、Python/Conda 环境、内核版本等环境能力信息，为节点的能力声明提供完整数据。这些数据在节点注册和心跳过程中上报至 Master，构成全局资源视图。

## 2. Provider 模式架构

```mermaid
classDiagram
    class Detector {
        -providers []Provider
        -logger Logger
        +DetectAll(ctx) []Info
        +Update(ctx, gpu) error
        +GetAvailableProviders() []string
    }

    class Provider {
        <<interface>>
        +Name() string
        +Vendor() string
        +IsAvailable() bool
        +Detect(ctx) []Info
        +Update(ctx, gpu) error
    }

    class nvidiaProvider {
        +IsAvailable() nvidia-smi 路径检测
        +Detect() nvidia-smi CSV 解析
        +Update() 单卡指标刷新
    }

    class amdProvider {
        +IsAvailable() rocm-smi 路径检测
        +Detect() rocminfo / rocm-smi 多策略
        +Update() rocm-smi 动态指标
    }

    class amdProvider__SDK_ {
        +IsAvailable() amdsmi 初始化测试
        +Detect() amdsmi SDK 直接调用
        +Update() amdsmi SDK 动态指标
    }

    class intelProvider {
        +IsAvailable() 内核模块检测
        +Detect() lspci 设备解析
        +Update() 无动态指标
    }

    Detector --> Provider : 持有多个
    Provider <|.. nvidiaProvider : 实现
    Provider <|.. amdProvider : 实现（默认）
    Provider <|.. amdProvider__SDK_ : 实现（build tag: amdsmi）
    Provider <|.. intelProvider : 实现
```

`Detector` 作为门面（Facade），在构造时通过 `registerProviders` 注册所有 Provider 实例。`DetectAll` 方法遍历所有可用 Provider 并发执行检测，汇总结果。`Update` 方法根据 GPU 的 Vendor 字段找到对应 Provider 执行动态指标刷新。

## 3. GPU 信息模型

`Info` 结构体统一描述所有厂商的 GPU 信息：

| 字段 | 说明 |
|---|---|
| Index | GPU 索引编号 |
| Name | 设备名称（如 "NVIDIA RTX 4090"） |
| Vendor | 厂商标识（NVIDIA / AMD / Intel） |
| TotalMemory | 显存总量（字节） |
| UsedMemory | 已用显存（字节） |
| Temperature | 温度（摄氏度） |
| Utilization | 利用率（0-100 百分比） |
| PowerUsage | 功耗（瓦特） |
| DriverVersion | 驱动版本号 |

## 4. NVIDIA 检测流程

通过 `nvidia-smi --query-gpu` 命令获取 CSV 格式输出，解析以下字段：index、name、memory.total、memory.used、temperature.gpu、utilization.gpu、power.draw、driver_version。输出使用 `--format=csv,noheader,nounits` 参数，数值无需单位转换。动态更新时针对单卡执行相同命令，通过 `--id` 参数指定索引号。检测超时 10s，更新超时 5s。

## 5. AMD 检测流程

AMD 检测采用多策略降级方案，并支持条件编译切换为 SDK 实现：

**主策略 — rocminfo**：解析 `rocminfo` 输出中的 agent 块，提取 Marketing Name（设备名称）、Device Type（过滤非 GPU 设备）、Pool 1 Size（显存容量，支持 MB/KB 单位）。检测完成后调用 `enrichGPUInfo` 补充驱动版本和显存信息。

**备选策略 — rocm-smi**：使用 `rocm-smi --showproductname` 获取 GPU 名称，输出中查找 "card series" 行。

**动态更新**：`rocm-smi --showmeminfo vram --showtemp --showuse` 解析 VRAM 使用量、GPU 利用率、温度。

**驱动版本**：优先 `modinfo amdgpu`，备选读取 `/sys/module/amdgpu/version`。

**条件编译**：`amd.go` 使用 `!amdsmi` build tag（默认构建），`amd_sdk.go` 使用 `amdsmi` build tag。SDK 实现通过 `github.com/ROCm/amdsmi` 直接调用底层库，无需解析命令输出，支持获取 VBIOS 版本、GPU 忙碌百分比、微瓦级功耗数据。构建命令：`go build -tags amdsmi`。

## 6. Intel 检测流程

通过 `lspci -nn` 检测 Intel GPU 设备，在输出中查找同时包含 "vga" 和 "intel" 关键字的行。可用性检查通过 gopsutil 获取内核版本信息，判断是否包含 `i915` 或 `intel` 内核模块。Intel 集成显卡不支持动态指标更新，Update 方法为空操作。

## 7. llama-bench 集成

系统通过 llama-bench 工具获取 llama.cpp 可用的设备列表。`ParseLlamacppDeviceList` 函数解析 `llama-bench --list-devices` 的输出，按设备前缀（如 CUDA0、ROCm0、SYCL0）进行验证和去重，返回有效的设备标识列表。`FindLlamacppBinary` 函数在指定目录树中搜索 llama.cpp 可执行文件（支持 llama-server、llama-bench、main 等多种二进制名称），也可通过 `LLAMACPP_SERVER_PATH` 环境变量指定路径。

## 8. 环境能力检测

ResourceMonitor 在初始化时执行以下环境检测，检测结果写入 `NodeResources` 和 `NodeCapabilities`：

- **llama.cpp 二进制文件发现**：在配置路径中搜索可执行文件，检测版本号、GPU 后端（cuda/metal/opencl）、支持格式（gguf）
- **ROCm 版本检测**：四级降级策略 — `/opt/rocm/.info/version` → hipcc 路径提取 → rocm-smi lib 版本 → rocm-smi 工具版本
- **Python/Conda 环境检测**：Python 版本、Conda 路径和可用环境列表
- **内核版本获取**：通过 gopsutil/v3 的 host.Info() 获取

## 9. 设计决策

| 决策 | 原因 |
|---|---|
| Provider 模式 | 新厂商只需实现 Provider 接口，注册到 Detector 即可，无需修改现有逻辑 |
| Detector 门面 | 统一调用入口，隐藏多厂商差异，调用方无需关心底层实现 |
| 条件编译（build tag） | AMD SDK 依赖 ROCm 运行时和 CGO，非 AMD 环境无需安装，默认构建使用命令行解析 |
| Logger 接口而非直接依赖 | gpu 包定义自有 Logger 接口（noopLogger 默认实现），避免与 logger 包循环依赖 |
| 多策略降级 | AMD 环境工具链不统一，rocminfo → rocm-smi 降级确保最大兼容性 |
| 指标单位统一为字节 | 所有厂商的显存数据统一为字节单位，上层无需关心厂商差异 |

## 10. SDK

| SDK | 用途 |
|---|---|
| ROCm/amdsmi | AMD GPU 原生检测（条件编译，`-tags amdsmi`） |
| gopsutil/v3 | 系统信息采集（主机、内核版本等） |

## 11. 相关文档

- [全局架构](architecture.md) — 系统全局架构与启动流程
- [节点系统](node-system.md) — 节点资源监控模块使用 GPU 检测数据
- [通用进程引擎](process-engine.md) — 模型加载时使用 llama-bench 检测到的设备信息

> **平台说明**：当前 GPU 检测主要面向 Linux 环境。NVIDIA 检测依赖 `nvidia-smi`（跨平台可用），AMD 检测依赖 `rocminfo`/`rocm-smi`（Linux），Intel 检测依赖 `lspci`（Linux）。Windows 平台通过 NVIDIA `nvidia-smi.exe` 提供有限支持。
