// Package plugintest provides helpers for writing tests that exercise the
// backend Plugin contract without leaking init()-time global state.
package plugintest

import (
	"github.com/simonxluo/Shepherd/internal/backend"
)

// NewIsolatedRegistry returns a brand-new Registry with the given plugins
// pre-registered. The returned registry has the standard resolution rules
// installed but is fully detached from backend.Default(); use this in tests
// to avoid interactions with init()-time registration of real plugins.
func NewIsolatedRegistry(plugins ...backend.Plugin) *backend.Registry {
	r := backend.NewRegistry()
	for _, p := range plugins {
		if p == nil {
			continue
		}
		r.MustRegister(p)
	}
	return r
}

// FakePlugin is a configurable stub implementing backend.Plugin. Every
// callable returns a value supplied via the corresponding field; absent
// values yield reasonable defaults.
type FakePlugin struct {
	IDValue               ID
	DisplayNameValue      string
	DiscoverFn            func(cfg *Config) (*Info, error)
	BuildStartConfigFn    func(info *Info, req *LoadRequest) (*StartConfig, error)
	IsLoadCompleteFn      func(line string) bool
	CheckHealthFn         func(port int) (*HealthResult, error)
	SupportsModelFn       func(path string) bool
	SupportedEndpointsMap map[string]bool
	ServedModelNameFn     func(m ModelRef) string
	ConfigSchemaValue     ConfigSchema
	ParamSchemaValue      ParamSchema
	DecodeParamsFn        func(raw RawParams) (Params, error)
	ValidateParamsFn      func(raw RawParams) ValidationResult
}

// Aliases so the test file does not have to repeat backend.* qualifiers.
type (
	ID               = backend.ID
	Plugin           = backend.Plugin
	Info             = backend.Info
	Config           = backend.Config
	LoadRequest      = backend.LoadRequest
	StartConfig      = backend.StartConfig
	HealthResult     = backend.HealthResult
	ModelRef         = backend.ModelRef
	ConfigSchema     = backend.ConfigSchema
	ParamSchema      = backend.ParamSchema
	Params           = backend.Params
	RawParams        = backend.RawParams
	ValidationResult = backend.ValidationResult
)

// Compile-time assertion that *FakePlugin satisfies backend.Plugin.
var _ backend.Plugin = (*FakePlugin)(nil)

func (f *FakePlugin) ID() backend.ID {
	if f.IDValue == "" {
		return "fake"
	}
	return f.IDValue
}

func (f *FakePlugin) DisplayName() string {
	if f.DisplayNameValue == "" {
		return "Fake Plugin"
	}
	return f.DisplayNameValue
}

func (f *FakePlugin) Discover(cfg *backend.Config) (*backend.Info, error) {
	if f.DiscoverFn != nil {
		return f.DiscoverFn(cfg)
	}
	return &backend.Info{ID: f.ID(), DisplayName: f.DisplayName(), Available: true}, nil
}

func (f *FakePlugin) BuildStartConfig(info *backend.Info, req *backend.LoadRequest) (*backend.StartConfig, error) {
	if f.BuildStartConfigFn != nil {
		return f.BuildStartConfigFn(info, req)
	}
	return &backend.StartConfig{PluginID: f.ID()}, nil
}

func (f *FakePlugin) IsLoadComplete(line string) bool {
	if f.IsLoadCompleteFn != nil {
		return f.IsLoadCompleteFn(line)
	}
	return false
}

func (f *FakePlugin) CheckHealth(port int) (*backend.HealthResult, error) {
	if f.CheckHealthFn != nil {
		return f.CheckHealthFn(port)
	}
	return &backend.HealthResult{Healthy: true}, nil
}

func (f *FakePlugin) SupportsModel(path string) bool {
	if f.SupportsModelFn != nil {
		return f.SupportsModelFn(path)
	}
	return true
}

func (f *FakePlugin) SupportedEndpoints() map[string]bool {
	if f.SupportedEndpointsMap != nil {
		return f.SupportedEndpointsMap
	}
	return map[string]bool{}
}

func (f *FakePlugin) ServedModelName(m backend.ModelRef) string {
	if f.ServedModelNameFn != nil {
		return f.ServedModelNameFn(m)
	}
	return m.Name
}

func (f *FakePlugin) ConfigSchema() backend.ConfigSchema { return f.ConfigSchemaValue }

func (f *FakePlugin) ParamSchema() backend.ParamSchema { return f.ParamSchemaValue }

func (f *FakePlugin) DecodeParams(raw backend.RawParams) (backend.Params, error) {
	if f.DecodeParamsFn != nil {
		return f.DecodeParamsFn(raw)
	}
	return nil, nil
}

func (f *FakePlugin) ValidateParams(raw backend.RawParams) backend.ValidationResult {
	if f.ValidateParamsFn != nil {
		return f.ValidateParamsFn(raw)
	}
	return backend.ValidationResult{Valid: true}
}
