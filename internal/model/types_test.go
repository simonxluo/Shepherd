// Package model provides model scanning and management functionality.
package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadRequestBackwardCompatibility verifies old configs without new fields still work
func TestLoadRequestBackwardCompatibility(t *testing.T) {
	// This is an old config format with only the original 11 fields
	oldConfigJSON := `{
		"modelId": "test-model-123",
		"nodeId": "node-1",
		"ctxSize": 4096,
		"batchSize": 512,
		"threads": 8,
		"gpuLayers": 35,
		"temperature": 0.7,
		"topP": 0.9,
		"topK": 40,
		"repeatPenalty": 1.1,
		"seed": 42,
		"nPredict": 2048
	}`

	var req LoadRequest
	err := json.Unmarshal([]byte(oldConfigJSON), &req)
	require.NoError(t, err, "Old config format should deserialize without error")

	// Verify all original fields are correctly parsed
	assert.Equal(t, "test-model-123", req.ModelID)
	assert.Equal(t, "node-1", req.NodeID)
	assert.Equal(t, 4096, req.CtxSize)
	assert.Equal(t, 512, req.BatchSize)
	assert.Equal(t, 8, req.Threads)
	assert.Equal(t, 35, req.GPULayers)
	assert.InDelta(t, 0.7, req.Temperature, 0.001)
	assert.InDelta(t, 0.9, req.TopP, 0.001)
	assert.Equal(t, 40, req.TopK)
	assert.InDelta(t, 1.1, req.RepeatPenalty, 0.001)
	assert.Equal(t, 42, req.Seed)
	assert.Equal(t, 2048, req.NPredict)
}

// TestLoadRequestAllNewFields verifies all new fields deserialize correctly
func TestLoadRequestAllNewFields(t *testing.T) {
	// Full config with all new fields
	fullConfigJSON := `{
		"modelId": "test-model-456",
		"nodeId": "node-2",
		"ctxSize": 8192,
		"batchSize": 1024,
		"threads": 16,
		"gpuLayers": 99,
		"temperature": 0.8,
		"topP": 0.95,
		"topK": 50,
		"repeatPenalty": 1.2,
		"seed": 12345,
		"nPredict": 4096,
		"devices": ["cuda:0", "cuda:1"],
		"mainGpu": 0,
		"llamaCppPath": "/custom/llama-server",
		"extraArgs": "--special-flag",
		"mmprojPath": "/models/mmproj.gguf",
		"enableVision": true,
		"flashAttention": true,
		"noMmap": true,
		"lockMemory": true,
		"noWebUI": true,
		"enableMetrics": true,
		"slotSavePath": "/slots",
		"cacheRam": 2048,
		"chatTemplateFile": "/templates/chat.tmpl",
		"timeout": 300,
		"alias": "my-custom-alias",
		"ubatchSize": 128,
		"parallelSlots": 4,
		"kvCacheTypeK": "f16",
		"kvCacheTypeV": "q8_0",
		"kvCacheUnified": true,
		"kvCacheSize": 16384,
		"logitsAll": true,
		"reranking": false,
		"minP": 0.05,
		"presencePenalty": 0.1,
		"frequencyPenalty": 0.2,
		"directIo": "5,5,5",
		"disableJinja": true,
		"chatTemplate": "chatml",
		"contextShift": true
	}`

	var req LoadRequest
	err := json.Unmarshal([]byte(fullConfigJSON), &req)
	require.NoError(t, err, "Full config should deserialize without error")

	// Verify original fields
	assert.Equal(t, "test-model-456", req.ModelID)
	assert.Equal(t, "node-2", req.NodeID)
	assert.Equal(t, 8192, req.CtxSize)

	// Verify new GPU fields
	assert.Equal(t, []string{"cuda:0", "cuda:1"}, req.Devices)
	assert.Equal(t, 0, req.MainGPU)

	// Verify custom command fields (use camelCase JSON tags)
	assert.Equal(t, "/custom/llama-server", req.CustomCmd)
	assert.Equal(t, "--special-flag", req.ExtraParams)

	// Verify vision/mmproj fields
	assert.Equal(t, "/models/mmproj.gguf", req.MmprojPath)
	assert.True(t, req.EnableVision)

	// Verify feature flags
	assert.True(t, req.FlashAttention)
	assert.True(t, req.NoMmap)
	assert.True(t, req.LockMemory)
	assert.True(t, req.NoWebUI)
	assert.True(t, req.EnableMetrics)

	// Verify paths and sizes
	assert.Equal(t, "/slots", req.SlotSavePath)
	assert.Equal(t, 2048, req.CacheRAM)
	assert.Equal(t, "/templates/chat.tmpl", req.ChatTemplateFile)

	// Verify timeout and alias
	assert.Equal(t, 300, req.Timeout)
	assert.Equal(t, "my-custom-alias", req.Alias)

	// Verify batch/parallel settings
	assert.Equal(t, 128, req.UBatchSize)
	assert.Equal(t, 4, req.ParallelSlots)

	// Verify KV cache settings
	assert.Equal(t, "f16", req.KVCacheTypeK)
	assert.Equal(t, "q8_0", req.KVCacheTypeV)
	assert.True(t, req.KVCacheUnified)
	assert.Equal(t, 16384, req.KVCacheSize)

	// Verify additional sampling parameters
	assert.True(t, req.LogitsAll)
	assert.False(t, req.Reranking)
	assert.InDelta(t, 0.05, req.MinP, 0.001)
	assert.InDelta(t, 0.1, req.PresencePenalty, 0.001)
	assert.InDelta(t, 0.2, req.FrequencyPenalty, 0.001)

	// Verify template and processing
	assert.Equal(t, "5,5,5", req.DirectIo)
	assert.True(t, req.DisableJinja)
	assert.Equal(t, "chatml", req.ChatTemplate)
	assert.True(t, req.ContextShift)
}

// TestLoadRequestMissingFieldsGetZeroValues verifies missing fields get sensible zero values
func TestLoadRequestMissingFieldsGetZeroValues(t *testing.T) {
	minimalJSON := `{
		"modelId": "minimal-model"
	}`

	var req LoadRequest
	err := json.Unmarshal([]byte(minimalJSON), &req)
	require.NoError(t, err)

	// Only ModelID should be set
	assert.Equal(t, "minimal-model", req.ModelID)

	// All other fields should be zero values
	assert.Empty(t, req.NodeID)
	assert.Zero(t, req.CtxSize)
	assert.Zero(t, req.BatchSize)
	assert.Zero(t, req.Threads)
	assert.Zero(t, req.GPULayers)
	assert.Zero(t, req.Temperature)
	assert.Zero(t, req.TopP)
	assert.Zero(t, req.TopK)
	assert.Zero(t, req.RepeatPenalty)
	assert.Zero(t, req.Seed)
	assert.Zero(t, req.NPredict)

	// New fields should also be zero values
	assert.Nil(t, req.Devices)
	assert.Zero(t, req.MainGPU)
	assert.Empty(t, req.CustomCmd)
	assert.Empty(t, req.ExtraParams)
	assert.Empty(t, req.MmprojPath)
	assert.False(t, req.EnableVision)
	assert.False(t, req.FlashAttention)
	assert.False(t, req.NoMmap)
	assert.False(t, req.LockMemory)
	assert.False(t, req.NoWebUI)
	assert.False(t, req.EnableMetrics)
	assert.Empty(t, req.SlotSavePath)
	assert.Zero(t, req.CacheRAM)
	assert.Empty(t, req.ChatTemplateFile)
	assert.Zero(t, req.Timeout)
	assert.Empty(t, req.Alias)
	assert.Zero(t, req.UBatchSize)
	assert.Zero(t, req.ParallelSlots)
	assert.Empty(t, req.KVCacheTypeK)
	assert.Empty(t, req.KVCacheTypeV)
	assert.False(t, req.KVCacheUnified)
	assert.Zero(t, req.KVCacheSize)
	// Additional sampling parameters
	assert.False(t, req.LogitsAll)
	assert.False(t, req.Reranking)
	assert.Zero(t, req.MinP)
	assert.Zero(t, req.PresencePenalty)
	assert.Zero(t, req.FrequencyPenalty)
	// Template and processing
	assert.Empty(t, req.DirectIo)
	assert.False(t, req.DisableJinja)
	assert.Empty(t, req.ChatTemplate)
	assert.False(t, req.ContextShift)
}

// TestLoadRequestJSONRoundTrip verifies marshal/unmarshal preserves all data
func TestLoadRequestJSONRoundTrip(t *testing.T) {
	original := LoadRequest{
		ModelID:          "roundtrip-test",
		NodeID:           "node-3",
		CtxSize:          4096,
		BatchSize:        512,
		Threads:          8,
		GPULayers:        35,
		Temperature:      0.7,
		TopP:             0.9,
		TopK:             40,
		RepeatPenalty:    1.1,
		Seed:             42,
		NPredict:         2048,
		Devices:          []string{"cuda:0", "cuda:1", "cuda:2"},
		MainGPU:          1,
		CustomCmd:        "/path/to/custom",
		ExtraParams:      "--flag1 --flag2",
		MmprojPath:       "/path/to/mmproj.gguf",
		EnableVision:     true,
		FlashAttention:   true,
		NoMmap:           false,
		LockMemory:       true,
		NoWebUI:          false,
		EnableMetrics:    true,
		SlotSavePath:     "/save/slots",
		CacheRAM:         1024,
		ChatTemplateFile: "/templates/custom.tmpl",
		Timeout:          120,
		Alias:            "test-alias",
		UBatchSize:       64,
		ParallelSlots:    2,
		KVCacheTypeK:     "q8_0",
		KVCacheTypeV:     "q4_0",
		KVCacheUnified:   true,
		KVCacheSize:      8192,
		// Additional sampling parameters
		LogitsAll:        true,
		Reranking:        false,
		MinP:             0.05,
		PresencePenalty:  0.1,
		FrequencyPenalty: 0.2,
		// Template and processing
		DirectIo:     "5,5,5",
		DisableJinja: true,
		ChatTemplate: "chatml",
		ContextShift: false,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err, "Marshal should succeed")

	// Unmarshal back
	var restored LoadRequest
	err = json.Unmarshal(data, &restored)
	require.NoError(t, err, "Unmarshal should succeed")

	// Verify all fields match
	assert.Equal(t, original.ModelID, restored.ModelID)
	assert.Equal(t, original.NodeID, restored.NodeID)
	assert.Equal(t, original.CtxSize, restored.CtxSize)
	assert.Equal(t, original.BatchSize, restored.BatchSize)
	assert.Equal(t, original.Threads, restored.Threads)
	assert.Equal(t, original.GPULayers, restored.GPULayers)
	assert.InDelta(t, original.Temperature, restored.Temperature, 0.001)
	assert.InDelta(t, original.TopP, restored.TopP, 0.001)
	assert.Equal(t, original.TopK, restored.TopK)
	assert.InDelta(t, original.RepeatPenalty, restored.RepeatPenalty, 0.001)
	assert.Equal(t, original.Seed, restored.Seed)
	assert.Equal(t, original.NPredict, restored.NPredict)
	assert.Equal(t, original.Devices, restored.Devices)
	assert.Equal(t, original.MainGPU, restored.MainGPU)
	assert.Equal(t, original.CustomCmd, restored.CustomCmd)
	assert.Equal(t, original.ExtraParams, restored.ExtraParams)
	assert.Equal(t, original.MmprojPath, restored.MmprojPath)
	assert.Equal(t, original.EnableVision, restored.EnableVision)
	assert.Equal(t, original.FlashAttention, restored.FlashAttention)
	assert.Equal(t, original.NoMmap, restored.NoMmap)
	assert.Equal(t, original.LockMemory, restored.LockMemory)
	assert.Equal(t, original.NoWebUI, restored.NoWebUI)
	assert.Equal(t, original.EnableMetrics, restored.EnableMetrics)
	assert.Equal(t, original.SlotSavePath, restored.SlotSavePath)
	assert.Equal(t, original.CacheRAM, restored.CacheRAM)
	assert.Equal(t, original.ChatTemplateFile, restored.ChatTemplateFile)
	assert.Equal(t, original.Timeout, restored.Timeout)
	assert.Equal(t, original.Alias, restored.Alias)
	assert.Equal(t, original.UBatchSize, restored.UBatchSize)
	assert.Equal(t, original.ParallelSlots, restored.ParallelSlots)
	assert.Equal(t, original.KVCacheTypeK, restored.KVCacheTypeK)
	assert.Equal(t, original.KVCacheTypeV, restored.KVCacheTypeV)
	assert.Equal(t, original.KVCacheUnified, restored.KVCacheUnified)
	assert.Equal(t, original.KVCacheSize, restored.KVCacheSize)
	// Additional sampling parameters
	assert.Equal(t, original.LogitsAll, restored.LogitsAll)
	assert.Equal(t, original.Reranking, restored.Reranking)
	assert.InDelta(t, original.MinP, restored.MinP, 0.001)
	assert.InDelta(t, original.PresencePenalty, restored.PresencePenalty, 0.001)
	assert.InDelta(t, original.FrequencyPenalty, restored.FrequencyPenalty, 0.001)
	// Template and processing
	assert.Equal(t, original.DirectIo, restored.DirectIo)
	assert.Equal(t, original.DisableJinja, restored.DisableJinja)
	assert.Equal(t, original.ChatTemplate, restored.ChatTemplate)
	assert.Equal(t, original.ContextShift, restored.ContextShift)
}

// TestLoadRequestPartialNewFields verifies partial new fields work correctly
func TestLoadRequestPartialNewFields(t *testing.T) {
	partialJSON := `{
		"modelId": "partial-test",
		"ctxSize": 2048,
		"devices": ["cuda:0"],
		"flashAttention": true,
		"ubatchSize": 32
	}`

	var req LoadRequest
	err := json.Unmarshal([]byte(partialJSON), &req)
	require.NoError(t, err)

	assert.Equal(t, "partial-test", req.ModelID)
	assert.Equal(t, 2048, req.CtxSize)
	assert.Equal(t, []string{"cuda:0"}, req.Devices)
	assert.True(t, req.FlashAttention)
	assert.Equal(t, 32, req.UBatchSize)

	// Unspecified new fields should be zero values
	assert.Zero(t, req.MainGPU)
	assert.Empty(t, req.CustomCmd)
	assert.False(t, req.EnableVision)
	assert.False(t, req.NoMmap)
}
