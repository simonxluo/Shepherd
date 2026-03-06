// Shepherd - llama.cpp 模型管理系统
// 这是主程序入口文件
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/api"
	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/langchain"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
	"github.com/shepherd-project/shepherd/Shepherd/internal/netutil"
	"github.com/shepherd-project/shepherd/Shepherd/internal/node"
	"github.com/shepherd-project/shepherd/Shepherd/internal/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/server"
	"github.com/shepherd-project/shepherd/Shepherd/internal/shutdown"
	"github.com/shepherd-project/shepherd/Shepherd/internal/storage"
)

// 版本信息（编译时注入）
var (
	Version   = "v0.6.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// App 表示应用程序实例，包含所有组件
type App struct {
	// 基础组件
	cfg           *config.Config
	configMgr     *config.Manager
	procMgr       *process.Manager
	portAllocator *port.PortAllocator
	storageMgr    *storage.Manager
	modelMgr      *model.Manager
	shutdownMgr   *shutdown.Manager
	srv           *server.Server

	// 分布式节点组件
	node        *node.Node       // 统一节点实例
	nodeAdapter *api.NodeAdapter // Node API 适配器

	// LangChainGo 组件
	langchainMgr     *langchain.Manager   // LangChainGo 管理器
	langchainHandler *langchain.Handler // LangChainGo API 处理器

	// 运行模式
	role string
}

func main() {
	// 命令行参数
	version := flag.Bool("version", false, "显示版本信息")
	configPath := flag.String("config", "", "配置文件路径 (可选)")
	flag.Parse()

	// 显示版本信息
	if *version {
		fmt.Printf("Shepherd v%s\n", Version)
		fmt.Printf("构建时间: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// 打印启动信息
	printBanner()

	// 创建应用程序实例
	app := &App{}

	// 初始化应用程序
	if err := app.Initialize(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 启动应用程序
	if err := app.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}

	// 等待关闭
	app.Wait()

	fmt.Println("✓ 服务器已关闭")
	fmt.Println("再见！")
}

// printBanner 打印启动横幅
func printBanner() {
	fmt.Print(`
╔═════════════════════════════════════════════════════╗
║                                                    ║ 
║   Shepherd - llama.cpp 模型管理系统                   ║
║   (C) 2026 Shepherd Project                         ║
║                                                      ║
║   分布式管理 - 更快、更轻、更简单              		║
║                                                      ║
╚═════════════════════════════════════════════════════╝
`)
	fmt.Printf("版本: %s\n", Version)
	fmt.Printf("Commit: %s\n\n", GitCommit)
}

// Initialize 初始化应用程序
func (app *App) Initialize(configPath string) error {
	// 创建配置管理器
	if configPath != "" {
		app.configMgr = config.NewManagerWithPath(configPath)
	} else {
		app.configMgr = config.NewManager()
	}

	// 加载配置
	cfg, err := app.configMgr.Load()
	if err != nil {
		fmt.Printf("警告: 无法加载配置文件，使用默认配置: %v\n", err)
		cfg = config.DefaultConfig()
	}
	app.cfg = cfg

	// 确定节点角色（直接从配置读取）
	app.role = app.determineRole()

	// 初始化日志系统
	if err := logger.InitLogger(&cfg.Log, app.role); err != nil {
		fmt.Printf("警告: 无法初始化日志系统: %v\n", err)
	}

	// 初始化日志流用于实时查看
	logger.InitLogStream(1000)

	logger.Info("Shepherd 正在启动...")
	logger.Infof("版本: %s", Version)
	logger.Infof("节点角色: %s", app.role)
	logger.Infof("配置文件: %s", app.configMgr.GetConfigPath())

	// 创建进程管理器
	app.procMgr = process.NewManager()

	// 创建端口分配器（统一管理所有模型服务端口）
	// 从配置文件读取端口范围，默认 8081-9000
	basePort, maxPort := 8081, 9000
	if cfg.Model.PortRange != "" {
		_, err := fmt.Sscanf(cfg.Model.PortRange, "%d-%d", &basePort, &maxPort)
		if err != nil {
			logger.Warnf("无效的端口范围配置: %s，使用默认值 8081-9000", cfg.Model.PortRange)
			basePort, maxPort = 8081, 9000
		} else {
			logger.Infof("使用配置的模型端口范围: %d-%d", basePort, maxPort)
		}
	}
	app.portAllocator = port.NewPortAllocator(basePort, maxPort)

	// 创建存储管理器（用于存储模型元数据等）
	storageCfg := cfg.Storage
	// 如果没有配置存储，使用默认的SQLite配置
	if storageCfg.Type == "" {
		storageCfg = storage.StorageConfig{
			Type: storage.StorageTypeSQLite,
			SQLite: &storage.SQLiteConfig{
				Path:      "./data/shepherd.db",
				EnableWAL: true,
			},
		}
	}
	storageMgr, err := storage.NewManager(&storageCfg)
	if err != nil {
		return fmt.Errorf("无法创建存储管理器: %w", err)
	}
	app.storageMgr = storageMgr
	logger.Info("存储管理器初始化成功", "type", storageCfg.Type)

	// 创建模型管理器（传入端口分配器和存储管理器）
	app.modelMgr = model.NewManager(cfg, app.configMgr, app.procMgr, app.portAllocator, app.storageMgr)

	// 创建 LangChainGo 管理器和 API 处理器
	app.langchainMgr = langchain.NewManager(app.modelMgr, logger.GetLogger())
	app.langchainHandler = langchain.NewHandler(app.langchainMgr, logger.GetLogger())
	logger.Info("LangChainGo 组件已初始化")

	// 根据角色初始化分布式组件
	if err := app.initDistributedComponents(); err != nil {
		return fmt.Errorf("初始化分布式组件失败: %w", err)
	}

	// 创建 HTTP 服务器
	serverCfg := &server.Config{
		WebPort:       cfg.Server.WebPort,
		AnthropicPort: cfg.Server.AnthropicPort,
		OllamaPort:    cfg.Server.OllamaPort,
		LMStudioPort:  cfg.Server.LMStudioPort,
		Host:          cfg.Server.Host,
		ReadTimeout:   time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:  time.Duration(cfg.Server.WriteTimeout) * time.Second,
		WebUIPath:     "./web/dist",
		ServerCfg:     cfg,
		ConfigMgr:     app.configMgr,
		Version:       Version,
		BuildTime:     BuildTime,
		GitCommit:     GitCommit,
	}

	// 创建 HTTP 服务器
	app.srv, err = server.NewServer(serverCfg, app.modelMgr)
	if err != nil {
		return fmt.Errorf("无法创建服务器: %w", err)
	}

	// 注册 LangChainGo API 路由（所有模式都可用）
	if app.langchainHandler != nil {
		app.srv.RegisterLangChainHandler(app.langchainHandler)
		logger.Info("LangChainGo API 已启用")
	}

	// 注册 Node API 路由（如果是 master 或 hybrid 模式）
	if app.role == "master" || app.role == "hybrid" {
		if app.nodeAdapter != nil {
			app.srv.RegisterNodeAdapter(app.nodeAdapter)
		}
	}

	// 创建优雅关闭管理器
	app.shutdownMgr = shutdown.NewManager(10 * time.Second)

	// 注册关闭钩子
	app.registerShutdownHooks()

	return nil
}

// determineRole 根据配置确定节点角色
func (app *App) determineRole() string {
	// 直接从配置读取节点角色
	role := app.cfg.Node.Role
	if role == "" {
		return "hybrid" // 默认使用 hybrid
	}
	return role
}
func (app *App) initDistributedComponents() error {
	logger.Infof("初始化分布式组件，角色: %s", app.role)

	switch app.role {
	case "master":
		// Master 模式：创建 Node + NodeAdapter
		if err := app.initMasterNode(); err != nil {
			return fmt.Errorf("初始化 master 节点失败: %w", err)
		}
		if err := app.initNodeAdapter(); err != nil {
			return fmt.Errorf("初始化 Node API 适配器失败: %w", err)
		}

	case "client":
		// Client 模式：创建 Node
		if err := app.initClientNode(); err != nil {
			return fmt.Errorf("初始化 client 节点失败: %w", err)
		}

	case "hybrid":
		// Hybrid 模式：创建 Node + NodeAdapter
		if err := app.initHybridNode(); err != nil {
			return fmt.Errorf("初始化 hybrid 节点失败: %w", err)
		}
		if err := app.initNodeAdapter(); err != nil {
			return fmt.Errorf("初始化 Node API 适配器失败: %w", err)
		}

	default:
		return fmt.Errorf("未知的节点角色: %s", app.role)
	}

	return nil
}

// initMasterNode 初始化 Master 模式的 Node
func (app *App) initMasterNode() error {
	nodeCfg := app.buildNodeConfig()
	nodeCfg.Role = node.NodeRoleMaster
	nodeCfg.Port = app.cfg.Node.MasterRole.Port

	n, err := node.NewNode(nodeCfg)
	if err != nil {
		return err
	}

	app.node = n
	logger.Info("Master 节点已创建")
	return nil
}

// initClientNode 初始化 Client 模式的 Node
func (app *App) initClientNode() error {
	nodeCfg := app.buildNodeConfig()
	nodeCfg.Role = node.NodeRoleClient
	nodeCfg.MasterAddress = app.cfg.Node.ClientRole.MasterAddress

	n, err := node.NewNode(nodeCfg)
	if err != nil {
		return err
	}

	app.node = n
	logger.Info("Client 节点已创建")
	return nil
}

// initHybridNode 初始化 Hybrid 模式的 Node
func (app *App) initHybridNode() error {
	nodeCfg := app.buildNodeConfig()
	nodeCfg.Role = node.NodeRoleHybrid
	nodeCfg.Port = app.cfg.Node.MasterRole.Port
	nodeCfg.MasterAddress = app.cfg.Node.ClientRole.MasterAddress

	n, err := node.NewNode(nodeCfg)
	if err != nil {
		return err
	}

	app.node = n
	logger.Info("Hybrid 节点已创建")
	return nil
}

// generateNodeID 基于设备信息生成稳定的节点ID
// 优先使用 MAC 地址，其次使用主机名，确保每次启动生成相同的 ID
func generateNodeID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	hostname = strings.ToLower(hostname)

	// 获取所有 MAC 地址，按字典序排序确保稳定性
	interfaces, err := net.Interfaces()
	if err == nil {
		var macAddrs []string
		for _, iface := range interfaces {
			// 跳过回环接口和虚拟接口
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			if iface.Flags&net.FlagUp == 0 {
				continue
			}
			hwAddr := iface.HardwareAddr
			if len(hwAddr) > 0 {
				macAddrs = append(macAddrs, hwAddr.String())
			}
		}

		if len(macAddrs) > 0 {
			// 使用第一个可用的 MAC 地址生成 ID
			// 格式: hostname-macshort (例如: myserver-a1b2c3d4)
			mac := macAddrs[0]
			macClean := strings.ReplaceAll(mac, ":", "")
			macShort := macClean
			if len(macClean) > 8 {
				macShort = macClean[:8]
			}
			return fmt.Sprintf("%s-%s", hostname, macShort)
		}
	}

	// 如果没有 MAC 地址可用，使用主机名 + 机器 ID（如果可用）
	machineID := ""
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		machineID = strings.TrimSpace(string(data))
	} else if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		machineID = strings.TrimSpace(string(data))
	}

	if machineID != "" {
		machineShort := machineID
		if len(machineID) > 8 {
			machineShort = machineID[:8]
		}
		return fmt.Sprintf("%s-%s", hostname, machineShort)
	}

	// 最后的回退：使用主机名（不推荐，可能重复）
	return hostname
}

// buildNodeConfig 从应用配置构建 NodeConfig
func (app *App) buildNodeConfig() *node.NodeConfig {
	cfg := app.cfg

	nodeID := cfg.Node.ID
	if nodeID == "auto" || nodeID == "" {
		// 基于设备信息生成稳定的节点 ID
		nodeID = generateNodeID()
	}

	nodeName := cfg.Node.Name
	if nodeName == "" {
		nodeName = nodeID
	}

	// 如果地址是0.0.0.0或空，自动检测最佳本地IP
	address := cfg.Server.Host
	if address == "0.0.0.0" || address == "" {
		address = netutil.GetBestLocalIP()
	}

	// 构建节点能力配置
	capabilities := &node.NodeCapabilities{
		SupportsPython: cfg.Node.Capabilities.PythonEnabled,
	}
	if cfg.Node.Capabilities.PythonEnabled {
		capabilities.PythonVersion = "3.x"
		capabilities.CondaPath = cfg.Node.Capabilities.CondaPath
		capabilities.CondaEnvironments = cfg.Node.Capabilities.CondaEnvironments
	}

	return &node.NodeConfig{
		ID:                nodeID,
		Name:              nodeName,
		Address:           address,
		Port:              cfg.Server.WebPort,
		HeartbeatInterval: time.Duration(cfg.Node.ClientRole.HeartbeatInterval) * time.Second,
		Timeout:           time.Duration(cfg.Node.ClientRole.HeartbeatTimeout) * time.Second,
		MaxRetries:        cfg.Node.ClientRole.RegisterRetry,
		LogLevel:          cfg.Log.Level,
		EnableMetrics:     true,
		DataDir:           "", // 可从配置添加
		TempDir:           "", // 可从配置添加
		Tags:              cfg.Node.Tags,
		Metadata:          cfg.Node.Metadata,
		Capabilities:      capabilities,
	}
}

// initNodeAdapter 初始化 Node API 适配器
func (app *App) initNodeAdapter() error {
	if app.node != nil {
		schedulerCfg := &app.cfg.Master.Scheduler
		app.nodeAdapter = api.NewNodeAdapter(app.node, logger.GetLogger(), schedulerCfg)
		logger.Info("Node API 适配器已创建")
		return nil
	}
	return fmt.Errorf("节点未初始化，无法创建 API 适配器")
}

// registerShutdownHooks 注册优雅关闭钩子
func (app *App) registerShutdownHooks() {
	// 1. 优先级最高：停止接受新连接（HTTP服务器）
	app.shutdownMgr.Register("http-server", func(ctx context.Context) error {
		if app.srv != nil {
			return app.srv.Shutdown(ctx)
		}
		return nil
	}, shutdown.PriorityCritical)

	// 2. 优先级高：停止 Node（统一处理所有角色的 Node）
	if app.node != nil {
		app.shutdownMgr.Register("node", func(ctx context.Context) error {
			return app.node.Stop()
		}, shutdown.PriorityHigh)
	}

	// 3. 优先级高：停止所有模型加载和处理
	if app.modelMgr != nil {
		app.shutdownMgr.Register("models", func(ctx context.Context) error {
			app.modelMgr.Close() //errcheck:ignore
			return nil
		}, shutdown.PriorityHigh)
	}

	// 3.5. 优先级高：关闭存储管理器（在模型之后，进程之前）
	if app.storageMgr != nil {
		app.shutdownMgr.Register("storage", func(ctx context.Context) error {
			if err := app.storageMgr.Close(); err != nil {
				logger.Warnf("关闭存储管理器失败: %v", err)
			}
			return nil
		}, shutdown.PriorityHigh)
	}

	// 4. 优先级中：停止所有进程
	if app.procMgr != nil {
		app.shutdownMgr.Register("processes", func(ctx context.Context) error {
			app.procMgr.StopAll()
			return nil
		}, shutdown.PriorityNormal)
	}

	// 5. 优先级低：关闭日志系统
	app.shutdownMgr.Register("logger", func(ctx context.Context) error {
		logger.Info("日志系统已关闭")
		return nil
	}, shutdown.PriorityLow)
}

// Start 启动应用程序
func (app *App) Start() error {
	// 启动 Node（如果已创建）
	if app.node != nil {
		if err := app.node.Start(); err != nil {
			return fmt.Errorf("启动节点失败: %w", err)
		}
		logger.Info("节点已启动")
	}

	// 启动 HTTP 服务器
	if err := app.srv.Start(); err != nil {
		return fmt.Errorf("无法启动服务器: %w", err)
	}

	// 启动优雅关闭管理器
	app.shutdownMgr.Start()

	// 打印启动信息
	app.printStartupInfo()

	return nil
}

// printStartupInfo 打印启动信息
func (app *App) printStartupInfo() {
	logger.Infof("HTTP 服务器已启动，监听 %s:%d", app.cfg.Server.Host, app.cfg.Server.WebPort)
	fmt.Printf("✓ 节点角色: %s\n", app.role)
	fmt.Printf("✓ HTTP 服务器已启动，监听 %s:%d\n", app.cfg.Server.Host, app.cfg.Server.WebPort)
	fmt.Printf("✓ Web UI: http://localhost:%d\n", app.cfg.Server.WebPort)
	fmt.Printf("✓ OpenAI API: http://localhost:%d/v1\n", app.cfg.Server.WebPort)

	if app.cfg.Compatibility.Ollama.Enabled {
		fmt.Printf("✓ Ollama API: http://localhost:%d\n", app.cfg.Server.OllamaPort)
	}

	if app.role == "master" || app.role == "hybrid" {
		fmt.Printf("✓ Master API: http://localhost:%d/api/master\n", app.cfg.Server.WebPort)
	}

	if app.role == "client" && app.node != nil {
		// 从 Node 获取 Master 地址
		masterAddr := app.cfg.Node.ClientRole.MasterAddress
		if masterAddr != "" {
			fmt.Printf("✓ 连接到 Master: %s\n", masterAddr)
		}
	}

	fmt.Println("\n按 Ctrl+C 停止服务器...")
}

// Wait 等待应用程序关闭
func (app *App) Wait() {
	// 等待关闭信号或上下文取消
	select {
	case <-app.shutdownMgr.Context().Done():
		// Shutdown initiated
	case <-app.shutdownMgr.Done():
		// Shutdown complete
	}

	// 等待所有关闭钩子完成
	app.shutdownMgr.Wait()

	logger.Info("服务器已关闭")
}
