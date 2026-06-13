package vllm

import (
	"fmt"

	"github.com/simonxluo/Shepherd/internal/backend"
)

// Plugin implements backend.Plugin for vLLM.
type Plugin struct{}

// New creates a new vLLM plugin instance.
func New() *Plugin { return &Plugin{} }

var _ backend.Plugin = (*Plugin)(nil)

func (*Plugin) ID() backend.ID      { return backend.IDVLLM }
func (*Plugin) DisplayName() string { return "vLLM" }

func (*Plugin) Discover(cfg *backend.Config) (*backend.Info, error) {
	return DiscoverVariant(cfg, backend.IDVLLM, "vLLM", "vllm")
}

func (*Plugin) BuildStartConfig(info *backend.Info, req *backend.LoadRequest) (*backend.StartConfig, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if info.CondaEnv == "" && info.BinPath == "" {
		return nil, fmt.Errorf("vLLM requires a conda environment or binary path")
	}
	p, ok := req.Params.(*Params)
	if !ok && req.Params != nil {
		return nil, backend.ErrParamTypeMismatch
	}
	if p == nil {
		p = &Params{}
	}

	args := BuildPrefix(info, req.ModelPath, "vllm")
	args, err := AppendArgs(args, req, p)
	if err != nil {
		return nil, err
	}
	return BuildStartResult(args, info, p, backend.IDVLLM)
}

func (*Plugin) IsLoadComplete(line string) bool {
	return IsLoadCompleteUvicorn(line)
}

func (*Plugin) CheckHealth(port int) (*backend.HealthResult, error) {
	return CheckHTTPHealth(port)
}

func (*Plugin) SupportsModel(modelPath string) bool {
	return backend.IsSafeTensorsModel(modelPath)
}

func (*Plugin) SupportedEndpoints() map[string]bool {
	return EndpointsWithoutAudio()
}

// ServedModelName returns the on-disk model path, which is what vLLM
// expects in the request body's "model" field.
func (*Plugin) ServedModelName(m backend.ModelRef) string {
	return m.Path
}

func (*Plugin) ConfigSchema() backend.ConfigSchema {
	return backend.ConfigSchema{PluginID: backend.IDVLLM}
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
