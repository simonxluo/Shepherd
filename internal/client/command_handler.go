// Package client provides client node functionality for the distributed architecture.
// CommandHandler 处理 Master 下发的各种命令
package client

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/model"
	"github.com/shepherd-project/shepherd/Shepherd/internal/node"
	"github.com/shepherd-project/shepherd/Shepherd/internal/process"
	"github.com/shepherd-project/shepherd/Shepherd/internal/client/tester"
	"github.com/shepherd-project/shepherd/Shepherd/internal/types"
)

// CommandHandler 处理从 Master 接收到的命令
type CommandHandler struct {
	// executor 用于执行基础命令
	executor *node.CommandExecutor
	// modelManager 管理模型加载和卸载
	modelManager *model.Manager
	// processManager 管理进程生命周期
	processManager *process.Manager
	// logger 日志记录器
	logger *logger.Logger
	// nodeID 当前节点的 ID
	nodeID string
}

// CommandHandlerConfig 包含 CommandHandler 的配置
type CommandHandlerConfig struct {
	// Executor 基础命令执行器
	Executor *node.CommandExecutor
	// ModelManager 模型管理器
	ModelManager *model.Manager
	// ProcessManager 进程管理器
	ProcessManager *process.Manager
	// Logger 日志记录器
	Logger *logger.Logger
	// NodeID 当前节点 ID
	NodeID string
}

// NewCommandHandler 创建一个新的命令处理器
func NewCommandHandler(config *CommandHandlerConfig) *CommandHandler {
	if config == nil {
		config = &CommandHandlerConfig{}
	}

	return &CommandHandler{
		executor:       config.Executor,
		modelManager:   config.ModelManager,
		processManager: config.ProcessManager,
		logger:         config.Logger,
		nodeID:         config.NodeID,
	}
}

// Handle 处理单个命令并返回结果
// 这是 CommandHandler 的主要入口方法
func (ch *CommandHandler) Handle(command *node.Command) (*node.CommandResult, error) {
	if command == nil {
		return nil, fmt.Errorf("命令不能为空")
	}

	if ch.logger != nil {
		ch.logger.Info(fmt.Sprintf("处理命令: %s (类型: %s)", command.ID, command.Type))
	}

	startTime := time.Now()

	// 创建结果对象
	result := &node.CommandResult{
		CommandID:   command.ID,
		FromNodeID:  ch.nodeID,
		ToNodeID:    command.FromNodeID,
		CompletedAt: time.Now(),
	}

	// 根据命令类型分发处理
	var err error
	switch command.Type {
	case node.CommandTypeLoadModel:
		err = ch.handleLoadModel(command, result)
	case node.CommandTypeUnloadModel:
		err = ch.handleUnloadModel(command, result)
	case node.CommandTypeRunLlamacpp:
		err = ch.handleRunLlamacpp(command, result)
	case node.CommandTypeStopProcess:
		err = ch.handleStopProcess(command, result)
	case node.CommandTypeScanModels:
		err = ch.handleScanModels(command, result)
	case node.CommandTypeTestLlamacpp:
		err = ch.handleTestLlamacpp(command, result)
	case node.CommandTypeGetConfig:
		err = ch.handleGetConfig(command, result)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("未知的命令类型: %s", command.Type)
		err = fmt.Errorf("未知的命令类型: %s", command.Type)
	}

	// 计算执行时长
	result.Duration = time.Since(startTime).Milliseconds()
	result.CompletedAt = time.Now()

	if err != nil && result.Error == "" {
		result.Error = err.Error()
	}

	if ch.logger != nil {
		if result.Success {
			ch.logger.Info(fmt.Sprintf("命令执行成功: %s (耗时: %dms)", command.ID, result.Duration))
		} else {
			ch.logger.Error(fmt.Sprintf("命令执行失败: %s - %s", command.ID, result.Error))
		}
	}

	return result, err
}

// handleLoadModel 处理加载模型命令
// Payload 期望包含:
//   - model_id: 模型 ID (必需)
//   - ctx_size: 上下文大小 (可选)
//   - gpu_layers: GPU 层数 (可选)
//   - threads: 线程数 (可选)
//   - batch_size: 批次大小 (可选)
//   - temperature: 温度参数 (可选)
//   - top_p: top_p 参数 (可选)
//   - top_k: top_k 参数 (可选)
func (ch *CommandHandler) handleLoadModel(command *node.Command, result *node.CommandResult) error {
	modelID, ok := command.Payload["model_id"].(string)
	if !ok || modelID == "" {
		result.Success = false
		result.Error = "缺少必需的参数: model_id"
		return fmt.Errorf("缺少必需的参数: model_id")
	}

	if ch.logger != nil {
		ch.logger.Info(fmt.Sprintf("开始加载模型: %s", modelID))
	}

	if ch.modelManager == nil {
		result.Success = false
		result.Error = "模型管理器未初始化"
		if ch.logger != nil {
			ch.logger.Error("模型管理器未初始化")
		}
		return fmt.Errorf("模型管理器未初始化")
	}

	// 构建加载请求
	req := &model.LoadRequest{
		ModelID: modelID,
	}

	// 解析可选参数并记录
	if ctxSize, ok := command.Payload["ctx_size"].(float64); ok {
		req.CtxSize = int(ctxSize)
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("上下文大小: %d", req.CtxSize))
		}
	}
	if gpuLayers, ok := command.Payload["gpu_layers"].(float64); ok {
		req.GPULayers = int(gpuLayers)
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("GPU 层数: %d", req.GPULayers))
		}
	}
	if threads, ok := command.Payload["threads"].(float64); ok {
		req.Threads = int(threads)
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("线程数: %d", req.Threads))
		}
	}
	if batchSize, ok := command.Payload["batch_size"].(float64); ok {
		req.BatchSize = int(batchSize)
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("批次大小: %d", req.BatchSize))
		}
	}
	if temperature, ok := command.Payload["temperature"].(float64); ok {
		req.Temperature = temperature
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("温度: %.2f", req.Temperature))
		}
	}
	if topP, ok := command.Payload["top_p"].(float64); ok {
		req.TopP = topP
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("Top-P: %.2f", req.TopP))
		}
	}
	if topK, ok := command.Payload["top_k"].(float64); ok {
		req.TopK = int(topK)
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("Top-K: %d", req.TopK))
		}
	}
	if repeatPenalty, ok := command.Payload["repeat_penalty"].(float64); ok {
		req.RepeatPenalty = repeatPenalty
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("重复惩罚: %.2f", req.RepeatPenalty))
		}
	}
	if nPredict, ok := command.Payload["n_predict"].(float64); ok {
		req.NPredict = int(nPredict)
		if ch.logger != nil {
			ch.logger.Debug(fmt.Sprintf("预测令牌数: %d", req.NPredict))
		}
	}
	// === GPU Configuration ===
	// Devices ([]string from []interface{})
	if val, ok := command.Payload["devices"].([]interface{}); ok {
		devices := make([]string, 0, len(val))
		for _, v := range val {
			if s, ok := v.(string); ok {
				devices = append(devices, s)
			}
		}
		req.Devices = devices
	}
	// MainGPU (float64 -> int)
	if val, ok := command.Payload["mainGpu"].(float64); ok {
		req.MainGPU = int(val)
	}
	// SplitMode (string)
	if val, ok := command.Payload["splitMode"].(string); ok {
		req.SplitMode = val
	}
	// TensorSplit (string)
	if val, ok := command.Payload["tensorSplit"].(string); ok {
		req.TensorSplit = val
	}

	// === Vision/Multimodal ===
	if val, ok := command.Payload["mmprojPath"].(string); ok {
		req.MmprojPath = val
	}
	if val, ok := command.Payload["enableVision"].(bool); ok {
		req.EnableVision = val
	}

	// === Performance ===
	if val, ok := command.Payload["flashAttention"].(bool); ok {
		req.FlashAttention = val
	}
	if val, ok := command.Payload["noMmap"].(bool); ok {
		req.NoMmap = val
	}
	if val, ok := command.Payload["lockMemory"].(bool); ok {
		req.LockMemory = val
	}

	// === Server Features ===
	if val, ok := command.Payload["noWebUI"].(bool); ok {
		req.NoWebUI = val
	}
	if val, ok := command.Payload["enableMetrics"].(bool); ok {
		req.EnableMetrics = val
	}
	if val, ok := command.Payload["slotSavePath"].(string); ok {
		req.SlotSavePath = val
	}
	if val, ok := command.Payload["cacheRam"].(float64); ok {
		req.CacheRAM = int(val)
	}

	// === Chat Template ===
	if val, ok := command.Payload["chatTemplateFile"].(string); ok {
		req.ChatTemplateFile = val
	}
	if val, ok := command.Payload["chatTemplate"].(string); ok {
		req.ChatTemplate = val
	}
	if val, ok := command.Payload["disableJinja"].(bool); ok {
		req.DisableJinja = val
	}

	// === Runtime ===
	if val, ok := command.Payload["timeout"].(float64); ok {
		req.Timeout = int(val)
	}
	if val, ok := command.Payload["alias"].(string); ok {
		req.Alias = val
	}
	if val, ok := command.Payload["llamaCppPath"].(string); ok {
		req.CustomCmd = val
	}
	if val, ok := command.Payload["extraArgs"].(string); ok {
		req.ExtraParams = val
	}

	// === Batch Processing ===
	if val, ok := command.Payload["uBatchSize"].(float64); ok {
		req.UBatchSize = int(val)
	}
	if val, ok := command.Payload["parallelSlots"].(float64); ok {
		req.ParallelSlots = int(val)
	}

	// === KV Cache ===
	if val, ok := command.Payload["kvCacheTypeK"].(string); ok {
		req.KVCacheTypeK = val
	}
	if val, ok := command.Payload["kvCacheTypeV"].(string); ok {
		req.KVCacheTypeV = val
	}
	if val, ok := command.Payload["kvCacheUnified"].(bool); ok {
		req.KVCacheUnified = val
	}
	if val, ok := command.Payload["kvCacheSize"].(float64); ok {
		req.KVCacheSize = int(val)
	}

	// === Extended Sampling ===
	if val, ok := command.Payload["seed"].(float64); ok {
		req.Seed = int(val)
	}
	if val, ok := command.Payload["minP"].(float64); ok {
		req.MinP = val
	}
	if val, ok := command.Payload["presencePenalty"].(float64); ok {
		req.PresencePenalty = val
	}
	if val, ok := command.Payload["frequencyPenalty"].(float64); ok {
		req.FrequencyPenalty = val
	}
	if val, ok := command.Payload["repeatLastN"].(float64); ok {
		req.RepeatLastN = int(val)
	}
	if val, ok := command.Payload["typicalP"].(float64); ok {
		req.TypicalP = val
	}
	if val, ok := command.Payload["logitsAll"].(bool); ok {
		req.LogitsAll = val
	}
	if val, ok := command.Payload["reranking"].(bool); ok {
		req.Reranking = val
	}
	if val, ok := command.Payload["ignoreEos"].(bool); ok {
		req.IgnoreEOS = val
	}

	// === Thread Configuration ===
	if val, ok := command.Payload["threadsBatch"].(float64); ok {
		req.ThreadsBatch = int(val)
	}

	// === I/O Configuration ===
	if val, ok := command.Payload["directIo"].(string); ok {
		req.DirectIo = val
	}
	if val, ok := command.Payload["contextShift"].(bool); ok {
		req.ContextShift = val
	}

	// === Advanced ===
	if val, ok := command.Payload["contBatching"].(bool); ok {
		req.ContBatching = val
	}
	if val, ok := command.Payload["cachePrompt"].(bool); ok {
		req.CachePrompt = val
	}
	if val, ok := command.Payload["grammar"].(string); ok {
		req.Grammar = val
	}
	if val, ok := command.Payload["grammarFile"].(string); ok {
		req.GrammarFile = val
	}
	if val, ok := command.Payload["lora"].(string); ok {
		req.Lora = val
	}
	if val, ok := command.Payload["loraScaled"].(string); ok {
		req.LoraScaled = val
	}
	if val, ok := command.Payload["chatTemplateKwargs"].(string); ok {
		req.ChatTemplateKwargs = val
	}

	// === RoPE Scaling ===
	if val, ok := command.Payload["ropeScaling"].(string); ok {
		req.RopeScaling = val
	}
	if val, ok := command.Payload["ropeScale"].(float64); ok {
		req.RopeScale = val
	}
	if val, ok := command.Payload["ropeFreqBase"].(float64); ok {
		req.RopeFreqBase = val
	}
	if val, ok := command.Payload["ropeFreqScale"].(float64); ok {
		req.RopeFreqScale = val
	}

	if ch.logger != nil {
		ch.logger.Info(fmt.Sprintf("模型参数解析完成，准备执行加载: %s", modelID))
	}

	// 执行加载
	if ch.logger != nil {
		ch.logger.Debug(fmt.Sprintf("调用 ModelManager.Load() 加载模型: %s", modelID))
	}
	loadResult, err := ch.modelManager.Load(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("加载模型失败: %v", err)
		if ch.logger != nil {
			ch.logger.Error(fmt.Sprintf("加载模型 %s 失败: %v", modelID, err))
		}
		return err
	}

	if ch.logger != nil {
		if loadResult.Success {
			ch.logger.Info(fmt.Sprintf("模型 %s 加载成功 (端口: %d, 耗时: %dms, 上下文大小: %d)",
				loadResult.ModelID, loadResult.Port, loadResult.Duration.Milliseconds(), loadResult.CtxSize))
		} else {
			ch.logger.Error(fmt.Sprintf("模型 %s 加载失败: %s", modelID, loadResult.Error.Error()))
		}
	}

	result.Success = loadResult.Success
	if loadResult.Success {
		result.Result = map[string]interface{}{
			"model_id": loadResult.ModelID,
			"port":     loadResult.Port,
			"ctx_size": loadResult.CtxSize,
			"duration": loadResult.Duration.Milliseconds(),
		}
	} else {
		result.Error = loadResult.Error.Error()
	}

	return nil
}

// handleUnloadModel 处理卸载模型命令
// Payload 期望包含:
//   - model_id: 模型 ID (必需)
func (ch *CommandHandler) handleUnloadModel(command *node.Command, result *node.CommandResult) error {
	modelID, ok := command.Payload["model_id"].(string)
	if !ok || modelID == "" {
		result.Success = false
		result.Error = "缺少必需的参数: model_id"
		if ch.logger != nil {
			ch.logger.Error("卸载模型失败: 缺少必需的参数 model_id")
		}
		return fmt.Errorf("缺少必需的参数: model_id")
	}

	if ch.logger != nil {
		ch.logger.Info(fmt.Sprintf("开始卸载模型: %s", modelID))
	}

	if ch.modelManager == nil {
		result.Success = false
		result.Error = "模型管理器未初始化"
		if ch.logger != nil {
			ch.logger.Error("卸载模型失败: 模型管理器未初始化")
		}
		return fmt.Errorf("模型管理器未初始化")
	}

	// 执行卸载
	if ch.logger != nil {
		ch.logger.Debug(fmt.Sprintf("调用 ModelManager.Unload() 卸载模型: %s", modelID))
	}
	err := ch.modelManager.Unload(modelID)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("卸载模型失败: %v", err)
		if ch.logger != nil {
			ch.logger.Error(fmt.Sprintf("卸载模型 %s 失败: %v", modelID, err))
		}
		return err
	}

	result.Success = true
	result.Result = map[string]interface{}{
		"model_id": modelID,
		"unloaded": true,
	}

	if ch.logger != nil {
		ch.logger.Info(fmt.Sprintf("模型 %s 卸载成功", modelID))
	}

	return nil
}

// handleRunLlamacpp 直接运行 llama.cpp 命令
// Payload 期望包含:
//   - binary_path: llama.cpp 可执行文件路径 (必需)
//   - model_path: 模型文件路径 (必需)
//   - args: 额外参数列表 (可选)
//   - timeout: 超时时间(秒) (可选)
func (ch *CommandHandler) handleRunLlamacpp(command *node.Command, result *node.CommandResult) error {
	binaryPath, ok := command.Payload["binary_path"].(string)
	if !ok || binaryPath == "" {
		result.Success = false
		result.Error = "缺少必需的参数: binary_path"
		return fmt.Errorf("缺少必需的参数: binary_path")
	}

	modelPath, ok := command.Payload["model_path"].(string)
	if !ok || modelPath == "" {
		result.Success = false
		result.Error = "缺少必需的参数: model_path"
		return fmt.Errorf("缺少必需的参数: model_path")
	}

	// 构建命令参数
	args := []string{"-m", modelPath}

	// 添加额外参数
	if extraArgs, ok := command.Payload["args"].([]interface{}); ok {
		for _, arg := range extraArgs {
			if strArg, ok := arg.(string); ok {
				args = append(args, strArg)
			}
		}
	}

	// 设置超时
	timeout := 300 * time.Second
	if timeoutSec, ok := command.Payload["timeout"].(float64); ok && timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}
	if command.Timeout != nil {
		timeout = *command.Timeout
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 执行命令
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.Error = "命令执行超时"
		result.Result = map[string]interface{}{
			"output": string(output),
		}
		return fmt.Errorf("命令执行超时")
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("命令执行失败: %v", err)
		result.Result = map[string]interface{}{
			"output":    string(output),
			"exit_code": cmd.ProcessState.ExitCode(),
		}
		return err
	}

	result.Success = true
	result.Result = map[string]interface{}{
		"output":    string(output),
		"exit_code": cmd.ProcessState.ExitCode(),
	}

	return nil
}

// handleStopProcess 处理停止进程命令
// Payload 期望包含:
//   - model_id: 模型 ID (必需)
//   - force: 是否强制停止 (可选，默认 false)
func (ch *CommandHandler) handleStopProcess(command *node.Command, result *node.CommandResult) error {
	modelID, ok := command.Payload["model_id"].(string)
	if !ok || modelID == "" {
		result.Success = false
		result.Error = "缺少必需的参数: model_id"
		return fmt.Errorf("缺少必需的参数: model_id")
	}

	if ch.processManager == nil {
		result.Success = false
		result.Error = "进程管理器未初始化"
		return fmt.Errorf("进程管理器未初始化")
	}

	// 检查是否强制停止
	force := false
	if f, ok := command.Payload["force"].(bool); ok {
		force = f
	}

	// 获取进程信息
	_, exists := ch.processManager.Get(modelID)
	if !exists {
		result.Success = false
		result.Error = fmt.Sprintf("进程不存在: %s", modelID)
		return fmt.Errorf("进程不存在: %s", modelID)
	}

	// 停止进程 - Process.Stop() 内部已实现优雅关闭和强制终止逻辑
	err := ch.processManager.Stop(modelID)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("停止进程失败: %v", err)
		return err
	}

	result.Success = true
	result.Result = map[string]interface{}{
		"model_id": modelID,
		"stopped":  true,
		"force":    force,
	}

	return nil
}

// handleScanModels 处理扫描模型命令
// Payload 可选包含:
//   - paths: 要扫描的路径列表 (可选，默认使用配置的扫描路径)
func (ch *CommandHandler) handleScanModels(command *node.Command, result *node.CommandResult) error {
	if ch.modelManager == nil {
		result.Success = false
		result.Error = "模型管理器未初始化"
		return fmt.Errorf("模型管理器未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 执行扫描
	scanResult, err := ch.modelManager.Scan(ctx)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("扫描模型失败: %v", err)
		return err
	}

	// 构建模型信息列表
	models := make([]map[string]interface{}, 0, len(scanResult.Models))
	for _, m := range scanResult.Models {
		modelInfo := map[string]interface{}{
			"id":   m.ID,
			"name": m.Name,
			"path": m.Path,
			"size": m.Size,
		}
		if m.Metadata != nil {
			modelInfo["architecture"] = m.Metadata.Architecture
			modelInfo["context_length"] = m.Metadata.ContextLength
			modelInfo["embedding_length"] = m.Metadata.EmbeddingLength
		}
		models = append(models, modelInfo)
	}

	// 构建错误信息列表
	errors := make([]map[string]string, 0, len(scanResult.Errors))
	for _, e := range scanResult.Errors {
		errors = append(errors, map[string]string{
			"path":  e.Path,
			"error": e.Error,
		})
	}

	result.Success = true
	result.Result = map[string]interface{}{
		"models_found":  len(scanResult.Models),
		"models":        models,
		"errors":        errors,
		"error_count":   len(scanResult.Errors),
		"duration_ms":   scanResult.Duration.Milliseconds(),
		"total_files":   scanResult.TotalFiles,
		"matched_files": scanResult.MatchedFiles,
		"scanned_at":    scanResult.ScannedAt.Format(time.RFC3339),
	}

	return nil
}

// GetModelManager 返回模型管理器
func (ch *CommandHandler) GetModelManager() *model.Manager {
	return ch.modelManager
}

// GetProcessManager 返回进程管理器
func (ch *CommandHandler) GetProcessManager() *process.Manager {
	return ch.processManager
}

// GetExecutor 返回命令执行器
func (ch *CommandHandler) GetExecutor() *node.CommandExecutor {
	return ch.executor
}

// SetNodeID 设置节点 ID
func (ch *CommandHandler) SetNodeID(nodeID string) {
	ch.nodeID = nodeID
}

// GetNodeID 获取节点 ID
func (ch *CommandHandler) GetNodeID() string {
	return ch.nodeID
}

// handleTestLlamacpp 处理测试 llama.cpp 可用性命令
// Payload 可选包含:
//   - binary_path: 指定测试的二进制路径 (可选，默认测试所有)
func (ch *CommandHandler) handleTestLlamacpp(command *node.Command, result *node.CommandResult) error {
	llamacppTester := tester.NewTester(30 * time.Second)

	var testResult *types.LlamacppTestResult
	if binaryPath, ok := command.Payload["binary_path"].(string); ok && binaryPath != "" {
		testResult = llamacppTester.TestSpecific(binaryPath)
	} else {
		testResult = llamacppTester.TestAll()
	}

	result.Success = testResult.Success
	result.Result = map[string]interface{}{
		"success":     testResult.Success,
		"binary_path": testResult.BinaryPath,
		"version":     testResult.Version,
		"output":      testResult.Output,
		"error":       testResult.Error,
		"tested_at":   testResult.TestedAt,
		"duration":    testResult.Duration,
	}

	if !testResult.Success {
		result.Error = testResult.Error
	}

	return nil
}

// handleGetConfig 处理获取配置信息命令
func (ch *CommandHandler) handleGetConfig(command *node.Command, result *node.CommandResult) error {
	// 配置收集器需要在创建 CommandHandler 时传入，这里简化处理
	// 实际实现时应该通过 config 获取
	result.Success = true
	result.Result = map[string]interface{}{
		"message": "配置信息收集功能需要在初始化时提供 config 对象",
	}
	return nil
}
