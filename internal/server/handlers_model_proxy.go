package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/simonxluo/Shepherd/internal/comm/types"
	api "github.com/simonxluo/Shepherd/internal/handler"
	"github.com/simonxluo/Shepherd/internal/service/model"
)

// getModelPort returns the port for a loaded model, or 0 if not loaded.
func (s *Server) getModelPort(modelID string) (int, error) {
	st, ok := s.modelMgr.GetStatus(modelID)
	if !ok {
		return 0, fmt.Errorf("模型未找到: %s", modelID)
	}
	if st.State != model.StateLoaded {
		return 0, fmt.Errorf("模型未加载: %s", modelID)
	}
	if st.Port <= 0 {
		return 0, fmt.Errorf("模型端口未分配: %s", modelID)
	}
	return st.Port, nil
}

// proxyToModel forwards a request to the running llama.cpp process for a model.
func (s *Server) proxyToModel(c *gin.Context, modelID, method, path string, body io.Reader) {
	port, err := s.getModelPort(modelID)
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, url, body)
	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("创建代理请求失败: %v", err))
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("代理请求失败: %v", err))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("读取代理响应失败: %v", err))
		return
	}

	// Forward the response as-is
	c.Data(resp.StatusCode, "application/json", respBody)
}

// HandleModelTokenize proxies POST /api/models/:id/tokenize to the running model.
func (s *Server) HandleModelTokenize(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	s.proxyToModel(c, id, http.MethodPost, "/tokenize", c.Request.Body)
}

// HandleModelApplyTemplate proxies POST /api/models/:id/apply-template to the running model.
func (s *Server) HandleModelApplyTemplate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	s.proxyToModel(c, id, http.MethodPost, "/apply-template", c.Request.Body)
}

// HandleModelSlots proxies GET /api/models/:id/slots to the running model.
func (s *Server) HandleModelSlots(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "模型ID不能为空")
		return
	}

	port, err := s.getModelPort(id)
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/slots", port)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("创建代理请求失败: %v", err))
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("获取slots失败: %v", err))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		api.Error(c, types.ErrInternalError, fmt.Sprintf("读取响应失败: %v", err))
		return
	}

	// Parse the response and wrap in standard format
	var slots json.RawMessage
	if err := json.Unmarshal(respBody, &slots); err != nil {
		// Return raw if not valid JSON
		c.Data(resp.StatusCode, "application/json", respBody)
		return
	}

	api.Success(c, gin.H{
		"slots": slots,
	})
}
