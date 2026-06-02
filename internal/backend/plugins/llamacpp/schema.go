package llamacpp

import (
	"fmt"
	"math"

	"github.com/simonxluo/Shepherd/internal/backend"
)

func numberPtr(v float64) *float64 { return &v }

func paramSchema() backend.ParamSchema {
	return backend.ParamSchema{
		PluginID: backend.IDLlamaCpp,
		Params: []backend.ParamDef{
			{Name: "Context Size", JSONName: "ctxSize", Flag: "-c", Type: backend.ParamTypeInt, Group: "basic", Description: "Prompt context size", Min: numberPtr(1)},
			{Name: "Batch Size", JSONName: "batchSize", Flag: "-b", Type: backend.ParamTypeInt, Group: "basic", Description: "Logical maximum batch size", Min: numberPtr(1)},
			{Name: "Threads", JSONName: "threads", Flag: "-t", Type: backend.ParamTypeInt, Group: "basic", Description: "Number of generation threads; -1 lets llama.cpp choose", Min: numberPtr(-1)},
			{Name: "GPU Layers", JSONName: "gpuLayers", Flag: "-ngl", Type: backend.ParamTypeInt, Group: "gpu", Description: "Number of model layers to offload to GPU", Min: numberPtr(0)},
			{Name: "Devices", JSONName: "devices", Flag: "-dev", Type: backend.ParamTypeStrings, Group: "gpu", Description: "Comma-separated GPU devices", Advanced: true},
			{Name: "Main GPU", JSONName: "mainGpu", Flag: "-mg", Type: backend.ParamTypeInt, Group: "gpu", Description: "Main GPU index", Min: numberPtr(0), Advanced: true},
			{Name: "Split Mode", JSONName: "splitMode", Flag: "-sm", Type: backend.ParamTypeString, Group: "gpu", Description: "GPU split mode", Options: []any{"none", "layer", "row", "tensor"}, Advanced: true},
			{Name: "Tensor Split", JSONName: "tensorSplit", Flag: "-ts", Type: backend.ParamTypeString, Group: "gpu", Description: "Tensor split fractions", Advanced: true},
			{Name: "KV Cache Type K", JSONName: "kvCacheTypeK", Flag: "-ctk", Type: backend.ParamTypeString, Group: "kv-cache", Description: "KV cache K type", Options: []any{"f32", "f16", "q8_0", "q5_1", "q4_1", "q4_0", "iq4_nl"}, Advanced: true},
			{Name: "KV Cache Type V", JSONName: "kvCacheTypeV", Flag: "-ctv", Type: backend.ParamTypeString, Group: "kv-cache", Description: "KV cache V type", Options: []any{"f32", "f16", "q8_0", "q5_1", "q4_1", "q4_0", "iq4_nl"}, Advanced: true},
			{Name: "Unified KV Cache", JSONName: "kvCacheUnified", Flag: "-kvu", Type: backend.ParamTypeBool, Group: "kv-cache", Description: "Use a single unified KV buffer", Advanced: true},
			{Name: "KV Offload", JSONName: "kvOffload", Flag: "--kv-offload", Type: backend.ParamTypeBool, Group: "kv-cache", Description: "Offload KV cache to device", Advanced: true},
			{Name: "Parallel Slots", JSONName: "parallelSlots", Flag: "--parallel", Type: backend.ParamTypeInt, Group: "server", Description: "Number of parallel request slots", Min: numberPtr(1)},
			{Name: "HTTP Threads", JSONName: "threadsHttp", Flag: "--threads-http", Type: backend.ParamTypeInt, Group: "server", Description: "HTTP handler threads", Min: numberPtr(1), Advanced: true},
			{Name: "Timeout", JSONName: "timeout", Flag: "--timeout", Type: backend.ParamTypeInt, Group: "server", Description: "Request timeout in seconds", Min: numberPtr(1)},
			{Name: "Alias", JSONName: "alias", Flag: "--alias", Type: backend.ParamTypeString, Group: "server", Description: "Served model alias"},
			{Name: "No Web UI", JSONName: "noWebUI", Flag: "--no-webui", Type: backend.ParamTypeBool, Group: "server", Description: "Disable llama.cpp web UI"},
			{Name: "Metrics", JSONName: "enableMetrics", Flag: "--metrics", Type: backend.ParamTypeBool, Group: "server", Description: "Enable metrics endpoint"},
			{Name: "Temperature", JSONName: "temperature", Flag: "--temp", Type: backend.ParamTypeFloat, Group: "sampling", Description: "Sampling temperature", Min: numberPtr(0)},
			{Name: "Top P", JSONName: "topP", Flag: "--top-p", Type: backend.ParamTypeFloat, Group: "sampling", Description: "Nucleus sampling probability", Min: numberPtr(0), Max: numberPtr(1)},
			{Name: "Top K", JSONName: "topK", Flag: "--top-k", Type: backend.ParamTypeInt, Group: "sampling", Description: "Top-k sampling candidates", Min: numberPtr(0)},
			{Name: "Min P", JSONName: "minP", Flag: "--min-p", Type: backend.ParamTypeFloat, Group: "sampling", Description: "Minimum probability sampling", Min: numberPtr(0), Max: numberPtr(1), Advanced: true},
			{Name: "Repeat Penalty", JSONName: "repeatPenalty", Flag: "--repeat-penalty", Type: backend.ParamTypeFloat, Group: "sampling", Description: "Repetition penalty", Min: numberPtr(0)},
			{Name: "Seed", JSONName: "seed", Flag: "--seed", Type: backend.ParamTypeInt, Group: "sampling", Description: "Sampling seed; -1 means random", Min: numberPtr(-1)},
			{Name: "Predict Tokens", JSONName: "nPredict", Flag: "-n", Type: backend.ParamTypeInt, Group: "sampling", Description: "Maximum predicted tokens; -1 means unlimited", Min: numberPtr(-1)},
			{Name: "mmproj", JSONName: "mmprojPath", Flag: "--mmproj", Type: backend.ParamTypeString, Group: "multimodal", Description: "Path to multimodal projector"},
			{Name: "mmproj Offload", JSONName: "mmprojOffload", Flag: "--mmproj-offload", Type: backend.ParamTypeBool, Group: "multimodal", Description: "Offload multimodal projector", Advanced: true},
			{Name: "Flash Attention", JSONName: "flashAttention", Flag: "-fa", Type: backend.ParamTypeBool, Group: "performance", Description: "Enable flash attention"},
			{Name: "No mmap", JSONName: "noMmap", Flag: "--no-mmap", Type: backend.ParamTypeBool, Group: "performance", Description: "Disable memory-mapped model loading", Advanced: true},
			{Name: "Lock Memory", JSONName: "lockMemory", Flag: "--mlock", Type: backend.ParamTypeBool, Group: "performance", Description: "Lock model memory", Advanced: true},
			{Name: "Extra Args", JSONName: "extraArgs", Flag: "", Type: backend.ParamTypeString, Group: "advanced", Description: "Raw CLI arguments appended verbatim", Advanced: true},
		},
	}
}

func decodeParams(raw backend.RawParams) (*Params, error) {
	p := &Params{}
	if raw == nil {
		return p, nil
	}
	// Only decode fields that the schema exposes. Full field coverage would
	// require a JSON round-trip; we decode the high-frequency ones used in
	// buildArgs. Fields not decoded here are zero-valued and have no effect
	// on the command line.
	setInt(raw, "ctxSize", &p.CtxSize)
	setInt(raw, "batchSize", &p.BatchSize)
	setInt(raw, "threads", &p.Threads)
	setInt(raw, "gpuLayers", &p.GPULayers)
	setInt(raw, "parallelSlots", &p.ParallelSlots)
	setInt(raw, "threadsHttp", &p.ThreadsHTTP)
	setInt(raw, "timeout", &p.Timeout)
	setInt(raw, "seed", &p.Seed)
	setInt(raw, "nPredict", &p.NPredict)
	setInt(raw, "topK", &p.TopK)
	setInt(raw, "mainGpu", &p.MainGPU)

	setFloat(raw, "temperature", &p.Temperature)
	setFloat(raw, "topP", &p.TopP)
	setFloat(raw, "minP", &p.MinP)
	setFloat(raw, "repeatPenalty", &p.RepeatPenalty)
	setFloat(raw, "gpuMemoryUtilization", nil) // not a llamacpp field; ignored

	setBool(raw, "flashAttention", &p.FlashAttention)
	setBool(raw, "noMmap", &p.NoMmap)
	setBool(raw, "lockMemory", &p.LockMemory)
	setBool(raw, "noWebUI", &p.NoWebUI)
	setBool(raw, "enableMetrics", &p.EnableMetrics)
	setBool(raw, "kvCacheUnified", &p.KVCacheUnified)
	setBool(raw, "kvOffload", &p.KVOffload)
	setBool(raw, "mmprojOffload", &p.MmprojOffload)

	setString(raw, "splitMode", &p.SplitMode)
	setString(raw, "tensorSplit", &p.TensorSplit)
	setString(raw, "kvCacheTypeK", &p.KVCacheTypeK)
	setString(raw, "kvCacheTypeV", &p.KVCacheTypeV)
	setString(raw, "alias", &p.Alias)
	setString(raw, "mmprojPath", &p.MmprojPath)
	setString(raw, "extraArgs", &p.ExtraParams)

	return p, nil
}

func validateParams(raw backend.RawParams) backend.ValidationResult {
	schema := paramSchema()
	defs := make(map[string]backend.ParamDef, len(schema.Params))
	for _, d := range schema.Params {
		defs[d.JSONName] = d
	}

	result := backend.ValidationResult{Valid: true}
	for key, value := range raw {
		def, ok := defs[key]
		if !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf("unknown parameter: %s", key))
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

func validateParamValue(def backend.ParamDef, value any) error {
	switch def.Type {
	case backend.ParamTypeInt:
		n, ok := numericValue(value)
		if !ok || math.Trunc(n) != n {
			return fmt.Errorf("%s must be an integer", def.JSONName)
		}
		return validateRange(def, n)
	case backend.ParamTypeFloat:
		n, ok := numericValue(value)
		if !ok {
			return fmt.Errorf("%s must be a number", def.JSONName)
		}
		return validateRange(def, n)
	case backend.ParamTypeString:
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a string", def.JSONName)
		}
		if len(def.Options) > 0 && s != "" {
			for _, opt := range def.Options {
				if s == fmt.Sprint(opt) {
					return nil
				}
			}
			return fmt.Errorf("%s must be one of: %v", def.JSONName, def.Options)
		}
	case backend.ParamTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", def.JSONName)
		}
	case backend.ParamTypeStrings:
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

func validateRange(def backend.ParamDef, v float64) error {
	if def.Min != nil && v < *def.Min {
		return fmt.Errorf("%s must be >= %v", def.JSONName, *def.Min)
	}
	if def.Max != nil && v > *def.Max {
		return fmt.Errorf("%s must be <= %v", def.JSONName, *def.Max)
	}
	return nil
}

func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	}
	return 0, false
}

func setInt(raw backend.RawParams, key string, dst *int) {
	if v, ok := numericValue(raw[key]); ok {
		*dst = int(v)
	}
}

func setFloat(raw backend.RawParams, key string, dst *float64) {
	if dst == nil {
		return
	}
	if v, ok := numericValue(raw[key]); ok {
		*dst = v
	}
}

func setBool(raw backend.RawParams, key string, dst *bool) {
	if v, ok := raw[key].(bool); ok {
		*dst = v
	}
}

func setString(raw backend.RawParams, key string, dst *string) {
	if v, ok := raw[key].(string); ok {
		*dst = v
	}
}
