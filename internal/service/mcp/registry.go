package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// Registry manages MCP server configurations and tool indexing.
// Data is persisted to a JSON file (config/node/mcp-tools.json).
type Registry struct {
	path       string
	servers    map[string]*ServerInfo    // server ID -> server info
	toolToID   map[string]string         // tool name -> server ID
	mu         sync.RWMutex
}

// NewRegistry creates a new registry with the given file path.
func NewRegistry(path string) *Registry {
	return &Registry{
		path:     path,
		servers:  make(map[string]*ServerInfo),
		toolToID: make(map[string]string),
	}
}

// Load reads the registry from disk.
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no file yet, start empty
		}
		return fmt.Errorf("read registry file: %w", err)
	}

	var file RegistryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse registry file: %w", err)
	}

	if file.Servers != nil {
		r.servers = file.Servers
	}

	// Rebuild tool index
	r.rebuildIndex()

	return nil
}

// Save writes the registry to disk.
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file := RegistryFile{
		Servers: r.servers,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0o644); err != nil {
		return fmt.Errorf("write registry file: %w", err)
	}

	return nil
}

// GetServers returns all registered servers.
func (r *Registry) GetServers() []*ServerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ServerInfo, 0, len(r.servers))
	for _, s := range r.servers {
		copy := *s
		result = append(result, &copy)
	}
	return result
}

// GetServer returns a server by ID.
func (r *Registry) GetServer(id string) (*ServerInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.servers[id]
	if !ok {
		return nil, false
	}
	copy := *s
	return &copy, true
}

// UpsertServer adds or updates a server configuration.
func (r *Registry) UpsertServer(info *ServerInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if info.ID == "" {
		return fmt.Errorf("server ID is required")
	}

	info.SavedAt = time.Now().Unix()
	r.servers[info.ID] = info
	r.rebuildIndex()

	return nil
}

// RemoveServer removes a server by ID.
func (r *Registry) RemoveServer(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.servers[id]; !ok {
		return fmt.Errorf("server not found: %s", id)
	}

	delete(r.servers, id)
	r.rebuildIndex()

	return nil
}

// UpdateServerTools updates the cached tools for a server.
func (r *Registry) UpdateServerTools(id string, tools []Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.servers[id]; ok {
		s.Tools = tools
		s.Status = "connected"
		s.Error = ""
		s.SavedAt = time.Now().Unix()
		r.rebuildIndex()
	}
}

// SetServerError sets error status for a server.
func (r *Registry) SetServerError(id string, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.servers[id]; ok {
		s.Status = "error"
		s.Error = errMsg
	}
}

// FindServerByTool returns the server ID for a given tool name.
func (r *Registry) FindServerByTool(toolName string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.toolToID[toolName]
	return id, ok
}

// GetAllTools returns all indexed tools across all active servers.
func (r *Registry) GetAllTools() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []ToolInfo
	for _, s := range r.servers {
		if !s.IsActive {
			continue
		}
		for _, t := range s.Tools {
			tools = append(tools, ToolInfo{
				Tool:       t,
				ServerID:   s.ID,
				ServerURL:  s.URL,
				ServerName: s.Name,
			})
		}
	}
	return tools
}

// rebuildIndex rebuilds the tool→server ID mapping.
// Must be called with write lock held.
func (r *Registry) rebuildIndex() {
	r.toolToID = make(map[string]string)
	for id, s := range r.servers {
		if !s.IsActive {
			continue
		}
		for _, tool := range s.Tools {
			if _, exists := r.toolToID[tool.Name]; !exists {
				r.toolToID[tool.Name] = id
			} else {
				logger.Debugf("MCP tool %q already registered, skipping duplicate from server %s", tool.Name, id)
			}
		}
	}
}

// resolveEnvPlaceholders resolves ${ENV_VAR} placeholders in a string.
func resolveEnvPlaceholders(s string) string {
	return os.Expand(s, os.Getenv)
}

// ResolveHeaders resolves environment variable placeholders in server headers.
func ResolveHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	resolved := make(map[string]string, len(headers))
	for k, v := range headers {
		resolved[k] = resolveEnvPlaceholders(v)
	}
	return resolved
}
