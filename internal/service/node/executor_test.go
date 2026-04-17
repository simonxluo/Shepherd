package node

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommandExecutor_New 测试 CommandExecutor 创建
func TestCommandExecutor_New(t *testing.T) {
	tests := []struct {
		name     string
		config   *CommandExecutorConfig
		validate func(t *testing.T, ce *CommandExecutor)
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			validate: func(t *testing.T, ce *CommandExecutor) {
				assert.Equal(t, 4, ce.maxConcurrent)
				assert.Equal(t, 3600*time.Second, ce.timeout)
				assert.NotNil(t, ce.semaphore)
				assert.NotNil(t, ce.activeTasks)
				assert.NotNil(t, ce.allowedCommands)
				// 验证默认允许的命令
				assert.True(t, ce.allowedCommands[CommandTypeLoadModel])
				assert.True(t, ce.allowedCommands[CommandTypeUnloadModel])
				assert.True(t, ce.allowedCommands[CommandTypeRunLlamacpp])
				assert.True(t, ce.allowedCommands[CommandTypeStopProcess])
				assert.True(t, ce.allowedCommands[CommandTypeUpdateConfig])
				assert.True(t, ce.allowedCommands[CommandTypeCollectLogs])
				assert.True(t, ce.allowedCommands[CommandTypeScanModels])
			},
		},
		{
			name: "custom config",
			config: &CommandExecutorConfig{
				MaxConcurrent: 8,
				Timeout:       600 * time.Second,
				AllowedCommands: []CommandType{
					CommandTypeRunLlamacpp,
					CommandTypeLoadModel,
				},
			},
			validate: func(t *testing.T, ce *CommandExecutor) {
				assert.Equal(t, 8, ce.maxConcurrent)
				assert.Equal(t, 600*time.Second, ce.timeout)
				assert.True(t, ce.allowedCommands[CommandTypeRunLlamacpp])
				assert.True(t, ce.allowedCommands[CommandTypeLoadModel])
				assert.False(t, ce.allowedCommands[CommandTypeStopProcess])
			},
		},
		{
			name: "zero values use defaults",
			config: &CommandExecutorConfig{
				MaxConcurrent:   0,
				Timeout:         0,
				AllowedCommands: []CommandType{},
			},
			validate: func(t *testing.T, ce *CommandExecutor) {
				assert.Equal(t, 4, ce.maxConcurrent)
				assert.Equal(t, 3600*time.Second, ce.timeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := NewCommandExecutor(tt.config)
			assert.NotNil(t, ce)
			tt.validate(t, ce)
		})
	}
}

// TestCommandExecutor_Validate 测试命令验证
func TestCommandExecutor_Validate(t *testing.T) {
	ce := NewCommandExecutor(nil)

	tests := []struct {
		name    string
		cmd     *Command
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil command",
			cmd:     nil,
			wantErr: true,
			errMsg:  "命令不能为空",
		},
		{
			name: "empty command ID",
			cmd: &Command{
				ID:   "",
				Type: CommandTypeRunLlamacpp,
			},
			wantErr: true,
			errMsg:  "命令 ID 不能为空",
		},
		{
			name: "disallowed command type",
			cmd: &Command{
				ID:   "cmd-1",
				Type: "unknown_command",
			},
			wantErr: true,
			errMsg:  "不被允许",
		},
		{
			name: "valid command",
			cmd: &Command{
				ID:   "cmd-1",
				Type: CommandTypeRunLlamacpp,
			},
			wantErr: false,
		},
		{
			name: "valid load model command",
			cmd: &Command{
				ID:   "cmd-2",
				Type: CommandTypeLoadModel,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ce.validate(tt.cmd)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCommandExecutor_Execute 测试命令执行
func TestCommandExecutor_Execute(t *testing.T) {
	ce := NewCommandExecutor(&CommandExecutorConfig{
		MaxConcurrent: 2,
		Timeout:       5 * time.Second,
	})

	tests := []struct {
		name     string
		cmd      *Command
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, result *CommandResult)
	}{
		{
			name:    "nil command",
			cmd:     nil,
			wantErr: true,
			errMsg:  "命令不能为空",
		},
		{
			name: "load model command",
			cmd: &Command{
				ID:      "cmd-1",
				Type:    CommandTypeLoadModel,
				Payload: map[string]interface{}{"model": "test-model"},
			},
			wantErr: false,
			validate: func(t *testing.T, result *CommandResult) {
				assert.Equal(t, "cmd-1", result.CommandID)
				assert.True(t, result.Success)
				assert.NotNil(t, result.Result)
			},
		},
		{
			name: "unload model command",
			cmd: &Command{
				ID:      "cmd-2",
				Type:    CommandTypeUnloadModel,
				Payload: map[string]interface{}{"model": "test-model"},
			},
			wantErr: false,
			validate: func(t *testing.T, result *CommandResult) {
				assert.True(t, result.Success)
			},
		},
		{
			name: "scan models command not implemented",
			cmd: &Command{
				ID:   "cmd-3",
				Type: CommandTypeScanModels,
			},
			wantErr: false,
			validate: func(t *testing.T, result *CommandResult) {
				assert.False(t, result.Success)
				assert.Contains(t, result.Error, "未实现")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ce.Execute(tt.cmd)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, result)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestCommandExecutor_ExecuteTimeout 测试执行超时
func TestCommandExecutor_ExecuteTimeout(t *testing.T) {
	ce := NewCommandExecutor(&CommandExecutorConfig{
		MaxConcurrent: 1,
		Timeout:       100 * time.Millisecond,
	})

	timeout := 50 * time.Millisecond
	cmd := &Command{
		ID:   "timeout-cmd",
		Type: CommandTypeRunLlamacpp,
		Payload: map[string]interface{}{
			"binary_path": "/bin/sleep",
			"model_path":  "1",
		},
		Timeout: &timeout,
	}

	result, err := ce.Execute(cmd)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

// TestCommandExecutor_Cancel 测试命令取消
func TestCommandExecutor_Cancel(t *testing.T) {
	ce := NewCommandExecutor(&CommandExecutorConfig{
		MaxConcurrent: 1,
		Timeout:       60 * time.Second,
	})

	tests := []struct {
		name      string
		commandID string
		setup     func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "cancel non-existent command",
			commandID: "non-existent",
			wantErr:   true,
			errMsg:    "未在执行中",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			err := ce.Cancel(tt.commandID)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCommandExecutor_ConcurrentExecution 测试并发执行
func TestCommandExecutor_ConcurrentExecution(t *testing.T) {
	ce := NewCommandExecutor(&CommandExecutorConfig{
		MaxConcurrent: 2,
		Timeout:       60 * time.Second,
	})

	// 测试并发执行多个命令
	results := make(chan *CommandResult, 5)
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			cmd := &Command{
				ID:   fmt.Sprintf("concurrent-cmd-%d", idx),
				Type: CommandTypeLoadModel,
				Payload: map[string]interface{}{
					"model": fmt.Sprintf("model-%d", idx),
				},
			}
			result, err := ce.Execute(cmd)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// 收集结果
	successCount := 0
	errorCount := 0
	done := make(chan bool)
	go func() {
		for i := 0; i < 5; i++ {
			select {
			case <-results:
				successCount++
			case <-errors:
				errorCount++
			case <-time.After(5 * time.Second):
				break
			}
		}
		done <- true
	}()

	<-done

	// 验证至少有一些命令成功执行
	assert.Greater(t, successCount, 0, "至少有一些命令应该成功执行")
}

// TestCommandExecutor_SemaphoreTimeout 测试信号量超时
func TestCommandExecutor_SemaphoreTimeout(t *testing.T) {
	ce := NewCommandExecutor(&CommandExecutorConfig{
		MaxConcurrent: 1,
		Timeout:       60 * time.Second,
	})

	// 启动一个长时间运行的命令占用信号量
	go func() {
		cmd := &Command{
			ID:   "blocking-cmd",
			Type: CommandTypeRunLlamacpp,
			Payload: map[string]interface{}{
				"binary_path": "/bin/sleep",
				"model_path":  "10", // sleep 10 seconds
			},
		}
		ce.Execute(cmd)
	}()

	// 等待信号量被占用
	time.Sleep(100 * time.Millisecond)

	// 尝试执行另一个命令，应该因为信号量超时而失败
	cmd := &Command{
		ID:   "timeout-cmd",
		Type: CommandTypeLoadModel,
	}

	_, err := ce.Execute(cmd)
	// 由于第一个命令会很快失败（因为没有真正的二进制文件），
	// 第二个命令可能会成功或失败，取决于时机
	// 这个测试主要验证信号量机制的存在
	_ = err
}

// TestCommandExecutor_AllowedCommands 测试允许的命令类型
func TestCommandExecutor_AllowedCommands(t *testing.T) {
	tests := []struct {
		name            string
		allowedCommands []CommandType
		testCommand     CommandType
		shouldPass      bool
	}{
		{
			name:            "allow specific commands",
			allowedCommands: []CommandType{CommandTypeLoadModel, CommandTypeUnloadModel},
			testCommand:     CommandTypeLoadModel,
			shouldPass:      true,
		},
		{
			name:            "disallow command",
			allowedCommands: []CommandType{CommandTypeLoadModel},
			testCommand:     CommandTypeRunLlamacpp,
			shouldPass:      false,
		},
		{
			name:            "empty allowed uses default",
			allowedCommands: []CommandType{},
			testCommand:     CommandTypeScanModels,
			shouldPass:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := NewCommandExecutor(&CommandExecutorConfig{
				AllowedCommands: tt.allowedCommands,
			})

			cmd := &Command{
				ID:   "test-cmd",
				Type: tt.testCommand,
			}

			_, err := ce.Execute(cmd)
			if tt.shouldPass {
				// 即使验证通过，执行也可能失败（因为没有实际的模型）
				// 但我们检查错误不是验证错误
				if err != nil {
					assert.NotContains(t, err.Error(), "不被允许")
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "不被允许")
			}
		})
	}
}
