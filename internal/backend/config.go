package backend

// PluginConfigDecoder is implemented by plugins that want to decode their own
// per-plugin slice of server.config.yaml. It is consulted by Registry.SyncFromConfig
// (PR3) when the operator's YAML carries a `backends.<id>:` block.
//
// PR1 ships only the contract; the resolver and registry plumbing land in
// PR1, and the YAML loader migration that calls DecodeConfig lands in PR3.
type PluginConfigDecoder interface {
	// DecodeConfig consumes the raw map decoded from a YAML node and
	// returns a typed config struct. The returned value is opaque to the
	// registry; the plugin reads it back from Config.Raw[pluginInternalKey]
	// when needed.
	//
	// Implementations should accept nil/empty maps as "use defaults".
	DecodeConfig(raw map[string]any) (any, error)
}

// PathEntry is the {path, name} pair accepted under the `paths` key in any
// plugin's YAML config (e.g. multiple llama.cpp install locations to probe).
// Lifted to the package level so plugins do not have to redeclare it.
type PathEntry struct {
	Path string `yaml:"path" json:"path"`
	Name string `yaml:"name" json:"name"`
}

// MergeRawMaps combines a sequence of raw YAML maps left-to-right, with
// later entries overriding earlier ones. Keys missing from later maps are
// preserved from earlier. Used by plugin config decoders that want to layer
// "defaults <- global config <- per-model overrides".
//
// Returns a freshly allocated map; inputs are not modified. nil inputs are
// treated as empty.
func MergeRawMaps(maps ...map[string]any) map[string]any {
	out := make(map[string]any)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
