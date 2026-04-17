// Package model provides integration tests for model loading/unloading
package model

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/stretchr/testify/require"
)

// newIntegrationTestLogger 创建集成测试用 logger
func newIntegrationTestLogger() *logger.Logger {
	logCfg := &config.LogConfig{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	}
	log, _ := logger.NewLogger(logCfg, "integration-test")
	return log
}

// findLlamacppServerBinary 查找 llama.cpp server 二进制目录
func findLlamacppServerBinary(t *testing.T) string {
	candidates := []string{
		"/usr/local/bin",
		"/home/user/workspace/llama.cpp/build/bin",
		filepath.Join(os.Getenv("HOME"), "llama.cpp/build/bin"),
		"/home/user/miniconda3/envs/rocm7.2/bin",
	}

	for _, path := range candidates {
		serverPath := filepath.Join(path, "llama-server")
		if info, err := os.Stat(serverPath); err == nil && !info.IsDir() {
			t.Logf("找到 llama.cpp 目录: %s", path)
			return path
		}
	}

	t.Skip("未找到 llama.cpp server，跳过集成测试")
	return ""
}

// findTestModel 查找测试用的 GGUF 模型文件
func findTestModel(t *testing.T) string {
	// 使用硬编码的小模型路径（Qwen3-Embedding-0.6B，约1.2GB）
	smallModel := "/home/user/workspace/LlamacppServer/build/models/Qwen/Qwen3-Embedding-0.6B-GGUF/Qwen3-Embedding-0.6B-f16/Qwen3-Embedding-0.6B-f16.gguf"

	if _, err := os.Stat(smallModel); err == nil {
		t.Logf("找到测试模型: %s", smallModel)
		return smallModel
	}

	t.Skip("未找到测试用的 GGUF 模型文件: " + smallModel)
	return ""
}

// waitForServerReady 等待服务器 HTTP 就绪（通过健康检查）
func waitForServerReady(port int, timeout time.Duration, log *logger.Logger) bool {
	url := fmt.Sprintf("http://localhost:%d/health", port)
	client := &http.Client{Timeout: 2 * time.Second}
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					log.Info(fmt.Sprintf("✓ 服务器就绪: %s", url))
					return true
				}
			}
		case <-timeoutChan:
			log.Error(fmt.Sprintf("✗ 服务器就绪超时: %s", url))
			return false
		}
	}
}

// TestIntegrationProcessStartOnly 快速测试：仅测试进程启动（不等待完全加载）
func TestIntegrationProcessStartOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 (使用 -short 标志)")
	}

	binPath := findLlamacppServerBinary(t)
	if binPath == "" {
		return
	}

	modelPath := findTestModel(t)
	if modelPath == "" {
		return
	}

	log := newIntegrationTestLogger()
	llamaServerPath := filepath.Join(binPath, "llama-server")

	// 查找可用端口
	port := 18181
	args := []string{
		"-m", modelPath,
		"--port", fmt.Sprintf("%d", port),
		"-c", "128", // 极小上下文
		"--n-gpu-layers", "0",
		"--threads", "1",
		"--no-mmap",
		"-fa", "on", // flash attention
	}

	log.Info(fmt.Sprintf("启动命令: %s %s", llamaServerPath, strings.Join(args, " ")))

	cmd := exec.Command(llamaServerPath, args...)

	// 启动进程（不捕获输出，仅验证启动）
	err := cmd.Start()
	require.NoError(t, err)

	pid := cmd.Process.Pid
	log.Info(fmt.Sprintf("进程已启动: PID=%d", pid))

	// 等待 5 秒确认进程运行
	time.Sleep(5 * time.Second)

	// 检查进程是否仍在运行（通过 /proc 检查）
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
		t.Errorf("进程意外退出 (PID=%d)", pid)
	} else {
		log.Info("✓ 进程运行正常")
	}

	log.Info("开始清理...")

	// 停止进程
	cmd.Process.Kill()
	cmd.Wait()

	log.Info("✓ 进程已停止")
}

// TestIntegrationModelLoadUnload 测试完整的模型加载和卸载流程
func TestIntegrationModelLoadUnload(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 (使用 -short 标志)")
	}

	binPath := findLlamacppServerBinary(t)
	if binPath == "" {
		return
	}

	modelPath := findTestModel(t)
	if modelPath == "" {
		return
	}

	log := newIntegrationTestLogger()
	llamaServerPath := filepath.Join(binPath, "llama-server")

	// 分配端口
	port := 18182

	log.Info(fmt.Sprintf("使用端口: %d", port))
	log.Info(fmt.Sprintf("使用模型: %s", modelPath))

	// 构建命令 - 使用最小配置加快加载
	args := []string{
		"-m", modelPath,
		"--port", fmt.Sprintf("%d", port),
		"-c", "512",
		"--n-gpu-layers", "0",
		"--threads", "2",
		"--no-mmap",
		"-fa", "on", // flash attention
	}

	log.Info(fmt.Sprintf("启动命令: %s %s", llamaServerPath, strings.Join(args, " ")))

	// 直接创建命令
	cmd := exec.Command(llamaServerPath, args...)

	// 启动进程
	err := cmd.Start()
	require.NoError(t, err)

	pid := cmd.Process.Pid
	log.Info(fmt.Sprintf("进程已启动: PID=%d", pid))

	// 等待服务器就绪（使用 HTTP 健康检查）
	log.Info("等待模型加载完成（最多 2 分钟）...")
	ready := waitForServerReady(port, 2*time.Minute, log)

	if !ready {
		// 超时，但进程可能仍在运行
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatal("模型加载超时（2 分钟）")
	}

	log.Info("✓ 模型加载完成")

	// 验证进程仍在运行
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
		t.Fatal("加载完成后进程意外退出")
	}

	log.Info("✓ 进程运行正常")

	// 测试卸载
	log.Info("开始卸载模型...")
	cmd.Process.Kill()
	cmd.Wait()

	time.Sleep(1 * time.Second)

	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err == nil {
		t.Error("进程应该已停止")
	}

	log.Info("✓ 模型已卸载")
}

// TestIntegrationHealthCheck 测试 HTTP 健康检查
func TestIntegrationHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 (使用 -short 标志)")
	}

	binPath := findLlamacppServerBinary(t)
	if binPath == "" {
		return
	}

	modelPath := findTestModel(t)
	if modelPath == "" {
		return
	}

	log := newIntegrationTestLogger()
	llamaServerPath := filepath.Join(binPath, "llama-server")
	port := 18183

	args := []string{
		"-m", modelPath,
		"--port", fmt.Sprintf("%d", port),
		"-c", "128",
		"--n-gpu-layers", "0",
		"--threads", "1",
		"--no-mmap",
		"-fa", "on",
	}

	cmd := exec.Command(llamaServerPath, args...)
	err := cmd.Start()
	require.NoError(t, err)

	pid := cmd.Process.Pid
	log.Info(fmt.Sprintf("进程已启动: PID=%d", pid))

	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
		log.Info("✓ 清理完成")
	}()

	// 等待服务器就绪
	if !waitForServerReady(port, 2*time.Minute, log) {
		t.Fatal("服务器未能在 2 分钟内就绪")
	}

	// 测试各种 API 端点
	client := &http.Client{Timeout: 5 * time.Second}

	// 1. 健康检查
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	log.Info("✓ 健康检查通过")

	// 2. 模型信息
	resp, err = client.Get(fmt.Sprintf("http://localhost:%d/v1/models", port))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode)
	log.Info("✓ 模型信息获取成功")

	log.Info("✓ 所有健康检查通过")
}

// TestIntegrationProcessStopCrash 测试进程崩溃时的处理
func TestIntegrationProcessStopCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 (使用 -short 标志)")
	}

	binPath := findLlamacppServerBinary(t)
	if binPath == "" {
		return
	}

	modelPath := findTestModel(t)
	if modelPath == "" {
		return
	}

	log := newIntegrationTestLogger()
	llamaServerPath := filepath.Join(binPath, "llama-server")
	port := 18184

	args := []string{
		"-m", modelPath,
		"--port", fmt.Sprintf("%d", port),
		"-c", "128",
		"--n-gpu-layers", "0",
		"--threads", "1",
		"--no-mmap",
		"-fa", "on",
	}

	cmd := exec.Command(llamaServerPath, args...)
	err := cmd.Start()
	require.NoError(t, err)

	pid := cmd.Process.Pid
	log.Info(fmt.Sprintf("进程已启动: PID=%d", pid))

	// 等待服务器启动
	if !waitForServerReady(port, 2*time.Minute, log) {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatal("服务器未能就绪")
	}

	log.Info("✓ 服务器已就绪")

	// 强制杀死进程
	log.Info("强制杀死进程...")
	cmd.Process.Kill()
	cmd.Wait()

	// 等待进程完全退出
	time.Sleep(2 * time.Second)

	// 验证进程已停止
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err == nil {
		t.Error("进程应该已停止")
	}

	log.Info("✓ 进程已正确停止")
}

// TestIntegrationMultiplePorts 测试在不同端口启动多个实例
func TestIntegrationMultiplePorts(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试 (使用 -short 标志)")
	}

	binPath := findLlamacppServerBinary(t)
	if binPath == "" {
		return
	}

	modelPath := findTestModel(t)
	if modelPath == "" {
		return
	}

	log := newIntegrationTestLogger()
	llamaServerPath := filepath.Join(binPath, "llama-server")

	// 启动 3 个实例
	numInstances := 3
	basePort := 18190
	processes := make([]*exec.Cmd, numInstances)

	for i := 0; i < numInstances; i++ {
		port := basePort + i
		args := []string{
			"-m", modelPath,
			"--port", fmt.Sprintf("%d", port),
			"-c", "128",
			"--n-gpu-layers", "0",
			"--threads", "1",
			"--no-mmap",
			"-fa", "on",
		}

		cmd := exec.Command(llamaServerPath, args...)
		err := cmd.Start()
		require.NoError(t, err)

		processes[i] = cmd
		log.Info(fmt.Sprintf("实例 %d: PID=%d, Port=%d", i, cmd.Process.Pid, port))
	}

	// 清理所有进程
	defer func() {
		for i, cmd := range processes {
			if cmd != nil && cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
				log.Info(fmt.Sprintf("✓ 实例 %d 已停止", i))
			}
		}
	}()

	// 等待所有服务器就绪
	log.Info("等待所有服务器就绪...")
	for i := 0; i < numInstances; i++ {
		port := basePort + i
		if !waitForServerReady(port, 2*time.Minute, log) {
			t.Errorf("实例 %d (端口 %d) 未能就绪", i, port)
		}
	}

	log.Info(fmt.Sprintf("✓ 所有 %d 个实例都已就绪", numInstances))
}
