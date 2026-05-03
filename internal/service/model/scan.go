package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/gguf"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/huggingface"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
)

var (
	reHuggingFacePath = regexp.MustCompile(`models--.+--.+\.gguf$`)
	reSplitGGUF       = regexp.MustCompile(`^(.*?)-(\d{5})-of-(\d{5})\.gguf$`)
	reSplitSuffix     = regexp.MustCompile(`-\d{5}-of-\d{5}$`)
)

// Scan scans for models in configured paths
func (m *Manager) Scan(ctx context.Context) (*ScanResult, error) {
	m.mu.Lock()
	if m.scanStatus.Scanning {
		m.mu.Unlock()
		return nil, fmt.Errorf("scan already in progress")
	}
	m.scanStatus.Scanning = true
	m.scanStatus.StartedAt = time.Now()
	m.scanStatus.Errors = nil
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.scanStatus.Scanning = false
		m.mu.Unlock()
	}()

	result := &ScanResult{
		Models:    []*Model{},
		Errors:    []ScanError{},
		ScannedAt: time.Now(),
	}

	scanPaths := m.getScanPaths()
	logger.Infof("开始扫描模型: pathCount=%d", len(scanPaths))

	// Scan each configured path
	for _, scanPath := range scanPaths {
		logger.Infof("正在扫描路径: %s", scanPath)
		pathModels, pathErrors := m.scanPath(ctx, scanPath)
		logger.Infof("路径扫描完成: path=%s, modelCount=%d, errorCount=%d", scanPath, len(pathModels), len(pathErrors))
		result.Models = append(result.Models, pathModels...)
		result.Errors = append(result.Errors, pathErrors...)
		result.TotalFiles += len(pathModels) + len(pathErrors)
		result.MatchedFiles += len(pathModels)
	}

	result.Duration = time.Since(result.ScannedAt)
	logger.Infof("模型扫描完成: totalModels=%d, duration=%s, totalErrors=%d", len(result.Models), result.Duration.String(), len(result.Errors))

	// ========== 修复：从数据库加载用户设置的属性 ==========
	// 从数据库加载所有模型的元数据（别名、收藏、标签、描述、使用统计等）
	modelMetadataMap := make(map[string]*storage.ModelMetadata)
	if m.storageMgr != nil {
		store := m.storageMgr.GetStore()
		metadata, err := store.GetAllModelMetadata(ctx)
		if err != nil {
			logger.Warnf("从数据库加载模型元数据失败: %v", err)
		} else {
			modelMetadataMap = metadata
			logger.Infof("从数据库加载模型元数据: count=%d", len(metadata))
		}
	}
	// ==============================================

	// Update models map（合并扫描结果，保留扫描路径之外的模型）
	m.mu.Lock()

	// 构建已扫描模型 ID 集合
	scannedIDs := make(map[string]bool, len(result.Models))
	for _, model := range result.Models {
		m.models[model.ID] = model
		scannedIDs[model.ID] = true
	}

	// 移除在扫描路径下但未被重新发现的模型（已从磁盘删除）
	cleanScanPaths := make([]string, len(scanPaths))
	for i, sp := range scanPaths {
		cleanScanPaths[i] = filepath.Clean(sp)
	}
	for id, existingModel := range m.models {
		if scannedIDs[id] {
			continue
		}
		modelPath := filepath.Clean(existingModel.Path)
		for _, csp := range cleanScanPaths {
			if strings.HasPrefix(modelPath, csp) {
				delete(m.models, id)
				break
			}
		}
	}

	// ========== 修复：恢复用户设置的属性 ==========
	// 将数据库中的用户属性合并回新扫描的模型中
	for id, model := range m.models {
		if metadata, exists := modelMetadataMap[id]; exists {
			// 恢复用户设置的属性
			model.Alias = metadata.Alias
			model.Favourite = metadata.Favourite
			model.Tags = metadata.Tags
			model.Description = metadata.Description
			model.LoadCount = metadata.LoadCount
			if metadata.LastLoaded != nil {
				model.LastLoaded = *metadata.LastLoaded
			}
			model.TotalTokens = metadata.TotalTokens
			logger.Debugf("恢复模型用户属性: id=%s, alias=%s, favourite=%v", id, metadata.Alias, metadata.Favourite)
		}
	}
	// ==============================================

	// ========== 新增：合并分卷文件 ==========
	mergedCount := m.mergeSplitModels()
	if mergedCount > 0 {
		logger.Infof("已合并分卷文件: mergedCount=%d", mergedCount)
	}

	modelCount := len(m.models)
	m.mu.Unlock()
	logger.Infof("模型缓存已更新: modelCount=%d", modelCount)

	// Save to config
	m.saveModels()
	logger.Infof("已保存模型到配置: savedCount=%d", len(m.models))

	// 更新 result.Models 为合并后的模型列表
	// 这样 Scan API 返回的是合并后的结果
	m.mu.RLock()
	result.Models = make([]*Model, 0, len(m.models))
	for _, model := range m.models {
		modelCopy := *model
		result.Models = append(result.Models, &modelCopy)
	}
	m.mu.RUnlock()

	return result, nil
}

// scanPath scans a single path for models with enhanced robustness
func (m *Manager) scanPath(ctx context.Context, scanPath string) ([]*Model, []ScanError) {
	var models []*Model
	var errors []ScanError
	var mu sync.Mutex
	var fileCount int
	var matchedCount int

	// Update scan status
	m.mu.Lock()
	m.scanStatus.CurrentPath = scanPath
	m.mu.Unlock()

	// Check if path exists
	info, err := os.Stat(scanPath)
	if err != nil {
		logger.Errorf("路径访问失败: %s, error: %v", scanPath, err)
		return nil, []ScanError{{Path: scanPath, Error: fmt.Sprintf("路径访问失败: %v", err)}}
	}

	pathType := "文件"
	if info.IsDir() {
		pathType = "目录"
	}
	logger.Debugf("开始扫描路径: %s, type=%s", scanPath, pathType)

	// Check if path is readable
	if info.IsDir() {
		// Test read permission by trying to open the directory
		f, err := os.Open(scanPath)
		if err != nil {
			return nil, []ScanError{{Path: scanPath, Error: fmt.Sprintf("目录读取失败: %v", err)}}
		}
		utils.CloseQuietly(f)
	}

	// Use concurrent processing for directories
	if info.IsDir() {
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 10)

		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil {
				mu.Lock()
				errors = append(errors, ScanError{
					Path:  path,
					Error: fmt.Sprintf("文件访问错误: %v", err),
				})
				mu.Unlock()
				return nil
			}

			if info.IsDir() {
				if path != scanPath && huggingface.IsHuggingFaceModelDir(path) {
					fileCount++
					matchedCount++
					logger.Debugf("找到 HuggingFace 模型目录: %s", path)

					wg.Add(1)
					semaphore <- struct{}{}

					go func(dirPath string) {
						defer wg.Done()
						defer func() { <-semaphore }()

						model, loadErr := m.loadHuggingFaceModel(dirPath)
						if loadErr != nil {
							logger.Warnf("加载 HuggingFace 模型失败: %s, error: %v", dirPath, loadErr)
							mu.Lock()
							errors = append(errors, ScanError{
								Path:  dirPath,
								Error: loadErr.Error(),
							})
							mu.Unlock()
						} else {
							logger.Infof("成功加载 HuggingFace 模型: %s, id=%s, path=%s", model.Name, model.ID, dirPath)
							mu.Lock()
							models = append(models, model)
							mu.Unlock()
						}
					}(path)

					return filepath.SkipDir
				}
				return nil
			}

			fileCount++

			// 检查是否为模型文件
			if m.isModelFile(path) {
				matchedCount++
				logger.Debugf("找到模型文件: %s", path)

				wg.Add(1)
				semaphore <- struct{}{}

				go func(filePath string) {
					defer wg.Done()
					defer func() { <-semaphore }()

					model, loadErr := m.loadModelWithValidation(filePath)
					if loadErr != nil {
						logger.Warnf("加载模型失败: %s, error: %v", filePath, loadErr)
						mu.Lock()
						errors = append(errors, ScanError{
							Path:  filePath,
							Error: loadErr.Error(),
						})
						mu.Unlock()
					} else {
						logger.Infof("成功加载模型: %s, id=%s, path=%s", model.Name, model.ID, filePath)
						mu.Lock()
						models = append(models, model)
						mu.Unlock()
					}
				}(path)
			}

			return nil
		})

		wg.Wait()

		logger.Debugf("路径扫描完成: path=%s, fileCount=%d, matchedCount=%d", scanPath, fileCount, matchedCount)

		if err != nil && err != ctx.Err() {
			errors = append(errors, ScanError{
				Path:  scanPath,
				Error: fmt.Sprintf("扫描中断: %v", err),
			})
		}
	} else if huggingface.IsHuggingFaceModelDir(scanPath) {
		logger.Infof("HuggingFace 模型目录: %s", scanPath)
		model, loadErr := m.loadHuggingFaceModel(scanPath)
		if loadErr != nil {
			errors = append(errors, ScanError{
				Path:  scanPath,
				Error: loadErr.Error(),
			})
		} else {
			models = append(models, model)
		}
	} else if m.isModelFile(scanPath) {
		logger.Infof("单文件模型: %s", scanPath)
		model, err := m.loadModelWithValidation(scanPath)
		if err != nil {
			errors = append(errors, ScanError{
				Path:  scanPath,
				Error: err.Error(),
			})
		} else {
			models = append(models, model)
		}
	} else {
		logger.Warnf("路径不是模型文件: %s", scanPath)
	}

	return models, errors
}

// isModelFile checks if a file is a supported model file (GGUF, SafeTensors, etc.)
func (m *Manager) isModelFile(path string) bool {
	base := filepath.Base(path)

	// 排除 mmproj 文件（这些是多模态投影器，应该作为主模型的附件，而非独立模型）
	// mmproj 文件命名模式: mmproj.gguf, mmproj-f16.gguf, mmproj-F32.gguf, xxx-mmproj.gguf
	if strings.Contains(base, "mmproj") || strings.HasPrefix(base, "mmproj") {
		return false
	}

	// 支持的模型格式
	// GGUF 格式 (主要支持，可被 llama.cpp 加载)
	patterns := []string{
		".gguf", // GGUF 格式
		".GGUF",
		"gguf-", // 分卷 GGUF
	}

	// 检查文件扩展名
	for _, pattern := range patterns {
		if strings.Contains(base, pattern) {
			return true
		}
	}

	// HuggingFace 缓存目录模式检查
	// 模式1: models--org--model/snapshots/hash/*.gguf
	if matched := reHuggingFacePath.MatchString(path); matched {
		return true
	}

	return false
}

// loadModel loads a model from a file path
func (m *Manager) loadModel(path string) (*Model, error) {
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Read GGUF metadata
	metadata, err := gguf.ReadMetadata(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	// Generate model ID
	modelID := m.generateModelID(path, metadata)

	// Calculate path prefix for duplicate identification
	pathPrefix := m.calculatePathPrefix(path)

	// Get model name
	modelName := metadata.Name
	if modelName == "" {
		modelName = filepath.Base(path)
		modelName = strings.TrimSuffix(modelName, ".gguf")
		modelName = strings.TrimSuffix(modelName, ".GGUF")
	}

	// Create display name with path prefix for duplicates
	displayName := modelName
	if pathPrefix != "" && pathPrefix != "models" {
		displayName = fmt.Sprintf("[%s]%s", pathPrefix, modelName)
	}

	// Create model
	model := &Model{
		ID:          modelID,
		Name:        modelName,
		DisplayName: displayName,
		Path:        path,
		PathPrefix:  pathPrefix,
		Size:        info.Size(),
		Metadata:    metadata,
		ScannedAt:   time.Now(),
		SourcePath:  filepath.Dir(path),
	}

	// Check for mmproj
	mmprojPath := m.findMmproj(path)
	if mmprojPath != "" {
		mmprojMeta, err := gguf.ReadMetadata(mmprojPath)
		if err == nil {
			model.MmprojPath = mmprojPath
			model.MmprojMeta = mmprojMeta
		}
	}

	// Auto-detect and save capabilities
	if model.Metadata != nil {
		detectedCaps := DetectCapabilities(model.Metadata)

		// Save capabilities to database
		ctx := context.Background()
		existingMeta, err := m.storageMgr.GetStore().GetModelMetadata(ctx, model.ID)
		if err == nil && existingMeta != nil {
			// Update existing metadata, preserve user settings
			existingMeta.Capabilities = detectedCaps
			if err := m.storageMgr.GetStore().SaveModelMetadata(ctx, existingMeta); err != nil {
				logger.Warnf("保存模型能力失败: modelId=%s, error=%v", model.ID, err)
			}
		} else {
			// Create new metadata entry with capabilities
			if err := m.storageMgr.GetStore().SaveModelMetadata(ctx, &storage.ModelMetadata{
				ModelID:      model.ID,
				Capabilities: detectedCaps,
			}); err != nil {
				logger.Warnf("保存模型能力失败: modelId=%s, error=%v", model.ID, err)
			}
		}
	}

	return model, nil
}

// loadModelWithValidation loads a model with additional validation
func (m *Manager) loadModelWithValidation(path string) (*Model, error) {
	// Validate file exists and is readable
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("无法访问模型文件: %w", err)
	}

	// Check file size (must be at least 1KB to be valid)
	if info.Size() < 1024 {
		return nil, fmt.Errorf("模型文件太小 (%d bytes), 可能已损坏", info.Size())
	}

	// Check file is readable
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取模型文件: %w", err)
	}
	utils.CloseQuietly(f)

	// Load model
	model, err := m.loadModel(path)
	if err != nil {
		return nil, err
	}

	// Validate metadata
	if model.Metadata == nil {
		return nil, fmt.Errorf("无法读取模型元数据")
	}

	return model, nil
}

// generateModelID generates a unique model ID using path hash
func (m *Manager) generateModelID(path string, metadata *gguf.Metadata) string {
	// Use hash of full path for uniqueness
	hash := sha256.Sum256([]byte(path))
	hashStr := hex.EncodeToString(hash[:8])

	// Get base name without extension
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".gguf")
	base = strings.TrimSuffix(base, ".GGUF")

	return fmt.Sprintf("%s-%s", base, hashStr)
}

// loadHuggingFaceModel loads a model from a HuggingFace model directory (safetensors format)
func (m *Manager) loadHuggingFaceModel(dirPath string) (*Model, error) {
	hfInfo, err := huggingface.ReadModelInfo(dirPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取 HuggingFace 模型信息: %w", err)
	}

	modelID := m.generateHFModelID(dirPath)
	pathPrefix := m.calculatePathPrefix(dirPath)
	displayName := hfInfo.Name
	if pathPrefix != "" && pathPrefix != "models" {
		displayName = fmt.Sprintf("[%s]%s", pathPrefix, hfInfo.Name)
	}

	model := &Model{
		ID:          modelID,
		Name:        hfInfo.Name,
		DisplayName: displayName,
		Path:        dirPath,
		PathPrefix:  pathPrefix,
		Size:        hfInfo.TotalSize,
		Metadata: &gguf.Metadata{
			Name:         hfInfo.Name,
			Architecture: strings.Join(hfInfo.Architectures, ","),
		},
		ScannedAt:  time.Now(),
		SourcePath: dirPath,
		SourceType: "huggingface",
	}

	detectedCaps := DetectCapabilitiesFromHF(hfInfo)
	ctx := context.Background()
	existingMeta, err := m.storageMgr.GetStore().GetModelMetadata(ctx, model.ID)
	if err == nil && existingMeta != nil {
		existingMeta.Capabilities = detectedCaps
		if saveErr := m.storageMgr.GetStore().SaveModelMetadata(ctx, existingMeta); saveErr != nil {
			logger.Warnf("保存模型能力失败: modelId=%s, error=%v", model.ID, saveErr)
		}
	} else {
		if saveErr := m.storageMgr.GetStore().SaveModelMetadata(ctx, &storage.ModelMetadata{
			ModelID:      model.ID,
			Capabilities: detectedCaps,
		}); saveErr != nil {
			logger.Warnf("保存模型能力失败: modelId=%s, error=%v", model.ID, saveErr)
		}
	}

	return model, nil
}

// generateHFModelID generates a unique model ID for a HuggingFace model directory
func (m *Manager) generateHFModelID(dirPath string) string {
	hash := sha256.Sum256([]byte(dirPath))
	hashStr := hex.EncodeToString(hash[:8])
	base := filepath.Base(dirPath)
	return fmt.Sprintf("%s-%s", base, hashStr)
}

// findMmproj looks for a multimodal projector file
func (m *Manager) findMmproj(modelPath string) string {
	dir := filepath.Dir(modelPath)
	base := filepath.Base(modelPath)

	// Remove .gguf extension
	base = strings.TrimSuffix(base, ".gguf")
	base = strings.TrimSuffix(base, ".GGUF")

	// Common mmproj patterns
	patterns := []string{
		filepath.Join(dir, base+"-mmproj.gguf"),
		filepath.Join(dir, base+"-mmproj-f16.gguf"),
		filepath.Join(dir, "mmproj.gguf"),
		filepath.Join(dir, "mmproj-model.gguf"),
	}

	for _, pattern := range patterns {
		if _, err := os.Stat(pattern); err == nil {
			return pattern
		}
	}

	return ""
}

// expandPath expands ~ to the user's home directory and converts relative paths to absolute.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		} else if u, err := user.Current(); err == nil {
			path = filepath.Join(u.HomeDir, path[2:])
		}
	} else if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			path = home
		} else if u, err := user.Current(); err == nil {
			path = u.HomeDir
		}
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// getScanPaths returns the list of scan paths (from PathConfigs or Paths)
// 从配置管理器获取最新配置，而不是使用初始化时的静态快照
func (m *Manager) getScanPaths() []string {
	// 获取配置:优先使用 configMgr,如果为 nil 则使用传入的 config
	var cfg *config.Config
	if m.configMgr != nil {
		cfg = m.configMgr.Get()
	} else {
		cfg = m.config
	}

	expandAll := func(raw []string) []string {
		out := make([]string, 0, len(raw))
		for _, p := range raw {
			out = append(out, expandPath(p))
		}
		return out
	}

	if len(cfg.Model.PathConfigs) > 0 {
		raw := make([]string, 0, len(cfg.Model.PathConfigs))
		for _, pc := range cfg.Model.PathConfigs {
			raw = append(raw, pc.Path)
		}
		for _, mp := range cfg.Backends.MultimodalPaths {
			raw = append(raw, mp.Path)
		}
		paths := expandAll(raw)
		logger.Debugf("getScanPaths: returning paths from PathConfigs: count=%d, paths=%v", len(paths), paths)
		return paths
	}
	raw := make([]string, 0, len(cfg.Model.Paths)+len(cfg.Backends.MultimodalPaths))
	raw = append(raw, cfg.Model.Paths...)
	for _, mp := range cfg.Backends.MultimodalPaths {
		raw = append(raw, mp.Path)
	}
	paths := expandAll(raw)
	logger.Debugf("getScanPaths: returning paths from Paths: count=%d, paths=%v", len(paths), paths)
	return paths
}

// calculatePathPrefix calculates a short path prefix for display
func (m *Manager) calculatePathPrefix(path string) string {
	// Get directory of the model file
	dir := filepath.Dir(path)

	// Check against configured scan paths
	for _, scanPath := range m.getScanPaths() {
		// Clean paths for comparison
		cleanScanPath := filepath.Clean(scanPath)
		cleanDir := filepath.Clean(dir)

		// Check if this path is under the scan path
		if strings.HasPrefix(cleanDir, cleanScanPath) {
			// Get relative path from scan root
			rel, err := filepath.Rel(cleanScanPath, cleanDir)
			if err != nil {
				continue
			}

			// Get scan path base name as root
			scanBase := filepath.Base(cleanScanPath)
			if scanBase == "." || scanBase == "/" {
				scanBase = "models"
			}

			// If relative path is "." (same dir), just return scan base
			if rel == "." {
				return scanBase
			}

			// Return scan base + first level subdir
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) > 0 && parts[0] != "" {
				return filepath.Join(scanBase, parts[0])
			}
			return scanBase
		}
	}

	// Fallback: use parent directory name
	return filepath.Base(dir)
}

// isSplitGGUF 检查是否为分卷文件
// 返回：是否为分卷、基础名称、分卷号、总分卷数
func isSplitGGUF(filename string) (bool, string, int, int) {
	// 匹配模式: "name-00001-of-00006.gguf"
	matches := reSplitGGUF.FindStringSubmatch(filename)
	if len(matches) == 4 {
		partNum, _ := strconv.Atoi(matches[2])
		totalParts, _ := strconv.Atoi(matches[3])
		return true, matches[1], partNum, totalParts
	}
	return false, "", 0, 0
}

// extractModelName 从文件名提取模型名称，移除分卷后缀
func extractModelName(filename string) string {
	// 移除扩展名
	name := strings.TrimSuffix(filename, ".gguf")
	name = strings.TrimSuffix(name, ".GGUF")

	// 移除分卷后缀
	name = reSplitSuffix.ReplaceAllString(name, "")

	return name
}

// generateUnifiedModelID 为分卷模型生成统一的模型ID
func generateUnifiedModelID(baseName string, partsCount int) string {
	hash := sha256.Sum256([]byte(baseName))
	hashStr := hex.EncodeToString(hash[:8])
	return fmt.Sprintf("%s-%dparts-%s", baseName, partsCount, hashStr)
}

// mergeSplitModels 合并分卷文件为单个模型
// 返回合并的组数量
func (m *Manager) mergeSplitModels() int {
	// 按目录和基础名称分组
	groups := make(map[string][]*Model)

	for _, model := range m.models {
		if isSplit, baseName, _, totalParts := isSplitGGUF(filepath.Base(model.Path)); isSplit {
			// 生成组键：目录 + 基础名称 + 总分卷数
			groupKey := fmt.Sprintf("%s/%s-%dparts", filepath.Dir(model.Path), baseName, totalParts)
			groups[groupKey] = append(groups[groupKey], model)
		}
	}

	mergedCount := 0

	// 对每组进行处理
	for groupKey, models := range groups {
		logger.Debugf("mergeSplitModels: processing group: groupKey=%s, found=%d", groupKey, len(models))

		if len(models) < 2 {
			// 只有一个分卷，不合并
			logger.Debug("mergeSplitModels: skipping group with fewer than 2 shards")
			continue
		}

		// 检查分卷是否完整（应该是连续的 1 到 n）
		// 按分卷号排序
		for i := 0; i < len(models); i++ {
			for j := i + 1; j < len(models); j++ {
				_, _, pi, _ := isSplitGGUF(filepath.Base(models[i].Path))
				_, _, pj, _ := isSplitGGUF(filepath.Base(models[j].Path))
				if pi > pj {
					models[i], models[j] = models[j], models[i]
				}
			}
		}

		// 验证分卷连续性
		_, _, firstPart, totalParts := isSplitGGUF(filepath.Base(models[0].Path))
		_ = firstPart // 避免未使用警告
		isComplete := true
		for i, model := range models {
			_, _, partNum, _ := isSplitGGUF(filepath.Base(model.Path))
			expectedPart := i + 1
			if partNum != expectedPart {
				isComplete = false
				logger.Warnf("shard file not contiguous: path=%s, expected=%d, actual=%d", model.Path, expectedPart, partNum)
			}
		}

		if !isComplete {
			logger.Warnf("shard group incomplete: groupKey=%s, found=%d, expected=%d", groupKey, len(models), totalParts)
		}

		// 使用第一卷作为主模型
		primary := models[0]

		// 计算总大小
		totalSize := int64(0)
		shardFiles := make([]string, len(models))
		for i, m := range models {
			totalSize += m.Size
			shardFiles[i] = m.Path
		}

		// ========== 查找并添加 mmproj 文件大小 ==========
		// 参考 LlamacppServer GGUFBundle.java 的实现
		mmprojSize := int64(0)
		mmprojPath := ""

		if len(models) > 0 {
			dir := filepath.Dir(models[0].Path)
			baseName := extractModelName(filepath.Base(models[0].Path))

			// 尝试多种 mmproj 命名模式（按优先级）
			candidates := []string{
				// 模式 1: mmproj-{basename}.gguf (最常见)
				filepath.Join(dir, "mmproj-"+baseName+".gguf"),
				// 模式 2: {basename}-mmproj.gguf
				filepath.Join(dir, baseName+"-mmproj.gguf"),
				// 模式 3: {basename}-mmproj-F32.gguf (精度变体)
				filepath.Join(dir, baseName+"-mmproj-F32.gguf"),
				filepath.Join(dir, baseName+"-mmproj-f32.gguf"),
				// 模式 4: {basename}-mmproj-F16.gguf
				filepath.Join(dir, baseName+"-mmproj-F16.gguf"),
				filepath.Join(dir, baseName+"-mmproj-f16.gguf"),
				// 模式 5: 目录内任何包含 "mmproj" 的 .gguf 文件（最后尝试）
			}

			// 首先尝试特定的命名模式
			for _, candidate := range candidates {
				if info, err := os.Stat(candidate); err == nil {
					mmprojSize = info.Size()
					mmprojPath = candidate
					logger.Infof("found mmproj file: name=%s, sizeGB=%.2f", filepath.Base(candidate), float64(mmprojSize)/(1024*1024*1024))
					break
				}
			}

			// 如果特定模式都找不到，尝试目录内搜索
			if mmprojPath == "" {
				entries, err := os.ReadDir(dir)
				if err == nil {
					for _, entry := range entries {
						if !entry.IsDir() && strings.Contains(strings.ToLower(entry.Name()), "mmproj") && strings.HasSuffix(strings.ToLower(entry.Name()), ".gguf") {
							fullPath := filepath.Join(dir, entry.Name())
							if info, err := os.Stat(fullPath); err == nil {
								mmprojSize = info.Size()
								mmprojPath = fullPath
								logger.Infof("found mmproj via directory search: name=%s, sizeGB=%.2f", entry.Name(), float64(mmprojSize)/(1024*1024*1024))
								break
							}
						}
					}
				}
			}
		}

		// 更新 TotalSize 包含 mmproj 文件
		totalSizeWithMmproj := totalSize + mmprojSize

		logger.Debugf("合并分卷模型前: name=%s, modelCount=%d, shardFiles=%d", primary.Name, len(models), len(shardFiles))

		// 更新主模型的属性
		primary.Name = extractModelName(filepath.Base(primary.Path))
		primary.TotalSize = totalSizeWithMmproj
		primary.ShardCount = len(models)
		primary.ShardFiles = shardFiles
		if mmprojPath != "" {
			primary.MmprojPath = mmprojPath
		}

		logger.Debugf("合并分卷模型后: name=%s, shardCount=%d, totalSizeGB=%s, mmprojSizeGB=%s, combinedSizeGB=%s",
			primary.Name, primary.ShardCount,
			fmt.Sprintf("%.2f", float64(totalSize)/(1024*1024*1024)),
			fmt.Sprintf("%.2f", float64(mmprojSize)/(1024*1024*1024)),
			fmt.Sprintf("%.2f", float64(primary.TotalSize)/(1024*1024*1024)),
		)

		// 删除其他分卷的模型记录
		for i := 1; i < len(models); i++ {
			delete(m.models, models[i].ID)
		}

		// 更新主模型 ID（使用统一的基础名称）
		newID := generateUnifiedModelID(primary.Name, len(models))
		m.models[newID] = primary
		delete(m.models, primary.ID)
		primary.ID = newID

		mergedCount++
		logger.Infof("已合并分卷模型: name=%s, shardCount=%d, totalSizeGB=%s",
			primary.Name, len(models),
			fmt.Sprintf("%.2f", float64(totalSize)/(1024*1024*1024)),
		)
	}

	return mergedCount
}
