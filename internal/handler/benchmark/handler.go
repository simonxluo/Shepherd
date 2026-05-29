// Package benchmark provides pressure testing API handlers
package benchmark

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/simonxluo/Shepherd/internal/comm/event"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/infra/taskmanager"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

const (
	// MaxConcurrentBenchmarks 最大并发压测任务数
	MaxConcurrentBenchmarks = 3
)

// Handler is the benchmark API handler.
type Handler struct {
	log        *logger.Logger
	store      storage.Store
	modelMgr   *model.Manager
	taskMgr    *taskmanager.Manager
	eventMgr   *event.Manager
	ctx        context.Context
	cancelFunc context.CancelFunc
	semaphore  chan struct{} // 用于限制并发数
}

// NewHandler creates a new benchmark handler.
func NewHandler(log *logger.Logger, store storage.Store, modelMgr *model.Manager, taskMgr *taskmanager.Manager, eventMgr *event.Manager) *Handler {
	ctx, cancel := context.WithCancel(context.Background())

	h := &Handler{
		log:        log,
		store:      store,
		modelMgr:   modelMgr,
		taskMgr:    taskMgr,
		eventMgr:   eventMgr,
		ctx:        ctx,
		cancelFunc: cancel,
		semaphore:  make(chan struct{}, MaxConcurrentBenchmarks),
	}

	// 启动清理 goroutine
	go h.cleanupFinishedTasks()

	return h
}

// BenchmarkParam defines a benchmark parameter.
type BenchmarkParam struct {
	FullName       string      `json:"fullName"`
	Name           string      `json:"name"`
	Abbreviation   string      `json:"abbreviation"`
	Description    string      `json:"description"`
	DefaultValue   string      `json:"defaultValue"`
	DefaultEnabled *bool       `json:"defaultEnabled,omitempty"`
	Type           string      `json:"type"`
	Values         interface{} `json:"values,omitempty"`
	Sort           int         `json:"sort"`
	Group          string      `json:"group,omitempty"`
	GroupOrder     int         `json:"groupOrder,omitempty"`
	GroupCollapsed bool        `json:"groupCollapsed,omitempty"`
}

// BenchmarkConfig defines benchmark configuration.
type BenchmarkConfig struct {
	ModelID      string            `json:"modelId"`
	ModelName    string            `json:"modelName"`
	LlamaCppPath string            `json:"llamaCppPath"`
	Devices      []string          `json:"devices"`
	Params       map[string]string `json:"params"`
	CreatedAt    string            `json:"createdAt"`
}

// cleanupFinishedTasks periodically cleans up finished tasks.
func (h *Handler) cleanupFinishedTasks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			tasks := h.taskMgr.List(taskmanager.TaskTypeBenchmark)
			for _, t := range tasks {
				if time.Since(t.CreatedAt) > time.Hour && t.GetStatus() != taskmanager.TaskStatusRunning {
					h.taskMgr.Remove(t.ID)
					h.log.Infof("Cleaned up old benchmark task %s", t.ID)
				}
			}
		}
	}
}

// isValidLlamaBinary validates that a llama.cpp binary is safe to execute.
func (h *Handler) isValidLlamaBinary(path string) error {
	// 检查路径是否为绝对路径
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute")
	}

	// 清理路径，防止路径遍历攻击
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("path contains directory traversal components")
	}

	// 检查文件是否存在
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// 检查是否为常规文件
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	// 检查是否可执行
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("file is not executable")
	}

	// 可选：检查文件名是否包含预期的名称（如 llama-server, main, llama-bench 等）
	baseName := strings.ToLower(filepath.Base(path))
	validNames := []string{"llama-server", "llama-cli", "llama-bench", "main", "llama-model-runner"}
	isValidName := false
	for _, name := range validNames {
		if strings.Contains(baseName, name) {
			isValidName = true
			break
		}
	}
	if !isValidName {
		h.log.Warnf("Binary name %q does not match expected llama.cpp binaries", baseName)
	}

	return nil
}

// findLlamaCli finds the llama-cli executable in the specified path.
// Uses the unified utility function utils.FindLlamacppBinary.
func (h *Handler) findLlamaCli(llamaBinPath string) string {
	return utils.FindLlamacppBinary(llamaBinPath, "cli")
}

// validatePathForDevices validates a path for device list queries.
// Accepts directory paths or executable file paths.
// Uses the unified utility function to find the binary.
func (h *Handler) validatePathForDevices(path string) error {
	// 清理路径
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("path contains directory traversal components")
	}

	// 检查路径是否存在
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	// 如果是文件，检查是否为常规文件
	if info.Mode().IsRegular() {
		return nil
	}

	// 如果是目录，使用统一的工具函数检查是否包含可执行文件
	if info.IsDir() {
		// 尝试查找 llama-cli（用于设备列表）
		if cliPath := utils.FindLlamacppBinary(path, "cli"); cliPath != "" {
			return nil
		}
		// 尝试查找 llama-server（用于启动服务）
		if serverPath := utils.FindLlamacppBinary(path, "server"); serverPath != "" {
			return nil
		}
		return fmt.Errorf("directory does not contain llama-cli or llama-server executable (checked: %s, %s/bin)",
			path, filepath.Base(path))
	}

	return fmt.Errorf("invalid path type")
}

// GetParams returns the list of available benchmark parameters.
// @Summary      Get benchmark parameters
// @Description  Returns the list of available llama.cpp benchmark parameters with their defaults
// @Tags         Benchmark
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/models/param/benchmark/list [get]
func (h *Handler) GetParams(c *gin.Context) {
	// Try to load from config/benchmark-params.json
	params, err := h.loadBenchmarkParams()
	if err != nil {
		h.log.Warnf("Failed to load benchmark params from file, using defaults: %v", err)
		params = h.getDefaultParams()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"params":  params,
	})
}

func (h *Handler) loadBenchmarkParams() ([]BenchmarkParam, error) {
	// Look for benchmark-params.json in multiple locations
	possiblePaths := []string{
		"config/benchmark-params.json",
		filepath.Join("config", "benchmark-params.json"),
	}

	// Also check relative to executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		possiblePaths = append(possiblePaths, filepath.Join(execDir, "config", "benchmark-params.json"))
	}

	var data []byte
	var readErr error
	for _, p := range possiblePaths {
		data, readErr = os.ReadFile(p)
		if readErr == nil {
			break
		}
	}
	if readErr != nil {
		return nil, fmt.Errorf("benchmark-params.json not found in any expected location: %w", readErr)
	}

	var params []BenchmarkParam
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("failed to parse benchmark-params.json: %w", err)
	}

	return params, nil
}

func (h *Handler) getDefaultParams() []BenchmarkParam {
	return []BenchmarkParam{
		{
			FullName:     "--n-prompt",
			Name:         "Prompt Tokens",
			Abbreviation: "-p",
			Description:  "Number of prompt tokens",
			DefaultValue: "512",
			Type:         "STRING",
			Sort:         10,
			Group:        "page.params.group.test_data",
			GroupOrder:   20,
		},
		{
			FullName:     "--n-gen",
			Name:         "Generation Tokens",
			Abbreviation: "-n",
			Description:  "Number of tokens to generate",
			DefaultValue: "128",
			Type:         "STRING",
			Sort:         11,
			Group:        "page.params.group.test_data",
			GroupOrder:   20,
		},
	}
}

// GetDevices returns available compute devices for benchmarking.
// @Summary      Get compute devices
// @Description  Returns available compute devices (GPUs) detected by llama.cpp
// @Tags         Benchmark
// @Produce      json
// @Param        llamaBinPath query string true "Path to llama.cpp binaries"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/model/device/list [get]
func (h *Handler) GetDevices(c *gin.Context) {
	llamaBinPath := c.Query("llamaBinPath")
	if llamaBinPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "llamaBinPath parameter is required",
		})
		return
	}

	// 验证路径（可以是目录或具体的可执行文件）
	// 不再使用 isValidLlamaBinary，因为它要求必须是可执行文件
	// 而用户配置的是 llama.cpp 目录路径
	if err := h.validatePathForDevices(llamaBinPath); err != nil {
		h.log.Errorf("Invalid llama path: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid llama path: %v", err),
		})
		return
	}

	// 确定使用哪个二进制文件（llama-cli 或 llama-server）
	// llama-cli 是首选，因为它包含 --list-devices 选项
	llamaCliPath := h.findLlamaCli(llamaBinPath)
	if llamaCliPath == "" {
		h.log.Errorf("llama-cli not found in directory: %s", llamaBinPath)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "llama-cli not found in the specified directory",
		})
		return
	}

	// 使用统一的设备列表解析函数
	devices, err := utils.GetLlamacppDeviceList(llamaCliPath)
	if err != nil {
		h.log.Errorf("Failed to list devices: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to list devices: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"devices": devices,
		},
	})
}

// GetRunningTasksCount returns the number of currently running tasks.
func (h *Handler) GetRunningTasksCount() int {
	return h.taskMgr.RunningCount(taskmanager.TaskTypeBenchmark)
}

// Create creates a new benchmark task.
// @Summary      Create benchmark task
// @Description  Creates and starts a new benchmark task using llama-bench
// @Tags         Benchmark
// @Accept       json
// @Produce      json
// @Param        request body object true "Benchmark creation request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Router       /api/models/benchmark [post]
func (h *Handler) Create(c *gin.Context) {
	var req struct {
		ModelID      string            `json:"modelId" binding:"required"`
		ModelName    string            `json:"modelName"`
		LlamaBinPath string            `json:"llamaBinPath" binding:"required"`
		Cmd          string            `json:"cmd"`
		Args         []string          `json:"args"`
		Config       map[string]string `json:"config"`
		NodeID       string            `json:"nodeId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// 验证并查找 llama-bench 可执行文件
	benchPath := utils.FindLlamacppBinary(req.LlamaBinPath, "bench")
	if benchPath == "" {
		h.log.Errorf("llama-bench not found in: %s", req.LlamaBinPath)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("llama-bench not found in %s", req.LlamaBinPath),
		})
		return
	}

	// 验证二进制路径是否有效
	if err := h.isValidLlamaBinary(benchPath); err != nil {
		h.log.Errorf("Invalid llama binary path: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid llama binary path: %v", err),
		})
		return
	}

	// 检查并发限制
	if h.GetRunningTasksCount() >= MaxConcurrentBenchmarks {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Maximum concurrent benchmark limit (%d) reached, please try again later", MaxConcurrentBenchmarks),
		})
		return
	}

	// 生成任务 ID
	taskID := uuid.New().String()

	// 转换配置类型：map[string]string -> map[string]interface{}
	config := make(map[string]interface{})
	for k, v := range req.Config {
		config[k] = v
	}
	if req.NodeID != "" {
		config["nodeId"] = req.NodeID
	}

	// 创建任务
	task := &storage.Benchmark{
		ID:        taskID,
		ModelID:   req.ModelID,
		ModelName: req.ModelName,
		Status:    "running",
		Command:   req.Cmd, // Store for display
		Config:    config,
		CreatedAt: time.Now(),
	}

	if task.Command == "" && len(req.Args) > 0 {
		task.Command = strings.Join(req.Args, " ")
	}

	// 保存到存储层
	if err := h.store.CreateBenchmark(h.ctx, task); err != nil {
		h.log.Errorf("Failed to create benchmark task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create benchmark task",
		})
		return
	}

	// 注册到 TaskManager
	benchTask := &taskmanager.Task{
		ID:         taskID,
		Type:       taskmanager.TaskTypeBenchmark,
		Status:     taskmanager.TaskStatusPending,
		Name:       fmt.Sprintf("Benchmark: %s", req.ModelName),
		ModelID:    req.ModelID,
		ModelName:  req.ModelName,
		Command:    task.Command,
		CreatedAt:  time.Now(),
	}
	if err := h.taskMgr.Register(benchTask); err != nil {
		h.log.Errorf("Failed to register benchmark task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to register benchmark task",
		})
		return
	}

	// 异步执行压测任务
	go h.runBenchmark(task, benchTask, benchPath, req.Args)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"benchmark": task,
		},
	})
}

// runBenchmark executes a benchmark task.
func (h *Handler) runBenchmark(task *storage.Benchmark, benchTask *taskmanager.Task, llamaBinPath string, args []string) {
	select {
	case h.semaphore <- struct{}{}:
		defer func() { <-h.semaphore }()
	case <-h.ctx.Done():
		benchTask.Error = "Handler shutdown"
		benchTask.SetStatus(taskmanager.TaskStatusCancelled)
		return
	}

	startedAt := time.Now()
	task.StartedAt = &startedAt
	benchTask.SetStatus(taskmanager.TaskStatusRunning)

	h.log.Infof("Starting benchmark task %s: %s", task.ID, task.Command)

	cmdParts := args
	if len(cmdParts) == 0 && task.Command != "" {
		cmdParts = strings.Fields(task.Command)
	}

	if len(cmdParts) == 0 {
		benchTask.Error = "Empty command"
		benchTask.SetStatus(taskmanager.TaskStatusFailed)
		now := time.Now()
		task.FinishedAt = &now
		task.Status = "failed"
		task.Error = "Empty command"
		_ = h.store.UpdateBenchmark(h.ctx, task)
		return
	}

	cmd := exec.CommandContext(benchTask.Context(), llamaBinPath, cmdParts...)
	output, err := cmd.CombinedOutput()
	finishedAt := time.Now()
	task.FinishedAt = &finishedAt

	// Save output to file
	taskNodeId, _ := task.Config["nodeId"].(string)
	h.saveBenchmarkOutput(task.ModelID, taskNodeId, string(output))

	// 解析输出提取指标
	metrics := h.parseBenchmarkOutput(string(output))
	benchTask.Metrics = metrics

	if err != nil {
		if benchTask.Context().Err() != nil {
			benchTask.SetStatus(taskmanager.TaskStatusCancelled)
			task.Status = "cancelled"
			h.log.Infof("Benchmark task %s was cancelled", task.ID)
		} else {
			h.log.Errorf("Benchmark task %s failed: %v", task.ID, err)
			benchTask.Error = err.Error()
			benchTask.SetStatus(taskmanager.TaskStatusFailed)
			task.Status = "failed"
			task.Error = err.Error()
		}
	} else {
		h.log.Infof("Benchmark task %s completed successfully", task.ID)
		benchTask.SetStatus(taskmanager.TaskStatusCompleted)
		task.Status = "completed"
	}

	task.Metrics = metrics
	if err := h.store.UpdateBenchmark(h.ctx, task); err != nil {
		h.log.Errorf("Failed to save benchmark result: %v", err)
	}
}

// parseBenchmarkOutput parses benchmark output to extract metrics.
func (h *Handler) parseBenchmarkOutput(output string) map[string]interface{} {
	metrics := make(map[string]interface{})

	// 尝试从输出中提取常见指标
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// 查找类似 "Total time: 1234 ms" 的模式
		if strings.Contains(line, "ms") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.Contains(part, "ms") && i > 0 {
					var ms float64
					_, _ = fmt.Sscanf(parts[i-1], "%f", &ms)
					metrics["total_time_ms"] = ms
				}
			}
		}
		// 查找 tokens/s 信息
		if strings.Contains(line, "tokens/s") || strings.Contains(line, "t/s") {
			parts := strings.Fields(line)
			for _, part := range parts {
				var tps float64
				if _, err := fmt.Sscanf(part, "%f", &tps); err == nil && tps > 0 {
					metrics["tokens_per_second"] = tps
					break
				}
			}
		}
	}

	// 保存原始输出（截断过长的输出）
	if len(output) > 10000 {
		metrics["raw_output"] = output[:10000] + "\n... (truncated)"
	} else {
		metrics["raw_output"] = output
	}

	return metrics
}

// List returns all benchmark tasks.
// @Summary      List benchmark tasks
// @Description  Returns all benchmark tasks with their results
// @Tags         Benchmark
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/benchmark/tasks [get]
func (h *Handler) List(c *gin.Context) {
	// 获取查询参数
	modelID := c.Query("modelId")
	limit := 100
	offset := 0

	// 从存储层获取任务列表
	tasks, err := h.store.ListBenchmarks(h.ctx, modelID, limit, offset)
	if err != nil {
		h.log.Errorf("Failed to list benchmarks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list benchmarks",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"benchmarks": tasks,
		},
	})
}

// Get returns a specific benchmark task.
// @Summary      Get benchmark task
// @Description  Returns details and results of a specific benchmark task
// @Tags         Benchmark
// @Produce      json
// @Param        benchmarkId path string true "Benchmark task ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /api/models/benchmark/tasks/{benchmarkId} [get]
func (h *Handler) Get(c *gin.Context) {
	taskID := c.Param("benchmarkId")

	task, err := h.store.GetBenchmark(h.ctx, taskID)
	if err != nil {
		h.log.Errorf("Failed to get benchmark: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Benchmark not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    task,
	})
}

// Cancel cancels a running benchmark task.
// @Summary      Cancel benchmark task
// @Description  Cancels a currently running benchmark task
// @Tags         Benchmark
// @Produce      json
// @Param        benchmarkId path string true "Benchmark task ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/benchmark/tasks/{benchmarkId}/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	taskID := c.Param("benchmarkId")

	// 通过 TaskManager 取消任务
	if err := h.taskMgr.Cancel(taskID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Benchmark not found",
		})
		return
	}

	// 更新存储层状态
	task, err := h.store.GetBenchmark(h.ctx, taskID)
	if err == nil {
		task.Status = "cancelled"
		now := time.Now()
		task.FinishedAt = &now
		_ = h.store.UpdateBenchmark(h.ctx, task)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// SaveConfig saves a benchmark configuration.
// @Summary      Save benchmark config
// @Description  Saves a named benchmark configuration for reuse
// @Tags         Benchmark
// @Accept       json
// @Produce      json
// @Param        request body object true "Config name and benchmark configuration"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/benchmark/configs [post]
func (h *Handler) SaveConfig(c *gin.Context) {
	var req struct {
		ConfigName string          `json:"configName" binding:"required"`
		Config     BenchmarkConfig `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// 验证二进制路径
	if err := h.isValidLlamaBinary(req.Config.LlamaCppPath); err != nil {
		h.log.Errorf("Invalid llama binary path: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid llama binary path: %v", err),
		})
		return
	}

	// 转换为存储层配置
	config := &storage.BenchmarkConfig{
		Name:         req.ConfigName,
		ModelID:      req.Config.ModelID,
		ModelName:    req.Config.ModelName,
		LlamaCppPath: req.Config.LlamaCppPath,
		Devices:      req.Config.Devices,
		Params:       req.Config.Params,
		CreatedAt:    time.Now(),
	}

	if err := h.store.CreateBenchmarkConfig(h.ctx, config); err != nil {
		h.log.Errorf("Failed to save benchmark config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save benchmark config",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// ListConfigs returns all saved benchmark configurations.
// @Summary      List benchmark configs
// @Description  Returns all saved benchmark configurations
// @Tags         Benchmark
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/benchmark/configs [get]
func (h *Handler) ListConfigs(c *gin.Context) {
	limit := 100
	offset := 0

	configs, err := h.store.ListBenchmarkConfigs(h.ctx, limit, offset)
	if err != nil {
		h.log.Errorf("Failed to list benchmark configs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list benchmark configs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"configs": configs,
		},
	})
}

// GetConfig returns a specific benchmark configuration by name.
// @Summary      Get benchmark config
// @Description  Returns a specific saved benchmark configuration
// @Tags         Benchmark
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /api/models/benchmark/configs/{name} [get]
func (h *Handler) GetConfig(c *gin.Context) {
	configName := c.Param("name")

	config, err := h.store.GetBenchmarkConfig(h.ctx, configName)
	if err != nil {
		h.log.Errorf("Failed to get benchmark config: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Config not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// DeleteConfig deletes a benchmark configuration by name.
// @Summary      Delete benchmark config
// @Description  Deletes a saved benchmark configuration
// @Tags         Benchmark
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Router       /api/models/benchmark/configs/{name} [delete]
func (h *Handler) DeleteConfig(c *gin.Context) {
	configName := c.Param("name")

	if err := h.store.DeleteBenchmarkConfig(h.ctx, configName); err != nil {
		h.log.Errorf("Failed to delete benchmark config: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Config not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// HistoryFile represents a benchmark history file entry
type HistoryFile struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// ListHistory returns benchmark result files for a model.
func (h *Handler) ListHistory(c *gin.Context) {
	modelId := c.Query("modelId")
	if modelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "modelId parameter is required",
		})
		return
	}
	nodeId := c.Query("nodeId")

	dir := benchmarkDir(modelId, nodeId)

	// Create directory if not exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		h.log.Errorf("Failed to create benchmark dir: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to access benchmark directory",
		})
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"files": []HistoryFile{},
			},
		})
		return
	}

	var files []HistoryFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, HistoryFile{
			Name:     entry.Name(),
			Size:     info.Size(),
			Modified: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"files": files,
		},
	})
}

// GetHistoryFile returns the content of a benchmark result file.
func (h *Handler) GetHistoryFile(c *gin.Context) {
	fileName := c.Query("fileName")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "fileName parameter is required",
		})
		return
	}
	nodeId := c.Query("nodeId")

	// Prevent path traversal
	cleanName := filepath.Base(fileName)
	if cleanName != fileName || strings.Contains(fileName, "..") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid file name",
		})
		return
	}

	// Search in all model directories under the appropriate base
	baseDir := filepath.Join("data", "benchmark")
	if nodeId != "" {
		baseDir = filepath.Join("data", "benchmark", sanitizeModelId(nodeId))
	}
	var filePath string

	if entries, err := os.ReadDir(baseDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			candidate := filepath.Join(baseDir, entry.Name(), cleanName)
			if _, err := os.Stat(candidate); err == nil {
				filePath = candidate
				break
			}
		}
	}

	if filePath == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "File not found",
		})
		return
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		h.log.Errorf("Failed to read benchmark file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to read file",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"rawOutput": string(content),
			"fileName":  cleanName,
		},
	})
}

// DeleteHistoryFile deletes a benchmark result file.
func (h *Handler) DeleteHistoryFile(c *gin.Context) {
	fileName := c.Query("fileName")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "fileName parameter is required",
		})
		return
	}
	nodeId := c.Query("nodeId")

	// Prevent path traversal
	cleanName := filepath.Base(fileName)
	if cleanName != fileName || strings.Contains(fileName, "..") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid file name",
		})
		return
	}

	// Search in all model directories under the appropriate base
	baseDir := filepath.Join("data", "benchmark")
	if nodeId != "" {
		baseDir = filepath.Join("data", "benchmark", sanitizeModelId(nodeId))
	}
	var filePath string

	if entries, err := os.ReadDir(baseDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			candidate := filepath.Join(baseDir, entry.Name(), cleanName)
			if _, err := os.Stat(candidate); err == nil {
				filePath = candidate
				break
			}
		}
	}

	if filePath == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "File not found",
		})
		return
	}

	if err := os.Remove(filePath); err != nil {
		h.log.Errorf("Failed to delete benchmark file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete file",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// sanitizeModelId creates a safe directory name from a model ID
func sanitizeModelId(modelId string) string {
	// Replace path separators and special chars with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		" ", "_",
		"..", "_",
	)
	safe := replacer.Replace(modelId)
	// Remove any remaining problematic characters
	result := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			return r
		}
		return '_'
	}, safe)
	if result == "" {
		result = "unknown"
	}
	return result
}

// benchmarkDir returns the benchmark directory path for a model, optionally scoped by nodeId.
// Format: data/benchmark/{nodeId}/{modelId}/ (with nodeId) or data/benchmark/{modelId}/ (without)
func benchmarkDir(modelId string, nodeId string) string {
	safeModelId := sanitizeModelId(modelId)
	if nodeId != "" {
		safeNodeId := sanitizeModelId(nodeId)
		return filepath.Join("data", "benchmark", safeNodeId, safeModelId)
	}
	return filepath.Join("data", "benchmark", safeModelId)
}

// Delete deletes a benchmark task by ID.
// @Summary      Delete benchmark task
// @Description  Deletes a benchmark task and its results
// @Tags         Benchmark
// @Produce      json
// @Param        benchmarkId path string true "Benchmark task ID"
// @Success      200 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/benchmark/tasks/{benchmarkId} [delete]
func (h *Handler) Delete(c *gin.Context) {
	taskID := c.Param("benchmarkId")

	if err := h.store.DeleteBenchmark(h.ctx, taskID); err != nil {
		h.log.Errorf("Failed to delete benchmark: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Benchmark not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// ListTasks returns currently tracked benchmark tasks.
// @Summary      List active benchmark tasks
// @Description  Returns currently tracked benchmark tasks from the TaskManager
// @Tags         Benchmark
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/models/benchmark/active-tasks [get]
func (h *Handler) ListTasks(c *gin.Context) {
	tasks := h.taskMgr.List(taskmanager.TaskTypeBenchmark)
	var result []map[string]interface{}
	for _, t := range tasks {
		result = append(result, t.ToMap())
	}
	if result == nil {
		result = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks": result,
		},
	})
}

// saveBenchmarkOutput saves the benchmark output to a timestamped file
func (h *Handler) saveBenchmarkOutput(modelId string, nodeId string, output string) {
	if output == "" {
		return
	}

	dir := benchmarkDir(modelId, nodeId)

	if err := os.MkdirAll(dir, 0755); err != nil {
		h.log.Errorf("Failed to create benchmark output dir: %v", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_%s.txt", sanitizeModelId(modelId), timestamp)
	filePath := filepath.Join(dir, fileName)

	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		h.log.Errorf("Failed to save benchmark output: %v", err)
	} else {
		h.log.Infof("Benchmark output saved to %s", filePath)
	}
}

// Shutdown gracefully shuts down the handler.
func (h *Handler) Shutdown() {
	h.log.Infof("Benchmark handler shutting down...")
	h.cancelFunc()
}

// ============================================================
// V2 Benchmark: Test loaded model throughput via chat completions
// ============================================================

// benchmarkV2Record represents a single V2 benchmark result stored in JSONL
type benchmarkV2Record struct {
	Timestamp    string                 `json:"timestamp"`
	ModelID      string                 `json:"modelId"`
	PromptTokens int                   `json:"promptTokens"`
	MaxTokens    int                   `json:"maxTokens"`
	Timings      map[string]interface{} `json:"timings"`
	Devices      []string               `json:"devices,omitempty"`
	Cmd          string                 `json:"cmd,omitempty"`
}

// CreateV2 runs a V2 benchmark by sending a synthetic prompt to a loaded model.
// @Summary      Run V2 benchmark
// @Description  Tests loaded model throughput via /v1/chat/completions
// @Tags         Benchmark
// @Accept       json
// @Produce      json
// @Param        request body object true "V2 benchmark request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/models/benchmark/v2 [post]
func (h *Handler) CreateV2(c *gin.Context) {
	var req struct {
		ModelID      string `json:"modelId" binding:"required"`
		PromptTokens int    `json:"promptTokens" binding:"required,min=1"`
		MaxTokens    int    `json:"maxTokens" binding:"required,min=1"`
		NodeID       string `json:"nodeId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Check model is loaded
	status, exists := h.modelMgr.GetStatusRef(req.ModelID)
	if !exists || status.State != model.StateLoaded || status.Port <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Model is not loaded",
		})
		return
	}

	port := status.Port

	// Generate synthetic prompt content: repeat " a" to approximate target token count
	// Each " a" is roughly 1 token for most tokenizers
	promptContent := strings.Repeat(" a", req.PromptTokens)

	// Build chat completions request
	chatReq := map[string]interface{}{
		"model": req.ModelID,
		"messages": []map[string]string{
			{"role": "user", "content": promptContent},
		},
		"max_tokens": req.MaxTokens,
		"stream":     false,
	}

	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to build request",
		})
		return
	}

	// Send request to loaded model
	targetURL := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port)
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", targetURL, bytes.NewReader(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create request",
		})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.Errorf("V2 benchmark request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Request to model failed: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to read response",
		})
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Model returned error: %s", string(respBody)),
		})
		return
	}

	// Parse response to extract timings
	var respObj map[string]interface{}
	if err := json.Unmarshal(respBody, &respObj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to parse model response",
		})
		return
	}

	timings, _ := respObj["timings"].(map[string]interface{})
	if timings == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Model response missing timings data",
		})
		return
	}

	// Build record
	timestamp := time.Now().Format("20060102_150405")
	record := &benchmarkV2Record{
		Timestamp:    timestamp,
		ModelID:      req.ModelID,
		PromptTokens: req.PromptTokens,
		MaxTokens:    req.MaxTokens,
		Timings:      timings,
	}

	// Save to JSONL file
	h.saveV2Record(record, req.NodeID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    record,
	})
}

// saveV2Record appends a V2 benchmark record to the model's JSONL file
func (h *Handler) saveV2Record(record *benchmarkV2Record, nodeId string) {
	dir := benchmarkDir(record.ModelID, nodeId)

	if err := os.MkdirAll(dir, 0755); err != nil {
		h.log.Errorf("Failed to create V2 benchmark dir: %v", err)
		return
	}

	safeModelId := sanitizeModelId(record.ModelID)
	fileName := fmt.Sprintf("%s_V2.jsonl", safeModelId)
	filePath := filepath.Join(dir, fileName)

	line, err := json.Marshal(record)
	if err != nil {
		h.log.Errorf("Failed to marshal V2 record: %v", err)
		return
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		h.log.Errorf("Failed to open V2 file: %v", err)
		return
	}
	defer f.Close()

	f.Write(line)
	f.Write([]byte("\n"))
}

// ListV2 returns V2 benchmark records for a model.
// @Summary      List V2 benchmark records
// @Description  Returns all V2 benchmark records from the JSONL file
// @Tags         Benchmark
// @Produce      json
// @Param        modelId query string true "Model ID"
// @Success      200 {object} map[string]interface{}
// @Router       /api/models/benchmark/v2/list [get]
func (h *Handler) ListV2(c *gin.Context) {
	modelId := c.Query("modelId")
	if modelId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "modelId parameter is required",
		})
		return
	}
	nodeId := c.Query("nodeId")

	dir := benchmarkDir(modelId, nodeId)
	safeModelId := sanitizeModelId(modelId)
	fileName := fmt.Sprintf("%s_V2.jsonl", safeModelId)
	filePath := filepath.Join(dir, fileName)

	records := make([]map[string]interface{}, 0)

	f, err := os.Open(filePath)
	if err != nil {
		// File doesn't exist yet - return empty
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"records": records,
			},
		})
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	lineNumber := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			lineNumber++
			continue
		}
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			lineNumber++
			continue
		}
		record["lineNumber"] = lineNumber
		records = append(records, record)
		lineNumber++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"records": records,
		},
	})
}

// DeleteV2 deletes a V2 benchmark record by line number.
// @Summary      Delete V2 benchmark record
// @Description  Removes a specific record from the JSONL file
// @Tags         Benchmark
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/models/benchmark/v2/delete [post]
func (h *Handler) DeleteV2(c *gin.Context) {
	var req struct {
		ModelID    string `json:"modelId" binding:"required"`
		LineNumber int    `json:"lineNumber"`
		NodeID     string `json:"nodeId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	dir := benchmarkDir(req.ModelID, req.NodeID)
	safeModelId := sanitizeModelId(req.ModelID)
	fileName := fmt.Sprintf("%s_V2.jsonl", safeModelId)
	filePath := filepath.Join(dir, fileName)

	// Read all lines
	f, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "File not found",
		})
		return
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	f.Close()

	if req.LineNumber < 0 || req.LineNumber >= len(lines) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid line number",
		})
		return
	}

	// Remove the line
	lines = append(lines[:req.LineNumber], lines[req.LineNumber+1:]...)

	// Write back
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		h.log.Errorf("Failed to write V2 file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete record",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
