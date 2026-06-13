package backend

import (
	"path/filepath"
	"strings"
)

// IsGGUFModel reports whether modelPath looks like a GGUF model.
// Case-insensitive on the .gguf extension.
func IsGGUFModel(modelPath string) bool {
	if modelPath == "" {
		return false
	}
	return strings.EqualFold(filepath.Ext(modelPath), ".gguf")
}

// IsSafeTensorsModel reports whether modelPath looks like a safetensors model
// or a HuggingFace `snapshots/<rev>` directory layout (which contains
// safetensors shards but is referenced by the directory path).
func IsSafeTensorsModel(modelPath string) bool {
	if modelPath == "" {
		return false
	}
	if strings.EqualFold(filepath.Ext(modelPath), ".safetensors") {
		return true
	}
	// HuggingFace cache layout: .../snapshots/<rev>/...
	return strings.Contains(modelPath, string(filepath.Separator)+"snapshots"+string(filepath.Separator))
}
