package model

import (
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/gguf"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/port"
	"github.com/shepherd-project/shepherd/Shepherd/internal/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/storage"
)

// Manager manages model scanning and loading
type Manager struct {
	config        *config.Config
	configMgr     *config.Manager
	processMgr    *process.Manager
	portAllocator *port.PortAllocator
	storageMgr    *storage.Manager // 数据库存储管理器

	models     map[string]*Model
	statuses   map[string]*ModelStatus
	scanStatus *ScanStatus

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
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

	m := &Manager{
		config:        cfg,
		configMgr:     cfgMgr,
		processMgr:    procMgr,
		portAllocator: portAllocator,
		storageMgr:    storageMgr,
		models:        make(map[string]*Model),
		statuses:      make(map[string]*ModelStatus),
		scanStatus:    &ScanStatus{},
		ctx:           ctx,
		cancel:        cancel,
	}

	// Log initialization info
	paths := m.getScanPaths()
	if len(paths) == 0 {
		logger.Warn("ModelManager: 未配置模型扫描路径")
	} else {
		logger.Info("ModelManager: 初始化完成", "paths", paths)
	}

	// Load saved models
	m.loadModels()
	logger.Info("ModelManager: 从配置加载模型完成", "modelCount", len(m.models))

	return m
}

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
	logger.Info("开始扫描模型", "pathCount", len(scanPaths))

	// Scan each configured path
	for _, scanPath := range scanPaths {
		logger.Info("正在扫描路径", "path", scanPath)
		pathModels, pathErrors := m.scanPath(ctx, scanPath)
		logger.Info("路径扫描完成", "path", scanPath, "modelCount", len(pathModels), "errorCount", len(pathErrors))
		result.Models = append(result.Models, pathModels...)
		result.Errors = append(result.Errors, pathErrors...)
		result.TotalFiles += len(pathModels) + len(pathErrors)
		result.MatchedFiles += len(pathModels)
	}

	result.Duration = time.Since(result.ScannedAt)
	logger.Info("模型扫描完成", "totalModels", len(result.Models), "duration", result.Duration.String(), "totalErrors", len(result.Errors))

	// ========== 修复：从数据库加载用户设置的属性 ==========
	// 从数据库加载所有模型的元数据（别名、收藏、标签、描述、使用统计等）
	modelMetadataMap := make(map[string]*storage.ModelMetadata)
	if m.storageMgr != nil {
		store := m.storageMgr.GetStore()
		metadata, err := store.GetAllModelMetadata(ctx)
		if err != nil {
			logger.Warn("从数据库加载模型元数据失败", "error", err)
		} else {
			modelMetadataMap = metadata
			logger.Info("从数据库加载模型元数据", "count", len(metadata))
		}
	}
	// ==============================================

	// Update models map（先清空，再添加）
	m.mu.Lock()
	m.models = make(map[string]*Model) // 清空旧数据
	for _, model := range result.Models {
		m.models[model.ID] = model
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
			logger.Debug("恢复模型用户属性", "id", id, "alias", metadata.Alias, "favourite", metadata.Favourite)
		}
	}
	// ==============================================

	// ========== 新增：合并分卷文件 ==========
	mergedCount := m.mergeSplitModels()
	if mergedCount > 0 {
		logger.Info("已合并分卷文件", "mergedCount", mergedCount)
	}

	modelCount := len(m.models)
	m.mu.Unlock()
	logger.Info("模型缓存已更新", "modelCount", modelCount)

	// Save to config
	m.saveModels()
	logger.Info("已保存模型到配置", "savedCount", len(m.models))

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
		logger.Error("路径访问失败", "path", scanPath, "error", err)
		return nil, []ScanError{{Path: scanPath, Error: fmt.Sprintf("路径访问失败: %v", err)}}
	}

	pathType := "文件"
	if info.IsDir() {
		pathType = "目录"
	}
	logger.Debug("开始扫描路径", "path", scanPath, "type", pathType)

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
				return nil
			}

			fileCount++

			// 检查是否为模型文件
			if m.isModelFile(path) {
				matchedCount++
				logger.Debug("找到模型文件", "path", path)

				wg.Add(1)
				semaphore <- struct{}{}

				go func(filePath string) {
					defer wg.Done()
					defer func() { <-semaphore }()

					model, err := m.loadModelWithValidation(filePath)
					if err != nil {
						logger.Warn("加载模型失败", "path", filePath, "error", err)
						mu.Lock()
						errors = append(errors, ScanError{
							Path:  filePath,
							Error: err.Error(),
						})
						mu.Unlock()
					} else {
						logger.Info("成功加载模型", "name", model.Name, "id", model.ID, "path", filePath)
						mu.Lock()
						models = append(models, model)
						mu.Unlock()
					}
				}(path)
			}

			return nil
		})

		wg.Wait()

		logger.Debug("路径扫描完成", "path", scanPath, "fileCount", fileCount, "matchedCount", matchedCount)

		if err != nil && err != ctx.Err() {
			errors = append(errors, ScanError{
				Path:  scanPath,
				Error: fmt.Sprintf("扫描中断: %v", err),
			})
		}
	} else if m.isModelFile(scanPath) {
		logger.Info("单文件模型", "path", scanPath)
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
		logger.Warn("路径不是模型文件", "path", scanPath)
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
	if matched, _ := regexp.MatchString(`models--.+--.+\.gguf$`, path); matched {
		return true
	}

	// 模式2: HuggingFace 缓存中的 snapshots 目录
	if strings.Contains(path, "snapshots") {
		// 检查是否包含模型文件扩展名
		if strings.HasSuffix(base, ".safetensors") ||
			strings.HasSuffix(base, ".bin") ||
			strings.HasSuffix(base, ".safetensors") {
			logger.Debug("找到 HuggingFace 格式模型", "path", path)
			return true
		}
	}

	return false
}

// isGGUFFile checks if a file is specifically a GGUF model file (deprecated, use isModelFile)
// 保留此方法以兼容性，内部调用 isModelFile
func (m *Manager) isGGUFFile(path string) bool {
	return m.isModelFile(path)
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
				logger.Warn("保存模型能力失败", "modelId", model.ID, "error", err)
			}
		} else {
			// Create new metadata entry with capabilities
			if err := m.storageMgr.GetStore().SaveModelMetadata(ctx, &storage.ModelMetadata{
				ModelID:      model.ID,
				Capabilities: detectedCaps,
			}); err != nil {
				logger.Warn("保存模型能力失败", "modelId", model.ID, "error", err)
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

	if len(cfg.Model.PathConfigs) > 0 {
		paths := make([]string, 0, len(cfg.Model.PathConfigs))
		for _, pc := range cfg.Model.PathConfigs {
			paths = append(paths, pc.Path)
		}
		fmt.Printf("[DEBUG] getScanPaths: 从 PathConfigs 返回 %d 个路径: %v\n", len(paths), paths)
		return paths
	}
	fmt.Printf("[DEBUG] getScanPaths: 从 Paths 返回 %d 个路径: %v\n", len(cfg.Model.Paths), cfg.Model.Paths)
	return cfg.Model.Paths
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

	// 如果内存中没有模型，自动触发一次扫描
	if modelCount == 0 {
		fmt.Printf("[INFO] ListModels: 内存中没有模型，触发自动扫描\n")
		// 使用 background context 进行自动扫描
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// 在 goroutine 中执行扫描，避免阻塞当前调用
		done := make(chan bool, 1)
		go func() {
			if _, err := m.Scan(ctx); err != nil {
				fmt.Printf("[WARN] ListModels: 自动扫描失败: %v\n", err)
			}
			done <- true
		}()

		// 等待扫描完成，最多等待 10 秒
		select {
		case <-done:
			fmt.Printf("[INFO] ListModels: 自动扫描完成\n")
		case <-time.After(10 * time.Second):
			fmt.Printf("[WARN] ListModels: 自动扫描超时，返回当前模型列表\n")
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

// Load loads a model
func (m *Manager) Load(req *LoadRequest) (*LoadResult, error) {
	// Get model
	model, exists := m.GetModel(req.ModelID)
	if !exists {
		logger.Warn("模型加载失败: 模型不存在", "modelId", req.ModelID)
		return nil, fmt.Errorf("model not found: %s", req.ModelID)
	}

	logger.Info("开始加载模型", "modelId", req.ModelID, "modelName", model.Name, "ctxSize", req.CtxSize, "gpuLayers", req.GPULayers)

	// Phase 1: 检查状态并创建初始 status（加锁）
	var status *ModelStatus
	m.mu.Lock()
	if existingStatus, exists := m.statuses[req.ModelID]; exists && existingStatus.State == StateLoading {
		m.mu.Unlock()
		logger.Warn("模型加载失败: 模型正在加载中", "modelId", req.ModelID)
		return nil, fmt.Errorf("model already loading: %s", req.ModelID)
	}

	// Create initial status
	status = &ModelStatus{
		ID:    req.ModelID,
		Name:  model.Name,
		State: StateLoading,
	}
	m.statuses[req.ModelID] = status
	m.mu.Unlock()

	startTime := time.Now()

	// Phase 2: 准备工作（无锁）- 所有可能失败的操作
	// Find llama.cpp binary
	binPath := m.findLlamaCppBinary()
	if binPath == "" {
		// 更新状态为错误
		m.mu.Lock()
		status.State = StateError
		status.Error = fmt.Errorf("llama.cpp binary not found")
		m.mu.Unlock()
		logger.Error("模型加载失败: llama.cpp 二进制文件未找到", "modelId", req.ModelID)
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   status.Error,
		}, status.Error
	}

	// Allocate port using centralized PortAllocator
	port, err := m.portAllocator.NextPort()
	if err != nil {
		m.mu.Lock()
		status.State = StateError
		status.Error = fmt.Errorf("no available ports: %w", err)
		m.mu.Unlock()
		logger.Error("模型加载失败: 无可用端口", "modelId", req.ModelID, "error", err)
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   status.Error,
		}, status.Error
	}

	// Determine model path
	modelPath := model.Path
	if len(model.ShardFiles) > 0 {
		modelPath = model.ShardFiles[0]
		fmt.Printf("[INFO] 使用分卷模型主文件: %s (共 %d 个分卷)\n", modelPath, len(model.ShardFiles))
	}

	// Build command
	procReq := toProcessLoadRequest(req, modelPath, port)
	cmd, err := process.BuildCommandFromRequest(procReq, binPath)
	if err != nil {
		// 释放已分配的端口
		m.portAllocator.Release(port)
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		return &LoadResult{
			Success: false,
			ModelID: req.ModelID,
			Error:   err,
		}, err
	}

	// Start process
	proc, err := m.processMgr.Start(req.ModelID, model.Name, cmd, binPath)
	if err != nil {
		// 释放已分配的端口
		m.portAllocator.Release(port)
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		logger.Error("模型加载失败: 启动进程失败", "modelId", req.ModelID, "error", err)
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

	// Phase 3: 更新状态为已加载（加锁）
	m.mu.Lock()
	status.State = StateLoaded
	status.ProcessID = proc.ID
	status.Port = port
	status.LoadedAt = time.Now()
	m.mu.Unlock()

	duration := time.Since(startTime)

	logger.Info("模型加载成功", "modelId", req.ModelID, "port", port, "duration", duration.String(), "pid", proc.GetPID())

	return &LoadResult{
		Success:  true,
		ModelID:  req.ModelID,
		Port:     port,
		CtxSize:  req.CtxSize,
		Duration: duration,
	}, nil
}

// LoadAsync 异步加载模型（立即返回，后台加载）
func (m *Manager) LoadAsync(req *LoadRequest) (*LoadResult, error) {
	// Get model
	model, exists := m.GetModel(req.ModelID)
	if !exists {
		return nil, fmt.Errorf("model not found: %s", req.ModelID)
	}

	// Check if already loaded
	m.mu.RLock()
	if status, exists := m.statuses[req.ModelID]; exists {
		if status.State == StateLoaded {
			m.mu.RUnlock()
			return &LoadResult{
				Success:  true,
				ModelID:  req.ModelID,
				Port:     status.Port,
				Async:    true,
				AlreadyLoaded: true,
			}, nil
		}
		if status.State == StateLoading {
			m.mu.RUnlock()
			return &LoadResult{
				Success:  true,
				ModelID:  req.ModelID,
				Async:    true,
				Loading:  true,
			}, nil
		}
	}
	m.mu.RUnlock()

	// 创建初始状态
	m.mu.Lock()
	status := &ModelStatus{
		ID:    req.ModelID,
		Name:  model.Name,
		State: StateLoading,
	}
	m.statuses[req.ModelID] = status
	m.mu.Unlock()

	// 启动异步加载（传入已获取的 model，避免在 goroutine 中再次获取锁）
	go m.loadModelAsync(req, status, model)

	return &LoadResult{
		Success:  true,
		ModelID:  req.ModelID,
		Async:    true,
		Loading:  true,
	}, nil
}

// loadModelAsync 后台异步加载模型
// model 参数已在 LoadAsync 中获取，避免在此 goroutine 中再次获取锁导致死锁
func (m *Manager) loadModelAsync(req *LoadRequest, status *ModelStatus, model *Model) {
	startTime := time.Now()

	logger.Info("开始异步加载模型", "modelId", req.ModelID, "modelName", model.Name)

	// Find llama.cpp binary
	binPath := m.findLlamaCppBinary()
	if binPath == "" {
		m.mu.Lock()
		status.State = StateError
		status.Error = fmt.Errorf("llama.cpp binary not found")
		m.mu.Unlock()
		logger.Error("异步模型加载失败: llama.cpp 二进制文件未找到", "modelId", req.ModelID)
		return
	}

	// Build command using BuildCommandFromRequest
	// Allocate port using centralized PortAllocator
	port, err := m.portAllocator.NextPort()
	if err != nil {
		m.mu.Lock()
		status.State = StateError
		status.Error = fmt.Errorf("no available ports: %w", err)
		m.mu.Unlock()
		logger.Error("异步模型加载失败: 无可用端口", "modelId", req.ModelID, "error", err)
		return
	}

	// 使用传入的 model 参数，不再调用 GetModel 避免死锁
	modelPath := model.Path
	if len(model.ShardFiles) > 0 {
		modelPath = model.ShardFiles[0]
		logger.Info("使用分卷模型主文件", "modelId", req.ModelID, "mainFile", modelPath, "shardCount", len(model.ShardFiles))
	}

	// 根据模型大小计算超时时间
	timeout := m.calculateLoadTimeout(model.Size)
	logger.Info("模型加载超时设置", "modelId", req.ModelID, "sizeGB", float64(model.Size)/(1024*1024*1024), "timeout", timeout)

	// Convert to process.LoadRequest and build command
	procReq := toProcessLoadRequest(req, modelPath, port)
	logger.Info("构建进程命令", "modelId", req.ModelID, "port", port)
	cmd, err := process.BuildCommandFromRequest(procReq, binPath)
	if err != nil {
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		logger.Error("异步模型加载失败: 构建命令失败", "modelId", req.ModelID, "error", err)
		return
	}
	logger.Info("命令构建成功", "modelId", req.ModelID, "cmd", cmd)

	// Start process
	proc, err := m.processMgr.Start(req.ModelID, model.Name, cmd, binPath)
	if err != nil {
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		logger.Error("异步模型加载失败: 启动进程失败", "modelId", req.ModelID, "error", err)
		return
	}

	logger.Info("进程启动成功，准备获取 PID", "modelId", req.ModelID)

	// 获取 PID (可能在短时间内阻塞，但应该很快返回)
	pid := proc.GetPID()
	logger.Info("异步模型加载: 进程已启动", "modelId", req.ModelID, "pid", pid, "port", port)

	// 等待加载完成（监控进程输出）
	loadCompleted := make(chan bool, 1)
	loadError := make(chan error, 1)
	stopHealthCheck := make(chan bool, 1) // 用于停止健康检查

	// 启动进程健康检查 goroutine
	logger.Info("启动健康检查 goroutine", "modelId", req.ModelID, "healthURL", fmt.Sprintf("http://localhost:%d/health", port))
	go func() {
		// 立即执行一次健康检查（不等待 ticker）
		httpClient := &http.Client{Timeout: 2 * time.Second}
		healthURL := fmt.Sprintf("http://localhost:%d/health", port)

		// 启动 ticker 进行定期检查
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		// 健康检查失败计数器，超过10次则放弃
		failureCount := 0
		const maxFailures = 10

		// 首次立即检查
		func() {
			logger.Info("首次健康检查", "modelId", req.ModelID, "url", healthURL)
			if !proc.IsRunning() {
				logger.Error("首次健康检查发现进程已退出", "modelId", req.ModelID)
				select {
				case loadError <- fmt.Errorf("进程启动后立即退出"):
				default:
				}
				return
			}

			resp, err := httpClient.Get(healthURL)
			if err != nil {
				failureCount++
				logger.Warn("首次健康检查请求失败", "modelId", req.ModelID, "error", err, "failureCount", failureCount, "maxFailures", maxFailures)
			} else if resp.StatusCode != 200 {
				failureCount++
				logger.Warn("首次健康检查返回非200状态", "modelId", req.ModelID, "statusCode", resp.StatusCode, "failureCount", failureCount, "maxFailures", maxFailures)
				utils.CloseQuietly(resp.Body)
			} else {
				body, _ := io.ReadAll(resp.Body)
				utils.CloseQuietly(resp.Body)
				logger.Info("首次健康检查响应", "modelId", req.ModelID, "body", string(body))
				if strings.Contains(string(body), `"status":"ok"`) {
					logger.Info("首次健康检查成功，模型已就绪", "modelId", req.ModelID, "port", port)
					select {
					case loadCompleted <- true:
					default:
					}
					return
				}
			}
		}()

		// 定期检查
		checkCount := 0
		for {
			select {
			case <-stopHealthCheck:
				// 收到停止信号，退出健康检查
				logger.Info("健康检查收到停止信号", "modelId", req.ModelID, "checkCount", checkCount)
				return
			case <-ticker.C:
				checkCount++
				logger.Info("执行定期健康检查", "modelId", req.ModelID, "checkCount", checkCount, "url", healthURL)

				// 定期检查进程是否仍在运行
				if !proc.IsRunning() {
					// 进程意外退出
					logger.Error("定期健康检查发现进程已退出", "modelId", req.ModelID, "pid", proc.GetPID())
					select {
					case loadError <- fmt.Errorf("进程意外退出 (PID: %d)", proc.GetPID()):
					default:
					}
					return
				}

				// 检查 HTTP 健康端点（备用检测机制）
				resp, err := httpClient.Get(healthURL)
				if err != nil {
					failureCount++
					logger.Warn("HTTP 健康检查请求失败", "modelId", req.ModelID, "url", healthURL, "error", err, "failureCount", failureCount, "maxFailures", maxFailures)

					// 检查是否超过最大失败次数
					if failureCount >= maxFailures {
						logger.Error("健康检查失败次数超过限制，终止进程", "modelId", req.ModelID, "failureCount", failureCount, "maxFailures", maxFailures)
						// 杀死进程
						utils.StopQuietly(proc)
						// 返回错误
						select {
						case loadError <- fmt.Errorf("健康检查连续失败 %d 次，已终止进程 (PID: %d)", failureCount, proc.GetPID()):
						default:
						}
						return
					}
				} else if resp.StatusCode != 200 {
					failureCount++
					logger.Warn("HTTP 健康检查返回非200状态", "modelId", req.ModelID, "statusCode", resp.StatusCode, "failureCount", failureCount, "maxFailures", maxFailures)
					utils.CloseQuietly(resp.Body)

					// 检查是否超过最大失败次数
					if failureCount >= maxFailures {
						logger.Error("健康检查失败次数超过限制，终止进程", "modelId", req.ModelID, "failureCount", failureCount, "maxFailures", maxFailures)
						// 杀死进程
						utils.StopQuietly(proc)
						// 返回错误
						select {
						case loadError <- fmt.Errorf("健康检查连续失败 %d 次，已终止进程 (PID: %d)", failureCount, proc.GetPID()):
						default:
						}
						return
					}
				} else {
					body, _ := io.ReadAll(resp.Body)
					utils.CloseQuietly(resp.Body)
					logger.Info("HTTP 健康检查响应", "modelId", req.ModelID, "body", string(body))
					// 检查响应内容是否为 {"status":"ok"}
					if strings.Contains(string(body), `"status":"ok"`) {
						logger.Info("HTTP 健康检查成功，模型已就绪", "modelId", req.ModelID, "port", port, "checkCount", checkCount)
						select {
						case loadCompleted <- true:
						default:
						}
						return
					}
					// 响应成功但状态不是 ok，重置失败计数
					failureCount = 0
				}
			}
		}
	}()

	// 设置输出处理器检测加载完成并转发日志
	proc.SetOutputHandler(func(line string) {
		// 将 llama.cpp 输出转发到日志系统
		// 过滤掉过于频繁的日志
		if !strings.Contains(line, "update_slots") && !strings.Contains(line, "log_server_r") {
			// 使用 debug 级别记录 llama.cpp 输出，避免日志过多
			logger.Debug(fmt.Sprintf("[%s] %s", req.ModelID, line))
		}

		// 检测加载完成
		if strings.Contains(line, "all slots are idle") {
			select {
			case loadCompleted <- true:
			default:
			}
		}
	})

	// 等待加载完成或超时
	select {
	case <-loadCompleted:
		// 停止健康检查
		close(stopHealthCheck)
		m.mu.Lock()
		status.State = StateLoaded
		status.ProcessID = proc.ID
		status.Port = port
		status.LoadedAt = time.Now()
		m.mu.Unlock()
		duration := time.Since(startTime)
		logger.Info("异步模型加载成功", "modelId", req.ModelID, "port", port, "duration", duration.String())

	case err := <-loadError:
		// 停止健康检查
		close(stopHealthCheck)
		m.mu.Lock()
		status.State = StateError
		status.Error = err
		m.mu.Unlock()
		logger.Error("异步模型加载失败", "modelId", req.ModelID, "error", err)
		// 清理进程和端口
		m.processMgr.Stop(req.ModelID) //errcheck:ignore
		m.portAllocator.Release(port)

	case <-time.After(timeout):
		// 停止健康检查
		close(stopHealthCheck)
		m.mu.Lock()
		status.State = StateError
		status.Error = fmt.Errorf("模型加载超时 (%v)", timeout)
		m.mu.Unlock()
		logger.Error("异步模型加载超时", "modelId", req.ModelID, "timeout", timeout)
		// 清理进程和端口
		m.processMgr.Stop(req.ModelID) //errcheck:ignore
		m.portAllocator.Release(port)
	}
}

// calculateLoadTimeout 根据模型大小计算动态超时时间
// 规则：每GB 1分钟，最少5分钟，最多30分钟
func (m *Manager) calculateLoadTimeout(modelSize int64) time.Duration {
	const (
		minTimeout     = 5 * time.Minute
		maxTimeout     = 30 * time.Minute
		minutesPerGB   = 1 * time.Minute
		gigabyte       = int64(1024 * 1024 * 1024)
	)

	// 计算基于模型大小的超时时间
	sizeGB := float64(modelSize) / float64(gigabyte)
	dynamicTimeout := time.Duration(sizeGB) * minutesPerGB

	// 限制在最小和最大值之间
	if dynamicTimeout < minTimeout {
		dynamicTimeout = minTimeout
	}
	if dynamicTimeout > maxTimeout {
		dynamicTimeout = maxTimeout
	}

	return dynamicTimeout
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

// Unload unloads a model
func (m *Manager) Unload(modelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, exists := m.statuses[modelID]
	if !exists {
		logger.Warn("模型卸载失败: 模型未加载", "modelId", modelID)
		return fmt.Errorf("model not loaded: %s", modelID)
	}

	if status.State != StateLoaded {
		logger.Warn("模型卸载失败: 模型未处于已加载状态", "modelId", modelID, "state", status.State)
		return fmt.Errorf("model not in loaded state: %s", modelID)
	}

	logger.Info("开始卸载模型", "modelId", modelID, "modelName", status.Name, "port", status.Port)

	// Stop process
	if err := m.processMgr.Stop(modelID); err != nil {
		logger.Error("模型卸载失败: 停止进程失败", "modelId", modelID, "error", err)
		return err
	}

	// Release allocated port
	if status.Port > 0 {
		m.portAllocator.Release(status.Port)
		logger.Info("已释放端口", "modelId", modelID, "port", status.Port)
	}

	// Update status
	status.State = StateUnloaded
	status.ProcessID = ""
	status.Port = 0

	logger.Info("模型卸载成功", "modelId", modelID, "modelName", status.Name)

	return nil
}

// GetStatus returns the status of a model
func (m *Manager) GetStatus(modelID string) (*ModelStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.statuses[modelID]
	if !exists {
		return nil, false
	}

	// Return a copy
	statusCopy := *status
	return &statusCopy, true
}

// ListStatus returns all model statuses
func (m *Manager) ListStatus() map[string]*ModelStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]*ModelStatus, len(m.statuses))
	for k, v := range m.statuses {
		statusCopy := *v
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

	// Save to database
	if m.storageMgr != nil {
		store := m.storageMgr.GetStore()

		// 获取或创建元数据
		metadata, err := store.GetModelMetadata(m.ctx, modelID)
		if err != nil {
			// 元数据不存在，创建新的
			metadata = &storage.ModelMetadata{
				ModelID:     modelID,
				StoragePath: filepath.Dir(model.Path),
				Favourite:   model.Favourite,
				Tags:        model.Tags,
				LoadCount:   model.LoadCount,
			}
			if !model.LastLoaded.IsZero() {
				metadata.LastLoaded = &model.LastLoaded
			}
			metadata.TotalTokens = model.TotalTokens
		}

		metadata.Alias = alias

		if err := store.SaveModelMetadata(m.ctx, metadata); err != nil {
			logger.Error("保存模型别名到数据库失败", "modelId", modelID, "error", err)
			return fmt.Errorf("failed to save alias to database: %w", err)
		}
		logger.Info("模型别名已保存到数据库", "modelId", modelID, "alias", alias)
	}

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

	// Save to database
	if m.storageMgr != nil {
		store := m.storageMgr.GetStore()

		// 获取或创建元数据
		metadata, err := store.GetModelMetadata(m.ctx, modelID)
		if err != nil {
			// 元数据不存在，创建新的
			metadata = &storage.ModelMetadata{
				ModelID:     modelID,
				StoragePath: filepath.Dir(model.Path),
				Alias:       model.Alias,
				Tags:        model.Tags,
				LoadCount:   model.LoadCount,
			}
			if !model.LastLoaded.IsZero() {
				metadata.LastLoaded = &model.LastLoaded
			}
			metadata.TotalTokens = model.TotalTokens
		}

		metadata.Favourite = favourite

		if err := store.SaveModelMetadata(m.ctx, metadata); err != nil {
			logger.Error("保存模型收藏状态到数据库失败", "modelId", modelID, "error", err)
			return fmt.Errorf("failed to save favourite to database: %w", err)
		}
		logger.Info("模型收藏状态已保存到数据库", "modelId", modelID, "favourite", favourite)
	}

	return nil
}

// loadModels loads models from config
func (m *Manager) loadModels() {
	if m.configMgr == nil {
		fmt.Printf("[WARN] loadModels: configMgr is nil, skipping load\n")
		return
	}

	configModels, err := m.configMgr.LoadModelsConfig()
	if err != nil {
		fmt.Printf("[ERROR] loadModels: failed to load models config: %v\n", err)
		return
	}

	fmt.Printf("[INFO] loadModels: loaded %d models from config\n", len(configModels))

	// Load aliases and favourites
	aliases, _ := m.configMgr.LoadAliasMap()
	favourites, _ := m.configMgr.LoadFavouriteMap()

	loadedCount := 0
	for _, cfgModel := range configModels {
		// 跳过 mmproj 文件（这些是多模态投影器，应该作为主模型的附件）
		base := filepath.Base(cfgModel.Path)
		if strings.Contains(base, "mmproj") || strings.HasPrefix(base, "mmproj") {
			fmt.Printf("[INFO] loadModels: 跳过 mmproj 文件: %s\n", cfgModel.Path)
			continue
		}

		// Try to load the model from disk
		if info, err := os.Stat(cfgModel.Path); err == nil && !info.IsDir() {
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
					fmt.Printf("[INFO] loadModels: 加载分卷模型 %s，共 %d 个分卷，总大小 %.2f GB\n",
						model.Name, model.ShardCount, float64(model.TotalSize)/(1024*1024*1024))
				}

				// 加载 mmproj 路径（如果配置中有保存）
				if cfgModel.Mmproj != nil && cfgModel.Mmproj.FileName != "" {
					mmprojPath := filepath.Join(filepath.Dir(cfgModel.Path), cfgModel.Mmproj.FileName)
					if info, err := os.Stat(mmprojPath); err == nil {
						model.MmprojPath = mmprojPath
						fmt.Printf("[INFO] loadModels: 加载 mmproj 文件 %s (%.2f GB)\n",
							cfgModel.Mmproj.FileName, float64(info.Size())/(1024*1024*1024))
					} else {
						fmt.Printf("[WARN] loadModels: mmproj 文件不存在: %s\n", mmprojPath)
					}
				}

				m.models[model.ID] = model
				loadedCount++
			} else {
				fmt.Printf("[WARN] loadModels: failed to load model %s: %v\n", cfgModel.Path, err)
			}
		} else {
			fmt.Printf("[WARN] loadModels: model file not found: %s\n", cfgModel.Path)
		}
	}
	fmt.Printf("[INFO] loadModels: successfully loaded %d models into cache\n", loadedCount)

	// ========== 合并分卷文件 ==========
	// 注意：如果配置中已经保存了分卷信息，这里不需要再次合并
	// 但如果配置中没有分卷信息，则尝试合并
	mergedCount := m.mergeSplitModels()
	if mergedCount > 0 {
		fmt.Printf("[INFO] loadModels: 已合并 %d 组分卷文件\n", mergedCount)
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

	if err := m.configMgr.SaveModelsConfig(configModels); err != nil { logger.Warn("保存模型配置失败", "error", err) }
}

// findLlamaCppBinary finds the llama.cpp binary
// 从配置管理器获取最新配置，而不是使用初始化时的静态快照
func (m *Manager) findLlamaCppBinary() string {
	// 从配置管理器获取最新配置
	cfg := m.configMgr.Get()

	// Check configured paths
	for _, llamacppPath := range cfg.Llamacpp.Paths {
		binaryPath := filepath.Join(llamacppPath.Path, "llama-server")
		if _, err := os.Stat(binaryPath); err == nil {
			return llamacppPath.Path
		}
	}

	// Check common locations
	commonPaths := []string{
		"/usr/local/bin",
		"/usr/bin",
		"./llama.cpp",
	}

	for _, path := range commonPaths {
		binaryPath := filepath.Join(path, "llama-server")
		if _, err := os.Stat(binaryPath); err == nil {
			return path
		}
	}

	return ""
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

// GetLoadedModelCount returns the number of currently loaded models
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

// ValidationResult represents the result of model validation
type ValidationResult struct {
	Valid         bool
	Errors        []string
	Warnings      []string
	ModelCount    int
	InvalidModels []string
	MissingFiles  []string
}

// ValidateModels validates all models in the cache
func (m *Manager) ValidateModels() *ValidationResult {
	result := &ValidationResult{
		Valid:         true,
		Errors:        []string{},
		Warnings:      []string{},
		InvalidModels: []string{},
		MissingFiles:  []string{},
	}

	m.mu.RLock()
	models := make(map[string]*Model)
	for id, model := range m.models {
		models[id] = model
	}
	m.mu.RUnlock()

	result.ModelCount = len(models)

	for id, model := range models {
		// Check if file exists
		if _, err := os.Stat(model.Path); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("模型 %s 文件不存在: %v", id, err))
			result.MissingFiles = append(result.MissingFiles, model.Path)
			result.InvalidModels = append(result.InvalidModels, id)
			continue
		}

		// Check file size consistency
		if info, err := os.Stat(model.Path); err == nil {
			if info.Size() != model.Size {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("模型 %s 文件大小已变更: 缓存 %d, 实际 %d", id, model.Size, info.Size()))
			}
		}

		// Validate metadata
		if model.Metadata == nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("模型 %s 缺少元数据", id))
			result.InvalidModels = append(result.InvalidModels, id)
		}
	}

	return result
}

// CleanInvalidModels removes invalid models from cache
func (m *Manager) CleanInvalidModels() int {
	result := m.ValidateModels()
	if !result.Valid {
		m.mu.Lock()
		for _, id := range result.InvalidModels {
			delete(m.models, id)
			delete(m.statuses, id)
		}
		m.mu.Unlock()

		// Save updated models
		m.saveModels()
	}

	return len(result.InvalidModels)
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

// isSplitGGUF 检查是否为分卷文件
// 返回：是否为分卷、基础名称、分卷号、总分卷数
func isSplitGGUF(filename string) (bool, string, int, int) {
	// 匹配模式: "name-00001-of-00006.gguf"
	re := regexp.MustCompile(`^(.*?)-(\d{5})-of-(\d{5})\.gguf$`)
	matches := re.FindStringSubmatch(filename)
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
	re := regexp.MustCompile(`-\d{5}-of-\d{5}$`)
	name = re.ReplaceAllString(name, "")

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
		fmt.Printf("[DEBUG] 处理分卷组: %s，找到 %d 个文件\n", groupKey, len(models))

		if len(models) < 2 {
			// 只有一个分卷，不合并
			fmt.Printf("[DEBUG] 跳过（少于 2 个分卷）\n")
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
				fmt.Printf("[WARN] 分卷文件不连续: %s，期望分卷 %d，实际是 %d\n",
					model.Path, expectedPart, partNum)
			}
		}

		if !isComplete {
			fmt.Printf("[WARN] 分卷组 %s 不完整，只有 %d/%d 个分卷\n",
				groupKey, len(models), totalParts)
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
					fmt.Printf("[INFO] 找到 mmproj 文件: %s (%.2f GB)\n",
						filepath.Base(candidate), float64(mmprojSize)/(1024*1024*1024))
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
								fmt.Printf("[INFO] 通过目录搜索找到 mmproj 文件: %s (%.2f GB)\n",
									entry.Name(), float64(mmprojSize)/(1024*1024*1024))
								break
							}
						}
					}
				}
			}
		}

		// 更新 TotalSize 包含 mmproj 文件
		totalSizeWithMmproj := totalSize + mmprojSize

		fmt.Printf("[DEBUG] 合并前: primary.Name=%s, len(models)=%d, len(shardFiles)=%d\n",
			primary.Name, len(models), len(shardFiles))

		// 更新主模型的属性
		primary.Name = extractModelName(filepath.Base(primary.Path))
		primary.TotalSize = totalSizeWithMmproj
		primary.ShardCount = len(models)
		primary.ShardFiles = shardFiles
		if mmprojPath != "" {
			primary.MmprojPath = mmprojPath
		}

		fmt.Printf("[DEBUG] 合并后: primary.Name=%s, ShardCount=%d, TotalSize=%.2f GB (分卷) + %.2f GB (mmproj) = %.2f GB\n",
			primary.Name, primary.ShardCount,
			float64(totalSize)/(1024*1024*1024),
			float64(mmprojSize)/(1024*1024*1024),
			float64(primary.TotalSize)/(1024*1024*1024))

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
		fmt.Printf("[INFO] 已合并分卷模型: %s (%d 个分卷, 总大小: %.2f GB)\n",
			primary.Name, len(models), float64(totalSize)/(1024*1024*1024))
	}

	return mergedCount
}


// toProcessLoadRequest converts model.LoadRequest to process.LoadRequest for command building
// This bridges the canonical LoadRequest (with ModelID/NodeID) to the command-building LoadRequest (with ModelPath/Port)
func toProcessLoadRequest(req *LoadRequest, modelPath string, port int) *process.LoadRequest {
	return &process.LoadRequest{
		ModelPath:        modelPath,
		Port:             port,
		CtxSize:          req.CtxSize,
		BatchSize:        req.BatchSize,
		Threads:          req.Threads,
		GPULayers:        req.GPULayers,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		TopK:             req.TopK,
		RepeatPenalty:    req.RepeatPenalty,
		Seed:             req.Seed,
		NPredict:         req.NPredict,
		Devices:          req.Devices,
		MainGPU:          req.MainGPU,
		CustomCmd:        req.CustomCmd,
		ExtraParams:      req.ExtraParams,
		MmprojPath:       req.MmprojPath,
		EnableVision:     req.EnableVision,
		FlashAttention:   req.FlashAttention,
		NoMmap:           req.NoMmap,
		LockMemory:       req.LockMemory,
		NoWebUI:          req.NoWebUI,
		EnableMetrics:    req.EnableMetrics,
		SlotSavePath:     req.SlotSavePath,
		CacheRAM:         req.CacheRAM,
		ChatTemplateFile: req.ChatTemplateFile,
		Timeout:          req.Timeout,
		Alias:            req.Alias,
		UBatchSize:       req.UBatchSize,
		ParallelSlots:    req.ParallelSlots,
		KVCacheTypeK:     req.KVCacheTypeK,
		KVCacheTypeV:     req.KVCacheTypeV,
		KVCacheUnified:   req.KVCacheUnified,
		KVCacheSize:      req.KVCacheSize,
		// Additional sampling parameters
		LogitsAll:        req.LogitsAll,
		Reranking:        req.Reranking,
		MinP:             req.MinP,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
		// Template and processing
		DirectIo:         req.DirectIo,
		DisableJinja:     req.DisableJinja,
		ChatTemplate:     req.ChatTemplate,
		ContextShift:     req.ContextShift,
		// Thread configuration
		ThreadsBatch:     req.ThreadsBatch,
		// Extended sampling parameters
		RepeatLastN:      req.RepeatLastN,
		TypicalP:         req.TypicalP,
		IgnoreEOS:        req.IgnoreEOS,
		// Multi-GPU configuration
		SplitMode:        req.SplitMode,
		TensorSplit:      req.TensorSplit,
		// Server optimization
		ContBatching:     req.ContBatching,
		CachePrompt:      req.CachePrompt,
		// Structured generation
		Grammar:          req.Grammar,
		GrammarFile:      req.GrammarFile,
		// LoRA adapter support
		Lora:             req.Lora,
		LoraScaled:       req.LoraScaled,
		// Chat template kwargs
		ChatTemplateKwargs: req.ChatTemplateKwargs,
		// RoPE scaling
		RopeScaling:      req.RopeScaling,
		RopeScale:        req.RopeScale,
		RopeFreqBase:     req.RopeFreqBase,
		RopeFreqScale:    req.RopeFreqScale,
	}
}