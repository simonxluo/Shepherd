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

	"github.com/simonxluo/Shepherd/internal/comm/storage/sqlcgen"
	"github.com/simonxluo/Shepherd/internal/comm/utils"
	_ "modernc.org/sqlite" // Use modernc.org/sqlite for pure Go SQLite (CGO-free)
)

// SQLiteStore implements Store interface with SQLite backend
type SQLiteStore struct {
	db      *sql.DB
	queries *sqlcgen.Queries
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
		queries: sqlcgen.New(db),
		path:    config.Path,
	}

	return store, nil
}

// LaunchProfile operations

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

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO launch_profiles (id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, profile.ID, profile.Name, profile.BackendType, toNullString(profile.InstallationID),
		toNullString(profile.ModelScope), string(paramsJSON), toNullString(string(envJSON)),
		toNullString(profile.ExtraArgs), profile.CreatedAt.Unix(), profile.UpdatedAt.Unix())
	return err
}

func (s *SQLiteStore) GetLaunchProfile(ctx context.Context, id string) (*LaunchProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at
		FROM launch_profiles WHERE id = ?
	`, id)

	var p LaunchProfile
	var installationID, modelScope, env, extraArgs sql.NullString
	var paramsStr string
	var createdAt, updatedAt int64

	err := row.Scan(&p.ID, &p.Name, &p.BackendType, &installationID, &modelScope,
		&paramsStr, &env, &extraArgs, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrModelLoadConfigNotFound
	}
	if err != nil {
		return nil, err
	}

	p.InstallationID = installationID.String
	p.ModelScope = modelScope.String
	p.ExtraArgs = extraArgs.String
	p.CreatedAt = time.Unix(createdAt, 0).UTC()
	p.UpdatedAt = time.Unix(updatedAt, 0).UTC()

	if paramsStr != "" {
		if err := json.Unmarshal([]byte(paramsStr), &p.Params); err != nil {
			return nil, err
		}
	}
	if env.Valid && env.String != "" {
		if err := json.Unmarshal([]byte(env.String), &p.Env); err != nil {
			return nil, err
		}
	}

	return &p, nil
}

func (s *SQLiteStore) ListLaunchProfiles(ctx context.Context, backendType, modelScope string) ([]*LaunchProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at
		FROM launch_profiles
		WHERE (? = '' OR backend_type = ?)
		  AND (? = '' OR model_scope = '' OR model_scope = ?)
		ORDER BY name ASC
	`, backendType, backendType, modelScope, modelScope)
	if err != nil {
		return nil, err
	}
	defer utils.CloseQuietly(rows)

	var profiles []*LaunchProfile
	for rows.Next() {
		var p LaunchProfile
		var installationID, modelScopeVal, env, extraArgs sql.NullString
		var paramsStr string
		var createdAt, updatedAt int64

		if err := rows.Scan(&p.ID, &p.Name, &p.BackendType, &installationID, &modelScopeVal,
			&paramsStr, &env, &extraArgs, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		p.InstallationID = installationID.String
		p.ModelScope = modelScopeVal.String
		p.ExtraArgs = extraArgs.String
		p.CreatedAt = time.Unix(createdAt, 0).UTC()
		p.UpdatedAt = time.Unix(updatedAt, 0).UTC()

		if paramsStr != "" {
			if err := json.Unmarshal([]byte(paramsStr), &p.Params); err != nil {
				return nil, err
			}
		}
		if env.Valid && env.String != "" {
			if err := json.Unmarshal([]byte(env.String), &p.Env); err != nil {
				return nil, err
			}
		}

		profiles = append(profiles, &p)
	}
	return profiles, rows.Err()
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

// Close and Stats

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
