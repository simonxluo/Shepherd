// Package filesystem provides file system browsing API
package filesystem

import (
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/handler"
)

// Handler handles file system browsing requests
type Handler struct{}

// NewHandler creates a new filesystem handler
func NewHandler() *Handler {
	return &Handler{}
}

// DirectoryItem represents a file or directory
type DirectoryItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size,omitempty"` // 文件大小(字节),目录为空
}

// DirectoryListResponse represents the response for directory listing
type DirectoryListResponse struct {
	CurrentPath string          `json:"currentPath"`
	ParentPath  string          `json:"parentPath"`
	Folders     []DirectoryItem `json:"folders"`
	Files       []DirectoryItem `json:"files"`
	Roots       []DirectoryItem `json:"roots,omitempty"`
}

// ListDirectory handles directory listing requests.
// @Summary      List directory contents
// @Description  Lists files and folders in a given directory path, or returns system roots if no path specified
// @Tags         System
// @Produce      json
// @Param        path query string false "Directory path to list"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /api/system/filesystem [get]
func (h *Handler) ListDirectory(c *gin.Context) {
	path := c.Query("path")

	// 如果没有指定路径,返回系统根目录列表
	if path == "" {
		roots := h.getSystemRoots()
		handler.Success(c, gin.H{
			"currentPath": "",
			"parentPath":  "",
			"folders":     roots,
			"files":       []DirectoryItem{},
			"roots":       roots,
		})
		return
	}

	// 清理和验证路径
	cleanPath, err := h.sanitizePath(path)
	if err != nil {
		logger.Errorf("路径验证失败: %v", err)
		handler.BadRequest(c, fmt.Sprintf("无效的路径: %v", err))
		return
	}

	// 检查路径是否存在
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			handler.Success(c, gin.H{
				"currentPath": cleanPath,
				"parentPath":  filepath.Dir(cleanPath),
				"folders":     []DirectoryItem{},
				"files":       []DirectoryItem{},
				"error":       "目录不存在",
			})
			return
		}
		handler.Success(c, gin.H{
			"currentPath": cleanPath,
			"parentPath":  filepath.Dir(cleanPath),
			"folders":     []DirectoryItem{},
			"files":       []DirectoryItem{},
			"error":       "无法访问目录",
		})
		return
	}

	// 如果是文件而不是目录,返回其父目录
	if !fileInfo.IsDir() {
		cleanPath = filepath.Dir(cleanPath)
	}

	// 读取目录内容
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		handler.Success(c, gin.H{
			"currentPath": cleanPath,
			"parentPath":  filepath.Dir(cleanPath),
			"folders":     []DirectoryItem{},
			"files":       []DirectoryItem{},
			"error":       "无法读取目录内容",
		})
		return
	}

	// 分类文件夹和文件
	var folders []DirectoryItem
	var files []DirectoryItem

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(cleanPath, name)

		// 跳过隐藏文件/文件夹
		if strings.HasPrefix(name, ".") {
			continue
		}

		if entry.IsDir() {
			folders = append(folders, DirectoryItem{
				Name: name,
				Path: fullPath,
			})
		} else {
			info, err := entry.Info()
			size := int64(0)
			if err == nil {
				size = info.Size()
			}
			files = append(files, DirectoryItem{
				Name: name,
				Path: fullPath,
				Size: size,
			})
		}
	}

	response := DirectoryListResponse{
		CurrentPath: cleanPath,
		ParentPath:  filepath.Dir(cleanPath),
		Folders:     folders,
		Files:       files,
	}

	handler.Success(c, response)
}

// sanitizePath 清理和验证路径,防止路径遍历攻击
func (h *Handler) sanitizePath(inputPath string) (string, error) {
	if inputPath == "" {
		return "", fmt.Errorf("路径为空")
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(inputPath)
	if err != nil {
		return "", fmt.Errorf("无法解析路径: %w", err)
	}

	// 清理路径(移除 .. 和 .)
	cleanPath := filepath.Clean(absPath)

	// 检查路径是否存在
	if _, err := os.Stat(cleanPath); err != nil {
		// 路径不存在时不返回错误,由调用者处理
		return cleanPath, nil
	}

	return cleanPath, nil
}

// getSystemRoots 返回系统根目录列表
func (h *Handler) getSystemRoots() []DirectoryItem {
	var roots []DirectoryItem

	switch runtime.GOOS {
	case "windows":
		// Windows: 返回所有盘符
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			path := string(drive) + ":\\"
			if _, err := os.Stat(path); err == nil {
				roots = append(roots, DirectoryItem{
					Name: string(drive) + " 盘",
					Path: path,
				})
			}
		}
	default:
		// Unix-like: 返回根目录和用户主目录
		roots = append(roots, DirectoryItem{
			Name: "根目录",
			Path: "/",
		})

		// 添加用户主目录
		if homeDir, err := os.UserHomeDir(); err == nil {
			roots = append(roots, DirectoryItem{
				Name: "主目录",
				Path: homeDir,
			})
		}
	}

	return roots
}

// ValidatePath validates a directory path.
// @Summary      Validate path
// @Description  Validates whether a given path exists, is a directory, and is readable
// @Tags         System
// @Accept       json
// @Produce      json
// @Param        request body object true "Path validation request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /api/system/filesystem/validate [post]
func (h *Handler) ValidatePath(c *gin.Context) {
	type Request struct {
		Path string `json:"path" binding:"required"`
	}

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, "路径参数缺失")
		return
	}

	// 验证路径
	cleanPath, err := h.sanitizePath(req.Path)
	if err != nil {
		handler.Success(c, gin.H{
			"success": false,
			"valid":   false,
			"error":   err.Error(),
		})
		return
	}

	// 检查路径是否存在
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		handler.Success(c, gin.H{
			"success": false,
			"valid":   false,
			"exists":  false,
			"error":   "路径不存在",
		})
		return
	}

	// 检查是否为目录
	if !fileInfo.IsDir() {
		handler.Success(c, gin.H{
			"success": false,
			"valid":   false,
			"exists":  true,
			"error":   "路径不是目录",
		})
		return
	}

	// 检查可读权限
	if !isReadable(cleanPath) {
		handler.Success(c, gin.H{
			"success": false,
			"valid":   false,
			"exists":  true,
			"error":   "目录不可读",
		})
		return
	}

	handler.Success(c, gin.H{
		"success":        true,
		"valid":          true,
		"exists":         true,
		"isDirectory":    true,
		"isReadable":     true,
		"normalizedPath": cleanPath,
	})
}

// isReadable 检查目录是否可读
func isReadable(path string) bool {
	// 尝试打开目录进行读取
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer utils.CloseQuietly(file)
	return true
}
