package backend

import (
	"errors"
	"testing"
)

// stubPlugin is a minimal Plugin used by registry/resolver tests. It avoids
// importing plugintest (which lives in a subpackage and would create a test
// dependency cycle here).
type stubPlugin struct {
	id              ID
	displayName     string
	supports        func(string) bool
	endpoints       map[string]bool
	servedModelName func(ModelRef) string
}

func (s *stubPlugin) ID() ID                                                    { return s.id }
func (s *stubPlugin) DisplayName() string                                       { return s.displayName }
func (s *stubPlugin) Discover(*Config) (*Info, error)                           { return &Info{ID: s.id, Available: true}, nil }
func (s *stubPlugin) BuildStartConfig(*Info, *LoadRequest) (*StartConfig, error){ return &StartConfig{PluginID: s.id}, nil }
func (s *stubPlugin) IsLoadComplete(string) bool                                { return false }
func (s *stubPlugin) CheckHealth(int) (*HealthResult, error)                    { return &HealthResult{Healthy: true}, nil }
func (s *stubPlugin) SupportsModel(p string) bool {
	if s.supports == nil {
		return true
	}
	return s.supports(p)
}
func (s *stubPlugin) SupportedEndpoints() map[string]bool {
	if s.endpoints == nil {
		return map[string]bool{}
	}
	return s.endpoints
}
func (s *stubPlugin) ServedModelName(m ModelRef) string {
	if s.servedModelName == nil {
		return m.Name
	}
	return s.servedModelName(m)
}
func (s *stubPlugin) ConfigSchema() ConfigSchema { return ConfigSchema{PluginID: s.id} }
func (s *stubPlugin) ParamSchema() ParamSchema   { return ParamSchema{PluginID: s.id} }
func (s *stubPlugin) DecodeParams(RawParams) (Params, error) {
	return nil, nil
}
func (s *stubPlugin) ValidateParams(RawParams) ValidationResult {
	return ValidationResult{Valid: true}
}

func newStub(id ID) *stubPlugin {
	return &stubPlugin{id: id, displayName: string(id)}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub("foo"))
	p, ok := r.Get("foo")
	if !ok || p == nil {
		t.Fatalf("Get after Register failed")
	}
	if p.ID() != "foo" {
		t.Errorf("Get returned wrong plugin: %s", p.ID())
	}
}

func TestRegistry_RegisterNilNoOp(t *testing.T) {
	r := NewRegistry()
	r.Register(nil)
	if got := r.List(); len(got) != 0 {
		t.Errorf("nil Register added an entry: %v", got)
	}
}

func TestRegistry_MustRegisterPanicsOnDuplicateDifferentInstance(t *testing.T) {
	r := NewRegistry()
	r.MustRegister(newStub("dup"))
	defer func() {
		if recover() == nil {
			t.Errorf("MustRegister with different instance under same ID did not panic")
		}
	}()
	r.MustRegister(newStub("dup")) // different instance, same ID
}

func TestRegistry_MustRegisterIdempotentSameInstance(t *testing.T) {
	r := NewRegistry()
	p := newStub("ok")
	r.MustRegister(p)
	r.MustRegister(p) // same instance, must not panic
}

func TestRegistry_MustRegisterNilPanics(t *testing.T) {
	r := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Errorf("MustRegister(nil) did not panic")
		}
	}()
	r.MustRegister(nil)
}

func TestRegistry_ListSortedByID(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub("z"))
	r.Register(newStub("a"))
	r.Register(newStub("m"))
	got := r.List()
	if len(got) != 3 {
		t.Fatalf("List returned %d, want 3", len(got))
	}
	if got[0].ID() != "a" || got[1].ID() != "m" || got[2].ID() != "z" {
		t.Errorf("List not sorted: %s, %s, %s", got[0].ID(), got[1].ID(), got[2].ID())
	}
}

func TestRegistry_Configure(t *testing.T) {
	r := NewRegistry()
	r.Configure("foo", &Config{ID: "foo"})
	cfg, ok := r.GetConfig("foo")
	if !ok || cfg == nil {
		t.Fatalf("GetConfig after Configure failed")
	}
	r.Configure("foo", nil)
	if _, ok := r.GetConfig("foo"); ok {
		t.Errorf("Configure(nil) did not delete config")
	}
}

func TestRegistry_DiscoverAll_OnlyConfigured(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub("withcfg"))
	r.Register(newStub("nocfg"))
	r.Configure("withcfg", &Config{ID: "withcfg"})

	out := r.DiscoverAll()
	if _, ok := out["withcfg"]; !ok {
		t.Errorf("DiscoverAll missing configured plugin")
	}
	if _, ok := out["nocfg"]; ok {
		t.Errorf("DiscoverAll included unconfigured plugin")
	}
}

func TestRegistry_Reset(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub("foo"))
	r.Configure("foo", &Config{ID: "foo"})
	r.Reset()
	if _, ok := r.Get("foo"); ok {
		t.Errorf("Reset did not clear plugins")
	}
	if _, ok := r.GetConfig("foo"); ok {
		t.Errorf("Reset did not clear configs")
	}
}

// -----------------------------------------------------------------------------
// Resolve scenarios
// -----------------------------------------------------------------------------

func TestResolve_ExplicitID_Wins(t *testing.T) {
	r := NewRegistry()
	llama := newStub(IDLlamaCpp)
	vllm := newStub(IDVLLM)
	r.Register(llama)
	r.Register(vllm)
	r.Configure(IDLlamaCpp, &Config{ID: IDLlamaCpp})
	r.Configure(IDVLLM, &Config{ID: IDVLLM})

	p, _, err := r.Resolve("/models/foo.gguf", IDVLLM)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if p.ID() != IDVLLM {
		t.Errorf("explicit ID ignored, got %s", p.ID())
	}
}

func TestResolve_CapabilityHint_PrefersVLLMOmni(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub(IDLlamaCpp))
	omni := &stubPlugin{
		id: IDVLLMOmni,
		supports: func(p string) bool { return true },
	}
	r.Register(omni)
	r.Configure(IDLlamaCpp, &Config{ID: IDLlamaCpp})
	r.Configure(IDVLLMOmni, &Config{ID: IDVLLMOmni})

	p, _, err := r.Resolve("/models/whatever.gguf", "", &CapabilityHint{TTS: true})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if p.ID() != IDVLLMOmni {
		t.Errorf("capability hint did not pick vllmomni, got %s", p.ID())
	}
}

func TestResolve_CapabilityHint_FallsThroughIfUnavailable(t *testing.T) {
	r := NewRegistry()
	llama := newStub(IDLlamaCpp)
	r.Register(llama)
	r.Configure(IDLlamaCpp, &Config{ID: IDLlamaCpp})
	// vllmomni not registered

	p, _, err := r.Resolve("/models/foo.gguf", "", &CapabilityHint{TTS: true})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if p.ID() != IDLlamaCpp {
		t.Errorf("capability hint should have fallen through to GGUF default, got %s", p.ID())
	}
}

func TestResolve_FormatAutoDetect_PicksConfiguredVLLMForSafetensors(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub(IDLlamaCpp)) // supports anything by default; would otherwise win at end
	vllm := &stubPlugin{
		id:       IDVLLM,
		supports: func(p string) bool { return IsSafeTensorsModel(p) },
	}
	r.Register(vllm)
	r.Configure(IDLlamaCpp, &Config{ID: IDLlamaCpp})
	r.Configure(IDVLLM, &Config{ID: IDVLLM})

	p, _, err := r.Resolve("/models/foo.safetensors", "")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if p.ID() != IDVLLM {
		t.Errorf("auto-detect should pick vllm, got %s", p.ID())
	}
}

func TestResolve_DefaultGGUF_FallsBackToLlamaCpp(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub(IDLlamaCpp))
	// llama.cpp does not need a Config to be picked by the default rule.

	p, _, err := r.Resolve("/models/foo.gguf", "")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if p.ID() != IDLlamaCpp {
		t.Errorf("GGUF default did not pick llamacpp, got %s", p.ID())
	}
}

func TestResolve_NoSuitableBackend_NonGGUFNoConfig(t *testing.T) {
	r := NewRegistry()
	// Register all three but configure none. Non-GGUF model has nowhere to go.
	r.Register(newStub(IDLlamaCpp))
	r.Register(newStub(IDVLLM))
	r.Register(newStub(IDVLLMOmni))

	_, _, err := r.Resolve("/models/foo.safetensors", "")
	if err == nil {
		t.Fatalf("expected ErrNoSuitableBackend, got nil")
	}
	if !errors.Is(err, ErrNoSuitableBackend) {
		t.Errorf("expected ErrNoSuitableBackend, got %v", err)
	}
}

func TestResolve_ExplicitUnregistered_FallsThrough(t *testing.T) {
	// ExplicitIDRule deliberately fails-soft so an unknown ID does not short
	// the chain: subsequent rules can still pick a default. The legacy
	// resolver had the same behaviour.
	r := NewRegistry()
	r.Register(newStub(IDLlamaCpp))

	p, _, err := r.Resolve("/models/foo.gguf", "missing")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if p.ID() != IDLlamaCpp {
		t.Errorf("explicit unknown ID did not fall through to default, got %s", p.ID())
	}
}

func TestResolve_HintMerging(t *testing.T) {
	merged := mergeHints(&CapabilityHint{TTS: true}, &CapabilityHint{ASR: true}, nil)
	if merged == nil {
		t.Fatalf("mergeHints returned nil")
	}
	if !merged.TTS || !merged.ASR {
		t.Errorf("mergeHints lost flags: %+v", merged)
	}
}

func TestRegistry_RegisterRule_Append(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub(IDLlamaCpp))
	called := false
	r.RegisterRule(ruleFunc(func(ctx ResolveCtx) (Plugin, *Config, bool) {
		called = true
		return nil, nil, false
	}))
	_, _, _ = r.Resolve("/models/foo.gguf", "") // GGUF default fires before our appended rule
	if called {
		t.Errorf("appended rule should not fire when an earlier rule already returned a verdict")
	}
}

func TestRegistry_PrependRule_RunsFirst(t *testing.T) {
	r := NewRegistry()
	r.Register(newStub(IDLlamaCpp))
	stub := newStub("forced")
	r.Register(stub)
	r.PrependRule(ruleFunc(func(ctx ResolveCtx) (Plugin, *Config, bool) {
		return stub, &Config{ID: "forced"}, true
	}))
	p, _, err := r.Resolve("/models/foo.gguf", "")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if p.ID() != "forced" {
		t.Errorf("prepended rule did not run first, got %s", p.ID())
	}
}

// ruleFunc adapts a function to ResolutionRule for test purposes.
type ruleFunc func(ctx ResolveCtx) (Plugin, *Config, bool)

func (f ruleFunc) Apply(ctx ResolveCtx) (Plugin, *Config, bool) { return f(ctx) }

// LoadRequest.Validate — light coverage for completeness.
func TestLoadRequest_Validate(t *testing.T) {
	cases := []struct {
		name    string
		req     *LoadRequest
		wantErr bool
	}{
		{"nil", nil, true},
		{"empty path", &LoadRequest{Port: 8080}, true},
		{"zero port", &LoadRequest{ModelPath: "/models/x"}, true},
		{"ok", &LoadRequest{ModelPath: "/models/x", Port: 8080}, false},
	}
	for _, tc := range cases {
		err := tc.req.Validate()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: Validate err=%v wantErr=%v", tc.name, err, tc.wantErr)
		}
	}
}
