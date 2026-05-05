package backend

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

// VLLMOmniBackend implements Backend for vLLM-omni (multimodal vLLM fork)
type VLLMOmniBackend struct {
	vllm VLLMBackend
}

// NewVLLMOmniBackend creates a new vLLM-omni backend instance
func NewVLLMOmniBackend() *VLLMOmniBackend {
	return &VLLMOmniBackend{}
}

func (b *VLLMOmniBackend) Type() BackendType { return BackendVLLMOmni }

// buildEnvWithVars 构建包含自定义环境变量的进程环境
func buildEnvWithVars(envVars []string) []string {
	env := os.Environ()
	for _, ev := range envVars {
		if idx := strings.Index(ev, "="); idx > 0 {
			key := ev[:idx]
			prefix := key + "="
			found := false
			for i, e := range env {
				if strings.HasPrefix(e, prefix) {
					env[i] = ev
					found = true
					break
				}
			}
			if !found {
				env = append(env, ev)
			}
		}
	}
	return env
}

// Discover validates that vllm-omni is available in the configured conda environment
func (b *VLLMOmniBackend) Discover(cfg *BackendConfig) (*BackendInfo, error) {
	info := &BackendInfo{
		Type: BackendVLLMOmni,
		Name: "vLLM-Omni",
	}

	if cfg == nil {
		info.Available = false
		return info, nil
	}

	env := buildEnvWithVars(cfg.EnvVars)

	// 优先检查 ServeBin（直接指定 vllm-omni 二进制路径）
	if cfg.ServeBin != "" {
		cmd := exec.Command(cfg.ServeBin, "--version")
		cmd.Env = env
		if output, err := cmd.CombinedOutput(); err == nil {
			info.Version = strings.TrimSpace(string(output))
			info.Available = true
			info.BinPath = cfg.ServeBin
			return info, nil
		}
	}

	// 检查 BinPaths 配置的路径中是否有 vllm-omni 二进制
	if len(cfg.BinPaths) > 0 {
		for _, p := range cfg.BinPaths {
			candidate := filepath.Join(p, "vllm-omni")
			cmd := exec.Command(candidate, "--version")
			cmd.Env = env
			if output, err := cmd.CombinedOutput(); err == nil {
				info.Version = strings.TrimSpace(string(output))
				info.Available = true
				info.BinPath = candidate
				return info, nil
			}
		}
	}

	if cfg.CondaEnv == "" {
		info.Available = false
		return info, nil
	}

	// 通过 conda run 检查 vllm-omni 是否可用
	condaPath := cfg.CondaPath
	if condaPath == "" {
		condaPath = "conda"
	}

	cmd := exec.Command(condaPath, "run", "--no-banner", "-n", cfg.CondaEnv, "vllm-omni", "--version")
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warnf("vLLM-omni discovery failed: condaEnv=%s, error=%v, output=%s", cfg.CondaEnv, err, string(output))
		info.Available = false
		return info, nil
	}

	version := strings.TrimSpace(string(output))
	info.Version = version
	info.Available = true
	info.CondaEnv = cfg.CondaEnv
	info.CondaPath = cfg.CondaPath

	if cfg.ServeBin != "" {
		info.BinPath = cfg.ServeBin
	}

	return info, nil
}

// BuildStartConfig constructs the vllm-omni serve command with multimodal parameters
func (b *VLLMOmniBackend) BuildStartConfig(info *BackendInfo, req *LoadRequest) (*StartConfig, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if info.CondaEnv == "" && info.BinPath == "" {
		return nil, fmt.Errorf("vLLM-omni requires a conda environment or binary path")
	}

	p := req.VLLOmniParams
	if p == nil {
		p = &VLLOmniLoadParams{}
	}

	// Create a vLLM LoadRequest with the embedded params
	vllmReq := &LoadRequest{
		ModelPath:  req.ModelPath,
		Port:       req.Port,
		CtxSize:    req.CtxSize,
		GPULayers:  req.GPULayers,
		Threads:    req.Threads,
		Devices:    req.Devices,
		VLLMParams: &p.VLLMLoadParams,
	}

	startCfg, err := b.vllm.BuildStartConfig(info, vllmReq)
	if err != nil {
		return nil, err
	}

	// Replace "vllm serve" with "vllm-omni serve" in the command
	cmd := startCfg.Command
	cmd = strings.Replace(cmd, "vllm serve", "vllm-omni serve", 1)

	// 添加 --omni 标志（启用 omni 多模态模式）
	if p.Omni {
		cmd += " --omni"
	}

	// Append multimodal-specific parameters
	if p.VideoPruningRate > 0 {
		cmd += fmt.Sprintf(" --video-pruning-rate %.2f", p.VideoPruningRate)
	}
	if p.MMTensorIPC {
		cmd += " --mm-tensor-ipc"
	}

	startCfg.Command = cmd
	startCfg.BackendType = BackendVLLMOmni
	return startCfg, nil
}

// IsLoadComplete detects vLLM-omni load completion (same signal as vLLM)
func (b *VLLMOmniBackend) IsLoadComplete(outputLine string) bool {
	return b.vllm.IsLoadComplete(outputLine)
}

// CheckHealth performs an HTTP health check (same as vLLM)
func (b *VLLMOmniBackend) CheckHealth(port int) (*HealthResult, error) {
	return b.vllm.CheckHealth(port)
}

// SupportsModel returns true for safetensors/HuggingFace directories (multimodal models)
func (b *VLLMOmniBackend) SupportsModel(modelPath string) bool {
	return b.vllm.SupportsModel(modelPath)
}

// SupportedEndpoints returns the endpoints supported by vLLM-omni (includes audio)
func (b *VLLMOmniBackend) SupportedEndpoints() map[string]bool {
	return endpointsWithAudio()
}
