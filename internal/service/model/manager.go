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

// isGGUFFile checks if a file is specifically a GGUF model file (deprecated, use isModelFile)
// 保留此方法以兼容性，内部调用 isModelFile
func (m *Manager) isGGUFFile(path string) bool {
	return m.isModelFile(path)
}

// Load loads a model
func (m *Manager) Load(req *LoadRequest) (*LoadResult, error) {
	// Get model
	model, exists := m.GetModel(req.ModelID)
	if !exists {
		logger.Warnf("模型加载失败: 模型不存在: modelId=%s", req.ModelID)
		return nil, fmt.Errorf("model not found: %s", req.ModelID)
	}

	logger.Infof("开始加载模型: modelId=%s, modelName=%s, ctxSize=%d, gpuLayers=%s", req.ModelID, model.Name, req.CtxSize, req.GPULayers)

	// Phase 1: 检查状态并创建初始 status（加锁）
	var status *ModelStatus
	m.mu.Lock()
	if existingStatus, exists := m.statuses[req.ModelID]; exists && existingStatus.State == StateLoading {
		m.mu.Unlock()
		logger.Warnf("模型加载失败: 模型正在加载中: modelId=%s", req.ModelID)
		return nil, fmt.Errorf("model already loading: %s", req.ModelID)
	}

	// Create initial status
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

	// Phase 2: 准备工作（无锁）- 使用后端接口
	// Determine model path
	modelPath := model.Path
	if len(model.ShardFiles) > 0 {
		modelPath = model.ShardFiles[0]
		fmt.Printf("[INFO] 使用分卷模型主文件: %s (共 %d 个分卷)\n", modelPath, len(model.ShardFiles))
	}

	// Resolve backend
	bt, parseErr := backend.ParseBackendType(req.BackendType)
	if parseErr != nil {
		status.transitionTo(StateError)
		status.Error = parseErr
		logger.Errorf("模型加载失败: 无效的后端类型: modelId=%s, error=%v", req.ModelID, parseErr)
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   status.Error,
		}, status.Error
	}
	b, bcfg, resolveErr := m.backendRegistry.Resolve(modelPath, bt)
	if resolveErr != nil {
		status.transitionTo(StateError)
		status.Error = resolveErr
		logger.Errorf("模型加载失败: 无法解析后端: modelId=%s, error=%v", req.ModelID, resolveErr)
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   status.Error,
		}, status.Error
	}

	// Discover backend
	info, discoverErr := b.Discover(bcfg)
	if discoverErr != nil {
		status.transitionTo(StateError)
		status.Error = discoverErr
		logger.Errorf("模型加载失败: 后端发现失败: modelId=%s, backend=%v, error=%v", req.ModelID, b.Type(), discoverErr)
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   status.Error,
		}, status.Error
	}
	if !info.Available {
		err := fmt.Errorf("backend %s is not available", b.Type())
		status.transitionTo(StateError)
		status.Error = err
		logger.Errorf("模型加载失败: 后端不可用: modelId=%s, backend=%v", req.ModelID, b.Type())
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   status.Error,
		}, status.Error
	}

	// Allocate port using centralized PortAllocator
	port, err := m.portAllocator.NextPort()
	if err != nil {
		status.transitionTo(StateError)
		status.Error = fmt.Errorf("no available ports: %w", err)
		logger.Errorf("模型加载失败: 无可用端口: modelId=%s, error=%v", req.ModelID, err)
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   status.Error,
		}, status.Error
	}

	// Build command via backend interface
	backendReq := m.toBackendLoadRequest(req, modelPath, port)
	startCfg, cmdErr := b.BuildStartConfig(info, backendReq)
	if cmdErr != nil {
		m.portAllocator.Release(port)
		status.transitionTo(StateError)
		status.Error = cmdErr
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   cmdErr,
		}, cmdErr
	}

	// Start process
	proc, err := m.processMgr.Start(req.ModelID, model.Name, startCfg.Command, startCfg.BinPath)
	if err != nil {
		m.portAllocator.Release(port)
		status.transitionTo(StateError)
		status.Error = err
		logger.Errorf("模型加载失败: 启动进程失败: modelId=%s, error=%v", req.ModelID, err)
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   err,
		}, err
	}

	// 设置输出处理器转发日志
	proc.SetOutputHandler(func(line string) {
		// 过滤掉过于频繁的日志
		if !strings.Contains(line, "update_slots") && !strings.Contains(line, "log_server_r") {
			logger.Debug(fmt.Sprintf("[%s] %s", req.ModelID, line))
		}
	})

	status.transitionTo(StateLoaded)
	m.mu.Lock()
	status.ProcessID = proc.ID
	status.Port = port
	status.LoadedAt = time.Now()
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

// isLoading 检查模型是否正在加载
func (m *Manager) isLoading(modelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if status, exists := m.statuses[modelID]; exists {
		return status.State == StateLoading
	}
	return false
}

// findAvailablePort finds an available port for the model server
func (m *Manager) findAvailablePort() int {
	// Start from base port and find available
	basePort := 8081

	statuses := m.ListStatus()
	usedPorts := make(map[int]bool)
	for _, status := range statuses {
		if status.Port > 0 {
			usedPorts[status.Port] = true
		}
	}

	for port := basePort; port < basePort+100; port++ {
		if !usedPorts[port] {
			return port
		}
	}

	return basePort
}

// SearchModels searches and filters models based on criteria
// 如果内存中没有模型，会自动触发一次扫描
func (m *Manager) SearchModels(filter *ModelFilter, sort *ModelSort) *ModelSearchResult {
	m.mu.RLock()
	modelCount := len(m.models)
	m.mu.RUnlock()

	// 如果内存中没有模型，自动触发一次扫描
	if modelCount == 0 {
		fmt.Printf("[INFO] SearchModels: 内存中没有模型，触发自动扫描\n")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		done := make(chan bool, 1)
		go func() {
			if _, err := m.Scan(ctx); err != nil {
				fmt.Printf("[WARN] SearchModels: 自动扫描失败: %v\n", err)
			}
			done <- true
		}()

		select {
		case <-done:
			fmt.Printf("[INFO] SearchModels: 自动扫描完成\n")
		case <-time.After(10 * time.Second):
			fmt.Printf("[WARN] SearchModels: 自动扫描超时\n")
		}
	}

	m.mu.RLock()
	allModels := make([]*Model, 0, len(m.models))
	for _, model := range m.models {
		modelCopy := *model
		allModels = append(allModels, &modelCopy)
	}
	m.mu.RUnlock()

	result := &ModelSearchResult{
		Models:        []*Model{},
		Total:         len(allModels),
		Tags:          make(map[string]int),
		Architectures: make(map[string]int),
	}

	// Collect statistics
	for _, model := range allModels {
		if model.Metadata != nil && model.Metadata.Architecture != "" {
			result.Architectures[model.Metadata.Architecture]++
		}
		for _, tag := range model.Tags {
			result.Tags[tag]++
		}
	}

	// Apply filters
	filtered := make([]*Model, 0)
	for _, model := range allModels {
		if m.matchesFilter(model, filter) {
			filtered = append(filtered, model)
		}
	}

	// Apply sorting
	if sort != nil {
		m.sortModels(filtered, sort)
	}

	result.Models = filtered
	result.Filtered = len(filtered)

	return result
}

// matchesFilter checks if a model matches the filter criteria
func (m *Manager) matchesFilter(model *Model, filter *ModelFilter) bool {
	if filter == nil {
		return true
	}

	// Tags filter
	if len(filter.Tags) > 0 {
		hasTag := false
		for _, tag := range filter.Tags {
			for _, modelTag := range model.Tags {
				if strings.EqualFold(tag, modelTag) {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	// Architecture filter
	if filter.Architecture != "" && model.Metadata != nil {
		if !strings.EqualFold(model.Metadata.Architecture, filter.Architecture) {
			return false
		}
	}

	// Min context filter
	if filter.MinContext > 0 && model.Metadata != nil {
		if model.Metadata.ContextLength < filter.MinContext {
			return false
		}
	}

	// Max size filter
	if filter.MaxSize > 0 && model.Size > filter.MaxSize {
		return false
	}

	// Loaded only filter
	if filter.LoadedOnly {
		m.mu.RLock()
		status, exists := m.statuses[model.ID]
		m.mu.RUnlock()
		if !exists || status.State != StateLoaded {
			return false
		}
	}

	// Favourites filter
	if filter.Favourites && !model.Favourite {
		return false
	}

	// Search query
	if filter.SearchQuery != "" {
		query := strings.ToLower(filter.SearchQuery)
		match := false
		if strings.Contains(strings.ToLower(model.Name), query) {
			match = true
		}
		if strings.Contains(strings.ToLower(model.Alias), query) {
			match = true
		}
		if strings.Contains(strings.ToLower(model.Description), query) {
			match = true
		}
		if model.Metadata != nil {
			if strings.Contains(strings.ToLower(model.Metadata.Architecture), query) {
				match = true
			}
		}
		if !match {
			return false
		}
	}

	// Source type filter
	if filter.SourceType != "" && model.SourceType != filter.SourceType {
		return false
	}

	// License filter
	if filter.License != "" && !strings.EqualFold(model.License, filter.License) {
		return false
	}

	return true
}

// sortModels sorts models based on sort criteria
func (m *Manager) sortModels(models []*Model, sort *ModelSort) {
	if sort == nil || sort.Field == "" {
		return
	}

	less := func(i, j int) bool {
		switch sort.Field {
		case "name":
			if sort.Direction == "desc" {
				return models[i].Name > models[j].Name
			}
			return models[i].Name < models[j].Name
		case "size":
			if sort.Direction == "desc" {
				return models[i].Size > models[j].Size
			}
			return models[i].Size < models[j].Size
		case "scanned_at":
			if sort.Direction == "desc" {
				return models[i].ScannedAt.After(models[j].ScannedAt)
			}
			return models[i].ScannedAt.Before(models[j].ScannedAt)
		case "load_count":
			if sort.Direction == "desc" {
				return models[i].LoadCount > models[j].LoadCount
			}
			return models[i].LoadCount < models[j].LoadCount
		default:
			return models[i].Name < models[j].Name
		}
	}

	// Simple bubble sort for demonstration
	for i := 0; i < len(models); i++ {
		for j := i + 1; j < len(models); j++ {
			if !less(i, j) {
				models[i], models[j] = models[j], models[i]
			}
		}
	}
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
