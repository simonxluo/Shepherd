// Package paths provides API handlers for path configuration management.
package paths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/types"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler"
)

// Handler handles path configuration requests
type Handler struct {
	configManager *config.Manager
}

// NewHandler creates a new paths handler
func NewHandler(configManager *config.Manager) *Handler {
	return &Handler{
		configManager: configManager,
	}
}

// GetLlamaCppPaths returns all configured llama.cpp paths
func (h *Handler) GetLlamaCppPaths(c *gin.Context) {
	cfg := h.configManager.Get()
	paths := cfg.Llamacpp.Paths

	handler.Success(c, gin.H{
		"items": paths,
		"count": len(paths),
	})
}

// AddLlamaCppPath adds a new llama.cpp path
func (h *Handler) AddLlamaCppPath(c *gin.Context) {
	var req config.LlamacppPath
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}

	// Validate path
	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	// Normalize and validate path
	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath

	// Load current config
	cfg := h.configManager.Get()

	// Check for duplicate
	for _, p := range cfg.Llamacpp.Paths {
		if p.Path == req.Path {
			handler.Error(c, types.ErrConflict, "Path already exists")
			return
		}
	}

	// Add path
	cfg.Llamacpp.Paths = append(cfg.Llamacpp.Paths, req)

	// Save config
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Llama.cpp path added successfully",
		"added":   req,
		"count":   len(cfg.Llamacpp.Paths),
	})
}

// RemoveLlamaCppPath removes a llama.cpp path
func (h *Handler) RemoveLlamaCppPath(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		handler.BadRequest(c, "Path query parameter is required")
		return
	}

	// Load current config
	cfg := h.configManager.Get()

	// Find and remove
	found := false
	newPaths := make([]config.LlamacppPath, 0, len(cfg.Llamacpp.Paths))
	for _, p := range cfg.Llamacpp.Paths {
		if p.Path != path {
			newPaths = append(newPaths, p)
		} else {
			found = true
		}
	}

	if !found {
		handler.NotFound(c, "Path")
		return
	}

	cfg.Llamacpp.Paths = newPaths

	// Save config
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Llama.cpp path removed successfully",
		"removed": path,
		"count":   len(cfg.Llamacpp.Paths),
	})
}

// UpdateLlamaCppPath updates an existing llama.cpp path
func (h *Handler) UpdateLlamaCppPath(c *gin.Context) {
	var req struct {
		OriginalPath string `json:"originalPath"` // 原始路径，用于匹配（可选）
		Path         string `json:"path"`         // 新路径（必填）
		Name         string `json:"name"`         // 新名称（可选）
		Description  string `json:"description"`  // 新描述（可选）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	// Validate new path
	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	// Normalize and validate new path
	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}

	// Also normalize the original path if provided (for comparison)
	var normalizedOriginalPath string
	if req.OriginalPath != "" {
		normalizedOriginalPath, _ = h.validateAndNormalizePath(req.OriginalPath)
		// If normalization fails, use the original as-is for comparison
		if normalizedOriginalPath == "" {
			normalizedOriginalPath = req.OriginalPath
		}
	}

	// Load current config
	cfg := h.configManager.Get()

	// Find and update path
	found := false
	var updatedIndex = -1

	for i, p := range cfg.Llamacpp.Paths {
		// Normalize the existing path for comparison
		normalizedExistingPath, _ := h.validateAndNormalizePath(p.Path)
		if normalizedExistingPath == "" {
			normalizedExistingPath = p.Path
		}

		// Match by original path (if provided) - highest priority
		if req.OriginalPath != "" && normalizedExistingPath == normalizedOriginalPath {
			updatedIndex = i
			found = true
			break
		}

		// If no originalPath provided, try to match by name (if name provided)
		if req.OriginalPath == "" && req.Name != "" && p.Name == req.Name {
			updatedIndex = i
			found = true
			break
		}

		// Lowest priority: match by exact path (user might not be changing the path)
		if !found && normalizedExistingPath == normalizedPath {
			updatedIndex = i
			found = true
			break
		}
	}

	if !found {
		handler.NotFound(c, "Path not found")
		return
	}

	// Update the path entry
	cfg.Llamacpp.Paths[updatedIndex] = config.LlamacppPath{
		Path:        normalizedPath,
		Name:        req.Name,
		Description: req.Description,
	}

	// Save config
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Llama.cpp path updated successfully",
		"updated": cfg.Llamacpp.Paths,
	})
}

// TestLlamaCppPath tests if a llama.cpp path is valid
func (h *Handler) TestLlamaCppPath(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}

	// Validate path
	_, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.Success(c, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	handler.Success(c, gin.H{
		"valid":   true,
		"message": "Path is valid",
	})
}

// GetModelPaths returns all configured model paths
func (h *Handler) GetModelPaths(c *gin.Context) {
	cfg := h.configManager.Get()
	paths := cfg.Model.PathConfigs

	handler.Success(c, gin.H{
		"items": paths,
		"count": len(paths),
	})
}

// AddModelPath adds a new model path
func (h *Handler) AddModelPath(c *gin.Context) {
	var req config.ModelPath
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}

	// Validate path
	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	// Normalize and validate path
	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath

	// Load current config
	cfg := h.configManager.Get()

	// Check for duplicate
	for _, p := range cfg.Model.PathConfigs {
		if p.Path == req.Path {
			handler.Error(c, types.ErrConflict, "Path already exists")
			return
		}
	}

	// Add path
	cfg.Model.PathConfigs = append(cfg.Model.PathConfigs, req)

	// Save config
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Model path added successfully",
		"added":   req,
		"count":   len(cfg.Model.PathConfigs),
	})
}

// UpdateModelPath updates an existing model path
func (h *Handler) UpdateModelPath(c *gin.Context) {
	var req struct {
		OriginalPath string `json:"originalPath"` // 原始路径，用于匹配（可选）
		Path         string `json:"path"`         // 新路径（必填）
		Name         string `json:"name"`         // 新名称（可选）
		Description  string `json:"description"`  // 新描述（可选）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	// Validate path
	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	// Normalize and validate path
	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}

	// Load current config
	cfg := h.configManager.Get()

	// Find and update path
	found := false
	var updatedIndex = -1

	for i, p := range cfg.Model.PathConfigs {
		// Use the path directly for comparison (avoid normalization issues in tests)
		existingPath := p.Path

		// Match by original path (if provided) - highest priority
		if req.OriginalPath != "" && existingPath == req.OriginalPath {
			updatedIndex = i
			found = true
			break
		}

		// If no originalPath provided, try to match by name (if name provided)
		if req.OriginalPath == "" && req.Name != "" && p.Name == req.Name {
			updatedIndex = i
			found = true
			break
		}

		// Lowest priority: match by exact path (user might not be changing the path)
		if !found && existingPath == req.Path {
			updatedIndex = i
			found = true
			break
		}
	}

	if !found {
		handler.NotFound(c, "Path not found")
		return
	}

	// Update the path entry with normalized path
	cfg.Model.PathConfigs[updatedIndex] = config.ModelPath{
		Path:        normalizedPath,
		Name:        req.Name,
		Description: req.Description,
	}

	// Save config
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Model path updated successfully",
		"updated": cfg.Model.PathConfigs[updatedIndex],
	})
}

// RemoveModelPath removes a model path
func (h *Handler) RemoveModelPath(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		handler.BadRequest(c, "Path query parameter is required")
		return
	}

	// Load current config
	cfg := h.configManager.Get()

	// Find and remove
	found := false
	newPaths := make([]config.ModelPath, 0, len(cfg.Model.PathConfigs))
	for _, p := range cfg.Model.PathConfigs {
		if p.Path != path {
			newPaths = append(newPaths, p)
		} else {
			found = true
		}
	}

	if !found {
		handler.NotFound(c, "Path")
		return
	}

	cfg.Model.PathConfigs = newPaths

	// Save config
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Model path removed successfully",
		"removed": path,
		"count":   len(cfg.Model.PathConfigs),
	})
}

// validateAndNormalizePath validates and normalizes a directory path
func (h *Handler) validateAndNormalizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", absPath)
		}
		return "", fmt.Errorf("failed to access path: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		realPath, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve symlink: %w", err)
		}
		absPath = realPath

		info, err = os.Stat(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to access resolved path: %w", err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("resolved path is not a directory: %s", absPath)
		}
	}

	return filepath.Clean(absPath), nil
}

// GetMultimodalPaths returns all configured multimodal model paths
func (h *Handler) GetMultimodalPaths(c *gin.Context) {
	cfg := h.configManager.Get()
	paths := cfg.Backends.MultimodalPaths
	if paths == nil {
		paths = []config.MultimodalPath{}
	}
	handler.Success(c, gin.H{
		"items": paths,
		"count": len(paths),
	})
}

// AddMultimodalPath adds a new multimodal model path
func (h *Handler) AddMultimodalPath(c *gin.Context) {
	var req config.MultimodalPath
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}

	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath

	cfg := h.configManager.Get()

	for _, p := range cfg.Backends.MultimodalPaths {
		if p.Path == req.Path {
			handler.Error(c, types.ErrConflict, "Path already exists")
			return
		}
	}

	cfg.Backends.MultimodalPaths = append(cfg.Backends.MultimodalPaths, req)

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Multimodal path added successfully",
		"added":   req,
		"count":   len(cfg.Backends.MultimodalPaths),
	})
}

// UpdateMultimodalPath updates an existing multimodal model path
func (h *Handler) UpdateMultimodalPath(c *gin.Context) {
	var req struct {
		OriginalPath string `json:"originalPath"`
		Path         string `json:"path"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		Backend      string `json:"backend"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}

	cfg := h.configManager.Get()

	found := false
	updatedIndex := -1
	for i, p := range cfg.Backends.MultimodalPaths {
		if req.OriginalPath != "" && p.Path == req.OriginalPath {
			updatedIndex = i
			found = true
			break
		}
		if !found && p.Path == req.Path {
			updatedIndex = i
			found = true
			break
		}
	}

	if !found {
		handler.NotFound(c, "Path not found")
		return
	}

	cfg.Backends.MultimodalPaths[updatedIndex] = config.MultimodalPath{
		Path:        normalizedPath,
		Name:        req.Name,
		Description: req.Description,
		Backend:     req.Backend,
	}

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Multimodal path updated successfully",
		"updated": cfg.Backends.MultimodalPaths[updatedIndex],
	})
}

// RemoveMultimodalPath removes a multimodal model path
func (h *Handler) RemoveMultimodalPath(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		handler.BadRequest(c, "Path query parameter is required")
		return
	}

	cfg := h.configManager.Get()

	found := false
	newPaths := make([]config.MultimodalPath, 0, len(cfg.Backends.MultimodalPaths))
	for _, p := range cfg.Backends.MultimodalPaths {
		if p.Path != path {
			newPaths = append(newPaths, p)
		} else {
			found = true
		}
	}

	if !found {
		handler.NotFound(c, "Path")
		return
	}

	cfg.Backends.MultimodalPaths = newPaths

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "Multimodal path removed successfully",
		"removed": path,
		"count":   len(cfg.Backends.MultimodalPaths),
	})
}

// GetVLLMPaths returns all configured vLLM paths
func (h *Handler) GetVLLMPaths(c *gin.Context) {
	cfg := h.configManager.Get()
	if cfg.Backends.VLLM == nil {
		handler.Success(c, gin.H{"items": []config.BackendPath{}, "count": 0})
		return
	}
	paths := cfg.Backends.VLLM.Paths
	if paths == nil {
		paths = []config.BackendPath{}
	}
	handler.Success(c, gin.H{"items": paths, "count": len(paths)})
}

// AddVLLMPath adds a new vLLM path
func (h *Handler) AddVLLMPath(c *gin.Context) {
	var req config.BackendPath
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}

	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath

	cfg := h.configManager.Get()
	if cfg.Backends.VLLM == nil {
		cfg.Backends.VLLM = &config.VLLMBackendConfig{Enabled: true}
	}

	for _, p := range cfg.Backends.VLLM.Paths {
		if p.Path == req.Path {
			handler.Error(c, types.ErrConflict, "Path already exists")
			return
		}
	}

	cfg.Backends.VLLM.Paths = append(cfg.Backends.VLLM.Paths, req)

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "vLLM path added successfully",
		"added":   req,
		"count":   len(cfg.Backends.VLLM.Paths),
	})
}

// UpdateVLLMPath updates an existing vLLM path
func (h *Handler) UpdateVLLMPath(c *gin.Context) {
	var req struct {
		OriginalPath string `json:"originalPath"`
		config.BackendPath
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}

	cfg := h.configManager.Get()
	if cfg.Backends.VLLM == nil {
		handler.NotFound(c, "vLLM not configured")
		return
	}

	found := false
	updatedIndex := -1
	for i, p := range cfg.Backends.VLLM.Paths {
		if req.OriginalPath != "" && p.Path == req.OriginalPath {
			updatedIndex = i
			found = true
			break
		}
		if !found && p.Path == req.Path {
			updatedIndex = i
			found = true
			break
		}
	}

	if !found {
		handler.NotFound(c, "Path not found")
		return
	}

	cfg.Backends.VLLM.Paths[updatedIndex] = config.BackendPath{
		Path:        normalizedPath,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "vLLM path updated successfully",
		"updated": cfg.Backends.VLLM.Paths[updatedIndex],
	})
}

// RemoveVLLMPath removes a vLLM path
func (h *Handler) RemoveVLLMPath(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		handler.BadRequest(c, "Path query parameter is required")
		return
	}

	cfg := h.configManager.Get()
	if cfg.Backends.VLLM == nil {
		handler.NotFound(c, "vLLM not configured")
		return
	}

	found := false
	newPaths := make([]config.BackendPath, 0, len(cfg.Backends.VLLM.Paths))
	for _, p := range cfg.Backends.VLLM.Paths {
		if p.Path != path {
			newPaths = append(newPaths, p)
		} else {
			found = true
		}
	}

	if !found {
		handler.NotFound(c, "Path")
		return
	}

	cfg.Backends.VLLM.Paths = newPaths

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "vLLM path removed successfully",
		"removed": path,
		"count":   len(cfg.Backends.VLLM.Paths),
	})
}

// GetVLLMOmniPaths returns all configured vLLM-Omni paths
func (h *Handler) GetVLLMOmniPaths(c *gin.Context) {
	cfg := h.configManager.Get()
	if cfg.Backends.VLLMOmni == nil {
		handler.Success(c, gin.H{"items": []config.BackendPath{}, "count": 0})
		return
	}
	paths := cfg.Backends.VLLMOmni.Paths
	if paths == nil {
		paths = []config.BackendPath{}
	}
	handler.Success(c, gin.H{"items": paths, "count": len(paths)})
}

// AddVLLMOmniPath adds a new vLLM-Omni path
func (h *Handler) AddVLLMOmniPath(c *gin.Context) {
	var req config.BackendPath
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}

	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath

	cfg := h.configManager.Get()
	if cfg.Backends.VLLMOmni == nil {
		cfg.Backends.VLLMOmni = &config.VLLMBackendConfig{Enabled: true}
	}

	for _, p := range cfg.Backends.VLLMOmni.Paths {
		if p.Path == req.Path {
			handler.Error(c, types.ErrConflict, "Path already exists")
			return
		}
	}

	cfg.Backends.VLLMOmni.Paths = append(cfg.Backends.VLLMOmni.Paths, req)

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "vLLM-Omni path added successfully",
		"added":   req,
		"count":   len(cfg.Backends.VLLMOmni.Paths),
	})
}

// UpdateVLLMOmniPath updates an existing vLLM-Omni path
func (h *Handler) UpdateVLLMOmniPath(c *gin.Context) {
	var req struct {
		OriginalPath string `json:"originalPath"`
		config.BackendPath
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}

	cfg := h.configManager.Get()
	if cfg.Backends.VLLMOmni == nil {
		handler.NotFound(c, "vLLM-Omni not configured")
		return
	}

	found := false
	updatedIndex := -1
	for i, p := range cfg.Backends.VLLMOmni.Paths {
		if req.OriginalPath != "" && p.Path == req.OriginalPath {
			updatedIndex = i
			found = true
			break
		}
		if !found && p.Path == req.Path {
			updatedIndex = i
			found = true
			break
		}
	}

	if !found {
		handler.NotFound(c, "Path not found")
		return
	}

	cfg.Backends.VLLMOmni.Paths[updatedIndex] = config.BackendPath{
		Path:        normalizedPath,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "vLLM-Omni path updated successfully",
		"updated": cfg.Backends.VLLMOmni.Paths[updatedIndex],
	})
}

// RemoveVLLMOmniPath removes a vLLM-Omni path
func (h *Handler) RemoveVLLMOmniPath(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		handler.BadRequest(c, "Path query parameter is required")
		return
	}

	cfg := h.configManager.Get()
	if cfg.Backends.VLLMOmni == nil {
		handler.NotFound(c, "vLLM-Omni not configured")
		return
	}

	found := false
	newPaths := make([]config.BackendPath, 0, len(cfg.Backends.VLLMOmni.Paths))
	for _, p := range cfg.Backends.VLLMOmni.Paths {
		if p.Path != path {
			newPaths = append(newPaths, p)
		} else {
			found = true
		}
	}

	if !found {
		handler.NotFound(c, "Path")
		return
	}

	cfg.Backends.VLLMOmni.Paths = newPaths

	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": "vLLM-Omni path removed successfully",
		"removed": path,
		"count":   len(cfg.Backends.VLLMOmni.Paths),
	})
}
