package model

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/infra/huggingface"
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

// ensureScanned triggers a one-time background scan if no models have been loaded yet.
// It blocks up to 10s for the scan to complete, then returns regardless.
func (m *Manager) ensureScanned() {
	m.mu.RLock()
	modelCount := len(m.models)
	m.mu.RUnlock()

	if modelCount > 0 {
		return
	}

	m.mu.Lock()
	if m.scannedOnce {
		m.mu.Unlock()
		return
	}
	m.scannedOnce = true
	m.mu.Unlock()

	logger.Info("ensureScanned: no models in memory, triggering auto-scan")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	done := make(chan bool, 1)
	go func() {
		if _, err := m.Scan(ctx); err != nil {
			logger.Warnf("ensureScanned: auto-scan failed: error=%v", err)
		}
		done <- true
	}()

	select {
	case <-done:
		logger.Info("ensureScanned: auto-scan complete")
	case <-time.After(10 * time.Second):
		logger.Warn("ensureScanned: auto-scan timed out, returning current model list")
	}
}

// ListModels returns all models sorted by name (alphabetical order).
func (m *Manager) ListModels() []*Model {
	m.ensureScanned()

	m.mu.RLock()
	defer m.mu.RUnlock()

	models := make([]*Model, 0, len(m.models))
	for _, model := range m.models {
		modelCopy := *model
		models = append(models, &modelCopy)
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})

	return models
}

// GetStatus returns a copy of the specified model's status (safe copy without sync primitives).
func (m *Manager) GetStatus(modelID string) (*ModelStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.statuses[modelID]
	if !exists {
		return nil, false
	}

	return copyModelStatus(status), true
}

// GetStatusRef returns a direct reference to the model status (not a copy).
// Callers must handle synchronization carefully.
func (m *Manager) GetStatusRef(modelID string) (*ModelStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.statuses[modelID]
	return status, exists
}

// ListStatus returns copies of all model statuses.
func (m *Manager) ListStatus() map[string]*ModelStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]*ModelStatus, len(m.statuses))
	for k, v := range m.statuses {
		statuses[k] = copyModelStatus(v)
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
	m.bumpVersion()

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
	// 合并分卷文件
	// 注意：如果配置中已经保存了分卷信息，这里不需要再次合并
	// 但如果配置中没有分卷信息，则尝试合并
	mergedCount := m.mergeSplitModels()
	if mergedCount > 0 {
		logger.Infof("loadModels: merged shard files: mergedCount=%d", mergedCount)
	}
	m.bumpVersion()
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

// GetLoadedModelCount returns the number of models currently in StateLoaded.
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

// saveCapabilities persists detected model capabilities to the database.
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

// updateModelMetadata retrieves or creates model metadata, applies the update function, and saves it.
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

// copyModelStatus creates a shallow copy of ModelStatus with sync primitives reset to zero values.
// This avoids the copylocks issue when returning status snapshots to callers.
func copyModelStatus(s *ModelStatus) *ModelStatus {
	s.mu.Lock()
	s.tokenMu.Lock()
	cp := &ModelStatus{
		ID:                    s.ID,
		InstanceID:            s.InstanceID,
		Name:                  s.Name,
		State:                 s.State,
		ProcessID:             s.ProcessID,
		Port:                  s.Port,
		CtxSize:               s.CtxSize,
		LoadedAt:              s.LoadedAt,
		BackendType:           s.BackendType,
		Error:                 s.Error,
		LastRequestTime:       s.LastRequestTime,
		ConcurrencyLimit:      s.ConcurrencyLimit,
		UnloadAfter:           s.UnloadAfter,
		TotalPromptTokens:     s.TotalPromptTokens,
		TotalCompletionTokens: s.TotalCompletionTokens,
		RequestCount:          s.RequestCount,
		ErrorCount:            s.ErrorCount,
		TotalLatencyMs:        s.TotalLatencyMs,
		FirstRequestAt:        s.FirstRequestAt,
		LastRequestAt:         s.LastRequestAt,
	}
	s.tokenMu.Unlock()
	s.mu.Unlock()
	return cp
}
