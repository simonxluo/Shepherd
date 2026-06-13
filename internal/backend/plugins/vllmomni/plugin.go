package vllmomni

import (
	"fmt"

	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/backend/plugins/vllm"
)

// Plugin implements backend.Plugin for vLLM-Omni.
type Plugin struct{}

// New creates a new vLLM-Omni plugin instance.
func New() *Plugin { return &Plugin{} }

var _ backend.Plugin = (*Plugin)(nil)

func (*Plugin) ID() backend.ID      { return backend.IDVLLMOmni }
func (*Plugin) DisplayName() string { return "vLLM-Omni" }

func (*Plugin) Discover(cfg *backend.Config) (*backend.Info, error) {
	return vllm.DiscoverVariant(cfg, backend.IDVLLMOmni, "vLLM-Omni", "vllm-omni")
}

func (*Plugin) BuildStartConfig(info *backend.Info, req *backend.LoadRequest) (*backend.StartConfig, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if info.CondaEnv == "" && info.BinPath == "" {
		return nil, fmt.Errorf("vLLM-Omni requires a conda environment or binary path")
	}
	p, ok := req.Params.(*Params)
	if !ok && req.Params != nil {
		return nil, backend.ErrParamTypeMismatch
	}
	if p == nil {
		p = &Params{}
	}

	args := vllm.BuildPrefix(info, req.ModelPath, "vllm-omni")
	var err error
	args, err = vllm.AppendArgs(args, req, &p.Base)
	if err != nil {
		return nil, err
	}

	// --omni is always set (core purpose of this backend).
	args = append(args, "--omni")

	if p.VideoPruningRate > 0 {
		args = append(args, "--video-pruning-rate", fmt.Sprintf("%.2f", p.VideoPruningRate))
	}
	if p.MMTensorIPC {
		args = append(args, "--mm-tensor-ipc")
	}

	return vllm.BuildStartResult(args, info, &p.Base, backend.IDVLLMOmni)
}

func (*Plugin) IsLoadComplete(line string) bool {
	return vllm.IsLoadCompleteUvicorn(line)
}

func (*Plugin) CheckHealth(port int) (*backend.HealthResult, error) {
	return vllm.CheckHTTPHealth(port)
}

// SupportsModel returns true for safetensors/HuggingFace directories and
// GGUF files (multimodal models may be in GGUF format).
func (*Plugin) SupportsModel(modelPath string) bool {
	return backend.IsSafeTensorsModel(modelPath) || backend.IsGGUFModel(modelPath)
}

func (*Plugin) SupportedEndpoints() map[string]bool {
	return vllm.EndpointsWithAudio()
}

// ServedModelName returns the on-disk model path (same as vLLM).
func (*Plugin) ServedModelName(m backend.ModelRef) string {
	return m.Path
}

func (*Plugin) ConfigSchema() backend.ConfigSchema {
	return backend.ConfigSchema{PluginID: backend.IDVLLMOmni}
}

func (*Plugin) ParamSchema() backend.ParamSchema {
	return paramSchema()
}

func (*Plugin) DecodeParams(raw backend.RawParams) (backend.Params, error) {
	return decodeParams(raw)
}

func (*Plugin) ValidateParams(raw backend.RawParams) backend.ValidationResult {
	return validateParams(raw)
}
