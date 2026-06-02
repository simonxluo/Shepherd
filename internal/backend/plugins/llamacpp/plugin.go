package llamacpp

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
)

// Plugin implements backend.Plugin for llama.cpp (llama-server).
type Plugin struct{}

// New creates a new llama.cpp plugin instance.
func New() *Plugin { return &Plugin{} }

var _ backend.Plugin = (*Plugin)(nil)

func (*Plugin) ID() backend.ID      { return backend.IDLlamaCpp }
func (*Plugin) DisplayName() string { return "llama.cpp" }

func (*Plugin) Discover(cfg *backend.Config) (*backend.Info, error) {
	info := &backend.Info{
		ID:          backend.IDLlamaCpp,
		DisplayName: "llama.cpp",
	}

	// Resolve from BinPaths first, then common fallback locations.
	paths := cfg.BinPaths
	if len(paths) == 0 {
		paths = []string{"/usr/local/bin", "/usr/bin", "./llama.cpp"}
	}
	for _, p := range paths {
		if p == "" {
			continue
		}
		probe, err := ProbeInstallation(p)
		if err != nil {
			return nil, err
		}
		if probe.Available {
			info.BinPath = p
			info.Version = probe.Version
			info.Available = true
			return info, nil
		}
	}
	return info, nil
}

func (*Plugin) BuildStartConfig(info *backend.Info, req *backend.LoadRequest) (*backend.StartConfig, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if info.BinPath == "" {
		return nil, fmt.Errorf("llama.cpp binary path not set")
	}

	p, ok := req.Params.(*Params)
	if !ok && req.Params != nil {
		return nil, backend.ErrParamTypeMismatch
	}
	if p == nil {
		p = &Params{}
	}

	serverBin, err := findServerBin(info.BinPath)
	if err != nil {
		return nil, err
	}

	cmdSpec, err := buildArgs(p, req, serverBin)
	if err != nil {
		return nil, err
	}
	if info.GlobalExtraArgs != "" {
		cmdSpec = cmdSpec.AppendRaw(info.GlobalExtraArgs)
	}

	return &backend.StartConfig{
		CommandSpec: &cmdSpec,
		BinPath:     info.BinPath,
		PluginID:    backend.IDLlamaCpp,
	}, nil
}

func (*Plugin) IsLoadComplete(line string) bool {
	return strings.Contains(line, "all slots are idle")
}

func (*Plugin) CheckHealth(port int) (*backend.HealthResult, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return &backend.HealthResult{Healthy: false}, err
	}
	defer utils.CloseQuietly(resp.Body)

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &backend.HealthResult{Healthy: false, Body: string(body)}, nil
	}
	healthy := strings.Contains(string(body), `"status":"ok"`)
	return &backend.HealthResult{Healthy: healthy, Body: string(body)}, nil
}

func (*Plugin) SupportsModel(modelPath string) bool {
	return backend.IsGGUFModel(modelPath)
}

func (*Plugin) SupportedEndpoints() map[string]bool {
	ep := map[string]bool{
		"/v1/chat/completions": true,
		"/v1/completions":      true,
		"/v1/models":           true,
		"/v1/embeddings":       true,
	}
	ep["/v1/audio/speech"] = false
	ep["/v1/audio/voices"] = false
	ep["/v1/audio/transcriptions"] = false
	ep["/v1/audio/translations"] = false
	ep["/v1/audio/music"] = false
	return ep
}

// ServedModelName returns the model's friendly name, which is what
// llama-server expects in the request body's "model" field.
func (*Plugin) ServedModelName(m backend.ModelRef) string {
	return m.Name
}

func (*Plugin) ConfigSchema() backend.ConfigSchema {
	return backend.ConfigSchema{PluginID: backend.IDLlamaCpp}
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
