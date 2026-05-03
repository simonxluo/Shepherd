// Package storage provides persistence layer with multiple backend support
package storage

import (
	"context"
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

// Store defines the storage interface
type Store interface {
	// Conversation operations
	CreateConversation(ctx context.Context, conv *Conversation) error
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error)
	UpdateConversation(ctx context.Context, conv *Conversation) error
	DeleteConversation(ctx context.Context, id string) error

	// Message operations
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error)
	DeleteMessages(ctx context.Context, conversationID string) error

	// Benchmark operations
	CreateBenchmark(ctx context.Context, benchmark *Benchmark) error
	GetBenchmark(ctx context.Context, id string) (*Benchmark, error)
	ListBenchmarks(ctx context.Context, modelID string, limit, offset int) ([]*Benchmark, error)
	UpdateBenchmark(ctx context.Context, benchmark *Benchmark) error
	DeleteBenchmark(ctx context.Context, id string) error

	// BenchmarkConfig operations
	CreateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error
	GetBenchmarkConfig(ctx context.Context, name string) (*BenchmarkConfig, error)
	ListBenchmarkConfigs(ctx context.Context, limit, offset int) ([]*BenchmarkConfig, error)
	UpdateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error
	DeleteBenchmarkConfig(ctx context.Context, name string) error

	// ModelLoadConfig operations
	SaveModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error
	GetModelLoadConfig(ctx context.Context, nodeID, modelID string) (*ModelLoadConfig, error)
	DeleteModelLoadConfig(ctx context.Context, nodeID, modelID string) error

	// ModelMetadata operations - 用户设置的模型元数据
	SaveModelMetadata(ctx context.Context, metadata *ModelMetadata) error
	GetModelMetadata(ctx context.Context, modelID string) (*ModelMetadata, error)
	ListModelMetadata(ctx context.Context, limit, offset int) ([]*ModelMetadata, error)
	DeleteModelMetadata(ctx context.Context, modelID string) error
	GetAllModelMetadata(ctx context.Context) (map[string]*ModelMetadata, error) // 批量获取所有模型元数据

	// Cleanup
	Close() error
}

// Manager manages the storage backend
type Manager struct {
	store  Store
	config *StorageConfig
}

// NewManager creates a new storage manager
func NewManager(config *StorageConfig) (*Manager, error) {
	mgr := &Manager{
		config: config,
	}

	var store Store
	var err error

	switch config.Type {
	case StorageTypeMemory:
		store, err = NewMemoryStore()
	case StorageTypeSQLite:
		if config.SQLite == nil {
			return nil, ErrMissingSQLiteConfig
		}
		store, err = NewSQLiteStore(config.SQLite)
	case StorageTypePostgreSQL:
		return nil, ErrPostgreSQLNotSupported
	default:
		return nil, ErrInvalidStorageType
	}

	if err != nil {
		return nil, err
	}

	mgr.store = store
	return mgr, nil
}

// GetStore returns the underlying store
func (m *Manager) GetStore() Store {
	return m.store
}

// Close closes the storage manager
func (m *Manager) Close() error {
	if m.store != nil {
		return m.store.Close()
	}
	return nil
}

// Errors
var (
	ErrInvalidStorageType      = &StorageError{Code: "INVALID_TYPE", Message: "Invalid storage type"}
	ErrMissingSQLiteConfig     = &StorageError{Code: "MISSING_CONFIG", Message: "Missing SQLite configuration"}
	ErrPostgreSQLNotSupported  = &StorageError{Code: "NOT_SUPPORTED", Message: "PostgreSQL support is not yet implemented"}
	ErrConversationNotFound    = &StorageError{Code: "NOT_FOUND", Message: "Conversation not found"}
	ErrMessageNotFound         = &StorageError{Code: "NOT_FOUND", Message: "Message not found"}
	ErrBenchmarkNotFound       = &StorageError{Code: "NOT_FOUND", Message: "Benchmark not found"}
	ErrBenchmarkConfigNotFound = &StorageError{Code: "NOT_FOUND", Message: "Benchmark config not found"}
	ErrModelLoadConfigNotFound = &StorageError{Code: "NOT_FOUND", Message: "Model load config not found"}
	ErrModelMetadataNotFound   = &StorageError{Code: "NOT_FOUND", Message: "Model metadata not found"}
)

// StorageError represents a storage error
type StorageError struct {
	Code    string
	Message string
	Err     error
}

func (e *StorageError) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *StorageError) Unwrap() error {
	return e.Err
}
