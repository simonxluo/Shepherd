// Package mcp provides HTTP handlers for MCP (Model Context Protocol) management.
package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/simonxluo/Shepherd/internal/comm/config"
	"github.com/simonxluo/Shepherd/internal/comm/logger"
	mcpservice "github.com/simonxluo/Shepherd/internal/service/mcp"
)

// Handler handles MCP management API requests.
type Handler struct {
	configMgr  *config.Manager
	mcpService *mcpservice.Service
}

// NewHandler creates a new MCP handler.
func NewHandler(configMgr *config.Manager, mcpService *mcpservice.Service) *Handler {
	return &Handler{
		configMgr:  configMgr,
		mcpService: mcpService,
	}
}

// --- Client Management API ---

// ListServers returns all configured MCP servers.
// GET /api/mcp/servers
func (h *Handler) ListServers(c *gin.Context) {
	servers := h.mcpService.ListServers()
	c.JSON(http.StatusOK, gin.H{"servers": servers})
}

// AddServer adds a new MCP server.
// POST /api/mcp/servers
func (h *Handler) AddServer(c *gin.Context) {
	var cfg mcpservice.ServerConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if cfg.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}
	if cfg.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	if cfg.Name == "" {
		cfg.Name = cfg.ID
	}
	if cfg.Type == "" {
		cfg.Type = mcpservice.TransportSSE
	}

	if err := h.mcpService.AddServer(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateServer updates an existing MCP server.
// PUT /api/mcp/servers/:id
func (h *Handler) UpdateServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	var info mcpservice.ServerInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}
	info.ID = id

	if err := h.mcpService.UpdateServer(&info); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RemoveServer removes an MCP server.
// DELETE /api/mcp/servers/:id
func (h *Handler) RemoveServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	if err := h.mcpService.RemoveServer(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RefreshServer re-discovers tools from a server.
// POST /api/mcp/servers/:id/refresh
func (h *Handler) RefreshServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	tools, err := h.mcpService.RefreshServer(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tools": tools})
}

// GetServerTools returns tools for a specific server.
// GET /api/mcp/servers/:id/tools
func (h *Handler) GetServerTools(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	server, ok := h.mcpService.Registry.GetServer(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tools": server.Tools})
}

// ListAllTools returns all tools from all active servers.
// GET /api/mcp/tools
func (h *Handler) ListAllTools(c *gin.Context) {
	tools := h.mcpService.ListAllTools()
	c.JSON(http.StatusOK, gin.H{"tools": tools})
}

// CallTool invokes a tool by name.
// POST /api/mcp/tools/call
func (h *Handler) CallTool(c *gin.Context) {
	var req struct {
		ToolName  string         `json:"toolName"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if req.ToolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "toolName is required"})
		return
	}

	result, err := h.mcpService.CallTool(req.ToolName, req.Arguments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetConfig returns MCP configuration.
// GET /api/mcp/config
func (h *Handler) GetConfig(c *gin.Context) {
	cfg := h.mcpService.GetMCPConfig()
	c.JSON(http.StatusOK, cfg)
}

// UpdateConfig updates MCP configuration.
// PUT /api/mcp/config
func (h *Handler) UpdateConfig(c *gin.Context) {
	var cfg config.MCPConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if err := h.mcpService.UpdateMCPConfig(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// --- MCP Server Protocol Endpoints ---

// HandleMCPSSE handles SSE connections from external MCP clients.
// GET /mcp/sse
func (h *Handler) HandleMCPSSE(c *gin.Context) {
	cfg := h.mcpService.GetMCPConfig()
	if !cfg.Server.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MCP server is disabled"})
		return
	}

	sessionID := h.mcpService.Server.GetSessionManager().Create()

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Send endpoint event with POST URL
	postURL := fmt.Sprintf("/mcp/message?sessionId=%s", sessionID)
	c.SSEvent("endpoint", postURL)
	c.Writer.Flush()

	// Keep connection alive until client disconnects
	<-c.Request.Context().Done()

	// Cleanup session
	h.mcpService.Server.GetSessionManager().Delete(sessionID)
}

// HandleMCPMessage handles JSON-RPC messages from SSE-connected MCP clients.
// POST /mcp/message
func (h *Handler) HandleMCPMessage(c *gin.Context) {
	cfg := h.mcpService.GetMCPConfig()
	if !cfg.Server.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MCP server is disabled"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var req mcpservice.JsonRpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"jsonrpc": "2.0",
			"error":   map[string]any{"code": -32700, "message": "Parse error"},
		})
		return
	}

	resp := h.mcpService.Server.DispatchRequest(&req)
	if resp == nil {
		// Notification - no response needed
		c.Status(http.StatusAccepted)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleMCPStreamable handles Streamable HTTP requests from MCP clients.
// POST /mcp
func (h *Handler) HandleMCPStreamable(c *gin.Context) {
	cfg := h.mcpService.GetMCPConfig()
	if !cfg.Server.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MCP server is disabled"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var req mcpservice.JsonRpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"jsonrpc": "2.0",
			"error":   map[string]any{"code": -32700, "message": "Parse error"},
		})
		return
	}

	// Handle session
	sessionID := c.GetHeader(mcpservice.SessionHeader)
	if req.Method == "initialize" {
		sessionID = h.mcpService.Server.GetSessionManager().Create()
	} else if sessionID != "" {
		if _, ok := h.mcpService.Server.GetSessionManager().Get(sessionID); !ok {
			logger.Warnf("MCP: unknown session %s, creating new one", sessionID)
			sessionID = h.mcpService.Server.GetSessionManager().Create()
		}
	}

	resp := h.mcpService.Server.DispatchRequest(&req)
	if resp == nil {
		c.Status(http.StatusAccepted)
		return
	}

	if sessionID != "" {
		c.Header(mcpservice.SessionHeader, sessionID)
	}
	c.JSON(http.StatusOK, resp)
}

// HandleMCPSessionDelete handles session termination.
// DELETE /mcp
func (h *Handler) HandleMCPSessionDelete(c *gin.Context) {
	sessionID := c.GetHeader(mcpservice.SessionHeader)
	if sessionID != "" {
		h.mcpService.Server.GetSessionManager().Delete(sessionID)
	}
	c.Status(http.StatusNoContent)
}
