package backend

import (
	"os"
	"strings"
)

// BuildEnvWithVars produces a process environment that starts from os.Environ()
// and applies overrides from envVars in NAME=VALUE form. Later overrides win.
//
// Empty strings and entries without '=' are skipped (with no error — they are
// typically operator typos in YAML and should not block startup).
func BuildEnvWithVars(envVars []string) []string {
	if len(envVars) == 0 {
		// Cheap path: caller likely wants the inherited env unchanged.
		return os.Environ()
	}

	// Start with current env, indexed by NAME for quick override.
	base := os.Environ()
	idx := make(map[string]int, len(base))
	for i, kv := range base {
		if eq := strings.IndexByte(kv, '='); eq > 0 {
			idx[kv[:eq]] = i
		}
	}

	for _, kv := range envVars {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		name := kv[:eq]
		if i, ok := idx[name]; ok {
			base[i] = kv
		} else {
			idx[name] = len(base)
			base = append(base, kv)
		}
	}
	return base
}
