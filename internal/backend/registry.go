package backend

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is the central directory of installed plugins, their static
// configuration, and the resolution rules that pick a plugin for a given
// model load request.
//
// The Default() registry is populated by plugin subpackages at process start
// via init() Register / MustRegister calls; tests should construct an
// isolated Registry via NewRegistry() (or plugintest.NewIsolatedRegistry).
type Registry struct {
	mu      sync.RWMutex
	plugins map[ID]Plugin
	configs map[ID]*Config
	rules   []ResolutionRule
}

// NewRegistry creates an empty Registry pre-loaded with the standard
// resolution rules:
//
//  1. ExplicitIDRule           — explicit plugin ID overrides everything
//  2. CapabilityHintRule       — TTS/ASR/ImageGen prefers vllmomni
//  3. FormatAutoDetectRule     — first plugin where SupportsModel(modelPath) is true
//  4. DefaultForGGUFRule       — GGUF falls back to llamacpp
//  5. ErrorRule                — non-GGUF without a configured backend errors
//
// Plugins may install additional rules via Registry.RegisterRule.
func NewRegistry() *Registry {
	r := &Registry{
		plugins: make(map[ID]Plugin),
		configs: make(map[ID]*Config),
	}
	r.rules = []ResolutionRule{
		ExplicitIDRule{},
		CapabilityHintRule{PreferredID: IDVLLMOmni},
		FormatAutoDetectRule{},
		DefaultForGGUFRule{LlamaCppID: IDLlamaCpp},
		ErrorRule{},
	}
	return r
}

// Default returns the process-global Registry. Use this from production code
// paths; tests should always use NewRegistry to avoid leaking init()-time
// state across cases.
func Default() *Registry { return defaultRegistry }

var defaultRegistry = NewRegistry()

// Canonical IDs for the three built-in plugins. Defined here (not in the
// plugin subpackages) so the resolver rules above can reference them without
// importing those subpackages.
const (
	IDLlamaCpp ID = "llamacpp"
	IDVLLM     ID = "vllm"
	IDVLLMOmni ID = "vllmomni"
)

// Register adds (or replaces) a plugin in the registry. Last-writer-wins for
// the same ID; use MustRegister to surface duplicates as panics during
// init().
func (r *Registry) Register(p Plugin) {
	if p == nil {
		return
	}
	r.mu.Lock()
	r.plugins[p.ID()] = p
	r.mu.Unlock()
}

// MustRegister adds a plugin and panics if a different plugin instance is
// already registered under that ID. Intended for plugin subpackages' init()
// calls so duplicate IDs surface immediately at process start.
func (r *Registry) MustRegister(p Plugin) {
	if p == nil {
		panic("backend.MustRegister: nil plugin")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.plugins[p.ID()]; ok && existing != p {
		panic(fmt.Sprintf("backend.MustRegister: duplicate plugin ID %q", p.ID()))
	}
	r.plugins[p.ID()] = p
}

// MustRegister registers a plugin into the default registry. Used by plugin
// subpackages from init():
//
//	func init() { backend.MustRegister(New()) }
func MustRegister(p Plugin) { defaultRegistry.MustRegister(p) }

// Get returns the plugin registered for the given ID, or false.
func (r *Registry) Get(id ID) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[id]
	return p, ok
}

// List returns all registered plugins sorted by ID.
func (r *Registry) List() []Plugin {
	r.mu.RLock()
	out := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Configure stores the given Config under the plugin's ID. A nil cfg removes
// any prior configuration.
func (r *Registry) Configure(id ID, cfg *Config) {
	r.mu.Lock()
	if cfg == nil {
		delete(r.configs, id)
	} else {
		r.configs[id] = cfg
	}
	r.mu.Unlock()
}

// GetConfig returns the Config registered for the given plugin ID, or false.
func (r *Registry) GetConfig(id ID) (*Config, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cfg, ok := r.configs[id]
	return cfg, ok
}

// DiscoverAll runs Discover for every registered plugin that has a Config
// attached, and returns a map of results. Plugins without a Config are
// skipped (they are considered "not configured" by the operator).
func (r *Registry) DiscoverAll() map[ID]*Info {
	r.mu.RLock()
	pairs := make([]struct {
		id  ID
		p   Plugin
		cfg *Config
	}, 0, len(r.plugins))
	for id, p := range r.plugins {
		cfg := r.configs[id]
		pairs = append(pairs, struct {
			id  ID
			p   Plugin
			cfg *Config
		}{id, p, cfg})
	}
	r.mu.RUnlock()

	result := make(map[ID]*Info, len(pairs))
	for _, pair := range pairs {
		if pair.cfg == nil {
			continue
		}
		info, err := pair.p.Discover(pair.cfg)
		if err != nil {
			// Discovery error: surface as not-available with the error
			// message stamped into Version. Callers wanting the typed
			// error can call Discover directly.
			result[pair.id] = &Info{
				ID:          pair.id,
				DisplayName: pair.p.DisplayName(),
				Version:     "discover error: " + err.Error(),
				Available:   false,
			}
			continue
		}
		if info != nil {
			info.ID = pair.id
			if info.DisplayName == "" {
				info.DisplayName = pair.p.DisplayName()
			}
			result[pair.id] = info
		}
	}
	return result
}

// Resolve picks the right plugin for a model. The arguments are:
//
//   - modelPath:   filesystem path or HF directory of the model
//   - explicit:    optional plugin ID forced by the caller (empty → auto)
//   - hints:       optional capability hints (TTS / ASR / image gen)
//
// Resolution walks the registered rules in order. The first rule that
// returns ok=true wins; otherwise ErrNoSuitableBackend is returned.
//
// Resolve never returns a plugin without a configured Config except via
// ExplicitIDRule (where the caller has already taken responsibility for the
// backend not being usable).
func (r *Registry) Resolve(modelPath string, explicit ID, hints ...*CapabilityHint) (Plugin, *Config, error) {
	r.mu.RLock()
	plugins := make(map[ID]Plugin, len(r.plugins))
	for id, p := range r.plugins {
		plugins[id] = p
	}
	configs := make(map[ID]*Config, len(r.configs))
	for id, c := range r.configs {
		configs[id] = c
	}
	rules := append([]ResolutionRule(nil), r.rules...)
	r.mu.RUnlock()

	hint := mergeHints(hints...)
	ctx := ResolveCtx{
		ModelPath: modelPath,
		Explicit:  explicit,
		Hint:      hint,
		Plugins:   plugins,
		Configs:   configs,
	}
	for _, rule := range rules {
		if p, cfg, ok := rule.Apply(ctx); ok {
			return p, cfg, nil
		}
	}
	return nil, nil, NoSuitableBackendError(modelPath)
}

// RegisterRule appends a resolution rule. New rules are evaluated *after*
// the built-in ones; if you need to run before the built-ins, prepend with
// PrependRule.
func (r *Registry) RegisterRule(rule ResolutionRule) {
	r.mu.Lock()
	r.rules = append(r.rules, rule)
	r.mu.Unlock()
}

// PrependRule inserts a rule at the front of the resolution chain.
func (r *Registry) PrependRule(rule ResolutionRule) {
	r.mu.Lock()
	r.rules = append([]ResolutionRule{rule}, r.rules...)
	r.mu.Unlock()
}

// Reset clears registered plugins, configs, and resolution rules, then
// reinstalls the standard rules. Intended for tests; do not call in
// production code paths.
func (r *Registry) Reset() {
	r.mu.Lock()
	r.plugins = make(map[ID]Plugin)
	r.configs = make(map[ID]*Config)
	r.rules = []ResolutionRule{
		ExplicitIDRule{},
		CapabilityHintRule{PreferredID: IDVLLMOmni},
		FormatAutoDetectRule{},
		DefaultForGGUFRule{LlamaCppID: IDLlamaCpp},
		ErrorRule{},
	}
	r.mu.Unlock()
}

// mergeHints folds zero-or-more CapabilityHint pointers into a single hint.
// Multiple hints OR their flags together; nil hints are ignored.
func mergeHints(hints ...*CapabilityHint) *CapabilityHint {
	var out *CapabilityHint
	for _, h := range hints {
		if h == nil {
			continue
		}
		if out == nil {
			out = &CapabilityHint{}
		}
		out.TTS = out.TTS || h.TTS
		out.ASR = out.ASR || h.ASR
		out.ImageGeneration = out.ImageGeneration || h.ImageGeneration
	}
	return out
}
