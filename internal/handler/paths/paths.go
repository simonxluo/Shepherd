// Package paths provides API handlers for path configuration management.
package paths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/types"
	"github.com/simonxluo/Shepherd/internal/handler"
	"github.com/simonxluo/Shepherd/internal/service/model/backend"
)

// pathCRUD provides generic CRUD operations for path configuration slices.
type pathCRUD[T any] struct {
	configMgr *config.Manager
	getter    func(*config.Config) []T
	setter    func(*config.Config, []T)
	getPath   func(*T) string
	name      string // human-readable name for messages
}

func (pc *pathCRUD[T]) list(c *gin.Context) {
	cfg := pc.configMgr.Get()
	items := pc.getter(cfg)
	if items == nil {
		items = make([]T, 0)
	}
	handler.Success(c, gin.H{
		"items": items,
		"count": len(items),
	})
}

func (pc *pathCRUD[T]) add(c *gin.Context, req *T, validate func(string) (string, error)) {
	path := pc.getPath(req)
	if path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}

	normalizedPath, err := validate(path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	// Update the path in the request via pointer (caller handles this)
	_ = normalizedPath

	cfg := pc.configMgr.Get()
	items := pc.getter(cfg)

	for i := range items {
		if pc.getPath(&items[i]) == normalizedPath {
			handler.Error(c, types.ErrConflict, "Path already exists")
			return
		}
	}

	items = append(items, *req)
	pc.setter(cfg, items)

	if err := pc.configMgr.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": pc.name + " path added successfully",
		"added":   req,
		"count":   len(items),
	})
}

func (pc *pathCRUD[T]) remove(c *gin.Context, validate func(string) (string, error)) {
	path := c.Query("path")
	if path == "" {
		handler.BadRequest(c, "Path query parameter is required")
		return
	}

	cfg := pc.configMgr.Get()
	items := pc.getter(cfg)

	found := false
	newItems := make([]T, 0, len(items))
	for i := range items {
		if pc.getPath(&items[i]) != path {
			newItems = append(newItems, items[i])
		} else {
			found = true
		}
	}

	if !found {
		handler.NotFound(c, "Path")
		return
	}

	pc.setter(cfg, newItems)

	if err := pc.configMgr.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}

	handler.Success(c, gin.H{
		"message": pc.name + " path removed successfully",
		"removed": path,
		"count":   len(newItems),
	})
}

// Handler handles path configuration requests
type Handler struct {
	configManager *config.Manager
	llamacpp      *pathCRUD[config.LlamacppPath]
	model         *pathCRUD[config.ModelPath]
	multimodal    *pathCRUD[config.MultimodalPath]
	vllm          *pathCRUD[config.BackendPath]
	vllmOmni      *pathCRUD[config.BackendPath]
}

// NewHandler creates a new paths handler
func NewHandler(configManager *config.Manager) *Handler {
	h := &Handler{
		configManager: configManager,
	}
	h.llamacpp = &pathCRUD[config.LlamacppPath]{
		configMgr: configManager,
		getter:    func(cfg *config.Config) []config.LlamacppPath { return cfg.Llamacpp.Paths },
		setter:    func(cfg *config.Config, v []config.LlamacppPath) { cfg.Llamacpp.Paths = v },
		getPath:   func(p *config.LlamacppPath) string { return p.Path },
		name:      "Llama.cpp",
	}
	h.model = &pathCRUD[config.ModelPath]{
		configMgr: configManager,
		getter:    func(cfg *config.Config) []config.ModelPath { return cfg.Model.PathConfigs },
		setter:    func(cfg *config.Config, v []config.ModelPath) { cfg.Model.PathConfigs = v },
		getPath:   func(p *config.ModelPath) string { return p.Path },
		name:      "Model",
	}
	h.multimodal = &pathCRUD[config.MultimodalPath]{
		configMgr: configManager,
		getter: func(cfg *config.Config) []config.MultimodalPath {
			return cfg.Backends.MultimodalPaths
		},
		setter: func(cfg *config.Config, v []config.MultimodalPath) {
			cfg.Backends.MultimodalPaths = v
		},
		getPath: func(p *config.MultimodalPath) string { return p.Path },
		name:    "Multimodal",
	}
	h.vllm = &pathCRUD[config.BackendPath]{
		configMgr: configManager,
		getter: func(cfg *config.Config) []config.BackendPath {
			if cfg.Backends.VLLM == nil {
				return nil
			}
			return cfg.Backends.VLLM.Paths
		},
		setter: func(cfg *config.Config, v []config.BackendPath) {
			if cfg.Backends.VLLM == nil {
				cfg.Backends.VLLM = &config.VLLMBackendConfig{Enabled: true}
			}
			cfg.Backends.VLLM.Paths = v
		},
		getPath: func(p *config.BackendPath) string { return p.Path },
		name:    "vLLM",
	}
	h.vllmOmni = &pathCRUD[config.BackendPath]{
		configMgr: configManager,
		getter: func(cfg *config.Config) []config.BackendPath {
			if cfg.Backends.VLLMOmni == nil {
				return nil
			}
			return cfg.Backends.VLLMOmni.Paths
		},
		setter: func(cfg *config.Config, v []config.BackendPath) {
			if cfg.Backends.VLLMOmni == nil {
				cfg.Backends.VLLMOmni = &config.VLLMBackendConfig{Enabled: true}
			}
			cfg.Backends.VLLMOmni.Paths = v
		},
		getPath: func(p *config.BackendPath) string { return p.Path },
		name:    "vLLM-Omni",
	}
	return h
}

// --- LlamaCpp paths ---

func (h *Handler) GetLlamaCppPaths(c *gin.Context) { h.llamacpp.list(c) }
func (h *Handler) RemoveLlamaCppPath(c *gin.Context) {
	h.llamacpp.remove(c, h.validateAndNormalizeBinaryPath)
}

func (h *Handler) AddLlamaCppPath(c *gin.Context) {
	var req config.LlamacppPath
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}
	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath
	h.llamacpp.add(c, &req, h.validateAndNormalizeBinaryPath)
}

func (h *Handler) UpdateLlamaCppPath(c *gin.Context) {
	var req struct {
		OriginalPath string `json:"originalPath"`
		Path         string `json:"path"`
		Name         string `json:"name"`
		Description  string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}
	if req.Path == "" {
		handler.BadRequest(c, "Path is required")
		return
	}
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}

	var normalizedOriginalPath string
	if req.OriginalPath != "" {
		normalizedOriginalPath, _ = h.validateAndNormalizeBinaryPath(req.OriginalPath)
		if normalizedOriginalPath == "" {
			normalizedOriginalPath = req.OriginalPath
		}
	}

	cfg := h.configManager.Get()
	found := false
	updatedIndex := -1
	for i, p := range cfg.Llamacpp.Paths {
		normalizedExistingPath, _ := h.validateAndNormalizeBinaryPath(p.Path)
		if normalizedExistingPath == "" {
			normalizedExistingPath = p.Path
		}
		if req.OriginalPath != "" && normalizedExistingPath == normalizedOriginalPath {
			updatedIndex = i
			found = true
			break
		}
		if req.OriginalPath == "" && req.Name != "" && p.Name == req.Name {
			updatedIndex = i
			found = true
			break
		}
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
	cfg.Llamacpp.Paths[updatedIndex] = config.LlamacppPath{
		Path: normalizedPath, Name: req.Name, Description: req.Description,
	}
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}
	handler.Success(c, gin.H{"message": "Llama.cpp path updated successfully", "updated": cfg.Llamacpp.Paths})
}

// TestLlamaCppPath tests if a llama.cpp path is valid and contains the required binary.
func (h *Handler) TestLlamaCppPath(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
	if err != nil {
		handler.Success(c, gin.H{"valid": false, "error": err.Error()})
		return
	}
	probe, err := backend.ProbeLlamaCppInstallation(normalizedPath)
	if err != nil {
		handler.Success(c, gin.H{
			"valid":    false,
			"error":    err.Error(),
			"path":     normalizedPath,
			"warnings": []string{err.Error()},
		})
		return
	}
	if !probe.Available {
		errorMessage := fmt.Sprintf("llama-server not found in path: %s (looked for llama-server, server in the directory and bin/, build/bin/ subdirectories)", normalizedPath)
		if len(probe.Warnings) > 0 {
			errorMessage = probe.Warnings[0]
		}
		handler.Success(c, gin.H{
			"valid":    false,
			"error":    errorMessage,
			"path":     normalizedPath,
			"binary":   probe.Binary,
			"version":  probe.Version,
			"warnings": probe.Warnings,
		})
		return
	}
	handler.Success(c, gin.H{
		"valid":    true,
		"message":  "Path is valid",
		"binary":   probe.Binary,
		"version":  probe.Version,
		"warnings": probe.Warnings,
		"path":     normalizedPath,
	})
}

// --- Model paths ---

func (h *Handler) GetModelPaths(c *gin.Context)   { h.model.list(c) }
func (h *Handler) RemoveModelPath(c *gin.Context) { h.model.remove(c, h.validateAndNormalizePath) }

func (h *Handler) AddModelPath(c *gin.Context) {
	var req config.ModelPath
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
	h.model.add(c, &req, h.validateAndNormalizePath)
}

func (h *Handler) UpdateModelPath(c *gin.Context) {
	var req struct {
		OriginalPath string `json:"originalPath"`
		Path         string `json:"path"`
		Name         string `json:"name"`
		Description  string `json:"description"`
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
	for i, p := range cfg.Model.PathConfigs {
		if req.OriginalPath != "" && p.Path == req.OriginalPath {
			updatedIndex = i
			found = true
			break
		}
		if req.OriginalPath == "" && req.Name != "" && p.Name == req.Name {
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
	cfg.Model.PathConfigs[updatedIndex] = config.ModelPath{
		Path: normalizedPath, Name: req.Name, Description: req.Description,
	}
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}
	handler.Success(c, gin.H{"message": "Model path updated successfully", "updated": cfg.Model.PathConfigs[updatedIndex]})
}

// --- Multimodal paths ---

func (h *Handler) GetMultimodalPaths(c *gin.Context) { h.multimodal.list(c) }
func (h *Handler) RemoveMultimodalPath(c *gin.Context) {
	h.multimodal.remove(c, h.validateAndNormalizePath)
}

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
	h.multimodal.add(c, &req, h.validateAndNormalizePath)
}

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
		Path: normalizedPath, Name: req.Name, Description: req.Description, Backend: req.Backend,
	}
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}
	handler.Success(c, gin.H{"message": "Multimodal path updated successfully", "updated": cfg.Backends.MultimodalPaths[updatedIndex]})
}

// --- vLLM paths ---

func (h *Handler) GetVLLMPaths(c *gin.Context)   { h.vllm.list(c) }
func (h *Handler) RemoveVLLMPath(c *gin.Context) { h.vllm.remove(c, h.validateAndNormalizeBinaryPath) }

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
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath
	h.vllm.add(c, &req, h.validateAndNormalizeBinaryPath)
}

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
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
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
		Path: normalizedPath, Name: req.Name, Description: req.Description,
	}
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}
	handler.Success(c, gin.H{"message": "vLLM path updated successfully", "updated": cfg.Backends.VLLM.Paths[updatedIndex]})
}

// TestVLLMPath tests if a vLLM path is valid and contains the vllm binary.
func (h *Handler) TestVLLMPath(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
	if err != nil {
		handler.Success(c, gin.H{"valid": false, "error": err.Error()})
		return
	}
	// Check if it's a directory containing 'vllm' or is a vllm binary itself
	binary := findVLLMBinary(normalizedPath, "vllm")
	if binary == "" {
		handler.Success(c, gin.H{
			"valid": false,
			"error": fmt.Sprintf("vllm binary not found at path: %s (looked for 'vllm' executable in directory or the file itself)", normalizedPath),
			"path":  normalizedPath,
		})
		return
	}
	handler.Success(c, gin.H{"valid": true, "message": "Path is valid", "binary": binary, "path": normalizedPath})
}

// --- vLLM-Omni paths ---

func (h *Handler) GetVLLMOmniPaths(c *gin.Context) { h.vllmOmni.list(c) }
func (h *Handler) RemoveVLLMOmniPath(c *gin.Context) {
	h.vllmOmni.remove(c, h.validateAndNormalizeBinaryPath)
}

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
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath
	h.vllmOmni.add(c, &req, h.validateAndNormalizeBinaryPath)
}

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
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
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
		Path: normalizedPath, Name: req.Name, Description: req.Description,
	}
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}
	handler.Success(c, gin.H{"message": "vLLM-Omni path updated successfully", "updated": cfg.Backends.VLLMOmni.Paths[updatedIndex]})
}

// TestVLLMOmniPath tests if a vLLM-Omni path is valid and contains the vllm-omni binary.
func (h *Handler) TestVLLMOmniPath(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}
	normalizedPath, err := h.validateAndNormalizeBinaryPath(req.Path)
	if err != nil {
		handler.Success(c, gin.H{"valid": false, "error": err.Error()})
		return
	}
	// Check if it's a directory containing 'vllm-omni' or is a vllm-omni binary itself
	binary := findVLLMBinary(normalizedPath, "vllm-omni")
	if binary == "" {
		handler.Success(c, gin.H{
			"valid": false,
			"error": fmt.Sprintf("vllm-omni binary not found at path: %s (looked for 'vllm-omni' executable in directory or the file itself)", normalizedPath),
			"path":  normalizedPath,
		})
		return
	}
	handler.Success(c, gin.H{"valid": true, "message": "Path is valid", "binary": binary, "path": normalizedPath})
}

// --- Shared helpers ---

// validateAndNormalizePath validates and normalizes a directory path.
// Used for model paths and multimodal paths that must be directories.
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

// validateAndNormalizeBinaryPath validates and normalizes a binary path.
// It accepts both:
// 1. A directory containing the relevant binary (e.g., /usr/local/bin/ containing llama-server)
// 2. A direct path to the executable file (e.g., /usr/local/bin/llama-server)
//
// For directories, only existence and accessibility are checked.
// For files, it verifies the file is executable.
func (h *Handler) validateAndNormalizeBinaryPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Resolve symlinks first
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", absPath)
		}
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	absPath = realPath

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", absPath)
		}
		return "", fmt.Errorf("failed to access path: %w", err)
	}

	if info.IsDir() {
		// Directory is acceptable — backend will search for binary inside it
		return filepath.Clean(absPath), nil
	}

	// It's a file — check if it's executable
	if info.Mode().IsRegular() {
		if info.Mode().Perm()&0111 == 0 {
			return "", fmt.Errorf("file is not executable: %s", absPath)
		}
		// Valid executable file — return its parent directory for consistency
		// with how the backend registry uses BinPaths (it joins dir + binary name).
		// However, we store the actual path the user provided (could be the file itself)
		// since FindLlamacppBinary and discoverVLLMVariant handle both cases.
		return filepath.Clean(absPath), nil
	}

	return "", fmt.Errorf("path is not a regular file or directory: %s", absPath)
}

// findVLLMBinary finds a vLLM variant binary at the given path.
// If path is a file and executable, it's returned directly.
// If path is a directory, it searches for the named binary inside it.
func findVLLMBinary(path string, binaryName string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	// If it's an executable file, return it directly
	if info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
		return path
	}

	// If it's a directory, look for the binary inside
	if info.IsDir() {
		candidate := filepath.Join(path, binaryName)
		if fi, err := os.Stat(candidate); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0111 != 0 {
			return candidate
		}
		// Also check bin/ subdirectory
		candidate = filepath.Join(path, "bin", binaryName)
		if fi, err := os.Stat(candidate); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0111 != 0 {
			return candidate
		}
	}

	return ""
}
