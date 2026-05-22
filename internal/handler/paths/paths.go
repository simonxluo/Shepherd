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
	h.llamacpp.remove(c, h.validateAndNormalizePath)
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
	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath
	h.llamacpp.add(c, &req, h.validateAndNormalizePath)
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
	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}

	var normalizedOriginalPath string
	if req.OriginalPath != "" {
		normalizedOriginalPath, _ = h.validateAndNormalizePath(req.OriginalPath)
		if normalizedOriginalPath == "" {
			normalizedOriginalPath = req.OriginalPath
		}
	}

	cfg := h.configManager.Get()
	found := false
	updatedIndex := -1
	for i, p := range cfg.Llamacpp.Paths {
		normalizedExistingPath, _ := h.validateAndNormalizePath(p.Path)
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

// TestLlamaCppPath tests if a llama.cpp path is valid.
func (h *Handler) TestLlamaCppPath(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "Invalid request body")
		return
	}
	_, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.Success(c, gin.H{"valid": false, "error": err.Error()})
		return
	}
	handler.Success(c, gin.H{"valid": true, "message": "Path is valid"})
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
func (h *Handler) RemoveVLLMPath(c *gin.Context) { h.vllm.remove(c, h.validateAndNormalizePath) }

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
	h.vllm.add(c, &req, h.validateAndNormalizePath)
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
		Path: normalizedPath, Name: req.Name, Description: req.Description,
	}
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}
	handler.Success(c, gin.H{"message": "vLLM path updated successfully", "updated": cfg.Backends.VLLM.Paths[updatedIndex]})
}

// --- vLLM-Omni paths ---

func (h *Handler) GetVLLMOmniPaths(c *gin.Context) { h.vllmOmni.list(c) }
func (h *Handler) RemoveVLLMOmniPath(c *gin.Context) {
	h.vllmOmni.remove(c, h.validateAndNormalizePath)
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
	normalizedPath, err := h.validateAndNormalizePath(req.Path)
	if err != nil {
		handler.BadRequest(c, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	req.Path = normalizedPath
	h.vllmOmni.add(c, &req, h.validateAndNormalizePath)
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
		Path: normalizedPath, Name: req.Name, Description: req.Description,
	}
	if err := h.configManager.Save(cfg); err != nil {
		handler.ErrorWithDetails(c, types.ErrInternalError, "Failed to save config", err.Error())
		return
	}
	handler.Success(c, gin.H{"message": "vLLM-Omni path updated successfully", "updated": cfg.Backends.VLLMOmni.Paths[updatedIndex]})
}

// --- Shared helpers ---

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
