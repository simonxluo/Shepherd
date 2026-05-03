package cli

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/netutil"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/shutdown"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
	"github.com/shepherd-project/shepherd/Shepherd/internal/server"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/langchain"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/node"
)

var (
	runServerConfig string
	runServerHost   string
	runServerPort   int
)

var runServerCmd = &cobra.Command{
	Use:    "run-server",
	Short:  "内部命令：启动服务器",
	Hidden: true,
	RunE:   runServer,
}

func init() {
	runServerCmd.Flags().StringVar(&runServerConfig, "config", "", "配置文件路径")
	runServerCmd.Flags().StringVar(&runServerHost, "host", "", "监听地址")
	runServerCmd.Flags().IntVar(&runServerPort, "port", 0, "监听端口")
	rootCmd.AddCommand(runServerCmd)
}

type App struct {
	cfg           *config.Config
	configMgr     *config.Manager
	procMgr       *process.Manager
	portAllocator *port.PortAllocator
	storageMgr    *storage.Manager
	modelMgr      *model.Manager
	shutdownMgr   *shutdown.Manager
	srv           *server.Server

	node        *node.Node
	nodeAdapter *handler.NodeAdapter

	langchainMgr     *langchain.Manager
	langchainHandler *langchain.Handler

	role string
}

func runServer(cmd *cobra.Command, args []string) error {
	printBanner()

	app := &App{}

	if err := app.Initialize(runServerConfig); err != nil {
		return fmt.Errorf("初始化失败: %w", err)
	}

	if runServerHost != "" {
		app.cfg.Server.Host = runServerHost
	}
	if runServerPort > 0 {
		app.cfg.Server.WebPort = runServerPort
	}

	if err := app.Start(); err != nil {
		return fmt.Errorf("启动失败: %w", err)
	}

	app.Wait()

	fmt.Println("服务器已关闭")
	return nil
}

func printBanner() {
	fmt.Print(`
 ╔═════════════════════════════════════════════════════╗
 ║   Shepherd - llama.cpp 模型管理系统                   ║
 ║   (C) 2026 Shepherd Project                         ║
 ║   分布式管理 - 更快、更轻、更简单                      ║
 ╚═════════════════════════════════════════════════════╝
`)
	fmt.Printf("版本: %s\n", Version)
	fmt.Printf("Commit: %s\n\n", GitCommit)
}

func (app *App) Initialize(configPath string) error {
	if configPath != "" {
		app.configMgr = config.NewManagerWithPath(configPath)
	} else {
		app.configMgr = config.NewManager()
	}

	cfg, err := app.configMgr.Load()
	if err != nil {
		fmt.Printf("警告: 无法加载配置文件，使用默认配置: %v\n", err)
		cfg = config.DefaultConfig()
	}
	app.cfg = cfg

	app.role = app.determineRole()

	if err := logger.InitLogger(&cfg.Log, app.role); err != nil {
		fmt.Printf("警告: 无法初始化日志系统: %v\n", err)
	}

	logger.Info("Shepherd 正在启动...")
	logger.Infof("版本: %s", Version)
	logger.Infof("节点角色: %s", app.role)
	logger.Infof("配置文件: %s", app.configMgr.GetConfigPath())

	app.procMgr = process.NewManager()

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

	storageCfg := cfg.Storage
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

	app.modelMgr = model.NewManager(cfg, app.configMgr, app.procMgr, app.portAllocator, app.storageMgr)

	app.langchainMgr = langchain.NewManager(app.modelMgr, logger.GetLogger())
	app.langchainHandler = langchain.NewHandler(app.langchainMgr, logger.GetLogger())
	logger.Info("LangChainGo 组件已初始化")

	if err := app.initDistributedComponents(); err != nil {
		return fmt.Errorf("初始化分布式组件失败: %w", err)
	}

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

	app.srv, err = server.NewServer(serverCfg, app.modelMgr)
	if err != nil {
		return fmt.Errorf("无法创建服务器: %w", err)
	}

	if app.langchainHandler != nil {
		app.srv.RegisterLangChainHandler(app.langchainHandler)
		logger.Info("LangChainGo API 已启用")
	}

	if app.role == "master" || app.role == "hybrid" {
		if app.nodeAdapter != nil {
			app.srv.RegisterNodeAdapter(app.nodeAdapter)
		}
	}

	app.srv.SetupRoutes()

	app.shutdownMgr = shutdown.NewManager(10 * time.Second)
	app.registerShutdownHooks()

	return nil
}

func (app *App) determineRole() string {
	role := app.cfg.Node.Role
	if role == "" {
		return "hybrid"
	}
	return role
}

func (app *App) initDistributedComponents() error {
	logger.Infof("初始化分布式组件，角色: %s", app.role)

	switch app.role {
	case "master":
		if err := app.initMasterNode(); err != nil {
			return fmt.Errorf("初始化 master 节点失败: %w", err)
		}
		if err := app.initNodeAdapter(); err != nil {
			return fmt.Errorf("初始化 Node API 适配器失败: %w", err)
		}
	case "client":
		if err := app.initClientNode(); err != nil {
			return fmt.Errorf("初始化 client 节点失败: %w", err)
		}
	case "hybrid":
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

func generateNodeID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	hostname = strings.ToLower(hostname)

	interfaces, err := net.Interfaces()
	if err == nil {
		var macAddrs []string
		for _, iface := range interfaces {
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}
			hwAddr := iface.HardwareAddr
			if len(hwAddr) > 0 {
				macAddrs = append(macAddrs, hwAddr.String())
			}
		}

		if len(macAddrs) > 0 {
			mac := macAddrs[0]
			macClean := strings.ReplaceAll(mac, ":", "")
			macShort := macClean
			if len(macClean) > 8 {
				macShort = macClean[:8]
			}
			return fmt.Sprintf("%s-%s", hostname, macShort)
		}
	}

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

	return hostname
}

func (app *App) buildNodeConfig() *node.NodeConfig {
	cfg := app.cfg

	nodeID := cfg.Node.ID
	if nodeID == "auto" || nodeID == "" {
		nodeID = generateNodeID()
	}

	nodeName := cfg.Node.Name
	if nodeName == "" {
		nodeName = nodeID
	}

	address := cfg.Server.Host
	if address == "0.0.0.0" || address == "" {
		address = netutil.GetBestLocalIP()
	}

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
		Tags:              cfg.Node.Tags,
		Metadata:          cfg.Node.Metadata,
		Capabilities:      capabilities,
	}
}

func (app *App) initNodeAdapter() error {
	if app.node != nil {
		schedulerCfg := &app.cfg.Node.MasterRole.Scheduler
		app.nodeAdapter = handler.NewNodeAdapter(app.node, logger.GetLogger(), schedulerCfg)
		logger.Info("Node API 适配器已创建")
		return nil
	}
	return fmt.Errorf("节点未初始化，无法创建 API 适配器")
}

func (app *App) registerShutdownHooks() {
	app.shutdownMgr.Register("http-server", func(ctx context.Context) error {
		if app.srv != nil {
			return app.srv.Shutdown(ctx)
		}
		return nil
	}, shutdown.PriorityCritical)

	if app.node != nil {
		app.shutdownMgr.Register("node", func(ctx context.Context) error {
			return app.node.Stop()
		}, shutdown.PriorityHigh)
	}

	if app.modelMgr != nil {
		app.shutdownMgr.Register("models", func(ctx context.Context) error {
			app.modelMgr.Close()
			return nil
		}, shutdown.PriorityHigh)
	}

	if app.storageMgr != nil {
		app.shutdownMgr.Register("storage", func(ctx context.Context) error {
			if err := app.storageMgr.Close(); err != nil {
				logger.Warnf("关闭存储管理器失败: %v", err)
			}
			return nil
		}, shutdown.PriorityHigh)
	}

	if app.procMgr != nil {
		app.shutdownMgr.Register("processes", func(ctx context.Context) error {
			app.procMgr.StopAll()
			return nil
		}, shutdown.PriorityNormal)
	}

	app.shutdownMgr.Register("logger", func(ctx context.Context) error {
		logger.Info("日志系统已关闭")
		return nil
	}, shutdown.PriorityLow)
}

func (app *App) Start() error {
	if app.node != nil {
		if err := app.node.Start(); err != nil {
			return fmt.Errorf("启动节点失败: %w", err)
		}
		logger.Info("节点已启动")
	}

	if err := app.srv.Start(); err != nil {
		return fmt.Errorf("无法启动服务器: %w", err)
	}

	app.shutdownMgr.Start()
	app.printStartupInfo()

	return nil
}

func (app *App) printStartupInfo() {
	logger.Infof("HTTP 服务器已启动，监听 %s:%d", app.cfg.Server.Host, app.cfg.Server.WebPort)
	fmt.Printf("  节点角色: %s\n", app.role)
	fmt.Printf("  HTTP 服务器已启动，监听 %s:%d\n", app.cfg.Server.Host, app.cfg.Server.WebPort)
	fmt.Printf("  Web UI: http://localhost:%d\n", app.cfg.Server.WebPort)
	fmt.Printf("  OpenAI API: http://localhost:%d/v1\n", app.cfg.Server.WebPort)

	if app.cfg.Compatibility.Ollama.Enabled {
		fmt.Printf("  Ollama API: http://localhost:%d\n", app.cfg.Server.OllamaPort)
	}

	if app.role == "master" || app.role == "hybrid" {
		fmt.Printf("  Master API: http://localhost:%d/api/master\n", app.cfg.Server.WebPort)
	}

	if app.role == "client" && app.node != nil {
		masterAddr := app.cfg.Node.ClientRole.MasterAddress
		if masterAddr != "" {
			fmt.Printf("  连接到 Master: %s\n", masterAddr)
		}
	}

	fmt.Println("\n按 Ctrl+C 停止服务器...")
}

func (app *App) Wait() {
	select {
	case <-app.shutdownMgr.Context().Done():
	case <-app.shutdownMgr.Done():
	}

	app.shutdownMgr.Wait()
	logger.Info("服务器已关闭")
}
