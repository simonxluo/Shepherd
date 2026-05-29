package gguf

import (
	"fmt"

	ggufparser "github.com/gpustack/gguf-parser-go"
)

// Parser wraps gguf-parser-go to parse GGUF model files.
type Parser struct {
	path string
	file *ggufparser.GGUFFile
}

// NewParser creates a new GGUF parser for the given file path.
func NewParser(path string) (*Parser, error) {
	file, err := ggufparser.ParseGGUFFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GGUF file: %w", err)
	}

	return &Parser{
		path: path,
		file: file,
	}, nil
}

// Close closes the parser and releases resources.
func (p *Parser) Close() error {
	// gguf-parser-go manages resources automatically
	return nil
}

// GetMetadata returns parsed metadata from the GGUF file.
func (p *Parser) GetMetadata() (*Metadata, error) {
	if p.file == nil {
		return nil, fmt.Errorf("GGUF file not loaded")
	}

	// Call gguf-parser-go's Metadata() method
	gmeta := p.file.Metadata()

	meta := &Metadata{
		Extra: make(map[string]interface{}),
	}

	// Metadata directly from gguf-parser-go

	// Basic info
	meta.Name = gmeta.Name
	meta.Architecture = gmeta.Architecture
	meta.Type = gmeta.Type
	meta.Author = gmeta.Author
	meta.URL = gmeta.URL
	meta.Description = gmeta.Description
	meta.License = gmeta.License

	// Quantization info
	meta.FileType = uint32(gmeta.FileType)
	meta.FileTypeDescriptor = gmeta.FileTypeDescriptor
	meta.QuantizationVersion = gmeta.QuantizationVersion

	// Model parameters
	meta.Parameters = float64(gmeta.Parameters)
	meta.BitsPerWeight = float64(gmeta.BitsPerWeight)

	// File info
	meta.Alignment = gmeta.Alignment
	meta.LittleEndian = gmeta.LittleEndian
	meta.FileSize = uint64(gmeta.FileSize)
	meta.ModelSize = uint64(gmeta.Size)

	// Read architecture-specific fields from Header.MetadataKV
	metadataKV := p.file.Header.MetadataKV

	// Helper: get value from KV map
	getKV := func(key string) (ggufparser.GGUFMetadataKV, bool) {
		kvs, found := metadataKV.Index([]string{key})
		if found > 0 {
			return kvs[key], true
		}
		return ggufparser.GGUFMetadataKV{}, false
	}

	// Helper: safely get integer value (handles Uint32/Uint64/Int32/Int64)
	getIntValue := func(kv ggufparser.GGUFMetadataKV) int {
		switch kv.ValueType {
		case ggufparser.GGUFMetadataValueTypeUint32:
			return int(kv.ValueUint32())
		case ggufparser.GGUFMetadataValueTypeUint64:
			return int(kv.ValueUint64())
		case ggufparser.GGUFMetadataValueTypeInt32:
			return int(kv.ValueInt32())
		case ggufparser.GGUFMetadataValueTypeInt64:
			return int(kv.ValueInt64())
		default:
			return 0
		}
	}

	// Read architecture-specific fields
	// Different architectures use different prefixes (e.g., llama.context_length, qwen3next.context_length)
	// Build key names dynamically using the architecture name

	arch := meta.Architecture
	if arch == "" {
		// Try to get from general.architecture
		if kv, ok := getKV("general.architecture"); ok {
			arch = kv.ValueString()
		}
	}

	// Define common field names (without prefix)
	commonFields := []struct {
		key         string
		setter      func(int)
		floatSetter func(float64)
	}{
		{"context_length", func(v int) { meta.ContextLength = v }, nil},
		{"embedding_length", func(v int) { meta.EmbeddingLength = v }, nil},
		{"block_count", func(v int) { meta.BlockCount = v }, nil},
		{"feed_forward_length", func(v int) { meta.FeedForwardLength = v }, nil},
		{"attention.head_count", func(v int) { meta.HeadCount = v }, nil},
		{"attention.head_count_kv", func(v int) { meta.HeadCountKV = v }, nil},
		{"rope.dimension_count", func(v int) { meta.RopeDim = v }, nil},
		{"attention.layer_norm_rms_epsilon", nil, func(v float64) { meta.LayerNormRMS_EPS = v }},
		{"rope.freq_base", nil, func(v float64) { meta.RopeFreqBase = v }},
		{"rope.freq_scale", nil, func(v float64) { meta.RopeFreqScale = v }},
	}

	// Try reading with architecture prefix
	for _, field := range commonFields {
		if arch != "" {
			// Try {architecture}.{field} format
			archKey := fmt.Sprintf("%s.%s", arch, field.key)
			if kv, ok := getKV(archKey); ok {
				if field.setter != nil && (kv.ValueType == ggufparser.GGUFMetadataValueTypeUint32 ||
					kv.ValueType == ggufparser.GGUFMetadataValueTypeUint64 ||
					kv.ValueType == ggufparser.GGUFMetadataValueTypeInt32 ||
					kv.ValueType == ggufparser.GGUFMetadataValueTypeInt64) {
					field.setter(getIntValue(kv))
				} else if field.floatSetter != nil && kv.ValueType == ggufparser.GGUFMetadataValueTypeFloat32 {
					field.floatSetter(float64(kv.ValueFloat32()))
				}
				continue
			}
		}

		// Fall back to llama prefix (backward compatibility)
		llamaKey := fmt.Sprintf("llama.%s", field.key)
		if kv, ok := getKV(llamaKey); ok {
			if field.setter != nil {
				field.setter(getIntValue(kv))
			} else if field.floatSetter != nil {
				field.floatSetter(float64(kv.ValueFloat32()))
			}
		}
	}

	// Tokenizer info
	if kv, ok := getKV("tokenizer.ggml.model"); ok {
		meta.TokenizerModel = kv.ValueString()
	}

	// Tokenizer Token IDs
	if kv, ok := getKV("tokenizer.ggml.bos_token_id"); ok {
		meta.BosTokenID = getIntValue(kv)
	}
	if kv, ok := getKV("tokenizer.ggml.eos_token_id"); ok {
		meta.EosTokenID = getIntValue(kv)
	}
	if kv, ok := getKV("tokenizer.ggml.padding_token_id"); ok {
		meta.PadTokenID = getIntValue(kv)
	}
	if kv, ok := getKV("tokenizer.ggml.unknown_token_id"); ok {
		meta.UncTokenID = getIntValue(kv)
	}
	if kv, ok := getKV("tokenizer.ggml.pre"); ok {
		meta.PreToken = kv.ValueString()
	}
	if kv, ok := getKV("tokenizer.ggml.post"); ok {
		meta.PostToken = kv.ValueString()
	}

	// Token vocabulary size
	if kv, ok := getKV("tokenizer.ggml.token_count"); ok {
		meta.TokenCount = getIntValue(kv)
	} else if kv, ok := getKV("tokenizer.token_list"); ok {
		// token_list is an array; use its length
		if arr := kv.ValueArray(); arr.Len > 0 {
			meta.TokenCount = int(arr.Len)
		}
	}

	// Fallback Tokenizer Token IDs (some models use different key names)
	if meta.BosTokenID == 0 {
		if kv, ok := getKV("tokenizer.bos_token_id"); ok {
			meta.BosTokenID = getIntValue(kv)
		}
	}
	if meta.EosTokenID == 0 {
		if kv, ok := getKV("tokenizer.eos_token_id"); ok {
			meta.EosTokenID = getIntValue(kv)
		}
	}

	// Chat Template - used for capability detection
	// Read tokenizer.chat_template (Jinja template for conversation formatting)
	if kv, ok := getKV("tokenizer.chat_template"); ok {
		meta.ChatTemplate = kv.ValueString()
	}

	// Architecture-specific info
	// Read pooling_type and other architecture-level metadata
	archInfo := p.file.Architecture()
	meta.PoolingType = archInfo.PoolingType

	// Compute quantization string
	meta.Quantization = meta.GetQuantizationString()

	return meta, nil
}

func (p *Parser) GetArchitecture() string {
	if p.file != nil {
		meta := p.file.Metadata()
		return meta.Architecture
	}
	return ""
}
