// Package compatibility provides API compatibility configuration management
package compatibility

import (
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
)

// Handler handles compatibility configuration requests
type Handler struct {
	configMgr     *config.Manager
	serverManager *ServerManager
	client        *http.Client
}

// NewHandler creates a new compatibility handler
func NewHandler(configMgr *config.Manager, serverManager *ServerManager) *Handler {
	return &Handler{
		configMgr:     configMgr,
		serverManager: serverManager,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// GetCompatibility returns the current compatibility configuration
func (h *Handler) GetCompatibility(c *gin.Context) {
	cfg := h.configMgr.Get()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"ollama": gin.H{
				"enabled": cfg.Compatibility.Ollama.Enabled,
				"port":    cfg.Compatibility.Ollama.Port,
			},
			"lmstudio": gin.H{
				"enabled": cfg.Compatibility.LMStudio.Enabled,
				"port":    cfg.Compatibility.LMStudio.Port,
			},
		},
	})
}

// PortCheckResult represents the result of checking a port
type PortCheckResult struct {
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"errorType,omitempty"` // "in_use", "permission", "invalid", "unknown"
}

// checkPortAvailability checks if a port is available for use
func checkPortAvailability(port int) PortCheckResult {
	if port < 1 || port > 65535 {
		return PortCheckResult{
			Available: false,
			Error:     fmt.Sprintf("端口 %d 不在有效范围 (1-65535)", port),
			ErrorType: "invalid",
		}
	}

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if opErr, ok := err.(*net.OpError); ok {
			if sysErr, ok := opErr.Err.(*syscall.Errno); ok {
				if *sysErr == syscall.EADDRINUSE {
					return PortCheckResult{
						Available: false,
						Error:     fmt.Sprintf("端口 %d 已被占用", port),
						ErrorType: "in_use",
					}
				}
				if *sysErr == syscall.EACCES {
					return PortCheckResult{
						Available: false,
						Error:     fmt.Sprintf("端口 %d 需要管理员权限", port),
						ErrorType: "permission",
					}
				}
			}
		}
		return PortCheckResult{
			Available: false,
			Error:     fmt.Sprintf("无法使用端口 %d: %v", port, err),
			ErrorType: "unknown",
		}
	}
	utils.CloseQuietly(listener)
	return PortCheckResult{Available: true}
}

// UpdateCompatibility updates the compatibility configuration
func (h *Handler) UpdateCompatibility(c *gin.Context) {
	var req struct {
		Ollama struct {
			Enabled bool `json:"enabled"`
			Port    int  `json:"port"`
		} `json:"ollama"`
		LMStudio struct {
			Enabled bool `json:"enabled"`
			Port    int  `json:"port"`
		} `json:"lmstudio"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("无效的请求格式: %v", err),
		})
		return
	}

	// Get current config to check if we're enabling
	cfg := h.configMgr.Get()
	ollamaEnabling := req.Ollama.Enabled && !cfg.Compatibility.Ollama.Enabled
	lmstudioEnabling := req.LMStudio.Enabled && !cfg.Compatibility.LMStudio.Enabled

	// Check port availability if enabling
	if ollamaEnabling {
		result := checkPortAvailability(req.Ollama.Port)
		if !result.Available {
			logger.Warnf("Ollama API 端口检查失败: %s", result.Error)
			c.JSON(http.StatusOK, gin.H{
				"success":      false,
				"error":        result.Error,
				"errorType":    result.ErrorType,
				"service":      "ollama",
				"autoDisabled": true,
				"data": gin.H{
					"ollama": gin.H{
						"enabled": false,
						"port":    req.Ollama.Port,
					},
					"lmstudio": gin.H{
						"enabled": cfg.Compatibility.LMStudio.Enabled,
						"port":    cfg.Compatibility.LMStudio.Port,
					},
				},
			})
			return
		}
	}

	if lmstudioEnabling {
		result := checkPortAvailability(req.LMStudio.Port)
		if !result.Available {
			logger.Warnf("LM Studio API 端口检查失败: %s", result.Error)
			c.JSON(http.StatusOK, gin.H{
				"success":      false,
				"error":        result.Error,
				"errorType":    result.ErrorType,
				"service":      "lmstudio",
				"autoDisabled": true,
				"data": gin.H{
					"ollama": gin.H{
						"enabled": cfg.Compatibility.Ollama.Enabled,
						"port":    cfg.Compatibility.Ollama.Port,
					},
					"lmstudio": gin.H{
						"enabled": false,
						"port":    req.LMStudio.Port,
					},
				},
			})
			return
		}
	}

	// Validate port ranges
	if req.Ollama.Port < 1 || req.Ollama.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Ollama 端口必须在 1-65535 范围内",
		})
		return
	}

	if req.LMStudio.Port < 1 || req.LMStudio.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "LM Studio 端口必须在 1-65535 范围内",
		})
		return
	}

	// Save original enabled states before modifying cfg
	ollamaWasEnabled := cfg.Compatibility.Ollama.Enabled
	lmstudioWasEnabled := cfg.Compatibility.LMStudio.Enabled

	// Update configuration
	cfg.Compatibility.Ollama.Enabled = req.Ollama.Enabled
	cfg.Compatibility.Ollama.Port = req.Ollama.Port
	cfg.Compatibility.LMStudio.Enabled = req.LMStudio.Enabled
	cfg.Compatibility.LMStudio.Port = req.LMStudio.Port

	// Start/stop servers based on configuration changes
	if h.serverManager != nil {
		// Handle Ollama server
		if req.Ollama.Enabled && !ollamaWasEnabled {
			// Starting Ollama server
			if err := h.serverManager.StartOllamaServer(req.Ollama.Port); err != nil {
				logger.Errorf("启动 Ollama 服务器失败: %v", err)
				c.JSON(http.StatusOK, gin.H{
					"success":      false,
					"error":        fmt.Sprintf("启动 Ollama 服务器失败: %v", err),
					"errorType":    "start_failed",
					"service":      "ollama",
					"autoDisabled": true,
					"data": gin.H{
						"ollama": gin.H{
							"enabled": false,
							"port":    req.Ollama.Port,
						},
						"lmstudio": gin.H{
							"enabled": cfg.Compatibility.LMStudio.Enabled,
							"port":    cfg.Compatibility.LMStudio.Port,
						},
					},
				})
				return
			}
		} else if !req.Ollama.Enabled && ollamaWasEnabled {
			// Stopping Ollama server
			if err := h.serverManager.StopOllamaServer(); err != nil {
				logger.Errorf("停止 Ollama 服务器失败: %v", err)
			}
		}

		// Handle LM Studio server
		if req.LMStudio.Enabled && !lmstudioWasEnabled {
			// Starting LM Studio server
			if err := h.serverManager.StartLMStudioServer(req.LMStudio.Port); err != nil {
				logger.Errorf("启动 LM Studio 服务器失败: %v", err)
				c.JSON(http.StatusOK, gin.H{
					"success":      false,
					"error":        fmt.Sprintf("启动 LM Studio 服务器失败: %v", err),
					"errorType":    "start_failed",
					"service":      "lmstudio",
					"autoDisabled": true,
					"data": gin.H{
						"ollama": gin.H{
							"enabled": req.Ollama.Enabled,
							"port":    req.Ollama.Port,
						},
						"lmstudio": gin.H{
							"enabled": false,
							"port":    req.LMStudio.Port,
						},
					},
				})
				return
			}
		} else if !req.LMStudio.Enabled && lmstudioWasEnabled {
			// Stopping LM Studio server
			if err := h.serverManager.StopLMStudioServer(); err != nil {
				logger.Errorf("停止 LM Studio 服务器失败: %v", err)
			}
		}
	}

	// Save to file
	if err := h.configMgr.Save(cfg); err != nil {
		logger.Errorf("保存兼容性配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("保存配置失败: %v", err),
		})
		return
	}

	logger.Infof("兼容性配置已更新: Ollama(enabled=%v,port=%d), LM Studio(enabled=%v,port=%d)",
		req.Ollama.Enabled, req.Ollama.Port, req.LMStudio.Enabled, req.LMStudio.Port)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已更新",
		"data": gin.H{
			"ollama": gin.H{
				"enabled": req.Ollama.Enabled,
				"port":    req.Ollama.Port,
			},
			"lmstudio": gin.H{
				"enabled": req.LMStudio.Enabled,
				"port":    req.LMStudio.Port,
			},
		},
	})
}

// TestConnection tests if a port is accessible
func (h *Handler) TestConnection(c *gin.Context) {
	var req struct {
		Port int    `json:"port" binding:"required"`
		Type string `json:"type"` // "ollama" or "lmstudio"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "缺少 port 参数",
		})
		return
	}

	// Validate port
	if req.Port < 1 || req.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "端口必须在 1-65535 范围内",
		})
		return
	}

	// Test connection based on type
	testPath := "/api/tags"
	if req.Type == "lmstudio" {
		testPath = "/v1/models"
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", req.Port, testPath)

	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("创建请求失败: %v", err),
		})
		return
	}

	resp, err := h.client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"valid":   false,
			"error":   fmt.Sprintf("连接失败: %v", err),
		})
		return
	}
	defer utils.CloseQuietly(resp.Body)

	if resp.StatusCode == http.StatusOK {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"valid":   true,
			"message": "连接成功",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"valid":   false,
			"error":   fmt.Sprintf("HTTP 状态码: %d", resp.StatusCode),
		})
	}
}
