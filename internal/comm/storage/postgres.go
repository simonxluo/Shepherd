// Package storage provides PostgreSQL storage implementation
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	pgdb "github.com/simonxluo/Shepherd/internal/comm/storage/sqlc/postgres/db"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgreSQLStore implements Store interface with PostgreSQL backend
type PostgreSQLStore struct {
	db      *sql.DB
	queries *pgdb.Queries
}

// NewPostgreSQLStore creates a new PostgreSQL store
func NewPostgreSQLStore(config *PostgreSQLConfig) (*PostgreSQLStore, error) {
	if config == nil {
		return nil, ErrMissingPostgreSQLConfig
	}

	port := config.Port
	if port == 0 {
		port = 5432
	}

	sslMode := config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.Username, config.Password, config.Host, port, config.Database, sslMode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		utils.CloseQuietly(db)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := RunMigrations(db, StorageTypePostgreSQL); err != nil {
		utils.CloseQuietly(db)
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	store := &PostgreSQLStore{
		db:      db,
		queries: pgdb.New(db),
	}

	return store, nil
}

// --- Conversation operations ---

func (s *PostgreSQLStore) CreateConversation(ctx context.Context, conv *Conversation) error {
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

	return s.queries.CreateConversation(ctx, pgdb.CreateConversationParams{
		ID:           conv.ID,
		Model:        conv.Model,
		Title:        toNullString(conv.Title),
		SystemPrompt: toNullString(conv.SystemPrompt),
		MessageCount: sql.NullInt32{Int32: int32(conv.MessageCount), Valid: true},
		CreatedAt:    conv.CreatedAt.Unix(),
		UpdatedAt:    conv.UpdatedAt.Unix(),
		Metadata:     toNullString(string(metadataJSON)),
	})
}

func (s *PostgreSQLStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
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
		MessageCount: int(row.MessageCount.Int32),
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

func (s *PostgreSQLStore) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	rows, err := s.queries.ListConversations(ctx, pgdb.ListConversationsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
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
			MessageCount: int(row.MessageCount.Int32),
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

func (s *PostgreSQLStore) UpdateConversation(ctx context.Context, conv *Conversation) error {
	conv.UpdatedAt = timeNow()
	metadataJSON, _ := json.Marshal(conv.Metadata)

	result, err := s.db.ExecContext(ctx, `
		UPDATE conversations
		SET model = $1, title = $2, system_prompt = $3, message_count = $4, updated_at = $5, metadata = $6
		WHERE id = $7
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

func (s *PostgreSQLStore) DeleteConversation(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM conversations WHERE id = $1", id)
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

func (s *PostgreSQLStore) CreateMessage(ctx context.Context, msg *Message) error {
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

	err = qtx.CreateMessage(ctx, pgdb.CreateMessageParams{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		Role:           msg.Role,
		Content:        msg.Content,
		Name:           toNullString(msg.Name),
		TokenCount:     sql.NullInt32{Int32: int32(msg.TokenCount), Valid: true},
		CreatedAt:      msg.CreatedAt.Unix(),
		Metadata:       toNullString(string(metadataJSON)),
	})
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	err = qtx.IncrementMessageCount(ctx, pgdb.IncrementMessageCountParams{
		UpdatedAt: timeNow().Unix(),
		ID:        msg.ConversationID,
	})
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	return tx.Commit()
}

func (s *PostgreSQLStore) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	rows, err := s.queries.GetMessages(ctx, pgdb.GetMessagesParams{
		ConversationID: conversationID,
		Limit:          int32(limit),
		Offset:         int32(offset),
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
			TokenCount:     int(row.TokenCount.Int32),
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

func (s *PostgreSQLStore) DeleteMessages(ctx context.Context, conversationID string) error {
	if err := s.queries.DeleteMessages(ctx, conversationID); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return s.queries.ResetMessageCount(ctx, pgdb.ResetMessageCountParams{
		UpdatedAt: timeNow().Unix(),
		ID:        conversationID,
	})
}

// --- Benchmark operations ---

func (s *PostgreSQLStore) CreateBenchmark(ctx context.Context, benchmark *Benchmark) error {
	configJSON, err := json.Marshal(benchmark.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark config: %w", err)
	}
	metricsJSON, err := json.Marshal(benchmark.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark metrics: %w", err)
	}

	return s.queries.CreateBenchmark(ctx, pgdb.CreateBenchmarkParams{
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

func (s *PostgreSQLStore) GetBenchmark(ctx context.Context, id string) (*Benchmark, error) {
	row, err := s.queries.GetBenchmark(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrBenchmarkNotFound
	}
	if err != nil {
		return nil, err
	}

	return pgBenchmarkFromRow(row), nil
}

func (s *PostgreSQLStore) ListBenchmarks(ctx context.Context, modelID string, limit, offset int) ([]*Benchmark, error) {
	if modelID != "" {
		rows, err := s.queries.ListBenchmarksByModel(ctx, pgdb.ListBenchmarksByModelParams{
			ModelID: modelID,
			Limit:   int32(limit),
			Offset:  int32(offset),
		})
		if err != nil {
			return nil, err
		}
		benchmarks := make([]*Benchmark, 0, len(rows))
		for _, row := range rows {
			benchmarks = append(benchmarks, pgBenchmarkFromRow(row))
		}
		return benchmarks, nil
	}

	rows, err := s.queries.ListBenchmarks(ctx, pgdb.ListBenchmarksParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	benchmarks := make([]*Benchmark, 0, len(rows))
	for _, row := range rows {
		benchmarks = append(benchmarks, pgBenchmarkFromRow(row))
	}
	return benchmarks, nil
}

func (s *PostgreSQLStore) UpdateBenchmark(ctx context.Context, benchmark *Benchmark) error {
	configJSON, err := json.Marshal(benchmark.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark config: %w", err)
	}
	metricsJSON, err := json.Marshal(benchmark.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark metrics: %w", err)
	}

	return s.queries.UpdateBenchmark(ctx, pgdb.UpdateBenchmarkParams{
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

func (s *PostgreSQLStore) DeleteBenchmark(ctx context.Context, id string) error {
	return s.queries.DeleteBenchmark(ctx, id)
}

// --- BenchmarkConfig operations ---

func (s *PostgreSQLStore) CreateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	devicesJSON, _ := json.Marshal(config.Devices)
	paramsJSON, _ := json.Marshal(config.Params)

	return s.queries.CreateBenchmarkConfig(ctx, pgdb.CreateBenchmarkConfigParams{
		Name:         config.Name,
		ModelID:      config.ModelID,
		ModelName:    config.ModelName,
		LlamacppPath: config.LlamaCppPath,
		Devices:      toNullString(string(devicesJSON)),
		Params:       toNullString(string(paramsJSON)),
		CreatedAt:    config.CreatedAt.Unix(),
	})
}

func (s *PostgreSQLStore) GetBenchmarkConfig(ctx context.Context, name string) (*BenchmarkConfig, error) {
	row, err := s.queries.GetBenchmarkConfig(ctx, name)
	if err == sql.ErrNoRows {
		return nil, ErrBenchmarkConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	return pgBenchmarkConfigFromRow(row), nil
}

func (s *PostgreSQLStore) ListBenchmarkConfigs(ctx context.Context, limit, offset int) ([]*BenchmarkConfig, error) {
	rows, err := s.queries.ListBenchmarkConfigs(ctx, pgdb.ListBenchmarkConfigsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	configs := make([]*BenchmarkConfig, 0, len(rows))
	for _, row := range rows {
		configs = append(configs, pgBenchmarkConfigFromRow(row))
	}
	return configs, nil
}

func (s *PostgreSQLStore) UpdateBenchmarkConfig(ctx context.Context, config *BenchmarkConfig) error {
	devicesJSON, _ := json.Marshal(config.Devices)
	paramsJSON, _ := json.Marshal(config.Params)

	return s.queries.UpdateBenchmarkConfig(ctx, pgdb.UpdateBenchmarkConfigParams{
		ModelID:      config.ModelID,
		ModelName:    config.ModelName,
		LlamacppPath: config.LlamaCppPath,
		Devices:      toNullString(string(devicesJSON)),
		Params:       toNullString(string(paramsJSON)),
		Name:         config.Name,
	})
}

func (s *PostgreSQLStore) DeleteBenchmarkConfig(ctx context.Context, name string) error {
	return s.queries.DeleteBenchmarkConfig(ctx, name)
}

// --- ModelLoadConfig operations ---

func (s *PostgreSQLStore) SaveModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
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

	return s.queries.UpsertModelLoadConfig(ctx, pgdb.UpsertModelLoadConfigParams{
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

func (s *PostgreSQLStore) GetModelLoadConfig(ctx context.Context, nodeID, modelID string) (*ModelLoadConfig, error) {
	row, err := s.queries.GetModelLoadConfig(ctx, pgdb.GetModelLoadConfigParams{
		NodeID:  nodeID,
		ModelID: modelID,
	})
	if err == sql.ErrNoRows {
		return nil, ErrModelLoadConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model load config: %w", err)
	}

	return pgModelLoadConfigFromRow(row)
}

func (s *PostgreSQLStore) DeleteModelLoadConfig(ctx context.Context, nodeID, modelID string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM model_load_configs WHERE node_id = $1 AND model_id = $2 AND name = ''",
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

func (s *PostgreSQLStore) ListModelLoadConfigs(ctx context.Context, nodeID, modelID string) ([]*ModelLoadConfig, error) {
	rows, err := s.queries.ListModelLoadConfigs(ctx, pgdb.ListModelLoadConfigsParams{
		NodeID:  nodeID,
		ModelID: modelID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list model load configs: %w", err)
	}

	configs := make([]*ModelLoadConfig, 0, len(rows))
	for _, row := range rows {
		c, err := pgModelLoadConfigFromRow(row)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, nil
}

func (s *PostgreSQLStore) SaveNamedModelLoadConfig(ctx context.Context, config *ModelLoadConfig) error {
	if config.Name == "" {
		return fmt.Errorf("named config requires a non-empty name")
	}

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

	return s.queries.UpsertModelLoadConfig(ctx, pgdb.UpsertModelLoadConfigParams{
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

func (s *PostgreSQLStore) DeleteNamedModelLoadConfig(ctx context.Context, nodeID, modelID, name string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM model_load_configs WHERE node_id = $1 AND model_id = $2 AND name = $3",
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

func (s *PostgreSQLStore) CreateLaunchProfile(ctx context.Context, profile *LaunchProfile) error {
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

	return s.queries.CreateLaunchProfile(ctx, pgdb.CreateLaunchProfileParams{
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

func (s *PostgreSQLStore) GetLaunchProfile(ctx context.Context, id string) (*LaunchProfile, error) {
	row, err := s.queries.GetLaunchProfile(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrModelLoadConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	return pgLaunchProfileFromRow(row)
}

func (s *PostgreSQLStore) ListLaunchProfiles(ctx context.Context, backendType, modelScope string) ([]*LaunchProfile, error) {
	rows, err := s.queries.ListLaunchProfiles(ctx, pgdb.ListLaunchProfilesParams{
		BackendTypeFilter: backendType,
		ModelScopeFilter:  modelScope,
	})
	if err != nil {
		return nil, err
	}

	profiles := make([]*LaunchProfile, 0, len(rows))
	for _, row := range rows {
		p, err := pgLaunchProfileFromRow(row)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (s *PostgreSQLStore) UpdateLaunchProfile(ctx context.Context, profile *LaunchProfile) error {
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
		SET name = $1, backend_type = $2, installation_id = $3, model_scope = $4, params = $5, env = $6, extra_args = $7, updated_at = $8
		WHERE id = $9
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

func (s *PostgreSQLStore) DeleteLaunchProfile(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM launch_profiles WHERE id = $1", id)
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

func (s *PostgreSQLStore) SaveModelMetadata(ctx context.Context, metadata *ModelMetadata) error {
	now := timeNow()

	existingTime, err := s.queries.GetModelMetadataCreatedAt(ctx, metadata.ModelID)
	if err == sql.ErrNoRows {
		// Insert new record
		if metadata.CreatedAt.IsZero() {
			metadata.CreatedAt = now
		}
		metadata.UpdatedAt = now

		tagsJSON, _ := json.Marshal(metadata.Tags)
		capsJSON, _ := json.Marshal(metadata.Capabilities)

		return s.queries.InsertModelMetadata(ctx, pgdb.InsertModelMetadataParams{
			ModelID:      metadata.ModelID,
			NodeID:       toNullString(metadata.NodeID),
			StoragePath:  toNullString(metadata.StoragePath),
			Alias:        toNullString(metadata.Alias),
			Favourite:    boolToNullInt32(metadata.Favourite),
			Tags:         toNullString(string(tagsJSON)),
			Description:  toNullString(metadata.Description),
			LoadCount:    sql.NullInt32{Int32: int32(metadata.LoadCount), Valid: true},
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

	return s.queries.UpdateModelMetadata(ctx, pgdb.UpdateModelMetadataParams{
		NodeID:       toNullString(metadata.NodeID),
		StoragePath:  toNullString(metadata.StoragePath),
		Alias:        toNullString(metadata.Alias),
		Favourite:    boolToNullInt32(metadata.Favourite),
		Tags:         toNullString(string(tagsJSON)),
		Description:  toNullString(metadata.Description),
		LoadCount:    sql.NullInt32{Int32: int32(metadata.LoadCount), Valid: true},
		LastLoaded:   timeToUnix(metadata.LastLoaded),
		TotalTokens:  sql.NullInt64{Int64: metadata.TotalTokens, Valid: true},
		Capabilities: toNullString(string(capsJSON)),
		UpdatedAt:    metadata.UpdatedAt.Unix(),
		ModelID:      metadata.ModelID,
	})
}

func (s *PostgreSQLStore) GetModelMetadata(ctx context.Context, modelID string) (*ModelMetadata, error) {
	row, err := s.queries.GetModelMetadata(ctx, modelID)
	if err == sql.ErrNoRows {
		return nil, ErrModelMetadataNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model metadata: %w", err)
	}

	return pgModelMetadataFromRow(row), nil
}

func (s *PostgreSQLStore) ListModelMetadata(ctx context.Context, limit, offset int) ([]*ModelMetadata, error) {
	rows, err := s.queries.ListModelMetadata(ctx, pgdb.ListModelMetadataParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list model metadata: %w", err)
	}

	metadatas := make([]*ModelMetadata, 0, len(rows))
	for _, row := range rows {
		metadatas = append(metadatas, pgModelMetadataFromRow(row))
	}
	return metadatas, nil
}

func (s *PostgreSQLStore) DeleteModelMetadata(ctx context.Context, modelID string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM model_metadata WHERE model_id = $1", modelID)
	if err != nil {
		return fmt.Errorf("failed to delete model metadata: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrModelMetadataNotFound
	}
	return nil
}

func (s *PostgreSQLStore) GetAllModelMetadata(ctx context.Context) (map[string]*ModelMetadata, error) {
	rows, err := s.queries.GetAllModelMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all model metadata: %w", err)
	}

	result := make(map[string]*ModelMetadata, len(rows))
	for _, row := range rows {
		m := pgModelMetadataFromRow(row)
		result[m.ModelID] = m
	}
	return result, nil
}

// --- TTS History operations ---

func (s *PostgreSQLStore) CreateTTSHistory(ctx context.Context, item *TTSHistoryItem) error {
	if item.ID == "" {
		item.ID = generateID("tts")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = timeNow()
	}

	paramsJSON, _ := json.Marshal(item.Params)

	return s.queries.CreateTTSHistory(ctx, pgdb.CreateTTSHistoryParams{
		ID:        item.ID,
		Model:     item.Model,
		InputText: item.InputText,
		AudioPath: item.AudioPath,
		Format:    item.Format,
		Duration:  sql.NullFloat64{Float64: item.Duration, Valid: true},
		Favourite: boolToNullInt32(item.Favourite),
		Params:    toNullString(string(paramsJSON)),
		CreatedAt: item.CreatedAt.Unix(),
	})
}

func (s *PostgreSQLStore) GetTTSHistory(ctx context.Context, id string) (*TTSHistoryItem, error) {
	row, err := s.queries.GetTTSHistory(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrTTSHistoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get TTS history: %w", err)
	}

	return pgTTSHistoryFromRow(row), nil
}

func (s *PostgreSQLStore) ListTTSHistory(ctx context.Context, limit, offset int, favouriteOnly *bool) ([]*TTSHistoryItem, error) {
	if favouriteOnly != nil && *favouriteOnly {
		rows, err := s.queries.ListTTSHistoryFavourite(ctx, pgdb.ListTTSHistoryFavouriteParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list TTS history: %w", err)
		}
		items := make([]*TTSHistoryItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, pgTTSHistoryFromRow(row))
		}
		return items, nil
	}

	rows, err := s.queries.ListTTSHistory(ctx, pgdb.ListTTSHistoryParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list TTS history: %w", err)
	}
	items := make([]*TTSHistoryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, pgTTSHistoryFromRow(row))
	}
	return items, nil
}

func (s *PostgreSQLStore) UpdateTTSHistoryFavourite(ctx context.Context, id string, favourite bool) error {
	result, err := s.db.ExecContext(ctx, "UPDATE tts_history SET favourite = $1 WHERE id = $2", boolToInt(favourite), id)
	if err != nil {
		return fmt.Errorf("failed to update TTS history favourite: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrTTSHistoryNotFound
	}
	return nil
}

func (s *PostgreSQLStore) DeleteTTSHistory(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM tts_history WHERE id = $1", id)
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

func (s *PostgreSQLStore) CreateDownloadTask(ctx context.Context, task *DownloadTask) error {
	if task.ID == "" {
		task.ID = generateID("dl")
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = timeNow()
	}

	return s.queries.CreateDownloadTask(ctx, pgdb.CreateDownloadTaskParams{
		ID:              task.ID,
		Url:             task.URL,
		Path:            task.Path,
		FileName:        task.FileName,
		State:           task.State,
		DownloadedBytes: sql.NullInt64{Int64: task.DownloadedBytes, Valid: true},
		TotalBytes:      sql.NullInt64{Int64: task.TotalBytes, Valid: true},
		Etag:            toNullString(task.ETag),
		RangeSupported:  boolToNullInt32(task.RangeSupported),
		FinalUrl:        toNullString(task.FinalURL),
		TempFileName:    toNullString(task.TempFileName),
		PartsTotal:      sql.NullInt32{Int32: int32(task.PartsTotal), Valid: true},
		PartsCompleted:  sql.NullInt32{Int32: int32(task.PartsCompleted), Valid: true},
		FileType:        toNullString(task.FileType),
		SourceType:      toNullString(task.SourceType),
		RepoID:          toNullString(task.RepoID),
		ErrorMessage:    toNullString(task.ErrorMessage),
		RetryCount:      sql.NullInt32{Int32: int32(task.RetryCount), Valid: true},
		MaxRetries:      sql.NullInt32{Int32: int32(task.MaxRetries), Valid: true},
		CreatedAt:       task.CreatedAt.Unix(),
		StartedAt:       sql.NullInt64{Int64: task.StartedAt.Unix(), Valid: true},
		FinishedAt:      sql.NullInt64{Int64: task.FinishedAt.Unix(), Valid: true},
	})
}

func (s *PostgreSQLStore) GetDownloadTask(ctx context.Context, id string) (*DownloadTask, error) {
	row, err := s.queries.GetDownloadTask(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrDownloadTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get download task: %w", err)
	}

	return pgDownloadTaskFromRow(row), nil
}

func (s *PostgreSQLStore) ListDownloadTasks(ctx context.Context, limit, offset int) ([]*DownloadTask, error) {
	rows, err := s.queries.ListDownloadTasks(ctx, pgdb.ListDownloadTasksParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list download tasks: %w", err)
	}

	tasks := make([]*DownloadTask, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, pgDownloadTaskFromRow(row))
	}
	return tasks, nil
}

func (s *PostgreSQLStore) UpdateDownloadTask(ctx context.Context, task *DownloadTask) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE download_tasks
		SET url = $1, path = $2, file_name = $3, state = $4, downloaded_bytes = $5, total_bytes = $6,
			etag = $7, range_supported = $8, final_url = $9, temp_file_name = $10,
			parts_total = $11, parts_completed = $12, file_type = $13, source_type = $14,
			repo_id = $15, error_message = $16, retry_count = $17, max_retries = $18,
			started_at = $19, finished_at = $20
		WHERE id = $21
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

func (s *PostgreSQLStore) DeleteDownloadTask(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM download_tasks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete download task: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDownloadTaskNotFound
	}
	return nil
}

func (s *PostgreSQLStore) ListActiveDownloadTasks(ctx context.Context) ([]*DownloadTask, error) {
	rows, err := s.queries.ListActiveDownloadTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active download tasks: %w", err)
	}

	tasks := make([]*DownloadTask, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, pgDownloadTaskFromRow(row))
	}
	return tasks, nil
}

// --- Close and Stats ---

func (s *PostgreSQLStore) Close() error {
	return s.db.Close()
}

// Stats returns statistics about the database
func (s *PostgreSQLStore) Stats() (map[string]interface{}, error) {
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
	stats["type"] = "postgresql"

	return stats, nil
}

// --- Helper functions ---

func boolToNullInt32(b bool) sql.NullInt32 {
	if b {
		return sql.NullInt32{Int32: 1, Valid: true}
	}
	return sql.NullInt32{Int32: 0, Valid: true}
}

// --- Row conversion helpers ---

func pgBenchmarkFromRow(row pgdb.Benchmark) *Benchmark {
	b := &Benchmark{
		ID:         row.ID,
		ModelID:    row.ModelID,
		ModelName:  row.ModelName,
		Status:     row.Status,
		Command:    row.Command.String,
		Error:      row.Error.String,
		CreatedAt:  time.Unix(row.CreatedAt, 0).UTC(),
		StartedAt:  unixToTime(row.StartedAt),
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

func pgBenchmarkConfigFromRow(row pgdb.BenchmarkConfig) *BenchmarkConfig {
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

func pgModelLoadConfigFromRow(row pgdb.ModelLoadConfig) (*ModelLoadConfig, error) {
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

func pgLaunchProfileFromRow(row pgdb.LaunchProfile) (*LaunchProfile, error) {
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

func pgModelMetadataFromRow(row pgdb.ModelMetadatum) *ModelMetadata {
	m := &ModelMetadata{
		ModelID:     row.ModelID,
		NodeID:      row.NodeID.String,
		StoragePath: row.StoragePath.String,
		Alias:       row.Alias.String,
		Favourite:   row.Favourite.Valid && row.Favourite.Int32 != 0,
		Description: row.Description.String,
		LoadCount:   int(row.LoadCount.Int32),
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

func pgTTSHistoryFromRow(row pgdb.TtsHistory) *TTSHistoryItem {
	item := &TTSHistoryItem{
		ID:        row.ID,
		Model:     row.Model,
		InputText: row.InputText,
		AudioPath: row.AudioPath,
		Format:    row.Format,
		Duration:  row.Duration.Float64,
		Favourite: row.Favourite.Valid && row.Favourite.Int32 != 0,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
	}

	if row.Params.Valid && row.Params.String != "" {
		if err := json.Unmarshal([]byte(row.Params.String), &item.Params); err != nil {
			item.Params = make(map[string]interface{})
		}
	}

	return item
}

func pgDownloadTaskFromRow(row pgdb.DownloadTask) *DownloadTask {
	task := &DownloadTask{
		ID:              row.ID,
		URL:             row.Url,
		Path:            row.Path,
		FileName:        row.FileName,
		State:           row.State,
		DownloadedBytes: row.DownloadedBytes.Int64,
		TotalBytes:      row.TotalBytes.Int64,
		ETag:            row.Etag.String,
		RangeSupported:  row.RangeSupported.Valid && row.RangeSupported.Int32 != 0,
		FinalURL:        row.FinalUrl.String,
		TempFileName:    row.TempFileName.String,
		PartsTotal:      int(row.PartsTotal.Int32),
		PartsCompleted:  int(row.PartsCompleted.Int32),
		FileType:        row.FileType.String,
		SourceType:      row.SourceType.String,
		RepoID:          row.RepoID.String,
		ErrorMessage:    row.ErrorMessage.String,
		RetryCount:      int(row.RetryCount.Int32),
		MaxRetries:      int(row.MaxRetries.Int32),
		CreatedAt:       time.Unix(row.CreatedAt, 0).UTC(),
		StartedAt:       time.Unix(row.StartedAt.Int64, 0).UTC(),
		FinishedAt:      time.Unix(row.FinishedAt.Int64, 0).UTC(),
	}
	return task
}
