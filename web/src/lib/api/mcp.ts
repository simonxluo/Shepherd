/**
 * MCP (Model Context Protocol) API client
 */

import { apiClient } from './client';

/**
 * MCP server transport type
 */
export type MCPTransportType = 'sse' | 'streamable-http';

/**
 * MCP server configuration
 */
export interface MCPServerConfig {
  id: string;
  name: string;
  description?: string;
  url: string;
  type: MCPTransportType;
  isActive: boolean;
  headers?: Record<string, string>;
}

/**
 * MCP tool definition
 */
export interface MCPTool {
  name: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
}

/**
 * MCP tool with server context
 */
export interface MCPToolInfo extends MCPTool {
  mcpServerId: string;
  mcpServerUrl: string;
  mcpServerName?: string;
}

/**
 * MCP server info (config + discovered tools)
 */
export interface MCPServerInfo extends MCPServerConfig {
  tools?: MCPTool[];
  status?: string;
  error?: string;
  savedAt?: number;
}

/**
 * MCP configuration
 */
export interface MCPConfig {
  client: {
    enabled: boolean;
    callTimeout: number;
    readyTimeout: number;
  };
  server: {
    enabled: boolean;
    exposeTts: boolean;
    exposeAsr: boolean;
    exposeChat: boolean;
  };
}

/**
 * Tool call result
 */
export interface MCPToolResult {
  content: Array<{
    type: string;
    text?: string;
    data?: string;
    mimeType?: string;
  }>;
  isError?: boolean;
}

/**
 * List all configured MCP servers
 */
export function listMCPServers() {
  return apiClient.get<{ servers: MCPServerInfo[] }>('/mcp/servers');
}

/**
 * Add a new MCP server
 */
export function addMCPServer(config: MCPServerConfig) {
  return apiClient.post<{ success: boolean }>('/mcp/servers', config);
}

/**
 * Update an existing MCP server
 */
export function updateMCPServer(id: string, config: Partial<MCPServerInfo>) {
  return apiClient.put<{ success: boolean }>(`/mcp/servers/${encodeURIComponent(id)}`, config);
}

/**
 * Remove an MCP server
 */
export function removeMCPServer(id: string) {
  return apiClient.delete<{ success: boolean }>(`/mcp/servers/${encodeURIComponent(id)}`);
}

/**
 * Refresh tools from a server (re-discover)
 */
export function refreshMCPServer(id: string) {
  return apiClient.post<{ tools: MCPTool[] }>(`/mcp/servers/${encodeURIComponent(id)}/refresh`);
}

/**
 * Get tools for a specific server
 */
export function getMCPServerTools(id: string) {
  return apiClient.get<{ tools: MCPTool[] }>(`/mcp/servers/${encodeURIComponent(id)}/tools`);
}

/**
 * List all tools from all active servers
 */
export function listAllMCPTools() {
  return apiClient.get<{ tools: MCPToolInfo[] }>('/mcp/tools');
}

/**
 * Call a tool by name
 */
export function callMCPTool(toolName: string, args: Record<string, unknown>) {
  return apiClient.post<MCPToolResult>('/mcp/tools/call', { toolName, arguments: args });
}

/**
 * Get MCP configuration
 */
export function getMCPConfig() {
  return apiClient.get<MCPConfig>('/mcp/config');
}

/**
 * Update MCP configuration
 */
export function updateMCPConfig(config: MCPConfig) {
  return apiClient.put<{ success: boolean }>('/mcp/config', config);
}
