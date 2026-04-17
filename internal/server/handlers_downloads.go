// Package server provides download and model repository HTTP handler methods.
package server

import (
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	api "github.com/shepherd-project/shepherd/Shepherd/internal/handler"
	"github.com/shepherd-project/shepherd/Shepherd/internal/download"
	modelrepoclient "github.com/shepherd-project/shepherd/Shepherd/internal/modelrepo"
	"github.com/shepherd-project/shepherd/Shepherd/internal/types"
)

func (s *Server) HandleListDownloads(c *gin.Context) {
	tasks := s.downloadMgr.ListTasks()

	downloads := make([]gin.H, 0, len(tasks))
	for _, t := range tasks {
		downloads = append(downloads, mapDownloadTask(t))
	}

	api.Success(c, gin.H{
		"downloads": downloads,
		"total":     len(downloads),
	})
}

// mapDownloadTask transforms download.Task to the frontend DownloadTask format
func mapDownloadTask(t *download.Task) gin.H {
	progress := float64(0)
	if t.TotalBytes > 0 {
		progress = float64(t.DownloadedBytes) / float64(t.TotalBytes)
	}

	errStr := ""
	if t.Error != nil {
		errStr = t.Error.Error()
	}

	var completedAt string
	if !t.FinishedAt.IsZero() {
		completedAt = t.FinishedAt.Format(time.RFC3339)
	}

	result := gin.H{
		"id":              t.ID,
		"source":          t.SourceType,
		"repoId":          t.RepoID,
		"fileName":        t.FileName,
		"path":            t.Path,
		"state":           t.State.String(),
		"downloadedBytes": t.DownloadedBytes,
		"totalBytes":      t.TotalBytes,
		"partsCompleted":  t.PartsCompleted,
		"partsTotal":      t.PartsTotal,
		"progress":        progress,
		"speed":           t.Speed,
		"eta":             t.ETA,
		"error":           errStr,
		"createdAt":       t.CreatedAt.Format(time.RFC3339),
	}

	if completedAt != "" {
		result["completedAt"] = completedAt
	}

	return result
}

func (s *Server) HandleCreateDownload(c *gin.Context) {
	// 支持两种请求格式:
	// 1. 新格式: { source, repoId, fileName, path } - 用于从模型仓库下载
	// 2. 旧格式: { url, target_path } - 直接URL下载(向后兼容)

	var req struct {
		Source   modelrepoclient.Source `json:"source"`
		RepoID   string                 `json:"repoId"`
		FileName string                 `json:"fileName"`
		Path     string                 `json:"path"`

		// 旧格式参数(向后兼容)
		URL        string `json:"url"`
		TargetPath string `json:"target_path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "无效的请求格式: "+err.Error())
		return
	}

	var downloadURL string
	var downloadDir string
	var fileName string
	var source string
	var repoId string

	// 使用新格式(source + repoId)
	if req.Source != "" && req.RepoID != "" {
		// 生成下载 URL
		url, err := s.repoClient.GenerateDownloadURL(req.Source, req.RepoID, req.FileName)
		if err != nil {
			api.ErrorWithDetails(c, types.ErrInvalidRequest, "生成下载URL失败", err.Error())
			return
		}
		downloadURL = url
		source = string(req.Source)
		repoId = req.RepoID
		fileName = req.FileName

		// Determine directory
		if req.Path != "" {
			downloadDir = req.Path
		} else {
			// Default path logic
			cfg := s.config.ConfigMgr.Get()
			if len(cfg.Model.Paths) > 0 {
				downloadDir = cfg.Model.Paths[0]
			} else {
				downloadDir = "./models"
			}
		}
	} else if req.URL != "" {
		// 使用旧格式(直接URL)
		downloadURL = req.URL
		source = "url"
		downloadDir = filepath.Dir(req.TargetPath)
		fileName = filepath.Base(req.TargetPath)
	} else {
		api.BadRequest(c, "缺少必要参数: 请提供 source/repoId 或 url")
		return
	}

	taskId, err := s.downloadMgr.CreateTask(downloadURL, downloadDir, fileName, source, repoId)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "创建下载失败", err.Error())
		return
	}

	task, exists := s.downloadMgr.GetTask(taskId)
	if !exists {
		api.ErrorWithDetails(c, types.ErrInternalError, "创建下载失败", "无法获取任务信息")
		return
	}

	// Get request ID from context
	requestID := "unknown"
	if id := c.GetString("requestId"); id != "" {
		requestID = id
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponse(mapDownloadTask(task), requestID))
}

func (s *Server) HandleGetDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "下载ID不能为空")
		return
	}

	task, exists := s.downloadMgr.GetTask(id)
	if !exists {
		api.NotFound(c, "下载任务")
		return
	}

	api.Success(c, mapDownloadTask(task))
}

func (s *Server) HandlePauseDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "下载ID不能为空")
		return
	}

	if err := s.downloadMgr.Pause(id); err != nil {
		api.NotFound(c, err.Error())
		return
	}

	api.SuccessWithMessage(c, "下载已暂停")
}

func (s *Server) HandleResumeDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "下载ID不能为空")
		return
	}

	if err := s.downloadMgr.Resume(id); err != nil {
		api.NotFound(c, err.Error())
		return
	}

	api.SuccessWithMessage(c, "下载已恢复")
}

func (s *Server) HandleDeleteDownload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "下载ID不能为空")
		return
	}

	if err := s.downloadMgr.Delete(id); err != nil {
		api.NotFound(c, err.Error())
		return
	}

	api.SuccessWithMessage(c, "下载任务已删除")
}

// handleListModelFiles handles requests to list model files from a repository
func (s *Server) HandleListModelFiles(c *gin.Context) {
	// 使用查询参数而不是路径参数，以支持 repoId 中包含斜杠
	source := c.Query("source")
	repoID := c.Query("repoId")

	if source == "" || repoID == "" {
		api.BadRequest(c, "缺少必要参数: 需要 source 和 repoId 查询参数")
		return
	}

	// 目前只支持 HuggingFace
	if source != "huggingface" {
		api.BadRequest(c, "目前只支持 HuggingFace 源")
		return
	}

	files, err := s.repoClient.ListGGUFFiles(repoID)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "获取文件列表失败", err.Error())
		return
	}

	api.Success(c, files)
}

// handleSearchModels handles requests to search for models on HuggingFace
func (s *Server) HandleSearchModels(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		api.BadRequest(c, "缺少必要参数: 需要 q 查询参数")
		return
	}

	// Parse limit parameter (default 20)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Parse format filter parameter (e.g., "gguf", "safetensors", "onnx")
	format := c.Query("format")

	result, err := s.repoClient.SearchHuggingFaceModels(query, limit, format)
	if err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "搜索模型失败", err.Error())
		return
	}

	api.Success(c, result)
}

// handleGetModelRepoConfig returns the current model repository configuration
func (s *Server) HandleGetModelRepoConfig(c *gin.Context) {
	cfg := s.config.ConfigMgr.Get()
	api.Success(c, gin.H{
		"endpoint": cfg.ModelRepo.Endpoint,
		"token":    maskToken(cfg.ModelRepo.Token),
		"timeout":  cfg.ModelRepo.Timeout,
	})
}

// maskToken masks the token for security
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

// handleUpdateModelRepoConfig updates the model repository configuration
func (s *Server) HandleUpdateModelRepoConfig(c *gin.Context) {
	var req struct {
		Endpoint string `json:"endpoint"`
		Token    string `json:"token"`
		Timeout  int    `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "无效的请求数据")
		return
	}

	cfg := s.config.ConfigMgr.Get()

	// Update endpoint if provided
	if req.Endpoint != "" {
		cfg.ModelRepo.Endpoint = req.Endpoint
	}

	// Update token if provided (allow empty string to clear token)
	if req.Token != "" {
		cfg.ModelRepo.Token = req.Token
	}

	// Update timeout if provided
	if req.Timeout > 0 {
		cfg.ModelRepo.Timeout = req.Timeout
	}

	// Save config
	if err := s.config.ConfigMgr.Save(cfg); err != nil {
		api.ErrorWithDetails(c, types.ErrInternalError, "保存配置失败", err.Error())
		return
	}

	// Update the repo client with new settings
	timeout := time.Duration(cfg.ModelRepo.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	s.repoClient = modelrepoclient.NewClientWithConfig(cfg.ModelRepo.Endpoint, cfg.ModelRepo.Token, timeout)

	api.Success(c, gin.H{
		"endpoint": cfg.ModelRepo.Endpoint,
		"token":    maskToken(cfg.ModelRepo.Token),
		"timeout":  cfg.ModelRepo.Timeout,
	})
}

// handleGetAvailableEndpoints returns available HuggingFace endpoints
func (s *Server) HandleGetAvailableEndpoints(c *gin.Context) {
	endpoints := modelrepoclient.GetAvailableEndpoints()
	api.Success(c, endpoints)
}
