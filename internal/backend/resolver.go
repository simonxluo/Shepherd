package backend

// ResolveCtx is the read-only context handed to ResolutionRule.Apply.
// It exposes immutable snapshots of the registry state plus the caller's
// inputs. Rules must not mutate the maps; they are shared-by-reference for
// efficiency only.
type ResolveCtx struct {
	ModelPath string
	Explicit  ID
	Hint      *CapabilityHint
	Plugins   map[ID]Plugin
	Configs   map[ID]*Config
}

// ResolutionRule is one step in the chain that picks a plugin for a load.
// Each rule decides whether it has a verdict; if not it returns ok=false and
// the next rule runs.
type ResolutionRule interface {
	Apply(ctx ResolveCtx) (Plugin, *Config, bool)
}

// ExplicitIDRule honours an explicitly requested plugin ID, if it is
// registered. The caller takes responsibility for the plugin being usable;
// this rule does not consult Configs (an explicit request with no config
// returns the plugin and a nil Config so the caller can surface a more
// specific error downstream).
type ExplicitIDRule struct{}

func (ExplicitIDRule) Apply(ctx ResolveCtx) (Plugin, *Config, bool) {
	if ctx.Explicit == "" {
		return nil, nil, false
	}
	p, ok := ctx.Plugins[ctx.Explicit]
	if !ok {
		// Explicit but unregistered: still "decisive" — surface as no-config
		// hit so callers can return PluginNotFoundError above. Returning ok=false
		// here would let later rules pick something else, which is wrong.
		return nil, nil, false
	}
	cfg := ctx.Configs[ctx.Explicit]
	return p, cfg, true
}

// CapabilityHintRule biases selection toward a preferred plugin (typically
// vllmomni) when the model declares multimodal needs (TTS / ASR / image
// generation). Only fires when the preferred plugin is registered, has a
// Config, and SupportsModel returns true.
type CapabilityHintRule struct {
	PreferredID ID
}

func (r CapabilityHintRule) Apply(ctx ResolveCtx) (Plugin, *Config, bool) {
	if !ctx.Hint.NeedsMultimodal() {
		return nil, nil, false
	}
	p, ok := ctx.Plugins[r.PreferredID]
	if !ok {
		return nil, nil, false
	}
	if !p.SupportsModel(ctx.ModelPath) {
		return nil, nil, false
	}
	cfg := ctx.Configs[r.PreferredID]
	if cfg == nil {
		return nil, nil, false
	}
	return p, cfg, true
}

// FormatAutoDetectRule iterates registered plugins (in deterministic order
// by ID, modulo the special-case below) and picks the first whose
// SupportsModel returns true and which has a Config attached.
//
// llamacpp is intentionally evaluated last so the GGUF default falls through
// to DefaultForGGUFRule rather than getting picked here. Callers wanting
// llamacpp can either set the explicit ID or rely on the default rule.
type FormatAutoDetectRule struct{}

func (FormatAutoDetectRule) Apply(ctx ResolveCtx) (Plugin, *Config, bool) {
	// Order: every registered ID alphabetically except llamacpp, which goes
	// last. Deterministic so the result does not depend on map iteration.
	ids := make([]ID, 0, len(ctx.Plugins))
	var hasLlama bool
	for id := range ctx.Plugins {
		if id == IDLlamaCpp {
			hasLlama = true
			continue
		}
		ids = append(ids, id)
	}
	sortIDs(ids)
	if hasLlama {
		ids = append(ids, IDLlamaCpp)
	}

	for _, id := range ids {
		p := ctx.Plugins[id]
		if !p.SupportsModel(ctx.ModelPath) {
			continue
		}
		cfg := ctx.Configs[id]
		if cfg == nil {
			continue
		}
		return p, cfg, true
	}
	return nil, nil, false
}

// DefaultForGGUFRule returns llamacpp for GGUF files when nothing more
// specific has matched. llamacpp does not require a Config to function (the
// binary path can come from the legacy global default).
type DefaultForGGUFRule struct {
	LlamaCppID ID
}

func (r DefaultForGGUFRule) Apply(ctx ResolveCtx) (Plugin, *Config, bool) {
	if !IsGGUFModel(ctx.ModelPath) {
		return nil, nil, false
	}
	p, ok := ctx.Plugins[r.LlamaCppID]
	if !ok {
		return nil, nil, false
	}
	cfg := ctx.Configs[r.LlamaCppID]
	return p, cfg, true
}

// ErrorRule never matches; it exists as the terminal rule in the chain so
// Resolve always returns a NoSuitableBackend error after walking every rule.
type ErrorRule struct{}

func (ErrorRule) Apply(ctx ResolveCtx) (Plugin, *Config, bool) {
	return nil, nil, false
}

// sortIDs sorts a slice of ID values in lexicographic order in place.
func sortIDs(ids []ID) {
	// Tiny custom sort to avoid pulling in the sort package signature dance.
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j-1] > ids[j]; j-- {
			ids[j-1], ids[j] = ids[j], ids[j-1]
		}
	}
}
