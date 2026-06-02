package backend

// PluginConfigDecoder lets a plugin decode its own `backends.<id>:` YAML
// slice. Registry.SyncFromConfig calls DecodeConfig and stashes the result
// on Config.Decoded.
type PluginConfigDecoder interface {
	// DecodeConfig converts a raw YAML map into a plugin-specific typed
	// value, stored on Config.Decoded. nil/empty input means "use defaults".
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
