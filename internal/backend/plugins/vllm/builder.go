package vllm

import (
	"fmt"
	"strings"

	"github.com/simonxluo/Shepherd/internal/backend"
)

// BuildPrefix builds the command prefix (binary/conda + serve + modelPath).
// Exported for reuse by vllmomni.
func BuildPrefix(info *backend.Info, modelPath, binaryName string) []string {
	if info.BinPath != "" {
		return []string{info.BinPath, "serve", modelPath}
	}
	condaPath := info.CondaPath
	if condaPath == "" {
		condaPath = "conda"
	}
	return []string{condaPath, "run", "--no-banner", "-n", info.CondaEnv, binaryName, "serve", modelPath}
}

// AppendArgs appends common vLLM CLI flags (port, host, dtype, parallelism, etc.)
// to an existing arg slice. Exported for reuse by vllmomni.
func AppendArgs(args []string, req *backend.LoadRequest, p *Params) ([]string, error) {
	args = append(args, "--port", fmt.Sprintf("%d", req.Port))
	args = append(args, "--host", req.BindHost)

	if p.MaxModelLen > 0 {
		args = append(args, "--max-model-len", fmt.Sprintf("%d", p.MaxModelLen))
	}
	if p.DataType != "" {
		args = append(args, "--dtype", p.DataType)
	}
	if p.GPUMemoryUtilization > 1.0 {
		return nil, fmt.Errorf("gpu_memory_utilization must be between 0 and 1.0, got %.2f", p.GPUMemoryUtilization)
	}
	if p.GPUMemoryUtilization > 0 {
		args = append(args, "--gpu-memory-utilization", fmt.Sprintf("%.2f", p.GPUMemoryUtilization))
	}
	if p.TensorParallelSize > 0 {
		args = append(args, "--tensor-parallel-size", fmt.Sprintf("%d", p.TensorParallelSize))
	} else if len(req.Devices) > 1 {
		args = append(args, "--tensor-parallel-size", fmt.Sprintf("%d", len(req.Devices)))
	}
	if p.PipelineParallelSize > 0 {
		args = append(args, "--pipeline-parallel-size", fmt.Sprintf("%d", p.PipelineParallelSize))
	}
	if p.TrustRemoteCode {
		args = append(args, "--trust-remote-code")
	}
	if p.ServedModelName != "" {
		args = append(args, "--served-model-name", p.ServedModelName)
	}
	if p.Quantization != "" {
		args = append(args, "--quantization", p.Quantization)
	}
	if p.MaxNumSeqs > 0 {
		args = append(args, "--max-num-seqs", fmt.Sprintf("%d", p.MaxNumSeqs))
	}
	if p.MaxNumBatchedTokens > 0 {
		args = append(args, "--max-num-batched-tokens", fmt.Sprintf("%d", p.MaxNumBatchedTokens))
	}
	if p.EnablePrefixCaching {
		args = append(args, "--enable-prefix-caching")
	}
	if p.EnableChunkedPrefill {
		args = append(args, "--enable-chunked-prefill")
	}
	if p.DisableLogRequests {
		args = append(args, "--disable-log-requests")
	}
	if p.EnforceEager {
		args = append(args, "--enforce-eager")
	}
	return args, nil
}

// BuildStartResult constructs the backend.StartConfig from assembled args.
// Exported for reuse by vllmomni.
func BuildStartResult(args []string, info *backend.Info, p *Params, pluginID backend.ID) (*backend.StartConfig, error) {
	spec := backend.NewCommandSpec(args[0], args[1:], nil, "")
	if info.GlobalExtraArgs != "" {
		spec = spec.AppendRaw(info.GlobalExtraArgs)
	}
	if p.ExtraArgs != "" {
		spec = spec.AppendRaw(strings.TrimSpace(p.ExtraArgs))
	}

	skipLD := info.CondaEnv != "" && info.BinPath == ""
	return &backend.StartConfig{
		CommandSpec:       &spec,
		BinPath:           info.BinPath,
		PluginID:          pluginID,
		SkipLDLibraryPath: skipLD,
		CondaPath:         info.CondaPath,
	}, nil
}

// IsLoadCompleteUvicorn returns true when the output line indicates Uvicorn
// has started. Exported for reuse by vllmomni.
func IsLoadCompleteUvicorn(line string) bool {
	return strings.Contains(line, "Uvicorn running on")
}

// CheckHTTPHealth probes a /health endpoint on localhost:port. Returns
// healthy=true on HTTP 200. Exported for reuse by vllmomni.
func CheckHTTPHealth(port int) (*backend.HealthResult, error) {
	return checkHealth(port)
}

// EndpointsWithoutAudio returns the standard API endpoint set with audio
// endpoints disabled. Exported for reuse by vllmomni.
func EndpointsWithoutAudio() map[string]bool {
	return endpointsWithoutAudio()
}
