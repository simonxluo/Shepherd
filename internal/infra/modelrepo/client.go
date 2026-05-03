// Package modelrepo provides model repository integration for HuggingFace and ModelScope.
// It integrates two SDKs for advanced functionality:
//   - github.com/gomlx/go-huggingface/hub: For basic Hub operations and file downloads
//   - github.com/bodaay/HuggingFaceModelDownloader: For advanced downloads (resumable/multipart)
package modelrepo

import (
	"encoding/json"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	hfhub "github.com/gomlx/go-huggingface/hub"
)

// Source represents the model repository source
type Source string

const (
	SourceHuggingFace Source = "huggingface"
	SourceModelScope  Source = "modelscope"
)

// Endpoint URLs
const (
	EndpointHuggingFace = "huggingface.co"
)

// DownloadMode determines which download method to use
type DownloadMode string

const (
	DownloadModeBasic    DownloadMode = "basic"    // Use go-huggingface/hub (simple, reliable)
	DownloadModeAdvanced DownloadMode = "advanced" // Use bodaay/HuggingFaceModelDownloader (multipart, resumable)
)

// Client is a model repository client
type Client struct {
	httpClient *http.Client
	hfToken    string
	endpoint   string
	cacheDir   string
}

// NewClientWithConfig creates a new model repository client with full configuration
func NewClientWithConfig(endpoint, token string, timeout time.Duration) *Client {
	if endpoint == "" {
		endpoint = EndpointHuggingFace
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Support HTTP proxy
	if proxyURL := os.Getenv("HTTPS_PROXY"); proxyURL != "" {
		if parsedURL, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(parsedURL)
		}
	} else if proxyURL := os.Getenv("HTTP_PROXY"); proxyURL != "" {
		if parsedURL, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(parsedURL)
		}
	}

	cacheDir := hfhub.DefaultCacheDir()
	if envDir := os.Getenv("HF_HOME"); envDir != "" {
		cacheDir = filepath.Join(envDir, "hub")
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		hfToken:  token,
		endpoint: endpoint,
		cacheDir: cacheDir,
	}
}

// GetAvailableEndpoints returns available HuggingFace endpoints
func GetAvailableEndpoints() map[string]string {
	return map[string]string{
		"huggingface.co": "HuggingFace 官方",
		"hf-mirror.com":  "HuggingFace 镜像",
	}
}

// FileInfo represents information about a model file
type FileInfo struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
}

// GenerateDownloadURL generates a download URL from repository information
func (c *Client) GenerateDownloadURL(source Source, repoID, fileName string) (string, error) {
	switch source {
	case SourceHuggingFace:
		return c.generateHuggingFaceURL(repoID, fileName)
	case SourceModelScope:
		return c.generateModelScopeURL(repoID, fileName)
	default:
		return "", fmt.Errorf("unsupported source: %s", source)
	}
}

// generateHuggingFaceURL generates a HuggingFace download URL
func (c *Client) generateHuggingFaceURL(repoID, fileName string) (string, error) {
	base := fmt.Sprintf("https://%s/%s/resolve/main/", c.endpoint, repoID)
	if fileName != "" {
		base += fileName
	}
	return base, nil
}

// generateModelScopeURL generates a ModelScope download URL
func (c *Client) generateModelScopeURL(repoID, fileName string) (string, error) {
	base := fmt.Sprintf("https://www.modelscope.cn/api/v1/models/%s/repo?Revision=master&FilePath=", repoID)
	if fileName != "" {
		base += fileName
	}
	return base, nil
}

// ListGGUFFiles lists GGUF files in a HuggingFace repository
func (c *Client) ListGGUFFiles(repoID string) ([]FileInfo, error) {
	// 使用 tree/main 端点获取带文件大小的文件列表
	apiURL := fmt.Sprintf("https://%s/api/models/%s/tree/main", c.endpoint, repoID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	if c.hfToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.hfToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch model info: %s", resp.Status)
	}

	// HuggingFace tree API 返回文件列表，每个元素包含 path, type, size
	var treeFiles []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int64  `json:"size"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&treeFiles); err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, item := range treeFiles {
		if item.Type == "file" && isGGUFFile(item.Path) {
			files = append(files, FileInfo{
				Name:        item.Path,
				Size:        item.Size,
				DownloadURL: fmt.Sprintf("https://%s/%s/resolve/main/%s", c.endpoint, repoID, item.Path),
			})
		}
	}

	return files, nil
}

// isGGUFFile checks if a file is a GGUF model file
func isGGUFFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".gguf")
}

type HuggingFaceModel struct {
	ID           string   `json:"id"`
	ModelID      string   `json:"modelId"`
	Author       string   `json:"author"`
	SHA          string   `json:"sha"`
	Private      bool     `json:"private"`
	CreatedAt    string   `json:"createdAt"`
	LastModified string   `json:"lastModified"`
	Tags         []string `json:"tags"`
	Downloads    int      `json:"downloads"`
	Likes        int      `json:"likes"`
	LibraryName  string   `json:"library_name"`
}

// SearchResult represents the search response from HuggingFace
type SearchResult struct {
	Models []HuggingFaceModel `json:"models"`
	Count  int                `json:"count"`
	Total  int                `json:"total"`
}

// SearchHuggingFaceModels searches for models on HuggingFace
// formatFilter can be used to filter by format (e.g., "gguf", "safetensors", "onnx")
func (c *Client) SearchHuggingFaceModels(query string, limit int, formatFilter string) (*SearchResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	apiURL := fmt.Sprintf("https://%s/api/models?search=%s&limit=%d",
		c.endpoint, url.QueryEscape(query), limit)

	// Add filter parameter for format
	if formatFilter != "" && formatFilter != "all" {
		apiURL = fmt.Sprintf("%s&filter=%s", apiURL, url.QueryEscape(formatFilter))
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	if c.hfToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.hfToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to search models: %s", resp.Status)
	}

	var models []HuggingFaceModel
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, err
	}

	result := &SearchResult{
		Models: models,
		Count:  len(models),
		Total:  len(models),
	}

	return result, nil
}
