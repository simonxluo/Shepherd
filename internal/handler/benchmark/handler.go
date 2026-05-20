// Package benchmark provides pressure testing API handlers
package benchmark

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
)

const (
	// MaxConcurrentBenchmarks 最大并发压测任务数
	MaxConcurrentBenchmarks = 3
)

// runningTask 运行中的任务信息
type runningTask struct {
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	taskID    string
	startedAt time.Time
}

// Handler 压测 API 处理器
type Handler struct {
	log          *logger.Logger
	store        storage.Store
	ctx          context.Context
	cancelFunc   context.CancelFunc
	runningTasks map[string]*runningTask
	taskMutex    sync.RWMutex
	semaphore    chan struct{} // 用于限制并发数
}

// NewHandler 创建新的压测处理器
func NewHandler(log *logger.Logger, store storage.Store) *Handler {
	ctx, cancel := context.WithCancel(context.Background())

	h := &Handler{
		log:          log,
		store:        store,
		ctx:          ctx,
		cancelFunc:   cancel,
		runningTasks: make(map[string]*runningTask),
		semaphore:    make(chan struct{}, MaxConcurrentBenchmarks),
	}

	// 启动清理 goroutine
	go h.cleanupFinishedTasks()

	return h
}

// BenchmarkParam 压测参数定义
type BenchmarkParam struct {
	FullName     string   `json:"fullName"`
	Name         string   `json:"name"`
	Abbreviation string   `json:"abbreviation"`
	Description  string   `json:"description"`
	DefaultValue string   `json:"defaultValue"`
	Type         string   `json:"type"`
	Values       []string `json:"values,omitempty"`
	Sort         int      `json:"sort"`
}

// BenchmarkConfig 压测配置
type BenchmarkConfig struct {
	ModelID      string            `json:"modelId"`
	ModelName    string            `json:"modelName"`
	LlamaCppPath string            `json:"llamaCppPath"`
	Devices      []string          `json:"devices"`
	Params       map[string]string `json:"params"`
	CreatedAt    string            `json:"createdAt"`
}

// cleanupFinishedTasks 定期清理已完成的任务
func (h *Handler) cleanupFinishedTasks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.taskMutex.Lock()
			for taskID, task := range h.runningTasks {
				// 如果任务运行超过 1 小时，清理它
				if time.Since(task.startedAt) > time.Hour {
					h.log.Warnf("Cleaning up stale task %s", taskID)
					delete(h.runningTasks, taskID)
				}
			}
			h.taskMutex.Unlock()
		}
	}
}

// isValidLlamaBinary 验证 llama.cpp 二进制文件是否安全可执行
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

// findLlamaCli 在指定路径中查找 llama-cli 可执行文件
// 使用统一的工具函数 utils.FindLlamacppBinary
func (h *Handler) findLlamaCli(llamaBinPath string) string {
	return utils.FindLlamacppBinary(llamaBinPath, "cli")
}

// validatePathForDevices 验证路径是否有效（用于设备列表查询）
// 允许目录路径或可执行文件路径
// 使用统一的工具函数来查找二进制文件
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
	// 返回常见的 llama.cpp 压测参数
	params := []BenchmarkParam{
		{
			FullName:     "-p",
			Name:         "Prompt Tokens",
			Abbreviation: "-p",
			Description:  "提示词 token 数量 (建议 8192)",
			DefaultValue: "8192",
			Type:         "INTEGER",
			Sort:         1,
		},
		{
			FullName:     "-n",
			Name:         "Max Tokens",
			Abbreviation: "-n",
			Description:  "生成的 token 数量 (建议 256)",
			DefaultValue: "256",
			Type:         "INTEGER",
			Sort:         2,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"params":  params,
	})
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

// GetRunningTasksCount 获取当前运行中的任务数量
func (h *Handler) GetRunningTasksCount() int {
	h.taskMutex.RLock()
	defer h.taskMutex.RUnlock()
	return len(h.runningTasks)
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

	// 异步执行压测任务
	go h.runBenchmark(task, benchPath, req.Args)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"benchmark": task,
		},
	})
}

// runBenchmark 执行压测任务
func (h *Handler) runBenchmark(task *storage.Benchmark, llamaBinPath string, args []string) {

	// 获取信号量（限制并发）
	select {
	case h.semaphore <- struct{}{}:
		defer func() { <-h.semaphore }()
	case <-h.ctx.Done():
		task.Status = "cancelled"
		task.Error = "Handler shutdown"
		finishedAt := time.Now()
		task.FinishedAt = &finishedAt
		if err := h.store.UpdateBenchmark(h.ctx, task); err != nil {
			h.log.Errorf("Failed to update benchmark: %v", err)
		}
		return
	}

	startedAt := time.Now()
	task.StartedAt = &startedAt

	h.log.Infof("Starting benchmark task %s: %s", task.ID, task.Command)

	// 创建带取消的上下文
	taskCtx, cancel := context.WithCancel(h.ctx)

	// 构建命令
	cmdParts := args
	if len(cmdParts) == 0 && task.Command != "" {
		cmdParts = strings.Fields(task.Command)
	}

	if len(cmdParts) == 0 {
		task.Status = "failed"
		task.Error = "Empty command"
		finishedAt := time.Now()
		task.FinishedAt = &finishedAt
		if err := h.store.UpdateBenchmark(h.ctx, task); err != nil {
			h.log.Errorf("Failed to update benchmark: %v", err)
		}
		cancel()
		return
	}

	// 构建完整命令（添加 llama.cpp 路径）
	cmd := exec.CommandContext(taskCtx, llamaBinPath, cmdParts...)

	// 记录运行中的任务
	h.taskMutex.Lock()
	h.runningTasks[task.ID] = &runningTask{
		cmd:       cmd,
		cancel:    cancel,
		taskID:    task.ID,
		startedAt: startedAt,
	}
	h.taskMutex.Unlock()

	// 确保清理
	defer func() {
		h.taskMutex.Lock()
		delete(h.runningTasks, task.ID)
		h.taskMutex.Unlock()
		cancel()
	}()

	// 保存启动状态
	if err := h.store.UpdateBenchmark(h.ctx, task); err != nil {
		h.log.Warnf("Failed to save benchmark status: %v", err)
	}

	// 执行并捕获输出
	output, err := cmd.CombinedOutput()
	finishedAt := time.Now()
	task.FinishedAt = &finishedAt

	// 解析输出提取指标
	metrics := h.parseBenchmarkOutput(string(output))

	// 检查是否被取消
	h.taskMutex.RLock()
	_, wasRunning := h.runningTasks[task.ID]
	h.taskMutex.RUnlock()

	if !wasRunning {
		// 任务被取消
		task.Status = "cancelled"
		h.log.Infof("Benchmark task %s was cancelled", task.ID)
	} else if err != nil {
		// 检查是否是因为上下文取消导致的错误
		if taskCtx.Err() != nil {
			task.Status = "cancelled"
			h.log.Infof("Benchmark task %s was cancelled", task.ID)
		} else {
			h.log.Errorf("Benchmark task %s failed: %v", task.ID, err)
			task.Status = "failed"
			task.Error = err.Error()
		}
	} else {
		h.log.Infof("Benchmark task %s completed successfully", task.ID)
		task.Status = "completed"
	}

	// 保存指标到数据库
	task.Metrics = metrics
	if err := h.store.UpdateBenchmark(h.ctx, task); err != nil {
		h.log.Errorf("Failed to save benchmark result: %v", err)
	}
}

// parseBenchmarkOutput 解析压测输出提取指标
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

	// 获取任务
	task, err := h.store.GetBenchmark(h.ctx, taskID)
	if err != nil {
		h.log.Errorf("Failed to get benchmark: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Benchmark not found",
		})
		return
	}

	// 检查任务状态
	if task.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Task is not running",
		})
		return
	}

	// 获取运行中的任务并取消
	h.taskMutex.Lock()
	runningTask, exists := h.runningTasks[taskID]
	if exists && runningTask.cancel != nil {
		// 取消上下文
		runningTask.cancel()

		// 尝试终止进程
		if runningTask.cmd != nil && runningTask.cmd.Process != nil {
			h.log.Infof("Sending SIGTERM to benchmark process %s", taskID)
			if err := runningTask.cmd.Process.Signal(syscall.SIGTERM); err != nil {
				h.log.Warnf("Failed to send SIGTERM, trying SIGKILL: %v", err)
				_ = runningTask.cmd.Process.Kill()
			}
		}
	}
	h.taskMutex.Unlock()

	// 更新状态为已取消
	task.Status = "cancelled"
	finishedAt := time.Now()
	task.FinishedAt = &finishedAt

	if err := h.store.UpdateBenchmark(h.ctx, task); err != nil {
		h.log.Errorf("Failed to cancel benchmark: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to cancel benchmark",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    task,
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

// Shutdown 优雅关闭处理器
func (h *Handler) Shutdown() {
	h.log.Infof("Benchmark handler shutting down, waiting for running tasks to complete...")

	// 取消上下文
	h.cancelFunc()

	// 等待所有任务完成（最多等待 30 秒）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.log.Warnf("Timeout waiting for tasks to complete, %d tasks still running", h.GetRunningTasksCount())
			return
		case <-ticker.C:
			if h.GetRunningTasksCount() == 0 {
				h.log.Infof("All benchmark tasks completed")
				return
			}
		}
	}
}
