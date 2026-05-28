package storage

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

//go:embed migrations/mysql/*.sql
var mysqlMigrations embed.FS

// RunMigrations runs database migrations for the given engine type.
// For existing databases (tables already present but no goose version table),
// the initial migration uses CREATE TABLE IF NOT EXISTS ensuring idempotency.
func RunMigrations(db *sql.DB, engine StorageType) error {
	var dir string
	var dialect string

	switch engine {
	case StorageTypeSQLite:
		goose.SetBaseFS(sqliteMigrations)
		dir = "migrations/sqlite"
		dialect = "sqlite3"
	case StorageTypePostgreSQL:
		goose.SetBaseFS(postgresMigrations)
		dir = "migrations/postgres"
		dialect = "postgres"
	case StorageTypeMySQL:
		goose.SetBaseFS(mysqlMigrations)
		dir = "migrations/mysql"
		dialect = "mysql"
	default:
		return fmt.Errorf("migrations not supported for engine: %s", engine)
	}

	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
