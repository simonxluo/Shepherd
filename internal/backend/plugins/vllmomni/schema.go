package vllmomni

import (
	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/backend/plugins/vllm"
)

func paramSchema() backend.ParamSchema {
	// Start from vllm's schema and add omni-specific params.
	base := vllm.New().ParamSchema()
	extra := []backend.ParamDef{
		{Name: "Video Pruning Rate", JSONName: "videoPruningRate", Flag: "--video-pruning-rate", Type: backend.ParamTypeFloat, Group: "multimodal", Description: "Video frame pruning rate (0-1)", Min: numberPtr(0), Max: numberPtr(1)},
		{Name: "MM Tensor IPC", JSONName: "mmTensorIpc", Flag: "--mm-tensor-ipc", Type: backend.ParamTypeBool, Group: "multimodal", Description: "Enable multimodal tensor IPC"},
	}
	return backend.ParamSchema{
		PluginID: backend.IDVLLMOmni,
		Params:   append(base.Params, extra...),
	}
}

func decodeParams(raw backend.RawParams) (*Params, error) {
	// Decode the base vllm params first.
	baseParams, err := vllm.New().DecodeParams(raw)
	if err != nil {
		return nil, err
	}
	vp, _ := baseParams.(*vllm.Params)
	if vp == nil {
		vp = &vllm.Params{}
	}

	p := &Params{Base: *vp}
	if v, ok := raw["videoPruningRate"]; ok {
		if f, ok := v.(float64); ok {
			p.VideoPruningRate = f
		}
	}
	if v, ok := raw["mmTensorIpc"].(bool); ok {
		p.MMTensorIPC = v
	}
	return p, nil
}

func validateParams(raw backend.RawParams) backend.ValidationResult {
	return vllm.New().ValidateParams(raw)
}

func numberPtr(v float64) *float64 { return &v }
