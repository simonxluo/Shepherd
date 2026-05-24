// Package storage provides SQLite storage implementation
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
	"github.com/simonxluo/Shepherd/internal/infra/storage/sqlcgen"
	_ "modernc.org/sqlite" // Use modernc.org/sqlite for pure Go SQLite (CGO-free)
)

// SQLiteStore implements Store interface with SQLite backend
type SQLiteStore struct {
	db      *sql.DB
	path    string
	mu      sync.RWMutex
	queries *sqlcgen.Queries
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

	store := &SQLiteStore{
		db:      db,
		path:    config.Path,
		queries: sqlcgen.New(db),
	}

	// Initialize schema
	if err := store.initSchema(config); err != nil {
		utils.CloseQuietly(db)
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
		name TEXT NOT NULL DEFAULT ''
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
		capabilities TEXT,
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

	CREATE TABLE IF NOT EXISTS tts_history (
		id TEXT PRIMARY KEY,
		model TEXT NOT NULL,
		input_text TEXT NOT NULL,
		audio_path TEXT NOT NULL,
		format TEXT NOT NULL,
		duration REAL DEFAULT 0,
		favourite INTEGER DEFAULT 0,
		params TEXT,
		created_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_tts_history_created ON tts_history(created_at);
	CREATE INDEX IF NOT EXISTS idx_tts_history_favourite ON tts_history(favourite);
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

	// Database migration: add capabilities column if not exists
	if err := s.migrateCapabilitiesColumn(); err != nil {
		return fmt.Errorf("failed to migrate capabilities column: %w", err)
	}

	// Database migration: add name column to model_load_configs for multi-preset support
	if err := s.migrateModelLoadConfigsTable(); err != nil {
		return fmt.Errorf("failed to migrate model_load_configs table: %w", err)
	}

	// Database migration: ensure tts_history table exists for existing databases
	if err := s.migrateTTSHistoryTable(); err != nil {
		return fmt.Errorf("failed to migrate tts_history table: %w", err)
	}

	// Database migration: ensure download_tasks table exists
	if err := s.migrateDownloadTasksTable(); err != nil {
		return fmt.Errorf("failed to migrate download_tasks table: %w", err)
	}

	return nil
}

// migrateCapabilitiesColumn adds capabilities column to existing databases
func (s *SQLiteStore) migrateCapabilitiesColumn() error {
	capabilitiesExists := false
	rows, err := s.db.Query("PRAGMA table_info(model_metadata)")
	if err != nil {
		return err
	}
	defer utils.CloseQuietly(rows)

	for rows.Next() {
		var cid int
		var name, colType string
		var notnull, pk int
		var dflt_value interface{}

		err := rows.Scan(&cid, &name, &colType, &notnull, &dflt_value, &pk)
		if err != nil {
			return err
		}

		if name == "capabilities" {
			capabilitiesExists = true
			rows.Close()
			break
		}
	}

	if !capabilitiesExists {
		_, err := s.db.Exec("ALTER TABLE model_metadata ADD COLUMN capabilities TEXT")
		if err != nil {
			return fmt.Errorf("failed to add capabilities column: %w", err)
		}
	}

	return nil
}

// migrateModelLoadConfigsTable adds name column and unique index for multi-preset support.
func (s *SQLiteStore) migrateModelLoadConfigsTable() error {
	var hasLegacyConstraint bool
	var sql string
	err := s.db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='model_load_configs'").Scan(&sql)
	if err != nil {
		return fmt.Errorf("failed to check table schema: %w", err)
	}
	hasLegacyConstraint = strings.Contains(sql, "UNIQUE(node_id, model_id)")

	if hasLegacyConstraint {
		steps := []string{
			"ALTER TABLE model_load_configs RENAME TO model_load_configs_old",
			`CREATE TABLE model_load_configs (
				id TEXT PRIMARY KEY,
				node_id TEXT NOT NULL,
				model_id TEXT NOT NULL,
				model_name TEXT NOT NULL,
				config TEXT NOT NULL,
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,
				name TEXT NOT NULL DEFAULT ''
			)`,
			"INSERT INTO model_load_configs SELECT id, node_id, model_id, model_name, config, created_at, updated_at, '' FROM model_load_configs_old",
			"DROP TABLE model_load_configs_old",
		}
		for _, stmt := range steps {
			if _, err := s.db.Exec(stmt); err != nil {
				return fmt.Errorf("failed to rebuild model_load_configs table (%s): %w", stmt, err)
			}
		}
	} else {
		nameExists := false
		rows, err := s.db.Query("PRAGMA table_info(model_load_configs)")
		if err != nil {
			return err
		}
		defer utils.CloseQuietly(rows)

		for rows.Next() {
			var cid int
			var name, colType string
			var notnull, pk int
			var dflt_value interface{}
			if err := rows.Scan(&cid, &name, &colType, &notnull, &dflt_value, &pk); err != nil {
				return err
			}
			if name == "name" {
				nameExists = true
				rows.Close()
				break
			}
		}

		if !nameExists {
			if _, err := s.db.Exec("ALTER TABLE model_load_configs ADD COLUMN name TEXT NOT NULL DEFAULT ''"); err != nil {
				return fmt.Errorf("failed to add name column: %w", err)
			}
		}
	}

	if _, err := s.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_mlc_node_model_name ON model_load_configs(node_id, model_id, name)"); err != nil {
		return fmt.Errorf("failed to create new index: %w", err)
	}

	s.db.Exec("DROP INDEX IF EXISTS idx_model_load_configs_node_model")

	return nil
}

// migrateTTSHistoryTable ensures the tts_history table exists for databases created before this feature
func (s *SQLiteStore) migrateTTSHistoryTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS tts_history (
		id TEXT PRIMARY KEY,
		model TEXT NOT NULL,
		input_text TEXT NOT NULL,
		audio_path TEXT NOT NULL,
		format TEXT NOT NULL,
		duration REAL DEFAULT 0,
		favourite INTEGER DEFAULT 0,
		params TEXT,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_tts_history_created ON tts_history(created_at);
	CREATE INDEX IF NOT EXISTS idx_tts_history_favourite ON tts_history(favourite);
	`
	_, err := s.db.Exec(query)
	return err
}

// migrateDownloadTasksTable ensures the download_tasks table exists for databases created before this feature
func (s *SQLiteStore) migrateDownloadTasksTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS download_tasks (
		id TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		path TEXT NOT NULL,
		file_name TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL DEFAULT 'idle',
		downloaded_bytes INTEGER DEFAULT 0,
		total_bytes INTEGER DEFAULT 0,
		etag TEXT DEFAULT '',
		range_supported INTEGER DEFAULT 0,
		final_url TEXT DEFAULT '',
		temp_file_name TEXT DEFAULT '',
		parts_total INTEGER DEFAULT 0,
		parts_completed INTEGER DEFAULT 0,
		file_type TEXT DEFAULT '',
		source_type TEXT DEFAULT '',
		repo_id TEXT DEFAULT '',
		error_message TEXT DEFAULT '',
		retry_count INTEGER DEFAULT 0,
		max_retries INTEGER DEFAULT 5,
		created_at INTEGER NOT NULL,
		started_at INTEGER DEFAULT 0,
		finished_at INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_download_tasks_state ON download_tasks(state);
	CREATE INDEX IF NOT EXISTS idx_download_tasks_created ON download_tasks(created_at);
	`
	_, err := s.db.Exec(query)
	return err
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
