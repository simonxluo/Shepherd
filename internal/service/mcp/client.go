package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
)

// Client manages connections to external MCP servers.
type Client struct {
	sseTransport        *SSETransport
	streamableTransport *StreamableHTTPTransport
	registry            *Registry
	callTimeout         time.Duration
	mu                  sync.RWMutex
}

// NewClient creates a new MCP client.
func NewClient(registry *Registry, callTimeout time.Duration) *Client {
	return &Client{
		sseTransport:        NewSSETransport(callTimeout),
		streamableTransport: NewStreamableHTTPTransport(callTimeout),
		registry:            registry,
		callTimeout:         callTimeout,
	}
}

// ConnectAndIndex connects to a server, performs handshake, discovers tools, and indexes them.
func (c *Client) ConnectAndIndex(cfg *ServerConfig) ([]Tool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.callTimeout)
	defer cancel()

	headers := ResolveHeaders(cfg.Headers)
	transport := c.getTransport(cfg.Type)

	tools, err := transport.ListTools(ctx, cfg.URL, headers)
	if err != nil {
		c.registry.SetServerError(cfg.ID, err.Error())
		return nil, fmt.Errorf("discover tools from %s: %w", cfg.URL, err)
	}

	c.registry.UpdateServerTools(cfg.ID, tools)
	logger.Infof("MCP client indexed %d tools from server %s (%s)", len(tools), cfg.Name, cfg.URL)

	return tools, nil
}

// ListTools fetches tools from a specific server.
func (c *Client) ListTools(cfg *ServerConfig) ([]Tool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.callTimeout)
	defer cancel()

	headers := ResolveHeaders(cfg.Headers)
	transport := c.getTransport(cfg.Type)

	return transport.ListTools(ctx, cfg.URL, headers)
}

// CallTool invokes a tool on its registered server.
func (c *Client) CallTool(toolName string, args map[string]any) (*ToolsCallResult, error) {
	serverID, ok := c.registry.FindServerByTool(toolName)
	if !ok {
		return nil, fmt.Errorf("tool %q not found in any registered server", toolName)
	}

	server, ok := c.registry.GetServer(serverID)
	if !ok {
		return nil, fmt.Errorf("server %s not found", serverID)
	}

	return c.CallToolOnServer(server, toolName, args)
}

// CallToolOnServer invokes a tool on a specific server.
func (c *Client) CallToolOnServer(server *ServerInfo, toolName string, args map[string]any) (*ToolsCallResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.callTimeout)
	defer cancel()

	headers := ResolveHeaders(server.Headers)
	transport := c.getTransport(server.Type)

	params := ToolsCallParams{
		Name:      toolName,
		Arguments: args,
	}

	result, err := transport.CallTool(ctx, server.URL, params, headers)
	if err != nil {
		return nil, fmt.Errorf("call tool %q on %s: %w", toolName, server.URL, err)
	}

	return result, nil
}

func (c *Client) getTransport(t TransportType) Transport {
	switch t {
	case TransportStreamableHTTP:
		return c.streamableTransport
	default:
		return c.sseTransport
	}
}
