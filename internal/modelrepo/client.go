// Package modelrepo provides model repository integration for HuggingFace and ModelScope.
// It integrates two SDKs for advanced functionality:
//   - github.com/gomlx/go-huggingface/hub: For basic Hub operations and file downloads
//   - github.com/bodaay/HuggingFaceModelDownloader: For advanced downloads (resumable/multipart)
package modelrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/utils"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	hfdownloader "github.com/bodaay/HuggingFaceModelDownloader/pkg/hfdownloader"
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
	EndpointHuggingFace   = "huggingface.co"
	EndpointHuggingMirror = "hf-mirror.com"
)

// DownloadMode determines which download method to use
type DownloadMode string

const (
	DownloadModeBasic    DownloadMode = "basic"    // Use go-huggingface/hub (simple, reliable)
	DownloadModeAdvanced DownloadMode = "advanced" // Use bodaay/HuggingFaceModelDownloader (multipart, resumable)
)

// Client is a model repository client
type Client struct {
	httpClient   *http.Client
	hfToken      string
	endpoint     string
	cacheDir     string
	downloadMode DownloadMode
}

// NewClient creates a new model repository client
func NewClient() *Client {
	return NewClientWithConfig(EndpointHuggingFace, "", 30*time.Second)
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
		hfToken:      token,
		endpoint:     endpoint,
		cacheDir:     cacheDir,
		downloadMode: DownloadModeBasic, // Default to basic mode
	}
}

// SetEndpoint sets the HuggingFace endpoint
func (c *Client) SetEndpoint(endpoint string) {
	if endpoint != "" {
		c.endpoint = endpoint
	}
}

// SetHFToken sets the HuggingFace authentication token
func (c *Client) SetHFToken(token string) {
	c.hfToken = token
}

// SetDownloadMode sets the download mode (basic or advanced)
func (c *Client) SetDownloadMode(mode DownloadMode) {
	c.downloadMode = mode
}

// GetEndpoint returns the current endpoint
func (c *Client) GetEndpoint() string {
	return c.endpoint
}

// GetHFToken returns the current token (masked)
func (c *Client) GetHFToken() string {
	if c.hfToken == "" {
		return ""
	}
	if len(c.hfToken) <= 4 {
		return "***"
	}
	return c.hfToken[:len(c.hfToken)-4] + "****"
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

// Progress represents download progress information
type Progress struct {
	DownloadedBytes int64   `json:"downloaded_bytes"`
	TotalBytes      int64   `json:"total_bytes"`
	Percentage      float64 `json:"percentage"`
	Speed           int64   `json:"speed,omitempty"` // bytes per second
	ETA             int64   `json:"eta,omitempty"`   // seconds remaining
}

// ProgressCallback is called during download to report progress
type ProgressCallback func(progress Progress)

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

// ParseRepoID validates and parses a repository ID
func ParseRepoID(repoID string) (owner, model string, err error) {
	parts := strings.Split(repoID, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo ID format (expected 'owner/model'): %s", repoID)
	}
	return parts[0], parts[1], nil
}

// HuggingFaceModel represents a model from HuggingFace search results
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

// HasGGUFFiles checks if a repository contains GGUF files
func (c *Client) HasGGUFFiles(repoID string) (bool, error) {
	files, err := c.ListGGUFFiles(repoID)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// ============================================
// Advanced features using SDK integrations
// ============================================

// ListModelFiles lists all files in a HuggingFace model repository
func (c *Client) ListModelFiles(repoID string, revision string) ([]FileInfo, error) {
	if revision == "" {
		revision = "main"
	}

	repo := hfhub.New(repoID).
		WithRevision(revision).
		WithEndpoint("https://" + c.endpoint).
		WithAuth(c.hfToken).
		WithCacheDir(c.cacheDir)
	repo.Verbosity = 0

	if err := repo.DownloadInfo(false); err != nil {
		return nil, fmt.Errorf("failed to fetch repo info: %w", err)
	}

	info := repo.Info()
	files := make([]FileInfo, 0, len(info.Siblings))
	for _, sibling := range info.Siblings {
		downloadURL := fmt.Sprintf("https://%s/%s/resolve/%s/%s",
			c.endpoint, repoID, revision, sibling.Name)
		files = append(files, FileInfo{
			Name:        sibling.Name,
			Size:        0, // go-huggingface/hub doesn't provide file size
			DownloadURL: downloadURL,
		})
	}

	return files, nil
}

// DownloadFile downloads a single file from HuggingFace
func (c *Client) DownloadFile(ctx context.Context, repoID, fileName, targetPath string, progress ProgressCallback) error {
	return c.DownloadFileWithRevision(ctx, repoID, "main", fileName, targetPath, progress)
}

// DownloadFileWithRevision downloads a file with specific revision
func (c *Client) DownloadFileWithRevision(ctx context.Context, repoID, revision, fileName, targetPath string, progress ProgressCallback) error {
	if revision == "" {
		revision = "main"
	}

	if c.downloadMode == DownloadModeAdvanced {
		return c.downloadFileAdvanced(ctx, repoID, revision, fileName, targetPath, progress)
	}

	return c.downloadFileBasic(ctx, repoID, revision, fileName, targetPath, progress)
}

// downloadFileBasic uses go-huggingface/hub for simple download
func (c *Client) downloadFileBasic(ctx context.Context, repoID, revision, fileName, targetPath string, progress ProgressCallback) error {
	repo := hfhub.New(repoID).
		WithRevision(revision).
		WithEndpoint("https://" + c.endpoint).
		WithAuth(c.hfToken).
		WithCacheDir(c.cacheDir)
	repo.Verbosity = 0
	repo.MaxParallelDownload = 1

	// Create target directory
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Download file to cache first
	downloadedFiles, err := repo.DownloadFiles(fileName)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Copy from cache to target location
	if len(downloadedFiles) > 0 {
		if progress != nil {
			progress(Progress{
				DownloadedBytes: 100, // Completed
				TotalBytes:      100,
				Percentage:      100,
			})
		}
		return copyFile(downloadedFiles[0], targetPath)
	}

	return fmt.Errorf("no files downloaded")
}

// downloadFileAdvanced uses bodaay/HuggingFaceModelDownloader for advanced features
func (c *Client) downloadFileAdvanced(ctx context.Context, repoID, revision, fileName, targetPath string, progress ProgressCallback) error {
	settings := hfdownloader.Settings{
		CacheDir:           c.cacheDir,
		Concurrency:        8,
		MaxActiveDownloads: 4,
		MultipartThreshold: "32MiB",
		Verify:             "sha256",
		Retries:            4,
		Token:              c.hfToken,
		Endpoint:           "https://" + c.endpoint,
	}

	job := hfdownloader.Job{
		Repo:     repoID,
		Revision: revision,
		Filters:  []string{fileName},
	}

	// Create progress callback
	var progressFunc hfdownloader.ProgressFunc
	if progress != nil {
		progressFunc = func(event hfdownloader.ProgressEvent) {
			if event.Event == "file_progress" {
				progress(Progress{
					DownloadedBytes: event.Downloaded,
					TotalBytes:      event.Total,
					Percentage:      float64(event.Downloaded) / float64(event.Total) * 100,
				})
			}
		}
	}

	// Execute download
	if err := hfdownloader.Download(ctx, job, settings, progressFunc); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Find and copy the downloaded file from cache
	matches, _ := filepath.Glob(filepath.Join(c.cacheDir, "hub", "**", fileName))
	if len(matches) > 0 {
		return copyFile(matches[0], targetPath)
	}

	return fmt.Errorf("downloaded file not found in cache")
}

// DownloadRepository downloads an entire repository with optional filters
func (c *Client) DownloadRepository(ctx context.Context, repoID, revision string, filters, excludes []string, progress ProgressCallback) error {
	if revision == "" {
		revision = "main"
	}

	settings := hfdownloader.Settings{
		CacheDir:           c.cacheDir,
		Concurrency:        8,
		MaxActiveDownloads: 4,
		MultipartThreshold: "32MiB",
		Verify:             "sha256",
		Retries:            4,
		Token:              c.hfToken,
		Endpoint:           "https://" + c.endpoint,
	}

	job := hfdownloader.Job{
		Repo:     repoID,
		Revision: revision,
		Filters:  filters,
		Excludes: excludes,
	}

	var progressFunc hfdownloader.ProgressFunc
	if progress != nil {
		progressFunc = func(event hfdownloader.ProgressEvent) {
			if event.Event == "file_progress" {
				progress(Progress{
					DownloadedBytes: event.Downloaded,
					TotalBytes:      event.Total,
					Percentage:      float64(event.Downloaded) / float64(event.Total) * 100,
				})
			}
		}
	}

	return hfdownloader.Download(ctx, job, settings, progressFunc)
}

// GetModelInfo retrieves metadata about a model repository
func (c *Client) GetModelInfo(repoID string, revision string) (*hfhub.RepoInfo, error) {
	if revision == "" {
		revision = "main"
	}

	repo := hfhub.New(repoID).
		WithRevision(revision).
		WithEndpoint("https://" + c.endpoint).
		WithAuth(c.hfToken).
		WithCacheDir(c.cacheDir)
	repo.Verbosity = 0

	if err := repo.DownloadInfo(false); err != nil {
		return nil, fmt.Errorf("failed to fetch model info: %w", err)
	}

	return repo.Info(), nil
}

// NewHuggingFaceRepo creates a new HuggingFace repository reference for advanced usage
func (c *Client) NewHuggingFaceRepo(repoID string) *hfhub.Repo {
	return hfhub.New(repoID).
		WithEndpoint("https://" + c.endpoint).
		WithAuth(c.hfToken).
		WithCacheDir(c.cacheDir)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer utils.CloseQuietly(source)

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer utils.CloseQuietly(destination)

	_, err = io.Copy(destination, source)
	return err
}
