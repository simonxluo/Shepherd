package backend

import (
	"fmt"
	"math"
)

// ParamType describes the JSON value type accepted by a backend parameter.
type ParamType string

const (
	ParamTypeInt     ParamType = "int"
	ParamTypeFloat   ParamType = "float"
	ParamTypeString  ParamType = "string"
	ParamTypeBool    ParamType = "bool"
	ParamTypeStrings ParamType = "strings"
)

// ParamDef describes one llama.cpp launch parameter.
type ParamDef struct {
	Name         string    `json:"name"`
	JSONName     string    `json:"jsonName"`
	Flag         string    `json:"flag"`
	Type         ParamType `json:"type"`
	Group        string    `json:"group"`
	Description  string    `json:"description"`
	Default      any       `json:"default,omitempty"`
	Min          *float64  `json:"min,omitempty"`
	Max          *float64  `json:"max,omitempty"`
	Options      []string  `json:"options,omitempty"`
	Advanced     bool      `json:"advanced,omitempty"`
	SinceVersion string    `json:"sinceVersion,omitempty"`
}

// ParamRegistry groups backend parameter definitions.
type ParamRegistry struct {
	Backend string     `json:"backend"`
	Params  []ParamDef `json:"params"`
}

// ParamValidationResult contains non-fatal warnings and fatal validation errors.
type ParamValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func numberPtr(v float64) *float64 { return &v }

// LlamaCppParamRegistry returns the high-value llama.cpp launch parameter schema.
func LlamaCppParamRegistry() ParamRegistry {
	return ParamRegistry{
		Backend: string(BackendLlamaCpp),
		Params: []ParamDef{
			{Name: "Context Size", JSONName: "ctxSize", Flag: "-c", Type: ParamTypeInt, Group: "basic", Description: "Prompt context size", Min: numberPtr(1)},
			{Name: "Batch Size", JSONName: "batchSize", Flag: "-b", Type: ParamTypeInt, Group: "basic", Description: "Logical maximum batch size", Min: numberPtr(1)},
			{Name: "Threads", JSONName: "threads", Flag: "-t", Type: ParamTypeInt, Group: "basic", Description: "Number of generation threads; -1 lets llama.cpp choose", Min: numberPtr(-1)},
			{Name: "GPU Layers", JSONName: "gpuLayers", Flag: "-ngl", Type: ParamTypeInt, Group: "gpu", Description: "Number of model layers to offload to GPU", Min: numberPtr(0)},
			{Name: "Devices", JSONName: "devices", Flag: "-dev", Type: ParamTypeStrings, Group: "gpu", Description: "Comma-separated GPU devices", Advanced: true},
			{Name: "Main GPU", JSONName: "mainGpu", Flag: "-mg", Type: ParamTypeInt, Group: "gpu", Description: "Main GPU index", Min: numberPtr(0), Advanced: true},
			{Name: "Split Mode", JSONName: "splitMode", Flag: "-sm", Type: ParamTypeString, Group: "gpu", Description: "GPU split mode", Options: []string{"none", "layer", "row", "tensor"}, Advanced: true},
			{Name: "Tensor Split", JSONName: "tensorSplit", Flag: "-ts", Type: ParamTypeString, Group: "gpu", Description: "Tensor split fractions", Advanced: true},
			{Name: "KV Cache Type K", JSONName: "kvCacheTypeK", Flag: "-ctk", Type: ParamTypeString, Group: "kv-cache", Description: "KV cache K type", Options: []string{"f32", "f16", "q8_0", "q5_1", "q4_1", "q4_0", "iq4_nl"}, Advanced: true},
			{Name: "KV Cache Type V", JSONName: "kvCacheTypeV", Flag: "-ctv", Type: ParamTypeString, Group: "kv-cache", Description: "KV cache V type", Options: []string{"f32", "f16", "q8_0", "q5_1", "q4_1", "q4_0", "iq4_nl"}, Advanced: true},
			{Name: "Unified KV Cache", JSONName: "kvCacheUnified", Flag: "-kvu", Type: ParamTypeBool, Group: "kv-cache", Description: "Use a single unified KV buffer", Advanced: true},
			{Name: "KV Offload", JSONName: "kvOffload", Flag: "--kv-offload", Type: ParamTypeBool, Group: "kv-cache", Description: "Offload KV cache to device", Advanced: true},
			{Name: "Parallel Slots", JSONName: "parallelSlots", Flag: "--parallel", Type: ParamTypeInt, Group: "server", Description: "Number of parallel request slots", Min: numberPtr(1)},
			{Name: "HTTP Threads", JSONName: "threadsHttp", Flag: "--threads-http", Type: ParamTypeInt, Group: "server", Description: "HTTP handler threads", Min: numberPtr(1), Advanced: true},
			{Name: "Timeout", JSONName: "timeout", Flag: "--timeout", Type: ParamTypeInt, Group: "server", Description: "Request timeout in seconds", Min: numberPtr(1)},
			{Name: "Alias", JSONName: "alias", Flag: "--alias", Type: ParamTypeString, Group: "server", Description: "Served model alias"},
			{Name: "No Web UI", JSONName: "noWebUI", Flag: "--no-webui", Type: ParamTypeBool, Group: "server", Description: "Disable llama.cpp web UI"},
			{Name: "Metrics", JSONName: "enableMetrics", Flag: "--metrics", Type: ParamTypeBool, Group: "server", Description: "Enable metrics endpoint"},
			{Name: "Temperature", JSONName: "temperature", Flag: "--temp", Type: ParamTypeFloat, Group: "sampling", Description: "Sampling temperature", Min: numberPtr(0)},
			{Name: "Top P", JSONName: "topP", Flag: "--top-p", Type: ParamTypeFloat, Group: "sampling", Description: "Nucleus sampling probability", Min: numberPtr(0), Max: numberPtr(1)},
			{Name: "Top K", JSONName: "topK", Flag: "--top-k", Type: ParamTypeInt, Group: "sampling", Description: "Top-k sampling candidates", Min: numberPtr(0)},
			{Name: "Min P", JSONName: "minP", Flag: "--min-p", Type: ParamTypeFloat, Group: "sampling", Description: "Minimum probability sampling", Min: numberPtr(0), Max: numberPtr(1), Advanced: true},
			{Name: "Repeat Penalty", JSONName: "repeatPenalty", Flag: "--repeat-penalty", Type: ParamTypeFloat, Group: "sampling", Description: "Repetition penalty", Min: numberPtr(0)},
			{Name: "Seed", JSONName: "seed", Flag: "--seed", Type: ParamTypeInt, Group: "sampling", Description: "Sampling seed; -1 means random", Min: numberPtr(-1)},
			{Name: "Predict Tokens", JSONName: "nPredict", Flag: "-n", Type: ParamTypeInt, Group: "sampling", Description: "Maximum predicted tokens; -1 means unlimited", Min: numberPtr(-1)},
			{Name: "mmproj", JSONName: "mmprojPath", Flag: "--mmproj", Type: ParamTypeString, Group: "multimodal", Description: "Path to multimodal projector"},
			{Name: "mmproj Offload", JSONName: "mmprojOffload", Flag: "--mmproj-offload", Type: ParamTypeBool, Group: "multimodal", Description: "Offload multimodal projector", Advanced: true},
			{Name: "Flash Attention", JSONName: "flashAttention", Flag: "-fa", Type: ParamTypeBool, Group: "performance", Description: "Enable flash attention"},
			{Name: "No mmap", JSONName: "noMmap", Flag: "--no-mmap", Type: ParamTypeBool, Group: "performance", Description: "Disable memory-mapped model loading", Advanced: true},
			{Name: "Lock Memory", JSONName: "lockMemory", Flag: "--mlock", Type: ParamTypeBool, Group: "performance", Description: "Lock model memory", Advanced: true},
			{Name: "Extra Args", JSONName: "extraArgs", Flag: "", Type: ParamTypeString, Group: "advanced", Description: "Raw llama.cpp CLI arguments appended verbatim", Advanced: true},
		},
	}
}

// ValidateLlamaCppParamMap validates JSON-style llama.cpp parameter values.
func ValidateLlamaCppParamMap(params map[string]any) ParamValidationResult {
	registry := LlamaCppParamRegistry()
	defs := make(map[string]ParamDef, len(registry.Params))
	for _, def := range registry.Params {
		defs[def.JSONName] = def
	}

	result := ParamValidationResult{Valid: true, Errors: []string{}, Warnings: []string{}}
	for name, value := range params {
		def, ok := defs[name]
		if !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf("unknown llama.cpp parameter: %s", name))
			continue
		}
		if value == nil {
			continue
		}
		if err := validateParamValue(def, value); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}
	result.Valid = len(result.Errors) == 0
	return result
}

func validateParamValue(def ParamDef, value any) error {
	switch def.Type {
	case ParamTypeInt:
		n, ok := numericValue(value)
		if !ok || math.Trunc(n) != n {
			return fmt.Errorf("%s must be an integer", def.JSONName)
		}
		return validateNumberRange(def, n)
	case ParamTypeFloat:
		n, ok := numericValue(value)
		if !ok {
			return fmt.Errorf("%s must be a number", def.JSONName)
		}
		return validateNumberRange(def, n)
	case ParamTypeString:
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a string", def.JSONName)
		}
		if len(def.Options) > 0 && s != "" {
			for _, option := range def.Options {
				if s == option {
					return nil
				}
			}
			return fmt.Errorf("%s must be one of: %v", def.JSONName, def.Options)
		}
	case ParamTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", def.JSONName)
		}
	case ParamTypeStrings:
		if _, ok := value.([]string); ok {
			return nil
		}
		values, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be an array of strings", def.JSONName)
		}
		for _, item := range values {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("%s must be an array of strings", def.JSONName)
			}
		}
	}
	return nil
}

func validateNumberRange(def ParamDef, value float64) error {
	if def.Min != nil && value < *def.Min {
		return fmt.Errorf("%s must be >= %v", def.JSONName, *def.Min)
	}
	if def.Max != nil && value > *def.Max {
		return fmt.Errorf("%s must be <= %v", def.JSONName, *def.Max)
	}
	return nil
}

func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}
