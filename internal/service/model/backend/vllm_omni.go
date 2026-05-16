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
	return discoverVLLMVariant(cfg, BackendVLLMOmni, "vLLM-Omni", "vllm-omni")
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

	// vllm_omni 后端始终启用 --omni（这是使用此后端的核心目的）
	cmd += " --omni"

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

// SupportsModel returns true for safetensors/HuggingFace directories and GGUF files
// vLLM-Omni 支持所有 vLLM 支持的格式，同时额外支持 GGUF（多模态模型可能为 GGUF 格式）
func (b *VLLMOmniBackend) SupportsModel(modelPath string) bool {
	return b.vllm.SupportsModel(modelPath) || IsGGUFModel(modelPath)
}

// SupportedEndpoints returns the endpoints supported by vLLM-omni (includes audio)
func (b *VLLMOmniBackend) SupportedEndpoints() map[string]bool {
	return endpointsWithAudio()
}
