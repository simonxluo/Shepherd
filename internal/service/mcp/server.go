package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/simonxluo/Shepherd/internal/comm/logger"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

// MCPServer handles incoming MCP protocol requests and exposes local models as tools.
type MCPServer struct {
	modelMgr       *model.Manager
	sessionMgr     *SessionManager
	tools          []Tool
	exposeTTS      bool
	exposeASR      bool
	exposeChat     bool
}

// NewMCPServer creates a new MCP server instance.
func NewMCPServer(modelMgr *model.Manager) *MCPServer {
	s := &MCPServer{
		modelMgr:   modelMgr,
		sessionMgr: NewSessionManager(),
		exposeTTS:  true,
	}
	s.registerTools()
	return s
}

// SetExposure configures which capabilities to expose.
func (s *MCPServer) SetExposure(tts, asr, chat bool) {
	s.exposeTTS = tts
	s.exposeASR = asr
	s.exposeChat = chat
	s.registerTools()
}

// GetSessionManager returns the session manager.
func (s *MCPServer) GetSessionManager() *SessionManager {
	return s.sessionMgr
}

// registerTools builds the list of tools based on exposure settings.
func (s *MCPServer) registerTools() {
	s.tools = nil

	if s.exposeTTS {
		s.tools = append(s.tools, ttsTool)
	}
	if s.exposeASR {
		s.tools = append(s.tools, asrTool)
	}
	if s.exposeChat {
		s.tools = append(s.tools, chatTool)
	}
}

// HandleInitialize processes the MCP initialize request.
func (s *MCPServer) HandleInitialize(req *JsonRpcRequest) *JsonRpcResponse {
	result := InitializeResult{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities: ServerCaps{
			Tools: &ToolsCap{ListChanged: false},
		},
		ServerInfo: ServerMeta{
			Name:    "Shepherd",
			Version: "1.0.0",
		},
	}

	return &JsonRpcResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

// HandleToolsList processes the tools/list request.
func (s *MCPServer) HandleToolsList(req *JsonRpcRequest) *JsonRpcResponse {
	result := ToolsListResult{
		Tools: s.tools,
	}

	return &JsonRpcResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

// HandleToolsCall processes the tools/call request.
func (s *MCPServer) HandleToolsCall(req *JsonRpcRequest) *JsonRpcResponse {
	var params ToolsCallParams
	if err := remarshal(req.Params, &params); err != nil {
		return &JsonRpcResponse{
			Jsonrpc: JSONRPCVersion,
			ID:      req.ID,
			Error: &JsonRpcError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
		}
	}

	var result *ToolsCallResult
	var err error

	switch params.Name {
	case "text_to_speech":
		result, err = s.executeTTS(params.Arguments)
	case "speech_to_text":
		result, err = s.executeASR(params.Arguments)
	case "chat_completion":
		result, err = s.executeChat(params.Arguments)
	default:
		return &JsonRpcResponse{
			Jsonrpc: JSONRPCVersion,
			ID:      req.ID,
			Error: &JsonRpcError{
				Code:    -32602,
				Message: fmt.Sprintf("Unknown tool: %s", params.Name),
			},
		}
	}

	if err != nil {
		return &JsonRpcResponse{
			Jsonrpc: JSONRPCVersion,
			ID:      req.ID,
			Result: ToolsCallResult{
				Content: []ContentBlock{{Type: "text", Text: err.Error()}},
				IsError: true,
			},
		}
	}

	return &JsonRpcResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

// DispatchRequest routes a JSON-RPC request to the appropriate handler.
func (s *MCPServer) DispatchRequest(req *JsonRpcRequest) *JsonRpcResponse {
	switch req.Method {
	case "initialize":
		return s.HandleInitialize(req)
	case "notifications/initialized":
		// Notification, no response needed
		return nil
	case "tools/list":
		return s.HandleToolsList(req)
	case "tools/call":
		return s.HandleToolsCall(req)
	default:
		return &JsonRpcResponse{
			Jsonrpc: JSONRPCVersion,
			ID:      req.ID,
			Error: &JsonRpcError{
				Code:    -32601,
				Message: "Method not found: " + req.Method,
			},
		}
	}
}

// Tool Execution

func (s *MCPServer) executeTTS(args map[string]any) (*ToolsCallResult, error) {
	modelName, _ := args["model"].(string)
	input, _ := args["input"].(string)

	if modelName == "" {
		return nil, fmt.Errorf("missing required parameter: model")
	}
	if input == "" {
		return nil, fmt.Errorf("missing required parameter: input")
	}

	// Find model by name or alias
	modelID, err := s.findTTSModel(modelName)
	if err != nil {
		return nil, err
	}

	// Get port for the loaded model
	port, err := s.getModelPort(modelID)
	if err != nil {
		return nil, fmt.Errorf("model not loaded or port unavailable: %w", err)
	}

	// Build request body (same as AudioHandler)
	reqBody := map[string]any{
		"model": modelName,
		"input": input,
	}
	// Copy optional params
	for _, key := range []string{"voice", "response_format", "speed", "language", "instructions", "ref_audio", "ref_text", "prompt_audio", "prompt_text", "max_new_tokens", "seed"} {
		if v, ok := args[key]; ok && v != nil {
			reqBody[key] = v
		}
	}

	// Forward to backend
	audioData, contentType, err := s.forwardToBackend(port, "/v1/audio/speech", reqBody, modelID)
	if err != nil {
		return nil, err
	}

	// Return audio as base64 content block
	encoded := base64.StdEncoding.EncodeToString(audioData)
	mimeType := contentType
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}

	return &ToolsCallResult{
		Content: []ContentBlock{
			{
				Type:     "resource",
				Data:     encoded,
				MimeType: mimeType,
			},
		},
	}, nil
}

func (s *MCPServer) executeASR(args map[string]any) (*ToolsCallResult, error) {
	// ASR requires audio input - for MCP, accept base64-encoded audio
	modelName, _ := args["model"].(string)
	audioData, _ := args["audio"].(string) // base64-encoded

	if modelName == "" {
		return nil, fmt.Errorf("missing required parameter: model")
	}
	if audioData == "" {
		return nil, fmt.Errorf("missing required parameter: audio (base64-encoded)")
	}

	return &ToolsCallResult{
		Content: []ContentBlock{
			{Type: "text", Text: "ASR tool execution not yet implemented"},
		},
		IsError: true,
	}, nil
}

func (s *MCPServer) executeChat(args map[string]any) (*ToolsCallResult, error) {
	return &ToolsCallResult{
		Content: []ContentBlock{
			{Type: "text", Text: "Chat tool execution not yet implemented"},
		},
		IsError: true,
	}, nil
}

func (s *MCPServer) findTTSModel(modelName string) (string, error) {
	// Use model manager to find model by name/alias
	models := s.modelMgr.ListModels()
	for _, m := range models {
		if m.Name == modelName || m.ID == modelName {
			caps := s.modelMgr.GetModelCapabilities(m.ID)
			if caps != nil && caps.TTS {
				return m.ID, nil
			}
		}
		if m.Alias == modelName {
			caps := s.modelMgr.GetModelCapabilities(m.ID)
			if caps != nil && caps.TTS {
				return m.ID, nil
			}
		}
	}
	return "", fmt.Errorf("TTS model %q not found or does not have TTS capability", modelName)
}

// getModelPort resolves the port for a loaded model.
func (s *MCPServer) getModelPort(modelID string) (int, error) {
	status, exists := s.modelMgr.GetStatusRef(modelID)
	if !exists {
		return 0, fmt.Errorf("model not loaded: %s", modelID)
	}
	if status.Port == 0 {
		return 0, fmt.Errorf("model port not available: %s", modelID)
	}
	return status.Port, nil
}

func (s *MCPServer) forwardToBackend(port int, path string, reqBody map[string]any, modelID string) ([]byte, string, error) {
	// Acquire concurrency slot
	if status, exists := s.modelMgr.GetStatusRef(modelID); exists {
		if !status.AcquireSlot() {
			return nil, "", fmt.Errorf("model is at concurrent request limit")
		}
		defer status.ReleaseSlot()
		status.InflightAdd()
		defer status.InflightDone()
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", err
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, "", fmt.Errorf("backend returned status %d: %s", resp.StatusCode, string(errBody))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	logger.Infof("MCP TTS tool: generated %d bytes of audio (content-type: %s)", len(data), contentType)

	return data, contentType, nil
}

// Tool Definitions

var ttsTool = Tool{
	Name:        "text_to_speech",
	Description: "Convert text to speech audio using a loaded TTS model. Returns audio data encoded in base64.",
	InputSchema: &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"model":           {Type: "string", Description: "The TTS model name or ID to use"},
			"input":           {Type: "string", Description: "The text to convert to speech"},
			"voice":           {Type: "string", Description: "The voice to use for generation"},
			"response_format": {Type: "string", Description: "Audio format: mp3, wav, opus, flac, aac, pcm", Enum: []string{"mp3", "wav", "opus", "flac", "aac", "pcm"}},
			"speed":           {Type: "number", Description: "Speech speed (0.25 to 4.0)"},
			"language":        {Type: "string", Description: "Language code for generation"},
			"instructions":    {Type: "string", Description: "Instructions for voice style/emotion"},
			"ref_audio":       {Type: "string", Description: "Reference audio URL or base64 for voice cloning"},
			"ref_text":        {Type: "string", Description: "Text content of the reference audio"},
			"prompt_audio":    {Type: "string", Description: "Prompt audio URL or base64"},
			"prompt_text":     {Type: "string", Description: "Prompt text for conditioning"},
			"max_new_tokens":  {Type: "integer", Description: "Maximum tokens to generate"},
			"seed":            {Type: "integer", Description: "Random seed for reproducibility"},
		},
		Required: []string{"model", "input"},
	},
}

var asrTool = Tool{
	Name:        "speech_to_text",
	Description: "Transcribe audio to text using a loaded ASR model.",
	InputSchema: &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"model":    {Type: "string", Description: "The ASR model name or ID to use"},
			"audio":    {Type: "string", Description: "Base64-encoded audio data"},
			"language": {Type: "string", Description: "Language of the audio"},
		},
		Required: []string{"model", "audio"},
	},
}

var chatTool = Tool{
	Name:        "chat_completion",
	Description: "Generate a chat completion using a loaded language model.",
	InputSchema: &JSONSchema{
		Type: "object",
		Properties: map[string]*JSONSchema{
			"model":   {Type: "string", Description: "The model name or ID to use"},
			"message": {Type: "string", Description: "The user message to send"},
		},
		Required: []string{"model", "message"},
	},
}
