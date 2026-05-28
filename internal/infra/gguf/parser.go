package gguf

import (
	"fmt"

	ggufparser "github.com/gpustack/gguf-parser-go"
)

// Parser 使用 gguf-parser-go 库解析 GGUF 文件
type Parser struct {
	path string
	file *ggufparser.GGUFFile
}

// NewParser 创建新的 GGUF 解析器
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

// Close 关闭解析器并释放资源
func (p *Parser) Close() error {
	// gguf-parser-go 会自动管理资源
	return nil
}

// GetMetadata 获取解析后的元数据
func (p *Parser) GetMetadata() (*Metadata, error) {
	if p.file == nil {
		return nil, fmt.Errorf("GGUF file not loaded")
	}

	// 调用 gguf-parser-go 的 Metadata() 方法获取元数据
	gmeta := p.file.Metadata()

	meta := &Metadata{
		Extra: make(map[string]interface{}),
	}

	// 从 gguf-parser-go 直接获取的元数据

	// 基本信息
	meta.Name = gmeta.Name
	meta.Architecture = gmeta.Architecture
	meta.Type = gmeta.Type
	meta.Author = gmeta.Author
	meta.URL = gmeta.URL
	meta.Description = gmeta.Description
	meta.License = gmeta.License

	// 量化信息
	meta.FileType = uint32(gmeta.FileType)
	meta.FileTypeDescriptor = gmeta.FileTypeDescriptor
	meta.QuantizationVersion = gmeta.QuantizationVersion

	// 模型参数
	meta.Parameters = float64(gmeta.Parameters)
	meta.BitsPerWeight = float64(gmeta.BitsPerWeight)

	// 文件信息
	meta.Alignment = gmeta.Alignment
	meta.LittleEndian = gmeta.LittleEndian
	meta.FileSize = uint64(gmeta.FileSize)
	meta.ModelSize = uint64(gmeta.Size)

	// 从 Header.MetadataKV 中读取架构特定字段
	metadataKV := p.file.Header.MetadataKV

	// 辅助函数：从 KV map 中获取值
	getKV := func(key string) (ggufparser.GGUFMetadataKV, bool) {
		kvs, found := metadataKV.Index([]string{key})
		if found > 0 {
			return kvs[key], true
		}
		return ggufparser.GGUFMetadataKV{}, false
	}

	// 辅助函数：安全获取整数类型（处理 Uint32/Uint64/Int32/Int64）
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

	// 读取架构特定字段
	// 不同架构使用不同的前缀，如 llama.context_length, qwen3next.context_length, gpt-oss.context_length
	// 使用架构名称动态构建键名

	arch := meta.Architecture
	if arch == "" {
		// 尝试从 general.architecture 获取
		if kv, ok := getKV("general.architecture"); ok {
			arch = kv.ValueString()
		}
	}

	// 定义要读取的通用字段名（不带前缀）
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

	// 尝试使用架构前缀读取
	for _, field := range commonFields {
		if arch != "" {
			// 尝试 {architecture}.{field} 格式
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

		// 回退到 llama 前缀（兼容旧代码）
		llamaKey := fmt.Sprintf("llama.%s", field.key)
		if kv, ok := getKV(llamaKey); ok {
			if field.setter != nil {
				field.setter(getIntValue(kv))
			} else if field.floatSetter != nil {
				field.floatSetter(float64(kv.ValueFloat32()))
			}
		}
	}

	// Tokenizer 信息
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

	// Token 词表大小
	if kv, ok := getKV("tokenizer.ggml.token_count"); ok {
		meta.TokenCount = getIntValue(kv)
	} else if kv, ok := getKV("tokenizer.token_list"); ok {
		// token_list 是一个数组，使用其长度
		if arr := kv.ValueArray(); arr.Len > 0 {
			meta.TokenCount = int(arr.Len)
		}
	}

	// 备用 Tokenizer Token IDs（某些模型使用不同的键名）
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

	// Chat Template - 用于能力检测
	// 读取 tokenizer.chat_template（Jinja 模板用于对话格式化）
	if kv, ok := getKV("tokenizer.chat_template"); ok {
		meta.ChatTemplate = kv.ValueString()
	}

	// 架构特定信息
	// 读取 pooling_type 等架构级别的元数据
	archInfo := p.file.Architecture()
	meta.PoolingType = archInfo.PoolingType

	// 计算量化字符串
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
