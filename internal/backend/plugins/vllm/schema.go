package vllm

import (
	"fmt"
	"math"

	"github.com/simonxluo/Shepherd/internal/backend"
)

func numberPtr(v float64) *float64 { return &v }

func paramSchema() backend.ParamSchema {
	return backend.ParamSchema{
		PluginID: backend.IDVLLM,
		Params: []backend.ParamDef{
			{Name: "Max Model Length", JSONName: "maxModelLen", Flag: "--max-model-len", Type: backend.ParamTypeInt, Group: "basic", Description: "Maximum model context length", Min: numberPtr(1)},
			{Name: "Data Type", JSONName: "dataType", Flag: "--dtype", Type: backend.ParamTypeString, Group: "basic", Description: "Model data type", Options: []any{"auto", "float16", "bfloat16", "float32"}},
			{Name: "GPU Memory Utilization", JSONName: "gpuMemoryUtilization", Flag: "--gpu-memory-utilization", Type: backend.ParamTypeFloat, Group: "gpu", Description: "GPU memory utilization (0-1)", Min: numberPtr(0), Max: numberPtr(1)},
			{Name: "Tensor Parallel Size", JSONName: "tensorParallelSize", Flag: "--tensor-parallel-size", Type: backend.ParamTypeInt, Group: "gpu", Description: "Tensor parallel size", Min: numberPtr(1)},
			{Name: "Pipeline Parallel Size", JSONName: "pipelineParallelSize", Flag: "--pipeline-parallel-size", Type: backend.ParamTypeInt, Group: "gpu", Description: "Pipeline parallel size", Min: numberPtr(1), Advanced: true},
			{Name: "Trust Remote Code", JSONName: "trustRemoteCode", Flag: "--trust-remote-code", Type: backend.ParamTypeBool, Group: "basic", Description: "Trust remote code from HuggingFace"},
			{Name: "Served Model Name", JSONName: "servedModelName", Flag: "--served-model-name", Type: backend.ParamTypeString, Group: "basic", Description: "Served model name override"},
			{Name: "Quantization", JSONName: "quantization", Flag: "--quantization", Type: backend.ParamTypeString, Group: "basic", Description: "Quantization method", Options: []any{"awq", "gptq", "squeezellm"}},
			{Name: "Max Num Seqs", JSONName: "maxNumSeqs", Flag: "--max-num-seqs", Type: backend.ParamTypeInt, Group: "server", Description: "Maximum concurrent sequences", Min: numberPtr(1), Advanced: true},
			{Name: "Max Num Batched Tokens", JSONName: "maxNumBatchedTokens", Flag: "--max-num-batched-tokens", Type: backend.ParamTypeInt, Group: "server", Description: "Maximum batched tokens", Min: numberPtr(1), Advanced: true},
			{Name: "Enable Prefix Caching", JSONName: "enablePrefixCaching", Flag: "--enable-prefix-caching", Type: backend.ParamTypeBool, Group: "performance", Description: "Enable prefix caching", Advanced: true},
			{Name: "Enable Chunked Prefill", JSONName: "enableChunkedPrefill", Flag: "--enable-chunked-prefill", Type: backend.ParamTypeBool, Group: "performance", Description: "Enable chunked prefill", Advanced: true},
			{Name: "Disable Log Requests", JSONName: "disableLogRequests", Flag: "--disable-log-requests", Type: backend.ParamTypeBool, Group: "server", Description: "Disable request logging"},
			{Name: "Enforce Eager", JSONName: "enforceEager", Flag: "--enforce-eager", Type: backend.ParamTypeBool, Group: "performance", Description: "Enforce eager execution", Advanced: true},
			{Name: "Extra Args", JSONName: "extraArgs", Flag: "", Type: backend.ParamTypeString, Group: "advanced", Description: "Raw CLI arguments appended verbatim", Advanced: true},
		},
	}
}

// decodeParams converts a RawParams map into a typed *Params.
func decodeParams(raw backend.RawParams) (*Params, error) {
	p := &Params{}
	if raw == nil {
		return p, nil
	}
	if v, ok := raw["dataType"].(string); ok {
		p.DataType = v
	}
	if v, ok := numericValue(raw["maxModelLen"]); ok {
		p.MaxModelLen = int(v)
	}
	if v, ok := numericValue(raw["gpuMemoryUtilization"]); ok {
		p.GPUMemoryUtilization = v
	}
	if v, ok := numericValue(raw["tensorParallelSize"]); ok {
		p.TensorParallelSize = int(v)
	}
	if v, ok := numericValue(raw["pipelineParallelSize"]); ok {
		p.PipelineParallelSize = int(v)
	}
	if v, ok := raw["trustRemoteCode"].(bool); ok {
		p.TrustRemoteCode = v
	}
	if v, ok := raw["servedModelName"].(string); ok {
		p.ServedModelName = v
	}
	if v, ok := raw["quantization"].(string); ok {
		p.Quantization = v
	}
	if v, ok := numericValue(raw["maxNumSeqs"]); ok {
		p.MaxNumSeqs = int(v)
	}
	if v, ok := numericValue(raw["maxNumBatchedTokens"]); ok {
		p.MaxNumBatchedTokens = int(v)
	}
	if v, ok := raw["enablePrefixCaching"].(bool); ok {
		p.EnablePrefixCaching = v
	}
	if v, ok := raw["enableChunkedPrefill"].(bool); ok {
		p.EnableChunkedPrefill = v
	}
	if v, ok := raw["disableLogRequests"].(bool); ok {
		p.DisableLogRequests = v
	}
	if v, ok := raw["enforceEager"].(bool); ok {
		p.EnforceEager = v
	}
	if v, ok := raw["extraArgs"].(string); ok {
		p.ExtraArgs = v
	}
	return p, nil
}

// validateParams checks a RawParams map against the schema.
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
