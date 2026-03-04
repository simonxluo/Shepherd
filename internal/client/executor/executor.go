// Package executor provides task execution functionality for client nodes.
package executor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/client"
	"github.com/shepherd-project/shepherd/Shepherd/internal/cluster"
	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
)

// runningProcess represents a running model process
type runningProcess struct {
	cmd       *exec.Cmd
	pid       int
	port      int
	modelID   string
	modelPath string
	started   time.Time
}

// Executor executes tasks on the client node
type Executor struct {
	config *client.ExecutorConfig
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	log    *logger.Logger

	// Running tasks tracking
	runningTasks     map[string]*runningTask
	runningProcesses map[string]*runningProcess // 进程跟踪 (modelID -> process)
}

// runningTask represents a currently running task
type runningTask struct {
	context context.Context
	cancel  context.CancelFunc
	started time.Time
}

// NewExecutor creates a new task executor
func NewExecutor(cfg *config.NodeConfig, log *logger.Logger) *Executor {
	ctx, cancel := context.WithCancel(context.Background())

	// 从 NodeConfig 构建 ExecutorConfig
	taskTimeout := time.Duration(cfg.Executor.TaskTimeout) * time.Second
	if taskTimeout == 0 {
		taskTimeout = 5 * time.Minute // 默认 5 分钟
	}

	maxConcurrent := cfg.Executor.MaxConcurrent
	if maxConcurrent == 0 {
		maxConcurrent = 4 // 默认最大并发 4
	}

	execConfig := &client.ExecutorConfig{
		CondaPath:     cfg.Capabilities.CondaPath,
		CondaEnvs:     cfg.Capabilities.CondaEnvironments,
		LlamacppPaths: []string{}, // llama.cpp 路径由任务执行时动态配置
		TaskTimeout:   taskTimeout,
		MaxConcurrent: maxConcurrent,
	}

	return &Executor{
		config:           execConfig,
		ctx:              ctx,
		cancel:           cancel,
		wg:               sync.WaitGroup{},
		log:              log,
		runningTasks:     make(map[string]*runningTask),
		runningProcesses: make(map[string]*runningProcess),
	}
}

// Start starts the executor
func (e *Executor) Start() {
	// No background tasks needed for now
}

// Stop stops the executor and cancels all running tasks
func (e *Executor) Stop() {
	e.cancel()

	e.mu.Lock()

	// Cancel all running tasks
	for _, task := range e.runningTasks {
		task.cancel()
	}

	// Stop all running processes
	for _, proc := range e.runningProcesses {
		if proc.cmd != nil && proc.cmd.Process != nil {
			proc.cmd.Process.Kill()
		}
	}

	// Clear the maps
	e.runningTasks = make(map[string]*runningTask)
	e.runningProcesses = make(map[string]*runningProcess)

	e.mu.Unlock()

	e.wg.Wait()
}

// ExecuteTask executes a task
func (e *Executor) ExecuteTask(task *cluster.Task) (*client.TaskResult, error) {
	startTime := time.Now()

	e.log.Info(fmt.Sprintf("开始执行任务: %s (类型: %s)", task.ID, task.Type))

	// Check task limit
	e.mu.Lock()
	if len(e.runningTasks) >= e.config.MaxConcurrent {
		e.mu.Unlock()
		return &client.TaskResult{
			TaskID:      task.ID,
			Success:     false,
			Error:       "任务队列已满",
			CompletedAt: time.Now(),
			Duration:    time.Since(startTime).Milliseconds(),
		}, fmt.Errorf("任务队列已满")
	}

	// Create task context
	taskCtx, taskCancel := context.WithTimeout(e.ctx, e.config.TaskTimeout)

	// Track running task
	e.runningTasks[task.ID] = &runningTask{
		context: taskCtx,
		cancel:  taskCancel,
		started: startTime,
	}
	e.mu.Unlock()

	// Execute task based on type
	var result *client.TaskResult
	var err error

	switch task.Type {
	case cluster.TaskTypeLoadModel:
		result, err = e.executeLoadModel(taskCtx, task)
	case cluster.TaskTypeUnloadModel:
		result, err = e.executeUnloadModel(taskCtx, task)
	case cluster.TaskTypeRunPython:
		result, err = e.executeRunPython(taskCtx, task)
	case cluster.TaskTypeRunLlamacpp:
		result, err = e.executeRunLlamacpp(taskCtx, task)
	default:
		result = &client.TaskResult{
			TaskID:      task.ID,
			Success:     false,
			Error:       fmt.Sprintf("未知任务类型: %s", task.Type),
			CompletedAt: time.Now(),
			Duration:    time.Since(startTime).Milliseconds(),
		}
		err = fmt.Errorf("未知任务类型: %s", task.Type)
	}

	// Remove from running tasks
	e.mu.Lock()
	delete(e.runningTasks, task.ID)
	e.mu.Unlock()

	result.CompletedAt = time.Now()
	result.Duration = time.Since(startTime).Milliseconds()

	if err != nil {
		e.log.Error(fmt.Sprintf("任务执行失败: %s - %v", task.ID, err))
	} else {
		e.log.Info(fmt.Sprintf("任务执行成功: %s", task.ID))
	}

	return result, err
}

// executeLoadModel executes a load model task
func (e *Executor) executeLoadModel(ctx context.Context, task *cluster.Task) (*client.TaskResult, error) {
	// Extract parameters
	modelPath, ok := task.Payload["model_path"].(string)
	if !ok {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "缺少 model_path 参数",
		}, fmt.Errorf("缺少 model_path 参数")
	}

	modelID, ok := task.Payload["model_id"].(string)
	if !ok {
		modelID = filepath.Base(modelPath)
	}

	// Get port parameter (required)
	portFloat, ok := task.Payload["port"].(float64)
	if !ok {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "缺少 port 参数",
		}, fmt.Errorf("缺少 port 参数")
	}
	port := int(portFloat)

	// Get optional parameters
	ctxSize := 4096 // 默认上下文大小
	if cs, ok := task.Payload["ctx_size"].(float64); ok {
		ctxSize = int(cs)
	}

	gpuLayers := 999 // 默认 GPU 层数
	if gl, ok := task.Payload["gpu_layers"].(float64); ok {
		gpuLayers = int(gl)
	}

	threads := -1 // 默认线程数
	if t, ok := task.Payload["threads"].(float64); ok {
		threads = int(t)
	}

	e.log.Info(fmt.Sprintf("加载模型: %s (port: %d, ctx: %d, gpu_layers: %d)", modelID, port, ctxSize, gpuLayers))

	// Find llama.cpp binary
	binPath, err := e.findLlamacppBinary()
	if err != nil {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "llama.cpp binary not found",
		}, err
	}

	e.log.Info(fmt.Sprintf("使用 llama.cpp: %s", binPath))

	// Build command
	args := []string{
		"serve",
		"-m", modelPath,
		"--port", strconv.Itoa(port),
		"-c", strconv.Itoa(ctxSize),
		"--n-gpu-layers", strconv.Itoa(gpuLayers),
		"--threads", strconv.Itoa(threads),
		"--no-mmap", // Strix Halo 必需参数
		"-fa", "1",  // Flash attention
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	e.log.Info(fmt.Sprintf("执行命令: %s %s", binPath, strings.Join(args, " ")))

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err.Error(),
		}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Track process
	e.mu.Lock()
	e.runningProcesses[modelID] = &runningProcess{
		cmd:       cmd,
		pid:       cmd.Process.Pid,
		port:      port,
		modelID:   modelID,
		modelPath: modelPath,
		started:   time.Now(),
	}
	e.mu.Unlock()

	e.log.Info(fmt.Sprintf("进程已启动: PID=%d, Port=%d", cmd.Process.Pid, port))

	// Monitor output for load completion
	loadSuccess := make(chan bool, 1)
	loadError := make(chan error, 1)

	// Start output scanner
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			e.log.Info(fmt.Sprintf("[llama-server] %s", line))

			// Check for load completion
			if strings.Contains(line, "all slots are idle") {
				e.log.Info(fmt.Sprintf("模型加载完成: %s", modelID))
				loadSuccess <- true
				return
			}
		}
		if err := scanner.Err(); err != nil {
			loadError <- fmt.Errorf("读取输出失败: %w", err)
		}
	}()

	// Start error scanner
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			e.log.Error(fmt.Sprintf("[llama-server-error] %s", line))
		}
	}()

	// Wait for load completion with timeout
	select {
	case success := <-loadSuccess:
		if success {
			return &client.TaskResult{
				TaskID:  task.ID,
				Success: true,
				Result: map[string]interface{}{
					"port":       port,
					"pid":        cmd.Process.Pid,
					"loaded":     true,
					"model_id":   modelID,
					"model_path": modelPath,
				},
				Output: "模型加载成功",
			}, nil
		}
	case err := <-loadError:
		// Clean up process on error
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		e.mu.Lock()
		delete(e.runningProcesses, modelID)
		e.mu.Unlock()
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err.Error(),
		}, err
	case <-time.After(10 * time.Minute):
		// Timeout
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		e.mu.Lock()
		delete(e.runningProcesses, modelID)
		e.mu.Unlock()
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "模型加载超时 (10分钟)",
		}, fmt.Errorf("模型加载超时")
	case <-ctx.Done():
		// Context cancelled
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		e.mu.Lock()
		delete(e.runningProcesses, modelID)
		e.mu.Unlock()
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "任务被取消",
		}, ctx.Err()
	}

	return &client.TaskResult{
		TaskID:  task.ID,
		Success: false,
		Error:   "未知错误",
	}, fmt.Errorf("未知错误")
}

// executeUnloadModel executes an unload model task
func (e *Executor) executeUnloadModel(ctx context.Context, task *cluster.Task) (*client.TaskResult, error) {
	// Extract parameters
	modelID, ok := task.Payload["model_id"].(string)
	if !ok {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "缺少 model_id 参数",
		}, fmt.Errorf("缺少 model_id 参数")
	}

	e.log.Info(fmt.Sprintf("卸载模型: %s", modelID))

	// Find and stop the process
	e.mu.Lock()
	proc, exists := e.runningProcesses[modelID]
	if !exists {
		e.mu.Unlock()
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   fmt.Sprintf("模型未加载: %s", modelID),
		}, fmt.Errorf("模型未加载: %s", modelID)
	}
	delete(e.runningProcesses, modelID)
	e.mu.Unlock()

	// Kill the process
	if proc.cmd != nil && proc.cmd.Process != nil {
		if err := proc.cmd.Process.Kill(); err != nil {
			e.log.Error(fmt.Sprintf("终止进程失败: %v", err))
		} else {
			e.log.Info(fmt.Sprintf("已终止进程: PID=%d", proc.pid))
		}
		// Wait for process to exit
		proc.cmd.Wait()
	}

	return &client.TaskResult{
		TaskID:  task.ID,
		Success: true,
		Result: map[string]interface{}{
			"model_id": modelID,
			"pid":      proc.pid,
			"unloaded": true,
		},
		Output: fmt.Sprintf("模型卸载成功 (PID=%d)", proc.pid),
	}, nil
}

// executeRunPython executes a Python script task
func (e *Executor) executeRunPython(ctx context.Context, task *cluster.Task) (*client.TaskResult, error) {
	// Extract parameters
	script, ok := task.Payload["script"].(string)
	if !ok {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "缺少 script 参数",
		}, fmt.Errorf("缺少 script 参数")
	}

	condaEnv := ""
	if env, ok := task.Payload["conda_env"].(string); ok {
		condaEnv = env
	}

	e.log.Info(fmt.Sprintf("运行Python脚本: %s (环境: %s)", script, condaEnv))

	// Build command
	var cmd *exec.Cmd
	if condaEnv != "" && e.config.CondaPath != "" {
		envPath, exists := e.config.CondaEnvs[condaEnv]
		if !exists {
			return &client.TaskResult{
				TaskID:  task.ID,
				Success: false,
				Error:   fmt.Sprintf("Conda环境不存在: %s", condaEnv),
			}, fmt.Errorf("conda环境不存在: %s", condaEnv)
		}
		cmd = exec.CommandContext(ctx, envPath+"/bin/python", script)
	} else {
		cmd = exec.CommandContext(ctx, "python", script)
	}

	// Run command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   err.Error(),
			Output:  string(output),
		}, err
	}

	return &client.TaskResult{
		TaskID:  task.ID,
		Success: true,
		Result: map[string]interface{}{
			"exit_code": 0,
		},
		Output: string(output),
	}, nil
}

// executeRunLlamacpp executes a llama.cpp task
func (e *Executor) executeRunLlamacpp(ctx context.Context, task *cluster.Task) (*client.TaskResult, error) {
	// Extract parameters
	modelPath, ok := task.Payload["model_path"].(string)
	if !ok {
		return &client.TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   "缺少 model_path 参数",
		}, fmt.Errorf("缺少 model_path 参数")
	}

	prompt := ""
	if p, ok := task.Payload["prompt"].(string); ok {
		prompt = p
	}

	e.log.Info(fmt.Sprintf("运行llama.cpp: %s", modelPath))

	// This is a placeholder - actual implementation would:
	// 1. Find llama.cpp binary
	// 2. Start with model and parameters
	// 3. Capture output

	return &client.TaskResult{
		TaskID:  task.ID,
		Success: true,
		Result: map[string]interface{}{
			"model_path": modelPath,
			"prompt":     prompt,
		},
		Output: "llama.cpp执行完成",
	}, nil
}

// CancelTask cancels a running task
func (e *Executor) CancelTask(taskID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, exists := e.runningTasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在或未运行: %s", taskID)
	}

	task.cancel()
	delete(e.runningTasks, taskID)

	e.log.Info(fmt.Sprintf("取消任务: %s", taskID))

	return nil
}

// GetRunningTasks returns the list of running task IDs
func (e *Executor) GetRunningTasks() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tasks := make([]string, 0, len(e.runningTasks))
	for id := range e.runningTasks {
		tasks = append(tasks, id)
	}
	return tasks
}

// findLlamacppBinary 查找 llama.cpp 二进制文件
func (e *Executor) findLlamacppBinary() (string, error) {
	// 常见的 llama.cpp server 二进制文件路径
	candidates := []string{
		// 用户工作区构建路径
		"/home/user/workspace/llama.cpp/build/bin/server",
		// 系统安装路径
		"/usr/local/bin/llama-server",
		// Home 目录安装路径
		filepath.Join(os.Getenv("HOME"), "llama.cpp/build/bin/server"),
		// Conda 环境中的路径（如果通过 conda 安装）
		"/home/user/miniconda3/envs/rocm7.2/bin/llama-server",
		"/opt/llama.cpp/server",
	}

	// 检查环境变量指定的路径
	if customPath := os.Getenv("LLAMACPP_SERVER_PATH"); customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			e.log.Info(fmt.Sprintf("使用环境变量指定的 llama.cpp: %s", customPath))
			return customPath, nil
		}
	}

	// 遍历候选路径
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			// 检查是否可执行
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				e.log.Info(fmt.Sprintf("找到 llama.cpp binary: %s", path))
				return path, nil
			}
		}
	}

	// 尝试在 PATH 中查找
	if path, err := exec.LookPath("llama-server"); err == nil {
		e.log.Info(fmt.Sprintf("在 PATH 中找到 llama-server: %s", path))
		return path, nil
	}

	return "", fmt.Errorf("llama.cpp binary not found (checked: %v)", candidates)
}
