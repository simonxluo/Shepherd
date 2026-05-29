package backend

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// VLLMBackend implements Backend for vLLM
type VLLMBackend struct{}

// NewVLLMBackend creates a new vLLM backend instance
func NewVLLMBackend() *VLLMBackend {
	return &VLLMBackend{}
}

func (b *VLLMBackend) Type() BackendType { return BackendVLLM }

// Discover validates that vLLM is available in the configured conda environment
func (b *VLLMBackend) Discover(cfg *BackendConfig) (*BackendInfo, error) {
	return discoverVLLMVariant(cfg, BackendVLLM, "vLLM", "vllm")
}

// BuildStartConfig constructs the vllm serve command
func (b *VLLMBackend) BuildStartConfig(info *BackendInfo, req *LoadRequest) (*StartConfig, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if info.CondaEnv == "" && info.BinPath == "" {
		return nil, fmt.Errorf("vLLM requires a conda environment or binary path")
	}

	p := req.VLLMParams
	if p == nil {
		p = &VLLMLoadParams{}
	}

	// Build prefix args (binary/conda + serve + model)
	args := buildVLLMPrefix(info, req, "vllm")

	// Append common vLLM parameters
	args, err := appendVLLMArgs(args, req, p)
	if err != nil {
		return nil, err
	}

	cmd := quoteAndJoin(args)

	// Append extra args: global config ExtraArgs first, then model-level ExtraArgs (higher priority)
	if info.ExtraArgs != "" {
		cmd += " " + strings.TrimSpace(info.ExtraArgs)
	}
	if p.ExtraArgs != "" {
		cmd += " " + strings.TrimSpace(p.ExtraArgs)
	}

	// SkipLDLibraryPath: conda manages its own env, but direct binary mode may need LD_LIBRARY_PATH
	skipLD := info.CondaEnv != "" && info.BinPath == ""

	return &StartConfig{
		Command:           cmd,
		BinPath:           info.BinPath,
		BackendType:       BackendVLLM,
		SkipLDLibraryPath: skipLD,
		CondaPath:         info.CondaPath,
	}, nil
}

// buildVLLMPrefix builds the command prefix for vLLM/vLLM-omni.
// binaryName is "vllm" or "vllm-omni".
func buildVLLMPrefix(info *BackendInfo, req *LoadRequest, binaryName string) []string {
	var args []string

	if info.BinPath != "" {
		args = append(args, info.BinPath, "serve", req.ModelPath)
	} else {
		condaPath := info.CondaPath
		if condaPath == "" {
			condaPath = "conda"
		}
		args = append(args, condaPath, "run", "--no-banner", "-n", info.CondaEnv, binaryName, "serve", req.ModelPath)
	}

	return args
}

// appendVLLMArgs appends common vLLM parameters (port, host, dtype, parallelism, etc.) to args.
// This is shared between VLLMBackend and VLLMOmniBackend.
func appendVLLMArgs(args []string, req *LoadRequest, p *VLLMLoadParams) ([]string, error) {
	// Port & host
	args = append(args, "--port", fmt.Sprintf("%d", req.Port))
	bindHost := req.BindHost
	if bindHost == "" {
		bindHost = "0.0.0.0"
	}
	args = append(args, "--host", bindHost)

	// Context size
	if req.CtxSize > 0 {
		args = append(args, "--max-model-len", fmt.Sprintf("%d", req.CtxSize))
	} else if p.MaxModelLen > 0 {
		args = append(args, "--max-model-len", fmt.Sprintf("%d", p.MaxModelLen))
	}

	// Data type
	if p.DataType != "" {
		args = append(args, "--dtype", p.DataType)
	}

	// GPU memory utilization
	if p.GPUMemoryUtilization > 1.0 {
		return nil, fmt.Errorf("gpu_memory_utilization must be between 0 and 1.0, got %.2f", p.GPUMemoryUtilization)
	}
	if p.GPUMemoryUtilization > 0 {
		args = append(args, "--gpu-memory-utilization", fmt.Sprintf("%.2f", p.GPUMemoryUtilization))
	}

	// Tensor parallelism
	if p.TensorParallelSize > 0 {
		args = append(args, "--tensor-parallel-size", fmt.Sprintf("%d", p.TensorParallelSize))
	} else if len(req.Devices) > 1 {
		args = append(args, "--tensor-parallel-size", fmt.Sprintf("%d", len(req.Devices)))
	}

	// Pipeline parallelism
	if p.PipelineParallelSize > 0 {
		args = append(args, "--pipeline-parallel-size", fmt.Sprintf("%d", p.PipelineParallelSize))
	}

	// Trust remote code
	if p.TrustRemoteCode {
		args = append(args, "--trust-remote-code")
	}

	// Served model name
	if p.ServedModelName != "" {
		args = append(args, "--served-model-name", p.ServedModelName)
	}

	// Quantization
	if p.Quantization != "" {
		args = append(args, "--quantization", p.Quantization)
	}

	// Max sequences
	if p.MaxNumSeqs > 0 {
		args = append(args, "--max-num-seqs", fmt.Sprintf("%d", p.MaxNumSeqs))
	}

	// Max batched tokens
	if p.MaxNumBatchedTokens > 0 {
		args = append(args, "--max-num-batched-tokens", fmt.Sprintf("%d", p.MaxNumBatchedTokens))
	}

	// Prefix caching
	if p.EnablePrefixCaching {
		args = append(args, "--enable-prefix-caching")
	}

	// Chunked prefill
	if p.EnableChunkedPrefill {
		args = append(args, "--enable-chunked-prefill")
	}

	// Disable log requests
	if p.DisableLogRequests {
		args = append(args, "--disable-log-requests")
	}

	// Enforce eager execution
	if p.EnforceEager {
		args = append(args, "--enforce-eager")
	}

	// GPU layers: vLLM doesn't use -ngl, just log if set
	if req.GPULayers > 0 {
		logger.Debug("vLLM ignores gpuLayers setting - it manages GPU offloading automatically")
	}

	return args, nil
}

// IsLoadComplete detects vLLM load completion from stdout
func (b *VLLMBackend) IsLoadComplete(outputLine string) bool {
	return strings.Contains(outputLine, "Uvicorn running on")
}

// CheckHealth performs an HTTP health check against the vLLM server.
// Always uses localhost since health checks are issued from the local machine.
func (b *VLLMBackend) CheckHealth(port int) (*HealthResult, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return &HealthResult{Healthy: false}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return &HealthResult{Healthy: false, Body: string(body)}, nil
	}

	return &HealthResult{Healthy: true, Body: string(body)}, nil
}

// SupportsModel returns true for safetensors/HuggingFace directories
func (b *VLLMBackend) SupportsModel(modelPath string) bool {
	return IsSafeTensorsModel(modelPath)
}

// SupportedEndpoints returns the endpoints supported by vLLM
func (b *VLLMBackend) SupportedEndpoints() map[string]bool {
	return endpointsWithoutAudio()
}
