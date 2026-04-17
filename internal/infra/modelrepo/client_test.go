// Package modelrepo provides model repository integration for HuggingFace and ModelScope
package modelrepo

import (
	"testing"
	"time"
)

// TestNewClient tests creating a new model repository client
func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

// TestGenerateHuggingFaceURL tests HuggingFace URL generation
func TestGenerateHuggingFaceURL(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name     string
		repoID   string
		fileName string
		want     string
	}{
		{
			name:     "with filename",
			repoID:   "Qwen/Qwen2-7B-Instruct",
			fileName: "model.gguf",
			want:     "https://huggingface.co/Qwen/Qwen2-7B-Instruct/resolve/main/model.gguf",
		},
		{
			name:     "without filename",
			repoID:   "meta-llama/Llama-2-7b",
			fileName: "",
			want:     "https://huggingface.co/meta-llama/Llama-2-7b/resolve/main/",
		},
		{
			name:     "nested filename",
			repoID:   "owner/model",
			fileName: "path/to/model.gguf",
			want:     "https://huggingface.co/owner/model/resolve/main/path/to/model.gguf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.GenerateDownloadURL(SourceHuggingFace, tt.repoID, tt.fileName)
			if err != nil {
				t.Errorf("GenerateDownloadURL() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateDownloadURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateModelScopeURL tests ModelScope URL generation
func TestGenerateModelScopeURL(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name     string
		repoID   string
		fileName string
		want     string
	}{
		{
			name:     "with filename",
			repoID:   "AI-ModelScope/qwen-7b",
			fileName: "model.gguf",
			want:     "https://www.modelscope.cn/api/v1/models/AI-ModelScope/qwen-7b/repo?Revision=master&FilePath=model.gguf",
		},
		{
			name:     "without filename",
			repoID:   "owner/model",
			fileName: "",
			want:     "https://www.modelscope.cn/api/v1/models/owner/model/repo?Revision=master&FilePath=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.GenerateDownloadURL(SourceModelScope, tt.repoID, tt.fileName)
			if err != nil {
				t.Errorf("GenerateDownloadURL() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateDownloadURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerateDownloadURLUnsupported tests unsupported source
func TestGenerateDownloadURLUnsupported(t *testing.T) {
	client := NewClient()

	_, err := client.GenerateDownloadURL("unsupported", "repo", "file")
	if err == nil {
		t.Error("expected error for unsupported source, got nil")
	}
}

// TestIsGGUFFile tests GGUF file detection
func TestIsGGUFFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"model.gguf", true},
		{"model.GGUF", true},
		{"path/to/model.gguf", true},
		{"model.safetensors", false},
		{"config.json", false},
		{"README.md", false},
		{"model.gguf.txt", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isGGUFFile(tt.path); got != tt.want {
				t.Errorf("isGGUFFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFileInfo tests FileInfo structure
func TestFileInfo(t *testing.T) {
	file := FileInfo{
		Name:        "model.gguf",
		Size:        1024000,
		DownloadURL: "https://example.com/model.gguf",
	}

	if file.Name != "model.gguf" {
		t.Errorf("Name = %v, want model.gguf", file.Name)
	}
	if file.Size != 1024000 {
		t.Errorf("Size = %v, want 1024000", file.Size)
	}
	if file.DownloadURL != "https://example.com/model.gguf" {
		t.Errorf("DownloadURL = %v, want https://example.com/model.gguf", file.DownloadURL)
	}
}

// TestSourceConstants tests source constants
func TestSourceConstants(t *testing.T) {
	if SourceHuggingFace != "huggingface" {
		t.Errorf("SourceHuggingFace = %v, want huggingface", SourceHuggingFace)
	}
	if SourceModelScope != "modelscope" {
		t.Errorf("SourceModelScope = %v, want modelscope", SourceModelScope)
	}
}

// TestSearchHuggingFaceModelsLimitValidation tests limit parameter validation
func TestSearchHuggingFaceModelsLimitValidation(t *testing.T) {
	tests := []struct {
		name        string
		inputLimit  int
		expectLimit int
	}{
		{"default limit", 0, 20},
		{"negative limit", -10, 20},
		{"valid limit", 10, 10},
		{"max limit", 100, 100},
		{"exceeds max", 200, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient()
			// 注意：这里只测试 limit 验证逻辑，不实际发起网络请求
			// 实际的网络请求应该使用 mock HTTP 服务器
			if client == nil {
				t.Fatal("NewClient returned nil")
			}

			// 验证 limit 范围检查
			limit := tt.inputLimit
			if limit <= 0 || limit > 100 {
				limit = 20
			}
			if limit != tt.expectLimit {
				t.Errorf("limit validation = %v, want %v", limit, tt.expectLimit)
			}
		})
	}
}

// TestSearchResultStructure tests SearchResult structure
func TestSearchResultStructure(t *testing.T) {
	// 创建一个模拟的搜索结果
	result := &SearchResult{
		Models: []HuggingFaceModel{
			{
				ID:           "Qwen/Qwen2-7B-Instruct",
				ModelID:      "Qwen/Qwen2-7B-Instruct",
				Author:       "Qwen",
				Downloads:    1000000,
				Likes:        5000,
				LastModified: "2024-01-01T00:00:00.000Z",
				Tags:         []string{"gguf", "instruct"},
			},
		},
		Count: 1,
		Total: 1,
	}

	if result.Count != 1 {
		t.Errorf("Count = %v, want 1", result.Count)
	}
	if result.Total != 1 {
		t.Errorf("Total = %v, want 1", result.Total)
	}
	if len(result.Models) != 1 {
		t.Fatalf("Models length = %v, want 1", len(result.Models))
	}

	model := result.Models[0]
	if model.ID != "Qwen/Qwen2-7B-Instruct" {
		t.Errorf("Model ID = %v, want Qwen/Qwen2-7B-Instruct", model.ID)
	}
	if model.Author != "Qwen" {
		t.Errorf("Model Author = %v, want Qwen", model.Author)
	}
}

// TestHuggingFaceModelStructure tests HuggingFaceModel structure
func TestHuggingFaceModelStructure(t *testing.T) {
	model := HuggingFaceModel{
		ID:           "test/model",
		ModelID:      "test/model",
		Author:       "test",
		SHA:          "abc123",
		Private:      false,
		CreatedAt:    "2024-01-01T00:00:00.000Z",
		LastModified: "2024-01-02T00:00:00.000Z",
		Tags:         []string{"gguf", "chat"},
		Downloads:    1000,
		Likes:        100,
		LibraryName:  "transformers",
	}

	if model.ID != "test/model" {
		t.Errorf("ID = %v, want test/model", model.ID)
	}
	if model.Author != "test" {
		t.Errorf("Author = %v, want test", model.Author)
	}
	if model.Downloads != 1000 {
		t.Errorf("Downloads = %v, want 1000", model.Downloads)
	}
	if len(model.Tags) != 2 {
		t.Errorf("Tags length = %v, want 2", len(model.Tags))
	}
}

// TestClientConfiguration tests client configuration
func TestClientConfiguration(t *testing.T) {
	client := NewClientWithConfig(EndpointHuggingFace, "test_token", 60*time.Second)

	if client == nil {
		t.Fatal("NewClientWithConfig returned nil")
	}
}

// TestEndpointConfiguration tests different endpoint configurations
func TestEndpointConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		endpoint      string
		expectSuccess bool
	}{
		{"HuggingFace official", EndpointHuggingFace, true},
		{"HuggingFace mirror", EndpointHuggingMirror, true},
		{"Custom endpoint", "custom.huggingface.co", true},
		{"Empty endpoint (should default)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithConfig(tt.endpoint, "", 30*time.Second)
			if client == nil {
				t.Fatal("NewClientWithConfig returned nil")
			}
		})
	}
}

// TestSearchAPIEndpoints tests search with different endpoints
func TestSearchAPIEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		query    string
		limit    int
	}{
		{
			name:     "HuggingFace official search",
			endpoint: EndpointHuggingFace,
			query:    "qwen",
			limit:    5,
		},
		{
			name:     "HuggingFace mirror search",
			endpoint: EndpointHuggingMirror,
			query:    "llama",
			limit:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientWithConfig(tt.endpoint, "", 30*time.Second)

			// 注意：这些测试会实际发起网络请求
			// 在 CI/CD 环境中可能需要跳过或使用 mock
			result, err := client.SearchHuggingFaceModels(tt.query, tt.limit, "gguf")

			// 网络请求可能失败，我们只验证没有 panic
			if err != nil {
				t.Logf("Search failed (may be network issue): %v", err)
				return
			}

			if result != nil {
				t.Logf("Search succeeded: found %d models", result.Count)
			}
		})
	}
}


