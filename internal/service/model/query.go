package model

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/huggingface"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
)

// GetModel returns a model by ID
func (m *Manager) GetModel(id string) (*Model, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	model, exists := m.models[id]
	if !exists {
		return nil, false
	}

	// Return a copy
	modelCopy := *model
	return &modelCopy, true
}

// ListModels returns all models
func (m *Manager) ListModels() []*Model {
	m.mu.RLock()
	modelCount := len(m.models)
	m.mu.RUnlock()

	if modelCount == 0 {
		m.mu.Lock()
		if !m.scannedOnce {
			m.scannedOnce = true
			m.mu.Unlock()
			logger.Info("ListModels: no models in memory, triggering auto-scan")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			done := make(chan bool, 1)
			go func() {
				if _, err := m.Scan(ctx); err != nil {
					logger.Warnf("ListModels: auto-scan failed: error=%v", err)
				}
				done <- true
			}()

			select {
			case <-done:
				logger.Info("ListModels: auto-scan complete")
			case <-time.After(10 * time.Second):
				logger.Warn("ListModels: auto-scan timed out, returning current model list")
			}
		} else {
			m.mu.Unlock()
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	models := make([]*Model, 0, len(m.models))
	for _, model := range m.models {
		modelCopy := *model
		models = append(models, &modelCopy)
	}

	return models
}

// GetStatus returns the status of a model
func (m *Manager) GetStatus(modelID string) (*ModelStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.statuses[modelID]
	if !exists {
		return nil, false
	}

	// Return a copy (reset sync fields to avoid copylocks)
	statusCopy := *status //nolint:copylocks
	statusCopy.mu = sync.Mutex{}
	statusCopy.tokenMu = sync.Mutex{}
	statusCopy.LoadWait = sync.WaitGroup{}
	statusCopy.InflightWg = sync.WaitGroup{}
	statusCopy.ConcurrencySem = nil
	return &statusCopy, true
}

func (m *Manager) GetStatusRef(modelID string) (*ModelStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.statuses[modelID]
	return status, exists
}

// ListStatus returns all model statuses
func (m *Manager) ListStatus() map[string]*ModelStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]*ModelStatus, len(m.statuses))
	for k, v := range m.statuses {
		statusCopy := *v //nolint:copylocks
		statusCopy.mu = sync.Mutex{}
		statusCopy.tokenMu = sync.Mutex{}
		statusCopy.LoadWait = sync.WaitGroup{}
		statusCopy.InflightWg = sync.WaitGroup{}
		statusCopy.ConcurrencySem = nil
		statuses[k] = &statusCopy
	}

	return statuses
}

// GetScanStatus returns the current scan status
func (m *Manager) GetScanStatus() *ScanStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statusCopy := *m.scanStatus
	return &statusCopy
}

// SetAlias sets the alias for a model
func (m *Manager) SetAlias(modelID, alias string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	model, exists := m.models[modelID]
	if !exists {
		return fmt.Errorf("model not found: %s", modelID)
	}

	model.Alias = alias

	if err := m.updateModelMetadata(modelID, model, func(meta *storage.ModelMetadata) {
		meta.Alias = alias
	}); err != nil {
		logger.Errorf("保存模型别名到数据库失败: modelId=%s, error=%v", modelID, err)
		return err
	}
	logger.Infof("模型别名已保存到数据库: modelId=%s, alias=%s", modelID, alias)

	return nil
}

// SetFavourite sets the favourite flag for a model
func (m *Manager) SetFavourite(modelID string, favourite bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	model, exists := m.models[modelID]
	if !exists {
		return fmt.Errorf("model not found: %s", modelID)
	}

	model.Favourite = favourite

	if err := m.updateModelMetadata(modelID, model, func(meta *storage.ModelMetadata) {
		meta.Favourite = favourite
	}); err != nil {
		logger.Errorf("保存模型收藏状态到数据库失败: modelId=%s, error=%v", modelID, err)
		return err
	}
	logger.Infof("模型收藏状态已保存到数据库: modelId=%s, favourite=%v", modelID, favourite)

	return nil
}

// AutoDetectCapabilities detects model capabilities from GGUF metadata and saves them to the database.
func (m *Manager) AutoDetectCapabilities(modelId string) (*storage.Capabilities, error) {
	model, exists := m.GetModel(modelId)
	if !exists {
		return nil, fmt.Errorf("model not found: %s", modelId)
	}

	if model.Metadata == nil {
		return &storage.Capabilities{}, nil
	}

	detectedCaps := DetectCapabilities(model.Metadata)
	m.saveCapabilities(modelId, detectedCaps)

	return detectedCaps, nil
}

// loadModels loads models from config
func (m *Manager) loadModels() {
	if m.configMgr == nil {
		logger.Warn("loadModels: configMgr is nil, skipping load")
		return
	}

	configModels, err := m.configMgr.LoadModelsConfig()
	if err != nil {
		logger.Errorf("loadModels: failed to load models config: error=%v", err)
		return
	}

	logger.Infof("loadModels: loaded models from config: count=%d", len(configModels))

	// Load aliases and favourites
	aliases, _ := m.configMgr.LoadAliasMap()
	favourites, _ := m.configMgr.LoadFavouriteMap()

	loadedCount := 0
	for _, cfgModel := range configModels {
		// 跳过 mmproj 文件（这些是多模态投影器，应该作为主模型的附件）
		base := filepath.Base(cfgModel.Path)
		if strings.Contains(base, "mmproj") || strings.HasPrefix(base, "mmproj") {
			logger.Infof("loadModels: skipping mmproj file: path=%s", cfgModel.Path)
			continue
		}

		// Try to load the model from disk
		if info, err := os.Stat(cfgModel.Path); err == nil {
			if info.IsDir() {
				// 目录型模型（HuggingFace/safetensors）
				if huggingface.IsHuggingFaceModelDir(cfgModel.Path) {
					model, loadErr := m.loadHuggingFaceModel(cfgModel.Path)
					if loadErr == nil {
						model.ID = cfgModel.ModelID
						if alias, ok := aliases[model.ID]; ok {
							model.Alias = alias
						}
						if fav, ok := favourites[model.ID]; ok {
							model.Favourite = fav
						}
						m.models[model.ID] = model
						loadedCount++
					} else {
						logger.Warnf("loadModels: failed to load HF model: path=%s, error=%v", cfgModel.Path, loadErr)
					}
				} else {
					logger.Warnf("loadModels: directory is not a valid HuggingFace model: path=%s", cfgModel.Path)
				}
			} else {
				model, err := m.loadModel(cfgModel.Path)
				if err == nil {
					model.ID = cfgModel.ModelID
					if alias, ok := aliases[model.ID]; ok {
						model.Alias = alias
					}
					if fav, ok := favourites[model.ID]; ok {
						model.Favourite = fav
					}

					// 加载分卷模型信息（如果配置中有保存）
					if cfgModel.ShardCount > 0 && len(cfgModel.ShardFiles) > 0 {
						model.TotalSize = cfgModel.TotalSize
						model.ShardCount = cfgModel.ShardCount
						model.ShardFiles = cfgModel.ShardFiles
						logger.Infof("loadModels: loaded shard model: name=%s, shardCount=%d, totalSizeGB=%.2f", model.Name, model.ShardCount, float64(model.TotalSize)/(1024*1024*1024))
					}

					// 加载 mmproj 路径（如果配置中有保存）
					if cfgModel.Mmproj != nil && cfgModel.Mmproj.FileName != "" {
						mmprojPath := filepath.Join(filepath.Dir(cfgModel.Path), cfgModel.Mmproj.FileName)
						if info, err := os.Stat(mmprojPath); err == nil {
							model.MmprojPath = mmprojPath
							logger.Infof("loadModels: loaded mmproj file: fileName=%s, sizeGB=%.2f", cfgModel.Mmproj.FileName, float64(info.Size())/(1024*1024*1024))
						} else {
							logger.Warnf("loadModels: mmproj file not found: path=%s", mmprojPath)
						}
					}

					m.models[model.ID] = model
					loadedCount++
				} else {
					logger.Warnf("loadModels: failed to load model: path=%s, error=%v", cfgModel.Path, err)
				}
			}
		} else {
			logger.Warnf("loadModels: model file not found: path=%s", cfgModel.Path)
		}
	}
	logger.Infof("loadModels: successfully loaded models into cache: count=%d", loadedCount)
	// ========== 合并分卷文件 ==========
	// 注意：如果配置中已经保存了分卷信息，这里不需要再次合并
	// 但如果配置中没有分卷信息，则尝试合并
	mergedCount := m.mergeSplitModels()
	if mergedCount > 0 {
		logger.Infof("loadModels: merged shard files: mergedCount=%d", mergedCount)
	}
}

// saveModels saves models to config
func (m *Manager) saveModels() {
	if m.configMgr == nil {
		return
	}

	// Convert models to config entries
	var configModels []config.ModelConfigEntry
	for _, model := range m.models {
		entry := config.ModelConfigEntry{
			ModelID:   model.ID,
			Path:      model.Path,
			Size:      model.Size,
			Alias:     model.Alias,
			Favourite: model.Favourite,
		}

		// 保存分卷模型信息
		if model.ShardCount > 0 {
			entry.TotalSize = model.TotalSize
			entry.ShardCount = model.ShardCount
			entry.ShardFiles = model.ShardFiles
		}

		// Add primary model info if available
		if model.Metadata != nil {
			entry.PrimaryModel = &config.PrimaryModelInfo{
				FileName:        filepath.Base(model.Path),
				Name:            model.Metadata.Name,
				Architecture:    model.Metadata.Architecture,
				ContextLength:   model.Metadata.ContextLength,
				EmbeddingLength: model.Metadata.EmbeddingLength,
			}
		}

		// Add mmproj info if available
		if model.MmprojPath != "" {
			// 获取 mmproj 文件大小
			mmprojSize := int64(0)
			if info, err := os.Stat(model.MmprojPath); err == nil {
				mmprojSize = info.Size()
			}

			entry.Mmproj = &config.MmprojInfo{
				FileName: filepath.Base(model.MmprojPath),
				Size:     mmprojSize,
			}
			// 如果有元数据，也保存
			if model.MmprojMeta != nil {
				entry.Mmproj.Name = model.MmprojMeta.Name
				entry.Mmproj.Architecture = model.MmprojMeta.Architecture
			}
		}

		configModels = append(configModels, entry)
	}

	if err := m.configMgr.SaveModelsConfig(configModels); err != nil {
		logger.Warnf("保存模型配置失败: error=%v", err)
	}
}

// GetModelTokenCounts returns token usage for a model
func (m *Manager) GetModelTokenCounts(modelID string) (prompt, completion int64, found bool) {
	m.mu.RLock()
	status, exists := m.statuses[modelID]
	m.mu.RUnlock()
	if !exists {
		return 0, 0, false
	}
	p, c := status.GetTokenCounts()
	return p, c, true
}


func (m *Manager) GetLoadedModelCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, status := range m.statuses {
		if status.State == StateLoaded {
			count++
		}
	}
	return count
}

// saveCapabilities 将检测到的模型能力保存到数据库
func (m *Manager) saveCapabilities(modelID string, caps *storage.Capabilities) {
	ctx := context.Background()
	existingMeta, err := m.storageMgr.GetStore().GetModelMetadata(ctx, modelID)
	if err == nil && existingMeta != nil {
		existingMeta.Capabilities = caps
		if saveErr := m.storageMgr.GetStore().SaveModelMetadata(ctx, existingMeta); saveErr != nil {
			logger.Warnf("保存模型能力失败: modelId=%s, error=%v", modelID, saveErr)
		}
	} else {
		if saveErr := m.storageMgr.GetStore().SaveModelMetadata(ctx, &storage.ModelMetadata{
			ModelID:      modelID,
			Capabilities: caps,
		}); saveErr != nil {
			logger.Warnf("保存模型能力失败: modelId=%s, error=%v", modelID, saveErr)
		}
	}
}

// updateModelMetadata 获取或创建模型元数据，应用更新函数后保存
func (m *Manager) updateModelMetadata(modelID string, model *Model, updateFn func(*storage.ModelMetadata)) error {
	if m.storageMgr == nil {
		return nil
	}
	store := m.storageMgr.GetStore()

	metadata, err := store.GetModelMetadata(m.ctx, modelID)
	if err != nil {
		metadata = &storage.ModelMetadata{
			ModelID:     modelID,
			StoragePath: filepath.Dir(model.Path),
			Alias:       model.Alias,
			Favourite:   model.Favourite,
			Tags:        model.Tags,
			LoadCount:   model.LoadCount,
		}
		if !model.LastLoaded.IsZero() {
			metadata.LastLoaded = &model.LastLoaded
		}
		metadata.TotalTokens = model.TotalTokens
	}

	updateFn(metadata)

	if err := store.SaveModelMetadata(m.ctx, metadata); err != nil {
		return fmt.Errorf("failed to save model metadata: %w", err)
	}
	return nil
}

// SearchModels 搜索和过滤模型
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

	for _, model := range allModels {
		if model.Metadata != nil && model.Metadata.Architecture != "" {
			result.Architectures[model.Metadata.Architecture]++
		}
		for _, tag := range model.Tags {
			result.Tags[tag]++
		}
	}

	filtered := make([]*Model, 0)
	for _, model := range allModels {
		if m.matchesFilter(model, filter) {
			filtered = append(filtered, model)
		}
	}

	if sort != nil {
		m.sortModels(filtered, sort)
	}

	result.Models = filtered
	result.Filtered = len(filtered)

	return result
}

// matchesFilter 检查模型是否匹配过滤条件
func (m *Manager) matchesFilter(model *Model, filter *ModelFilter) bool {
	if filter == nil {
		return true
	}

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

	if filter.Architecture != "" && model.Metadata != nil {
		if !strings.EqualFold(model.Metadata.Architecture, filter.Architecture) {
			return false
		}
	}

	if filter.MinContext > 0 && model.Metadata != nil {
		if model.Metadata.ContextLength < filter.MinContext {
			return false
		}
	}

	if filter.MaxSize > 0 && model.Size > filter.MaxSize {
		return false
	}

	if filter.LoadedOnly {
		m.mu.RLock()
		status, exists := m.statuses[model.ID]
		m.mu.RUnlock()
		if !exists || status.State != StateLoaded {
			return false
		}
	}

	if filter.Favourites && !model.Favourite {
		return false
	}

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

	if filter.SourceType != "" && model.SourceType != filter.SourceType {
		return false
	}

	if filter.License != "" && !strings.EqualFold(model.License, filter.License) {
		return false
	}

	return true
}

// sortModels 根据排序条件排序模型
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

	for i := 0; i < len(models); i++ {
		for j := i + 1; j < len(models); j++ {
			if !less(i, j) {
				models[i], models[j] = models[j], models[i]
			}
		}
	}
}
