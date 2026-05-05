package model

import (
	"context"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/model/backend"
)

// Manager manages model scanning and loading
type Manager struct {
	config        *config.Config
	configMgr     *config.Manager
	processMgr    *process.Manager
	portAllocator *port.PortAllocator
	storageMgr    *storage.Manager // 数据库存储管理器

	// Backend registry for multi-backend support
	backendRegistry *backend.Registry

	models     map[string]*Model
	statuses   map[string]*ModelStatus
	scanStatus *ScanStatus
	groups     map[string]*ModelGroup

	mu          sync.RWMutex
	scannedOnce bool
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// ScanStatus represents the current scan status
type ScanStatus struct {
	Scanning    bool
	Progress    float64
	CurrentPath string
	StartedAt   time.Time
	Errors      []ScanError
}

// NewManager creates a new model manager
func NewManager(cfg *config.Config, cfgMgr *config.Manager, procMgr *process.Manager, portAllocator *port.PortAllocator, storageMgr *storage.Manager) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize backend registry
	backendRegistry := backend.NewRegistry()
	backendRegistry.SyncFromConfig(cfg)

	m := &Manager{
		config:          cfg,
		configMgr:       cfgMgr,
		processMgr:      procMgr,
		portAllocator:   portAllocator,
		storageMgr:      storageMgr,
		backendRegistry: backendRegistry,
		models:          make(map[string]*Model),
		statuses:        make(map[string]*ModelStatus),
		scanStatus:      &ScanStatus{},
		groups:          make(map[string]*ModelGroup),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Log initialization info
	paths := m.getScanPaths()
	if len(paths) == 0 {
		logger.Warn("ModelManager: 未配置模型扫描路径")
	} else {
		logger.Infof("ModelManager: 初始化完成: paths=%v", paths)
	}

	// Load saved models
	m.loadModels()
	logger.Infof("ModelManager: 从配置加载模型完成: modelCount=%d", len(m.models))

	m.StartTTLChecker()

	return m
}

// Close closes the manager
func (m *Manager) Close() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

// GetProcessManager returns the process manager
func (m *Manager) GetProcessManager() *process.Manager {
	return m.processMgr
}

// GetModelCapabilities 从存储获取模型能力信息
func (m *Manager) GetModelCapabilities(modelID string) *storage.Capabilities {
	if m.storageMgr == nil {
		return nil
	}
	ctx := context.Background()
	meta, err := m.storageMgr.GetStore().GetModelMetadata(ctx, modelID)
	if err != nil {
		return nil
	}
	return meta.Capabilities
}

// GetBackendForModel 根据已加载模型的后端类型获取后端实例
func (m *Manager) GetBackendForModel(modelID string) backend.Backend {
	m.mu.RLock()
	status, exists := m.statuses[modelID]
	m.mu.RUnlock()
	if !exists || status.BackendType == "" {
		return nil
	}
	bt, err := backend.ParseBackendType(status.BackendType)
	if err != nil {
		return nil
	}
	b, _, err := m.backendRegistry.Resolve("", bt)
	if err != nil {
		return nil
	}
	return b
}
