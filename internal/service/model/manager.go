package model

import (
	"context"
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


// SearchModels searches and filters models based on criteria
// 如果内存中没有模型，会自动触发一次扫描
func (m *Manager) SearchModels(filter *ModelFilter, sort *ModelSort) *ModelSearchResult {
	m.mu.RLock()
	modelCount := len(m.models)
	m.mu.RUnlock()

	// 如果内存中没有模型，自动触发一次扫描
	if modelCount == 0 {
		logger.Info("SearchModels: 内存中没有模型，触发自动扫描")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		done := make(chan bool, 1)
		go func() {
			if _, err := m.Scan(ctx); err != nil {
				logger.Warnf("SearchModels: 自动扫描失败: %v", err)
			}
			done <- true
		}()

		select {
		case <-done:
			logger.Info("SearchModels: 自动扫描完成")
		case <-time.After(10 * time.Second):
			logger.Warn("SearchModels: 自动扫描超时")
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
