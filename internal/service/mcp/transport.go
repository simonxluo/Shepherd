package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// Transport defines the interface for MCP communication transports.
type Transport interface {
	// Initialize performs the MCP handshake and returns server info.
	Initialize(ctx context.Context, url string, headers map[string]string) (*InitializeResult, error)
	// ListTools fetches available tools from the MCP server.
	ListTools(ctx context.Context, url string, headers map[string]string) ([]Tool, error)
	// CallTool invokes a tool on the MCP server.
	CallTool(ctx context.Context, url string, params ToolsCallParams, headers map[string]string) (*ToolsCallResult, error)
}

var requestIDCounter atomic.Int64

func nextRequestID() int64 {
	return requestIDCounter.Add(1)
}

// --- SSE Transport ---

// SSETransport implements the MCP SSE transport.
// Flow: GET /sse → receive endpoint event → POST JSON-RPC to that endpoint.
type SSETransport struct {
	httpClient *http.Client
}

func NewSSETransport(timeout time.Duration) *SSETransport {
	return &SSETransport{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (t *SSETransport) Initialize(ctx context.Context, serverURL string, headers map[string]string) (*InitializeResult, error) {
	endpoint, cleanup, err := t.connectSSE(ctx, serverURL, headers)
	if err != nil {
		return nil, fmt.Errorf("SSE connect failed: %w", err)
	}
	defer cleanup()

	params := InitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities:    ClientCaps{},
		ClientInfo:      ClientInfo{Name: "Shepherd", Version: "1.0.0"},
	}

	resp, err := t.postJsonRpc(ctx, endpoint, "initialize", params, headers)
	if err != nil {
		return nil, err
	}

	// Send initialized notification
	_ = t.postNotification(ctx, endpoint, "notifications/initialized", nil, headers)

	var result InitializeResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse initialize result: %w", err)
	}
	return &result, nil
}

func (t *SSETransport) ListTools(ctx context.Context, serverURL string, headers map[string]string) ([]Tool, error) {
	endpoint, cleanup, err := t.connectSSE(ctx, serverURL, headers)
	if err != nil {
		return nil, fmt.Errorf("SSE connect failed: %w", err)
	}
	defer cleanup()

	// Handshake
	params := InitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities:    ClientCaps{},
		ClientInfo:      ClientInfo{Name: "Shepherd", Version: "1.0.0"},
	}
	if _, err := t.postJsonRpc(ctx, endpoint, "initialize", params, headers); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}
	_ = t.postNotification(ctx, endpoint, "notifications/initialized", nil, headers)

	// List tools
	resp, err := t.postJsonRpc(ctx, endpoint, "tools/list", nil, headers)
	if err != nil {
		return nil, err
	}

	var result ToolsListResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse tools/list result: %w", err)
	}
	return result.Tools, nil
}

func (t *SSETransport) CallTool(ctx context.Context, serverURL string, params ToolsCallParams, headers map[string]string) (*ToolsCallResult, error) {
	endpoint, cleanup, err := t.connectSSE(ctx, serverURL, headers)
	if err != nil {
		return nil, fmt.Errorf("SSE connect failed: %w", err)
	}
	defer cleanup()

	// Handshake
	initParams := InitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities:    ClientCaps{},
		ClientInfo:      ClientInfo{Name: "Shepherd", Version: "1.0.0"},
	}
	if _, err := t.postJsonRpc(ctx, endpoint, "initialize", initParams, headers); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}
	_ = t.postNotification(ctx, endpoint, "notifications/initialized", nil, headers)

	// Call tool
	resp, err := t.postJsonRpc(ctx, endpoint, "tools/call", params, headers)
	if err != nil {
		return nil, err
	}

	var result ToolsCallResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse tools/call result: %w", err)
	}
	return &result, nil
}

// connectSSE establishes an SSE connection and waits for the endpoint event.
func (t *SSETransport) connectSSE(ctx context.Context, serverURL string, headers map[string]string) (endpoint string, cleanup func(), err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return "", nil, fmt.Errorf("SSE connection returned status %d", resp.StatusCode)
	}

	// Read SSE events until we get the endpoint
	scanner := bufio.NewScanner(resp.Body)
	endpointCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		var eventType string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if eventType == "endpoint" && data != "" {
					// Resolve relative URL
					if strings.HasPrefix(data, "/") {
						base := extractBaseURL(serverURL)
						data = base + data
					}
					endpointCh <- data
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		} else {
			errCh <- fmt.Errorf("SSE stream ended without endpoint event")
		}
	}()

	select {
	case ep := <-endpointCh:
		return ep, func() { resp.Body.Close() }, nil
	case err := <-errCh:
		resp.Body.Close()
		return "", nil, err
	case <-ctx.Done():
		resp.Body.Close()
		return "", nil, ctx.Err()
	case <-time.After(DefaultReadyTimeout):
		resp.Body.Close()
		return "", nil, fmt.Errorf("timeout waiting for SSE endpoint event")
	}
}

func (t *SSETransport) postJsonRpc(ctx context.Context, endpoint, method string, params any, headers map[string]string) (*JsonRpcResponse, error) {
	rpcReq := JsonRpcRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      nextRequestID(),
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JSON-RPC POST returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var rpcResp JsonRpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("parse JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

func (t *SSETransport) postNotification(ctx context.Context, endpoint, method string, params any, headers map[string]string) error {
	rpcReq := JsonRpcRequest{
		Jsonrpc: JSONRPCVersion,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// --- Streamable HTTP Transport ---

// StreamableHTTPTransport implements the MCP Streamable HTTP transport.
type StreamableHTTPTransport struct {
	httpClient *http.Client
}

func NewStreamableHTTPTransport(timeout time.Duration) *StreamableHTTPTransport {
	return &StreamableHTTPTransport{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (t *StreamableHTTPTransport) Initialize(ctx context.Context, serverURL string, headers map[string]string) (*InitializeResult, error) {
	params := InitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities:    ClientCaps{},
		ClientInfo:      ClientInfo{Name: "Shepherd", Version: "1.0.0"},
	}

	resp, sessionID, err := t.postJsonRpcWithSession(ctx, serverURL, "initialize", params, "", headers)
	if err != nil {
		return nil, err
	}

	// Send initialized notification
	_ = t.postNotificationWithSession(ctx, serverURL, "notifications/initialized", nil, sessionID, headers)

	var result InitializeResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse initialize result: %w", err)
	}
	return &result, nil
}

func (t *StreamableHTTPTransport) ListTools(ctx context.Context, serverURL string, headers map[string]string) ([]Tool, error) {
	// Handshake
	params := InitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities:    ClientCaps{},
		ClientInfo:      ClientInfo{Name: "Shepherd", Version: "1.0.0"},
	}
	_, sessionID, err := t.postJsonRpcWithSession(ctx, serverURL, "initialize", params, "", headers)
	if err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}
	_ = t.postNotificationWithSession(ctx, serverURL, "notifications/initialized", nil, sessionID, headers)

	// List tools
	resp, _, err := t.postJsonRpcWithSession(ctx, serverURL, "tools/list", nil, sessionID, headers)
	if err != nil {
		return nil, err
	}

	// Cleanup session
	t.deleteSession(ctx, serverURL, sessionID, headers)

	var result ToolsListResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse tools/list result: %w", err)
	}
	return result.Tools, nil
}

func (t *StreamableHTTPTransport) CallTool(ctx context.Context, serverURL string, toolParams ToolsCallParams, headers map[string]string) (*ToolsCallResult, error) {
	// Handshake
	params := InitializeParams{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities:    ClientCaps{},
		ClientInfo:      ClientInfo{Name: "Shepherd", Version: "1.0.0"},
	}
	_, sessionID, err := t.postJsonRpcWithSession(ctx, serverURL, "initialize", params, "", headers)
	if err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}
	_ = t.postNotificationWithSession(ctx, serverURL, "notifications/initialized", nil, sessionID, headers)

	// Call tool
	resp, _, err := t.postJsonRpcWithSession(ctx, serverURL, "tools/call", toolParams, sessionID, headers)
	if err != nil {
		return nil, err
	}

	// Cleanup session
	t.deleteSession(ctx, serverURL, sessionID, headers)

	var result ToolsCallResult
	if err := remarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse tools/call result: %w", err)
	}
	return &result, nil
}

func (t *StreamableHTTPTransport) postJsonRpcWithSession(ctx context.Context, url, method string, params any, sessionID string, headers map[string]string) (*JsonRpcResponse, string, error) {
	rpcReq := JsonRpcRequest{
		Jsonrpc: JSONRPCVersion,
		ID:      nextRequestID(),
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if sessionID != "" {
		req.Header.Set(SessionHeader, sessionID)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Streamable HTTP returned status %d: %s", resp.StatusCode, string(respBody))
	}

	newSessionID := resp.Header.Get(SessionHeader)
	if newSessionID == "" {
		newSessionID = sessionID
	}

	var rpcResp JsonRpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, newSessionID, fmt.Errorf("parse JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, newSessionID, fmt.Errorf("JSON-RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, newSessionID, nil
}

func (t *StreamableHTTPTransport) postNotificationWithSession(ctx context.Context, url, method string, params any, sessionID string, headers map[string]string) error {
	rpcReq := JsonRpcRequest{
		Jsonrpc: JSONRPCVersion,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.Header.Set(SessionHeader, sessionID)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (t *StreamableHTTPTransport) deleteSession(ctx context.Context, url, sessionID string, headers map[string]string) {
	if sessionID == "" {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return
	}
	req.Header.Set(SessionHeader, sessionID)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// --- Helpers ---

func extractBaseURL(fullURL string) string {
	// Extract scheme + host from URL
	idx := strings.Index(fullURL, "://")
	if idx < 0 {
		return fullURL
	}
	rest := fullURL[idx+3:]
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		return fullURL
	}
	return fullURL[:idx+3+slashIdx]
}

// remarshal converts an any (typically map[string]any from JSON) into a typed struct.
func remarshal(src any, dst any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
