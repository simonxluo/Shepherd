package gguf

import (
	"fmt"
)

var ggufFileTypeMap = map[uint32]string{
	0:  "F32",
	1:  "F16",
	2:  "Q4_0",
	3:  "Q4_1",
	4:  "Q4_2",
	5:  "Q4_3",
	6:  "Q5_0",
	7:  "Q5_1",
	8:  "Q8_0",
	9:  "Q8_1",
	10: "Q2_K",
	11: "Q3_K_S",
	12: "Q3_K_M",
	13: "Q3_K_L",
	14: "Q4_K_S",
	15: "Q4_K_M",
	16: "Q5_K_S",
	17: "Q5_K_M",
	18: "Q6_K",
	19: "Q1_K",
	20: "Q4_K_S",
	21: "Q3_K_S_XL",
	22: "Q2_K_XL",
	23: "IQ2_XXS",
	24: "IQ2_XS",
	25: "IQ3_XS",
	26: "IQ3_XXS",
	27: "IQ1_S",
	28: "IQ4_NL",
	29: "IQ3_S",
	30: "IQ3_M",
	31: "IQ2_S",
	32: "IQ2_M",
	33: "IQ4_XS",
	34: "IQ1_M",
	35: "BF16",
	36: "Q4_0_4_4",
	37: "Q4_0_4_8",
	38: "Q4_0_8_8",
	39: "TQ1_0",
	40: "TQ2_0",
	41: "MXFP4",
}

type Metadata struct {
	Name string

	Architecture string

	Quantization string

	Type string

	Author string

	URL string

	Description string

	License string

	Parameters float64

	ContextLength int

	EmbeddingLength int

	FeedForwardLength int

	BlockSize int

	HeadCount int

	HeadCountKV int

	LayerNormRMS_EPS float64

	TokenCount int

	TokenizerModel string

	BosTokenID int

	EosTokenID int

	PadTokenID int

	UncTokenID int

	RopeDim int

	RopeFreqBase float64

	RopeFreqScale float64

	QuantizationVersion uint32

	FileType uint32

	FileTypeDescriptor string

	Alignment uint32

	LittleEndian bool

	FileSize uint64

	ModelSize uint64

	BitsPerWeight float64

	PreToken string

	PostToken string

	PoolingType uint32

	ChatTemplate string

	Extra map[string]interface{}
}

func (m *Metadata) GetQuantizationString() string {
	if m.FileTypeDescriptor != "" {
		return m.FileTypeDescriptor
	}

	if m.FileType == 0 {
		return "F32"
	}

	if name, ok := ggufFileTypeMap[m.FileType]; ok {
		return name
	}

	return fmt.Sprintf("Type_%d", m.FileType)
}
