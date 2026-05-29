package storage

import "context"

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

	// Named ModelLoadConfig operations (multi-preset support)
	ListModelLoadConfigs(ctx context.Context, nodeID, modelID string) ([]*ModelLoadConfig, error)
	SaveNamedModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error
	DeleteNamedModelLoadConfig(ctx context.Context, nodeID, modelID, name string) error

	// LaunchProfile operations
	CreateLaunchProfile(ctx context.Context, profile *LaunchProfile) error
	GetLaunchProfile(ctx context.Context, id string) (*LaunchProfile, error)
	ListLaunchProfiles(ctx context.Context, backendType, modelScope string) ([]*LaunchProfile, error)
	UpdateLaunchProfile(ctx context.Context, profile *LaunchProfile) error
	DeleteLaunchProfile(ctx context.Context, id string) error

	// ModelMetadata operations - user-defined model metadata
	SaveModelMetadata(ctx context.Context, metadata *ModelMetadata) error
	GetModelMetadata(ctx context.Context, modelID string) (*ModelMetadata, error)
	ListModelMetadata(ctx context.Context, limit, offset int) ([]*ModelMetadata, error)
	DeleteModelMetadata(ctx context.Context, modelID string) error
	GetAllModelMetadata(ctx context.Context) (map[string]*ModelMetadata, error) // batch fetch all model metadata

	// TTS History operations
	CreateTTSHistory(ctx context.Context, item *TTSHistoryItem) error
	GetTTSHistory(ctx context.Context, id string) (*TTSHistoryItem, error)
	ListTTSHistory(ctx context.Context, limit, offset int, favouriteOnly *bool) ([]*TTSHistoryItem, error)
	UpdateTTSHistoryFavourite(ctx context.Context, id string, favourite bool) error
	DeleteTTSHistory(ctx context.Context, id string) error

	// Download task operations
	CreateDownloadTask(ctx context.Context, task *DownloadTask) error
	GetDownloadTask(ctx context.Context, id string) (*DownloadTask, error)
	ListDownloadTasks(ctx context.Context, limit, offset int) ([]*DownloadTask, error)
	UpdateDownloadTask(ctx context.Context, task *DownloadTask) error
	DeleteDownloadTask(ctx context.Context, id string) error
	ListActiveDownloadTasks(ctx context.Context) ([]*DownloadTask, error)

	// MCP Server operations
	CreateMCPServer(ctx context.Context, server *MCPServer) error
	GetMCPServer(ctx context.Context, id string) (*MCPServer, error)
	ListMCPServers(ctx context.Context) ([]*MCPServer, error)
	UpdateMCPServer(ctx context.Context, server *MCPServer) error
	DeleteMCPServer(ctx context.Context, id string) error

	// MCP Tool operations
	CreateMCPTool(ctx context.Context, tool *MCPTool) error
	ListMCPToolsByServer(ctx context.Context, serverID string) ([]*MCPTool, error)
	DeleteMCPToolsByServer(ctx context.Context, serverID string) error

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
	store, err := createStore(config)
	if err != nil {
		return nil, err
	}

	return &Manager{
		store:  store,
		config: config,
	}, nil
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
	ErrMissingPostgreSQLConfig = &StorageError{Code: "MISSING_CONFIG", Message: "Missing PostgreSQL configuration"}
	ErrMissingMySQLConfig      = &StorageError{Code: "MISSING_CONFIG", Message: "Missing MySQL configuration"}
	ErrConversationNotFound    = &StorageError{Code: "NOT_FOUND", Message: "Conversation not found"}
	ErrMessageNotFound         = &StorageError{Code: "NOT_FOUND", Message: "Message not found"}
	ErrBenchmarkNotFound       = &StorageError{Code: "NOT_FOUND", Message: "Benchmark not found"}
	ErrBenchmarkConfigNotFound = &StorageError{Code: "NOT_FOUND", Message: "Benchmark config not found"}
	ErrModelLoadConfigNotFound = &StorageError{Code: "NOT_FOUND", Message: "Model load config not found"}
	ErrModelMetadataNotFound   = &StorageError{Code: "NOT_FOUND", Message: "Model metadata not found"}
	ErrTTSHistoryNotFound      = &StorageError{Code: "NOT_FOUND", Message: "TTS history item not found"}
	ErrDownloadTaskNotFound    = &StorageError{Code: "NOT_FOUND", Message: "Download task not found"}
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

// Ensure StorageError implements the error interface.
var _ error = (*StorageError)(nil)
