package gguf

import (
	"errors"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"os"
	"syscall"
)

// Reader handles reading GGUF files and extracting metadata
type GGUFReader struct {
	path   string
	file   *os.File
	reader *Reader
	data   []byte // Mapped file data
	mapped bool   // Whether memory mapping is used
}

// NewGGUFReader creates a new GGUF file reader
func NewGGUFReader(path string) (*GGUFReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open GGUF file: %w", err)
	}

	return &GGUFReader{
		path:   path,
		file:   file,
		reader: NewReader(file),
		mapped: false,
	}, nil
}

// ReadMetadata reads and parses the metadata from a GGUF file
// This is the main entry point for GGUF metadata extraction
// Uses gguf-parser-go library for reliable parsing
func ReadMetadata(path string) (*Metadata, error) {
	// 使用新的 Parser（基于 gguf-parser-go）
	parser, err := NewParser(path)
	if err != nil {
		return nil, err
	}
	defer utils.CloseQuietly(parser)

	return parser.GetMetadata()
}

// memoryMap attempts to memory map the file for efficient reading
func (r *GGUFReader) memoryMap() error {
	// Get file info
	info, err := r.file.Stat()
	if err != nil {
		return err
	}

	// Don't map very large files, just map the metadata portion
	size := info.Size()
	if size > MaxMetadataSize {
		size = MaxMetadataSize
	}

	// Memory map the file
	data, err := syscall.Mmap(int(r.file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return err
	}

	r.data = data
	r.mapped = true

	// Create a new reader from the mapped data
	r.reader = NewReader(NewSeekReader(data))
	return nil
}

// Close closes the GGUF reader and releases resources
func (r *GGUFReader) Close() error {
	var err error

	if r.mapped && r.data != nil {
		if e := syscall.Munmap(r.data); e != nil {
			err = e
		}
		r.data = nil
		r.mapped = false
	}

	if r.file != nil {
		if e := r.file.Close(); e != nil && err == nil {
			err = e
		}
		r.file = nil
	}

	return err
}

// readValueOrSkip reads a value or skips it if it's a large array
func (r *Reader) readValueOrSkip(key string, vt ValueType) (interface{}, error) {
	// Special handling for large arrays that we don't need to fully load
	if shouldSkipValue(key) {
		return r.skipValue(vt)
	}
	return r.ReadValue(vt)
}

// shouldSkipValue returns true if a value should be skipped (not fully loaded)
func shouldSkipValue(key string) bool {
	// Large arrays that we only need metadata for
	skipKeys := []string{
		"tokenizer.ggml.tokens",
		"tokenizer.ggml.scores",
		"tokenizer.ggml.token_type",
		"tokenizer.ggml.merges",
		"tokenizer.tokens",
		"tokenizer.scores",
		"tokenizer.merges",
	}

	for _, skipKey := range skipKeys {
		if key == skipKey {
			return true
		}
	}

	return false
}

// setField sets a metadata field based on the key and value
func (m *Metadata) setField(key string, value interface{}, vt ValueType) error {
	switch key {
	// Basic information
	case "general.name":
		if s, ok := value.(string); ok {
			m.Name = s
		}
	case "general.architecture":
		if s, ok := value.(string); ok {
			m.Architecture = s
		}
	case "general.file_type":
		if u, ok := value.(uint32); ok {
			m.FileType = u
		} else if i, ok := value.(int64); ok {
			m.FileType = uint32(i)
		}

	// Model dimensions
	case "llama.context_length":
		if u, ok := value.(uint32); ok {
			m.ContextLength = int(u)
		} else if i, ok := value.(int64); ok {
			m.ContextLength = int(i)
		}
	case "llama.embedding_length":
		if u, ok := value.(uint32); ok {
			m.EmbeddingLength = int(u)
		} else if i, ok := value.(int64); ok {
			m.EmbeddingLength = int(i)
		}
	case "llama.block_count":
		if u, ok := value.(uint32); ok {
			m.BlockSize = int(u)
		} else if i, ok := value.(int64); ok {
			m.BlockSize = int(i)
		}
	case "llama.feed_forward_length":
		if u, ok := value.(uint32); ok {
			m.FeedForwardLength = int(u)
		} else if i, ok := value.(int64); ok {
			m.FeedForwardLength = int(i)
		}
	case "llama.attention.head_count":
		if u, ok := value.(uint32); ok {
			m.HeadCount = int(u)
		} else if i, ok := value.(int64); ok {
			m.HeadCount = int(i)
		}
	case "llama.attention.head_count_kv":
		if u, ok := value.(uint32); ok {
			m.HeadCountKV = int(u)
		} else if i, ok := value.(int64); ok {
			m.HeadCountKV = int(i)
		}
	case "llama.attention.layer_norm_rms_epsilon":
		if f, ok := value.(float32); ok {
			m.LayerNormRMS_EPS = float64(f)
		} else if f64, ok := value.(float64); ok {
			m.LayerNormRMS_EPS = f64
		}

	// Rope settings
	case "llama.rope.dimension_count":
		if u, ok := value.(uint32); ok {
			m.RopeDim = int(u)
		} else if i, ok := value.(int64); ok {
			m.RopeDim = int(i)
		}
	case "llama.rope.freq_base":
		if f, ok := value.(float32); ok {
			m.RopeFreqBase = float64(f)
		} else if f64, ok := value.(float64); ok {
			m.RopeFreqBase = f64
		}
	case "llama.rope.freq_scale":
		if f, ok := value.(float32); ok {
			m.RopeFreqScale = float64(f)
		} else if f64, ok := value.(float64); ok {
			m.RopeFreqScale = f64
		}

	// Quantization
	case "general.quantization_version":
		if u, ok := value.(uint32); ok {
			m.QuantizationVersion = u
		} else if i, ok := value.(int64); ok {
			m.QuantizationVersion = uint32(i)
		}

	// Tokenizer
	case "tokenizer.ggml.model":
		if s, ok := value.(string); ok {
			m.TokenizerModel = s
		}
	case "tokenizer.ggml.tokens":
		// Array is skipped, but we can get count if needed
		if arr, ok := value.(*Array); ok {
			m.TokenCount = int(arr.Len)
		}
	case "tokenizer.ggml.bos_token_id":
		if u, ok := value.(int32); ok {
			m.BosTokenID = int(u)
		} else if i, ok := value.(int64); ok {
			m.BosTokenID = int(i)
		}
	case "tokenizer.ggml.eos_token_id":
		if u, ok := value.(int32); ok {
			m.EosTokenID = int(u)
		} else if i, ok := value.(int64); ok {
			m.EosTokenID = int(i)
		}
	case "tokenizer.ggml.padding_token_id":
		if u, ok := value.(int32); ok {
			m.PadTokenID = int(u)
		} else if i, ok := value.(int64); ok {
			m.PadTokenID = int(i)
		}
	case "tokenizer.ggml.unknown_token_id":
		if u, ok := value.(int32); ok {
			m.UncTokenID = int(u)
		} else if i, ok := value.(int64); ok {
			m.UncTokenID = int(i)
		}
	case "tokenizer.ggml.pre":
		if s, ok := value.(string); ok {
			m.PreToken = s
		}
	case "tokenizer.ggml.post":
		if s, ok := value.(string); ok {
			m.PostToken = s
		}

	// Special tokens (alternative names)
	case "tokenizer.bos_token_id":
		if u, ok := value.(int32); ok {
			m.BosTokenID = int(u)
		} else if i, ok := value.(int64); ok {
			m.BosTokenID = int(i)
		}
	case "tokenizer.eos_token_id":
		if u, ok := value.(int32); ok {
			m.EosTokenID = int(u)
		} else if i, ok := value.(int64); ok {
			m.EosTokenID = int(i)
		}

	default:
		// Store unknown keys in Extra map
		m.Extra[key] = value
	}

	return nil
}

// SeekReader wraps a byte slice to implement io.ReadSeeker
type SeekReader struct {
	data []byte
	pos  int64
}

// NewSeekReader creates a new SeekReader from a byte slice
func NewSeekReader(data []byte) *SeekReader {
	return &SeekReader{
		data: data,
		pos:  0,
	}
}

// Read reads data into p
func (r *SeekReader) Read(p []byte) (int, error) {
	if r.pos >= int64(len(r.data)) {
		return 0, errors.New("EOF")
	}

	n := copy(p, r.data[r.pos:])
	r.pos += int64(n)

	if r.pos >= int64(len(r.data)) {
		return n, errors.New("EOF")
	}

	return n, nil
}

// Seek sets the offset for the next Read
func (r *SeekReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64

	switch whence {
	case 0: // Seek from start
		newPos = offset
	case 1: // Seek from current
		newPos = r.pos + offset
	case 2: // Seek from end
		newPos = int64(len(r.data)) + offset
	default:
		return 0, errors.New("invalid whence")
	}

	if newPos < 0 {
		return 0, errors.New("negative position")
	}

	if newPos > int64(len(r.data)) {
		return 0, errors.New("position beyond data")
	}

	r.pos = newPos
	return newPos, nil
}
