package backend

import "github.com/simonxluo/Shepherd/internal/comm/config"

// SyncFromConfig populates the registry's Config entries from the application
// config.Config. Iterates cfg.Backends generically — no per-plugin hardcoding.
func (r *Registry) SyncFromConfig(cfg *config.Config) {
	bindHost := cfg.Server.ModelBindHost
	if bindHost == "" {
		bindHost = "0.0.0.0"
	}

	for idStr, rawVal := range cfg.Backends {
		id := ID(idStr)
		plugin, ok := r.Get(id)
		if !ok {
			continue
		}

		raw, _ := rawVal.(map[string]any)

		// Skip disabled backends.
		if v, ok := raw["enabled"]; ok {
			if b, _ := v.(bool); !b {
				continue
			}
		}

		backendCfg := &Config{
			ID:          id,
			DisplayName: plugin.DisplayName(),
			BindHost:    bindHost,
			Raw:         raw,
		}

		// Extract bin paths from the "paths" field.
		if pathsVal, ok := raw["paths"]; ok {
			backendCfg.BinPaths = extractBinPaths(pathsVal)
		}

		// Let plugin decode its typed config if it implements PluginConfigDecoder.
		if decoder, ok := plugin.(PluginConfigDecoder); ok {
			decoded, err := decoder.DecodeConfig(raw)
			if err == nil {
				backendCfg.Decoded = decoded
			}
		}

		r.Configure(id, backendCfg)
	}
}

// extractBinPaths pulls path strings from a raw "paths" YAML value.
func extractBinPaths(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		switch val := item.(type) {
		case map[string]any:
			if p, _ := val["path"].(string); p != "" {
				out = append(out, p)
			}
		case string:
			if val != "" {
				out = append(out, val)
			}
		}
	}
	return out
}
