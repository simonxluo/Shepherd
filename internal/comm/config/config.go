// Package config provides configuration management for the Shepherd server.
// It handles loading, saving, and validating configuration from YAML files.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/shepherd-project/shepherd/Shepherd/internal/infra/storage"
)

const (
	// DefaultConfigDir is the default configuration directory
	DefaultConfigDir = "config"
	// DefaultConfigFile is the default configuration file name
	DefaultConfigFile = "server.config.yaml"
	// DefaultModelsConfigFile is the default models configuration file
	DefaultModelsConfigFile = "node/models.json"
	// DefaultLaunchConfigFile is the default launch configuration file
	DefaultLaunchConfigFile = "launch_config.json"
)

// Config represents the complete application configuration
type Config struct {
	Server        ServerConfig          `mapstructure:"server" yaml:"server" json:"server"`
	Model         ModelConfig           `mapstructure:"model" yaml:"model" json:"model"`
	Llamacpp      LlamacppConfig        `mapstructure:"llamacpp" yaml:"llamacpp" json:"llamacpp"`
	Download      DownloadConfig        `mapstructure:"download" yaml:"download" json:"download"`
	ModelRepo     ModelRepoConfig       `mapstructure:"model_repo" yaml:"model_repo" json:"modelRepo"`
	Security      SecurityConfig        `mapstructure:"security" yaml:"security" json:"security"`
	Compatibility CompatibilityConfig   `mapstructure:"compatibility" yaml:"compatibility" json:"compatibility"`
	Log           LogConfig             `mapstructure:"log" yaml:"log" json:"log"`
	Storage       storage.StorageConfig `mapstructure:"storage" yaml:"storage" json:"storage"`
	Node          NodeConfig            `mapstructure:"node" yaml:"node" json:"node"`
	Backends      BackendsConfig        `mapstructure:"backends" yaml:"backends" json:"backends"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	WebPort       int    `mapstructure:"web_port" yaml:"web_port" json:"webPort"`
	AnthropicPort int    `mapstructure:"anthropic_port" yaml:"anthropic_port" json:"anthropicPort"`
	OllamaPort    int    `mapstructure:"ollama_port" yaml:"ollama_port" json:"ollamaPort"`
	LMStudioPort  int    `mapstructure:"lmstudio_port" yaml:"lmstudio_port" json:"lmstudioPort"`
	Host          string `mapstructure:"host" yaml:"host" json:"host"`
	ReadTimeout   int    `mapstructure:"read_timeout" yaml:"read_timeout" json:"readTimeout"`    // seconds
	WriteTimeout  int    `mapstructure:"write_timeout" yaml:"write_timeout" json:"writeTimeout"` // seconds
}

// ModelConfig contains model scanning and management configuration
type ModelConfig struct {
	Paths       []string         `mapstructure:"paths" yaml:"paths" json:"paths"`
	PathConfigs []ModelPath      `mapstructure:"path_configs" yaml:"path_configs" json:"pathConfigs"`
	AutoScan    bool             `mapstructure:"auto_scan" yaml:"auto_scan" json:"autoScan"`
	PortRange   string           `mapstructure:"port_range" yaml:"port_range" json:"portRange"`
	Groups      []ModelGroupDef  `mapstructure:"groups" yaml:"groups" json:"groups,omitempty"`
}

// ModelGroupDef defines a model group for swap/exclusive loading (llama-swap style).
// Models within a swap group are mutually exclusive — loading one auto-unloads the others.
type ModelGroupDef struct {
	ID         string   `mapstructure:"id" yaml:"id" json:"id"`                           // Unique group ID
	Models     []string `mapstructure:"models" yaml:"models" json:"models"`                // Model IDs or name patterns
	Swap       bool     `mapstructure:"swap" yaml:"swap" json:"swap"`                      // If true, loading one unloads others in the group
	Exclusive  bool     `mapstructure:"exclusive" yaml:"exclusive" json:"exclusive"`        // If true, also unloads models from other non-persistent groups
	Persistent bool     `mapstructure:"persistent" yaml:"persistent" json:"persistent"`     // If true, not affected by other exclusive groups
}

// LlamacppConfig contains llama.cpp binary paths configuration
type LlamacppConfig struct {
	Paths []LlamacppPath `mapstructure:"paths" yaml:"paths" json:"paths"`
}

// LlamacppPath represents a llama.cpp binary path with metadata
type LlamacppPath struct {
	Path        string `mapstructure:"path" yaml:"path" json:"path"`
	Name        string `mapstructure:"name" yaml:"name" json:"name"`
	Description string `mapstructure:"description" yaml:"description" json:"description,omitempty"`
}

// ModelPath represents a model directory path with metadata
type ModelPath struct {
	Path        string `mapstructure:"path" yaml:"path" json:"path"`
	Name        string `mapstructure:"name" yaml:"name" json:"name,omitempty"`
	Description string `mapstructure:"description" yaml:"description" json:"description,omitempty"`
}

// DownloadConfig contains download manager configuration
type DownloadConfig struct {
	Directory     string `mapstructure:"directory" yaml:"directory" json:"directory"`
	MaxConcurrent int    `mapstructure:"max_concurrent" yaml:"max_concurrent" json:"maxConcurrent"`
	ChunkSize     int    `mapstructure:"chunk_size" yaml:"chunk_size" json:"chunkSize"` // bytes
	RetryCount    int    `mapstructure:"retry_count" yaml:"retry_count" json:"retryCount"`
	Timeout       int    `mapstructure:"timeout" yaml:"timeout" json:"timeout"` // seconds
}

// ModelRepoConfig contains model repository configuration
type ModelRepoConfig struct {
	Endpoint string `mapstructure:"endpoint" yaml:"endpoint" json:"endpoint"` // huggingface.co or hf-mirror.com
	Token    string `mapstructure:"token" yaml:"token" json:"token"`          // HuggingFace API token
	Timeout  int    `mapstructure:"timeout" yaml:"timeout" json:"timeout"`    // seconds
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	APIKeyEnabled  bool     `mapstructure:"api_key_enabled" yaml:"api_key_enabled" json:"apiKeyEnabled"`
	APIKey         string   `mapstructure:"api_key" yaml:"api_key" json:"apiKey"`
	CORSEnabled    bool     `mapstructure:"cors_enabled" yaml:"cors_enabled" json:"corsEnabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins" yaml:"allowed_origins" json:"allowedOrigins"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level" yaml:"level" json:"level"`                  // debug, info, warn, error
	Format     string `mapstructure:"format" yaml:"format" json:"format"`               // json, text
	Output     string `mapstructure:"output" yaml:"output" json:"output"`               // stdout, file, both
	Directory  string `mapstructure:"directory" yaml:"directory" json:"directory"`      // log directory
	MaxSize    int    `mapstructure:"max_size" yaml:"max_size" json:"maxSize"`          // MB
	MaxBackups int    `mapstructure:"max_backups" yaml:"max_backups" json:"maxBackups"` // number of backup files
	MaxAge     int    `mapstructure:"max_age" yaml:"max_age" json:"maxAge"`             // days
	Compress   bool   `mapstructure:"compress" yaml:"compress" json:"compress"`         // compress old logs
}

// CompatibilityConfig contains API compatibility layer settings
type CompatibilityConfig struct {
	Ollama   OllamaConfig   `mapstructure:"ollama" yaml:"ollama" json:"ollama"`
	LMStudio LMStudioConfig `mapstructure:"lmstudio" yaml:"lmstudio" json:"lmstudio"`
}

// OllamaConfig contains Ollama API compatibility settings
type OllamaConfig struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	Port    int  `mapstructure:"port" yaml:"port" json:"port"`
}

// LMStudioConfig contains LM Studio API compatibility settings
type LMStudioConfig struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	Port    int  `mapstructure:"port" yaml:"port" json:"port"`
}

// BackendsConfig contains backend configuration for different inference engines
type BackendsConfig struct {
	VLLM            *VLLMBackendConfig `mapstructure:"vllm" yaml:"vllm" json:"vllm,omitempty"`
	VLLMOmni        *VLLMBackendConfig `mapstructure:"vllm_omni" yaml:"vllm_omni" json:"vllmOmni,omitempty"`
	MultimodalPaths []MultimodalPath   `mapstructure:"multimodal_paths" yaml:"multimodal_paths" json:"multimodalPaths"`
}

// MultimodalPath represents a multimodal model path with metadata
type MultimodalPath struct {
	Path        string `mapstructure:"path" yaml:"path" json:"path"`
	Name        string `mapstructure:"name" yaml:"name" json:"name"`
	Description string `mapstructure:"description" yaml:"description" json:"description,omitempty"`
	Backend     string `mapstructure:"backend" yaml:"backend" json:"backend"`
}

// BackendPath represents a backend binary path with metadata
type BackendPath struct {
	Path        string `mapstructure:"path" yaml:"path" json:"path"`
	Name        string `mapstructure:"name" yaml:"name" json:"name"`
	Description string `mapstructure:"description" yaml:"description" json:"description,omitempty"`
}

// VLLMBackendConfig contains vLLM/vLLM-omni specific configuration
type VLLMBackendConfig struct {
	Enabled     bool          `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	CondaEnv    string        `mapstructure:"conda_env" yaml:"conda_env" json:"condaEnv"`
	CondaPath   string        `mapstructure:"conda_path" yaml:"conda_path" json:"condaPath"`
	ServeBin    string        `mapstructure:"serve_bin" yaml:"serve_bin" json:"serveBin"`
	ExtraArgs   string        `mapstructure:"extra_args" yaml:"extra_args" json:"extraArgs"`
	DefaultPort int           `mapstructure:"default_port" yaml:"default_port" json:"defaultPort"`
	Paths       []BackendPath `mapstructure:"paths" yaml:"paths" json:"paths"`
	Env         []string      `mapstructure:"env" yaml:"env" json:"env,omitempty"`
}

// SchedulerConfig contains task scheduler configuration
type SchedulerConfig struct {
	Strategy       string `mapstructure:"strategy" yaml:"strategy" json:"strategy"`
	MaxQueueSize   int    `mapstructure:"max_queue_size" yaml:"max_queue_size" json:"maxQueueSize"`
	TaskTimeout    int    `mapstructure:"task_timeout" yaml:"task_timeout" json:"taskTimeout"`
	RetryOnFailure bool   `mapstructure:"retry_on_failure" yaml:"retry_on_failure" json:"retryOnFailure"`
	MaxRetries     int    `mapstructure:"max_retries" yaml:"max_retries" json:"maxRetries"`
}

// NodeConfig contains node configuration for the new distributed architecture
type NodeConfig struct {
	ID       string            `mapstructure:"id" yaml:"id" json:"id"`
	Name     string            `mapstructure:"name" yaml:"name" json:"name"`
	Role     string            `mapstructure:"role" yaml:"role" json:"role"`
	Tags     []string          `mapstructure:"tags" yaml:"tags" json:"tags"`
	Metadata map[string]string `mapstructure:"metadata" yaml:"metadata" json:"metadata"`
	// 各角色配置
	MasterRole NodeMasterRoleConfig `mapstructure:"master_role" yaml:"master_role" json:"masterRole"`
	ClientRole NodeClientRoleConfig `mapstructure:"client_role" yaml:"client_role" json:"clientRole"`
	// 能力配置
	Capabilities NodeCapabilitiesConfig `mapstructure:"capabilities" yaml:"capabilities" json:"capabilities"`
}

// NodeMasterRoleConfig contains Master role specific configuration
type NodeMasterRoleConfig struct {
	Port      int             `mapstructure:"port" yaml:"port" json:"port"`
	Scheduler SchedulerConfig `mapstructure:"scheduler" yaml:"scheduler" json:"scheduler"`
}

// NodeClientRoleConfig contains Client role specific configuration
type NodeClientRoleConfig struct {
	Enabled           bool   `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	MasterAddress     string `mapstructure:"master_address" yaml:"master_address" json:"masterAddress"`
	RegisterRetry     int    `mapstructure:"register_retry" yaml:"register_retry" json:"registerRetry"`
	HeartbeatInterval int    `mapstructure:"heartbeat_interval" yaml:"heartbeat_interval" json:"heartbeatInterval"`
	HeartbeatTimeout  int    `mapstructure:"heartbeat_timeout" yaml:"heartbeat_timeout" json:"heartbeatTimeout"`
}

// NodeCapabilitiesConfig contains node capabilities configuration
type NodeCapabilitiesConfig struct {
	PythonEnabled     bool              `mapstructure:"python_enabled" yaml:"python_enabled" json:"pythonEnabled"`
	CondaPath         string            `mapstructure:"conda_path" yaml:"conda_path" json:"condaPath"`
	CondaEnvironments map[string]string `mapstructure:"conda_environments" yaml:"conda_environments" json:"condaEnvironments"`
}

// ModelConfigEntry represents a model configuration entry in models.json
type ModelConfigEntry struct {
	ModelID   string `json:"modelId"`
	Path      string `json:"path,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Alias     string `json:"alias,omitempty"`
	Favourite bool   `json:"favourite"`
	// 分卷模型相关字段
	TotalSize    int64             `json:"totalSize,omitempty"`  // 所有分卷的总大小
	ShardCount   int               `json:"shardCount,omitempty"` // 分卷数量
	ShardFiles   []string          `json:"shardFiles,omitempty"` // 所有分卷文件路径
	PrimaryModel *PrimaryModelInfo `json:"primaryModel,omitempty"`
	Mmproj       *MmprojInfo       `json:"mmproj,omitempty"`
}

// PrimaryModelInfo contains information about the primary model
type PrimaryModelInfo struct {
	FileName        string `json:"fileName"`
	Name            string `json:"name,omitempty"`
	Architecture    string `json:"architecture,omitempty"`
	ContextLength   int    `json:"contextLength,omitempty"`
	EmbeddingLength int    `json:"embeddingLength,omitempty"`
}

// MmprojInfo contains information about the multimodal projector
type MmprojInfo struct {
	FileName     string `json:"fileName"`
	Size         int64  `json:"size,omitempty"` // mmproj 文件大小（字节）
	Name         string `json:"name,omitempty"`
	Architecture string `json:"architecture,omitempty"`
}

// LaunchConfig represents model launch parameters
type LaunchConfig struct {
	CtxSize       int     `mapstructure:"ctx_size" yaml:"ctx_size" json:"ctxSize"`
	BatchSize     int     `mapstructure:"batch_size" yaml:"batch_size" json:"batchSize"`
	Threads       int     `mapstructure:"threads" yaml:"threads" json:"threads"`
	GPULayers     int     `mapstructure:"gpu_layers" yaml:"gpu_layers" json:"gpuLayers"`
	Temperature   float64 `mapstructure:"temperature" yaml:"temperature" json:"temperature"`
	TopP          float64 `mapstructure:"top_p" yaml:"top_p" json:"topP"`
	TopK          int     `mapstructure:"top_k" yaml:"top_k" json:"topK"`
	RepeatPenalty float64 `mapstructure:"repeat_penalty" yaml:"repeat_penalty" json:"repeatPenalty"`
	Seed          int     `mapstructure:"seed" yaml:"seed" json:"seed"`
	NPredict      int     `mapstructure:"n_predict" yaml:"n_predict" json:"nPredict"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	// Get current working directory or use default
	cwd, _ := os.Getwd()
	downloadDir := filepath.Join(cwd, "downloads")
	logDir := filepath.Join(cwd, "logs")

	// 🔧 FIX: 在测试环境中使用空路径,避免扫描模型文件导致超时
	var modelPaths []string
	autoScan := true
	if testing.Testing() {
		// 测试环境:使用空路径,禁用自动扫描
		modelPaths = []string{}
		autoScan = false
	} else {
		// 生产环境:使用默认路径,启用自动扫描
		modelPaths = []string{
			filepath.Join(cwd, "models"),
			filepath.Join(os.Getenv("HOME"), ".cache/huggingface/hub"),
		}
		autoScan = true
	}

	return &Config{
		Server: ServerConfig{
			WebPort:       9190,
			AnthropicPort: 9170,
			OllamaPort:    11434,
			LMStudioPort:  1234,
			Host:          "0.0.0.0",
			ReadTimeout:   60,
			WriteTimeout:  60,
		},
		Model: ModelConfig{
			Paths:    modelPaths,
			AutoScan: autoScan,
		},
		Llamacpp: LlamacppConfig{
			Paths: []LlamacppPath{
				{
					Path: filepath.Join(cwd, "llama.cpp"),
					Name: "Default",
				},
			},
		},
		Download: DownloadConfig{
			Directory:     downloadDir,
			MaxConcurrent: 4,
			ChunkSize:     1024 * 1024,
			RetryCount:    3,
			Timeout:       300,
		},
		Security: SecurityConfig{
			APIKeyEnabled:  false,
			APIKey:         "",
			CORSEnabled:    true,
			AllowedOrigins: []string{"*"},
		},
		Compatibility: CompatibilityConfig{
			Ollama: OllamaConfig{
				Enabled: true,
				Port:    11434,
			},
			LMStudio: LMStudioConfig{
				Enabled: false,
				Port:    1234,
			},
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "both",
			Directory:  logDir,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
		Storage: storage.StorageConfig{
			Type: storage.StorageTypeMemory,
			SQLite: &storage.SQLiteConfig{
				Path:      filepath.Join(cwd, "Shepherd", "data", "shepherd.db"),
				EnableWAL: true,
				Pragmas: map[string]string{
					"cache_size":  "-64000",
					"synchronous": "NORMAL",
				},
			},
		},
		Node: NodeConfig{
			ID:   "auto",
			Name: "",
			Role: "hybrid",
			Tags: []string{},
			Metadata: map[string]string{
				"os":   "linux",
				"arch": "amd64",
			},
			MasterRole: NodeMasterRoleConfig{
				Port: 9190,
				Scheduler: SchedulerConfig{
					Strategy:       "round_robin",
					MaxQueueSize:   100,
					TaskTimeout:    300,
					RetryOnFailure: true,
					MaxRetries:     3,
				},
			},
			ClientRole: NodeClientRoleConfig{
				Enabled:           false,
				MasterAddress:     "",
				RegisterRetry:     3,
				HeartbeatInterval: 5,
				HeartbeatTimeout:  15,
			},
			Capabilities: NodeCapabilitiesConfig{
				PythonEnabled: false,
				CondaPath:     "",
				CondaEnvironments: map[string]string{
					"shepherd": "",
				},
			},
		},
		Backends: BackendsConfig{
			MultimodalPaths: []MultimodalPath{},
		},
		ModelRepo: ModelRepoConfig{
			Endpoint: "huggingface.co",
			Token:    "",
			Timeout:  30,
		},
	}
}

// DefaultLaunchConfig returns default launch parameters
func DefaultLaunchConfig() *LaunchConfig {
	return &LaunchConfig{
		CtxSize:       4096,
		BatchSize:     512,
		Threads:       8,
		GPULayers:     99,
		Temperature:   0.7,
		TopP:          0.9,
		TopK:          40,
		RepeatPenalty: 1.1,
		Seed:          -1, // Random
		NPredict:      -1, // Unlimited
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server ports
	if c.Server.WebPort < 1 || c.Server.WebPort > 65535 {
		return fmt.Errorf("invalid web port: %d", c.Server.WebPort)
	}
	if c.Server.AnthropicPort < 1 || c.Server.AnthropicPort > 65535 {
		return fmt.Errorf("invalid anthropic port: %d", c.Server.AnthropicPort)
	}
	if c.Server.OllamaPort < 1 || c.Server.OllamaPort > 65535 {
		return fmt.Errorf("invalid ollama port: %d", c.Server.OllamaPort)
	}
	if c.Server.LMStudioPort < 1 || c.Server.LMStudioPort > 65535 {
		return fmt.Errorf("invalid lmstudio port: %d", c.Server.LMStudioPort)
	}

	// Check for port conflicts
	ports := map[int]string{
		c.Server.WebPort:       "web",
		c.Server.AnthropicPort: "anthropic",
	}
	if c.Compatibility.Ollama.Enabled {
		if _, exists := ports[c.Server.OllamaPort]; exists {
			return fmt.Errorf("port conflict: ollama port %d conflicts with another service", c.Server.OllamaPort)
		}
		ports[c.Server.OllamaPort] = "ollama"
	}
	if c.Compatibility.LMStudio.Enabled {
		if _, exists := ports[c.Server.LMStudioPort]; exists {
			return fmt.Errorf("port conflict: lmstudio port %d conflicts with another service", c.Server.LMStudioPort)
		}
		ports[c.Server.LMStudioPort] = "lmstudio"
	}

	// Validate download settings
	if c.Download.MaxConcurrent < 1 {
		return fmt.Errorf("max concurrent downloads must be at least 1")
	}
	if c.Download.ChunkSize < 1024 {
		return fmt.Errorf("chunk size too small (minimum 1024 bytes)")
	}

	// Validate model paths
	for _, path := range c.Model.Paths {
		if path == "" {
			return fmt.Errorf("model path cannot be empty")
		}
	}

	if err := c.validateNodeConfig(); err != nil {
		return err
	}

	return nil
}

// validateNodeConfig validates the Node configuration
func (c *Config) validateNodeConfig() error {
	validRoles := map[string]bool{"master": true, "client": true, "hybrid": true}
	if !validRoles[c.Node.Role] {
		return fmt.Errorf("invalid node role: %s (must be master, client, or hybrid)", c.Node.Role)
	}

	if c.Node.MasterRole.Port < 1 || c.Node.MasterRole.Port > 65535 {
		return fmt.Errorf("invalid master role port: %d", c.Node.MasterRole.Port)
	}

	if c.Node.ClientRole.Enabled {
		if c.Node.ClientRole.MasterAddress == "" {
			return fmt.Errorf("client role enabled but master address is empty")
		}
		if c.Node.ClientRole.HeartbeatInterval < 1 {
			return fmt.Errorf("heartbeat interval must be at least 1 second")
		}
		if c.Node.ClientRole.HeartbeatTimeout < c.Node.ClientRole.HeartbeatInterval {
			return fmt.Errorf("heartbeat timeout must be greater than heartbeat interval")
		}
	}

	return nil
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() string {
	// Allow override via environment variable
	if dir := os.Getenv("SHEPHERD_CONFIG_DIR"); dir != "" {
		return dir
	}
	return DefaultConfigDir
}

// EnsureConfigDir ensures the configuration directory exists
func EnsureConfigDir() error {
	configDir := GetConfigDir()
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}
	return nil
}

// Manager manages configuration loading and saving
type Manager struct {
	config           *Config
	configPath       string
	modelsConfigPath string
	launchConfigPath string
	mu               sync.RWMutex
	cachedModels     []ModelConfigEntry
	cachedModelsTime int64
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	configDir := GetConfigDir()
	return &Manager{
		configPath:       filepath.Join(configDir, DefaultConfigFile),
		modelsConfigPath: filepath.Join(configDir, DefaultModelsConfigFile),
		launchConfigPath: filepath.Join(configDir, DefaultLaunchConfigFile),
	}
}

// NewManagerWithPath creates a new configuration manager with a custom config path
func NewManagerWithPath(configPath string) *Manager {
	configDir := filepath.Dir(configPath)
	modelsDir := filepath.Join(GetConfigDir(), "node")
	return &Manager{
		configPath:       configPath,
		modelsConfigPath: filepath.Join(modelsDir, "models.json"),
		launchConfigPath: filepath.Join(configDir, DefaultLaunchConfigFile),
	}
}

// GetConfigPath returns the main configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetModelsConfigPath returns the models configuration file path
func (m *Manager) GetModelsConfigPath() string {
	return m.modelsConfigPath
}

// GetLaunchConfigPath returns the launch configuration file path
func (m *Manager) GetLaunchConfigPath() string {
	return m.launchConfigPath
}
