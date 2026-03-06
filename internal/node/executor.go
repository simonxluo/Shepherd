package node

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
)

type CommandExecutor struct {
	maxConcurrent   int
	timeout         time.Duration
	allowedCommands map[CommandType]bool
	activeTasks     map[string]*taskExec
	mu              sync.RWMutex
	semaphore       chan struct{}
	log             *logger.Logger
}

type taskExec struct {
	cmd       *Command
	startTime time.Time
	cancel    context.CancelFunc
	osCmd     *exec.Cmd
}

type CommandExecutorConfig struct {
	MaxConcurrent   int
	Timeout         time.Duration
	AllowedCommands []CommandType
	Logger          *logger.Logger
}

func NewCommandExecutor(config *CommandExecutorConfig) *CommandExecutor {
	if config == nil {
		config = &CommandExecutorConfig{}
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 4
	}
	if config.Timeout == 0 {
		config.Timeout = 3600 * time.Second
	}

	allowed := make(map[CommandType]bool)
	cmds := config.AllowedCommands
	if len(cmds) == 0 {
		cmds = []CommandType{
			CommandTypeLoadModel,
			CommandTypeUnloadModel,
			CommandTypeRunLlamacpp,
			CommandTypeStopProcess,
			CommandTypeUpdateConfig,
			CommandTypeCollectLogs,
			CommandTypeScanModels,
		}
	}
	for _, c := range cmds {
		allowed[c] = true
	}

	return &CommandExecutor{
		maxConcurrent:   config.MaxConcurrent,
		timeout:         config.Timeout,
		allowedCommands: allowed,
		activeTasks:     make(map[string]*taskExec),
		semaphore:       make(chan struct{}, config.MaxConcurrent),
		log:             config.Logger,
	}
}

func (ce *CommandExecutor) Execute(command *Command) (*CommandResult, error) {
	if err := ce.validate(command); err != nil {
		return nil, err
	}

	select {
	case ce.semaphore <- struct{}{}:
		defer func() { <-ce.semaphore }()
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("获取执行槽超时")
	}

	timeout := ce.timeout
	if command.Timeout != nil {
		timeout = *command.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	task := &taskExec{
		cmd:       command,
		startTime: time.Now(),
		cancel:    cancel,
	}

	ce.mu.Lock()
	ce.activeTasks[command.ID] = task
	ce.mu.Unlock()

	defer func() {
		ce.mu.Lock()
		delete(ce.activeTasks, command.ID)
		ce.mu.Unlock()
	}()

	result := ce.run(ctx, task)
	result.Duration = time.Since(task.startTime).Milliseconds()
	return result, nil
}

func (ce *CommandExecutor) Cancel(commandID string) error {
	ce.mu.RLock()
	task, exists := ce.activeTasks[commandID]
	ce.mu.RUnlock()

	if !exists {
		return fmt.Errorf("命令 %s 未在执行中", commandID)
	}

	task.cancel()
	if task.osCmd != nil && task.osCmd.Process != nil {
		utils.SignalQuietly(task.osCmd.Process, syscall.SIGTERM)
		go func() {
			time.Sleep(5 * time.Second)
			if task.osCmd != nil && task.osCmd.Process != nil {
				utils.KillQuietly(task.osCmd.Process)
			}
		}()
	}
	return nil
}

func (ce *CommandExecutor) validate(command *Command) error {
	if command == nil {
		return fmt.Errorf("命令不能为空")
	}
	if command.ID == "" {
		return fmt.Errorf("命令 ID 不能为空")
	}
	if !ce.allowedCommands[command.Type] {
		return fmt.Errorf("命令类型 %s 不被允许", command.Type)
	}
	return nil
}

func (ce *CommandExecutor) run(ctx context.Context, task *taskExec) *CommandResult {
	result := &CommandResult{
		CommandID: task.cmd.ID,
		Success:   false,
	}

	switch task.cmd.Type {
	case CommandTypeRunLlamacpp:
		return ce.runLlamacpp(ctx, task)
	case CommandTypeLoadModel, CommandTypeUnloadModel:
		result.Success = true
		result.Result = map[string]interface{}{"message": "命令已接收"}
	default:
		result.Error = fmt.Sprintf("未实现: %s", task.cmd.Type)
	}

	return result
}

func (ce *CommandExecutor) runLlamacpp(ctx context.Context, task *taskExec) *CommandResult {
	result := &CommandResult{
		CommandID: task.cmd.ID,
		Success:   false,
	}

	binaryPath, _ := task.cmd.Payload["binary_path"].(string)
	modelPath, _ := task.cmd.Payload["model_path"].(string)

	if binaryPath == "" || modelPath == "" {
		result.Error = "缺少必要参数"
		return result
	}

	args := []string{"-m", modelPath}
	if extraArgs, ok := task.cmd.Payload["args"].([]string); ok {
		args = append(args, extraArgs...)
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	task.osCmd = cmd

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = err.Error()
		result.Result = map[string]interface{}{"output": string(output)}
		return result
	}

	result.Success = true
	result.Result = map[string]interface{}{"output": string(output)}
	return result
}
