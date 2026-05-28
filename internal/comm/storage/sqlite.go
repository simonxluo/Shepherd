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

	sqlitedb "github.com/simonxluo/Shepherd/internal/comm/storage/sqlc/sqlite/db"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	_ "modernc.org/sqlite" // Use modernc.org/sqlite for pure Go SQLite (CGO-free)
)

// SQLiteStore implements Store interface with SQLite backend
type SQLiteStore struct {
	db      *sql.DB
	queries *sqlitedb.Queries
	path    string
	mu      sync.RWMutex
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
		utils.CloseQuietly(db)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 限制连接数，配合已有互斥锁消除 "database is locked" 错误
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Apply pragmas before migrations
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -64000", // 64MB cache
		"PRAGMA temp_store = memory",
	}
	if config.Pragmas != nil {
		for key, value := range config.Pragmas {
			pragmas = append(pragmas, fmt.Sprintf("PRAGMA %s = %s", key, value))
		}
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			utils.CloseQuietly(db)
			return nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	// Run schema migrations via goose
	if err := RunMigrations(db, StorageTypeSQLite); err != nil {
		utils.CloseQuietly(db)
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	store := &SQLiteStore{
		db:      db,
		queries: sqlitedb.New(db),
		path:    config.Path,
	}

	return store, nil
}

// --- Conversation operations ---

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

	return s.queries.CreateConversation(ctx, sqlitedb.CreateConversationParams{
		ID:           conv.ID,
		Model:        conv.Model,
		Title:        toNullString(conv.Title),
		SystemPrompt: toNullString(conv.SystemPrompt),
		MessageCount: sql.NullInt64{Int64: int64(conv.MessageCount), Valid: true},
		CreatedAt:    conv.CreatedAt.Unix(),
		UpdatedAt:    conv.UpdatedAt.Unix(),
		Metadata:     toNullString(string(metadataJSON)),
	})
}

func (s *SQLiteStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetConversation(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	conv := &Conversation{
		ID:           row.ID,
		Model:        row.Model,
		Title:        row.Title.String,
		SystemPrompt: row.SystemPrompt.String,
		MessageCount: int(row.MessageCount.Int64),
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:    time.Unix(row.UpdatedAt, 0).UTC(),
	}

	if row.Metadata.Valid && row.Metadata.String != "" {
		if !utils.UnmarshalQuietly([]byte(row.Metadata.String), &conv.Metadata, "会话元数据") {
			conv.Metadata = make(map[string]interface{})
		}
	}

	return conv, nil
}

func (s *SQLiteStore) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListConversations(ctx, sqlitedb.ListConversationsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	convs := make([]*Conversation, 0, len(rows))
	for _, row := range rows {
		conv := &Conversation{
			ID:           row.ID,
			Model:        row.Model,
			Title:        row.Title.String,
			SystemPrompt: row.SystemPrompt.String,
			MessageCount: int(row.MessageCount.Int64),
			CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
			UpdatedAt:    time.Unix(row.UpdatedAt, 0).UTC(),
		}

		if row.Metadata.Valid && row.Metadata.String != "" {
			if !utils.UnmarshalQuietly([]byte(row.Metadata.String), &conv.Metadata, "会话元数据") {
				conv.Metadata = make(map[string]interface{})
			}
		}

		convs = append(convs, conv)
	}

	return convs, nil
}

func (s *SQLiteStore) UpdateConversation(ctx context.Context, conv *Conversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv.UpdatedAt = timeNow()
	metadataJSON, _ := json.Marshal(conv.Metadata)

	result, err := s.db.ExecContext(ctx, `
		UPDATE conversations
		SET model = ?, title = ?, system_prompt = ?, message_count = ?, updated_at = ?, metadata = ?
		WHERE id = ?
	`, conv.Model, conv.Title, conv.SystemPrompt, conv.MessageCount, conv.UpdatedAt.Unix(), string(metadataJSON), conv.ID)
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrConversationNotFound
	}
	return nil
}

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

// --- Message operations ---

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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)

	err = qtx.CreateMessage(ctx, sqlitedb.CreateMessageParams{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		Role:           msg.Role,
		Content:        msg.Content,
		Name:           toNullString(msg.Name),
		TokenCount:     sql.NullInt64{Int64: int64(msg.TokenCount), Valid: true},
		CreatedAt:      msg.CreatedAt.Unix(),
		Metadata:       toNullString(string(metadataJSON)),
	})
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	err = qtx.IncrementMessageCount(ctx, sqlitedb.IncrementMessageCountParams{
		UpdatedAt: timeNow().Unix(),
		ID:        msg.ConversationID,
	})
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.GetMessages(ctx, sqlitedb.GetMessagesParams{
		ConversationID: conversationID,
		Limit:          int64(limit),
		Offset:         int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*Message, 0, len(rows))
	for _, row := range rows {
		msg := &Message{
			ID:             row.ID,
			ConversationID: row.ConversationID,
			Role:           row.Role,
			Content:        row.Content,
			Name:           row.Name.String,
			TokenCount:     int(row.TokenCount.Int64),
			CreatedAt:      time.Unix(row.CreatedAt, 0).UTC(),
		}

		if row.Metadata.Valid && row.Metadata.String != "" {
			if !utils.UnmarshalQuietly([]byte(row.Metadata.String), &msg.Metadata, "消息元数据") {
				msg.Metadata = make(map[string]interface{})
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *SQLiteStore) DeleteMessages(ctx context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.queries.DeleteMessages(ctx, conversationID); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return s.queries.ResetMessageCount(ctx, sqlitedb.ResetMessageCountParams{
		UpdatedAt: timeNow().Unix(),
		ID:        conversationID,
	})
}

// --- Benchmark operations ---

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

	return s.queries.CreateBenchmark(ctx, sqlitedb.CreateBenchmarkParams{
		ID:         benchmark.ID,
		ModelID:    benchmark.ModelID,
		ModelName:  benchmark.ModelName,
		Status:     benchmark.Status,
		Command:    toNullString(benchmark.Command),
		Config:     toNullString(string(configJSON)),
		Metrics:    toNullString(string(metricsJSON)),
		Error:      toNullString(benchmark.Error),
		CreatedAt:  benchmark.CreatedAt.Unix(),
		StartedAt:  timeToUnix(benchmark.StartedAt),
		FinishedAt: timeToUnix(benchmark.FinishedAt),
	})
}

func (s *SQLiteStore) GetBenchmark(ctx context.Context, id string) (*Benchmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetBenchmark(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrBenchmarkNotFound
	}
	if err != nil {
		return nil, err
	}

	return benchmarkFromRow(row), nil
}

func (s *SQLiteStore) ListBenchmarks(ctx context.Context, modelID string, limit, offset int) ([]*Benchmark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if modelID != "" {
		rows, err := s.queries.ListBenchmarksByModel(ctx, sqlitedb.ListBenchmarksByModelParams{
			ModelID: modelID,
			Limit:   int64(limit),
			Offset:  int64(offset),
		})
		if err != nil {
			return nil, err
		}
		benchmarks := make([]*Benchmark, 0, len(rows))
		for _, row := range rows {
			benchmarks = append(benchmarks, benchmarkFromRow(row))
		}
		return benchmarks, nil
	}

	rows, err := s.queries.ListBenchmarks(ctx, sqlitedb.ListBenchmarksParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}
	benchmarks := make([]*Benchmark, 0, len(rows))
	for _, row := range rows {
		benchmarks = append(benchmarks, benchmarkFromRow(row))
	}
	return benchmarks, nil
}

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

	return s.queries.UpdateBenchmark(ctx, sqlitedb.UpdateBenchmarkParams{
		ModelID:    benchmark.ModelID,
		ModelName:  benchmark.ModelName,
		Status:     benchmark.Status,
		Command:    toNullString(benchmark.Command),
		Config:     toNullString(string(configJSON)),
		Metrics:    toNullString(string(metricsJSON)),
		Error:      toNullString(benchmark.Error),
		StartedAt:  timeToUnix(benchmark.StartedAt),
		FinishedAt: timeToUnix(benchmark.FinishedAt),
		ID:         benchmark.ID,
	})
}

func (s *SQLiteStore) DeleteBenchmark(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.queries.DeleteBenchmark(ctx, id)
}

// --- BenchmarkConfig operations ---

func (s *SQLiteStore) CreateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	devicesJSON, _ := json.Marshal(config.Devices)
	paramsJSON, _ := json.Marshal(config.Params)

	return s.queries.CreateBenchmarkConfig(ctx, sqlitedb.CreateBenchmarkConfigParams{
		Name:         config.Name,
		ModelID:      config.ModelID,
		ModelName:    config.ModelName,
		LlamacppPath: config.LlamaCppPath,
		Devices:      toNullString(string(devicesJSON)),
		Params:       toNullString(string(paramsJSON)),
		CreatedAt:    config.CreatedAt.Unix(),
	})
}

func (s *SQLiteStore) GetBenchmarkConfig(ctx context.Context, name string) (*BenchmarkConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetBenchmarkConfig(ctx, name)
	if err == sql.ErrNoRows {
		return nil, ErrBenchmarkConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	return benchmarkConfigFromRow(row), nil
}

func (s *SQLiteStore) ListBenchmarkConfigs(ctx context.Context, limit, offset int) ([]*BenchmarkConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListBenchmarkConfigs(ctx, sqlitedb.ListBenchmarkConfigsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}

	configs := make([]*BenchmarkConfig, 0, len(rows))
	for _, row := range rows {
		configs = append(configs, benchmarkConfigFromRow(row))
	}
	return configs, nil
}

func (s *SQLiteStore) UpdateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	devicesJSON, _ := json.Marshal(config.Devices)
	paramsJSON, _ := json.Marshal(config.Params)

	return s.queries.UpdateBenchmarkConfig(ctx, sqlitedb.UpdateBenchmarkConfigParams{
		ModelID:      config.ModelID,
		ModelName:    config.ModelName,
		LlamacppPath: config.LlamaCppPath,
		Devices:      toNullString(string(devicesJSON)),
		Params:       toNullString(string(paramsJSON)),
		Name:         config.Name,
	})
}

func (s *SQLiteStore) DeleteBenchmarkConfig(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.queries.DeleteBenchmarkConfig(ctx, name)
}

// --- ModelLoadConfig operations ---

func (s *SQLiteStore) SaveModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.ID == "" {
		config.ID = generateID("mlcfg")
	}

	now := timeNow()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal model load config: %w", err)
	}

	return s.queries.UpsertModelLoadConfig(ctx, sqlitedb.UpsertModelLoadConfigParams{
		ID:        config.ID,
		NodeID:    config.NodeID,
		ModelID:   config.ModelID,
		ModelName: config.ModelName,
		Config:    string(configJSON),
		CreatedAt: config.CreatedAt.Unix(),
		UpdatedAt: config.UpdatedAt.Unix(),
		Name:      "",
	})
}

func (s *SQLiteStore) GetModelLoadConfig(ctx context.Context, nodeID, modelID string) (*ModelLoadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetModelLoadConfig(ctx, sqlitedb.GetModelLoadConfigParams{
		NodeID:  nodeID,
		ModelID: modelID,
	})
	if err == sql.ErrNoRows {
		return nil, ErrModelLoadConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model load config: %w", err)
	}

	return modelLoadConfigFromRow(row)
}

func (s *SQLiteStore) DeleteModelLoadConfig(ctx context.Context, nodeID, modelID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx,
		"DELETE FROM model_load_configs WHERE node_id = ? AND model_id = ? AND name = ''",
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

func (s *SQLiteStore) ListModelLoadConfigs(ctx context.Context, nodeID, modelID string) ([]*ModelLoadConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListModelLoadConfigs(ctx, sqlitedb.ListModelLoadConfigsParams{
		NodeID:  nodeID,
		ModelID: modelID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list model load configs: %w", err)
	}

	configs := make([]*ModelLoadConfig, 0, len(rows))
	for _, row := range rows {
		c, err := modelLoadConfigFromRow(row)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func (s *SQLiteStore) SaveNamedModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
	if config.Name == "" {
		return fmt.Errorf("named config requires a non-empty name")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if config.ID == "" {
		config.ID = generateID("mlcfg")
	}

	now := timeNow()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	configJSON, err := json.Marshal(config.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal model load config: %w", err)
	}

	return s.queries.UpsertModelLoadConfig(ctx, sqlitedb.UpsertModelLoadConfigParams{
		ID:        config.ID,
		NodeID:    config.NodeID,
		ModelID:   config.ModelID,
		ModelName: config.ModelName,
		Config:    string(configJSON),
		CreatedAt: config.CreatedAt.Unix(),
		UpdatedAt: config.UpdatedAt.Unix(),
		Name:      config.Name,
	})
}

func (s *SQLiteStore) DeleteNamedModelLoadConfig(ctx context.Context, nodeID, modelID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx,
		"DELETE FROM model_load_configs WHERE node_id = ? AND model_id = ? AND name = ?",
		nodeID, modelID, name,
	)
	if err != nil {
		return fmt.Errorf("failed to delete named model load config: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrModelLoadConfigNotFound
	}
	return nil
}

// --- LaunchProfile operations ---

func (s *SQLiteStore) CreateLaunchProfile(ctx context.Context, profile *LaunchProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if profile.ID == "" {
		profile.ID = generateID("lprof")
	}
	now := timeNow()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	paramsJSON, err := json.Marshal(profile.Params)
	if err != nil {
		return fmt.Errorf("failed to marshal launch profile params: %w", err)
	}
	envJSON, err := json.Marshal(profile.Env)
	if err != nil {
		return fmt.Errorf("failed to marshal launch profile env: %w", err)
	}

	return s.queries.CreateLaunchProfile(ctx, sqlitedb.CreateLaunchProfileParams{
		ID:             profile.ID,
		Name:           profile.Name,
		BackendType:    profile.BackendType,
		InstallationID: toNullString(profile.InstallationID),
		ModelScope:     toNullString(profile.ModelScope),
		Params:         string(paramsJSON),
		Env:            toNullString(string(envJSON)),
		ExtraArgs:      toNullString(profile.ExtraArgs),
		CreatedAt:      profile.CreatedAt.Unix(),
		UpdatedAt:      profile.UpdatedAt.Unix(),
	})
}

func (s *SQLiteStore) GetLaunchProfile(ctx context.Context, id string) (*LaunchProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetLaunchProfile(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrModelLoadConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	return launchProfileFromRow(row)
}

func (s *SQLiteStore) ListLaunchProfiles(ctx context.Context, backendType, modelScope string) ([]*LaunchProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListLaunchProfiles(ctx, sqlitedb.ListLaunchProfilesParams{
		BackendTypeFilter: backendType,
		ModelScopeFilter:  modelScope,
	})
	if err != nil {
		return nil, err
	}

	profiles := make([]*LaunchProfile, 0, len(rows))
	for _, row := range rows {
		p, err := launchProfileFromRow(row)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (s *SQLiteStore) UpdateLaunchProfile(ctx context.Context, profile *LaunchProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	paramsJSON, err := json.Marshal(profile.Params)
	if err != nil {
		return fmt.Errorf("failed to marshal launch profile params: %w", err)
	}
	envJSON, err := json.Marshal(profile.Env)
	if err != nil {
		return fmt.Errorf("failed to marshal launch profile env: %w", err)
	}
	profile.UpdatedAt = timeNow()

	result, err := s.db.ExecContext(ctx, `
		UPDATE launch_profiles
		SET name = ?, backend_type = ?, installation_id = ?, model_scope = ?, params = ?, env = ?, extra_args = ?, updated_at = ?
		WHERE id = ?
	`, profile.Name, profile.BackendType, profile.InstallationID, profile.ModelScope, string(paramsJSON), string(envJSON), profile.ExtraArgs, profile.UpdatedAt.Unix(), profile.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrModelLoadConfigNotFound
	}
	return nil
}

func (s *SQLiteStore) DeleteLaunchProfile(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, "DELETE FROM launch_profiles WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrModelLoadConfigNotFound
	}
	return nil
}

// --- ModelMetadata operations ---

func (s *SQLiteStore) SaveModelMetadata(ctx context.Context, metadata *ModelMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := timeNow()

	// Check if this is an insert or update
	existingTime, err := s.queries.GetModelMetadataCreatedAt(ctx, metadata.ModelID)
	if err == sql.ErrNoRows {
		// Insert new record
		if metadata.CreatedAt.IsZero() {
			metadata.CreatedAt = now
		}
		metadata.UpdatedAt = now

		tagsJSON, _ := json.Marshal(metadata.Tags)
		capsJSON, _ := json.Marshal(metadata.Capabilities)

		return s.queries.InsertModelMetadata(ctx, sqlitedb.InsertModelMetadataParams{
			ModelID:      metadata.ModelID,
			NodeID:       toNullString(metadata.NodeID),
			StoragePath:  toNullString(metadata.StoragePath),
			Alias:        toNullString(metadata.Alias),
			Favourite:    boolToNullInt64(metadata.Favourite),
			Tags:         toNullString(string(tagsJSON)),
			Description:  toNullString(metadata.Description),
			LoadCount:    sql.NullInt64{Int64: int64(metadata.LoadCount), Valid: true},
			LastLoaded:   timeToUnix(metadata.LastLoaded),
			TotalTokens:  sql.NullInt64{Int64: metadata.TotalTokens, Valid: true},
			Capabilities: toNullString(string(capsJSON)),
			CreatedAt:    metadata.CreatedAt.Unix(),
			UpdatedAt:    metadata.UpdatedAt.Unix(),
		})
	} else if err != nil {
		return fmt.Errorf("failed to query model metadata: %w", err)
	}

	// Update existing record
	metadata.CreatedAt = time.Unix(existingTime, 0)
	metadata.UpdatedAt = now

	tagsJSON, _ := json.Marshal(metadata.Tags)
	capsJSON, _ := json.Marshal(metadata.Capabilities)

	return s.queries.UpdateModelMetadata(ctx, sqlitedb.UpdateModelMetadataParams{
		NodeID:       toNullString(metadata.NodeID),
		StoragePath:  toNullString(metadata.StoragePath),
		Alias:        toNullString(metadata.Alias),
		Favourite:    boolToNullInt64(metadata.Favourite),
		Tags:         toNullString(string(tagsJSON)),
		Description:  toNullString(metadata.Description),
		LoadCount:    sql.NullInt64{Int64: int64(metadata.LoadCount), Valid: true},
		LastLoaded:   timeToUnix(metadata.LastLoaded),
		TotalTokens:  sql.NullInt64{Int64: metadata.TotalTokens, Valid: true},
		Capabilities: toNullString(string(capsJSON)),
		UpdatedAt:    metadata.UpdatedAt.Unix(),
		ModelID:      metadata.ModelID,
	})
}

func (s *SQLiteStore) GetModelMetadata(ctx context.Context, modelID string) (*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row, err := s.queries.GetModelMetadata(ctx, modelID)
	if err == sql.ErrNoRows {
		return nil, ErrModelMetadataNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model metadata: %w", err)
	}

	return modelMetadataFromRow(row), nil
}

func (s *SQLiteStore) ListModelMetadata(ctx context.Context, limit, offset int) ([]*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.queries.ListModelMetadata(ctx, sqlitedb.ListModelMetadataParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list model metadata: %w", err)
	}

	metadatas := make([]*ModelMetadata, 0, len(rows))
	for _, row := range rows {
		metadatas = append(metadatas, modelMetadataFromRow(row))
	}
	return metadatas, nil
}

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

func (s *SQLiteStore) GetAllModelMetadata(ctx context.Context) (map[string]*ModelMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.queries.GetAllModelMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all model metadata: %w", err)
	}

	result := make(map[string]*ModelMetadata, len(rows))
	for _, row := range rows {
		m := modelMetadataFromRow(row)
		result[m.ModelID] = m
	}
	return result, nil
}

// --- TTS History operations ---

func (s *SQLiteStore) CreateTTSHistory(ctx context.Context, item *TTSHistoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = generateID("tts")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = timeNow()
	}

	paramsJSON, _ := json.Marshal(item.Params)

	return s.queries.CreateTTSHistory(ctx, sqlitedb.CreateTTSHistoryParams{
		ID:        item.ID,
		Model:     item.Model,
		InputText: item.InputText,
		AudioPath: item.AudioPath,
		Format:    item.Format,
		Duration:  sql.NullFloat64{Float64: item.Duration, Valid: true},
		Favourite: boolToNullInt64(item.Favourite),
		Params:    toNullString(string(paramsJSON)),
		CreatedAt: item.CreatedAt.Unix(),
	})
}

func (s *SQLiteStore) GetTTSHistory(ctx context.Context, id string) (*TTSHistoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetTTSHistory(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrTTSHistoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get TTS history: %w", err)
	}

	return ttsHistoryFromRow(row), nil
}

func (s *SQLiteStore) ListTTSHistory(ctx context.Context, limit, offset int, favouriteOnly *bool) ([]*TTSHistoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if favouriteOnly != nil && *favouriteOnly {
		rows, err := s.queries.ListTTSHistoryFavourite(ctx, sqlitedb.ListTTSHistoryFavouriteParams{
			Limit:  int64(limit),
			Offset: int64(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list TTS history: %w", err)
		}
		items := make([]*TTSHistoryItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, ttsHistoryFromRow(row))
		}
		return items, nil
	}

	rows, err := s.queries.ListTTSHistory(ctx, sqlitedb.ListTTSHistoryParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list TTS history: %w", err)
	}
	items := make([]*TTSHistoryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ttsHistoryFromRow(row))
	}
	return items, nil
}

func (s *SQLiteStore) UpdateTTSHistoryFavourite(ctx context.Context, id string, favourite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, "UPDATE tts_history SET favourite = ? WHERE id = ?", boolToInt(favourite), id)
	if err != nil {
		return fmt.Errorf("failed to update TTS history favourite: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTTSHistoryNotFound
	}
	return nil
}

func (s *SQLiteStore) DeleteTTSHistory(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, "DELETE FROM tts_history WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete TTS history: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTTSHistoryNotFound
	}
	return nil
}

// --- Download Task operations ---

func (s *SQLiteStore) CreateDownloadTask(ctx context.Context, task *DownloadTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		task.ID = generateID("dl")
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = timeNow()
	}

	return s.queries.CreateDownloadTask(ctx, sqlitedb.CreateDownloadTaskParams{
		ID:              task.ID,
		Url:             task.URL,
		Path:            task.Path,
		FileName:        task.FileName,
		State:           task.State,
		DownloadedBytes: sql.NullInt64{Int64: task.DownloadedBytes, Valid: true},
		TotalBytes:      sql.NullInt64{Int64: task.TotalBytes, Valid: true},
		Etag:            toNullString(task.ETag),
		RangeSupported:  boolToNullInt64(task.RangeSupported),
		FinalUrl:        toNullString(task.FinalURL),
		TempFileName:    toNullString(task.TempFileName),
		PartsTotal:      sql.NullInt64{Int64: int64(task.PartsTotal), Valid: true},
		PartsCompleted:  sql.NullInt64{Int64: int64(task.PartsCompleted), Valid: true},
		FileType:        toNullString(task.FileType),
		SourceType:      toNullString(task.SourceType),
		RepoID:          toNullString(task.RepoID),
		ErrorMessage:    toNullString(task.ErrorMessage),
		RetryCount:      sql.NullInt64{Int64: int64(task.RetryCount), Valid: true},
		MaxRetries:      sql.NullInt64{Int64: int64(task.MaxRetries), Valid: true},
		CreatedAt:       task.CreatedAt.Unix(),
		StartedAt:       sql.NullInt64{Int64: task.StartedAt.Unix(), Valid: true},
		FinishedAt:      sql.NullInt64{Int64: task.FinishedAt.Unix(), Valid: true},
	})
}

func (s *SQLiteStore) GetDownloadTask(ctx context.Context, id string) (*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, err := s.queries.GetDownloadTask(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrDownloadTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get download task: %w", err)
	}

	return downloadTaskFromRow(row), nil
}

func (s *SQLiteStore) ListDownloadTasks(ctx context.Context, limit, offset int) ([]*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListDownloadTasks(ctx, sqlitedb.ListDownloadTasksParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list download tasks: %w", err)
	}

	tasks := make([]*DownloadTask, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, downloadTaskFromRow(row))
	}
	return tasks, nil
}

func (s *SQLiteStore) UpdateDownloadTask(ctx context.Context, task *DownloadTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, `
		UPDATE download_tasks
		SET url = ?, path = ?, file_name = ?, state = ?, downloaded_bytes = ?, total_bytes = ?,
			etag = ?, range_supported = ?, final_url = ?, temp_file_name = ?,
			parts_total = ?, parts_completed = ?, file_type = ?, source_type = ?,
			repo_id = ?, error_message = ?, retry_count = ?, max_retries = ?,
			started_at = ?, finished_at = ?
		WHERE id = ?
	`,
		task.URL, task.Path, task.FileName, task.State,
		task.DownloadedBytes, task.TotalBytes, task.ETag,
		boolToInt(task.RangeSupported), task.FinalURL, task.TempFileName,
		task.PartsTotal, task.PartsCompleted, task.FileType, task.SourceType,
		task.RepoID, task.ErrorMessage, task.RetryCount, task.MaxRetries,
		task.StartedAt.Unix(), task.FinishedAt.Unix(), task.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update download task: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDownloadTaskNotFound
	}
	return nil
}

func (s *SQLiteStore) DeleteDownloadTask(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, "DELETE FROM download_tasks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete download task: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDownloadTaskNotFound
	}
	return nil
}

func (s *SQLiteStore) ListActiveDownloadTasks(ctx context.Context) ([]*DownloadTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queries.ListActiveDownloadTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active download tasks: %w", err)
	}

	tasks := make([]*DownloadTask, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, downloadTaskFromRow(row))
	}
	return tasks, nil
}

// --- Close and Stats ---

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

	var convCount, msgCount int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&convCount); err != nil {
		return nil, fmt.Errorf("获取会话计数失败: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&msgCount); err != nil {
		return nil, fmt.Errorf("获取消息计数失败: %w", err)
	}

	stats["conversations"] = convCount
	stats["messages"] = msgCount
	stats["type"] = "sqlite"
	stats["path"] = s.path

	if info, err := os.Stat(s.path); err == nil {
		stats["size_bytes"] = info.Size()
	}

	return stats, nil
}

// --- Helper functions ---

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

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func boolToNullInt64(b bool) sql.NullInt64 {
	if b {
		return sql.NullInt64{Int64: 1, Valid: true}
	}
	return sql.NullInt64{Int64: 0, Valid: true}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- Row conversion helpers ---

func benchmarkFromRow(row sqlitedb.Benchmark) *Benchmark {
	b := &Benchmark{
		ID:        row.ID,
		ModelID:   row.ModelID,
		ModelName: row.ModelName,
		Status:    row.Status,
		Command:   row.Command.String,
		Error:     row.Error.String,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		StartedAt: unixToTime(row.StartedAt),
		FinishedAt: unixToTime(row.FinishedAt),
	}

	if row.Config.Valid && row.Config.String != "" {
		if err := json.Unmarshal([]byte(row.Config.String), &b.Config); err != nil {
			b.Config = make(map[string]interface{})
		}
	}
	if row.Metrics.Valid && row.Metrics.String != "" {
		if err := json.Unmarshal([]byte(row.Metrics.String), &b.Metrics); err != nil {
			b.Metrics = make(map[string]interface{})
		}
	}

	return b
}

func benchmarkConfigFromRow(row sqlitedb.BenchmarkConfig) *BenchmarkConfig {
	c := &BenchmarkConfig{
		Name:         row.Name,
		ModelID:      row.ModelID,
		ModelName:    row.ModelName,
		LlamaCppPath: row.LlamacppPath,
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
	}

	if row.Devices.Valid && row.Devices.String != "" {
		if err := json.Unmarshal([]byte(row.Devices.String), &c.Devices); err != nil {
			c.Devices = []string{}
		}
	}
	if row.Params.Valid && row.Params.String != "" {
		if err := json.Unmarshal([]byte(row.Params.String), &c.Params); err != nil {
			c.Params = make(map[string]string)
		}
	}

	return c
}

func modelLoadConfigFromRow(row sqlitedb.ModelLoadConfig) (*ModelLoadConfig, error) {
	c := &ModelLoadConfig{
		ID:        row.ID,
		NodeID:    row.NodeID,
		ModelID:   row.ModelID,
		ModelName: row.ModelName,
		Name:      row.Name,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt: time.Unix(row.UpdatedAt, 0).UTC(),
	}

	if row.Config != "" {
		if err := json.Unmarshal([]byte(row.Config), &c.Config); err != nil {
			c.Config = make(map[string]interface{})
		}
	}

	return c, nil
}

func launchProfileFromRow(row sqlitedb.LaunchProfile) (*LaunchProfile, error) {
	p := &LaunchProfile{
		ID:             row.ID,
		Name:           row.Name,
		BackendType:    row.BackendType,
		InstallationID: row.InstallationID.String,
		ModelScope:     row.ModelScope.String,
		ExtraArgs:      row.ExtraArgs.String,
		CreatedAt:      time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:      time.Unix(row.UpdatedAt, 0).UTC(),
	}

	if row.Params != "" {
		if err := json.Unmarshal([]byte(row.Params), &p.Params); err != nil {
			return nil, err
		}
	}
	if row.Env.Valid && row.Env.String != "" {
		if err := json.Unmarshal([]byte(row.Env.String), &p.Env); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func modelMetadataFromRow(row sqlitedb.ModelMetadatum) *ModelMetadata {
	m := &ModelMetadata{
		ModelID:     row.ModelID,
		NodeID:      row.NodeID.String,
		StoragePath: row.StoragePath.String,
		Alias:       row.Alias.String,
		Favourite:   row.Favourite.Valid && row.Favourite.Int64 != 0,
		Description: row.Description.String,
		LoadCount:   int(row.LoadCount.Int64),
		TotalTokens: row.TotalTokens.Int64,
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:   time.Unix(row.UpdatedAt, 0).UTC(),
		LastLoaded:  unixToTime(row.LastLoaded),
	}

	if row.Tags.Valid && row.Tags.String != "" {
		if err := json.Unmarshal([]byte(row.Tags.String), &m.Tags); err != nil {
			m.Tags = []string{}
		}
	}
	if row.Capabilities.Valid && row.Capabilities.String != "" {
		if err := json.Unmarshal([]byte(row.Capabilities.String), &m.Capabilities); err != nil {
			m.Capabilities = &Capabilities{}
		}
	}

	return m
}

func ttsHistoryFromRow(row sqlitedb.TtsHistory) *TTSHistoryItem {
	item := &TTSHistoryItem{
		ID:        row.ID,
		Model:     row.Model,
		InputText: row.InputText,
		AudioPath: row.AudioPath,
		Format:    row.Format,
		Duration:  row.Duration.Float64,
		Favourite: row.Favourite.Valid && row.Favourite.Int64 != 0,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}

	if row.Params.Valid && row.Params.String != "" {
		if err := json.Unmarshal([]byte(row.Params.String), &item.Params); err != nil {
			item.Params = make(map[string]interface{})
		}
	}

	return item
}

func downloadTaskFromRow(row sqlitedb.DownloadTask) *DownloadTask {
	task := &DownloadTask{
		ID:              row.ID,
		URL:             row.Url,
		Path:            row.Path,
		FileName:        row.FileName,
		State:           row.State,
		DownloadedBytes: row.DownloadedBytes.Int64,
		TotalBytes:      row.TotalBytes.Int64,
		ETag:            row.Etag.String,
		RangeSupported:  row.RangeSupported.Valid && row.RangeSupported.Int64 != 0,
		FinalURL:        row.FinalUrl.String,
		TempFileName:    row.TempFileName.String,
		PartsTotal:      int(row.PartsTotal.Int64),
		PartsCompleted:  int(row.PartsCompleted.Int64),
		FileType:        row.FileType.String,
		SourceType:      row.SourceType.String,
		RepoID:          row.RepoID.String,
		ErrorMessage:    row.ErrorMessage.String,
		RetryCount:      int(row.RetryCount.Int64),
		MaxRetries:      int(row.MaxRetries.Int64),
		CreatedAt:       time.Unix(row.CreatedAt, 0).UTC(),
		StartedAt:       time.Unix(row.StartedAt.Int64, 0).UTC(),
		FinishedAt:      time.Unix(row.FinishedAt.Int64, 0).UTC(),
	}
	return task
}
