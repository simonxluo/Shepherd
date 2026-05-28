package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// --- SQLite MCP Server operations ---

func (s *SQLiteStore) CreateMCPServer(ctx context.Context, server *MCPServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if server.ID == "" {
		server.ID = generateID("mcp")
	}
	now := time.Now()
	if server.CreatedAt.IsZero() {
		server.CreatedAt = now
	}
	server.UpdatedAt = now

	isActive := 0
	if server.IsActive {
		isActive = 1
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO mcp_servers (id, name, description, url, type, is_active, headers, status, error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		server.ID, server.Name, server.Description, server.URL, server.Type,
		isActive, server.Headers, server.Status, server.Error,
		server.CreatedAt.Unix(), server.UpdatedAt.Unix(),
	)
	return err
}

func (s *SQLiteStore) GetMCPServer(ctx context.Context, id string) (*MCPServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, url, type, is_active, headers, status, error, created_at, updated_at
		FROM mcp_servers WHERE id = ?`, id)
	return scanMCPServerFromRow(row)
}

func (s *SQLiteStore) ListMCPServers(ctx context.Context) ([]*MCPServer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, url, type, is_active, headers, status, error, created_at, updated_at
		FROM mcp_servers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMCPServerRows(rows)
}

func (s *SQLiteStore) UpdateMCPServer(ctx context.Context, server *MCPServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	server.UpdatedAt = time.Now()
	isActive := 0
	if server.IsActive {
		isActive = 1
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE mcp_servers SET name=?, description=?, url=?, type=?, is_active=?, headers=?, status=?, error=?, updated_at=? WHERE id=?`,
		server.Name, server.Description, server.URL, server.Type,
		isActive, server.Headers, server.Status, server.Error,
		server.UpdatedAt.Unix(), server.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteMCPServer(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_servers WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) CreateMCPTool(ctx context.Context, tool *MCPTool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return createMCPToolSQL(ctx, s.db, tool, "?")
}

func (s *SQLiteStore) ListMCPToolsByServer(ctx context.Context, serverID string) ([]*MCPTool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listMCPToolsByServerSQL(ctx, s.db, serverID, "?")
}

func (s *SQLiteStore) DeleteMCPToolsByServer(ctx context.Context, serverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_tools WHERE server_id = ?`, serverID)
	return err
}

// --- PostgreSQL MCP operations ---

func (s *PostgreSQLStore) CreateMCPServer(ctx context.Context, server *MCPServer) error {
	if server.ID == "" {
		server.ID = generateID("mcp")
	}
	now := time.Now()
	if server.CreatedAt.IsZero() {
		server.CreatedAt = now
	}
	server.UpdatedAt = now

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO mcp_servers (id, name, description, url, type, is_active, headers, status, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		server.ID, server.Name, server.Description, server.URL, server.Type,
		server.IsActive, server.Headers, server.Status, server.Error,
		server.CreatedAt.Unix(), server.UpdatedAt.Unix(),
	)
	return err
}

func (s *PostgreSQLStore) GetMCPServer(ctx context.Context, id string) (*MCPServer, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, url, type, is_active, headers, status, error, created_at, updated_at
		FROM mcp_servers WHERE id = $1`, id)
	return scanMCPServerFromRow(row)
}

func (s *PostgreSQLStore) ListMCPServers(ctx context.Context) ([]*MCPServer, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, url, type, is_active, headers, status, error, created_at, updated_at
		FROM mcp_servers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMCPServerRows(rows)
}

func (s *PostgreSQLStore) UpdateMCPServer(ctx context.Context, server *MCPServer) error {
	server.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE mcp_servers SET name=$1, description=$2, url=$3, type=$4, is_active=$5, headers=$6, status=$7, error=$8, updated_at=$9 WHERE id=$10`,
		server.Name, server.Description, server.URL, server.Type,
		server.IsActive, server.Headers, server.Status, server.Error,
		server.UpdatedAt.Unix(), server.ID,
	)
	return err
}

func (s *PostgreSQLStore) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_servers WHERE id = $1`, id)
	return err
}

func (s *PostgreSQLStore) CreateMCPTool(ctx context.Context, tool *MCPTool) error {
	return createMCPToolSQL(ctx, s.db, tool, "$")
}

func (s *PostgreSQLStore) ListMCPToolsByServer(ctx context.Context, serverID string) ([]*MCPTool, error) {
	return listMCPToolsByServerSQL(ctx, s.db, serverID, "$")
}

func (s *PostgreSQLStore) DeleteMCPToolsByServer(ctx context.Context, serverID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_tools WHERE server_id = $1`, serverID)
	return err
}

// --- MySQL MCP operations ---

func (s *MySQLStore) CreateMCPServer(ctx context.Context, server *MCPServer) error {
	if server.ID == "" {
		server.ID = generateID("mcp")
	}
	now := time.Now()
	if server.CreatedAt.IsZero() {
		server.CreatedAt = now
	}
	server.UpdatedAt = now

	isActive := 0
	if server.IsActive {
		isActive = 1
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO mcp_servers (id, name, description, url, type, is_active, headers, status, error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		server.ID, server.Name, server.Description, server.URL, server.Type,
		isActive, server.Headers, server.Status, server.Error,
		server.CreatedAt.Unix(), server.UpdatedAt.Unix(),
	)
	return err
}

func (s *MySQLStore) GetMCPServer(ctx context.Context, id string) (*MCPServer, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, url, type, is_active, headers, status, error, created_at, updated_at
		FROM mcp_servers WHERE id = ?`, id)
	return scanMCPServerFromRow(row)
}

func (s *MySQLStore) ListMCPServers(ctx context.Context) ([]*MCPServer, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, url, type, is_active, headers, status, error, created_at, updated_at
		FROM mcp_servers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMCPServerRows(rows)
}

func (s *MySQLStore) UpdateMCPServer(ctx context.Context, server *MCPServer) error {
	server.UpdatedAt = time.Now()
	isActive := 0
	if server.IsActive {
		isActive = 1
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE mcp_servers SET name=?, description=?, url=?, type=?, is_active=?, headers=?, status=?, error=?, updated_at=? WHERE id=?`,
		server.Name, server.Description, server.URL, server.Type,
		isActive, server.Headers, server.Status, server.Error,
		server.UpdatedAt.Unix(), server.ID,
	)
	return err
}

func (s *MySQLStore) DeleteMCPServer(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_servers WHERE id = ?`, id)
	return err
}

func (s *MySQLStore) CreateMCPTool(ctx context.Context, tool *MCPTool) error {
	return createMCPToolSQL(ctx, s.db, tool, "?")
}

func (s *MySQLStore) ListMCPToolsByServer(ctx context.Context, serverID string) ([]*MCPTool, error) {
	return listMCPToolsByServerSQL(ctx, s.db, serverID, "?")
}

func (s *MySQLStore) DeleteMCPToolsByServer(ctx context.Context, serverID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_tools WHERE server_id = ?`, serverID)
	return err
}

// --- Shared SQL helpers ---

func createMCPToolSQL(ctx context.Context, db *sql.DB, tool *MCPTool, placeholder string) error {
	if tool.ID == "" {
		tool.ID = generateID("mcptool")
	}
	now := time.Now()
	if tool.CreatedAt.IsZero() {
		tool.CreatedAt = now
	}
	tool.UpdatedAt = now

	var query string
	if placeholder == "$" {
		query = `INSERT INTO mcp_tools (id, server_id, name, description, input_schema, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`
	} else {
		query = `INSERT INTO mcp_tools (id, server_id, name, description, input_schema, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
	}

	_, err := db.ExecContext(ctx, query,
		tool.ID, tool.ServerID, tool.Name, tool.Description, tool.InputSchema,
		tool.CreatedAt.Unix(), tool.UpdatedAt.Unix(),
	)
	return err
}

func listMCPToolsByServerSQL(ctx context.Context, db *sql.DB, serverID string, placeholder string) ([]*MCPTool, error) {
	var query string
	if placeholder == "$" {
		query = `SELECT id, server_id, name, description, input_schema, created_at, updated_at
			FROM mcp_tools WHERE server_id = $1 ORDER BY name`
	} else {
		query = `SELECT id, server_id, name, description, input_schema, created_at, updated_at
			FROM mcp_tools WHERE server_id = ? ORDER BY name`
	}

	rows, err := db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []*MCPTool
	for rows.Next() {
		var tool MCPTool
		var createdAt, updatedAt int64
		if err := rows.Scan(&tool.ID, &tool.ServerID, &tool.Name, &tool.Description, &tool.InputSchema, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		tool.CreatedAt = time.Unix(createdAt, 0).UTC()
		tool.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		tools = append(tools, &tool)
	}
	return tools, rows.Err()
}

func scanMCPServerFromRow(row *sql.Row) (*MCPServer, error) {
	var server MCPServer
	var isActive int
	var createdAt, updatedAt int64

	err := row.Scan(
		&server.ID, &server.Name, &server.Description, &server.URL, &server.Type,
		&isActive, &server.Headers, &server.Status, &server.Error,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("MCP server not found")
	}
	if err != nil {
		return nil, err
	}

	server.IsActive = isActive != 0
	server.CreatedAt = time.Unix(createdAt, 0).UTC()
	server.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return &server, nil
}

func scanMCPServerRows(rows *sql.Rows) ([]*MCPServer, error) {
	var servers []*MCPServer
	for rows.Next() {
		var server MCPServer
		var isActive int
		var createdAt, updatedAt int64

		err := rows.Scan(
			&server.ID, &server.Name, &server.Description, &server.URL, &server.Type,
			&isActive, &server.Headers, &server.Status, &server.Error,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		server.IsActive = isActive != 0
		server.CreatedAt = time.Unix(createdAt, 0).UTC()
		server.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		servers = append(servers, &server)
	}
	return servers, rows.Err()
}
