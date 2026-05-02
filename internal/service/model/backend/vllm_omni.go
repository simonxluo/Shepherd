package backend

import (
	"fmt"
	"strings"
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

// Discover validates that vllm-omni is available in the configured conda environment
func (b *VLLMOmniBackend) Discover(cfg *BackendConfig) (*BackendInfo, error) {
	info := &BackendInfo{
		Type: BackendVLLMOmni,
		Name: "vLLM-Omni",
	}

	if cfg == nil || cfg.CondaEnv == "" {
		info.Available = false
		return info, nil
	}

	// Use the same discovery as vLLM but look for vllm-omni
	return b.vllm.Discover(&BackendConfig{
		Type:      BackendVLLMOmni,
		CondaEnv:  cfg.CondaEnv,
		CondaPath: cfg.CondaPath,
		ServeBin:  cfg.ServeBin,
	})
}

// BuildStartConfig constructs the vllm-omni serve command with multimodal parameters
func (b *VLLMOmniBackend) BuildStartConfig(info *BackendInfo, req *LoadRequest) (*StartConfig, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if info.CondaEnv == "" {
		return nil, fmt.Errorf("vLLM-omni requires a conda environment")
	}

	// Build the base vLLM command first, then replace "vllm serve" with "vllm-omni serve"
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
