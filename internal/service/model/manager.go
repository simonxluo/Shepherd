package model

import (
	"context"
	"fmt"
	"strings"
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

func (m *Manager) Load(req *LoadRequest) (*LoadResult, error) {
	model, exists := m.GetModel(req.ModelID)
	if !exists {
		logger.Warnf("模型加载失败: 模型不存在: modelId=%s", req.ModelID)
		return nil, fmt.Errorf("model not found: %s", req.ModelID)
	}

	logger.Infof("开始加载模型: modelId=%s, modelName=%s, ctxSize=%d, gpuLayers=%d", req.ModelID, model.Name, req.CtxSize, req.GPULayers)

	var status *ModelStatus
	m.mu.Lock()
	if existingStatus, exists := m.statuses[req.ModelID]; exists {
		if existingStatus.State == StateLoaded {
			m.mu.Unlock()
			logger.Infof("模型已加载，跳过: modelId=%s, port=%d", req.ModelID, existingStatus.Port)
			return &LoadResult{
				Success: true,
				ModelID: req.ModelID,
				Port:    existingStatus.Port,
			}, nil
		}
		if existingStatus.State == StateLoading {
			m.mu.Unlock()
			logger.Warnf("模型加载失败: 模型正在加载中: modelId=%s", req.ModelID)
			return nil, fmt.Errorf("model already loading: %s", req.ModelID)
		}
	}

	status = &ModelStatus{
		ID:   req.ModelID,
		Name: model.Name,
	}
	applyRuntimeConfig(status, req.UnloadAfterMinutes, req.ConcurrencyLimit)
	m.statuses[req.ModelID] = status
	m.mu.Unlock()

	m.swapBeforeLoad(req.ModelID)

	if err := status.transitionTo(StateLoading); err != nil {
		m.mu.Lock()
		delete(m.statuses, req.ModelID)
		m.mu.Unlock()
		logger.Warnf("模型加载失败: 状态转换错误: modelId=%s, error=%v", req.ModelID, err)
		return nil, fmt.Errorf("failed to transition to loading: %w", err)
	}

	startTime := time.Now()

	proc, port, b, err := m.prepareAndStartProcess(req, model, status)
	if err != nil {
		logger.Errorf("模型加载失败: modelId=%s, error=%v", req.ModelID, err)
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   err,
		}, err
	}

	proc.SetOutputHandler(func(line string) {
		if !strings.Contains(line, "update_slots") && !strings.Contains(line, "log_server_r") {
			logger.Debug(fmt.Sprintf("[%s] %s", req.ModelID, line))
		}
	})

	status.transitionTo(StateLoaded)
	m.mu.Lock()
	status.ProcessID = proc.ID
	status.Port = port
	status.LoadedAt = time.Now()
	status.BackendType = b.Type().String()
	m.mu.Unlock()

	duration := time.Since(startTime)
	logger.Infof("模型加载成功: modelId=%s, port=%d, duration=%s, pid=%d, backend=%v", req.ModelID, port, duration.String(), proc.GetPID(), b.Type())

	return &LoadResult{
		Success:  true,
		ModelID:  req.ModelID,
		Port:     port,
		CtxSize:  req.CtxSize,
		Duration: duration,
	}, nil
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
