package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/comm/storage"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

// Service orchestrates the MCP client and server functionality.
type Service struct {
	Client   *Client
	Server   *MCPServer
	Registry *Registry

	configMgr *config.Manager
	modelMgr  *model.Manager
	store     storage.Store
}

// NewService creates a new MCP service.
func NewService(configMgr *config.Manager, modelMgr *model.Manager, store storage.Store) *Service {
	// Determine registry path (fallback for file-based registry)
	registryPath := filepath.Join("config", "node", "mcp-tools.json")

	registry := NewRegistry(registryPath)

	// Get timeout from config
	callTimeout := DefaultCallTimeout
	cfg := configMgr.Get()
	if cfg.MCP.Client.CallTimeout > 0 {
		callTimeout = time.Duration(cfg.MCP.Client.CallTimeout) * time.Second
	}

	client := NewClient(registry, callTimeout)
	server := NewMCPServer(modelMgr)

	// Apply server exposure settings
	server.SetExposure(
		cfg.MCP.Server.ExposeTTS,
		cfg.MCP.Server.ExposeASR,
		cfg.MCP.Server.ExposeChat,
	)

	return &Service{
		Client:    client,
		Server:    server,
		Registry:  registry,
		configMgr: configMgr,
		modelMgr:  modelMgr,
		store:     store,
	}
}

// Initialize loads MCP servers from the database and indexes tools from active servers.
func (s *Service) Initialize() error {
	// Load servers from database into registry
	if err := s.loadFromDatabase(); err != nil {
		logger.Warnf("MCP: failed to load from database, falling back to file: %v", err)
		// Fallback to file-based registry
		if err := s.Registry.Load(); err != nil {
			return err
		}
	}

	// Index tools from active servers in background
	go s.indexActiveServers()

	return nil
}

// Stop performs cleanup.
func (s *Service) Stop() {
	// Save registry state to file as backup
	if err := s.Registry.Save(); err != nil {
		logger.Warnf("failed to save MCP registry: %v", err)
	}
}

// AddServer registers a new MCP server and discovers its tools.
func (s *Service) AddServer(cfg ServerConfig) error {
	info := &ServerInfo{
		ServerConfig: cfg,
		Status:       "inactive",
	}

	if err := s.Registry.UpsertServer(info); err != nil {
		return err
	}

	// Persist to database
	s.persistServerToDB(info)

	// If active, connect and discover tools
	if cfg.IsActive {
		tools, err := s.Client.ConnectAndIndex(&cfg)
		if err != nil {
			logger.Warnf("MCP: failed to connect to server %s: %v", cfg.Name, err)
		} else {
			logger.Infof("MCP: added server %s with %d tools", cfg.Name, len(tools))
			// Persist tools to database
			s.persistToolsToDB(cfg.ID, tools)
		}
	}

	return s.Registry.Save()
}

// RemoveServer removes a server and its tools.
func (s *Service) RemoveServer(id string) error {
	if err := s.Registry.RemoveServer(id); err != nil {
		return err
	}

	// Remove from database
	ctx := context.Background()
	if s.store != nil {
		_ = s.store.DeleteMCPToolsByServer(ctx, id)
		_ = s.store.DeleteMCPServer(ctx, id)
	}

	return s.Registry.Save()
}

// UpdateServer updates a server configuration.
func (s *Service) UpdateServer(info *ServerInfo) error {
	if err := s.Registry.UpsertServer(info); err != nil {
		return err
	}

	// Persist to database
	s.persistServerToDB(info)

	return s.Registry.Save()
}

// RefreshServer reconnects to a server and re-discovers its tools.
func (s *Service) RefreshServer(id string) ([]Tool, error) {
	server, ok := s.Registry.GetServer(id)
	if !ok {
		return nil, fmt.Errorf("server not found: %s", id)
	}

	tools, err := s.Client.ConnectAndIndex(&server.ServerConfig)
	if err != nil {
		return nil, err
	}

	// Persist tools to database
	s.persistToolsToDB(id, tools)

	if err := s.Registry.Save(); err != nil {
		logger.Warnf("MCP: failed to save registry after refresh: %v", err)
	}

	return tools, nil
}

// ListServers returns all configured servers.
func (s *Service) ListServers() []*ServerInfo {
	return s.Registry.GetServers()
}

// ListAllTools returns all indexed tools across all active servers.
func (s *Service) ListAllTools() []ToolInfo {
	return s.Registry.GetAllTools()
}

// CallTool invokes a tool by name.
func (s *Service) CallTool(toolName string, args map[string]any) (*ToolsCallResult, error) {
	return s.Client.CallTool(toolName, args)
}

// GetMCPConfig returns the current MCP configuration.
func (s *Service) GetMCPConfig() config.MCPConfig {
	return s.configMgr.Get().MCP
}

// UpdateMCPConfig updates the MCP configuration.
func (s *Service) UpdateMCPConfig(mcpCfg config.MCPConfig) error {
	cfg := s.configMgr.Get()
	cfg.MCP = mcpCfg
	if err := s.configMgr.Save(cfg); err != nil {
		return err
	}

	// Apply new server exposure settings
	s.Server.SetExposure(mcpCfg.Server.ExposeTTS, mcpCfg.Server.ExposeASR, mcpCfg.Server.ExposeChat)

	return nil
}

// indexActiveServers connects to all active servers and indexes their tools.
func (s *Service) indexActiveServers() {
	servers := s.Registry.GetServers()
	for _, server := range servers {
		if !server.IsActive {
			continue
		}
		if _, err := s.Client.ConnectAndIndex(&server.ServerConfig); err != nil {
			logger.Warnf("MCP: failed to index server %s (%s): %v", server.Name, server.URL, err)
		}
	}
}

// Database persistence helpers

// loadFromDatabase loads MCP servers and tools from the storage layer into the registry.
func (s *Service) loadFromDatabase() error {
	if s.store == nil {
		return fmt.Errorf("no store available")
	}

	ctx := context.Background()
	dbServers, err := s.store.ListMCPServers(ctx)
	if err != nil {
		return fmt.Errorf("list MCP servers from DB: %w", err)
	}

	for _, dbServer := range dbServers {
		info := dbServerToInfo(dbServer)

		// Load tools for this server
		dbTools, err := s.store.ListMCPToolsByServer(ctx, dbServer.ID)
		if err != nil {
			logger.Warnf("MCP: failed to load tools for server %s: %v", dbServer.ID, err)
			continue
		}

		for _, dbTool := range dbTools {
			tool := Tool{
				Name:        dbTool.Name,
				Description: dbTool.Description,
			}
			if dbTool.InputSchema != "" && dbTool.InputSchema != "{}" {
				var schema JSONSchema
				if err := json.Unmarshal([]byte(dbTool.InputSchema), &schema); err == nil {
					tool.InputSchema = &schema
				}
			}
			info.Tools = append(info.Tools, tool)
		}

		_ = s.Registry.UpsertServer(info)
	}

	logger.Infof("MCP: loaded %d servers from database", len(dbServers))
	return nil
}

// persistServerToDB saves a server config to the database.
func (s *Service) persistServerToDB(info *ServerInfo) {
	if s.store == nil {
		return
	}

	ctx := context.Background()
	dbServer := infoToDBServer(info)

	// Try update first, create if not found
	if err := s.store.UpdateMCPServer(ctx, dbServer); err != nil {
		if err2 := s.store.CreateMCPServer(ctx, dbServer); err2 != nil {
			logger.Warnf("MCP: failed to persist server %s to DB: %v", info.ID, err2)
		}
	}
}

// persistToolsToDB saves discovered tools to the database.
func (s *Service) persistToolsToDB(serverID string, tools []Tool) {
	if s.store == nil {
		return
	}

	ctx := context.Background()

	// Delete existing tools for this server
	_ = s.store.DeleteMCPToolsByServer(ctx, serverID)

	// Insert new tools
	for _, tool := range tools {
		schemaJSON := "{}"
		if tool.InputSchema != nil {
			if data, err := json.Marshal(tool.InputSchema); err == nil {
				schemaJSON = string(data)
			}
		}

		dbTool := &storage.MCPTool{
			ServerID:    serverID,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schemaJSON,
		}
		if err := s.store.CreateMCPTool(ctx, dbTool); err != nil {
			logger.Warnf("MCP: failed to persist tool %s: %v", tool.Name, err)
		}
	}
}

// Conversion helpers

func dbServerToInfo(db *storage.MCPServer) *ServerInfo {
	headers := make(map[string]string)
	if db.Headers != "" && db.Headers != "{}" {
		_ = json.Unmarshal([]byte(db.Headers), &headers)
	}

	return &ServerInfo{
		ServerConfig: ServerConfig{
			ID:          db.ID,
			Name:        db.Name,
			Description: db.Description,
			URL:         db.URL,
			Type:        TransportType(db.Type),
			IsActive:    db.IsActive,
			Headers:     headers,
		},
		Status:  db.Status,
		Error:   db.Error,
		SavedAt: db.UpdatedAt.Unix(),
	}
}

func infoToDBServer(info *ServerInfo) *storage.MCPServer {
	headersJSON := "{}"
	if len(info.Headers) > 0 {
		if data, err := json.Marshal(info.Headers); err == nil {
			headersJSON = string(data)
		}
	}

	return &storage.MCPServer{
		ID:          info.ID,
		Name:        info.Name,
		Description: info.Description,
		URL:         info.URL,
		Type:        string(info.Type),
		IsActive:    info.IsActive,
		Headers:     headersJSON,
		Status:      info.Status,
		Error:       info.Error,
	}
}
