// Package storage provides SQLite storage implementation
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite" // Use modernc.org/sqlite for pure Go SQLite (CGO-free)
)

// SQLiteStore implements Store interface with SQLite backend
type SQLiteStore struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(config *SQLiteConfig) (*SQLiteStore, error) {
	if config == nil {
		return nil, ErrMissingSQLiteConfig
	}

	// Ensure directory exists
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &SQLiteStore{
		db:   db,
		path: config.Path,
	}

	// Initialize schema
	if err := store.initSchema(config); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the database schema
func (s *SQLiteStore) initSchema(config *SQLiteConfig) error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		model TEXT NOT NULL,
		title TEXT,
		system_prompt TEXT,
		message_count INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		metadata TEXT
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		name TEXT,
		token_count INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		metadata TEXT,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS benchmarks (
		id TEXT PRIMARY KEY,
		model_id TEXT NOT NULL,
		model_name TEXT NOT NULL,
		status TEXT NOT NULL,
		command TEXT,
		config TEXT,
		metrics TEXT,
		error TEXT,
		created_at INTEGER NOT NULL,
		started_at INTEGER,
		finished_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS benchmark_configs (
		name TEXT PRIMARY KEY,
		model_id TEXT NOT NULL,
		model_name TEXT NOT NULL,
		llamacpp_path TEXT NOT NULL,
		devices TEXT,
		params TEXT,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS model_load_configs (
		id TEXT PRIMARY KEY,
		node_id TEXT NOT NULL,
		model_id TEXT NOT NULL,
		model_name TEXT NOT NULL,
		config TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		UNIQUE(node_id, model_id)
	);

	CREATE TABLE IF NOT EXISTS model_metadata (
		model_id TEXT PRIMARY KEY,
		node_id TEXT,
		storage_path TEXT,
		alias TEXT,
		favourite INTEGER DEFAULT 0,
		tags TEXT,
		description TEXT,
		load_count INTEGER DEFAULT 0,
		last_loaded INTEGER,
		total_tokens INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_conversations_created ON conversations(created_at);
	CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at);
	CREATE INDEX IF NOT EXISTS idx_benchmarks_model_id ON benchmarks(model_id);
	CREATE INDEX IF NOT EXISTS idx_benchmarks_status ON benchmarks(status);
	CREATE INDEX IF NOT EXISTS idx_benchmarks_created ON benchmarks(created_at);
	CREATE INDEX IF NOT EXISTS idx_model_load_configs_node_model ON model_load_configs(node_id, model_id);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Apply pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -64000", // 64MB cache
		"PRAGMA temp_store = memory",
	}

	// Apply custom pragmas from config
	if config.Pragmas != nil {
		for key, value := range config.Pragmas {
			pragmas = append(pragmas, fmt.Sprintf("PRAGMA %s = %s", key, value))
		}
	}

	for _, pragma := range pragmas {
		if _, err := s.db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	return nil
}

// CreateConversation creates a new conversation
func (s *SQLiteStore) CreateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if conv.ID == "" {
		conv.ID = generateID("conv")
	}

	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = timeNow()
	}

	if conv.UpdatedAt.IsZero() {
		conv.UpdatedAt = timeNow()
	}

	metadataJSON, _ := json.Marshal(conv.Metadata)

	query := `
	INSERT INTO conversations (id, model, title, system_prompt, message_count, created_at, updated_at, metadata)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		conv.ID,
		conv.Model,
		conv.Title,
		conv.SystemPrompt,
		conv.MessageCount,
		conv.CreatedAt.Unix(),
		conv.UpdatedAt.Unix(),
		string(metadataJSON),
	)

	return err
}

// GetConversation retrieves a conversation by ID
func (s *SQLiteStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT id, model, title, system_prompt, message_count, created_at, updated_at, metadata
	FROM conversations
	WHERE id = ?
	`

	var metadataJSON []byte
	var createdUnix, updatedUnix int64
	conv := &Conversation{}

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&conv.ID,
		&conv.Model,
		&conv.Title,
		&conv.SystemPrompt,
		&conv.MessageCount,
		&createdUnix,
		&updatedUnix,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, ErrConversationNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &conv.Metadata)
	}

	// Convert Unix timestamps to time.Time
	conv.CreatedAt = time.Unix(createdUnix, 0).UTC()
	conv.UpdatedAt = time.Unix(updatedUnix, 0).UTC()

	return conv, nil
}

// ListConversations lists all conversations
func (s *SQLiteStore) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT id, model, title, system_prompt, message_count, created_at, updated_at, metadata
	FROM conversations
	ORDER BY updated_at DESC
	LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	convs := []*Conversation{}

	for rows.Next() {
		var metadataJSON []byte
		var createdUnix, updatedUnix int64
		conv := &Conversation{}

		err := rows.Scan(
			&conv.ID,
			&conv.Model,
			&conv.Title,
			&conv.SystemPrompt,
			&conv.MessageCount,
			&createdUnix,
			&updatedUnix,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &conv.Metadata)
		}

		// Convert Unix timestamps
		conv.CreatedAt = time.Unix(createdUnix, 0).UTC()
		conv.UpdatedAt = time.Unix(updatedUnix, 0).UTC()

		convs = append(convs, conv)
	}

	return convs, nil
}

// UpdateConversation updates an existing conversation
func (s *SQLiteStore) UpdateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv.UpdatedAt = timeNow()

	metadataJSON, _ := json.Marshal(conv.Metadata)

	query := `
	UPDATE conversations
	SET model = ?, title = ?, system_prompt = ?, message_count = ?, updated_at = ?, metadata = ?
	WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		conv.Model,
		conv.Title,
		conv.SystemPrompt,
		conv.MessageCount,
		conv.UpdatedAt.Unix(),
		string(metadataJSON),
		conv.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrConversationNotFound
	}

	return nil
}

// DeleteConversation deletes a conversation and its messages
func (s *SQLiteStore) DeleteConversation(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, "DELETE FROM conversations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrConversationNotFound
	}

	return nil
}

// CreateMessage creates a new message
func (s *SQLiteStore) CreateMessage(ctx context.Context, msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		msg.ID = generateID("msg")
	}

	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = timeNow()
	}

	metadataJSON, _ := json.Marshal(msg.Metadata)

	query := `
	INSERT INTO messages (id, conversation_id, role, content, name, token_count, created_at, metadata)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		msg.ID,
		msg.ConversationID,
		msg.Role,
		msg.Content,
		msg.Name,
		msg.TokenCount,
		msg.CreatedAt.Unix(),
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Update conversation message count and timestamp
	_, err = s.db.ExecContext(ctx,
		"UPDATE conversations SET message_count = message_count + 1, updated_at = ? WHERE id = ?",
		timeNow().Unix(), msg.ConversationID,
	)

	return err
}

// GetMessages retrieves messages for a conversation
func (s *SQLiteStore) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT id, conversation_id, role, content, name, token_count, created_at, metadata
	FROM messages
	WHERE conversation_id = ?
	ORDER BY created_at ASC
	LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	messages := []*Message{}

	for rows.Next() {
		var metadataJSON []byte
		var createdUnix int64
		msg := &Message{}

		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&msg.Name,
			&msg.TokenCount,
			&createdUnix,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &msg.Metadata)
		}

		// Convert Unix timestamp
		msg.CreatedAt = time.Unix(createdUnix, 0).UTC()

		messages = append(messages, msg)
	}

	return messages, nil
}

// DeleteMessages deletes all messages for a conversation
func (s *SQLiteStore) DeleteMessages(ctx context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM messages WHERE conversation_id = ?", conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Reset conversation message count
	_, err = s.db.ExecContext(ctx,
		"UPDATE conversations SET message_count = 0, updated_at = ? WHERE id = ?",
		timeNow().Unix(), conversationID,
	)

	return err
}

// Benchmark operations

// CreateBenchmark creates a new benchmark task
func (s *SQLiteStore) CreateBenchmark(ctx context.Context, benchmark *Benchmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configJSON, err := json.Marshal(benchmark.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark config: %w", err)
	}

	metricsJSON, err := json.Marshal(benchmark.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark metrics: %w", err)
	}

	query := `
		INSERT INTO benchmarks (id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		benchmark.ID,
		benchmark.ModelID,
		benchmark.ModelName,
		benchmark.Status,
		benchmark.Command,
		string(configJSON),
		string(metricsJSON),
		benchmark.Error,
		benchmark.CreatedAt.Unix(),
		timeToUnix(benchmark.StartedAt),
		timeToUnix(benchmark.FinishedAt),
	)

	return err
}

// GetBenchmark retrieves a benchmark by ID
func (s *SQLiteStore) GetBenchmark(ctx context.Context, id string) (*Benchmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var b Benchmark
	var configJSON, metricsJSON sql.NullString
	var startedAt, finishedAt sql.NullInt64

	query := `
		SELECT id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at
		FROM benchmarks WHERE id = ?
	`

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.ModelID, &b.ModelName, &b.Status, &b.Command,
		&configJSON, &metricsJSON, &b.Error,
		&b.CreatedAt, &startedAt, &finishedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrBenchmarkNotFound
	}
	if err != nil {
		return nil, err
	}

	if len(configJSON.String) > 0 {
		if err := json.Unmarshal([]byte(configJSON.String), &b.Config); err != nil {
			// 配置可能是空的，这是正常情况
			b.Config = make(map[string]interface{})
		}
	}

	if len(metricsJSON.String) > 0 {
		if err := json.Unmarshal([]byte(metricsJSON.String), &b.Metrics); err != nil {
			// 指标可能是空的，这是正常情况
			b.Metrics = make(map[string]interface{})
		}
	}

	b.StartedAt = unixToTime(startedAt)
	b.FinishedAt = unixToTime(finishedAt)

	return &b, nil
}

// ListBenchmarks lists benchmarks with optional filtering
func (s *SQLiteStore) ListBenchmarks(ctx context.Context, modelID string, limit, offset int) ([]*Benchmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at
		FROM benchmarks
	`
	args := []interface{}{}

	if modelID != "" {
		query += " WHERE model_id = ?"
		args = append(args, modelID)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var benchmarks []*Benchmark
	for rows.Next() {
		var b Benchmark
		var configJSON, metricsJSON sql.NullString
		var startedAt, finishedAt sql.NullInt64

		err := rows.Scan(
			&b.ID, &b.ModelID, &b.ModelName, &b.Status, &b.Command,
			&configJSON, &metricsJSON, &b.Error,
			&b.CreatedAt, &startedAt, &finishedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(configJSON.String) > 0 {
			if err := json.Unmarshal([]byte(configJSON.String), &b.Config); err != nil {
				b.Config = make(map[string]interface{})
			}
		}

		if len(metricsJSON.String) > 0 {
			if err := json.Unmarshal([]byte(metricsJSON.String), &b.Metrics); err != nil {
				b.Metrics = make(map[string]interface{})
			}
		}

		b.StartedAt = unixToTime(startedAt)
		b.FinishedAt = unixToTime(finishedAt)

		benchmarks = append(benchmarks, &b)
	}

	return benchmarks, nil
}

// UpdateBenchmark updates an existing benchmark
func (s *SQLiteStore) UpdateBenchmark(ctx context.Context, benchmark *Benchmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configJSON, err := json.Marshal(benchmark.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark config: %w", err)
	}

	metricsJSON, err := json.Marshal(benchmark.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark metrics: %w", err)
	}

	query := `
		UPDATE benchmarks
		SET model_id = ?, model_name = ?, status = ?, command = ?, config = ?, metrics = ?, error = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, query,
		benchmark.ModelID, benchmark.ModelName, benchmark.Status, benchmark.Command,
		string(configJSON), string(metricsJSON), benchmark.Error,
		timeToUnix(benchmark.StartedAt), timeToUnix(benchmark.FinishedAt),
		benchmark.ID,
	)

	return err
}

// DeleteBenchmark deletes a benchmark by ID
func (s *SQLiteStore) DeleteBenchmark(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM benchmarks WHERE id = ?", id)
	return err
}

// BenchmarkConfig operations

// CreateBenchmarkConfig creates a new benchmark configuration
func (s *SQLiteStore) CreateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	devicesJSON, _ := json.Marshal(config.Devices)
	paramsJSON, _ := json.Marshal(config.Params)

	query := `
		INSERT INTO benchmark_configs (name, model_id, model_name, llamacpp_path, devices, params, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		config.Name, config.ModelID, config.ModelName, config.LlamaCppPath,
		string(devicesJSON), string(paramsJSON), config.CreatedAt.Unix(),
	)

	return err
}

// GetBenchmarkConfig retrieves a benchmark config by name
func (s *SQLiteStore) GetBenchmarkConfig(ctx context.Context, name string) (*BenchmarkConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var c BenchmarkConfig
	var devicesJSON, paramsJSON sql.NullString

	query := `
		SELECT name, model_id, model_name, llamacpp_path, devices, params, created_at
		FROM benchmark_configs WHERE name = ?
	`

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&c.Name, &c.ModelID, &c.ModelName, &c.LlamaCppPath,
		&devicesJSON, &paramsJSON, &c.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrBenchmarkConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(devicesJSON.String), &c.Devices)
	json.Unmarshal([]byte(paramsJSON.String), &c.Params)

	return &c, nil
}

// ListBenchmarkConfigs lists all benchmark configurations
func (s *SQLiteStore) ListBenchmarkConfigs(ctx context.Context, limit, offset int) ([]*BenchmarkConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT name, model_id, model_name, llamacpp_path, devices, params, created_at
		FROM benchmark_configs
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*BenchmarkConfig
	for rows.Next() {
		var c BenchmarkConfig
		var devicesJSON, paramsJSON sql.NullString

		err := rows.Scan(
			&c.Name, &c.ModelID, &c.ModelName, &c.LlamaCppPath,
			&devicesJSON, &paramsJSON, &c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(devicesJSON.String), &c.Devices)
		json.Unmarshal([]byte(paramsJSON.String), &c.Params)

		configs = append(configs, &c)
	}

	return configs, nil
}

// UpdateBenchmarkConfig updates an existing benchmark configuration
func (s *SQLiteStore) UpdateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	devicesJSON, _ := json.Marshal(config.Devices)
	paramsJSON, _ := json.Marshal(config.Params)

	query := `
		UPDATE benchmark_configs
		SET model_id = ?, model_name = ?, llamacpp_path = ?, devices = ?, params = ?
		WHERE name = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		config.ModelID, config.ModelName, config.LlamaCppPath,
		string(devicesJSON), string(paramsJSON), config.Name,
	)

	return err
}

// DeleteBenchmarkConfig deletes a benchmark configuration by name
func (s *SQLiteStore) DeleteBenchmarkConfig(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM benchmark_configs WHERE name = ?", name)
	return err
}

// ModelLoadConfig operations

// SaveModelLoadConfig saves or updates a model load configuration
func (s *SQLiteStore) SaveModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not provided
	if config.ID == "" {
		config.ID = generateID("mlcfg")
	}

	// Set timestamps
	now := timeNow()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// Marshal config to JSON
	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal model load config: %w", err)
	}

	// Use UPSERT (INSERT OR REPLACE)
	query := `
		INSERT INTO model_load_configs (id, node_id, model_id, model_name, config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id, model_id) DO UPDATE SET
			config = excluded.config,
			model_name = excluded.model_name,
			updated_at = excluded.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		config.ID,
		config.NodeID,
		config.ModelID,
		config.ModelName,
		string(configJSON),
		config.CreatedAt.Unix(),
		config.UpdatedAt.Unix(),
	)

	return err
}

// GetModelLoadConfig retrieves a model load configuration by node ID and model ID
func (s *SQLiteStore) GetModelLoadConfig(ctx context.Context, nodeID, modelID string) (*ModelLoadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var c ModelLoadConfig
	var configJSON []byte
	var createdUnix, updatedUnix int64

	query := `
		SELECT id, node_id, model_id, model_name, config, created_at, updated_at
		FROM model_load_configs
		WHERE node_id = ? AND model_id = ?
	`

	err := s.db.QueryRowContext(ctx, query, nodeID, modelID).Scan(
		&c.ID,
		&c.NodeID,
		&c.ModelID,
		&c.ModelName,
		&configJSON,
		&createdUnix,
		&updatedUnix,
	)

	if err == sql.ErrNoRows {
		return nil, ErrModelLoadConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model load config: %w", err)
	}

	// Parse config JSON
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &c.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal model load config: %w", err)
		}
	}

	// Convert Unix timestamps
	c.CreatedAt = time.Unix(createdUnix, 0).UTC()
	c.UpdatedAt = time.Unix(updatedUnix, 0).UTC()

	return &c, nil
}

// DeleteModelLoadConfig deletes a model load configuration by node ID and model ID
func (s *SQLiteStore) DeleteModelLoadConfig(ctx context.Context, nodeID, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx,
		"DELETE FROM model_load_configs WHERE node_id = ? AND model_id = ?",
		nodeID, modelID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete model load config: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrModelLoadConfigNotFound
	}

	return nil
}

// ========== ModelMetadata Operations ==========

// SaveModelMetadata saves or updates model metadata
func (s *SQLiteStore) SaveModelMetadata(ctx context.Context, metadata *ModelMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := timeNow()

	// Check if this is an insert or update
	var existingTime int64
	err := s.db.QueryRowContext(ctx, "SELECT created_at FROM model_metadata WHERE model_id = ?", metadata.ModelID).Scan(&existingTime)

	if err == sql.ErrNoRows {
		// Insert new record
		if metadata.CreatedAt.IsZero() {
			metadata.CreatedAt = now
		}
		metadata.UpdatedAt = now

		tagsJSON, _ := json.Marshal(metadata.Tags)

		query := `
		INSERT INTO model_metadata (model_id, node_id, storage_path, alias, favourite, tags, description,
			load_count, last_loaded, total_tokens, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err = s.db.ExecContext(ctx, query,
			metadata.ModelID,
			metadata.NodeID,
			metadata.StoragePath,
			metadata.Alias,
			metadata.Favourite,
			string(tagsJSON),
			metadata.Description,
			metadata.LoadCount,
			nil, // last_loaded handled below
			metadata.TotalTokens,
			metadata.CreatedAt.Unix(),
			metadata.UpdatedAt.Unix(),
		)
	} else if err != nil {
		return fmt.Errorf("failed to query model metadata: %w", err)
	} else {
		// Update existing record
		metadata.CreatedAt = time.Unix(existingTime, 0)
		metadata.UpdatedAt = now

		tagsJSON, _ := json.Marshal(metadata.Tags)

		var lastLoaded *int64
		if metadata.LastLoaded != nil {
			ld := metadata.LastLoaded.Unix()
			lastLoaded = &ld
		}

		query := `
		UPDATE model_metadata
		SET node_id = ?, storage_path = ?, alias = ?, favourite = ?, tags = ?, description = ?,
		    load_count = ?, last_loaded = ?, total_tokens = ?, updated_at = ?
		WHERE model_id = ?
		`

		_, err = s.db.ExecContext(ctx, query,
			metadata.NodeID,
			metadata.StoragePath,
			metadata.Alias,
			metadata.Favourite,
			string(tagsJSON),
			metadata.Description,
			metadata.LoadCount,
			lastLoaded,
			metadata.TotalTokens,
			metadata.UpdatedAt.Unix(),
			metadata.ModelID,
		)
	}

	return err
}

// GetModelMetadata retrieves metadata for a single model
func (s *SQLiteStore) GetModelMetadata(ctx context.Context, modelID string) (*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	SELECT model_id, node_id, storage_path, alias, favourite, tags, description,
	       load_count, last_loaded, total_tokens, created_at, updated_at
	FROM model_metadata
	WHERE model_id = ?
	`

	metadata := &ModelMetadata{}
	var tagsJSON string
	var lastLoaded *int64

	err := s.db.QueryRowContext(ctx, query, modelID).Scan(
		&metadata.ModelID,
		&metadata.NodeID,
		&metadata.StoragePath,
		&metadata.Alias,
		&metadata.Favourite,
		&tagsJSON,
		&metadata.Description,
		&metadata.LoadCount,
		&lastLoaded,
		&metadata.TotalTokens,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrModelMetadataNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model metadata: %w", err)
	}

	// Parse tags JSON
	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &metadata.Tags)
	}

	// Parse last_loaded
	if lastLoaded != nil {
		ld := time.Unix(*lastLoaded, 0)
		metadata.LastLoaded = &ld
	}

	return metadata, nil
}

// ListModelMetadata lists model metadata with pagination
func (s *SQLiteStore) ListModelMetadata(ctx context.Context, limit, offset int) ([]*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	SELECT model_id, node_id, storage_path, alias, favourite, tags, description,
	       load_count, last_loaded, total_tokens, created_at, updated_at
	FROM model_metadata
	ORDER BY updated_at DESC
	LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list model metadata: %w", err)
	}
	defer rows.Close()

	var metadatas []*ModelMetadata
	for rows.Next() {
		metadata := &ModelMetadata{}
		var tagsJSON string
		var lastLoaded *int64

		err := rows.Scan(
			&metadata.ModelID,
			&metadata.NodeID,
			&metadata.StoragePath,
			&metadata.Alias,
			&metadata.Favourite,
			&tagsJSON,
			&metadata.Description,
			&metadata.LoadCount,
			&lastLoaded,
			&metadata.TotalTokens,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model metadata: %w", err)
		}

		// Parse tags JSON
		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &metadata.Tags)
		}

		// Parse last_loaded
		if lastLoaded != nil {
			ld := time.Unix(*lastLoaded, 0)
			metadata.LastLoaded = &ld
		}

		metadatas = append(metadatas, metadata)
	}

	return metadatas, nil
}

// DeleteModelMetadata deletes metadata for a model
func (s *SQLiteStore) DeleteModelMetadata(ctx context.Context, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, "DELETE FROM model_metadata WHERE model_id = ?", modelID)
	if err != nil {
		return fmt.Errorf("failed to delete model metadata: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrModelMetadataNotFound
	}

	return nil
}

// GetAllModelMetadata retrieves all model metadata as a map
func (s *SQLiteStore) GetAllModelMetadata(ctx context.Context) (map[string]*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	SELECT model_id, node_id, storage_path, alias, favourite, tags, description,
	       load_count, last_loaded, total_tokens, created_at, updated_at
	FROM model_metadata
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all model metadata: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*ModelMetadata)
	for rows.Next() {
		metadata := &ModelMetadata{}
		var tagsJSON string
		var lastLoaded *int64
		var createdAt int64
		var updatedAt int64

		err := rows.Scan(
			&metadata.ModelID,
			&metadata.NodeID,
			&metadata.StoragePath,
			&metadata.Alias,
			&metadata.Favourite,
			&tagsJSON,
			&metadata.Description,
			&metadata.LoadCount,
			&lastLoaded,
			&metadata.TotalTokens,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model metadata: %w", err)
		}

		// Convert Unix timestamps to time.Time
		metadata.CreatedAt = time.Unix(createdAt, 0)
		metadata.UpdatedAt = time.Unix(updatedAt, 0)

		// Parse tags JSON
		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &metadata.Tags)
		}

		// Parse last_loaded
		if lastLoaded != nil {
			ld := time.Unix(*lastLoaded, 0)
			metadata.LastLoaded = &ld
		}

		result[metadata.ModelID] = metadata
	}

	return result, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Close()
}

// Stats returns statistics about the database
func (s *SQLiteStore) Stats() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})

	// Get table counts
	var convCount, msgCount int64

	s.db.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&convCount)
	s.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&msgCount)

	stats["conversations"] = convCount
	stats["messages"] = msgCount
	stats["type"] = "sqlite"
	stats["path"] = s.path

	// Get database size
	if info, err := os.Stat(s.path); err == nil {
		stats["size_bytes"] = info.Size()
	}

	return stats, nil
}

// Helper functions for time handling

func timeNow() time.Time {
	return time.Now().UTC()
}

func timeToUnix(t *time.Time) sql.NullInt64 {
	if t == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: t.Unix(), Valid: true}
}

func unixToTime(t sql.NullInt64) *time.Time {
	if !t.Valid {
		return nil
	}
	u := time.Unix(t.Int64, 0).UTC()
	return &u
}
