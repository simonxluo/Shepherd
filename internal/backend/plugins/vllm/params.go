// Package vllm implements the backend.Plugin contract for vLLM.
package vllm

import "github.com/simonxluo/Shepherd/internal/backend"

// Params holds vLLM-specific load parameters.
type Params struct {
	backend.ParamsBase
	DataType             string  `json:"dataType"`
	MaxModelLen          int     `json:"maxModelLen"`
	GPUMemoryUtilization float64 `json:"gpuMemoryUtilization"`
	TensorParallelSize   int     `json:"tensorParallelSize"`
	PipelineParallelSize int     `json:"pipelineParallelSize"`
	TrustRemoteCode      bool    `json:"trustRemoteCode"`
	ServedModelName      string  `json:"servedModelName"`
	Quantization         string  `json:"quantization"`
	MaxNumSeqs           int     `json:"maxNumSeqs"`
	MaxNumBatchedTokens  int     `json:"maxNumBatchedTokens"`
	EnablePrefixCaching  bool    `json:"enablePrefixCaching"`
	EnableChunkedPrefill bool    `json:"enableChunkedPrefill"`
	DisableLogRequests   bool    `json:"disableLogRequests"`
	EnforceEager         bool    `json:"enforceEager"`
	ExtraArgs            string  `json:"extraArgs"`
}

var _ backend.Params = (*Params)(nil)
