// Package vllmomni implements the backend.Plugin contract for vLLM-Omni.
package vllmomni

import (
	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/backend/plugins/vllm"
)

// Params holds vLLM-Omni-specific load parameters.
type Params struct {
	backend.ParamsBase
	Base             vllm.Params `json:"base"`
	Omni             bool        `json:"omni"`
	VideoPruningRate float64     `json:"videoPruningRate"`
	MMTensorIPC      bool        `json:"mmTensorIpc"`
}

var _ backend.Params = (*Params)(nil)
