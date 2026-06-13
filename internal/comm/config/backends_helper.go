package config

// BackendRaw returns the raw config map for a backend, or nil if not present.
func (c *Config) BackendRaw(id string) map[string]any {
	if c.Backends == nil {
		return nil
	}
	v, ok := c.Backends[id]
	if !ok {
		return nil
	}
	m, _ := v.(map[string]any)
	return m
}

// BackendEnabled checks if a backend is enabled in the config map.
// Backends without an explicit "enabled" field are considered enabled
// (e.g. llamacpp, which is always available when configured).
func (c *Config) BackendEnabled(id string) bool {
	raw := c.BackendRaw(id)
	if raw == nil {
		return false
	}
	v, ok := raw["enabled"]
	if !ok {
		return true // no "enabled" key means always on
	}
	b, _ := v.(bool)
	return b
}

// BackendPaths returns the paths list for a backend from the config map.
func (c *Config) BackendPaths(id string) []BackendPath {
	raw := c.BackendRaw(id)
	if raw == nil {
		return nil
	}
	return extractBackendPaths(raw["paths"])
}

// SetBackendPaths updates the paths list for a backend in the config map.
// Creates the backend entry if it does not exist.
func (c *Config) SetBackendPaths(id string, paths []BackendPath) {
	if c.Backends == nil {
		c.Backends = make(map[string]any)
	}
	raw := c.BackendRaw(id)
	if raw == nil {
		raw = make(map[string]any)
	}
	// Convert to []any for YAML-safe storage.
	out := make([]any, len(paths))
	for i, p := range paths {
		m := map[string]any{"path": p.Path, "name": p.Name}
		if p.Description != "" {
			m["description"] = p.Description
		}
		out[i] = m
	}
	raw["paths"] = out
	c.Backends[id] = raw
}

// BackendStringField reads a string field from a backend's raw config.
func (c *Config) BackendStringField(id, field string) string {
	raw := c.BackendRaw(id)
	if raw == nil {
		return ""
	}
	s, _ := raw[field].(string)
	return s
}

// extractBackendPaths converts a raw "paths" value into []BackendPath.
func extractBackendPaths(v any) []BackendPath {
	if v == nil {
		return nil
	}
	switch items := v.(type) {
	case []any:
		out := make([]BackendPath, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			bp := BackendPath{}
			bp.Path, _ = m["path"].(string)
			bp.Name, _ = m["name"].(string)
			bp.Description, _ = m["description"].(string)
			if bp.Path != "" {
				out = append(out, bp)
			}
		}
		return out
	case []map[string]any:
		out := make([]BackendPath, 0, len(items))
		for _, m := range items {
			bp := BackendPath{}
			bp.Path, _ = m["path"].(string)
			bp.Name, _ = m["name"].(string)
			bp.Description, _ = m["description"].(string)
			if bp.Path != "" {
				out = append(out, bp)
			}
		}
		return out
	}
	return nil
}
