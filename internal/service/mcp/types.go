// Package mcp implements the Model Context Protocol (MCP) client and server.
package mcp

import "time"

// Protocol constants
const (
	JSONRPCVersion     = "2.0"
	MCPProtocolVersion = "2024-11-05"
	SessionHeader      = "Mcp-Session-Id"
	DefaultCallTimeout = 120 * time.Second
	DefaultReadyTimeout = 30 * time.Second
)

// TransportType represents the MCP transport mechanism.
type TransportType string

const (
	TransportSSE            TransportType = "sse"
	TransportStreamableHTTP TransportType = "streamable-http"
)

// ServerConfig represents an MCP server configuration.
type ServerConfig struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string            `json:"url" yaml:"url"`
	Type        TransportType     `json:"type" yaml:"type"`
	IsActive    bool              `json:"isActive" yaml:"is_active"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// ServerInfo is the API-facing server representation with discovered tools.
type ServerInfo struct {
	ServerConfig `yaml:",inline"`
	Tools        []Tool `json:"tools,omitempty" yaml:"tools,omitempty"`
	Status       string `json:"status,omitempty" yaml:"status,omitempty"` // "connected", "error", "inactive"
	Error        string `json:"error,omitempty" yaml:"error,omitempty"`
	SavedAt      int64  `json:"savedAt,omitempty" yaml:"saved_at,omitempty"`
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema *JSONSchema `json:"inputSchema,omitempty"`
}

// JSONSchema is a simplified JSON Schema representation for tool parameters.
type JSONSchema struct {
	Type        string                 `json:"type"`
	Properties  map[string]*JSONSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Description string                 `json:"description,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Items       *JSONSchema            `json:"items,omitempty"`
	Default     any                    `json:"default,omitempty"`
}

// ToolInfo is a tool with its server context.
type ToolInfo struct {
	Tool       `yaml:",inline"`
	ServerID   string `json:"mcpServerId"`
	ServerURL  string `json:"mcpServerUrl"`
	ServerName string `json:"mcpServerName,omitempty"`
}

// ToolResult represents the result of calling an MCP tool.
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in a tool result.
type ContentBlock struct {
	Type     string `json:"type"` // "text", "image", "resource"
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`     // base64-encoded binary
	MimeType string `json:"mimeType,omitempty"` // MIME type for binary data
}

// JSON-RPC Protocol Types

// JsonRpcRequest represents a JSON-RPC 2.0 request.
type JsonRpcRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JsonRpcResponse represents a JSON-RPC 2.0 response.
type JsonRpcResponse struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      any          `json:"id,omitempty"`
	Result  any          `json:"result,omitempty"`
	Error   *JsonRpcError `json:"error,omitempty"`
}

// JsonRpcError represents a JSON-RPC error.
type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP Protocol Message Params

// InitializeParams is sent by the client during handshake.
type InitializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    ClientCaps `json:"capabilities"`
	ClientInfo      ClientInfo `json:"clientInfo"`
}

// ClientCaps represents client capabilities.
type ClientCaps struct {
	Roots *struct{} `json:"roots,omitempty"`
}

// ClientInfo identifies the MCP client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult is returned by the server after handshake.
type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    ServerCaps `json:"capabilities"`
	ServerInfo      ServerMeta `json:"serverInfo"`
}

// ServerCaps represents server capabilities.
type ServerCaps struct {
	Tools *ToolsCap `json:"tools,omitempty"`
}

// ToolsCap represents tools capability.
type ToolsCap struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerMeta identifies the MCP server.
type ServerMeta struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsListResult is returned by tools/list.
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolsCallParams is sent when calling a tool.
type ToolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolsCallResult is returned after calling a tool.
type ToolsCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// Registry File Format

// RegistryFile represents the persisted mcp-tools.json structure.
type RegistryFile struct {
	Servers map[string]*ServerInfo `json:"mcpServers"`
}
