// Package storage provides persistence layer with multiple backend support
package storage

import (
	"fmt"
	"time"
)

// StorageType represents the type of storage backend
type StorageType string

const (
	StorageTypeMemory     StorageType = "memory"     // In-memory storage (ephemeral)
	StorageTypeSQLite     StorageType = "sqlite"     // SQLite file-based storage
	StorageTypePostgreSQL StorageType = "postgresql" // PostgreSQL storage (future)
)

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type       StorageType       `mapstructure:"type" yaml:"type" json:"type"`
	SQLite     *SQLiteConfig     `mapstructure:"sqlite" yaml:"sqlite" json:"sqlite,omitempty"`
	PostgreSQL *PostgreSQLConfig `mapstructure:"postgresql" yaml:"postgresql" json:"postgresql,omitempty"`
}

// SQLiteConfig contains SQLite-specific configuration
type SQLiteConfig struct {
	Path      string            `mapstructure:"path" yaml:"path" json:"path"`                    // Database file path
	Pragmas   map[string]string `mapstructure:"pragmas" yaml:"pragmas" json:"pragmas,omitempty"` // SQLite pragmas
	EnableWAL bool              `mapstructure:"enable_wal" yaml:"enable_wal" json:"enableWAL"`   // Enable WAL mode
}

// PostgreSQLConfig contains PostgreSQL-specific configuration
type PostgreSQLConfig struct {
	Host     string `mapstructure:"host" yaml:"host" json:"host"`
	Port     int    `mapstructure:"port" yaml:"port" json:"port"`
	Database string `mapstructure:"database" yaml:"database" json:"database"`
	Username string `mapstructure:"username" yaml:"username" json:"username"`
	Password string `mapstructure:"password" yaml:"password" json:"password"`
	SSLMode  string `mapstructure:"sslmode" yaml:"sslmode" json:"sslmode"` // disable, require, verify-ca, verify-full
}

// Capabilities represents model capabilities configuration
type Capabilities struct {
	Thinking        bool `json:"thinking" db:"thinking"`
	Tools           bool `json:"tools" db:"tools"`
	Rerank          bool `json:"rerank" db:"rerank"`
	Embedding       bool `json:"embedding" db:"embedding"`
	TTS             bool `json:"tts" db:"tts"`
	ASR             bool `json:"asr" db:"asr"`
	ImageGeneration bool `json:"imageGeneration" db:"image_generation"`
	Music           bool `json:"music" db:"music"`
}

// Validate checks if the capabilities configuration is valid
func (c *Capabilities) Validate() error {
	if c.Rerank && c.Embedding {
		return fmt.Errorf("rerank and embedding cannot both be enabled")
	}
	return nil
}

// ApplyConstraints enforces mutual exclusion rules between capabilities.
// If rerank or embedding is enabled, thinking and tools are automatically disabled.
// TTS, ASR, ImageGeneration, Music are mutually exclusive with each other and with other capabilities.
func (c *Capabilities) ApplyConstraints() {
	if c.Rerank || c.Embedding || c.TTS || c.ASR || c.ImageGeneration || c.Music {
		c.Thinking = false
		c.Tools = false
	}
	if c.TTS {
		c.Rerank = false
		c.Embedding = false
		c.ASR = false
		c.ImageGeneration = false
		c.Music = false
	}
	if c.ASR {
		c.Rerank = false
		c.Embedding = false
		c.TTS = false
		c.ImageGeneration = false
		c.Music = false
	}
	if c.ImageGeneration {
		c.Rerank = false
		c.Embedding = false
		c.TTS = false
		c.ASR = false
		c.Music = false
	}
	if c.Music {
		c.Rerank = false
		c.Embedding = false
		c.TTS = false
		c.ASR = false
		c.ImageGeneration = false
	}
}

// Message represents a chat message
type Message struct {
	ID             string                 `json:"id" db:"id"`
	ConversationID string                 `json:"conversationId" db:"conversation_id"`
	Role           string                 `json:"role" db:"role"` // user, assistant, system
	Content        string                 `json:"content" db:"content"`
	Name           string                 `json:"name,omitempty" db:"name"`
	TokenCount     int                    `json:"tokenCount,omitempty" db:"token_count"`
	CreatedAt      time.Time              `json:"createdAt" db:"created_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata"` // JSON encoded
}

// Conversation represents a chat conversation
type Conversation struct {
	ID           string                 `json:"id" db:"id"`
	Model        string                 `json:"model" db:"model"`
	Title        string                 `json:"title,omitempty" db:"title"`
	SystemPrompt string                 `json:"systemPrompt,omitempty" db:"system_prompt"`
	MessageCount int                    `json:"messageCount" db:"message_count"`
	CreatedAt    time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time              `json:"updatedAt" db:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"` // JSON encoded
}

// Benchmark represents a benchmark task
type Benchmark struct {
	ID         string                 `json:"id" db:"id"`
	ModelID    string                 `json:"modelId" db:"model_id"`
	ModelName  string                 `json:"modelName" db:"model_name"`
	Status     string                 `json:"status" db:"status"` // running, completed, failed, cancelled
	Command    string                 `json:"command" db:"command"`
	Config     map[string]interface{} `json:"config,omitempty" db:"config"`   // JSON encoded
	Metrics    map[string]interface{} `json:"metrics,omitempty" db:"metrics"` // JSON encoded
	Error      string                 `json:"error,omitempty" db:"error"`
	CreatedAt  time.Time              `json:"createdAt" db:"created_at"`
	StartedAt  *time.Time             `json:"startedAt,omitempty" db:"started_at"`
	FinishedAt *time.Time             `json:"finishedAt,omitempty" db:"finished_at"`
}

// BenchmarkConfig represents a saved benchmark configuration
type BenchmarkConfig struct {
	Name         string            `json:"name" db:"name"`
	ModelID      string            `json:"modelId" db:"model_id"`
	ModelName    string            `json:"modelName" db:"model_name"`
	LlamaCppPath string            `json:"llamaCppPath" db:"llamacpp_path"`
	Devices      []string          `json:"devices" db:"devices"` // JSON array
	Params       map[string]string `json:"params" db:"params"`   // JSON encoded
	CreatedAt    time.Time         `json:"createdAt" db:"created_at"`
}

// ModelLoadConfig represents a saved model loading configuration
type ModelLoadConfig struct {
	ID        string                 `json:"id" db:"id"`
	NodeID    string                 `json:"nodeId" db:"node_id"`       // Machine/Node ID
	ModelID   string                 `json:"modelId" db:"model_id"`     // Model ID
	ModelName string                 `json:"modelName" db:"model_name"` // Model name (for reference)
	Name      string                 `json:"name" db:"name"`            // '' = default, non-empty = named preset
	Config    map[string]interface{} `json:"config" db:"config"`        // LoadModelParams as JSON
	CreatedAt time.Time              `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time              `json:"updatedAt" db:"updated_at"`
}

// ModelMetadata represents user-defined metadata for a model
type ModelMetadata struct {
	ModelID      string        `json:"modelId" db:"model_id"`                    // Model ID (primary key)
	NodeID       string        `json:"nodeId,omitempty" db:"node_id"`            // Node/Machine ID where model is located
	StoragePath  string        `json:"storagePath,omitempty" db:"storage_path"`  // Storage path (for distributed systems)
	Alias        string        `json:"alias,omitempty" db:"alias"`               // User-defined alias
	Favourite    bool          `json:"favourite" db:"favourite"`                 // Favorite flag
	Tags         []string      `json:"tags,omitempty" db:"tags"`                 // Tags (JSON array)
	Description  string        `json:"description,omitempty" db:"description"`   // User description
	LoadCount    int           `json:"loadCount" db:"load_count"`                // Number of times loaded
	LastLoaded   *time.Time    `json:"lastLoaded,omitempty" db:"last_loaded"`    // Last load time
	TotalTokens  int64         `json:"totalTokens" db:"total_tokens"`            // Total tokens generated
	Capabilities *Capabilities `json:"capabilities,omitempty" db:"capabilities"` // Model capabilities (auto-detected or user-defined)
	CreatedAt    time.Time     `json:"createdAt" db:"created_at"`                // Record creation time
	UpdatedAt    time.Time     `json:"updatedAt" db:"updated_at"`                // Last update time
}

// TTSHistoryItem represents a TTS generation history record
type TTSHistoryItem struct {
	ID        string                 `json:"id" db:"id"`
	Model     string                 `json:"model" db:"model"`
	InputText string                 `json:"inputText" db:"input_text"`
	AudioPath string                 `json:"audioPath" db:"audio_path"`
	Format    string                 `json:"format" db:"format"`
	Duration  float64                `json:"duration" db:"duration"`
	Favourite bool                   `json:"favourite" db:"favourite"`
	Params    map[string]interface{} `json:"params,omitempty" db:"params"`
	CreatedAt time.Time              `json:"createdAt" db:"created_at"`
}

// DownloadTask represents a persistent download task record
type DownloadTask struct {
	ID              string    `json:"id" db:"id"`
	URL             string    `json:"url" db:"url"`
	Path            string    `json:"path" db:"path"`
	FileName        string    `json:"fileName" db:"file_name"`
	State           string    `json:"state" db:"state"` // idle, preparing, downloading, merging, verifying, completed, failed, paused
	DownloadedBytes int64     `json:"downloadedBytes" db:"downloaded_bytes"`
	TotalBytes      int64     `json:"totalBytes" db:"total_bytes"`
	ETag            string    `json:"etag,omitempty" db:"etag"`
	RangeSupported  bool      `json:"rangeSupported" db:"range_supported"`
	FinalURL        string    `json:"finalUrl,omitempty" db:"final_url"`
	TempFileName    string    `json:"tempFileName,omitempty" db:"temp_file_name"`
	PartsTotal      int       `json:"partsTotal" db:"parts_total"`
	PartsCompleted  int       `json:"partsCompleted" db:"parts_completed"`
	FileType        string    `json:"fileType,omitempty" db:"file_type"`
	SourceType      string    `json:"sourceType,omitempty" db:"source_type"`
	RepoID          string    `json:"repoId,omitempty" db:"repo_id"`
	ErrorMessage    string    `json:"errorMessage,omitempty" db:"error_message"`
	RetryCount      int       `json:"retryCount" db:"retry_count"`
	MaxRetries      int       `json:"maxRetries" db:"max_retries"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	StartedAt       time.Time `json:"startedAt,omitempty" db:"started_at"`
	FinishedAt      time.Time `json:"finishedAt,omitempty" db:"finished_at"`
}
