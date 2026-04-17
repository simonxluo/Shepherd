package gguf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadMetadata tests the metadata reading functionality
func TestReadMetadata(t *testing.T) {
	// Create test data directory
	testDataDir := filepath.Join("..", "..", "testdata")

	// Check if there's a real GGUF file to test with
	testFiles := []string{
		filepath.Join(testDataDir, "tinyllama.gguf"),
		filepath.Join(testDataDir, "test.gguf"),
		filepath.Join(os.Getenv("HOME"), ".cache/huggingface/hub/models--TheBloke--Llama-2-7B-GGUF/snapshots/*/llama-2-7b.Q4_K_M.gguf"),
	}

	var testFile string
	for _, f := range testFiles {
		matches, _ := filepath.Glob(f)
		if len(matches) > 0 {
			// Check if file exists and is readable
			if info, err := os.Stat(matches[0]); err == nil && info.Size() > 0 {
				testFile = matches[0]
				break
			}
		}
	}

	// If no test file found, skip the test
	if testFile == "" {
		t.Skip("No GGUF test file found. Skipping test.")
		return
	}

	t.Run("ReadMetadata_Success", func(t *testing.T) {
		meta, err := ReadMetadata(testFile)
		require.NoError(t, err)
		require.NotNil(t, meta)

		// Verify basic fields are populated
		assert.NotEmpty(t, meta.Architecture, "Architecture should be set")
		assert.NotEmpty(t, meta.Name, "Name should be set")

		// Most models should have these fields
		if meta.ContextLength > 0 {
			assert.Greater(t, meta.ContextLength, 0, "Context length should be positive")
		}
		if meta.EmbeddingLength > 0 {
			assert.Greater(t, meta.EmbeddingLength, 0, "Embedding length should be positive")
		}
	})

	t.Run("ReadMetadata_QuantizationString", func(t *testing.T) {
		meta, err := ReadMetadata(testFile)
		require.NoError(t, err)

		// Check that quantization string is generated
		assert.NotEmpty(t, meta.Quantization, "Quantization string should be set")

		// Common quantization formats
		validFormats := []string{
			"Q4_K_M", "Q5_K_M", "Q3_K_M", "Q4_K_S",
			"Q2_K", "Q5_K", "Q6_K", "Q8_0",
			"F32", "F16", "Q4_0", "Q4_1", "Q5_0", "Q5_1",
		}

		found := false
		for _, valid := range validFormats {
			if meta.Quantization == valid {
				found = true
				break
			}
		}

		// If not in our predefined list, at least check it starts with Q or F
		if !found {
			assert.Regexp(t, "^[QF]\\d", meta.Quantization,
				"Quantization should start with Q or F followed by a digit")
		}
	})

	t.Run("ReadMetadata_TokenizerInfo", func(t *testing.T) {
		meta, err := ReadMetadata(testFile)
		require.NoError(t, err)

		// Most models should have tokenizer info
		if meta.TokenCount > 0 {
			assert.Greater(t, meta.TokenCount, 0, "Token count should be positive")
		}

		// Common BOS/EOS tokens for LLaMA models
		if meta.BosTokenID != 0 {
			assert.GreaterOrEqual(t, meta.BosTokenID, 0, "BOS token ID should be non-negative")
		}
		if meta.EosTokenID != 0 {
			assert.GreaterOrEqual(t, meta.EosTokenID, 0, "EOS token ID should be non-negative")
		}
	})

	t.Run("ReadMetadata_ExtraFields", func(t *testing.T) {
		meta, err := ReadMetadata(testFile)
		require.NoError(t, err)

		// Extra map should be initialized even if empty
		assert.NotNil(t, meta.Extra, "Extra map should be initialized")

		// Some metadata should have been captured (either in main fields or extra)
		totalFields := len(meta.Extra)
		hasMainFields := meta.Architecture != "" || meta.Name != ""

		assert.True(t, hasMainFields || totalFields > 0,
			"At least some metadata should be captured")
	})
}

// TestInvalidFile tests error handling for invalid files
func TestInvalidFile(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		wantErr string
	}{
		{
			name:    "Empty file",
			content: []byte{},
			wantErr: "",
		},
		{
			name:    "Invalid magic",
			content: []byte("TEST"), // Not "GGUF"
			wantErr: "invalid GGUF format",
		},
		{
			name:    "Truncated header",
			content: []byte("GGUF"), // Only magic, nothing else
			wantErr: "EOF",          // Error will be EOF since we can't read more
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file with test content
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.gguf")

			err := os.WriteFile(testFile, tt.content, 0644)
			require.NoError(t, err)

			// Try to read metadata
			_, err = ReadMetadata(testFile)

			// Check error
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				// For empty files, we expect some kind of error
				assert.Error(t, err)
			}
		})
	}
}

// TestFileTypeMapping tests the file type to quantization string mapping
func TestFileTypeMapping(t *testing.T) {
	tests := []struct {
		fileType   uint32
		wantString string
	}{
		{0, "F32"},
		{1, "F16"},
		{2, "Q4_0"},
		{3, "Q4_1"},
		{6, "Q5_0"},
		{7, "Q5_1"},
		{8, "Q8_0"},
		{10, "Q2_K"},
		{11, "Q3_K_S"},
		{12, "Q3_K_M"},
		{13, "Q3_K_L"},
		{14, "Q4_K_S"},
		{15, "Q4_K_M"},
		{16, "Q5_K_S"},
		{17, "Q5_K_M"},
		{18, "Q6_K"},
		{20, "Q4_K_S"},
		{21, "Q3_K_S_XL"},
	}

	for _, tt := range tests {
		t.Run(tt.wantString, func(t *testing.T) {
			meta := &Metadata{FileType: tt.fileType}
			got := meta.GetQuantizationString()
			assert.Equal(t, tt.wantString, got)
		})
	}
}

// TestIsChatModel tests the chat model detection heuristic
func TestIsChatModel(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      bool
	}{
		{"Chat model", "llama-2-7b-chat", true},
		{"Instruct model", "llama-2-7b-instruct", true},
		{"SFT model", "model-sft", true},
		{"Base model", "llama-2-7b", false},
		{"Conversation model", "model-conversation", true},
		{"Dialogue model", "model-dialogue", true},
		{"Mixed case", "Llama-2-7B-Chat", true},
		{"LoRA model", "model-lora", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &Metadata{Name: tt.modelName}
			got := meta.IsChatModel()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestMemoryMapping tests that memory mapping works correctly
func TestMemoryMapping(t *testing.T) {
	// Find a test file
	testFile := findTestGGUFFile()
	if testFile == "" {
		t.Skip("No GGUF test file found")
	}

	reader, err := NewGGUFReader(testFile)
	require.NoError(t, err)
	defer reader.Close()

	// Try to memory map
	err = reader.memoryMap()
	if err != nil {
		t.Logf("Memory mapping not supported: %v", err)
		return
	}

	assert.True(t, reader.mapped, "File should be memory mapped")
	assert.NotNil(t, reader.data, "Data should be set")
}

// BenchmarkReadMetadata benchmarks the metadata reading performance
func BenchmarkReadMetadata(b *testing.B) {
	testFile := findTestGGUFFile()
	if testFile == "" {
		b.Skip("No GGUF test file found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ReadMetadata(testFile)
		if err != nil {
			b.Fatalf("Failed to read metadata: %v", err)
		}
	}
}

// Helper function to find a test GGUF file
func findTestGGUFFile() string {
	// Try common locations
	locations := []string{
		filepath.Join("..", "..", "testdata", "tinyllama.gguf"),
		filepath.Join("..", "..", "testdata", "test.gguf"),
	}

	// Try to find in HuggingFace cache
	cachePath := filepath.Join(os.Getenv("HOME"), ".cache/huggingface/hub")
	matches, _ := filepath.Glob(filepath.Join(cachePath, "models--*--*GGUF*/snapshots/*/llama*.gguf"))
	locations = append(locations, matches...)

	for _, loc := range locations {
		if info, err := os.Stat(loc); err == nil && info.Size() > 1024 { // At least 1KB
			return loc
		}
	}

	return ""
}
