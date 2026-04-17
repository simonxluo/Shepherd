package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestNodeAdapter(t *testing.T) (*NodeAdapter, *node.Node, func()) {
	cfg := &node.NodeConfig{
		ID:   "test-master",
		Role: node.NodeRoleMaster,
	}

	n, err := node.NewNode(cfg)
	require.NoError(t, err)

	logCfg := &config.LogConfig{
		Level: "info",
	}
	log, err := logger.NewLogger(logCfg, "test")
	require.NoError(t, err)

	schedulerCfg := &config.SchedulerConfig{
		Strategy:     "round_robin",
		MaxQueueSize: 100,
	}
	adapter := NewNodeAdapter(n, log, schedulerCfg)

	cleanup := func() {
		n.Stop()
	}

	return adapter, n, cleanup
}

func TestNewNodeAdapter(t *testing.T) {
	adapter, _, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.node)
	assert.NotNil(t, adapter.log)
}

func TestNodeAdapter_RegisterNode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, _, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.POST("/api/master/nodes/register", adapter.RegisterNode)

	tests := []struct {
		name        string
		reqBody     interface{}
		wantStatus  int
		wantSuccess bool
	}{
		{
			name: "valid registration",
			reqBody: node.NodeInfo{
				ID:      "test-client-1",
				Address: "192.168.1.100",
				Port:    8080,
				Role:    node.NodeRoleClient,
				Status:  node.NodeStatusOnline,
			},
			wantStatus:  http.StatusOK,
			wantSuccess: true,
		},
		{
			name:        "missing id",
			reqBody:     node.NodeInfo{Address: "192.168.1.100", Port: 8080},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:        "missing address",
			reqBody:     node.NodeInfo{ID: "test-client-2", Port: 8080},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:        "invalid port 0",
			reqBody:     node.NodeInfo{ID: "test-client-3", Address: "192.168.1.100", Port: 0},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:        "invalid port 70000",
			reqBody:     node.NodeInfo{ID: "test-client-4", Address: "192.168.1.100", Port: 70000},
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
		{
			name:        "invalid json",
			reqBody:     "invalid",
			wantStatus:  http.StatusBadRequest,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/api/master/nodes/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestNodeAdapter_ListNodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, _, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/master/nodes", adapter.ListNodes)

	req := httptest.NewRequest("GET", "/api/master/nodes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// 新的统一响应格式，数据在 data 字段中
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.NotNil(t, data["nodes"])
	assert.NotNil(t, data["stats"])
}

func TestNodeAdapter_GetNode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, n, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/master/nodes/:id", adapter.GetNode)

	tests := []struct {
		name       string
		nodeID     string
		setup      func()
		wantStatus int
	}{
		{
			name:       "non-existent node",
			nodeID:     "non-existent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "existing node",
			nodeID: "test-client",
			setup: func() {
				n.RegisterClient(
					&node.NodeInfo{
						ID:      "test-client",
						Name:    "Test Client",
						Address: "192.168.1.100",
						Port:    8080,
						Role:    node.NodeRoleClient,
						Status:  node.NodeStatusOnline,
						Tags:    []string{"test"},
						Capabilities: &node.NodeCapabilities{
							CPUCount:       8,
							Memory:         16 * 1024 * 1024 * 1024, // 16GB
							GPUCount:       1,
							GPUMemory:      24 * 1024 * 1024 * 1024, // 24GB
							SupportsLlama:  true,
							SupportsPython: true,
						},
						Metadata: map[string]string{
							"version": "1.0.0",
						},
					})
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			url := "/api/master/nodes/" + tt.nodeID
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestNodeAdapter_UnregisterNode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, n, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.DELETE("/api/master/nodes/:id", adapter.UnregisterNode)

	tests := []struct {
		name       string
		nodeID     string
		setup      func()
		wantStatus int
	}{
		{
			name:       "non-existent node",
			nodeID:     "non-existent",
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "existing node",
			nodeID: "test-client",
			setup: func() {
				n.RegisterClient(
					&node.NodeInfo{
						ID:      "test-client",
						Address: "192.168.1.100",
						Port:    8080,
						Role:    node.NodeRoleClient,
					})
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			url := "/api/master/nodes/" + tt.nodeID
			req := httptest.NewRequest("DELETE", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestNodeAdapter_HandleHeartbeat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, n, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.POST("/api/master/heartbeat", adapter.HandleHeartbeat)

	tests := []struct {
		name       string
		reqBody    interface{}
		setup      func()
		wantStatus int
	}{
		{
			name: "valid heartbeat",
			reqBody: node.HeartbeatMessage{
				NodeID:    "test-client",
				Timestamp: time.Now(),
				Status:    node.NodeStatusOnline,
			},
			setup: func() {
				n.RegisterClient(&node.NodeInfo{
					ID:      "test-client",
					Address: "192.168.1.100",
					Port:    8080,
					Role:    node.NodeRoleClient,
				})
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing node id",
			reqBody:    node.HeartbeatMessage{Timestamp: time.Now()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			reqBody:    "invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "non-existent node",
			reqBody: node.HeartbeatMessage{
				NodeID:    "non-existent",
				Timestamp: time.Now(),
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/api/master/heartbeat", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestNodeAdapter_SendCommand(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, n, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.POST("/api/master/nodes/:id/command", adapter.SendCommand)

	tests := []struct {
		name       string
		nodeID     string
		reqBody    interface{}
		setup      func()
		wantStatus int
	}{
		{
			name:   "valid command",
			nodeID: "test-client",
			reqBody: node.Command{
				Type:    node.CommandTypeLoadModel,
				Payload: map[string]interface{}{"model": "test-model"},
			},
			setup: func() {
				n.RegisterClient(&node.NodeInfo{
					ID:      "test-client",
					Address: "192.168.1.100",
					Port:    8080,
					Role:    node.NodeRoleClient,
				})
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty node id",
			nodeID:     "",
			reqBody:    node.Command{Type: node.CommandTypeLoadModel},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			nodeID:     "test-client",
			reqBody:    "invalid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			body, _ := json.Marshal(tt.reqBody)
			url := "/api/master/nodes/" + tt.nodeID + "/command"
			req := httptest.NewRequest("POST", url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestNodeAdapter_GetCommands(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, _, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/master/nodes/:id/commands", adapter.GetCommands)

	tests := []struct {
		name       string
		nodeID     string
		wantStatus int
	}{
		{
			name:       "empty node id",
			nodeID:     "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid node",
			nodeID:     "test-client",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/master/nodes/" + tt.nodeID + "/commands"
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestNodeAdapter_ReportCommandResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, _, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.POST("/api/master/command/result", adapter.ReportCommandResult)

	tests := []struct {
		name       string
		reqBody    interface{}
		wantStatus int
	}{
		{
			name: "valid result",
			reqBody: map[string]interface{}{
				"node_id":    "test-client",
				"command_id": "cmd-123",
				"success":    true,
				"output":     "Command executed successfully",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "failed result",
			reqBody: map[string]interface{}{
				"node_id":    "test-client",
				"command_id": "cmd-456",
				"success":    false,
				"error":      "Command failed",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing node id",
			reqBody:    map[string]interface{}{"command_id": "cmd-789", "success": true},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing command id",
			reqBody:    map[string]interface{}{"node_id": "test-client", "success": true},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			reqBody:    "invalid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/api/master/command/result", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestNodeAdapter_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, _, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	api := router.Group("/api")
	adapter.RegisterRoutes(api)

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/master/nodes/register"},
		{"GET", "/api/master/nodes"},
		{"GET", "/api/master/nodes/test-id"},
		{"DELETE", "/api/master/nodes/test-id"},
		{"POST", "/api/master/nodes/test-id/command"},
		{"GET", "/api/master/nodes/test-id/commands"},
		{"POST", "/api/master/heartbeat"},
		{"POST", "/api/master/command/result"},
		{"POST", "/api/master/scan"},
		{"GET", "/api/master/scan/status"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var req *http.Request
			switch route.method {
			case "POST":
				req = httptest.NewRequest("POST", route.path, bytes.NewReader([]byte("{}")))
				req.Header.Set("Content-Type", "application/json")
			default:
				req = httptest.NewRequest(route.method, route.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.True(t, w.Code > 0, "Route should return a valid status code")
		})
	}
}

func TestNodeAdapter_GetNodeInstance(t *testing.T) {
	adapter, n, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	gotNode := adapter.GetNodeInstance()
	assert.Equal(t, n, gotNode)
}

func TestNodeAdapter_SetNodeInstance(t *testing.T) {
	adapter, n, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	newNode := &node.Node{}
	adapter.SetNodeInstance(newNode)
	assert.Equal(t, newNode, adapter.GetNodeInstance())

	adapter.SetNodeInstance(n)
}

func TestNodeAdapter_HandleScanClients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, _, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.POST("/api/master/scan", adapter.HandleScanClients)

	tests := []struct {
		name       string
		reqBody    interface{}
		wantStatus int
	}{
		{
			name:       "start scan with empty body",
			reqBody:    map[string]interface{}{},
			wantStatus: http.StatusOK,
		},
		{
			name: "start scan with params",
			reqBody: map[string]interface{}{
				"cidr":      "192.168.1.0/24",
				"portRange": "9191-9200",
				"timeout":   5,
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "start scan with invalid json",
			reqBody:    "invalid json",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if s, ok := tt.reqBody.(string); ok {
				body = []byte(s)
			} else {
				body, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest("POST", "/api/master/scan", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				// 新的统一响应格式，数据在 data 字段中
				assert.True(t, resp["success"].(bool))
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "网络扫描已启动", data["message"])
			}
		})
	}
}

func TestNodeAdapter_GetClientScanStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter, _, cleanup := setupTestNodeAdapter(t)
	defer cleanup()

	router := gin.New()
	router.GET("/api/master/scan/status", adapter.GetClientScanStatus)

	req := httptest.NewRequest("GET", "/api/master/scan/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// 新的统一响应格式，数据在 data 字段中
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Contains(t, data, "running")
	assert.IsType(t, false, data["running"])
}
