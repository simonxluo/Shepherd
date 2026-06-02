package model

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/infra/port"
	"github.com/simonxluo/Shepherd/internal/infra/process"
)

// Manager is the central model manager responsible for scanning, loading,
// unloading, and lifecycle management of inference models.
//
// Core responsibilities:
//   - Scan configured paths to discover GGUF/HuggingFace models
//   - Start inference processes via the Backend Registry
//   - Manage runtime state (loading/loaded/unloading/error)
//   - TTL-based auto-unload of idle models
//   - Inflight request tracking and concurrency limiting
//   - Model group swapping (llama-swap style)
//
// Concurrency: all public methods are thread-safe, protected by mu (RWMutex).
// The atomic version counter is incremented on every models/statuses mutation
// and can be used externally for cache invalidation.
type Manager struct {
	config        *config.Config
	configMgr     *config.Manager
	processMgr    *process.Manager
	portAllocator *port.PortAllocator
	storageMgr    *storage.Manager // database storage manager

	// Backend registry for multi-backend support
	backendRegistry *backend.Registry

	models         map[string]*Model
	statuses       map[string]*ModelStatus
	instances      map[string]*RuntimeInstance
	modelInstances map[string][]string
	scanStatus     *ScanStatus
	groups         map[string]*ModelGroup

	// version is incremented on every models/statuses map mutation.
	// Used by ModelLookupIndex to avoid full rebuilds on every request.
	version atomic.Int64

	mu          sync.RWMutex
	scannedOnce bool
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// ScanStatus represents the progress of an ongoing model scan operation.
type ScanStatus struct {
	Scanning    bool
	Progress    float64
	CurrentPath string
	StartedAt   time.Time
	Errors      []ScanError
}

// NewManager creates and initializes a model manager.
// Initialization sequence: create backend registry -> load saved models from config -> start TTL checker.
func NewManager(cfg *config.Config, cfgMgr *config.Manager, procMgr *process.Manager, portAllocator *port.PortAllocator, storageMgr *storage.Manager) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize backend registry (plugins self-register via init())
	backendRegistry := backend.Default()
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
		instances:       make(map[string]*RuntimeInstance),
		modelInstances:  make(map[string][]string),
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

	// Load model groups from config (llama-swap style swap/exclusive groups)
	m.loadGroupsFromConfig()

	// Load saved models
	m.loadModels()
	logger.Infof("ModelManager: 从配置加载模型完成: modelCount=%d", len(m.models))

	m.StartTTLChecker()

	return m
}

// Close shuts down the manager, cancels all background goroutines and waits for them to exit.
func (m *Manager) Close() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

// SyncBackendRegistry re-syncs backend configurations from the app config.
// Call this after config changes (e.g., model_bind_host update) to ensure
// new model loads use the updated settings.
func (m *Manager) SyncBackendRegistry(cfg *config.Config) {
	m.backendRegistry.SyncFromConfig(cfg)
}

// Version returns the current models version counter.
// This is incremented whenever models or statuses are mutated.
func (m *Manager) Version() int64 {
	return m.version.Load()
}

// bumpVersion increments the version counter. Must be called under m.mu lock
// or at points where models/statuses maps are mutated.
func (m *Manager) bumpVersion() {
	m.version.Add(1)
}

// GetProcessManager returns the process manager instance.
func (m *Manager) GetProcessManager() *process.Manager {
	return m.processMgr
}

// GetModelCapabilities retrieves model capabilities from storage.
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

// GetBackendForModel returns the plugin for a loaded model based on its backend type.
func (m *Manager) GetBackendForModel(modelID string) backend.Plugin {
	m.mu.RLock()
	status, exists := m.statuses[modelID]
	m.mu.RUnlock()
	if !exists || status.BackendType == "" {
		return nil
	}
	p, ok := m.backendRegistry.Get(backend.ID(status.BackendType))
	if !ok {
		return nil
	}
	return p
}
