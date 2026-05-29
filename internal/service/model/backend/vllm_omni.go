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

// BuildStartConfig constructs the vllm-omni serve command with multimodal parameters.
// Unlike the previous implementation, this builds the command directly instead of
// delegating to VLLMBackend.BuildStartConfig and doing string replacement.
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

	// Build command prefix using shared helper with "vllm-omni" binary name
	args := buildVLLMPrefix(info, req, "vllm-omni")

	// Append common vLLM parameters (port, host, dtype, parallelism, etc.)
	vllmParams := &p.VLLMLoadParams
	args, err := appendVLLMArgs(args, req, vllmParams)
	if err != nil {
		return nil, err
	}

	// vllm_omni backend always enables --omni (core purpose of this backend; omitting it causes errors)
	args = append(args, "--omni")

	// Append multimodal-specific parameters
	if p.VideoPruningRate > 0 {
		args = append(args, "--video-pruning-rate", fmt.Sprintf("%.2f", p.VideoPruningRate))
	}
	if p.MMTensorIPC {
		args = append(args, "--mm-tensor-ipc")
	}

	cmd := quoteAndJoin(args)

	// Append extra args: global config ExtraArgs first, then model-level ExtraArgs (higher priority)
	if info.ExtraArgs != "" {
		cmd += " " + strings.TrimSpace(info.ExtraArgs)
	}
	if vllmParams.ExtraArgs != "" {
		cmd += " " + strings.TrimSpace(vllmParams.ExtraArgs)
	}

	// SkipLDLibraryPath: conda manages its own env, but direct binary mode may need LD_LIBRARY_PATH
	skipLD := info.CondaEnv != "" && info.BinPath == ""

	return &StartConfig{
		Command:           cmd,
		BinPath:           info.BinPath,
		BackendType:       BackendVLLMOmni,
		SkipLDLibraryPath: skipLD,
		CondaPath:         info.CondaPath,
	}, nil
}

// IsLoadComplete detects vLLM-omni load completion (same signal as vLLM)
func (b *VLLMOmniBackend) IsLoadComplete(outputLine string) bool {
	return b.vllm.IsLoadComplete(outputLine)
}

// CheckHealth performs an HTTP health check (same as vLLM)
func (b *VLLMOmniBackend) CheckHealth(port int) (*HealthResult, error) {
	return b.vllm.CheckHealth(port)
}

// SupportsModel returns true for safetensors/HuggingFace directories and GGUF files.
// vLLM-Omni supports all vLLM formats plus GGUF (multimodal models may be in GGUF format).
func (b *VLLMOmniBackend) SupportsModel(modelPath string) bool {
	return b.vllm.SupportsModel(modelPath) || IsGGUFModel(modelPath)
}

// SupportedEndpoints returns the endpoints supported by vLLM-omni (includes audio)
func (b *VLLMOmniBackend) SupportedEndpoints() map[string]bool {
	return endpointsWithAudio()
}
