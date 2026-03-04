package gguf

import (
	"fmt"

	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
)

// Metadata represents the parsed GGUF model metadata
// It contains the most important fields needed for model management
// This structure is designed to fully leverage gguf-parser-go library
type Metadata struct {
	/* ========== Basic Information ========== */

	// Name is the human-readable name of the model
	Name string

	// Architecture describes what architecture this GGUF file implements
	// Examples: "llama", "qwen2", "mistral", "gpt2", "deepseek2", "qwen3next"
	Architecture string

	// Quantization is the human-readable quantization type string
	// Examples: "Q4_K_M", "Q8_0", "F16", "MXFP4"
	Quantization string

	// Type describes what type this GGUF file is
	// Examples: "model", "adapter", "projector", "imatrix"
	Type string

	// Author is the creator of the model
	Author string

	// URL is the model's homepage URL
	URL string

	// Description provides a detailed description of the model
	Description string

	// License is the SPDX license expression (e.g., "MIT OR Apache-2.0")
	License string

	/* ========== Model Parameters ========== */

	// Parameters is the total number of parameters in the model
	// Stored as raw count (use GetParametersInBillions() for human-readable format)
	Parameters float64

	// ContextLength is the maximum context window size
	ContextLength int

	// EmbeddingLength is the dimension of token embeddings
	EmbeddingLength int

	// FeedForwardLength is the size of feed-forward networks
	FeedForwardLength int

	// BlockSize is the number of transformer blocks/layers
	BlockSize int

	// HeadCount is the number of attention heads
	HeadCount int

	// HeadCountKV is the number of key-value heads (for GQA)
	HeadCountKV int

	// LayerNormRMS_EPS is the epsilon value for layer normalization
	LayerNormRMS_EPS float64

	/* ========== Tokenizer Information ========== */

	// TokenCount is the size of the tokenizer vocabulary
	TokenCount int

	// TokenizerModel is the type of tokenizer used
	// Examples: "llama", "gpt2", "rwkv"
	TokenizerModel string

	// BosTokenID is the beginning-of-sequence token ID
	BosTokenID int

	// EosTokenID is the end-of-sequence token ID
	EosTokenID int

	// PadTokenID is the padding token ID
	PadTokenID int

	// UncTokenID is the unknown token ID
	UncTokenID int

	/* ========== Rope Settings ========== */

	// RopeDim is the rotary position embedding dimension
	RopeDim int

	// RopeFreqBase is the frequency base for rotary embeddings
	RopeFreqBase float64

	// RopeFreqScale is the frequency scaling factor
	RopeFreqScale float64

	/* ========== Quantization Settings ========== */

	// QuantizationVersion is the version of the quantization format
	QuantizationVersion uint32

	// FileType is the GGUF file type code (determines quantization scheme)
	FileType uint32

	// FileTypeDescriptor provides detailed description of the file type
	// Examples: "Q4_K_M", "Q4_K_L", "Q5_0_L"
	FileTypeDescriptor string

	/* ========== File Information ========== */

	// Alignment is the byte alignment of the GGUF file (must be multiple of 8)
	Alignment uint32

	// LittleEndian indicates if the file uses little-endian byte order
	LittleEndian bool

	// FileSize is the total size of the GGUF file in bytes
	FileSize uint64

	// ModelSize is the size of the model data in bytes
	ModelSize uint64

	// BitsPerWeight is the average number of bits per weight
	BitsPerWeight float64

	/* ========== Special Tokens ========== */

	// PreToken is the string to prepend to tokens during tokenization
	PreToken string

	// PostToken is the string to append to tokens during tokenization
	PostToken string

	/* ========== Tokenizer Chat Template ========== */

	// ChatTemplate is the Jinja chat template for conversation formatting
	// This contains the template used for formatting conversations in llama.cpp
	// Used for automatic capability detection (tools, thinking, etc.)
	ChatTemplate string

	/* ========== Additional Metadata ========== */

	// Extra stores any additional metadata as key-value pairs
	Extra map[string]interface{}
}

// FileTypeMap maps GGUF file type codes to their string names
var FileTypeMap = map[uint32]string{
	0:  "ALL_F32",
	1:  "MOSTLY_F16",
	2:  "MOSTLY_Q4_0",
	3:  "MOSTLY_Q4_1",
	4:  "MOSTLY_Q4_2", // Unsupported
	5:  "MOSTLY_Q4_3", // Unsupported
	6:  "MOSTLY_Q5_0",
	7:  "MOSTLY_Q5_1",
	8:  "MOSTLY_Q8_0",
	9:  "MOSTLY_Q8_1",
	10: "MOSTLY_Q2_K", // Unsupported
	11: "MOSTLY_Q3_K",
	12: "MOSTLY_Q4_K",
	13: "MOSTLY_Q5_K",
	14: "MOSTLY_Q6_K",
	15: "MOSTLY_Q2_K_S",
	16: "MOSTLY_Q3_K_S",
	17: "MOSTLY_Q2_K_XL",
	18: "MOSTLY_Q3_K_XL",
	19: "MOSTLY_Q1_K", // Unsupported
	20: "MOSTLY_Q4_K_S",
	21: "MOSTLY_Q3_K_S_XL",
	22: "MOSTLY_Q2_K_XL",
}

// GetQuantizationString returns the human-readable quantization string
// 优先使用 FileTypeDescriptor（更准确），如果为空则使用 FileType 编号映射
func (m *Metadata) GetQuantizationString() string {
	logger.Infof("here get the Quante Type:%s,with model name:%s", m.FileTypeDescriptor, m.Name)
	// 优先使用 FileTypeDescriptor（如果存在）
	// 这通常包含更准确的量化类型描述，如 "Q6_K_XL"
	if m.FileTypeDescriptor != "" {
		return m.FileTypeDescriptor
	}

	if m.FileType == 0 {
		return "F32"
	}

	// GGUFFileType 到字符串的完整映射（与 gguf-parser-go 保持一致）
	typeMap := map[uint32]string{
		0:  "F32",       // MOSTLY_F32
		1:  "F16",       // MOSTLY_F16
		2:  "Q4_0",      // MOSTLY_Q4_0
		3:  "Q4_1",      // MOSTLY_Q4_1
		4:  "Q4_2",      // MOSTLY_Q4_2 (已废弃)
		5:  "Q4_3",      // MOSTLY_Q4_3 (已废弃)
		6:  "Q5_0",      // MOSTLY_Q5_0
		7:  "Q5_1",      // MOSTLY_Q5_1
		8:  "Q8_0",      // MOSTLY_Q8_0
		9:  "Q8_1",      // MOSTLY_Q8_1
		10: "Q2_K",      // MOSTLY_Q2_K
		11: "Q3_K_S",    // MOSTLY_Q3_K_S
		12: "Q3_K_M",    // MOSTLY_Q3_K_M
		13: "Q3_K_L",    // MOSTLY_Q3_K_L
		14: "Q4_K_S",    // MOSTLY_Q4_K_S
		15: "Q4_K_M",    // MOSTLY_Q4_K_M
		16: "Q5_K_S",    // MOSTLY_Q5_K_S
		17: "Q5_K_M",    // MOSTLY_Q5_K_M
		18: "Q6_K",      // MOSTLY_Q6_K
		19: "Q1_K",      // MOSTLY_Q1_K (已废弃)
		20: "Q4_K_S",    // MOSTLY_Q4_K_S (重复)
		21: "Q3_K_S_XL", // MOSTLY_Q3_K_S_XL
		22: "Q2_K_XL",   // MOSTLY_Q2_K_XL
		23: "IQ2_XXS",   // MOSTLY_IQ2_XXS
		24: "IQ2_XS",    // MOSTLY_IQ2_XS
		25: "IQ3_XS",    // MOSTLY_IQ3_XS
		26: "IQ3_XXS",   // MOSTLY_IQ3_XXS
		27: "IQ1_S",     // MOSTLY_IQ1_S
		28: "IQ4_NL",    // MOSTLY_IQ4_NL
		29: "IQ3_S",     // MOSTLY_IQ3_S
		30: "IQ3_M",     // MOSTLY_IQ3_M
		31: "IQ2_S",     // MOSTLY_IQ2_S
		32: "IQ2_M",     // MOSTLY_IQ2_M
		33: "IQ4_XS",    // MOSTLY_IQ4_XS
		34: "IQ1_M",     // MOSTLY_IQ1_M
		35: "BF16",      // MOSTLY_BF16
		36: "Q4_0_4_4",  // MOSTLY_Q4_0_4_4 (已废弃)
		37: "Q4_0_4_8",  // MOSTLY_Q4_0_4_8 (已废弃)
		38: "Q4_0_8_8",  // MOSTLY_Q4_0_8_8 (已废弃)
		39: "TQ1_0",     // MOSTLY_TQ1_0
		40: "TQ2_0",     // MOSTLY_TQ2_0
		41: "MXFP4",     // MOSTLY_MXFP4
	}

	if name, ok := typeMap[m.FileType]; ok {
		return name
	}

	// 对于未知类型，返回通用格式
	return fmt.Sprintf("Type_%d", m.FileType)
}

// GetParametersInBillions returns the parameter count in billions (B)
func (m *Metadata) GetParametersInBillions() float64 {
	if m.Parameters > 0 {
		return m.Parameters
	}
	// Estimate from layer count and embedding size if not directly specified
	if m.BlockSize > 0 && m.EmbeddingLength > 0 {
		// Rough estimation: 12 * n_layers * d_model^2 (for standard transformer)
		params := 12.0 * float64(m.BlockSize) * float64(m.EmbeddingLength*m.EmbeddingLength) / 1e9
		return params
	}
	return 0
}

// GetFileSizeString returns a human-readable file size string
func (m *Metadata) GetFileSizeString() string {
	return formatBytes(m.FileSize)
}

// GetModelSizeString returns a human-readable model size string
func (m *Metadata) GetModelSizeString() string {
	return formatBytes(m.ModelSize)
}

// GetBitsPerWeightString returns the bits per weight as a formatted string
func (m *Metadata) GetBitsPerWeightString() string {
	if m.BitsPerWeight > 0 {
		return fmt.Sprintf("%.2f bpw", m.BitsPerWeight)
	}
	return "N/A"
}

// GetFullDescription returns a comprehensive description of the model
func (m *Metadata) GetFullDescription() string {
	desc := m.Name
	if m.Architecture != "" {
		desc += " (" + m.Architecture + ")"
	}
	if m.Quantization != "" {
		desc += " - " + m.Quantization
	}
	return desc
}

// formatBytes converts a byte count to a human-readable string
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsChatModel returns true if this appears to be a chat/instruct model
// Heuristic: checks model name for common chat model suffixes
func (m *Metadata) IsChatModel() bool {
	name := m.Name
	chatSuffixes := []string{
		"-chat", "-instruct", "-sft", "-lora", "-adapter",
		"chat", "instruct", "sft", "conversation", "dialogue",
	}
	for _, suffix := range chatSuffixes {
		if contains(name, suffix) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	// Simple case-insensitive contains
	sLower := toLower(s)
	substrLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (simplified, ASCII only)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
